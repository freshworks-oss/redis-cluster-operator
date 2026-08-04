package distributedrediscluster

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime/pkg/reconcile"

	redisv1alpha1 "github.com/freshworks/redis-cluster-operator/api/v1alpha1"
	"github.com/freshworks/redis-cluster-operator/metrics"
	"github.com/freshworks/redis-cluster-operator/utils/k8sutil"
	"github.com/freshworks/redis-cluster-operator/utils/redisutil"
)

func (r *DistributedRedisClusterReconciler) reconcileBootstrapMode(
	syncCtx *syncContext,
	context context.Context,
	instance *redisv1alpha1.DistributedRedisCluster,
	admin redisutil.IAdmin,
	proxyAz string,
	reqLogger logr.Logger,
) (ctrl.Result, error) {
	reqLogger.Info("bootstrap mode active")

	clusterInfos, clusterInfoErr := admin.GetClusterInfos(context, instance.Namespace, instance.Name)
	if clusterInfoErr != nil {
		if clusterErr, ok := clusterInfoErr.(redisutil.ClusterInfosError); ok && clusterErr.Partial() {
			reqLogger.Info("bootstrap GetClusterInfos partial", "error", clusterInfoErr)
		} else if !ok {
			reqLogger.Info("bootstrap GetClusterInfos error", "error", clusterInfoErr)
		}
	}
	if clusterInfos != nil {
		if proxyAz != "" {
			clusterInfos.ProxyPodAz = proxyAz
		}
		if updateErr := UpdateClusterInfos(clusterInfos, syncCtx.pods, r.Client, reqLogger, r.MetricsClient, true); updateErr != nil {
			r.MetricsClient.ReconcileCluster(instance.Namespace, instance.Name, "UPDATE_CLUSTER_INFO", metrics.RECONCILE, metrics.FAIL, updateErr.Error())
			return ctrl.Result{RequeueAfter: requeueAfter}, nil
		}
	}

	if instance.BootstrapReconcileEnabled() {
		if err := admin.SetConfigIfNeed(context, instance.Spec.Config, instance.Namespace, instance.Name); err != nil {
			r.MetricsClient.ReconcileCluster(instance.Namespace, instance.Name, "SET_CONFIG", metrics.RECONCILE, metrics.FAIL, err.Error())
			// SetConfigIfNeed fails transiently whenever any currently-connected node briefly can't
			// take a CONFIG SET (e.g. still joining during bootstrap) — that must not skip the status
			// write below, or a DRC that hits this on every pass never gets a status at all.
			newStatus := instance.Status.DeepCopy()
			SetClusterNOK(newStatus, err.Error())
			r.updateClusterIfNeed(instance, newStatus, reqLogger)
			return ctrl.Result{RequeueAfter: requeueAfter}, nil
		}
		if err := r.syncBootstrapCluster(syncCtx, context, instance); err != nil {
			r.MetricsClient.ReconcileClusterGauge(instance.Namespace, instance.Name, "SYNC_BOOTSTRAP", float64(1))
			if GetType(err) == Requeue {
				newStatus := instance.Status.DeepCopy()
				SetBootstrapStatus(newStatus, redisv1alpha1.BootstrapPhaseInProgress, newStatus.Bootstrap.Progress, err.Error())
				SetClusterNOK(newStatus, err.Error())
				r.updateClusterIfNeed(instance, newStatus, reqLogger)
				return ctrl.Result{RequeueAfter: requeueAfter}, nil
			}
			// Everything else (StopRetry and any other error type) lands in Failed status the same
			// way; only the error actually handed back to controller-runtime differs — StopRetry
			// deliberately returns nil to skip its backoff-driven retry when retrying won't help,
			// everything else propagates the error so it does retry.
			newStatus := instance.Status.DeepCopy()
			SetBootstrapStatus(newStatus, redisv1alpha1.BootstrapPhaseFailed, "", err.Error())
			SetClusterFailed(newStatus, err.Error())
			r.updateClusterIfNeed(instance, newStatus, reqLogger)
			returnedErr := err
			if GetType(err) == StopRetry {
				returnedErr = nil
			}
			return ctrl.Result{}, returnedErr
		}
		r.MetricsClient.ReconcileClusterGauge(instance.Namespace, instance.Name, "SYNC_BOOTSTRAP", float64(0))
	}

	newClusterInfos, err := admin.GetClusterInfos(context, instance.Namespace, instance.Name)
	if proxyAz != "" && newClusterInfos != nil {
		newClusterInfos.ProxyPodAz = proxyAz
	}
	if err == nil && newClusterInfos != nil {
		if updateErr := UpdateClusterInfos(newClusterInfos, syncCtx.pods, r.Client, reqLogger, r.MetricsClient, true); updateErr != nil {
			reqLogger.Error(updateErr, "UpdateClusterInfos", "bootstrap mode")
		}
	}

	if newClusterInfos != nil {
		r.forgetGhostNodes(context, admin, newClusterInfos, instance.Namespace, instance.Name, reqLogger)
	}

	r.syncPodRoleMetadata(newClusterInfos, syncCtx.pods, instance.Namespace, reqLogger)

	newStatus := buildClusterStatus(newClusterInfos, syncCtx.pods, instance, reqLogger)
	bootstrapHealthy := CheckBootstrapHealth(newStatus, instance, admin, context, syncCtx.pods, reqLogger)
	r.updateClusterIfNeed(instance, newStatus, reqLogger)

	if bootstrapHealthy {
		r.MetricsClient.SetClusterOK(instance.Namespace, instance.Name, float64(1))
	} else {
		r.MetricsClient.SetClusterOK(instance.Namespace, instance.Name, float64(0))
		if instance.BootstrapReconcileEnabled() {
			return ctrl.Result{RequeueAfter: requeueAfter}, nil
		}
	}

	r.MetricsClient.ReconcileClusterGauge(instance.Namespace, instance.Name, "RECONCILE", float64(1))
	return ctrl.Result{RequeueAfter: time.Duration(reconcileTime) * time.Second}, nil
}

// syncPodRoleMetadata keeps each local pod's "role" label (Master/Slave) and
// cluster-autoscaler.kubernetes.io/safe-to-evict annotation in sync with its live cluster role. The
// main reconcile path does this via checkandUpdatePods/setRoleLabelIfNeeded/setPodAnnotationIfNeeded,
// but that path is never reached while bootstrap mode or the paused-source path is active, so those
// pods would otherwise never get either one set.
func (r *DistributedRedisClusterReconciler) syncPodRoleMetadata(clusterInfos *redisutil.ClusterInfos, pods []*corev1.Pod, namespace string, reqLogger logr.Logger) {
	if clusterInfos == nil {
		return
	}
	client := k8sutil.NewPodController(r.Client, r.MetricsClient)
	for _, pod := range pods {
		redisNodes, err := clusterInfos.GetNodes().GetNodesByFunc(func(node *redisutil.Node) bool {
			return node.IP == pod.Status.PodIP
		})
		if err != nil || len(redisNodes) != 1 {
			continue
		}
		nodeRole := redisNodes[0].GetRole()
		if nodeRole == redisv1alpha1.RedisClusterNodeRoleNone {
			continue
		}
		role := string(nodeRole)
		if err := r.setRoleLabelIfNeeded(pod.Status.PodIP, pod, role, namespace, client); err != nil {
			reqLogger.Error(err, "bootstrap: set role label failed", "pod", pod.Name)
		}
		if err := r.setPodAnnotationIfNeeded(role, pod, client); err != nil {
			reqLogger.Error(err, "bootstrap: set evict annotation failed", "pod", pod.Name)
		}
	}
}

// listGhostBootstrapNodes finds node ids safe to CLUSTER FORGET while bootstrap mode or the
// paused-source path is active. controllers/heal.Heal() (specifically FixUntrustedNodes) is
// skipped entirely in both these modes because it cross-references the RECONCILER'S OWN local
// pod list to decide if a node is orphaned — during a migration the merged cluster view also
// contains the other side's nodes, which this reconciler has no visibility into, so that check
// would misjudge a perfectly healthy remote node as orphaned just because it isn't in the local
// pod list. This instead only relies on two conditions that hold regardless of which side a node
// belongs to:
//  1. The node holds zero slots in every reported view. A node with any slots is never a forget
//     candidate, so this can never open a slot-coverage gap on either side.
//  2. A majority of currently-reachable peers already report it via Redis's own "fail" or
//     "noaddr" gossip flag (the transient, unconfirmed "pfail" state does not count) — i.e.
//     Redis's own consensus, not ours, has already concluded the node is dead.
//
// The majority threshold is sized off however many peers we can currently query (len(infos.Infos)),
// not a fixed spec-derived count, since that population grows mid-pass once
// meetBootstrapPods/applyRemoteBootstrapConfig have opened connections to the other side's nodes.
func listGhostBootstrapNodes(infos *redisutil.ClusterInfos) map[string]bool {
	ghostNodes := map[string]bool{}
	if infos == nil || len(infos.Infos) == 0 {
		return ghostNodes
	}

	hasSlots := map[string]bool{}
	failVotes := map[string]int32{}
	for _, nodeinfos := range infos.Infos {
		if nodeinfos == nil {
			continue
		}
		for _, node := range nodeinfos.Friends {
			if node == nil {
				continue
			}
			if node.TotalSlots() > 0 {
				hasSlots[node.ID] = true
				continue
			}
			if node.HasStatus(redisutil.NodeStatusFail) || node.HasStatus(redisutil.NodeStatusNoAddr) {
				failVotes[node.ID]++
			}
		}
	}

	// Strict majority of currently-reachable peers: more than half must vote fail/noaddr.
	// (len/2+1 == len/2 strictly-greater, e.g. 2/2→2 of 2, 3/2→2 of 3, 4/2→3 of 4.)
	majority := int32(len(infos.Infos))/2 + 1
	for id, votes := range failVotes {
		if hasSlots[id] {
			continue
		}
		if votes >= majority {
			ghostNodes[id] = true
		}
	}
	return ghostNodes
}

// forgetGhostNodes CLUSTER FORGETs any node id returned by listGhostBootstrapNodes. Safe to call
// from both the destination bootstrap path and the paused-source path — see
// listGhostBootstrapNodes for why this doesn't need either reconciler to see the other cluster's
// pods.
func (r *DistributedRedisClusterReconciler) forgetGhostNodes(ctx context.Context, admin redisutil.IAdmin, infos *redisutil.ClusterInfos, namespace, name string, reqLogger logr.Logger) {
	for id := range listGhostBootstrapNodes(infos) {
		reqLogger.Info("forgetting ghost node (zero slots, majority-reported fail/noaddr)", "nodeId", id)
		if err := admin.ForgetNode(ctx, id, namespace, name); err != nil {
			reqLogger.Error(err, "forget ghost node failed", "nodeId", id)
		}
	}
}
