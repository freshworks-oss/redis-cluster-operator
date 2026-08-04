package distributedrediscluster

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"

	redisv1alpha1 "github.com/freshworks/redis-cluster-operator/api/v1alpha1"
	utils "github.com/freshworks/redis-cluster-operator/utils/commonutils"
	"github.com/freshworks/redis-cluster-operator/utils/redisutil"
)

// replicaNoFailoverStatus summarizes CONFIG GET cluster-replica-no-failover on local pods.
type replicaNoFailoverStatus struct {
	Ready   int
	Total   int
	Pending []string
}

func SetBootstrapStatus(status *redisv1alpha1.DistributedRedisClusterStatus, phase redisv1alpha1.BootstrapPhase, progress, reason string) {
	status.Bootstrap.Phase = phase
	status.Bootstrap.Progress = progress
	status.Bootstrap.Reason = reason
}

func CheckBootstrapHealth(status *redisv1alpha1.DistributedRedisClusterStatus, cluster *redisv1alpha1.DistributedRedisCluster, admin redisutil.IAdmin, ctx context.Context, localPods []*corev1.Pod, reqLogger logr.Logger) bool {
	if !cluster.BootstrapMode() {
		return false
	}

	orderedPods, err := orderedBootstrapPods(localPods)
	if err != nil {
		SetBootstrapStatus(status, redisv1alpha1.BootstrapPhaseFailed, "", err.Error())
		SetClusterNOK(status, err.Error())
		return false
	}

	if !cluster.BootstrapReconcileEnabled() {
		podsToCheck := podsForReplicaNoFailoverCheck(status, orderedPods)
		noFailover := verifyLocalReplicaNoFailover(admin, ctx, cluster, podsToCheck)
		return checkBootstrapMigrationStatus(status, cluster, noFailover)
	}

	remoteMasters, err := buildBootstrapRemoteMasters(admin, ctx, cluster)
	if err != nil {
		SetBootstrapStatus(status, redisv1alpha1.BootstrapPhaseInProgress, "discovering remote masters", err.Error())
		SetClusterNOK(status, err.Error())
		return false
	}

	ready, reason := countBootstrapReadyReplicas(admin, ctx, cluster, orderedPods, remoteMasters)
	progress := bootstrapProgress(orderedPods, ready, "replication")

	if ready == len(orderedPods) {
		SetBootstrapStatus(status, redisv1alpha1.BootstrapPhaseHealthy, progress, "all local pods are replicas of expected remote masters")
		SetClusterOK(status, "bootstrap replication healthy")
		return true
	}

	SetBootstrapStatus(status, redisv1alpha1.BootstrapPhaseInProgress, progress, reason)
	SetClusterNOK(status, reason)
	return false
}

// verifyLocalReplicaNoFailover reads cluster-replica-no-failover on the given local pods (report-only).
func verifyLocalReplicaNoFailover(admin redisutil.IAdmin, ctx context.Context, cluster *redisv1alpha1.DistributedRedisCluster, podsToCheck []bootstrapPodRef) replicaNoFailoverStatus {
	result := replicaNoFailoverStatus{Total: len(podsToCheck)}
	for _, local := range podsToCheck {
		if err := admin.Connections().Add(ctx, local.ipPort); err != nil {
			result.Pending = append(result.Pending, local.pod.Name+"(unreachable)")
			continue
		}
		client, err := admin.Connections().Get(ctx, local.ipPort)
		if err != nil {
			result.Pending = append(result.Pending, local.pod.Name+"(unreachable)")
			continue
		}
		cfg, err := admin.GetAllConfig(ctx, client, local.ipPort, cluster.Namespace, cluster.Name)
		if err != nil {
			result.Pending = append(result.Pending, local.pod.Name+"(config error)")
			continue
		}
		if strings.EqualFold(cfg[bootstrapReplicaNoFailover], bootstrapReplicaNoFailoverDisabled) {
			result.Ready++
			continue
		}
		result.Pending = append(result.Pending, local.pod.Name)
	}
	return result
}

// podsForReplicaNoFailoverCheck selects pods on statefulsets that already have a local shard master.
// When there are no local masters yet, no config verification is performed.
func podsForReplicaNoFailoverCheck(status *redisv1alpha1.DistributedRedisClusterStatus, orderedPods []bootstrapPodRef) []bootstrapPodRef {
	if status.NumberOfMaster == 0 {
		return nil
	}
	masterStatefulSets := statefulSetsWithLocalMaster(status.Nodes)
	if len(masterStatefulSets) == 0 {
		return nil
	}
	out := make([]bootstrapPodRef, 0, len(orderedPods))
	for _, ref := range orderedPods {
		if _, ok := masterStatefulSets[podStatefulSetName(ref.pod)]; ok {
			out = append(out, ref)
		}
	}
	return out
}

func statefulSetsWithLocalMaster(nodes []redisv1alpha1.RedisClusterNode) map[string]struct{} {
	out := make(map[string]struct{})
	for _, node := range nodes {
		if node.Role != redisv1alpha1.RedisClusterNodeRoleMaster || len(node.Slots) == 0 || node.StatefulSet == "" {
			continue
		}
		out[node.StatefulSet] = struct{}{}
	}
	return out
}

func podStatefulSetName(pod *corev1.Pod) string {
	for _, ref := range pod.OwnerReferences {
		if ref.Kind == "StatefulSet" {
			return ref.Name
		}
	}
	return ""
}

// checkBootstrapMigrationStatus reports cutover progress when bootstrap reconcile is disabled.
// Local masters are counted from status.NumberOfMaster (local pods with slots), populated by buildClusterStatus.
func checkBootstrapMigrationStatus(status *redisv1alpha1.DistributedRedisClusterStatus, cluster *redisv1alpha1.DistributedRedisCluster, noFailover replicaNoFailoverStatus) bool {
	expected := cluster.Spec.MasterSize
	current := status.NumberOfMaster
	progress := bootstrapMigrationProgress(current, expected, noFailover)

	if current > expected {
		reason := bootstrapMigrationReason(current, expected, noFailover, "local shard masters %d exceeds expected %d", current, expected)
		SetBootstrapStatus(status, redisv1alpha1.BootstrapPhaseFailed, progress, reason)
		SetClusterNOK(status, reason)
		return false
	}
	if current == expected {
		reason := bootstrapMigrationReason(current, expected, noFailover, "cutover complete: %d of %d expected shard masters are local", current, expected)
		SetBootstrapStatus(status, redisv1alpha1.BootstrapPhaseMigrated, progress, reason)
		SetClusterOK(status, "bootstrap migration complete")
		return true
	}

	reason := bootstrapMigrationReason(current, expected, noFailover, "cutover in progress: %d of %d expected shard masters are local", current, expected)
	SetBootstrapStatus(status, redisv1alpha1.BootstrapPhaseMigrating, progress, reason)
	SetClusterNOK(status, reason)
	return false
}

func bootstrapMasterProgress(current, expected int32) string {
	return fmt.Sprintf("masters: %d/%d", current, expected)
}

func bootstrapMigrationProgress(current, expected int32, noFailover replicaNoFailoverStatus) string {
	if noFailover.Total == 0 {
		return bootstrapMasterProgress(current, expected)
	}
	return fmt.Sprintf("%s; nofailover: %d/%d", bootstrapMasterProgress(current, expected), noFailover.Ready, noFailover.Total)
}

func bootstrapMigrationReason(current, expected int32, noFailover replicaNoFailoverStatus, format string, args ...interface{}) string {
	reason := fmt.Sprintf(format, args...)
	if noFailover.Total == 0 {
		return reason
	}
	reason += fmt.Sprintf("; cluster-replica-no-failover=no on %d/%d pods on statefulsets with a local master", noFailover.Ready, noFailover.Total)
	if len(noFailover.Pending) > 0 {
		reason += fmt.Sprintf(" (pending: %s)", strings.Join(noFailover.Pending, ", "))
	}
	return reason
}

func compareBootstrapStatus(old, new redisv1alpha1.BootstrapStatus, reqLogger logr.Logger) bool {
	if utils.CompareStringValue("BootstrapPhase", string(old.Phase), string(new.Phase), reqLogger) {
		return true
	}
	if utils.CompareStringValue("BootstrapProgress", old.Progress, new.Progress, reqLogger) {
		return true
	}
	if utils.CompareStringValue("BootstrapReason", old.Reason, new.Reason, reqLogger) {
		return true
	}
	return false
}
