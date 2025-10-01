package heal

import (
	"context"

	"k8s.io/apimachinery/pkg/util/errors"

	redisv1alpha1 "github.com/freshworks/redis-cluster-operator/api/v1alpha1"
	"github.com/freshworks/redis-cluster-operator/utils/redisutil"
	"github.com/go-logr/logr"
)

// FixFailedNodes fix failed nodes: in some cases (cluster without enough master after crash or scale down), some nodes may still know about fail nodes
func (c *CheckAndHeal) FixFailedNodes(cluster *redisv1alpha1.DistributedRedisCluster, infos *redisutil.ClusterInfos, admin redisutil.IAdmin, context context.Context) (bool, error) {

	forgetSet := listGhostNodes(cluster, infos, c.Logger)
	var errs []error
	doneAnAction := false
	for id := range forgetSet {
		doneAnAction = true
		c.Logger.Info("[FixFailedNodes] Forgetting failed node, this command might fail, this is not an error", "node", id)
		if !c.DryRun {
			c.Logger.Info("[FixFailedNodes] try to forget node", "nodeId", id)
			err := admin.ForgetNode(context, id, cluster.Namespace, cluster.Name)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	return doneAnAction, errors.NewAggregate(errs)
}

// listGhostNodes : A Ghost node is a node still known by some redis node but which doesn't exists anymore
// meaning it is failed, and pod not in kubernetes, or without targetable IP
func listGhostNodes(cluster *redisv1alpha1.DistributedRedisCluster, infos *redisutil.ClusterInfos, log logr.Logger) map[string]bool {
	ghostNodesSet := map[string]bool{}
	ghostNodesDetect := map[string]int32{}
	if infos == nil || infos.Infos == nil {
		log.Info("No infos available")
		return ghostNodesSet
	}
	for _, nodeinfos := range infos.Infos {
		for _, node := range nodeinfos.Friends {
			// only forget it when no more part of kubernetes, or if noaddress
			if node.HasStatus(redisutil.NodeStatusNoAddr) {
				ghostNodesDetect[node.ID] += 1
			}
			//Pfail is a transient state and is a non majority declaration , this should not be used identifying a node as ghost
			//if node.HasStatus(redisutil.NodeStatusFail) || node.HasStatus(redisutil.NodeStatusPFail) {
			if node.HasStatus(redisutil.NodeStatusFail) {
				ghostNodesDetect[node.ID] += 1
			}
		}
	}

	for ghnodeid, count := range ghostNodesDetect {
		// if more than half of the cluster knows about it, it is a ghost
		if count >= ((cluster.Spec.ClusterReplicas+1)*cluster.Spec.MasterSize)/2 {
			ghostNodesSet[ghnodeid] = true
			log.Info("Node is ghost", "node", ghnodeid, "count", count)
		}
	}

	return ghostNodesSet
}
