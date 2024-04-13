package heal

import (
	"context"
	"fmt"

	redisv1alpha1 "github.com/dinesh-murugiah/rediscluster-operator/api/v1alpha1"
	"github.com/dinesh-murugiah/rediscluster-operator/controllers/clustering"
	"github.com/dinesh-murugiah/rediscluster-operator/utils/redisutil"
)

func (c *CheckAndHeal) CheckandRecoverCluster(cluster *redisv1alpha1.DistributedRedisCluster, infos *redisutil.ClusterInfos, admin redisutil.IAdmin, context context.Context) (bool, bool, error) {

	var stopReconcile bool = false
	haRanking := clustering.CreateRankingFromClusterInfo(cluster.Spec.HaConfig, infos, cluster.Spec.MasterSize, cluster.Spec.ClusterReplicas, cluster.Name, c.Logger)
	if haRanking == nil {
		errorR := fmt.Errorf("error Creating ranking object from cluster info")
		c.Logger.Error(errorR, "CheckandRecoverCluster", "error creating ranking object for", cluster.Name)
		return stopReconcile, false, errorR
	}

	// check if we need to recover the cluster
	doRecovery, recoverSTS := haRanking.IsClusterNeedsRecovery()
	if !doRecovery {
		stopReconcile = false
		return stopReconcile, false, nil
	}
	// if doRecovery flag is set but recoverSTS is empty, then we have to wait for cluster to heal itself
	if len(recoverSTS) == 0 {
		stopReconcile = false
		return stopReconcile, true, nil
	}

	haRanking.SetCurrentMastersZoneRank()
	// recover the cluster
	for _, stsName := range recoverSTS {
		promoteNode, err := haRanking.GetBestRankedNodeinAllZonesPerSTS(stsName)
		if err != nil {
			stopReconcile = true
			c.Logger.Error(err, "Error when trying to get master node with best rank ")
			return stopReconcile, false, err
		}
		if promoteNode.GetRole() != redisv1alpha1.RedisClusterNodeRoleSlave {
			c.Logger.Error(fmt.Errorf("node to recover is not a slave"), "Node is not a slave", "Node IP", promoteNode.IP)
			stopReconcile = true
			return stopReconcile, false, fmt.Errorf("node to recover is not a slave")
		}
		c.Logger.Info("Recovering the cluster by failover", "Promote Node IP", promoteNode.IP, "Promote Node role", promoteNode.GetRole(), "STS", stsName, "promote Node zone", promoteNode.Zonename)
		//Here we are assuming that slave chosen by ranking algorithm is the best slave to promote as master
		errFailover := admin.FailoverSlaveToMaster(context, promoteNode, redisutil.RedisFailoverTakeover)
		if errFailover != nil {
			c.Logger.Error(err, "Error while recovering the cluster by failover a slave to master", "Slave IP", promoteNode.IP)
			stopReconcile = true
			return stopReconcile, false, errFailover
		}
		c.Logger.Info("Cluster Recovery Sucessful by failover", "New Master IP", promoteNode.IP)

	}
	stopReconcile = false
	return stopReconcile, true, nil
}
