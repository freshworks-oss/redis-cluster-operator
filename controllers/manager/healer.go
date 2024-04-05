package manager

import (
	"context"
	"fmt"
	"time"

	redisv1alpha1 "github.com/dinesh-murugiah/rediscluster-operator/api/v1alpha1"
	"github.com/dinesh-murugiah/rediscluster-operator/controllers/heal"
	"github.com/dinesh-murugiah/rediscluster-operator/utils/redisutil"
)

type IHeal interface {
	Heal(cluster *redisv1alpha1.DistributedRedisCluster, admin redisutil.IAdmin, context context.Context) (bool, error)
	FixTerminatingPods(cluster *redisv1alpha1.DistributedRedisCluster, maxDuration time.Duration) (bool, error)
}

type realHeal struct {
	*heal.CheckAndHeal
}

func NewHealer(heal *heal.CheckAndHeal) IHeal {
	return &realHeal{heal}
}

func (h *realHeal) Heal(cluster *redisv1alpha1.DistributedRedisCluster, admin redisutil.IAdmin, context context.Context) (bool, error) {

	clusterInfos, clusterinfoerr := admin.GetClusterInfos(context)
	if clusterinfoerr != nil {
		if clusterErr, ok := clusterinfoerr.(redisutil.ClusterInfosError); ok {
			if clusterErr.Partial() {
				return true, fmt.Errorf("ClusterInfosPartial")
			}
		} else {
			return false, Redis.Wrap(clusterinfoerr, "GetClusterInfos")
		}
	}

	if actionDone, err := h.FixFailedNodes(cluster, clusterInfos, admin, context); err != nil {
		return actionDone, err
	} else if actionDone {
		return actionDone, nil
	}

	if actionDone, err := h.FixUntrustedNodes(cluster, clusterInfos, admin, context); err != nil {
		return actionDone, err
	} else if actionDone {
		return actionDone, nil
	}
	return false, nil
}
