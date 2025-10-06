/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package distributedrediscluster

import (
	"context"
	"fmt"
	"sort"
	"time"

	rediskunv1alpha1 "github.com/dinesh-murugiah/rediscluster-operator/api/v1alpha1"
	"github.com/dinesh-murugiah/rediscluster-operator/controllers/clustering"
	"github.com/dinesh-murugiah/rediscluster-operator/controllers/heal"
	clustermanger "github.com/dinesh-murugiah/rediscluster-operator/controllers/manager"
	"github.com/dinesh-murugiah/rediscluster-operator/metrics"
	config "github.com/dinesh-murugiah/rediscluster-operator/redisconfig"
	"github.com/dinesh-murugiah/rediscluster-operator/resources/statefulsets"
	utils "github.com/dinesh-murugiah/rediscluster-operator/utils/commonutils"
	"github.com/dinesh-murugiah/rediscluster-operator/utils/exec"
	"github.com/dinesh-murugiah/rediscluster-operator/utils/k8sutil"
	"github.com/dinesh-murugiah/rediscluster-operator/utils/redisutil"
	"github.com/go-logr/logr"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	logl = log.Log.WithName("controller_distributedrediscluster")

	controllerFlagSet *pflag.FlagSet
	// maxConcurrentReconciles is the maximum number of concurrent Reconciles which can be run. Defaults to 4.
	maxConcurrentReconciles int
	// reconcileTime is the delay between reconciliations. Defaults to 60s.
	reconcileTime int

	updatePredicateCalled = false

	pred predicate.Funcs
)

func init() {
	controllerFlagSet = pflag.NewFlagSet("controller", pflag.ExitOnError)
	controllerFlagSet.IntVar(&maxConcurrentReconciles, "ctr-maxconcurrent", 4, "the maximum number of concurrent Reconciles which can be run. Defaults to 4.")
	controllerFlagSet.IntVar(&reconcileTime, "ctr-reconciletime", 60, "")
}

func FlagSet() *pflag.FlagSet {
	return controllerFlagSet
}

func getProxyLabels(namespace string) map[string]string {
	return map[string]string{
		"app":       "redis-proxy",
		"service":   namespace,
		"component": "redis-cluster",
	}
}

// DistributedRedisClusterReconciler reconciles a DistributedRedisCluster object
type DistributedRedisClusterReconciler struct {
	client.Client
	Scheme                *runtime.Scheme
	Ensurer               clustermanger.IEnsureResource
	Checker               clustermanger.ICheck
	Execer                exec.IExec
	StatefulSetController k8sutil.IStatefulSetControl
	ServiceController     k8sutil.IServiceControl
	PdbController         k8sutil.IPodDisruptionBudgetControl
	PvcController         k8sutil.IPvcControl
	CrController          k8sutil.ICustomResource
	PodController         k8sutil.IPodControl
	NodeController        k8sutil.INodeControl
	MetricsClient         metrics.Recorder
}

//+kubebuilder:rbac:groups=redis.kun,resources=distributedredisclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=redis.kun,resources=distributedredisclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=redis.kun,resources=distributedredisclusters/finalizers,verbs=update
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;create;update

func redisclustercontrollerPredfunction() predicate.Funcs {

	pred = predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// returns false if DistributedRedisCluster is ignored (not managed) by this operator.
			if !utils.ShoudManage(e.ObjectNew) {
				return false
			}
			logl.WithValues("namespace", e.ObjectNew.GetNamespace(), "name", e.ObjectNew.GetName()).V(5).Info("Call UpdateFunc")
			// Ignore updates to CR status in which case metadata.Generation does not change
			if e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration() {
				logl.WithValues("namespace", e.ObjectNew.GetNamespace(), "name", e.ObjectNew.GetName()).Info("Generation change return true",
					"old", e.ObjectOld, "new", e.ObjectNew)
				updatePredicateCalled = true
				return true
			}
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// returns false if DistributedRedisCluster is ignored (not managed) by this operator.
			if !utils.ShoudManage(e.Object) {
				return false
			}
			logl.WithValues("namespace", e.Object.GetNamespace(), "name", e.Object.GetName()).Info("Call DeleteFunc")
			// Evaluates to false if the object has been confirmed deleted.
			return !e.DeleteStateUnknown
		},
		CreateFunc: func(e event.CreateEvent) bool {
			// returns false if DistributedRedisCluster is ignored (not managed) by this operator.
			if !utils.ShoudManage(e.Object) {
				return false
			}
			logl.WithValues("namespace", e.Object.GetNamespace(), "name", e.Object.GetName()).Info("Call CreateFunc")
			return true
		},
	}
	return pred
}

// SetupWithManager sets up the controller with the Manager.
func (r *DistributedRedisClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&rediskunv1alpha1.DistributedRedisCluster{}, builder.WithPredicates(redisclustercontrollerPredfunction())).
		WithOptions(controller.Options{MaxConcurrentReconciles: maxConcurrentReconciles}).
		Complete(r)
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DistributedRedisCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *DistributedRedisClusterReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	reqLogger := logl.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling DistributedRedisCluster")
	r.MetricsClient.ReconcileClusterGauge(request.Namespace, request.Name, "RECONCILE", float64(0))

	// Fetch the DistributedRedisCluster instance
	instance := &rediskunv1alpha1.DistributedRedisCluster{}
	err := r.Client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		r.MetricsClient.ReconcileCluster(request.Namespace, request.Name, "GET_DRC_CR", metrics.RECONCILE, metrics.FAIL, err.Error())
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	isSingleAzDeployment := verifyIfSingleAzDeployment(instance)
	reqLogger.V(4).Info("HA configuration check", "isSingleAzDeployment", isSingleAzDeployment, "haConfig", instance.Spec.HaConfig != nil)

	syncCtx := &syncContext{
		cluster:              instance,
		reqLogger:            reqLogger,
		client:               r.Client,
		configMapClient:      k8sutil.NewConfigMapController(r.Client, r.MetricsClient),
		isSingleAzDeployment: isSingleAzDeployment,
	}
	var STSEnsureStatus rediskunv1alpha1.StsEnsureStatus = rediskunv1alpha1.STSNotEnsured
	var haStatus rediskunv1alpha1.HaStatus = rediskunv1alpha1.HaStatusUnknown
	STSEnsureStatus, err = r.ensureCluster(syncCtx, updatePredicateCalled)

	// Make sure to update the predicate called flag to false
	updatePredicateCalled = false
	if err != nil {
		r.MetricsClient.ReconcileClusterGauge(request.Namespace, request.Name, "ENSURE_CLUSTER", float64(1))
		switch GetType(err) {
		case StopRetry:
			r.MetricsClient.ReconcileCluster(request.Namespace, request.Name, "ENSURE_CLUSTER", metrics.ENSURE, metrics.FATAL, err.Error())
			reqLogger.Info("error in ensureCluster, stopping reconcile for this CR", "err", err)
			return reconcile.Result{}, nil
		}
		reqLogger.WithValues("err", err).Info("ensureCluster")
		newStatus := instance.Status.DeepCopy()
		UpdateHaStatus(instance, newStatus, "", STSEnsureStatus)
		reqLogger.Info("ensureCluster", "STSEnsureStatus", STSEnsureStatus, "HaStatus", newStatus.HAStatus)
		SetClusterScaling(newStatus, err.Error())
		r.updateClusterIfNeed(instance, newStatus, reqLogger)
		r.MetricsClient.ReconcileCluster(request.Namespace, request.Name, "ENSURE_CLUSTER", metrics.RECONCILE, metrics.FAIL, err.Error())
		return reconcile.Result{RequeueAfter: requeueAfter}, nil
	} else {
		//var newStatus *rediskunv1alpha1.DistributedRedisClusterStatus
		newStatus := instance.Status.DeepCopy()
		err := UpdateSecretsStatus(instance, newStatus, r.Client, reqLogger)
		if err != nil {
			r.MetricsClient.ReconcileCluster(request.Namespace, request.Name, "GET_SECRETVERSION", metrics.ENSURE, metrics.FAIL, err.Error())
		}
		UpdateHaStatus(instance, newStatus, "", STSEnsureStatus)
		reqLogger.Info("ensureCluster", "STSEnsureStatus", STSEnsureStatus, "HaStatus", newStatus.HAStatus)
		r.updateClusterIfNeed(instance, newStatus, reqLogger)
	}
	r.MetricsClient.ReconcileClusterGauge(request.Namespace, request.Name, "ENSURE_CLUSTER", float64(0))

	matchLabels := getLabels(instance)
	redisClusterPods, err := r.StatefulSetController.GetStatefulSetPodsByLabels(instance.Namespace, matchLabels)
	if err != nil {
		r.MetricsClient.ReconcileCluster(request.Namespace, request.Name, "GET_STS_PODS_BY_LABELS", metrics.RECONCILE, metrics.FAIL, err.Error())
		return reconcile.Result{}, Kubernetes.Wrap(err, "GetStatefulSetPods")
	}

	syncCtx.pods = clusterPods(redisClusterPods.Items)
	reqLogger.V(6).Info("debug cluster pods", "", syncCtx.pods)

	proxyPodErrCheck := proxyPods(syncCtx, r, request)
	if proxyPodErrCheck != nil {
		r.MetricsClient.ReconcileCluster(request.Namespace, request.Name, "GET_PROXY_PODS_BY_LABELS", metrics.RECONCILE, metrics.FAIL, proxyPodErrCheck.Error())
		return reconcile.Result{}, Kubernetes.Wrap(err, "GetProxyPods")
	}

	syncCtx.healer = clustermanger.NewHealer(&heal.CheckAndHeal{
		Logger:        reqLogger,
		PodControl:    k8sutil.NewPodController(r.Client, r.MetricsClient),
		Pods:          syncCtx.pods,
		DryRun:        false,
		MetricsClient: r.MetricsClient,
	})
	err = r.waitPodReady(syncCtx)
	if err != nil {
		r.MetricsClient.ReconcileClusterGauge(request.Namespace, request.Name, "WAIT_POD_READY", float64(1))
		// note: reconcilation failure will not happen here, it will continue to wait until pod is ready
		r.MetricsClient.ReconcileCluster(request.Namespace, request.Name, "WAIT_FOR_POD_READY", metrics.RECONCILE, metrics.FAIL, err.Error())
		switch GetType(err) {
		case Kubernetes:
			return reconcile.Result{}, err
		}
		reqLogger.WithValues("err", err).Info("waitPodReady")
		newStatus := instance.Status.DeepCopy()
		SetClusterScaling(newStatus, err.Error())
		r.updateClusterIfNeed(instance, newStatus, reqLogger)
		return reconcile.Result{RequeueAfter: requeueAfter}, nil
	}
	r.MetricsClient.ReconcileClusterGauge(request.Namespace, request.Name, "WAIT_POD_READY", float64(0))

	password, err := statefulsets.GetRedisClusterAdminPassword(r.Client, syncCtx.cluster, reqLogger)
	if err != nil {
		//we need to record 2 metrics here, one for k8s operation and another for reconcile failure
		r.MetricsClient.ReconcileCluster(request.Namespace, request.Name, "GET_CLUSTER_PASSWORD", metrics.RECONCILE, metrics.FAIL, err.Error())
		r.MetricsClient.RecordK8sOperation(request.Namespace, "DistributedRedisCluster", request.Name, "GET_CLUSTER_PASSWORD", metrics.FAIL, err.Error())
		return reconcile.Result{}, Kubernetes.Wrap(err, "getClusterPassword")
	}

	context := context.Background()

	admin, err := newRedisAdmin(syncCtx.pods, password, config.RedisConf(), reqLogger, context, r.MetricsClient)
	if err != nil {
		// we need to record 2 metrics here, one for redis operation and another for reconcile failure
		r.MetricsClient.ReconcileCluster(request.Namespace, request.Name, metrics.CREATE_ADMIN_CONN, metrics.RECONCILE, metrics.FAIL, err.Error())
		r.MetricsClient.RecordRedisOperation(request.Namespace, request.Name, metrics.CREATE_ADMIN_CONN, "NA", metrics.FAIL, err.Error())
		return reconcile.Result{}, Redis.Wrap(err, "newRedisAdmin")
	}
	defer admin.Close(context)

	secretver, err := r.checkandUpdatePassword(admin, syncCtx, context, instance)
	newsecretStatus := instance.Status.DeepCopy()
	if err != nil {
		r.MetricsClient.ReconcileCluster(request.Namespace, request.Name, "CHECK_UPDATE_PASSWORD", metrics.RECONCILE, metrics.FAIL, err.Error())
		reqLogger.WithValues("err", err).Info("error checking / updating password")
		SetSecretStatus(newsecretStatus, "ACLupdateERR")
	}
	if secretver != nil {
		SetSecretStatus(newsecretStatus, "ACLupdateSuccess")
		newsecretStatus.SecretsVer = make(map[string]string)
		for key, value := range secretver {
			newsecretStatus.SecretsVer[key] = value
		}
	} else {
		if err == nil {
			SetSecretStatus(newsecretStatus, "ACLintact")
		}
	}
	r.updateClusterIfNeed(instance, newsecretStatus, reqLogger)
	//reqLogger.WithValues("adminpassword:", password).Info("please check adminpass")
	// Setting this context for request flow

	healClusterInfo, infoErr := admin.GetClusterInfos(context, request.Namespace, request.Name)
	if infoErr != nil {
		if clusterErr, ok := infoErr.(redisutil.ClusterInfosError); ok {
			r.MetricsClient.ReconcileCluster(request.Namespace, request.Name, "BUILD_CLUSTER_INFO", metrics.RECONCILE, metrics.FAIL, infoErr.Error())
			if clusterErr.Partial() {
				reqLogger.Info("GetClusterInfos", "clusterInfos Error", clusterErr)
				return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
			}
		}
	}
	proxyAz, err := findProxyAz(r, syncCtx)
	if err != nil {
		r.MetricsClient.ReconcileCluster(request.Namespace, request.Name, "FIND_PROXY_AZ", metrics.RECONCILE, metrics.FAIL, err.Error())
		reqLogger.Error(err, "findProxyAz", "Error when finding proxy az", err)
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	updateErr := UpdateClusterInfos(healClusterInfo, syncCtx.pods, r.Client, reqLogger, r.MetricsClient)
	if updateErr != nil {
		r.MetricsClient.ReconcileCluster(request.Namespace, request.Name, "UPDATE_CLUSTER_INFO", metrics.RECONCILE, metrics.FAIL, updateErr.Error())
		reqLogger.Error(updateErr, "UpdateClusterInfos", "Error when updating cluster infos", updateErr)
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// check if the cluster is in a state where it needs healing
	stopReconcile, requeue, err := syncCtx.healer.Heal(instance, healClusterInfo, admin, context)
	if err != nil {
		r.MetricsClient.ReconcileClusterGauge(request.Namespace, request.Name, "HEAL", float64(1))
		if stopReconcile {
			r.MetricsClient.ReconcileCluster(request.Namespace, request.Name, "HEAL", metrics.RECONCILE, metrics.FATAL, err.Error())
			reqLogger.Error(err, "StopReconcile", "Irrecoverable Error Manual Heal Required ", err)
			return reconcile.Result{}, nil
		}
		r.MetricsClient.ReconcileCluster(request.Namespace, request.Name, "HEAL", metrics.RECONCILE, metrics.FAIL, err.Error())
		return reconcile.Result{RequeueAfter: 20 * time.Second}, nil

	}
	if requeue {
		return reconcile.Result{RequeueAfter: requeueAfter}, nil
	}
	r.MetricsClient.ReconcileClusterGauge(request.Namespace, request.Name, "HEAL", float64(0))

	clusterInfos, clusterinfoerr := admin.GetClusterInfos(context, request.Namespace, request.Name)
	if clusterinfoerr != nil {
		reqLogger.V(6).Info("debug waitForClusterJoin", "cluster infos", syncCtx.clusterInfos)
		clusterErr := clusterinfoerr.(redisutil.ClusterInfosError)
		if clusterErr.Partial() {
			r.MetricsClient.ReconcileCluster(request.Namespace, request.Name, "BUILD_CLUSTER_INFO_PARTIAL", metrics.RECONCILE, metrics.FAIL, clusterinfoerr.Error())
			reqLogger.Info("GetClusterInfos", "clusterInfos Error", clusterinfoerr)
			return reconcile.Result{RequeueAfter: requeueAfter}, nil
		} else if clusterErr.Inconsistent() {
			r.MetricsClient.ReconcileCluster(request.Namespace, request.Name, "BUILD_CLUSTER_INFO_INCONSISTENT", metrics.RECONCILE, metrics.FAIL, clusterinfoerr.Error())
			reqLogger.Info("ClusterInfosError", "Cluster View inconsistent", "calling waitForClusterJoin")
			syncCtx.admin = admin
			syncCtx.clusterInfos = clusterInfos
			if err := r.waitForClusterJoin(syncCtx, context, instance); err != nil {
				r.MetricsClient.ReconcileCluster(request.Namespace, request.Name, metrics.CLUSTER_JOIN_FAILURE, metrics.RECONCILE, metrics.FAIL, err.Error())
				newStatus := instance.Status.DeepCopy()
				SetClusterFailed(newStatus, err.Error())
				r.updateClusterIfNeed(instance, newStatus, reqLogger)
				return reconcile.Result{}, err
			}

			if clusterInfos, err = syncCtx.admin.GetClusterInfos(context, request.Namespace, request.Name); err != nil || clusterInfos.Status == redisutil.ClusterInfosPartial {
				r.MetricsClient.ReconcileCluster(request.Namespace, request.Name, metrics.REQUEUE, metrics.RECONCILE, metrics.FAIL, err.Error())
				reqLogger.Info("Cluster view remains inconsistent or partial after waiting for join", "error", err)
				return reconcile.Result{RequeueAfter: requeueAfter}, nil
			}

		} else {
			r.MetricsClient.ReconcileCluster(request.Namespace, request.Name, "GET_CLUSTER_INFO", metrics.RECONCILE, metrics.FAIL, clusterinfoerr.Error())
			return reconcile.Result{}, Redis.Wrap(clusterinfoerr, "GetClusterInfos")
		}
	}

	syncCtx.admin = admin
	syncCtx.clusterInfos = clusterInfos
	if proxyAz != "" {
		clusterInfos.ProxyPodAz = proxyAz
	}

	err1 := UpdateClusterInfos(clusterInfos, syncCtx.pods, r.Client, reqLogger, r.MetricsClient)
	if err1 != nil {
		r.MetricsClient.ReconcileCluster(request.Namespace, request.Name, "UPDATE_CLUSTER_INFO", metrics.RECONCILE, metrics.FAIL, err1.Error())
		reqLogger.Error(err1, "UpdateClusterInfos", "Error when updating cluster infos", err1)
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}
	// mark .Status.Restore.Phase = RestorePhaseRestart, will
	// remove init container and restore volume that referenced in stateulset for
	// dump RDB file from backup, then the redis master node will be restart.
	if instance.IsRestoreFromBackup() && instance.IsRestoreRunning() {
		reqLogger.Info("update restore redis cluster cr")
		instance.Status.Restore.Phase = rediskunv1alpha1.RestorePhaseRestart
		if err := r.CrController.UpdateCRStatus(instance); err != nil {
			r.MetricsClient.ReconcileCluster(request.Namespace, request.Name, "UPDATE_CR_STATUS", metrics.RECONCILE, metrics.FAIL, err.Error())
			return reconcile.Result{}, err
		}
		if err := r.Ensurer.UpdateRedisStatefulsets(instance, getLabels(instance)); err != nil {
			r.MetricsClient.ReconcileCluster(request.Namespace, request.Name, "UPDATE_REDIS_STATEFULSETS", metrics.RECONCILE, metrics.FAIL, err.Error())
			return reconcile.Result{}, err
		}
		waiter := &waitStatefulSetUpdating{
			name:                  "waitMasterNodeRestarting",
			timeout:               60 * time.Second,
			tick:                  5 * time.Second,
			statefulSetController: r.StatefulSetController,
			cluster:               instance,
		}
		if err := waiting(waiter, syncCtx.reqLogger); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	}

	// restore succeeded, then update cr and wait for the next Reconcile loop
	if instance.IsRestoreFromBackup() && instance.IsRestoreRestarting() {
		reqLogger.Info("update restore redis cluster cr")
		instance.Status.Restore.Phase = rediskunv1alpha1.RestorePhaseSucceeded
		if err := r.CrController.UpdateCRStatus(instance); err != nil {
			r.MetricsClient.ReconcileCluster(request.Namespace, request.Name, "UPDATE_CR_STATUS", metrics.RECONCILE, metrics.FAIL, err.Error())
			return reconcile.Result{}, err
		}
		// set ClusterReplicas = Backup.Status.ClusterReplicas,
		// next Reconcile loop the statefulSet's replicas will increase by ClusterReplicas, then start the slave node
		//Dinesh Todo This flow Needs FIX ####, commented out below to fix status fileds
		/*
			instance.Spec.ClusterReplicas = instance.Status.Restore.Backup.Status.ClusterReplicas
			if err := r.crController.UpdateCR(instance); err != nil {
				return reconcile.Result{}, err
			}
		*/
		return reconcile.Result{}, nil
	}

	if err := admin.SetConfigIfNeed(context, instance.Spec.Config, instance.Namespace, instance.Name); err != nil {
		r.MetricsClient.ReconcileCluster(request.Namespace, request.Name, "SET_CONFIG", metrics.RECONCILE, metrics.FAIL, err.Error())
		return reconcile.Result{}, Redis.Wrap(err, "SetConfigIfNeed")
	}

	status := buildClusterStatus(clusterInfos, syncCtx.pods, instance, reqLogger)
	if is := r.isScalingDown(instance, reqLogger); is {
		SetClusterRebalancing(status, "scaling down")
	}
	reqLogger.V(4).Info("buildClusterStatus", "status", status)
	r.updateClusterIfNeed(instance, status, reqLogger)

	instance.Status = *status
	if needClusterOperation(instance, reqLogger) {
		reqLogger.Info(">>>>>> clustering")
		err = r.syncCluster(syncCtx, context, instance)
		if err != nil {
			r.MetricsClient.ReconcileClusterGauge(request.Namespace, request.Name, "SYNC_CLUSTER", float64(1))
			if GetType(err) == Requeue {
				r.MetricsClient.ReconcileCluster(instance.Namespace, instance.Name, metrics.CLUSTERING, metrics.RECONCILE, metrics.FAIL, err.Error())
				reqLogger.Info("syncCluster", "Requeue - Possibly for Trying to self heal the cluster", err)
				return reconcile.Result{RequeueAfter: requeueAfter}, nil
			} else if GetType(err) == StopRetry {
				r.MetricsClient.ReconcileCluster(instance.Namespace, instance.Name, metrics.CLUSTERING, metrics.RECONCILE, metrics.FATAL, err.Error())
				reqLogger.Error(err, "StopReconcile", "Irrecoverable Error Manual Heal Required ", err)
				return reconcile.Result{}, nil
			}
			r.MetricsClient.ReconcileCluster(instance.Namespace, instance.Name, metrics.CLUSTERING, metrics.RECONCILE, metrics.FAIL, err.Error())
			newStatus := instance.Status.DeepCopy()
			SetClusterFailed(newStatus, err.Error())
			r.updateClusterIfNeed(instance, newStatus, reqLogger)
			return reconcile.Result{}, err
		}
	}
	r.MetricsClient.ReconcileClusterGauge(request.Namespace, request.Name, "SYNC_CLUSTER", float64(0))

	newClusterInfos, err := admin.GetClusterInfos(context, instance.Namespace, instance.Name)
	if proxyAz != "" {
		newClusterInfos.ProxyPodAz = proxyAz
	}
	if err != nil {
		r.MetricsClient.ReconcileCluster(request.Namespace, request.Name, "GET_CLUSTER_INFO", metrics.RECONCILE, metrics.FAIL, err.Error())
		reqLogger.Info("GetClusterInfos", "clusterInfos.Status", clusterInfos.Status)
		if clusterInfos.Status == redisutil.ClusterInfosPartial {
			return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}
	err2 := UpdateClusterInfos(newClusterInfos, syncCtx.pods, r.Client, reqLogger, r.MetricsClient)
	if err2 != nil {
		r.MetricsClient.ReconcileCluster(request.Namespace, request.Name, "UPDATE_CLUSTER_INFO", metrics.RECONCILE, metrics.FAIL, err2.Error())
		reqLogger.Error(err1, "UpdateClusterInfos", "Error when updating cluster infos", err1)
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	newStatus := buildClusterStatus(newClusterInfos, syncCtx.pods, instance, reqLogger)
	if instance.Spec.HaConfig.HaEnabled {
		_, haStatus, _, _ = buildHAStatus(newClusterInfos, instance, reqLogger)
		SetHAStatus(newStatus, haStatus)
	}
	clusterHealthOk := CheckAndUpdateClusterHealth(newStatus, newClusterInfos, instance, reqLogger)
	r.updateClusterIfNeed(instance, newStatus, reqLogger)
	// This part of code is very un optimally called , First need to optimise checkandUpdatePods, the above
	// buildClusterStatus has to be called only once , that data needs to be used for calling both checkandUpdatePods and updateLabels
	// and cluster Haheal
	if clusterHealthOk {
		r.MetricsClient.SetClusterOK(instance.Namespace, instance.Name, float64(1))
		requeue, err = r.checkandUpdatePods(admin, syncCtx, context)
		if err != nil {
			r.MetricsClient.ReconcileCluster(request.Namespace, request.Name, "CHECK_AND_UPDATE_PODS", metrics.RECONCILE, metrics.FAIL, err.Error())
			newStatus := instance.Status.DeepCopy()
			SetClusterFailed(newStatus, err.Error())
			r.updateClusterIfNeed(instance, newStatus, reqLogger)
			return reconcile.Result{RequeueAfter: time.Duration(reconcileTime) * time.Second}, nil
		}

		if requeue {
			if newStatus.Status != rediskunv1alpha1.ClusterStatusOnDelete {
				newStatusUpadte := instance.Status.DeepCopy()
				SetClusterUpdating(newStatusUpadte, "Cluster Upgrade in Progress")
				r.updateClusterIfNeed(instance, newStatusUpadte, reqLogger)
			}
			return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}
	} else {
		r.MetricsClient.SetClusterOK(instance.Namespace, instance.Name, float64(0))
		reqLogger.Info("Cluster Health", "NOK", newStatus.Reason)
		return reconcile.Result{RequeueAfter: 20 * time.Second}, nil
	}

	// Ha Healing functionality
	//1. First check the Ha Status to see if ha needs healing
	//2. check if annotations for ha healing is present and is set to true
	//3. check if all availability zones are present
	//4. if all zone are  present then, we need to heal the cluster for master and node distrubution
	//5. if all zones are not present then leave it for zone failure condition to recover
	if instance.Spec.HaConfig.HaEnabled {
		if clusterHealthOk && haStatus == rediskunv1alpha1.HaStatusFailed {
			err := fmt.Errorf("unexpected Error - HaStatusFailed")
			r.MetricsClient.ReconcileCluster(request.Namespace, request.Name, "HA status failed", "HA_HEAL", metrics.FAIL, err.Error())
			reqLogger.Error(err, "HaStatusFailed")
			r.MetricsClient.SetClusterOK(instance.Namespace, instance.Name, float64(2))
			return reconcile.Result{RequeueAfter: time.Duration(reconcileTime) * time.Second}, nil
			// 2. If ha needs healing , then check if the cluster is in a stable state
		} else if clusterHealthOk && haStatus == rediskunv1alpha1.HaStatusRedistribute {
			r.MetricsClient.SetClusterOK(instance.Namespace, instance.Name, float64(2))
			//check if all Zones active now
			isAllactive, _, err := CheckallZonesActive(instance, reqLogger, r.Client, r.MetricsClient)
			if err != nil {
				r.MetricsClient.ReconcileCluster(request.Namespace, request.Name, "CHECK_ALL_ZONES_ACTIVE", metrics.HA_HEAL, metrics.FAIL, err.Error())
				reqLogger.Error(err, "CheckallZonesActive")
				return reconcile.Result{RequeueAfter: 20 * time.Second}, nil
			}

			// Checking if annotation "reDistributeNode" is true only then it should proceed for nodes rebalancing
			redisClusterCRAnnotations := instance.Annotations
			reDitributeNodes := false
			for key, val := range redisClusterCRAnnotations {
				if key == "reDistributeNode" {
					if val == "true" {
						reDitributeNodes = true
					}
				}
			}

			// 3. If all zones not active , then check if we need to add annotations to not evict master pods
			// Inorder for self heal to occur, set reDitributeNodes=true as annotation in respective cluster CRD.
			if isAllactive && reDitributeNodes {
				// starting ha heal timer loop here
				r.MetricsClient.ReconcileClusterGauge(request.Namespace, request.Name, "HA_HEAL", float64(1))
				//check if any masters needs to be failedover
				haRanking, haStatus, displacedMasters, _ := buildHAStatus(newClusterInfos, instance, reqLogger)
				haRanking.ProxyAz = clusterInfos.ProxyPodAz
				if haStatus == rediskunv1alpha1.HaStatusFailed {
					r.MetricsClient.ReconcileCluster(request.Namespace, request.Name, "BUILD_HA_STATUS", metrics.HA_HEAL, metrics.FAIL, "HaStatusFailed")
					reqLogger.Error(fmt.Errorf("unexpected Error - HaStatusFailed"), "HaStatusFailed When trying to Distribute Masters")
					return reconcile.Result{RequeueAfter: time.Duration(reconcileTime) * time.Second}, nil
				}

				numofNodestoDel, delNodes, err1 := r.getListofNodestoDelete(instance, haRanking, reqLogger)
				if err1 != nil {
					r.MetricsClient.ReconcileCluster(request.Namespace, request.Name, "GET_LIST_OF_NODES_TO_DELETE", metrics.HA_HEAL, metrics.FAIL, err1.Error())
					reqLogger.Error(err1, "HaRedistribute - getListofNodestoDelete")
					return reconcile.Result{RequeueAfter: time.Duration(reconcileTime) * time.Second}, nil

				} else if numofNodestoDel > 0 {
					r.DeletePodsForHA(instance, delNodes, reqLogger)
					return reconcile.Result{RequeueAfter: time.Duration(reconcileTime) * time.Second}, nil
				}

				if haStatus == rediskunv1alpha1.HaStatusRedistribute && len(displacedMasters) > 0 {
					reqLogger.Info("HaStatusRedistribute - Distribute Masters", "displacedMasters", displacedMasters)
					//Re Distribute Masters by Failover
					for _, master := range displacedMasters {
						reqLogger.Info("HaStatusRedistribute - Distribute Masters", "master", master)
						BestRankedMaster, _ := haRanking.GetHAPlacedNodeinSTS(master.StatefulSet)
						if BestRankedMaster == nil {
							r.MetricsClient.ReconcileCluster(request.Namespace, request.Name, "UNABLE_TO_FIND_BEST_MASTER", metrics.HA_HEAL, metrics.FAIL, "BestRankedMaster is nil")
							reqLogger.Error(fmt.Errorf("unexpected Error - BestRankedMaster is nil"), "HaStatusRedistribute - BestRankedMaster")
							return reconcile.Result{RequeueAfter: time.Duration(reconcileTime) * time.Second}, nil
						} else if BestRankedMaster == master {
							reqLogger.Info("HaStatusRedistribute - BestRankedMaster is same as displacedMaster so no failover needed", "master", master)
							continue
						}
						err := r.FailoverMastersbySTS(instance, master, newClusterInfos, BestRankedMaster, admin, context, reqLogger)
						if err != nil {
							if GetType(err) == Requeue {
								reqLogger.Error(err, "HaStatusRedistribute - FailoverMastersbySTS", "Failed Retrying", err)
								r.MetricsClient.ReconcileCluster(request.Namespace, request.Name, "FAILOVER_MASTERS_BY_STS", metrics.HA_HEAL, metrics.FAIL, err.Error())
								return reconcile.Result{RequeueAfter: time.Duration(reconcileTime) * time.Second}, nil
							} else if GetType(err) == StopRetry {
								r.MetricsClient.ReconcileCluster(request.Namespace, request.Name, "FAILOVER_MASTERS_BY_STS", metrics.HA_HEAL, metrics.FATAL, err.Error())
								reqLogger.Error(err, "HaStatusRedistribute - FailoverMastersbySTS", "Failed", err)
								return reconcile.Result{}, nil
							}
						} else {
							reqLogger.Info("HaStatusRedistribute - FailoverMastersbySTS - Success", "NewMaster", BestRankedMaster.IP, "zone", BestRankedMaster.Zonename)
							return reconcile.Result{RequeueAfter: time.Duration(reconcileTime) * time.Second}, nil
						}

					}
					return reconcile.Result{RequeueAfter: time.Duration(reconcileTime) * time.Second}, nil

				} else if haStatus == rediskunv1alpha1.HaStatusHealthy && len(displacedMasters) == 0 {
					newStatus := instance.Status.DeepCopy()
					SetHAStatus(newStatus, rediskunv1alpha1.HaStatusHealthy)
					haStatus = rediskunv1alpha1.HaStatusHealthy
					r.updateClusterIfNeed(instance, newStatus, reqLogger)
					r.MetricsClient.ReconcileClusterGauge(request.Namespace, request.Name, "HA_HEAL", float64(0))
					reqLogger.Info("HaStatusHealthy - HaHealing Completed , Updating annotation reDistributeNode to false in cluster CRD to stop selfheal.")
					//Update the annotation reDistributeNode to false in cluster CRD to stop selfheal.
					//Selfheal should occur only if human operator sets annotation reDistributeNode as true on cluster crd.
					//Once selfheal is completed, setting this annotation to false to stop auto selfheal as it is doesn't consider production traffic.
					if err := r.CrController.UpdateCrdAnnotations(instance, "reDistributeNode", "false"); err != nil {
						r.MetricsClient.ReconcileClusterGauge(request.Namespace, request.Name, "HA_HEAL", float64(1))
						reqLogger.Error(err, "UpdateCrdAnnotations", "Failed to update annotation", err)
						r.MetricsClient.ReconcileCluster(request.Namespace, request.Name, "UPDATE_CR_ANNOTATION", metrics.HA_HEAL, metrics.FAIL, err.Error())
						return reconcile.Result{}, err
					}
					return reconcile.Result{RequeueAfter: time.Duration(reconcileTime) * time.Second}, nil
				}
			} else {
				reqLogger.Info("HaStatusRedistribute - Not all zones active or Healing annotation reDistributeNode is unset to true!")
				r.MetricsClient.SetClusterOK(instance.Namespace, instance.Name, float64(2))
				return reconcile.Result{RequeueAfter: time.Duration(reconcileTime) * time.Second}, nil

			}
		} else {
			reqLogger.Info("HAhealing not required", "haStatus", haStatus)
			r.MetricsClient.SetClusterOK(instance.Namespace, instance.Name, float64(3))
			r.MetricsClient.ReconcileClusterGauge(request.Namespace, request.Name, "HA_HEAL", float64(0))
			if err := r.CrController.UpdateCrdAnnotations(instance, "reDistributeNode", "false"); err != nil {
				r.MetricsClient.ReconcileClusterGauge(request.Namespace, request.Name, "HA_HEAL", float64(1))
				r.MetricsClient.ReconcileCluster(request.Namespace, request.Name, "UPDATE_CR_ANNOTATION", metrics.HA_HEAL, metrics.FAIL, err.Error())
				reqLogger.Error(err, "UpdateCrdAnnotations", "Failed to update annotation", err)
				return reconcile.Result{}, err
			}
		}
	} else {
		r.MetricsClient.SetClusterOK(instance.Namespace, instance.Name, float64(1))
	}
	r.MetricsClient.ReconcileClusterGauge(request.Namespace, request.Name, "RECONCILE", float64(1))
	return reconcile.Result{RequeueAfter: time.Duration(reconcileTime) * time.Second}, nil
}

func (r *DistributedRedisClusterReconciler) isScalingDown(cluster *rediskunv1alpha1.DistributedRedisCluster, reqLogger logr.Logger) bool {
	stsList, err := r.StatefulSetController.ListStatefulSetByLabels(cluster.Namespace, getLabels(cluster))
	if err != nil {
		reqLogger.Error(err, "ListStatefulSetByLabels")
		return false
	}
	if len(stsList.Items) > int(cluster.Spec.MasterSize) {
		return true
	}
	return false
}

// This function is called to identify the list of replica nodes to delete per stateful set to bring back the HA distribution
// This function assumes that its called only when all the zone have become active again and there is atleast one node available
// per zone , even without any pods scheduled
// This function would return a list of nodes to be deleted per statefulset if there is a need, along with number of nodes to be deleted
// if there is no need for nodes to be deleted the list would be empty and the count would be zero
func (r *DistributedRedisClusterReconciler) getListofNodestoDelete(cluster *rediskunv1alpha1.DistributedRedisCluster, haRanking *clustering.RankingInfo, reqLogger logr.Logger) (int32, map[string][]*redisutil.Node, error) {
	var nodesToDelete map[string][]*redisutil.Node = make(map[string][]*redisutil.Node)
	var minNodesExpectedPerZone, maxNodesExpectedPerZone int32 = 0, 0
	var nodesDelCntPerSTS, nodesPickedFordel, possibleNodeDelCnt, totalnodesToDel int32 = 0, 0, 0, 0
	var isDeleteNodesbyMinNodecnt bool = false

	zones := make([]string, 0, len(cluster.Spec.HaConfig.ZonesInfo))
	for zone := range cluster.Spec.HaConfig.ZonesInfo {
		zones = append(zones, zone)
	}
	sort.Strings(zones)

	//In cases where the number of zones is greater than the number of redis replicas + master,
	//each zone is expected to have min 0 or 1 redis nodes only , so its good enough the get the number of nodes to delete based on the maxnumber of nodes avaiable in the zone, and min number of nodes expecected per zone is not needed
	if int(1+cluster.Spec.ClusterReplicas) > len(zones) {
		isDeleteNodesbyMinNodecnt = true
	}

	minNodesExpectedPerZone = (1 + cluster.Spec.ClusterReplicas) / int32(len(zones))
	maxNodesExpectedPerZone = ((1 + cluster.Spec.ClusterReplicas) + (int32(len(zones) - 1))) / int32(len(zones))

	reqLogger.Info("getListofNodestoDelete", "minNodesExpectedPerZone", minNodesExpectedPerZone)
	reqLogger.Info("getListofNodestoDelete", "maxNodesExpectedPerZone", maxNodesExpectedPerZone)

	for i := 0; i < int(cluster.Spec.MasterSize); i++ {
		stsname := statefulsets.ClusterStatefulSetName(cluster.Name, i)
		nodesDelCntPerSTS = 0
		//First Iteration get the count of nodes to be deleted per zone for that statefulset or shard
		for _, zone := range zones {

			//Get the number of nodes to be deleted in this zone by checking against the min number of nodes expected per zone
			if isDeleteNodesbyMinNodecnt && minNodesExpectedPerZone > 0 && (len(haRanking.NodesIndexedbyZoneAndSTS[zone][stsname]) < int(minNodesExpectedPerZone)) {
				reqLogger.Info("Less nodes in zone", "zone", zone, "nodes", len(haRanking.NodesIndexedbyZoneAndSTS[zone][stsname]), "stsname", stsname)
				nodesDelCntPerSTS = nodesDelCntPerSTS + (minNodesExpectedPerZone - int32(len(haRanking.NodesIndexedbyZoneAndSTS[zone][stsname])))
			}

			//Get the number of nodes to be deleted in this zone by checking against the max number of nodes expected per zone
			if !isDeleteNodesbyMinNodecnt && (len(haRanking.NodesIndexedbyZoneAndSTS[zone][stsname]) > int(maxNodesExpectedPerZone)) {
				reqLogger.Info("More nodes in zone", "zone", zone, "nodes", len(haRanking.NodesIndexedbyZoneAndSTS[zone][stsname]), "stsname", stsname)
				nodesDelCntPerSTS = nodesDelCntPerSTS + (int32(len(haRanking.NodesIndexedbyZoneAndSTS[zone][stsname])) - maxNodesExpectedPerZone)
			}

		}
		//Second iteraation identify the nodes to be deleted for the statefulset per zone
		//If there are nodes to be deleted per statefulset , try to identify and pick the nodes to be deleted
		if nodesDelCntPerSTS > 0 {

			copy_nodesDelCntPerSTS := nodesDelCntPerSTS
			totalnodesToDel = totalnodesToDel + nodesDelCntPerSTS
			for _, zone := range zones {
				//For each zone if nodes exists for this STS and its greater than or equal to the maxnodes expected per zone
				//Then delete replica nodes from this zone , until we reach the min node count for that zone
				if _, nodesExists := haRanking.NodesIndexedbyZoneAndSTS[zone][stsname]; nodesExists && len(haRanking.NodesIndexedbyZoneAndSTS[zone][stsname]) >= int(maxNodesExpectedPerZone) {

					reqLogger.Info("Trying to pick nodes to deleted in zone", "zone", zone, "for stsname", stsname)
					possibleNodeDelCnt = int32(len(haRanking.NodesIndexedbyZoneAndSTS[zone][stsname])) - minNodesExpectedPerZone

					for _, node := range haRanking.NodesIndexedbyZoneAndSTS[zone][stsname] {

						//Break the loop if the possible nodes that can be deleted from this satefulset has become zero
						if possibleNodeDelCnt == 0 {
							break
						}
						if node.Role == redisutil.RedisSlaveRole {
							nodesToDelete[stsname] = append(nodesToDelete[stsname], node)
							reqLogger.Info("Node to delete", "nodeID", node.ID, "nodeIP", node.IP, "nodeRole", node.Role, "nodeStatefulSet", node.StatefulSet, "nodeZone", node.Zonename, "nodePodName", node.PodName, "nodeNodeName", node.NodeName)
							possibleNodeDelCnt = possibleNodeDelCnt - 1
							nodesDelCntPerSTS = nodesDelCntPerSTS - 1
							nodesPickedFordel = nodesPickedFordel + 1
						}
					}
				}
				if nodesDelCntPerSTS == 0 {
					reqLogger.Info("getListofNodestoDelete", "Required number of nodes to be deleted identified for statefulset", stsname)
					break
				}
			}
			if nodesDelCntPerSTS > 0 {
				reqLogger.Error(fmt.Errorf("node deletion error"), "getListofNodestoDelete", "expecected number of nodes to delete", copy_nodesDelCntPerSTS, "actual number of nodes picked for deletion", nodesPickedFordel, "for statefuleset", stsname)
				return 0, nil, fmt.Errorf("unsucess/invalid number of nodes picked to delete")
			} else {
				reqLogger.Info("getListofNodestoDelete", "expecected number of nodes to delete", copy_nodesDelCntPerSTS, "actual number of nodes picked for deletion", nodesPickedFordel, "for statefuleset", stsname)
			}
		} else {
			reqLogger.Info("getListofNodestoDelete", "No nodes to delete in statefulset", stsname)
		}

	}
	if totalnodesToDel > 0 {
		if totalnodesToDel == nodesPickedFordel {
			return nodesPickedFordel, nodesToDelete, nil
		} else {
			reqLogger.Error(fmt.Errorf("unexpected error"), "getListofNodestoDelete", "Unexpected Error totalnodesToDel", totalnodesToDel, "nodesPickedFordel", nodesPickedFordel)

		}
	}
	return 0, nil, nil

}

func (r *DistributedRedisClusterReconciler) FailoverMastersbySTS(cluster *rediskunv1alpha1.DistributedRedisCluster, displacedMaster *redisutil.Node, clusterinfos *redisutil.ClusterInfos, BestRankedMaster *redisutil.Node, admin redisutil.IAdmin, context context.Context, reqLogger logr.Logger) error {

	failoverNode := BestRankedMaster

	if failoverNode.Role == redisutil.RedisMasterRole {
		if failoverNode.TotalSlots() > 0 {
			// Possibility of Multimaster scenario Manual operator involment needed
			reqLogger.Error(fmt.Errorf("possible Multimaster"), "Node already master in appropriate zone Possibility of Multimaster", "nodeID", failoverNode.ID, "nodeIP", failoverNode.IP, "nodeRole", failoverNode.Role, "nodeStatefulSet", failoverNode.StatefulSet, "nodeZone", failoverNode.Zonename, "nodePodName", failoverNode.PodName, "nodeNodeName", failoverNode.NodeName)

		} else {
			reqLogger.Error(fmt.Errorf("possible dataloss"), "Node to be promoted to master has no slots associated possibility of data loss", "nodeID", failoverNode.ID, "nodeIP", failoverNode.IP, "nodeRole", failoverNode.Role, "nodeStatefulSet", failoverNode.StatefulSet, "nodeZone", failoverNode.Zonename, "nodePodName", failoverNode.PodName, "nodeNodeName", failoverNode.NodeName)
		}
		return StopRetry.New("possible Multimaster or Dataloss by failover")

	} else if failoverNode.Role == redisutil.RedisSlaveRole && failoverNode.LinkState == redisutil.RedisLinkStateConnected {
		reqLogger.Info("Node to be promoted to master", "nodeID", failoverNode.ID, "nodeIP", failoverNode.IP, "nodeRole", failoverNode.Role, "nodeStatefulSet", failoverNode.StatefulSet, "nodeZone", failoverNode.Zonename, "nodePodName", failoverNode.PodName, "nodeNodeName", failoverNode.NodeName)

		//Promote node to master
		err := admin.FailoverSlaveToMaster(context, failoverNode, redisutil.RedisFailoverDefault, cluster.Namespace, cluster.Name)
		if err != nil {
			reqLogger.Error(err, "FailoverSlaveToMaster", "Failed", err)
			return Requeue.Wrap(err, "FailoverSlaveToMaster Failed")
		}

		// Drain clients in old master
		err = admin.DrainClientsInOldMaster(context, displacedMaster, cluster.Namespace, cluster.Name)
		if err != nil {
			reqLogger.Error(err, "DrainClientsInOldMaster", "Failed", err)
			return Requeue.Wrap(err, "DrainClientsInOldMaster Failed")
		}
	} else {
		err1 := fmt.Errorf("invalid Role %s", failoverNode.Role)
		return StopRetry.Wrap(err1, "FailoverMastersbySTS")
	}

	return nil
}

func (r *DistributedRedisClusterReconciler) DeletePodsForHA(instance *rediskunv1alpha1.DistributedRedisCluster, delNodes map[string][]*redisutil.Node, reqLogger logr.Logger) error {
	Podclient := k8sutil.NewPodController(r.Client, r.MetricsClient)
	for i := 0; i < int(instance.Spec.MasterSize); i++ {
		stsname := statefulsets.ClusterStatefulSetName(instance.Name, i)
		if _, ok := delNodes[stsname]; ok {
			for _, delRedisNode := range delNodes[stsname] {
				reqLogger.Info("Calling Delete Pod", "PodName", delRedisNode.PodName, "stsname", stsname, "NodeName", delRedisNode.NodeName, "Zone", delRedisNode.Zonename, "Role", delRedisNode.Role, "IP", delRedisNode.IP, "StatefulSet", delRedisNode.StatefulSet)
				err2 := Podclient.DeletePodByName(instance.Namespace, delRedisNode.PodName)
				if err2 != nil {
					reqLogger.Error(err2, "HaRedistribute - DeletePodByName")
					return err2
				}
			}
		}
	}
	return nil
}
