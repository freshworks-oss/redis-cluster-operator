package manager

import (
	"context"
	"time"

	redisv1alpha1 "github.com/dinesh-murugiah/rediscluster-operator/api/v1alpha1"
	"github.com/dinesh-murugiah/rediscluster-operator/controllers/heal"
	"github.com/dinesh-murugiah/rediscluster-operator/metrics"
	"github.com/dinesh-murugiah/rediscluster-operator/utils/redisutil"
)

type IHeal interface {
	Heal(cluster *redisv1alpha1.DistributedRedisCluster, clusterInfos *redisutil.ClusterInfos, admin redisutil.IAdmin, context context.Context) (bool, bool, error)
	FixTerminatingPods(cluster *redisv1alpha1.DistributedRedisCluster, maxDuration time.Duration) (bool, error)
}

type realHeal struct {
	*heal.CheckAndHeal
}

func NewHealer(heal *heal.CheckAndHeal) IHeal {
	return &realHeal{heal}
}

func (h *realHeal) Heal(cluster *redisv1alpha1.DistributedRedisCluster, clusterInfos *redisutil.ClusterInfos, admin redisutil.IAdmin, context context.Context) (bool, bool, error) {
	// Input validation
	if cluster == nil {
		return false, false, NoType.New("cluster cannot be nil")
	}
	if clusterInfos == nil {
		return false, false, NoType.New("clusterInfos cannot be nil")
	}
	if admin == nil {
		return false, false, NoType.New("admin cannot be nil")
	}
	if h.CheckAndHeal == nil {
		return false, false, NoType.New("CheckAndHeal is not initialized")
	}

	// Try each healing step in sequence
	healingSteps := []struct {
		name   string
		action func() (bool, bool, error)
	}{
		{
			name: "CHECK_AND_RECOVER_CLUSTER",
			action: func() (bool, bool, error) {
				return h.CheckandRecoverCluster(cluster, clusterInfos, admin, context)
			},
		},
		{
			name: "FIX_FAILED_NODES",
			action: func() (bool, bool, error) {
				actionDone, err := h.FixFailedNodes(cluster, clusterInfos, admin, context)
				return false, actionDone, err // FixFailedNodes doesn't return stopReconcile
			},
		},
		{
			name: "FIX_UNTRUSTED_NODES",
			action: func() (bool, bool, error) {
				actionDone, err := h.FixUntrustedNodes(cluster, clusterInfos, admin, context)
				return false, actionDone, err // FixUntrustedNodes doesn't return stopReconcile
			},
		},
	}

	for _, step := range healingSteps {
		stopReconcile, actionDone, err := step.action()
		if err != nil {
			h.recordFailure(cluster, step.name, err)
			return stopReconcile, actionDone, err
		}
		if actionDone {
			return stopReconcile, actionDone, nil
		}
	}

	return false, false, nil
}

// recordFailure records healing failure metrics
func (h *realHeal) recordFailure(cluster *redisv1alpha1.DistributedRedisCluster, operation string, err error) {
	if h.MetricsClient != nil {
		h.MetricsClient.ReconcileCluster(cluster.Name, cluster.Namespace, operation, metrics.CHECK_AND_HEAL, metrics.FAIL, err.Error())
	}
}

// FixTerminatingPods fixes pods that are stuck in terminating state
func (h *realHeal) FixTerminatingPods(cluster *redisv1alpha1.DistributedRedisCluster, maxDuration time.Duration) (bool, error) {
	// Input validation
	if cluster == nil {
		return false, NoType.New("cluster cannot be nil")
	}
	if h.CheckAndHeal == nil {
		return false, NoType.New("CheckAndHeal is not initialized")
	}

	// Delegate to the embedded CheckAndHeal
	return h.CheckAndHeal.FixTerminatingPods(cluster, maxDuration)
}
