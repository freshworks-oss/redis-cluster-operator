package distributedrediscluster

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	rediskunv1alpha1 "github.com/dinesh-murugiah/rediscluster-operator/api/v1alpha1"
	redisv1alpha1 "github.com/dinesh-murugiah/rediscluster-operator/api/v1alpha1"
	"github.com/dinesh-murugiah/rediscluster-operator/controllers/clustering"
	"github.com/dinesh-murugiah/rediscluster-operator/controllers/manager"
	"github.com/dinesh-murugiah/rediscluster-operator/metrics"
	"github.com/dinesh-murugiah/rediscluster-operator/resources/configmaps"
	"github.com/dinesh-murugiah/rediscluster-operator/resources/statefulsets"
	"github.com/dinesh-murugiah/rediscluster-operator/utils/k8sutil"
	"github.com/dinesh-murugiah/rediscluster-operator/utils/redisutil"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	requeueAfter                 = 15 * time.Second
	redisConfStorageVolumeName   = "redis-conf"
	redisBackupStorageVolumeName = "redis-backup"
)

type syncContext struct {
	cluster              *redisv1alpha1.DistributedRedisCluster
	clusterInfos         *redisutil.ClusterInfos
	admin                redisutil.IAdmin
	healer               manager.IHeal
	pods                 []*corev1.Pod
	proxyPods            []*corev1.Pod
	reqLogger            logr.Logger
	client               client.Client
	configMapClient      k8sutil.IConfigMapControl
	isSingleAzDeployment bool
}

func (r *DistributedRedisClusterReconciler) ensureCluster(ctx *syncContext, update bool) (redisv1alpha1.StsEnsureStatus, error) {

	ensureSTSStatus := rediskunv1alpha1.STSNotEnsured

	cluster := ctx.cluster
	if err := r.validateAndSetDefault(cluster, ctx.reqLogger); err != nil {
		r.MetricsClient.ReconcileCluster(cluster.Namespace, cluster.Name, "VALIDATE_SPEC", metrics.ENSURE, metrics.FAIL, err.Error())
		if k8sutil.IsRequestRetryable(err) {
			return ensureSTSStatus, InvalidSpec.Wrap(err, "Validate Requeue")
		}
		return ensureSTSStatus, StopRetry.Wrap(err, "stop retry")
	}

	// Redis only load db from append only file when AOF ON, because of
	// we only backed up the RDB file when doing data backup, so we set
	// "appendonly no" force here when do restore.
	dbLoadedFromDiskWhenRestore(cluster, ctx.reqLogger)
	labels := getLabels(cluster)
	if err := r.Ensurer.EnsureRedisConfigMap(cluster, labels); err != nil {
		r.MetricsClient.ReconcileCluster(cluster.Namespace, cluster.Name, "ENSURE_CONFIGMAP", metrics.ENSURE, metrics.FAIL, err.Error())
		switch GetType(err) {
		case StopRetry:
			return ensureSTSStatus, StopRetry.Wrap(err, "Irrecoverable error when EnsureRedisConfigMap")
		case Requeue:
			return ensureSTSStatus, EnsureConfigMap.Wrap(err, "EnsureRedisConfigMap Requeue")
		}
	}

	ensureSTSStatus, err := r.Ensurer.EnsureRedisStatefulsets(cluster, labels, update)
	if err != nil {
		ctx.reqLogger.Error(err, "EnsureRedisStatefulSets")
		r.MetricsClient.ReconcileCluster(cluster.Namespace, cluster.Name, "ENSURE_STATEFULSET", metrics.ENSURE, metrics.FAIL, err.Error())
		return ensureSTSStatus, EnsureSTS.Wrap(err, "EnsureRedisStatefulSets failed")
	}
	if err1 := r.Ensurer.EnsureRedisPDB(cluster, labels); err1 != nil {
		r.MetricsClient.ReconcileCluster(cluster.Namespace, cluster.Name, "ENSURE_PDB", metrics.ENSURE, metrics.FAIL, err1.Error())
		return ensureSTSStatus, EnsurePDB.Wrap(err1, "EnsureRedisPDB failed")
	}
	if err2 := r.Ensurer.EnsureClusterShardsHeadLessSvcs(cluster, labels); err2 != nil {
		r.MetricsClient.ReconcileCluster(cluster.Namespace, cluster.Name, "ENSURE_SHARD_SERVICE", metrics.ENSURE, metrics.FAIL, err2.Error())
		return ensureSTSStatus, EnsureService.Wrap(err2, "EnsureRedisHeadLessSvcs per shard failed")
	}
	if err3 := r.Ensurer.EnsureClusterHeadLessSvc(cluster, labels); err3 != nil {
		r.MetricsClient.ReconcileCluster(cluster.Namespace, cluster.Name, "ENSURE_CLUSTER_SERVICE", metrics.ENSURE, metrics.FAIL, err3.Error())
		return ensureSTSStatus, EnsureService.Wrap(err3, "EnsureRedisSvc for all shards failed")
	}
	if err4 := r.Ensurer.EnsureRedisRCloneSecret(cluster, labels); err4 != nil {
		r.MetricsClient.ReconcileCluster(cluster.Namespace, cluster.Name, "ENSURE_REDIS_RCLONE_SECRET", metrics.ENSURE, metrics.FAIL, err4.Error())
		if k8sutil.IsRequestRetryable(err4) {
			return ensureSTSStatus, Kubernetes.Wrap(err4, "EnsureRedisRCloneSecret")
		}
		return ensureSTSStatus, StopRetry.Wrap(err4, "stop retry")
	}
	return ensureSTSStatus, nil
}

func (r *DistributedRedisClusterReconciler) waitPodReady(ctx *syncContext) error {
	var terminationGracePeriod int64 = 30 // default value
	if ctx.cluster.Spec.TerminationGracePeriod != nil {
		terminationGracePeriod = *ctx.cluster.Spec.TerminationGracePeriod
	}
	if _, err := ctx.healer.FixTerminatingPods(ctx.cluster, time.Duration(terminationGracePeriod)); err != nil {
		return Kubernetes.Wrap(err, "FixTerminatingPods")
	}
	if err := r.Checker.CheckRedisNodeNum(ctx.cluster); err != nil {
		return Requeue.Wrap(err, "CheckRedisNodeNum")
	}

	if !ctx.isSingleAzDeployment {
		return nil
	}

	// This is a single AZ deployment for Proxy and Redis Masters.
	if len(ctx.proxyPods) == 0 {
		return Requeue.New("waiting for proxy pod creation")
	}

	for _, pod := range ctx.proxyPods {
		if pod.Status.Phase == "Running" {
			return nil // Found a running proxy pod, proceed further.
		}
	}

	// No running proxy pod found
	return Requeue.New("waiting for proxy pod to be in running state")
}

func (r *DistributedRedisClusterReconciler) validateAndSetDefault(cluster *redisv1alpha1.DistributedRedisCluster, reqLogger logr.Logger) error {
	var update bool
	var err error

	if strings.HasPrefix(cluster.Name, redisv1alpha1.CrResourcesprefix) {
		nameSpace := cluster.Name[len(redisv1alpha1.CrResourcesprefix):]
		if len(nameSpace) <= 0 {
			reqLogger.Error(err, "validateAndSetDefault", "possibly missing namespace prefix for: ", redisv1alpha1.CrResourcesprefix)
			err = fmt.Errorf("possibly missing namespace prefix for: %s", redisv1alpha1.CrResourcesprefix)
			return err
		}

	} else {
		reqLogger.Error(err, "validateAndSetDefault", "cluster name should start with:", redisv1alpha1.CrResourcesprefix)
		err = fmt.Errorf("cluster name should start with %s", redisv1alpha1.CrResourcesprefix)
		return err

	}
	// TODO: DR-1173 Move this to validation webhook
	if cluster.Spec.HaConfig == nil {
		err = fmt.Errorf("ha config expected missing")
		return err

	} else {
		if cluster.Spec.HaConfig != nil && cluster.Spec.HaConfig.HaEnabled {
			//limit the HA solution to 3 zones , neither lesser nor higher
			if cluster.Spec.HaConfig.ZonesInfo == nil || (len(cluster.Spec.HaConfig.ZonesInfo) != 3) {
				err = fmt.Errorf("haspec missing zone info or incorrect zone info")
				return err
			}
			if cluster.Spec.ClusterReplicas != int32(len(cluster.Spec.HaConfig.ZonesInfo)-1) {
				err = fmt.Errorf("cluster replicas should be equal to availabilty zones")
				return err
			}
		}

	}

	// TODO: DR-1173 Move this to validation webhook
	if cluster.Spec.Monitor != nil {
		for _, container := range *cluster.Spec.Monitor {
			if container.Image == "" || container.Prometheus == nil || (container.Prometheus.Port < 1024 || container.Prometheus.Port > 49151) {
				err = fmt.Errorf("incorrect / required fields missing for monitor")
				return err
			}
		}
	}

	if (cluster.Spec.ConfStorage != nil && cluster.Spec.ConfStorage.Type == "") || (cluster.Spec.BackupStorage != nil && cluster.Spec.BackupStorage.Type == "") {
		err = fmt.Errorf("storage defined, but storage type missing")
		return err
	}

	if (cluster.Spec.ConfStorage != nil && !statefulsets.IsStorageTypeSupported(cluster.Spec.ConfStorage.Type)) || (cluster.Spec.BackupStorage != nil && !statefulsets.IsStorageTypeSupported(cluster.Spec.BackupStorage.Type)) {
		err = fmt.Errorf("storage type not supported")
		return err
	}

	if (cluster.Spec.ConfStorage != nil && statefulsets.IsPersistentStorageType(cluster.Spec.ConfStorage.Type) && cluster.Spec.ConfStorage.Class == "") || (cluster.Spec.BackupStorage != nil && statefulsets.IsPersistentStorageType(cluster.Spec.BackupStorage.Type) && cluster.Spec.BackupStorage.Class == "") {
		err = fmt.Errorf("storage class missing for persistent storage")
		return err
	}

	if (cluster.Spec.ConfStorage != nil && statefulsets.IsPersistentStorageType(cluster.Spec.ConfStorage.Type) && cluster.Spec.BackupStorage != nil && statefulsets.IsPersistentStorageType(cluster.Spec.BackupStorage.Type)) && !statefulsets.IsPersistentStorageTypeSame(cluster.Spec.ConfStorage.Type, cluster.Spec.BackupStorage.Type) {
		err = fmt.Errorf("storage type should be same for both conf and backup storage if both are persistent")
		return err
	}

	//Create PVC here if not present
	if cluster.Spec.ConfStorage.Type == rediskunv1alpha1.PersistentClaimEfs {

		// Get PVC
		_, err := r.PvcController.GetPvc(cluster.Namespace, redisConfStorageVolumeName)

		if err != nil {
			if errors.IsNotFound(err) {
				// Create PVC
				claim := corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:            redisConfStorageVolumeName,
						Namespace:       cluster.Namespace,
						Labels:          cluster.Labels,
						OwnerReferences: redisv1alpha1.DefaultOwnerReferences(cluster),
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteMany,
						},
						StorageClassName: &cluster.Spec.ConfStorage.Class,
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: cluster.Spec.ConfStorage.Size,
							},
						},
					},
				}
				if err := r.PvcController.CreatePvc(&claim); err != nil {
					err1 := fmt.Errorf("unable to create PVC for redis conf for EFS %v in %v", err, cluster.Namespace)
					return err1
				}
			} else {
				err1 := fmt.Errorf("error during fetch PVC redis-conf %v in %v", err, cluster.Namespace)
				return err1
			}
		}
	}

	//Create PVC here if not present
	if cluster.Spec.BackupStorage.Type == rediskunv1alpha1.PersistentClaimEfs {

		// Get PVC
		_, err := r.PvcController.GetPvc(cluster.Namespace, redisBackupStorageVolumeName)
		if err != nil {
			if errors.IsNotFound(err) {
				// Create PVC
				claim := corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:            redisBackupStorageVolumeName,
						Namespace:       cluster.Namespace,
						Labels:          cluster.Labels,
						OwnerReferences: redisv1alpha1.DefaultOwnerReferences(cluster),
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteMany,
						},
						StorageClassName: &cluster.Spec.ConfStorage.Class,
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: cluster.Spec.BackupStorage.Size,
							},
						},
					},
				}
				if err := r.PvcController.CreatePvc(&claim); err != nil {
					err1 := fmt.Errorf("unable to create PVC for redis backup for EFS %v in %v", err, cluster.Namespace)
					return err1
				}
			} else {
				err1 := fmt.Errorf("error during fetch PVC redis-backup %v in %v", err, cluster.Namespace)
				return err1
			}
		}
	}

	if cluster.IsRestoreFromBackup() && cluster.ShouldInitRestorePhase() {
		//Dinesh Todo This flow Needs FIX ####, commented out below to fix status fileds
		err = fmt.Errorf("restore Not supported yet")
		return err
		/*
			reqLogger.Error("force appendonly = no when do restore")
			update, err = r.initRestore(cluster, reqLogger)
			if err != nil {
				return err
			}
		*/
	}

	if cluster.IsRestoreFromBackup() && (cluster.IsRestoreRunning() || cluster.IsRestoreRestarting()) {
		// Set ClusterReplicas = 0, only start master node in first reconcile loop when do restore
		cluster.Spec.ClusterReplicas = 0
	}

	updateDefault := cluster.DefaultSpec(reqLogger)
	if update || updateDefault {
		return r.CrController.UpdateCR(cluster)
	}

	return nil
}

func dbLoadedFromDiskWhenRestore(cluster *redisv1alpha1.DistributedRedisCluster, reqLogger logr.Logger) {
	if cluster.IsRestoreFromBackup() && !cluster.IsRestored() {
		if cluster.Spec.Config != nil {
			reqLogger.Info("force appendonly = no when do restore")
			cluster.Spec.Config["appendonly"] = "no"
		}
	}
}

func (r *DistributedRedisClusterReconciler) initRestore(cluster *redisv1alpha1.DistributedRedisCluster, reqLogger logr.Logger) (bool, error) {
	update := false
	//Dinesh Todo This flow Needs FIX ####, commented out below to fix status fileds
	/*
		if cluster.Status.Restore.Backup == nil {
			initSpec := cluster.Spec.Restore
			backup, err := r.crController.GetRedisClusterBackup(initSpec.BackupSource.Namespace, initSpec.BackupSource.Name)
			if err != nil {
				reqLogger.Error(err, "GetRedisClusterBackup")
				return update, err
			}
			if backup.Status.Phase != redisv1alpha1.BackupPhaseSucceeded {
				reqLogger.Error(nil, "backup is still running")
				return update, fmt.Errorf("backup is still running")
			}
			cluster.Status.Restore.Backup = backup
			cluster.Status.Restore.Phase = redisv1alpha1.RestorePhaseRunning
			if err := r.crController.UpdateCRStatus(cluster); err != nil {
				return update, err
			}
		}
		backup := cluster.Status.Restore.Backup
		if cluster.Spec.Image == "" {
			cluster.Spec.Image = backup.Status.ClusterImage
			update = true
		}
		if cluster.Spec.MasterSize != backup.Status.MasterSize {
			cluster.Spec.MasterSize = backup.Status.MasterSize
			update = true
		}
	*/
	return update, nil
}

func (r *DistributedRedisClusterReconciler) waitForClusterJoin(ctx *syncContext, context context.Context, cluster *rediskunv1alpha1.DistributedRedisCluster) error {

	var firstNode *redisutil.Node
	for _, nodeInfo := range ctx.clusterInfos.Infos {
		firstNode = nodeInfo.Node
		break
	}
	ctx.reqLogger.Info(">>> Sending CLUSTER MEET messages to join the cluster")
	err := ctx.admin.AttachNodeToCluster(context, firstNode.IPPort(), cluster.Namespace, cluster.Name)
	if err != nil {
		r.MetricsClient.ReconcileCluster(cluster.Namespace, cluster.Name, "ATTACH_NODE_TO_CLUSTER", metrics.SYNC_CLUSTER, metrics.FAIL, err.Error())
		return Redis.Wrap(err, "AttachNodeToCluster")
	}
	// Give one second for the join to start, in order to avoid that
	// waiting for cluster join will find all the nodes agree about
	// the config as they are still empty with unassigned slots.
	time.Sleep(1 * time.Second)
	/*
		_, err = ctx.admin.GetClusterInfos(context)
		if err != nil {
			return Requeue.Wrap(err, "wait for cluster join")
		}
	*/
	return nil
}

func (r *DistributedRedisClusterReconciler) syncCluster(ctx *syncContext, context context.Context, instance *rediskunv1alpha1.DistributedRedisCluster) error {
	cluster := ctx.cluster
	admin := ctx.admin
	clusterInfos := ctx.clusterInfos
	expectMasterNum := cluster.Spec.MasterSize
	rCluster, nodes, err := newRedisCluster(clusterInfos, cluster, ctx.reqLogger)
	if err != nil {
		r.MetricsClient.ReconcileCluster(cluster.Namespace, cluster.Name, "NEW_REDIS_CLUSTER", metrics.SYNC_CLUSTER, metrics.FAIL, err.Error())
		return Cluster.Wrap(err, "newRedisCluster")
	}
	clusterCtx := clustering.NewCtx(cluster.Spec.HaConfig, rCluster, nodes, cluster.Spec.MasterSize, cluster.Spec.ClusterReplicas, cluster.Name, ctx.reqLogger)
	//Create a New Ranking Info Every time we need to place masters
	if ctx.clusterInfos.ProxyPodAz != "" {
		clusterCtx.SetProxyAz(ctx.clusterInfos.ProxyPodAz)
	}
	if err := clusterCtx.DispatchMasters(); err != nil {
		r.MetricsClient.ReconcileCluster(cluster.Namespace, cluster.Name, "DISPATCH_MASTER", metrics.SYNC_CLUSTER, metrics.FAIL, err.Error())
		return Cluster.Wrap(err, "DispatchMasters")
	}
	//Check if there is any masters that needs to be recovered from failure before we proceed with cluster operations
	/*
		if err := clusterCtx.CheckAndHealCluster(admin, context, instance.Namespace, instance.Name); err != nil {
			if clustering.ErrorType(GetType(err)) == clustering.HealReconcile {
				//We need alert for this metric if its hapenning continually for more than a minute
				r.MetricsClient.RecordRedisOperation(instance.Namespace, instance.Name, metrics.SELFHEAL_CLUSTER, "NA", metrics.SUCCESS, err.Error())
				return Requeue.Wrap(err, "HealCluster")
			}
			r.MetricsClient.RecordRedisOperation(instance.Namespace, instance.Name, metrics.SELFHEAL_CLUSTER, "NA", metrics.FAIL, err.Error())
			ctx.reqLogger.Error(err, "HealCluster", "Error while healing cluster", "Manual intervention required")
			return StopRetry.Wrap(err, "HealCluster")
		}
	*/
	curMasters := clusterCtx.GetCurrentMasters()
	newMasters := clusterCtx.GetNewMasters()
	ctx.reqLogger.Info("masters", "newMasters", len(newMasters), "curMasters", len(curMasters))
	//This if condition will be satisfied under the following scenarios
	//1. Cluster is being created for the first time
	//2. Cluster is recovering from a failure where all the masters are lost
	//In both the above scenarios, masters will be chosen based on the ranking info and slots will be allocated to them
	if len(curMasters) == 0 {
		ctx.reqLogger.Info("Creating cluster")
		if err := clusterCtx.PlaceSlaves(); err != nil {
			r.MetricsClient.ReconcileCluster(cluster.Namespace, cluster.Name, "PLACE_SLAVES", metrics.SYNC_CLUSTER, metrics.FAIL, err.Error())
			return Cluster.Wrap(err, "PlaceSlaves")

		}
		if err := clusterCtx.AttachingSlavesToMaster(admin, context, instance.Namespace, instance.Name); err != nil {
			r.MetricsClient.ReconcileCluster(cluster.Namespace, cluster.Name, "ATTACH_SLAVES_TO_MASTER", metrics.SYNC_CLUSTER, metrics.FAIL, err.Error())
			return Cluster.Wrap(err, "AttachingSlavesToMaster")
		}

		if err := clusterCtx.AllocSlots(admin, newMasters, context); err != nil {
			r.MetricsClient.ReconcileCluster(cluster.Namespace, cluster.Name, "ALLOCATE_SLOTS", metrics.SYNC_CLUSTER, metrics.FAIL, err.Error())
			return Cluster.Wrap(err, "AllocSlots")
		}
	} else if len(newMasters) > len(curMasters) {
		ctx.reqLogger.Info("Scaling up")
		if err := clusterCtx.PlaceSlaves(); err != nil {
			r.MetricsClient.ReconcileCluster(cluster.Namespace, cluster.Name, "PLACE_SLAVES", metrics.SYNC_CLUSTER, metrics.FAIL, err.Error())
			return Cluster.Wrap(err, "PlaceSlaves")

		}
		if err := clusterCtx.AttachingSlavesToMaster(admin, context, instance.Namespace, instance.Name); err != nil {
			r.MetricsClient.ReconcileCluster(cluster.Namespace, cluster.Name, "ATTACH_SLAVES_TO_MASTER", metrics.SYNC_CLUSTER, metrics.FAIL, err.Error())
			return Cluster.Wrap(err, "AttachingSlavesToMaster")
		}

		if err := clusterCtx.RebalancedCluster(admin, newMasters, context, instance.Namespace, instance.Name); err != nil {
			r.MetricsClient.ReconcileCluster(cluster.Namespace, cluster.Name, "REBALANCED_CLUSTER", metrics.SYNC_CLUSTER, metrics.FAIL, err.Error())
			return Cluster.Wrap(err, "RebalancedCluster")
		}
	} else if cluster.Status.MinReplicationFactor < cluster.Spec.ClusterReplicas {
		ctx.reqLogger.Info("Scaling slave")
		if err := clusterCtx.PlaceSlaves(); err != nil {
			r.MetricsClient.ReconcileCluster(cluster.Namespace, cluster.Name, "PLACE_SLAVES", metrics.SYNC_CLUSTER, metrics.FAIL, err.Error())
			return Cluster.Wrap(err, "PlaceSlaves")

		}
		if err := clusterCtx.AttachingSlavesToMaster(admin, context, instance.Namespace, instance.Name); err != nil {
			r.MetricsClient.ReconcileCluster(cluster.Namespace, cluster.Name, "ATTACH_SLAVES_TO_MASTER", metrics.SYNC_CLUSTER, metrics.FAIL, err.Error())
			return Cluster.Wrap(err, "AttachingSlavesToMaster")
		}
	} else if len(curMasters) > int(expectMasterNum) {
		ctx.reqLogger.Info("Scaling down")
		var allMaster redisutil.Nodes
		allMaster = append(allMaster, newMasters...)
		allMaster = append(allMaster, curMasters...)
		if err := clusterCtx.DispatchSlotToNewMasters(admin, newMasters, curMasters, allMaster, instance.Namespace, instance.Name, context); err != nil {
			r.MetricsClient.ReconcileCluster(cluster.Namespace, cluster.Name, "DISPATCH_SLOT_TO_NEW_MASTER", metrics.SYNC_CLUSTER, metrics.FAIL, err.Error())
			return err
		}
		if err := r.scalingDown(ctx, len(curMasters), clusterCtx.GetStatefulsetNodes(), context, instance.Namespace, instance.Name); err != nil {
			r.MetricsClient.ReconcileCluster(cluster.Namespace, cluster.Name, "SCALING_DOWN", metrics.SYNC_CLUSTER, metrics.FAIL, err.Error())
			return err
		}
	}
	return nil
}

func (r *DistributedRedisClusterReconciler) scalingDown(ctx *syncContext, currentMasterNum int, statefulSetNodes map[string]redisutil.Nodes, context context.Context, namespace string, name string) error {
	cluster := ctx.cluster
	SetClusterRebalancing(&cluster.Status,
		fmt.Sprintf("scale down, currentMasterSize: %d, expectMasterSize %d", currentMasterNum, cluster.Spec.MasterSize))
	r.CrController.UpdateCRStatus(cluster)
	admin := ctx.admin
	expectMasterNum := int(cluster.Spec.MasterSize)
	for i := currentMasterNum - 1; i >= expectMasterNum; i-- {
		stsName := statefulsets.ClusterStatefulSetName(cluster.Name, i)
		for _, node := range statefulSetNodes[stsName] {
			admin.Connections().Remove(context, node.IPPort())
		}
	}
	for i := currentMasterNum - 1; i >= expectMasterNum; i-- {
		stsName := statefulsets.ClusterStatefulSetName(cluster.Name, i)
		ctx.reqLogger.Info("scaling down", "statefulSet", stsName)
		sts, err := r.StatefulSetController.GetStatefulSet(cluster.Namespace, stsName)
		if err != nil {
			return Kubernetes.Wrap(err, "GetStatefulSet")
		}
		for _, node := range statefulSetNodes[stsName] {
			ctx.reqLogger.Info("forgetNode", "id", node.ID, "ip", node.IP, "role", node.GetRole())
			if len(node.Slots) > 0 {
				return Redis.New(fmt.Sprintf("node %s is not empty! Reshard data away and try again", node.String()))
			}
			if err := admin.ForgetNode(context, node.ID, namespace, name); err != nil {
				return Redis.Wrap(err, "ForgetNode")
			}
		}
		// remove resource
		if err := r.StatefulSetController.DeleteStatefulSetByName(cluster.Namespace, stsName); err != nil {
			ctx.reqLogger.Error(err, "DeleteStatefulSetByName", "statefulSet", stsName)
		}
		svcName := statefulsets.ClusterHeadlessSvcName(cluster.Name, i)
		if err := r.ServiceController.DeleteServiceByName(cluster.Namespace, svcName); err != nil {
			ctx.reqLogger.Error(err, "DeleteServiceByName", "service", svcName)
		}

		if err := r.PdbController.DeletePodDisruptionBudgetByName(cluster.Namespace, stsName); err != nil {
			ctx.reqLogger.Error(err, "DeletePodDisruptionBudgetByName", "pdb", stsName)
		}

		if err := r.PvcController.DeletePvcByLabels(cluster.Namespace, sts.Labels); err != nil {
			ctx.reqLogger.Error(err, "DeletePvcByLabels", "labels", sts.Labels)
		}
		// wait pod Terminating
		waiter := &waitPodTerminating{
			name:                  "waitPodTerminating",
			statefulSet:           stsName,
			timeout:               30 * time.Second * time.Duration(cluster.Spec.ClusterReplicas+2),
			tick:                  5 * time.Second,
			statefulSetController: r.StatefulSetController,
			cluster:               cluster,
		}
		if err := waiting(waiter, ctx.reqLogger); err != nil {
			ctx.reqLogger.Error(err, "waitPodTerminating")
		}

	}
	return nil
}

func (r *DistributedRedisClusterReconciler) checkandUpdatePassword(admin redisutil.IAdmin, ctx *syncContext, context context.Context, cluster *rediskunv1alpha1.DistributedRedisCluster) (map[string]string, error) {

	passChanged, newaclconf, newsecretver, err := r.Checker.CheckPasswordChanged(ctx.cluster)
	if err != nil {
		ctx.reqLogger.Error(err, "Error while checking difference in ACL")
		return nil, err
	}
	if !passChanged {
		ctx.reqLogger.Info("No changes in acl secrets, Proceeding..")
		return nil, nil
	}
	ctx.reqLogger.Info("Difference in ACL found, trying to update..")
	// Update configmap also is required here, so that when redis get restarted,
	// it would come up with expected acl or else it would come up with old acl which is a disaster.
	_, err2 := ctx.configMapClient.GetConfigMap(cluster.Namespace, configmaps.AclConfigMapName(cluster.Name))
	if err2 == nil {
		aclCmData, _, conferr, abortcr := configmaps.NewACLConfigMap(ctx.reqLogger, ctx.client, cluster, getLabels(cluster))
		if conferr != nil || abortcr {
			ctx.reqLogger.Error(conferr, "error while generating acl data for updating configmap")
			return nil, conferr
		}
		err3 := ctx.configMapClient.UpdateConfigMap(aclCmData)
		if err3 != nil {
			return nil, err3
		}
	} else {
		ctx.reqLogger.Error(err2, "error while getting ACL configmap")
		return nil, err2
	}
	SetClusterResetPassword(&ctx.cluster.Status, "Updating ACL's")
	r.CrController.UpdateCRStatus(ctx.cluster)

	if err := r.Checker.CheckRedisNodeNum(ctx.cluster); err != nil {
		return nil, err
	}
	aclcommands, err := statefulsets.GenerateAclcommands(ctx.reqLogger, r.Client, ctx.cluster, newaclconf, newsecretver)
	if err != nil {
		return newsecretver, err

	}

	for _, acl := range aclcommands {
		for addr, c := range admin.Connections().GetAll() {
			ctx.reqLogger.Info("Setting ACL for", "address", addr)
			cmd := strings.Fields(acl)
			args := make([]interface{}, len(cmd))
			for i, cm := range cmd {
				args[i] = cm
			}
			resp := c.Do(context, args...)
			if err := admin.Connections().ValidateResp(context, resp, addr, "unable to set ACL"); err != nil {
				r.MetricsClient.RecordRedisOperation(cluster.Namespace, cluster.Name, metrics.SET_ACL, addr, metrics.FAIL, err.Error())
				return nil, err
			}
		}
	}

	return newsecretver, nil
}

func (r *DistributedRedisClusterReconciler) checkandUpdatePods(admin redisutil.IAdmin, ctx *syncContext, context context.Context) (bool, error) {
	var ok, updated bool = false, false
	master_ips, slave_ips, err := r.getRedisIPsByRole(admin, ctx.reqLogger)
	if err != nil {
		return false, err
	}
	ssUR, err := r.getStatefulsetUpdatedRevision(ctx)

	var updateMaster, updateSlave bool = false, false

	// Check update annotations
	if val, exists := ctx.cluster.Annotations["update_master"]; exists && val == "true" {
		updateMaster = true
	}
	if val, exists := ctx.cluster.Annotations["update_slave"]; exists && val == "true" {
		updateSlave = true
	}
	// Handle slave updates
	for _, slave_ip := range slave_ips {
		ok, err = r.IsSlaveReady(slave_ip, admin, context)
		if err != nil {
			return false, err
		}

		if updateSlave {
			// Add force-reschedule annotation to slave pods
			client := k8sutil.NewPodController(r.Client, r.MetricsClient)
			for _, pod := range ctx.pods {
				if err := r.setRescheduleAnnotation(string(redisv1alpha1.RedisClusterNodeRoleSlave), pod, client); err != nil {
					return false, err
				}
			}

			// Remove the annotation from CR
			if err := r.removeUpdateAnnotationIfExists(ctx, context, "update_slave"); err != nil {
				return false, err
			}
		}

		if ok {
			updated, err = r.updatePodIfNeeded(slave_ip, ctx.pods, ssUR, ctx.reqLogger, ctx.cluster.Namespace, string(redisv1alpha1.RedisClusterNodeRoleSlave))
			if err != nil {
				return false, err
			}
		}
	}
	if updated {
		waiter := &waitStatefulSetUpdating{
			name:                  "waitStatefulSetUpdating",
			timeout:               30 * time.Second * time.Duration(ctx.cluster.Spec.ClusterReplicas),
			tick:                  5 * time.Second,
			statefulSetController: r.StatefulSetController,
			cluster:               ctx.cluster,
		}
		if err = waiting(waiter, ctx.reqLogger); err != nil {
			return false, err
		}
		ctx.reqLogger.Info("Slaves are updated")
		return true, err
	}
	if !updated && !ok {
		ctx.reqLogger.Info("Slaves are not ready")
		return true, err
	}

	// Handle master updates
	if ok {
		if updateMaster {
			// Add force-reschedule annotation to master pods
			client := k8sutil.NewPodController(r.Client, r.MetricsClient)
			for _, pod := range ctx.pods {
				if err := r.setRescheduleAnnotation(string(redisv1alpha1.RedisClusterNodeRoleMaster), pod, client); err != nil {
					return false, err
				}
			}

			// Remove the annotation from CR
			if err := r.removeUpdateAnnotationIfExists(ctx, context, "update_master"); err != nil {
				return false, err
			}
		}

		// Update masters one by one
		for _, master_ip := range master_ips {
			updated, err = r.updatePodIfNeeded(master_ip, ctx.pods, ssUR, ctx.reqLogger, ctx.cluster.Namespace, string(redisv1alpha1.RedisClusterNodeRoleMaster))
			if err != nil {
				return false, err
			}
			if updated {
				waiter := &waitStatefulSetUpdating{
					name:                  "waitStatefulSetUpdating",
					timeout:               30 * time.Second * time.Duration(ctx.cluster.Spec.ClusterReplicas),
					tick:                  5 * time.Second,
					statefulSetController: r.StatefulSetController,
					cluster:               ctx.cluster,
				}
				if err := waiting(waiter, ctx.reqLogger); err != nil {
					ctx.reqLogger.Info("Error waiting for master update")
					return false, err
				}
				ctx.reqLogger.Info("Master is updated", "master_ip", master_ip)
				return true, nil // Return after each master update to ensure one-by-one updates
			}
		}
	}
	return false, nil
}

func (r *DistributedRedisClusterReconciler) IsSlaveReady(ip string, admin redisutil.IAdmin, ctx context.Context) (bool, error) {
	c, err := admin.Connections().Get(ctx, ip)
	if err != nil {
		return false, err
	}
	info, err := c.Info(context.TODO(), "replication").Result()
	if err != nil {
		return false, err
	}

	ok := !strings.Contains(info, string(redisv1alpha1.RedisSyncing)) &&
		!strings.Contains(info, string(redisv1alpha1.RedisMasterSillPending)) &&
		strings.Contains(info, string(redisv1alpha1.RedisLinkUp))
	return ok, nil
}

func (r *DistributedRedisClusterReconciler) updatePodIfNeeded(ip string, pods []*corev1.Pod, revision_hashes []string, log logr.Logger, namespace string, role string) (bool, error) {
	if index := strings.Index(ip, ":"); index > 0 {
		ip = ip[:index]
	}
	for _, pod := range pods {
		if ip == pod.Status.PodIP {
			pattern := `shard(\d+)`
			re := regexp.MustCompile(pattern)
			stsNumStr := re.FindStringSubmatch(pod.Name)
			if len(stsNumStr) < 2 {
				log.Info("pattern not found or unable to extract shard number")
				return false, fmt.Errorf("pattern not found or unable to extract shard number")
			}
			stsIdentifier, err := strconv.Atoi(stsNumStr[1])
			if err != nil {
				log.Info("unable to convert shard number to int")
				return false, fmt.Errorf("unable to convert pod hash identifier to int")
			}

			// Check if force update is needed for this pod
			forceUpdate := false
			if pod.Annotations != nil {
				if _, exists := pod.Annotations["force-reschedule"]; exists {
					forceUpdate = true
				}
			}

			pod_hash := strings.Split((pod.ObjectMeta.Labels[v1.ControllerRevisionHashLabelKey]), "-")
			if len(pod_hash) < 3 {
				return false, fmt.Errorf("pod hash is invalid")
			}
			client := k8sutil.NewPodController(r.Client, r.MetricsClient)

			// Delete pod if hash is different or force update is requested
			if revision_hashes[stsIdentifier] != pod_hash[len(pod_hash)-1] || forceUpdate {
				err := client.DeletePodByName(namespace, pod.Name)
				if err != nil {
					return false, err
				}
				return true, err
			} else {
				if err := r.setRoleLabelIfNeeded(ip, pod, role, namespace, client); err != nil {
					return false, err
				}
				if err1 := r.setPodAnnotationIfNeeded(role, pod, client); err1 != nil {
					return false, err1
				}
				return false, nil
			}
		}
	}
	return false, fmt.Errorf("pod not found")
}

func (r *DistributedRedisClusterReconciler) setRoleLabelIfNeeded(ip string, pod *corev1.Pod, role string, namespace string, client k8sutil.IPodControl) error {
	for labelKey, labelValue := range pod.ObjectMeta.Labels {
		if labelKey == redisv1alpha1.RedisRoleLabelKey && labelValue == role {
			return nil
		}
	}
	return client.UpdatePodLabels(pod, redisv1alpha1.RedisRoleLabelKey, role)
}

// Function to handle pod annotation updates for specific roles
func (r *DistributedRedisClusterReconciler) setRescheduleAnnotation(role string, pod *corev1.Pod, client k8sutil.IPodControl) error {
	if pod.Labels["role"] == role {
		if err := client.UpdatePodAnnotations(pod, "force-reschedule", "true"); err != nil {
			return err
		}
	}
	return nil
}

// Function to remove annotations from cluster resource
func (r *DistributedRedisClusterReconciler) removeUpdateAnnotationIfExists(ctx *syncContext, context context.Context, annotationKey string) error {
	if _, exists := ctx.cluster.Annotations[annotationKey]; exists {
		delete(ctx.cluster.Annotations, annotationKey)
		if err := r.Update(context, ctx.cluster); err != nil {
			return err
		}
		ctx.reqLogger.Info(annotationKey + " annotation cleaned up")
	}
	return nil
}

func (r *DistributedRedisClusterReconciler) setPodAnnotationIfNeeded(role string, pod *corev1.Pod, client k8sutil.IPodControl) error {

	if role == string(redisv1alpha1.RedisClusterNodeRoleMaster) {
		client.UpdatePodAnnotations(pod, redisv1alpha1.PodEvictAnnotation, "false")
	} else if role == string(redisv1alpha1.RedisClusterNodeRoleSlave) {
		client.UpdatePodAnnotations(pod, redisv1alpha1.PodEvictAnnotation, "true")
	} else {
		err := fmt.Errorf("setPodAnnotationIfNeeded - invalid role")
		return err
	}
	return nil
}

func (r *DistributedRedisClusterReconciler) getRedisIPsByRole(admin redisutil.IAdmin, log logr.Logger) ([]string, []string, error) {
	var masterNodes, slaveNodes []string
	for addr, c := range admin.Connections().GetAll() {
		info, err := c.Info(context.TODO(), "Replication").Result()
		if err != nil {
			return nil, nil, err
		}
		if strings.Contains(info, "role:master") {
			masterNodes = append(masterNodes, addr)
		} else if strings.Contains(info, "role:slave") {
			slaveNodes = append(slaveNodes, addr)
		}
	}
	return masterNodes, slaveNodes, nil
}

func (r *DistributedRedisClusterReconciler) getStatefulsetUpdatedRevision(ctx *syncContext) ([]string, error) {
	cluster := ctx.cluster
	masterNum := int(cluster.Spec.MasterSize)
	var hashes []string
	for i := 0; i < masterNum; i++ {
		stsName := statefulsets.ClusterStatefulSetName(cluster.Name, i)
		sts, err := r.StatefulSetController.GetStatefulSet(cluster.Namespace, stsName)
		if err != nil {
			return hashes, Kubernetes.Wrap(err, "GetStatefulSet")
		}
		cssUR := strings.Split(sts.Status.UpdateRevision, "-")
		if len(cssUR) > 0 {
			hashes = append(hashes, cssUR[len(cssUR)-1])
		}
	}
	return hashes, nil
}
