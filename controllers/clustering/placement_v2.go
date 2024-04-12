package clustering

import (
	"context"
	"fmt"
	"sort"

	"github.com/go-logr/logr"

	redisv1alpha1 "github.com/dinesh-murugiah/rediscluster-operator/api/v1alpha1"
	"github.com/dinesh-murugiah/rediscluster-operator/resources/statefulsets"
	"github.com/dinesh-murugiah/rediscluster-operator/utils/redisutil"
)

type Ctx struct {
	log                  logr.Logger
	expectedMasterNum    int
	expectedReplicaCount int
	clusterName          string
	cluster              *redisutil.Cluster
	nodes                map[string]redisutil.Nodes
	currentMasters       redisutil.Nodes
	newMastersBySts      map[string]*redisutil.Node
	slavesByMaster       map[string]redisutil.Nodes
	bestEffort           bool
	haconfig             *redisv1alpha1.HAspec
	hastatus             redisv1alpha1.HaStatus
}

func NewCtx(haconf *redisv1alpha1.HAspec, cluster *redisutil.Cluster, nodes redisutil.Nodes, masterNum int32, replicaCount int32, clusterName string, log logr.Logger) *Ctx {
	ctx := &Ctx{
		log:                  log,
		expectedMasterNum:    int(masterNum),
		expectedReplicaCount: int(replicaCount),
		clusterName:          clusterName,
		cluster:              cluster,
		slavesByMaster:       make(map[string]redisutil.Nodes),
		newMastersBySts:      make(map[string]*redisutil.Node),
		haconfig:             haconf,
		hastatus:             cluster.HaStatus,
	}
	ctx.nodes = ctx.sortRedisNodeByStatefulSet(nodes)
	return ctx
}

func (c *Ctx) sortRedisNodeByStatefulSet(nodes redisutil.Nodes) map[string]redisutil.Nodes {
	nodesByStatefulSet := make(map[string]redisutil.Nodes)

	if c.currentMasters == nil {
		c.currentMasters = make(redisutil.Nodes, 0)
	}

	for _, rNode := range nodes {
		cNode, err := c.cluster.GetNodeByID(rNode.ID)
		if err != nil {
			c.log.Error(err, "[sortRedisNodeByStatefulSet] unable fo found the Cluster.Node with redis", "ID", rNode.ID)
			continue // if not then next line with cNode.Pod will cause a panic since cNode is nil
		}
		ssName := unknownVMName
		if cNode.StatefulSet != "" {
			ssName = cNode.StatefulSet
		}
		if _, ok := nodesByStatefulSet[ssName]; !ok {
			nodesByStatefulSet[ssName] = redisutil.Nodes{}
		}
		nodesByStatefulSet[ssName] = append(nodesByStatefulSet[ssName], rNode)
		if (rNode.GetRole() == redisv1alpha1.RedisClusterNodeRoleMaster) && rNode.TotalSlots() > 0 {
			c.currentMasters = append(c.currentMasters, rNode)
		}
	}
	// sort the current masters by StatefulSet name
	sort.Slice(c.currentMasters, func(i, j int) bool {
		return c.currentMasters[i].StatefulSet < c.currentMasters[j].StatefulSet
	})

	return nodesByStatefulSet
}

func (c *Ctx) DispatchMasters() error {
	zoneRankingInfo := NewRankingInfo(c.haconfig, c.nodes, int32(c.expectedMasterNum), int32(c.expectedReplicaCount), c.clusterName, c.log)
	zoneRankingInfo.ApplyZoneRankingPerNodeDistribution()
	for i := 0; i < c.expectedMasterNum; i++ {
		stsName := statefulsets.ClusterStatefulSetName(c.clusterName, i)
		nodes, ok := c.nodes[stsName]
		if !ok {
			return fmt.Errorf("missing statefulset %s", stsName)
		}
		currentMasterNodes := nodes.FilterByFunc(redisutil.IsMasterWithSlot)
		if len(currentMasterNodes) == 0 {
			var master *redisutil.Node = nil
			if c.haconfig.HaEnabled {
				if c.hastatus != redisv1alpha1.HaStatusCreating {
					c.log.Error(fmt.Errorf("ha status not as expected when creating master %s", stsName), "ha status", "unexpected", c.hastatus)
				}
				master = c.PlaceMastersHA(stsName, zoneRankingInfo)
			} else {
				master = c.PlaceMasters(stsName)
			}
			if master == nil {
				return fmt.Errorf("unable to place master for statefulset %s", stsName)
			} else {
				c.log.Info("place master", "statefulSet", stsName, "podname", master.PodName, "ip", master.IP, "node", master.NodeName, "zone", master.Zonename, "current role", master.GetRole())
			}
			c.newMastersBySts[stsName] = master
		} else if len(currentMasterNodes) == 1 {
			c.newMastersBySts[stsName] = currentMasterNodes[0]
		} else if len(currentMasterNodes) > 1 {
			c.log.Error(fmt.Errorf("split brain"), "fix manually", "statefulSet", stsName, "masters", currentMasterNodes)
			return fmt.Errorf("split brain: %s", stsName)
		}
	}

	return nil
}

func (c *Ctx) PlaceMastersHA(ssName string, zoneRankingInfo *RankingInfo) *redisutil.Node {

	masterNode, err := zoneRankingInfo.GetBestRankedNodeinAllZonesPerSTS(ssName)
	if err != nil || masterNode == nil {
		c.log.Error(err, "Error when trying to get master node with best rank ")
		return nil
	}
	err1 := zoneRankingInfo.ZoneRankReducePostmasterSelect(masterNode.ID, masterNode.Zonename)
	if err1 != nil {
		c.log.Error(err, "Error when trying to get master node with best rank ")
		return nil
	}

	return masterNode

}

func (c *Ctx) PlaceMasters(ssName string) *redisutil.Node {
	var allMasters redisutil.Nodes
	allMasters = append(allMasters, c.currentMasters...)
	for _, master := range c.newMastersBySts {
		allMasters = append(allMasters, master)
	}
	nodes := c.nodes[ssName]
	for _, cNode := range nodes {
		_, err := allMasters.GetNodesByFunc(func(node *redisutil.Node) bool {
			return node.NodeName == cNode.NodeName
		})
		if err != nil {
			return cNode
		}
	}
	c.bestEffort = true
	c.log.Info("the pod are not spread enough on VMs to have only one master by VM", "select", nodes[0].IP)
	return nodes[0]
}

func (c *Ctx) PlaceSlaves() error {
	c.bestEffort = true
	for ssName, nodes := range c.nodes {
		master := c.newMastersBySts[ssName]
		for _, node := range nodes {
			if node.IP == master.IP {
				continue
			}
			if node.NodeName != master.NodeName {
				c.bestEffort = false
			}
			if node.GetRole() == redisv1alpha1.RedisClusterNodeRoleSlave {
				if node.MasterReferent != master.ID {
					/*
						c.log.Error(nil, "master referent conflict", "node ip", node.IP,
							"current masterID", node.MasterReferent, "expect masterID", master.ID, "master IP", master.IP)
					*/
					c.log.Info("PlaceSlaves", "Trying to attach slave with node ip", node.IP, "to a master which was posisbily a slave earlier", master.IP)
					c.slavesByMaster[master.ID] = append(c.slavesByMaster[master.ID], node)
				}
				continue
			}
			c.slavesByMaster[master.ID] = append(c.slavesByMaster[master.ID], node)
		}
	}
	return nil
}

func (c *Ctx) GetCurrentMasters() redisutil.Nodes {
	return c.currentMasters
}

func (c *Ctx) GetNewMasters() redisutil.Nodes {
	nodes := make(redisutil.Nodes, 0, len(c.newMastersBySts))
	for _, node := range c.newMastersBySts {
		nodes = append(nodes, node)
	}
	if len(nodes) != 0 {
		sort.Slice(nodes, func(i, j int) bool {
			return nodes[i].StatefulSet < nodes[j].StatefulSet
		})
	}
	return nodes
}

func (c *Ctx) GetSlaves() map[string]redisutil.Nodes {
	return c.slavesByMaster
}

func (c *Ctx) GetAllSlaves() redisutil.Nodes {
	slavenodes := make(redisutil.Nodes, 0)
	if len(c.nodes) != 0 {
		for _, nodes := range c.nodes {
			for _, n := range nodes {
				if n.GetRole() == redisv1alpha1.RedisClusterNodeRoleSlave {
					slavenodes = append(slavenodes, n)
				}
			}
		}
	}
	return slavenodes
}

func (c *Ctx) GetStatefulsetNodes() map[string]redisutil.Nodes {
	return c.nodes
}

func (c *Ctx) GetNodestoScaledandRecover(newMasters redisutil.Nodes, curMasters redisutil.Nodes) (redisutil.Nodes, redisutil.Nodes) {
	isSlaveinSTS := false
	recoverMasters := make(redisutil.Nodes, 0)
	scaleUpMasters := make(redisutil.Nodes, 0)
	var stsNodes redisutil.Nodes = make(redisutil.Nodes, 0)

	for _, newMaster := range newMasters {
		//First check if the new master is already in the current masters if so continue to the next new master
		if _, err := curMasters.GetNodeByID(newMaster.ID); err == nil {
			continue
		}
		isSlaveinSTS = false
		//If the new master is not in the current masters, then check if there are any slaves in the STS where the new master is placed
		stsNodes = c.nodes[newMaster.StatefulSet]
		for _, node := range stsNodes {

			if node.GetRole() == redisv1alpha1.RedisClusterNodeRoleSlave {
				isSlaveinSTS = true
				break
			}
		}
		//If the new master is not in the current masters and there are no slaves in the STS where the new master is placed, then add the new master to the scaled up masters list
		if !isSlaveinSTS {
			c.log.Info("New Shard is scaled up", "STS name", newMaster.StatefulSet, "New Master IP", newMaster.IP)
			scaleUpMasters = append(scaleUpMasters, newMaster)
		} else {
			c.log.Info("there are slaves in the STS where the new master is placed , this STS requires recovery", "NEW Master IP", newMaster.IP, "STS", newMaster.StatefulSet)
			recoverMasters = append(recoverMasters, newMaster)
		}

	}
	return scaleUpMasters, recoverMasters
}

func (c *Ctx) CheckAndHealCluster(admin redisutil.IAdmin, ctx context.Context) error {

	curMasters := c.GetCurrentMasters()
	newMasters := c.GetNewMasters()
	curSlaves := c.GetAllSlaves()

	//If GetCurrentMasters is equal to expected master , then healings are not required
	if len(curMasters) == c.expectedMasterNum {
		return nil
	}
	//If this is the first time cluster is being created, then no healing is required
	//First time cluster creation is identified by the fact that there are no current masters
	//and new masters are equal to expected master number  and hastatus is creating and there are no slaves in the clusters
	if len(curMasters) == 0 && len(newMasters) == c.expectedMasterNum && len(curSlaves) == 0 {
		return nil
	}

	//If the current masters is less than the new masters and if not a new cluster creation
	// then it could be one of the 2 possibilities , either a new masters have scaled up or
	//there is a cluster failure which might need a recovery get the list of masters which needs recovery and the list of masters which needs to be scaled up
	if len(curMasters) < len(newMasters) {

		_, recoverMasters := c.GetNodestoScaledandRecover(newMasters, curMasters)

		if len(recoverMasters) == 0 {
			return nil
		}
		//Now we have got a list of  masters which needs recovery we should recover only when the current master count is less than the expected number of masters to reach quorum
		//If there is a quorum then we should not recover the masters and let the cluster recover by itself
		if len(curMasters) > ((c.expectedMasterNum / 2) + 1) {
			c.log.Info("Current masters are more than quorum, no need to recover the cluster", "Current Master Count", len(curMasters))
			return HealReconcile.New("quorum available, Wait for cluster to recover")
		}
		c.log.Info("Recovering the cluster", "RecoverMasters", recoverMasters)
		zoneRankingInfo := NewRankingInfo(c.haconfig, c.nodes, int32(c.expectedMasterNum), int32(c.expectedReplicaCount), c.clusterName, c.log)
		zoneRankingInfo.ApplyZoneRankingPerNodeDistribution()
		for _, rnode := range recoverMasters {

			promoteNode, err := zoneRankingInfo.GetBestRankedNodeinAllZonesPerSTS(rnode.StatefulSet)
			if err != nil {
				c.log.Error(err, "Error when trying to get master node with best rank ")
				return HealFailure.Wrap(err, "Error when trying to get master node with best rank")
			}
			if promoteNode.GetRole() != redisv1alpha1.RedisClusterNodeRoleSlave {
				c.log.Error(fmt.Errorf("node to recover is not a slave"), "Node is not a slave", "Node IP", rnode.IP)
				return HealFailure.New("node to recover is not a slave")
			}
			//Here we are assuming that slave chosen by ranking algorithm is the best slave to promote as master
			errFailover := admin.FailoverSlaveToMaster(ctx, promoteNode, redisutil.RedisFailoverTakeover)
			if errFailover != nil {
				c.log.Error(err, "Error while recovering the cluster by failover a slave to master", "Slave IP", rnode.IP)
				return HealFailure.Wrap(errFailover, "Failover Takeover Failed")
			}
			c.log.Info("Cluster Recovery Sucessful by failover", "New Master IP", rnode.IP)

		}

	}
	err := fmt.Errorf("cluster Recovery Sucessful")
	return HealReconcile.Wrap(err, "checkAndHealCluster - cluster Recovery Sucessful by failover")
}
