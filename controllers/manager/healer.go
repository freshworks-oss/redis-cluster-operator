package manager

import (
	"context"
	"time"

	redisv1alpha1 "github.com/freshworks/redis-cluster-operator/api/v1alpha1"
	"github.com/freshworks/redis-cluster-operator/controllers/heal"
	"github.com/freshworks/redis-cluster-operator/metrics"
	"github.com/freshworks/redis-cluster-operator/utils/redisutil"
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

	var stopReconcile bool = false

	if cluster.BootstrapMode() {
		return false, false, nil
	}

	if stopReconcile, actionDone, err := h.CheckandRecoverCluster(cluster, clusterInfos, admin, context); err != nil {
		h.MetricsClient.ReconcileCluster(cluster.Name, cluster.Namespace, "CHECK_AND_RECOVER_CLUSTER", metrics.CHECK_AND_HEAL, metrics.FAIL, err.Error())
		return stopReconcile, actionDone, err
	} else if actionDone {
		return stopReconcile, actionDone, nil
	}

	if actionDone, err := h.FixFailedNodes(cluster, clusterInfos, admin, context); err != nil {
		h.MetricsClient.ReconcileCluster(cluster.Name, cluster.Namespace, "FIX_FAILED_NODES", metrics.CHECK_AND_HEAL, metrics.FAIL, err.Error())
		return stopReconcile, actionDone, err
	} else if actionDone {
		return stopReconcile, actionDone, nil
	}

	if actionDone, err := h.FixUntrustedNodes(cluster, clusterInfos, admin, context); err != nil {
		h.MetricsClient.ReconcileCluster(cluster.Name, cluster.Namespace, "FIX_UNTRUSTED_NODES", metrics.CHECK_AND_HEAL, metrics.FAIL, err.Error())
		return stopReconcile, actionDone, err
	} else if actionDone {
		return stopReconcile, actionDone, nil
	}
	return stopReconcile, false, nil
}
