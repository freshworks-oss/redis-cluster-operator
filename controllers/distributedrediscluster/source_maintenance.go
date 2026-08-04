package distributedrediscluster

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime/pkg/reconcile"

	redisv1alpha1 "github.com/freshworks/redis-cluster-operator/api/v1alpha1"
	"github.com/freshworks/redis-cluster-operator/metrics"
	"github.com/freshworks/redis-cluster-operator/utils/redisutil"
)

// reconcileSourcePaused runs the limited reconcile for a source cluster paused during a
// blue-green migration (bootstrap annotation set). It keeps this operator's own pods attached
// as replicas of their shard's current master (join + replicate) and deliberately skips heal,
// clustering, and SetConfigIfNeed. Structural maintenance (ensureCluster/secrets/HA) has already
// run in Reconcile before this is called.
func (r *DistributedRedisClusterReconciler) reconcileSourcePaused(
	syncCtx *syncContext,
	ctx context.Context,
	instance *redisv1alpha1.DistributedRedisCluster,
	admin redisutil.IAdmin,
	proxyAz string,
	reqLogger logr.Logger,
) (ctrl.Result, error) {
	reqLogger.Info("source paused mode: maintaining replicas only (heal, clustering and config resync skipped)")

	clusterInfos, err := admin.GetClusterInfos(ctx, instance.Namespace, instance.Name)
	if err != nil {
		if clusterErr, ok := err.(redisutil.ClusterInfosError); ok && clusterErr.Partial() {
			reqLogger.Info("source paused: GetClusterInfos partial", "error", err)
		} else if !ok {
			reqLogger.Info("source paused: GetClusterInfos error", "error", err)
		}
	}
	if clusterInfos == nil {
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}
	if proxyAz != "" {
		clusterInfos.ProxyPodAz = proxyAz
	}
	syncCtx.admin = admin
	syncCtx.clusterInfos = clusterInfos

	// Status update tolerates cluster nodes that have no local pod (the destination nodes).
	if updateErr := UpdateClusterInfos(clusterInfos, syncCtx.pods, r.Client, reqLogger, r.MetricsClient, true); updateErr != nil {
		reqLogger.Error(updateErr, "UpdateClusterInfos", "source paused mode")
	}

	// UpdateClusterInfos only enriches the in-memory node list (zone/pod name); it never persists
	// anything. Build and write status.Nodes/NumberOfMaster/etc. here the same way
	// reconcileBootstrapMode does for the destination, otherwise a paused source DRC never gets a
	// populated status beyond whatever ensureCluster/secrets/HA wrote earlier in Reconcile.
	newStatus := buildClusterStatus(clusterInfos, syncCtx.pods, instance, reqLogger)
	r.updateClusterIfNeed(instance, newStatus, reqLogger)

	// Heal() (and its ghost-node cleanup) is skipped while paused — see listGhostBootstrapNodes
	// for why the destination-side equivalent isn't safe to reuse as-is, and why this narrower
	// check is safe here too.
	r.forgetGhostNodes(ctx, admin, clusterInfos, instance.Namespace, instance.Name, reqLogger)

	// Same reasoning as status above: the main reconcile path's role-label/evict-annotation
	// maintenance (checkandUpdatePods) never runs while paused, so a source pod that changes role
	// after pausing (e.g. re-attached under a different master) would otherwise keep a stale label
	// and never get the safe-to-evict annotation it should have.
	r.syncPodRoleMetadata(clusterInfos, syncCtx.pods, instance.Namespace, reqLogger)

	if err := r.maintainSourceReplicas(syncCtx, ctx, instance); err != nil {
		if GetType(err) == Requeue {
			reqLogger.Info("source paused: re-attaching pods", "reason", err.Error())
		} else {
			r.MetricsClient.ReconcileCluster(instance.Namespace, instance.Name, "SOURCE_MAINTAIN", metrics.RECONCILE, metrics.FAIL, err.Error())
			reqLogger.Error(err, "maintainSourceReplicas")
		}
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	r.MetricsClient.ReconcileClusterGauge(instance.Namespace, instance.Name, "RECONCILE", float64(1))
	return ctrl.Result{RequeueAfter: time.Duration(reconcileTime) * time.Second}, nil
}

// maintainSourceReplicas re-joins and re-attaches source pods that have fallen out of the cluster
// (e.g. restarted with a lost PVC) as replicas of their shard's current master. It never assigns a
// master role, moves slots, or forgets nodes. A pod already serving slots or already replicating a
// master is left untouched.
func (r *DistributedRedisClusterReconciler) maintainSourceReplicas(ctx *syncContext, context context.Context, instance *redisv1alpha1.DistributedRedisCluster) error {
	infos := ctx.clusterInfos

	type attachWork struct {
		pod    *corev1.Pod
		node   *redisutil.Node
		master *redisutil.Node
	}
	var toAttach []attachWork
	var needJoin bool

	for _, pod := range ctx.pods {
		if pod.Status.PodIP == "" {
			continue
		}
		node := findSourceNodeByIP(infos, pod.Status.PodIP)
		if node == nil {
			// Not visible in the cluster yet; needs a CLUSTER MEET before it can be attached.
			needJoin = true
			continue
		}
		// Already a replica of some master (gossip healed it) or a master still serving slots
		// (pre-cutover source master): leave it alone.
		if node.Role == redisutil.RedisSlaveRole && node.MasterReferent != "" {
			continue
		}
		if redisutil.IsMasterWithSlot(node) {
			continue
		}
		// Empty/standalone node: re-attach as a replica of its shard's current master.
		master, err := shardCurrentMaster(infos, pod, ctx.pods)
		if err != nil {
			reqLogger := ctx.reqLogger
			reqLogger.Info("source paused: current master not resolvable yet", "pod", pod.Name, "reason", err.Error())
			needJoin = true
			continue
		}
		toAttach = append(toAttach, attachWork{pod: pod, node: node, master: master})
	}

	if len(toAttach) == 0 && !needJoin {
		return nil
	}

	// Introduce any not-yet-joined pods to the cluster (idempotent CLUSTER MEET).
	if err := r.waitForClusterJoin(ctx, context, instance); err != nil {
		return Requeue.Wrap(err, "waitForClusterJoin")
	}

	for _, w := range toAttach {
		if err := ctx.admin.AttachSlaveToMaster(context, w.node, w.master.ID, instance.Namespace, instance.Name); err != nil {
			r.MetricsClient.ReconcileCluster(instance.Namespace, instance.Name, "SOURCE_REPLICATE", metrics.SYNC_CLUSTER, metrics.FAIL, err.Error())
			return Redis.Wrap(err, fmt.Sprintf("attach source pod %s as replica of %s", w.pod.Name, w.master.IPPort()))
		}
		ctx.reqLogger.Info("source paused: re-attached pod as replica", "pod", w.pod.Name, "master", w.master.IPPort())
	}

	return Requeue.New(fmt.Sprintf("source pods re-attaching: %d joined/attached this pass", len(toAttach)))
}

// findSourceNodeByIP returns the cluster node with the given IP, or nil if not present.
func findSourceNodeByIP(infos *redisutil.ClusterInfos, ip string) *redisutil.Node {
	for _, node := range infos.GetNodes() {
		if node != nil && node.IP == ip {
			return node
		}
	}
	return nil
}

// shardCurrentMaster finds the node currently owning the slots for the given pod's shard, by
// inspecting the pod's StatefulSet peers that are already cluster members. The current master may
// be a source pod (pre-cutover) or a destination pod (post-cutover); either is a valid target.
func shardCurrentMaster(infos *redisutil.ClusterInfos, pod *corev1.Pod, pods []*corev1.Pod) (*redisutil.Node, error) {
	sts := podStatefulSetName(pod)
	if sts == "" {
		return nil, fmt.Errorf("no statefulset owner for pod %s", pod.Name)
	}

	siblingIPs := make(map[string]struct{})
	for _, p := range pods {
		if p.Name == pod.Name || p.Status.PodIP == "" {
			continue
		}
		if podStatefulSetName(p) == sts {
			siblingIPs[p.Status.PodIP] = struct{}{}
		}
	}

	nodes := infos.GetNodes()
	// A sibling that is itself the master serving this shard's slots.
	for _, node := range nodes {
		if node == nil {
			continue
		}
		if _, ok := siblingIPs[node.IP]; ok && redisutil.IsMasterWithSlot(node) {
			return node, nil
		}
	}
	// A sibling replica points at the shard's current master.
	for _, node := range nodes {
		if node == nil {
			continue
		}
		if _, ok := siblingIPs[node.IP]; !ok {
			continue
		}
		if node.Role == redisutil.RedisSlaveRole && node.MasterReferent != "" {
			if master, err := nodes.GetNodeByID(node.MasterReferent); err == nil && master != nil {
				return master, nil
			}
		}
	}
	return nil, fmt.Errorf("cannot determine current master for shard %s", sts)
}
