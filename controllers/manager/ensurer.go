package manager

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	redisv1alpha1 "github.com/dinesh-murugiah/rediscluster-operator/api/v1alpha1"
	"github.com/dinesh-murugiah/rediscluster-operator/metrics"
	"github.com/dinesh-murugiah/rediscluster-operator/resources/configmaps"
	"github.com/dinesh-murugiah/rediscluster-operator/resources/poddisruptionbudgets"
	"github.com/dinesh-murugiah/rediscluster-operator/resources/services"
	"github.com/dinesh-murugiah/rediscluster-operator/resources/statefulsets"
	"github.com/dinesh-murugiah/rediscluster-operator/utils/k8sutil"
)

type IEnsureResource interface {
	EnsureRedisStatefulsets(cluster *redisv1alpha1.DistributedRedisCluster, labels map[string]string, update bool, proxyPodAz string) (redisv1alpha1.StsEnsureStatus, error)
	EnsureRedisPDB(cluster *redisv1alpha1.DistributedRedisCluster, labels map[string]string) error
	EnsureClusterShardsHeadLessSvcs(cluster *redisv1alpha1.DistributedRedisCluster, labels map[string]string) error
	EnsureClusterHeadLessSvc(cluster *redisv1alpha1.DistributedRedisCluster, labels map[string]string) error
	EnsureRedisConfigMap(cluster *redisv1alpha1.DistributedRedisCluster, labels map[string]string) error
	EnsureRedisRCloneSecret(cluster *redisv1alpha1.DistributedRedisCluster, labels map[string]string) error
	UpdateRedisStatefulsets(cluster *redisv1alpha1.DistributedRedisCluster, labels map[string]string, proxyPodAz string) error
}

type realEnsureResource struct {
	statefulSetClient k8sutil.IStatefulSetControl
	svcClient         k8sutil.IServiceControl
	configMapClient   k8sutil.IConfigMapControl
	pdbClient         k8sutil.IPodDisruptionBudgetControl
	crClient          k8sutil.ICustomResource
	client            client.Client
	logger            logr.Logger
	metricsRecorder   metrics.Recorder
}

func NewEnsureResource(client client.Client, logger logr.Logger, metricsRecorder metrics.Recorder) IEnsureResource {
	return &realEnsureResource{
		statefulSetClient: k8sutil.NewStatefulSetController(client, metricsRecorder),
		svcClient:         k8sutil.NewServiceController(client, metricsRecorder),
		configMapClient:   k8sutil.NewConfigMapController(client, metricsRecorder),
		pdbClient:         k8sutil.NewPodDisruptionBudgetController(client, metricsRecorder),
		crClient:          k8sutil.NewCRControl(client, metricsRecorder),
		client:            client,
		logger:            logger,
		metricsRecorder:   metricsRecorder,
	}
}
func (r *realEnsureResource) EnsureRedisPDB(cluster *redisv1alpha1.DistributedRedisCluster, labels map[string]string) error {
	if cluster.Spec.PdbEnabled {
		for i := 0; i < int(cluster.Spec.MasterSize); i++ {
			name := statefulsets.ClusterStatefulSetName(cluster.Name, i)
			labels[redisv1alpha1.StatefulSetLabel] = name
			labels[redisv1alpha1.ShardLabel] = strconv.Itoa(i)

			if err := r.ensureRedisPDB(cluster, name, labels); err != nil {
				return EnsurePDB.Wrap(err, "Ensure PDB failed")
			}

		}
	}
	return nil
}
func (r *realEnsureResource) EnsureRedisStatefulsets(cluster *redisv1alpha1.DistributedRedisCluster, labels map[string]string, update bool, proxyPodAz string) (redisv1alpha1.StsEnsureStatus, error) {

	var StsCreatedcount int = 0
	var StsUpdateCount int = 0
	var StsNoUpdateCount int = 0
	var ensureSTSStatus redisv1alpha1.StsEnsureStatus = redisv1alpha1.STSNotEnsured

	for i := 0; i < int(cluster.Spec.MasterSize); i++ {
		name := statefulsets.ClusterStatefulSetName(cluster.Name, i)
		svcName := statefulsets.ClusterHeadlessSvcName(cluster.Spec.ServiceName, i)
		// assign label
		labels[redisv1alpha1.StatefulSetLabel] = name
		labels[redisv1alpha1.ShardLabel] = strconv.Itoa(i)

		ensureSTSStatus, err := r.ensureRedisStatefulset(cluster, name, svcName, labels, update, proxyPodAz)
		if err != nil {
			return ensureSTSStatus, EnsureSTS.Wrap(err, "Ensure STS failed")
		} else {
			switch ensureSTSStatus {
			case redisv1alpha1.STSCreated:
				StsCreatedcount++
			case redisv1alpha1.STSUpdated:
				StsUpdateCount++
			case redisv1alpha1.STSNochange:
				StsNoUpdateCount++
			default:
				r.logger.WithValues("StatefulSet.Namespace", cluster.Namespace, "StatefulSet.Name", name).Error(fmt.Errorf("ensure sts unexpected status: %v", ensureSTSStatus), "Ensure STS failed")

			}
		}

	}

	//r.logger.WithValues("StatefulSet.Namespace", cluster.Namespace, "StatefulSet.Name", cluster.Name).Info("StatefulSet Ensure status", "Created", StsCreatedcount, "Updated", StsUpdateCount, "Nochange", StsNoUpdateCount)

	//if both creation and updation of sts is success then return created and updated status
	if StsCreatedcount > 0 && StsUpdateCount > 0 {
		ensureSTSStatus = redisv1alpha1.STSCreateUpadate
	} else if StsCreatedcount > 0 {
		//if only creation of sts is success and possibly along with nochange then return created status
		ensureSTSStatus = redisv1alpha1.STSCreated
	} else if StsUpdateCount > 0 {
		//if only updation of sts is success and possibly along with nochange then return updated status
		ensureSTSStatus = redisv1alpha1.STSUpdated
	}

	return ensureSTSStatus, nil

}

func (r *realEnsureResource) ensureRedisStatefulset(cluster *redisv1alpha1.DistributedRedisCluster, ssName, svcName string,
	labels map[string]string, update bool, proxyPodAz string) (redisv1alpha1.StsEnsureStatus, error) {

	_, err := r.statefulSetClient.GetStatefulSet(cluster.Namespace, ssName)
	if err == nil {
		if update {
			r.logger.WithValues("StatefulSet.Namespace", cluster.Namespace, "StatefulSet.Name", ssName).
				Info("updating statefulSet")
			newSS, err := statefulsets.NewStatefulSetForCR(cluster, ssName, svcName, labels, proxyPodAz)
			if err != nil {
				return redisv1alpha1.STSUpdateError, EnsureSTS.Wrap(err, "Error creating StatefulSet for Update")
			}
			ssErr := r.statefulSetClient.UpdateStatefulSet(newSS)
			if ssErr != nil {
				return redisv1alpha1.STSUpdateError, EnsureSTS.Wrap(ssErr, "Error updating StatefulSet")
			}

			return redisv1alpha1.STSUpdated, nil
		} else {
			//r.logger.WithValues("StatefulSet.Namespace", cluster.Namespace, "StatefulSet.Name", ssName).Info("statefulSet is up to date")
			return redisv1alpha1.STSNochange, nil
		}

	} else if err != nil && errors.IsNotFound(err) {
		r.logger.WithValues("StatefulSet.Namespace", cluster.Namespace, "StatefulSet.Name", ssName).
			Info("creating a new statefulSet")
		newSS, err := statefulsets.NewStatefulSetForCR(cluster, ssName, svcName, labels, proxyPodAz)
		if err != nil {
			return redisv1alpha1.STSCreateError, EnsureSTS.Wrap(err, "Error creating StatefulSet for Create")
		}
		ssErr := r.statefulSetClient.CreateStatefulSet(newSS)
		if ssErr != nil {
			return redisv1alpha1.STSCreateError, EnsureSTS.Wrap(ssErr, "Error creating StatefulSet")
		}
		return redisv1alpha1.STSCreated, nil
	}
	return redisv1alpha1.STSgetError, EnsureSTS.Wrap(err, "Error getting StatefulSet")
}

// func shouldUpdateRedis(cluster *redisv1alpha1.DistributedRedisCluster, sts *appsv1.StatefulSet) bool {
// 	if (cluster.Spec.ClusterReplicas + 1) != *sts.Spec.Replicas {
// 		return true
// 	}
// 	if cluster.Spec.Image != sts.Spec.Template.Spec.Containers[0].Image {
// 		return true
// 	}
// 	/* // TODO FIX it by adding a new user for probes
// 	if statefulsets.IsPasswordChanged(cluster, sts) {
// 		return true
// 	}
// 	*/
// 	expectResource := cluster.Spec.Resources
// 	currentResource := sts.Spec.Template.Spec.Containers[0].Resources
// 	if result := expectResource.Requests.Memory().Cmp(*currentResource.Requests.Memory()); result != 0 {
// 		return true
// 	}
// 	if result := expectResource.Requests.Cpu().Cmp(*currentResource.Requests.Cpu()); result != 0 {
// 		return true
// 	}
// 	if result := expectResource.Limits.Memory().Cmp(*currentResource.Limits.Memory()); result != 0 {
// 		return true
// 	}
// 	if result := expectResource.Limits.Cpu().Cmp(*currentResource.Limits.Cpu()); result != 0 {
// 		return true
// 	}
// 	return false
// }

// func shouldUpdateMonitor(cluster *redisv1alpha1.DistributedRedisCluster, sts *appsv1.StatefulSet) bool {
// 	for _, crspec := range *cluster.Spec.Monitor {
// 		for _, cspec := range sts.Spec.Template.Spec.Containers {
// 			if crspec.Name == cspec.Name {
// 				if crspec.Image != cspec.Image {
// 					return true
// 				}

// 				//compare arguments
// 				sort.Strings(crspec.Args)
// 				sort.Strings(cspec.Args)
// 				if !reflect.DeepEqual(crspec.Args, cspec.Args) {
// 					return true
// 				}

// 				// Checking resources
// 				expectedResource := crspec.Resources
// 				currentResource := cspec.Resources
// 				if result := expectedResource.Requests.Memory().Cmp(*currentResource.Requests.Memory()); result != 0 {
// 					return true
// 				}
// 				if result := expectedResource.Requests.Cpu().Cmp(*currentResource.Requests.Cpu()); result != 0 {
// 					return true
// 				}
// 				if result := expectedResource.Limits.Memory().Cmp(*currentResource.Limits.Memory()); result != 0 {
// 					return true
// 				}
// 				if result := expectedResource.Limits.Cpu().Cmp(*currentResource.Limits.Cpu()); result != 0 {
// 					return true
// 				}
// 			}
// 		}
// 	}
// 	return false
// }

func (r *realEnsureResource) ensureRedisPDB(cluster *redisv1alpha1.DistributedRedisCluster, name string, labels map[string]string) error {
	_, err := r.pdbClient.GetPodDisruptionBudget(cluster.Namespace, name)
	if err != nil && errors.IsNotFound(err) {
		r.logger.WithValues("PDB.Namespace", cluster.Namespace, "PDB.Name", name).
			Info("creating a new PodDisruptionBudget")
		pdb := poddisruptionbudgets.NewPodDisruptionBudgetForCR(cluster, name, labels)
		_ = r.pdbClient.CreatePodDisruptionBudget(pdb)
		return err
	}
	return err
}

func (r *realEnsureResource) EnsureClusterShardsHeadLessSvcs(cluster *redisv1alpha1.DistributedRedisCluster, labels map[string]string) error {
	for i := 0; i < int(cluster.Spec.MasterSize); i++ {
		svcName := statefulsets.ClusterHeadlessSvcName(cluster.Spec.ServiceName, i)
		name := statefulsets.ClusterStatefulSetName(cluster.Name, i)
		// assign label
		labels[redisv1alpha1.StatefulSetLabel] = name
		labels[redisv1alpha1.ShardLabel] = strconv.Itoa(i)
		if err := r.ensureRedisHeadLessSvc(cluster, svcName, labels); err != nil {
			return err
		}
	}
	return nil
}

func (r *realEnsureResource) ensureRedisHeadLessSvc(cluster *redisv1alpha1.DistributedRedisCluster, name string, labels map[string]string) error {
	_, err := r.svcClient.GetService(cluster.Namespace, name)
	if err != nil && errors.IsNotFound(err) {
		r.logger.WithValues("Service.Namespace", cluster.Namespace, "Service.Name", cluster.Spec.ServiceName).
			Info("creating a new headless service")
		svc := services.NewHeadLessSvcForCR(cluster, name, labels)
		err = r.svcClient.CreateService(svc)
		return err
	}
	return err
}

func (r *realEnsureResource) EnsureClusterHeadLessSvc(cluster *redisv1alpha1.DistributedRedisCluster, labels map[string]string) error {
	svcName := cluster.Spec.ServiceName
	if strings.HasPrefix(cluster.Spec.ServiceName, redisv1alpha1.CrResourcesprefix) {
		svcName = fmt.Sprintf("%s-%s-%s", redisv1alpha1.CrNameShort, cluster.Spec.ServiceName[len(redisv1alpha1.CrResourcesprefix):], "svc-cluster")
	}
	delete(labels, redisv1alpha1.StatefulSetLabel)
	delete(labels, redisv1alpha1.ShardLabel)
	_, err := r.svcClient.GetService(cluster.Namespace, svcName)
	if err != nil && errors.IsNotFound(err) {
		r.logger.WithValues("Service.Namespace", cluster.Namespace, "Service.Name", cluster.Spec.ServiceName).
			Info("creating a new service")
		svc := services.NewHeadLessSvcForCR(cluster, svcName, labels)
		err = r.svcClient.CreateService(svc)
		return err
	}
	return err
}

func (r *realEnsureResource) EnsureRedisConfigMap(cluster *redisv1alpha1.DistributedRedisCluster, labels map[string]string) error {

	cmName := configmaps.RedisConfigMapName(cluster.Name)
	drcCm, err := r.configMapClient.GetConfigMap(cluster.Namespace, cmName)
	if err != nil {
		if errors.IsNotFound(err) {
			r.logger.WithValues("ConfigMap.Namespace", cluster.Namespace, "ConfigMap.Name", cmName).
				Info("creating a new configMap")
			cm := configmaps.NewConfigMapForCR(cluster, labels)

			err2 := r.configMapClient.CreateConfigMap(cm)
			if err2 != nil {
				return Requeue.Wrap(err2, "Error creating ConfigMap")
			}

		} else {
			return Requeue.Wrap(err, "Error getting ConfigMap")
		}
	} else {
		if isRedisConfChanged(drcCm.Data[configmaps.RedisConfKey], cluster.Spec.Config, r.logger) {
			cm := configmaps.NewConfigMapForCR(cluster, labels)
			err2 := r.configMapClient.UpdateConfigMap(cm)
			if err2 != nil {
				return Requeue.Wrap(err2, "Error updating ConfigMap")
			}
		}
	}
	aclcmName := configmaps.AclConfigMapName(cluster.Name)
	_, err2 := r.configMapClient.GetConfigMap(cluster.Namespace, aclcmName)
	if err2 != nil {
		if errors.IsNotFound(err2) {
			r.logger.WithValues("ConfigMap.Namespace", cluster.Namespace, "ConfigMap.Name", cmName).
				Info("creating a new ACL configMap")
			aclcm, _, conferr, abortcr := configmaps.NewACLConfigMap(r.logger, r.client, cluster, labels)
			if conferr != nil {
				if abortcr {
					StopRetry.Wrap(conferr, "Error generating ACL ConfigMap - Secret not found")
				} else {
					return Requeue.Wrap(conferr, "Error generating ACL ConfigMap - retrying")
				}

			}
			err3 := r.configMapClient.CreateConfigMap(aclcm)
			if err3 != nil {
				return Requeue.Wrap(err3, "Error creating ACL ConfigMap")
			}

		} else {
			return Requeue.Wrap(err2, "Error getting ACL ConfigMap")
		}
	}
	//Restore from backup is not supported yet
	/*
		if cluster.IsRestoreFromBackup() {
			restoreCmName := configmaps.RestoreConfigMapName(cluster.Name)
			restoreCm, err := r.configMapClient.GetConfigMap(cluster.Namespace, restoreCmName)
			k8sutil.RecordMetrics(cluster.Namespace, "ConfigMap", restoreCmName, "GET", err2, r.metricsRecorder)
			if err != nil {
				if errors.IsNotFound(err) {
					r.logger.WithValues("ConfigMap.Namespace", cluster.Namespace, "ConfigMap.Name", restoreCmName).
						Info("creating a new restore configMap")
					cm := configmaps.NewConfigMapForRestore(cluster, labels)
					err1 := r.configMapClient.CreateConfigMap(cm)
					k8sutil.RecordMetrics(cluster.Namespace, "ConfigMap", cm.Name, "CREATE", err1, r.metricsRecorder)
					if err1 != nil {
						return err1, false
					}
				} else {
					return err, false
				}
			}
			if cluster.Status.Restore.Phase == redisv1alpha1.RestorePhaseRestart && restoreCm.Data[configmaps.RestoreSucceeded] == "0" {
				cm := configmaps.NewConfigMapForRestore(cluster, labels)
				err := r.configMapClient.UpdateConfigMap(cm)
				k8sutil.RecordMetrics(cluster.Namespace, "ConfigMap", cm.Name, "UPDATE", err, r.metricsRecorder)
				if err != nil {
					return err, false
				}
			}
		}
	*/
	return nil
}

func (r *realEnsureResource) EnsureRedisRCloneSecret(cluster *redisv1alpha1.DistributedRedisCluster, labels map[string]string) error {
	if !cluster.IsRestoreFromBackup() || cluster.IsRestored() {
		return nil
	}
	//Dinesh Todo This flow Needs FIX ####, commented out below to fix status fileds
	err := fmt.Errorf("restore Not supported yet")
	return err
	/*
		backup := cluster.Status.Restore.Backup
		secret, err := osm.NewRcloneSecret(r.client, backup.RCloneSecretName(), cluster.Namespace, backup.Spec.Backend, redisv1alpha1.DefaultOwnerReferences(cluster))
		if err != nil {
			return err
		}
		if err := k8sutil.CreateSecret(r.client, secret, r.logger); err != nil {
			return err
		}
		return nil
	*/
}

func isRedisConfChanged(confInCm string, currentConf map[string]string, log logr.Logger) bool {
	lines := strings.Split(strings.TrimSuffix(confInCm, "\n"), "\n")
	if len(lines) != len(currentConf) {
		return true
	}
	for _, line := range lines {
		line = strings.TrimSuffix(line, " ")
		confLine := strings.SplitN(line, " ", 2)
		if len(confLine) == 2 {
			if valueInCurrentConf, ok := currentConf[confLine[0]]; !ok {
				return true
			} else {
				if valueInCurrentConf != confLine[1] {
					return true
				}
			}
		} else {
			log.Info("custom config is invalid", "raw", line, "split", confLine)
		}
	}
	return false
}

func (r *realEnsureResource) UpdateRedisStatefulsets(cluster *redisv1alpha1.DistributedRedisCluster, labels map[string]string, proxyPodAz string) error {
	for i := 0; i < int(cluster.Spec.MasterSize); i++ {
		name := statefulsets.ClusterStatefulSetName(cluster.Name, i)
		svcName := statefulsets.ClusterHeadlessSvcName(cluster.Spec.ServiceName, i)
		// assign label
		labels[redisv1alpha1.StatefulSetLabel] = name
		labels[redisv1alpha1.ShardLabel] = strconv.Itoa(i)
		if err := r.updateRedisStatefulset(cluster, name, svcName, labels, proxyPodAz); err != nil {
			return err
		}
	}
	return nil
}

func (r *realEnsureResource) updateRedisStatefulset(cluster *redisv1alpha1.DistributedRedisCluster, ssName, svcName string,
	labels map[string]string, proxyPodAz string) error {
	r.logger.WithValues("StatefulSet.Namespace", cluster.Namespace, "StatefulSet.Name", ssName).
		Info("updating statefulSet immediately")
	newSS, err := statefulsets.NewStatefulSetForCR(cluster, ssName, svcName, labels, proxyPodAz)
	if err != nil {
		return err
	}
	err = r.statefulSetClient.UpdateStatefulSet(newSS)
	return err
}
