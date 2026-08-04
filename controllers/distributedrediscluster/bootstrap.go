package distributedrediscluster

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"

	redisv1alpha1 "github.com/freshworks/redis-cluster-operator/api/v1alpha1"
	"github.com/freshworks/redis-cluster-operator/metrics"
	"github.com/freshworks/redis-cluster-operator/utils/redisutil"
)

const (
	bootstrapReplicaNoFailover            = "cluster-replica-no-failover"
	bootstrapAllowReplicaMigration        = "cluster-allow-replica-migration"
	bootstrapReplicaNoFailoverValue       = "yes"
	bootstrapReplicaNoFailoverDisabled    = "no"
	bootstrapAllowReplicaMigrationValue   = "no"
	bootstrapAllowReplicaMigrationEnabled = "yes"
)

var remoteBootstrapRedisConfig = map[string]string{
	bootstrapAllowReplicaMigration: bootstrapAllowReplicaMigrationValue,
}

type bootstrapPodRef struct {
	pod        *corev1.Pod
	shardIndex int
	ordinal    int
	ipPort     string
}

// applyBootstrapRedisConfig ensures bootstrap-specific redis settings are present in spec.Config
// for destination pods (applied via redis.conf at pod creation and CONFIG SET on local nodes).
// Both settings track bootstrap reconcile state the same way: guarding while reconcile is active
// (default), relaxed once reconcile is false so replacement pods behave correctly during cutover.
//   - cluster-replica-no-failover: yes while reconcile is active, no when reconcile is false, so
//     replacement pods allow automatic failover during cutover.
//   - cluster-allow-replica-migration: no while reconcile is active (guards against replica
//     migration disrupting the bootstrap join), yes when reconcile is false (cutover no longer
//     needs the guard).
func applyBootstrapRedisConfig(cluster *redisv1alpha1.DistributedRedisCluster) {
	if !cluster.BootstrapMode() {
		return
	}
	if cluster.Spec.Config == nil {
		cluster.Spec.Config = make(map[string]string)
	}
	noFailover := bootstrapReplicaNoFailoverDisabled
	allowReplicaMigration := bootstrapAllowReplicaMigrationValue
	if cluster.BootstrapReconcileEnabled() {
		noFailover = bootstrapReplicaNoFailoverValue
	} else {
		allowReplicaMigration = bootstrapAllowReplicaMigrationEnabled
	}
	cluster.Spec.Config[bootstrapReplicaNoFailover] = noFailover
	cluster.Spec.Config[bootstrapAllowReplicaMigration] = allowReplicaMigration
}

func (r *DistributedRedisClusterReconciler) syncBootstrapCluster(ctx *syncContext, context context.Context, instance *redisv1alpha1.DistributedRedisCluster) error {
	bootstrap := instance.Spec.BootstrapNode
	seedAddr := net.JoinHostPort(bootstrap.Host, instance.BootstrapSeedPort())

	if err := ctx.admin.Connections().Add(context, seedAddr); err != nil {
		return Redis.Wrap(err, "connect to bootstrap seed")
	}

	seedInfos, err := ctx.admin.GetClusterNodeInfos(context, seedAddr, instance.Namespace, instance.Name)
	if err != nil {
		r.MetricsClient.ReconcileCluster(instance.Namespace, instance.Name, "BOOTSTRAP_DISCOVER", metrics.SYNC_CLUSTER, metrics.FAIL, err.Error())
		return Redis.Wrap(err, "discover cluster from seed")
	}

	remoteNodes := append(seedInfos.Friends, seedInfos.Node)
	if err := r.applyRemoteBootstrapConfig(context, ctx, instance, remoteNodes); err != nil {
		return err
	}

	remoteMasters, err := sortRemoteMasters(remoteNodes)
	if err != nil {
		return Cluster.Wrap(err, "sort remote masters")
	}
	if len(remoteMasters) != int(instance.Spec.MasterSize) {
		// Requeue, not a hard failure: the seed's own failover/resharding can transiently drop a
		// master's slots (filtered out by sortRemoteMasters/IsMasterWithSlot), which self-heals on
		// the next pass. Same reasoning as the other transient-condition fixes in this file.
		return Requeue.New(fmt.Sprintf("seed cluster has %d masters, expected %d", len(remoteMasters), instance.Spec.MasterSize))
	}

	localPods, err := orderedBootstrapPods(ctx.pods)
	if err != nil {
		return Cluster.Wrap(err, "order bootstrap pods")
	}

	if err := r.meetBootstrapPods(context, ctx, instance, seedAddr, remoteNodes, localPods); err != nil {
		return err
	}

	if err := verifyBootstrapJoin(context, ctx, instance, remoteNodes, localPods); err != nil {
		if GetType(err) == Requeue {
			return err
		}
		return Redis.Wrap(err, "verify bootstrap join")
	}

	return r.replicateBootstrapPods(context, ctx, instance, remoteMasters, localPods)
}

// applyRemoteBootstrapConfig sets migration guardrails on source-cluster nodes discovered via the seed.
// Destination-local settings are applied through spec.Config and SetConfigIfNeed.
func (r *DistributedRedisClusterReconciler) applyRemoteBootstrapConfig(ctx context.Context, syncCtx *syncContext, instance *redisv1alpha1.DistributedRedisCluster, remoteNodes redisutil.Nodes) error {
	for _, node := range remoteNodes {
		if node == nil {
			continue
		}
		if node.LinkState != redisutil.RedisLinkStateConnected {
			// The seed's own CLUSTER NODES already reports this one as unreachable — e.g. a pod
			// that was deleted/recreated with a new IP, leaving its old entry lingering until
			// something CLUSTER FORGETs it. Trying (and failing) to connect to it every single
			// pass would otherwise block bootstrap health forever on a node that's simply gone.
			continue
		}
		addr := node.IPPort()

		if err := syncCtx.admin.Connections().Add(ctx, addr); err != nil {
			return Requeue.Wrap(err, fmt.Sprintf("connect remote node %s", addr))
		}
		if err := syncCtx.admin.SetConfigOnAddr(ctx, addr, remoteBootstrapRedisConfig, instance.Namespace, instance.Name); err != nil {
			r.MetricsClient.ReconcileCluster(instance.Namespace, instance.Name, "BOOTSTRAP_REMOTE_CONFIG", metrics.SYNC_CLUSTER, metrics.FAIL, err.Error())
			return Requeue.Wrap(err, fmt.Sprintf("set remote bootstrap config on %s", addr))
		}
	}
	return nil
}

func sortRemoteMasters(nodes redisutil.Nodes) (redisutil.Nodes, error) {
	masters, err := nodes.GetNodesByFunc(func(node *redisutil.Node) bool {
		return redisutil.IsMasterWithSlot(node)
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(masters, func(i, j int) bool {
		if len(masters[i].Slots) == 0 {
			return false
		}
		if len(masters[j].Slots) == 0 {
			return true
		}
		return masters[i].Slots[0] < masters[j].Slots[0]
	})
	return masters, nil
}

func orderedBootstrapPods(pods []*corev1.Pod) ([]bootstrapPodRef, error) {
	refs := make([]bootstrapPodRef, 0, len(pods))
	shardRE := regexp.MustCompile(`shard(\d+)`)
	for _, pod := range pods {
		shardMatch := shardRE.FindStringSubmatch(pod.Name)
		if len(shardMatch) < 2 {
			return nil, fmt.Errorf("unable to extract shard index from pod %s", pod.Name)
		}
		shardIndex, err := strconv.Atoi(shardMatch[1])
		if err != nil {
			return nil, fmt.Errorf("invalid shard index in pod %s: %w", pod.Name, err)
		}
		ordinal, err := statefulPodOrdinal(pod.Name)
		if err != nil {
			return nil, err
		}
		port := redisutil.DefaultRedisPort
		for _, container := range pod.Spec.Containers {
			if container.Name == "redis" {
				for _, p := range container.Ports {
					if p.Name == "client" {
						port = fmt.Sprintf("%d", p.ContainerPort)
					}
				}
			}
		}
		refs = append(refs, bootstrapPodRef{
			pod:        pod,
			shardIndex: shardIndex,
			ordinal:    ordinal,
			ipPort:     net.JoinHostPort(pod.Status.PodIP, port),
		})
	}
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].ordinal != refs[j].ordinal {
			return refs[i].ordinal < refs[j].ordinal
		}
		return refs[i].shardIndex < refs[j].shardIndex
	})
	return refs, nil
}

func statefulPodOrdinal(podName string) (int, error) {
	parts := strings.Split(podName, "-")
	if len(parts) == 0 {
		return 0, fmt.Errorf("invalid pod name %s", podName)
	}
	return strconv.Atoi(parts[len(parts)-1])
}

func (r *DistributedRedisClusterReconciler) meetBootstrapPods(ctx context.Context, syncCtx *syncContext, instance *redisv1alpha1.DistributedRedisCluster, seedAddr string, remoteNodes redisutil.Nodes, localPods []bootstrapPodRef) error {
	knownConnected := make(map[string]bool, len(remoteNodes))
	for _, n := range remoteNodes {
		if n != nil && n.LinkState == redisutil.RedisLinkStateConnected {
			knownConnected[n.IP] = true
		}
	}

	for _, local := range localPods {
		ip, port, err := net.SplitHostPort(local.ipPort)
		if err != nil {
			return Redis.Wrap(err, "split local pod address")
		}
		if err := syncCtx.admin.Connections().Add(ctx, local.ipPort); err != nil {
			return Redis.Wrap(err, fmt.Sprintf("connect local pod %s", local.pod.Name))
		}
		if knownConnected[ip] {
			continue
		}
		if err := syncCtx.admin.MeetNodeFromAddr(ctx, seedAddr, ip, port, instance.Namespace, instance.Name); err != nil {
			r.MetricsClient.ReconcileCluster(instance.Namespace, instance.Name, "BOOTSTRAP_MEET_SEED", metrics.SYNC_CLUSTER, metrics.FAIL, err.Error())
			return Redis.Wrap(err, fmt.Sprintf("seed meet local pod %s", local.pod.Name))
		}
		for _, remote := range remoteNodes {
			if remote == nil || remote.IP == ip {
				continue
			}
			if remote.LinkState != redisutil.RedisLinkStateConnected {
				// Mirror applyRemoteBootstrapConfig/verifyBootstrapJoin: don't attempt to introduce a
				// local pod to a remote the seed already reports as unreachable (e.g. a stale/ghost
				// entry left behind after a pod restart) — every pass would otherwise keep re-issuing
				// a handshake toward a dead IP.
				continue
			}
			if err := syncCtx.admin.MeetNodeFromAddr(ctx, local.ipPort, remote.IP, remote.Port, instance.Namespace, instance.Name); err != nil {
				r.MetricsClient.ReconcileCluster(instance.Namespace, instance.Name, "BOOTSTRAP_MEET_REMOTE", metrics.SYNC_CLUSTER, metrics.FAIL, err.Error())
				return Redis.Wrap(err, fmt.Sprintf("local pod %s meet remote %s", local.pod.Name, remote.IPPort()))
			}
		}
	}
	return nil
}

func verifyBootstrapJoin(ctx context.Context, syncCtx *syncContext, instance *redisv1alpha1.DistributedRedisCluster, remoteNodes redisutil.Nodes, localPods []bootstrapPodRef) error {
	localIPs := make(map[string]struct{}, len(localPods))
	for _, local := range localPods {
		ip, _, _ := net.SplitHostPort(local.ipPort)
		localIPs[ip] = struct{}{}
	}

	verified := false
	for _, remote := range remoteNodes {
		if remote == nil {
			continue
		}
		if remote.LinkState != redisutil.RedisLinkStateConnected {
			// Mirror applyRemoteBootstrapConfig: the seed already reports this node as unreachable
			// (e.g. a pod deleted/recreated with a new IP, leaving a stale entry until something
			// CLUSTER FORGETs it). Trying to query it every pass would wedge the join check forever.
			continue
		}
		addr := remote.IPPort()
		nodeInfos, err := syncCtx.admin.GetClusterNodeInfos(ctx, addr, instance.Namespace, instance.Name)
		if err != nil {
			return Requeue.Wrap(err, fmt.Sprintf("get cluster nodes from %s", addr))
		}
		knownNodes := make(map[string]*redisutil.Node)
		addKnown := func(n *redisutil.Node) {
			if existing, ok := knownNodes[n.IP]; ok && existing.LinkState == redisutil.RedisLinkStateConnected {
				return
			}
			knownNodes[n.IP] = n
		}
		for _, friend := range nodeInfos.Friends {
			addKnown(friend)
		}
		if nodeInfos.Node != nil {
			addKnown(nodeInfos.Node)
		}
		for ip := range localIPs {
			node, ok := knownNodes[ip]
			if !ok {
				return Requeue.New(fmt.Sprintf("local pod ip %s not yet visible on node %s", ip, addr))
			}
			if node.LinkState != redisutil.RedisLinkStateConnected {
				return Requeue.New(fmt.Sprintf("local pod ip %s link state %q (want %q) on node %s", ip, node.LinkState, redisutil.RedisLinkStateConnected, addr))
			}
		}
		verified = true
	}
	if !verified {
		// Every remote was skipped as disconnected (or remoteNodes was empty) — nothing was
		// actually checked, so falling through to a bare "nil" here would report a successful
		// join without having verified anything. Requeue instead of passing vacuously.
		return Requeue.New("no remote node was reachable to verify bootstrap join (all skipped as disconnected)")
	}
	return nil
}

func (r *DistributedRedisClusterReconciler) replicateBootstrapPods(ctx context.Context, syncCtx *syncContext, instance *redisv1alpha1.DistributedRedisCluster, remoteMasters redisutil.Nodes, localPods []bootstrapPodRef) error {
	// Progress is inferred from live Redis state each reconcile (CLUSTER NODES + INFO replication),
	// not from CR status. Within a shard only one pod runs CLUSTER REPLICATE at a time so we do not
	// overload a single master's replication buffer, but shards are advanced independently in the same
	// pass so at most one sync per shard runs in parallel across all shards and the step completes fast.

	// Group pods by shard, preserving the ordinal ordering from orderedBootstrapPods.
	shardPods := make(map[int][]bootstrapPodRef)
	shardOrder := make([]int, 0)
	for _, local := range localPods {
		if local.shardIndex >= len(remoteMasters) {
			// Defensive: syncBootstrapCluster already gates on len(remoteMasters) == MasterSize
			// before reaching here, so this should be unreachable in practice — but if the seed's
			// master count changes transiently between that check and this one, Requeue rather than
			// hard-fail, same reasoning as the caller's own gate.
			err := Requeue.New(fmt.Sprintf("shard index %d exceeds remote master count", local.shardIndex))
			syncCtx.reqLogger.Error(err, "bootstrap replicate: shard index out of range", "pod", local.pod.Name, "shardIndex", local.shardIndex, "remoteMasterCount", len(remoteMasters))
			return err
		}
		if _, ok := shardPods[local.shardIndex]; !ok {
			shardOrder = append(shardOrder, local.shardIndex)
		}
		shardPods[local.shardIndex] = append(shardPods[local.shardIndex], local)
	}

	var pending []string
	for _, shardIndex := range shardOrder {
		targetMaster := remoteMasters[shardIndex]
		// Advance a single shard: skip ready replicas and act on the first pod that is not yet a
		// ready replica of the target master. That is the one in-flight sync allowed for this shard.
		for _, local := range shardPods[shardIndex] {
			nodeInfos, err := syncCtx.admin.GetClusterNodeInfos(ctx, local.ipPort, instance.Namespace, instance.Name)
			if err != nil {
				syncCtx.reqLogger.Error(err, "bootstrap replicate: get node info failed", "pod", local.pod.Name, "addr", local.ipPort)
				return Redis.Wrap(err, fmt.Sprintf("get node info for %s", local.pod.Name))
			}
			if nodeInfos.Node != nil && nodeInfos.Node.Role == redisutil.RedisMasterRole && nodeInfos.Node.TotalSlots() > 0 {
				err := StopRetry.New(fmt.Sprintf("local pod %s is a master with slots; expected replica during bootstrap", local.pod.Name))
				syncCtx.reqLogger.Error(err, "bootstrap replicate: local pod unexpectedly a master with slots", "pod", local.pod.Name, "shardIndex", shardIndex)
				return err
			}
			if nodeInfos.Node != nil && nodeInfos.Node.MasterReferent == targetMaster.ID && nodeInfos.Node.Role == redisutil.RedisSlaveRole {
				status, _, err := syncCtx.admin.GetReplicationStatus(local.ipPort, ctx, instance.Namespace, instance.Name)
				if err == nil && status == redisutil.ReplicaStatusReady {
					continue
				}
				// This pod is mid-sync; this shard is busy, do not start another pod in it this pass.
				pending = append(pending, fmt.Sprintf("%s replicating from %s", local.pod.Name, targetMaster.IPPort()))
				break
			}
			if nodeInfos.Node == nil {
				err := Redis.New(fmt.Sprintf("missing node info for local pod %s", local.pod.Name))
				syncCtx.reqLogger.Error(err, "bootstrap replicate: missing node info", "pod", local.pod.Name, "addr", local.ipPort)
				return err
			}
			if err := syncCtx.admin.AttachSlaveToMaster(ctx, nodeInfos.Node, targetMaster.ID, instance.Namespace, instance.Name); err != nil {
				r.MetricsClient.ReconcileCluster(instance.Namespace, instance.Name, "BOOTSTRAP_REPLICATE", metrics.SYNC_CLUSTER, metrics.FAIL, err.Error())
				syncCtx.reqLogger.Error(err, "bootstrap replicate: attach slave to master failed", "pod", local.pod.Name, "master", targetMaster.IPPort())
				return Redis.Wrap(err, fmt.Sprintf("replicate %s to master %s", local.pod.Name, targetMaster.IPPort()))
			}
			syncCtx.reqLogger.Info("bootstrap replicate: started replication", "pod", local.pod.Name, "shardIndex", shardIndex, "master", targetMaster.IPPort())
			pending = append(pending, fmt.Sprintf("started %s from %s", local.pod.Name, targetMaster.IPPort()))
			break
		}
	}

	if len(pending) > 0 {
		syncCtx.reqLogger.Info("bootstrap replicate: replication in progress", "pending", pending)
		return Requeue.New(fmt.Sprintf("bootstrap replication in progress: %s", strings.Join(pending, "; ")))
	}
	syncCtx.reqLogger.Info("bootstrap replicate: all shards replicating and ready")
	return nil
}

func buildBootstrapRemoteMasters(admin redisutil.IAdmin, ctx context.Context, instance *redisv1alpha1.DistributedRedisCluster) (redisutil.Nodes, error) {
	seedAddr := net.JoinHostPort(instance.Spec.BootstrapNode.Host, instance.BootstrapSeedPort())
	if err := admin.Connections().Add(ctx, seedAddr); err != nil {
		return nil, err
	}
	seedInfos, err := admin.GetClusterNodeInfos(ctx, seedAddr, instance.Namespace, instance.Name)
	if err != nil {
		return nil, err
	}
	remoteNodes := append(seedInfos.Friends, seedInfos.Node)
	return sortRemoteMasters(remoteNodes)
}

func bootstrapProgress(localPods []bootstrapPodRef, ready int, stage string) string {
	return fmt.Sprintf("%s: %d/%d pods", stage, ready, len(localPods))
}

func expectedBootstrapMasterID(shardIndex int, remoteMasters redisutil.Nodes) (string, error) {
	if shardIndex < 0 || shardIndex >= len(remoteMasters) {
		return "", fmt.Errorf("invalid shard index %d", shardIndex)
	}
	return remoteMasters[shardIndex].ID, nil
}

func countBootstrapReadyReplicas(admin redisutil.IAdmin, ctx context.Context, instance *redisv1alpha1.DistributedRedisCluster, localPods []bootstrapPodRef, remoteMasters redisutil.Nodes) (int, string) {
	ready := 0
	var reasons []string
	for _, local := range localPods {
		masterID, err := expectedBootstrapMasterID(local.shardIndex, remoteMasters)
		if err != nil {
			reasons = append(reasons, err.Error())
			continue
		}
		nodeInfos, err := admin.GetClusterNodeInfos(ctx, local.ipPort, instance.Namespace, instance.Name)
		if err != nil {
			reasons = append(reasons, fmt.Sprintf("%s: %v", local.pod.Name, err))
			continue
		}
		if nodeInfos.Node == nil || nodeInfos.Node.MasterReferent != masterID || nodeInfos.Node.Role != redisutil.RedisSlaveRole {
			reasons = append(reasons, fmt.Sprintf("%s: not replicating expected master", local.pod.Name))
			continue
		}
		status, _, err := admin.GetReplicationStatus(local.ipPort, ctx, instance.Namespace, instance.Name)
		if err != nil || status != redisutil.ReplicaStatusReady {
			reasons = append(reasons, fmt.Sprintf("%s: replication not ready", local.pod.Name))
			continue
		}
		ready++
	}
	return ready, strings.Join(reasons, "; ")
}
