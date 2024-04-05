package clustering

import (
	"fmt"
	"sort"

	redisv1alpha1 "github.com/dinesh-murugiah/rediscluster-operator/api/v1alpha1"
	"github.com/dinesh-murugiah/rediscluster-operator/resources/statefulsets"
	"github.com/dinesh-murugiah/rediscluster-operator/utils/redisutil"
	"github.com/go-logr/logr"
)

const (
	NodesCountsPerZoneRankingWeight  = 30
	ZoneRankingTieBreakerWeightAdd   = 2
	MasterPerZoneWeightDivision      = 2
	ExistingReplicasWeightMultiplier = 2
)

type RankingInfo struct {
	log                      logr.Logger
	clusterName              string
	desieredMasterCount      int
	desieredReplicaCount     int
	zoneList                 []string
	masternodeBySTS          map[string]*redisutil.Node
	replicaNodesBySTS        map[string]redisutil.Nodes
	nodesByZone              map[string]redisutil.Nodes
	NodesIndexedbyZoneAndSTS map[string]map[string][]*redisutil.Node
	BestRankedMasterBySTS    map[string]*redisutil.Node
	//currentMasters    redisutil.Nodes
	haconfig *redisv1alpha1.HAspec
}

// This function will create a new RankingInfo object and initialize the ranking info for the nodes data
// this is a wrapper function to create the ranking info from the cluster info
func CreateRankingFromClusterInfo(haconf *redisv1alpha1.HAspec, clusterinfo *redisutil.ClusterInfos, expectedMasterCount int32, expectedReplicaCount int32, clusterName string, log logr.Logger) *RankingInfo {

	var stsnodes map[string]redisutil.Nodes = make(map[string]redisutil.Nodes)
	var err error
	nodes := clusterinfo.GetNodes()
	if nodes == nil || len(nodes) < (int(expectedMasterCount)+int(expectedReplicaCount)) {
		log.Error(fmt.Errorf("error in clusterinfo"), "CreateRankingFromClusterInfo", "Info for Nodes not found or less than expected")
		return nil
	}

	stsnodes, err = sortRedisNodeByStatefulSet(nodes)
	if err != nil {
		log.Error(err, "unable to sort the nodes by statefulset")
		return nil
	}
	return NewRankingInfo(haconf, stsnodes, expectedMasterCount, expectedReplicaCount, clusterName, log)
}

// This function will create a new RankingInfo object and initialize the ranking info for the nodes
// This will also initialize the zone list from the haconfig and initialize the master and replica nodes by STS and Zone
// The input to this function is the haconfig, nodes and the expected master and replica count
// The output is the RankingInfo object initialised with the nodes information and the haconfig , but ranking not yet applied
// the nodes given as a input is expceted to be sorted by statefulset
func NewRankingInfo(haconf *redisv1alpha1.HAspec, nodes map[string]redisutil.Nodes, expectedMasterCount int32, expectedReplicaCount int32, clustername string, log logr.Logger) *RankingInfo {
	nri := &RankingInfo{
		log:                      log,
		clusterName:              clustername,
		desieredMasterCount:      int(expectedMasterCount),
		desieredReplicaCount:     int(expectedReplicaCount),
		zoneList:                 []string{},
		masternodeBySTS:          make(map[string]*redisutil.Node),
		replicaNodesBySTS:        make(map[string]redisutil.Nodes),
		nodesByZone:              make(map[string]redisutil.Nodes),
		NodesIndexedbyZoneAndSTS: make(map[string]map[string][]*redisutil.Node),
		haconfig:                 haconf,
		BestRankedMasterBySTS:    make(map[string]*redisutil.Node),
	}
	nri.InitRankingInfo(nodes)
	for zone := range haconf.ZonesInfo {
		nri.zoneList = append(nri.zoneList, zone)
	}
	sort.Strings(nri.zoneList)
	return nri
}

func (c *RankingInfo) InitRankingInfo(nodes map[string]redisutil.Nodes) {

	for _, stsnodes := range nodes {
		for _, rNode := range stsnodes {

			//Init the ranking to zero
			rNode.ZoneDistributionRank = 0
			rNode.NodeDistributionRank = 0
			rNode.ShardDistributionRank = 0

			// update the nodes by Zone
			if _, ok := c.nodesByZone[rNode.Zonename]; !ok {
				c.nodesByZone[rNode.Zonename] = redisutil.Nodes{}
			}
			c.nodesByZone[rNode.Zonename] = append(c.nodesByZone[rNode.Zonename], rNode)

			//Update the nodes with zone and STS index
			if _, ok := c.NodesIndexedbyZoneAndSTS[rNode.Zonename]; !ok {
				c.NodesIndexedbyZoneAndSTS[rNode.Zonename] = make(map[string][]*redisutil.Node)
			}
			if _, ok := c.NodesIndexedbyZoneAndSTS[rNode.Zonename][rNode.StatefulSet]; !ok {
				c.NodesIndexedbyZoneAndSTS[rNode.Zonename][rNode.StatefulSet] = make([]*redisutil.Node, 0)
			}
			c.NodesIndexedbyZoneAndSTS[rNode.Zonename][rNode.StatefulSet] = append(c.NodesIndexedbyZoneAndSTS[rNode.Zonename][rNode.StatefulSet], rNode)

			//Update the master nodes per STS
			if (rNode.GetRole() == redisv1alpha1.RedisClusterNodeRoleMaster) && rNode.TotalSlots() > 0 {
				c.masternodeBySTS[rNode.StatefulSet] = rNode
			} else if rNode.GetRole() == redisv1alpha1.RedisClusterNodeRoleMaster && rNode.TotalSlots() == 0 {
				c.log.Info("InitRankingInfo", "Master with no slots", rNode.ID)
				c.masternodeBySTS[rNode.StatefulSet] = nil
			}
			//Update the replica nodes per STS
			if _, ok := c.replicaNodesBySTS[rNode.StatefulSet]; !ok {
				c.replicaNodesBySTS[rNode.StatefulSet] = redisutil.Nodes{}
			}

			if rNode.GetRole() == redisv1alpha1.RedisClusterNodeRoleSlave && rNode.MasterReferent != "" {
				c.replicaNodesBySTS[rNode.StatefulSet] = append(c.replicaNodesBySTS[rNode.StatefulSet], rNode)
			} else if rNode.GetRole() == redisv1alpha1.RedisClusterNodeRoleSlave && rNode.MasterReferent == "" {
				c.log.Info("InitRankingInfo", "Slave without master", rNode.ID, "Statefulset", rNode.StatefulSet, "IP", rNode.IP)
				c.replicaNodesBySTS[rNode.StatefulSet] = append(c.replicaNodesBySTS[rNode.StatefulSet], rNode)
			}
		}
	}

}

func (c *RankingInfo) ApplyZoneRankingPerNodeDistribution() {
	for _, nodes := range c.nodesByZone {
		if len(nodes) == 0 {
			continue
		}
		for _, node := range nodes {
			//For all the nodes in the zone, we will add the weight based on the number of nodes in the zone
			node.ZoneDistributionRank = node.ZoneDistributionRank + (NodesCountsPerZoneRankingWeight / len(nodes))
			//If the node is a slave without a master, and gossip is connected, then we will add weight to it
			// This is needed to rank this as a potential master on multi zone failures or when there are no masters for a shard and we need to promote a slave
			if node.GetRole() == redisv1alpha1.RedisClusterNodeRoleSlave && node.LinkState == redisutil.RedisLinkStateConnected {
				//Check if this statefulset has a master with slots or not, if not then add more weight to this node as this slave is a potential candidate for master if cluster is not able to find a master
				if c.masternodeBySTS[node.StatefulSet] == nil || c.masternodeBySTS[node.StatefulSet].TotalSlots() == 0 {
					node.ZoneDistributionRank = node.ZoneDistributionRank * ExistingReplicasWeightMultiplier
					//If the repliaction offset is less than 10, then add more weight to this node
					if (node.MasterReplOffset - node.SlaveReplOffset) < 10 {
						node.ZoneDistributionRank = node.ZoneDistributionRank + 10
					}
				}
			}

		}
	}
}

func (c *RankingInfo) GetBestRankedMasters() (map[string]*redisutil.Node, error) {

	c.BestRankedMasterBySTS = make(map[string]*redisutil.Node)

	//First apply generic Zone Ranking Per node distribution
	c.ApplyZoneRankingPerNodeDistribution()

	for i := 0; i < c.desieredMasterCount; i++ {
		stsName := statefulsets.ClusterStatefulSetName(c.clusterName, i)

		masterNode, err := c.GetBestRankedNodeinAllZonesPerSTS(stsName)
		if err != nil || masterNode == nil {
			c.log.Error(err, "Error when trying to get master node with best rank ")
			return nil, fmt.Errorf("master rank error")
		}
		err1 := c.ZoneRankReducePostmasterSelect(masterNode.ID, masterNode.Zonename)
		if err1 != nil {
			c.log.Error(err, "Error when trying to get master node with best rank ")
			return nil, fmt.Errorf("master rank error")
		}
		c.log.Info("GetBestRankedMasters", "BestRankedMaster", masterNode.IP, "zone", masterNode.Zonename, "Statefulset", masterNode.StatefulSet, "PodName", masterNode.PodName, "Rank", masterNode.ZoneDistributionRank)
		c.BestRankedMasterBySTS[stsName] = masterNode
	}
	return c.BestRankedMasterBySTS, nil

}

func (c *RankingInfo) ZoneRankReducePostmasterSelect(masternodeID string, zoneName string) error {

	nodes, ok := c.nodesByZone[zoneName]
	if !ok {
		return fmt.Errorf("unable to find zone %s", zoneName)
	}
	for _, node := range nodes {
		if node.ID != masternodeID {
			//reduce the ranks of the nodes in the zone where the master is selected currently to avoid selecting the same zone for the next master
			node.ZoneDistributionRank = node.ZoneDistributionRank / MasterPerZoneWeightDivision
		}
	}
	return nil
}

// This function will return the best ranked node for the given STS across the zones
func (c *RankingInfo) GetBestRankedNodeinAllZonesPerSTS(stsname string) (*redisutil.Node, error) {

	var bestRankedNode *redisutil.Node = nil
	var rankedNodes = make(map[int][]*redisutil.Node)

	for _, zone := range c.zoneList {
		nodes, ok := c.nodesByZone[zone]
		if !ok {
			continue
		}
		checkRankNode := getBestRankedNodeinZonePerSTS(stsname, nodes, zone, c.log)
		if checkRankNode == nil {
			c.log.Info("GetZoneRankedNodePerSTS", "Statefulset", stsname, "Not found in Zone", zone)
			continue
		} else {
			rankedNodes[checkRankNode.ZoneDistributionRank] = append(rankedNodes[checkRankNode.ZoneDistributionRank], checkRankNode)
		}
	}

	if len(rankedNodes) == 0 {
		return nil, fmt.Errorf("no nodes found for the statefulset %s", stsname)
	}

	// Extract the Ranks from the rankedNodes map to get a slice of rank so that it can be sorted.
	ranks := make([]int, 0, len(rankedNodes))
	for rank := range rankedNodes {
		ranks = append(ranks, rank)
	}
	// Sort the slice of ranks in descending order.
	sort.Slice(ranks, func(i, j int) bool {
		return ranks[i] > ranks[j]
	})

	if len(rankedNodes[ranks[0]]) == 1 {
		bestRankedNode = rankedNodes[ranks[0]][0]
	} else {
		bestRankedNode = TieBreakGetTopRankedNode(rankedNodes[ranks[0]], c.zoneList, c.log)
	}

	return bestRankedNode, nil
}

func TieBreakGetTopRankedNode(tiebreaknodes []*redisutil.Node, zoneList []string, log logr.Logger) *redisutil.Node {
	var bestrankedNode *redisutil.Node = nil

	//If there are slaves in the tiebreaknodes, then select the slave with the least replication offset difference with the master
	for _, selectnode := range tiebreaknodes {
		if selectnode.GetRole() == redisv1alpha1.RedisClusterNodeRoleSlave && selectnode.LinkState == redisutil.RedisLinkStateConnected {
			if bestrankedNode == nil {
				bestrankedNode = selectnode
			} else if (selectnode.MasterReplOffset - selectnode.SlaveReplOffset) < (bestrankedNode.MasterReplOffset - bestrankedNode.SlaveReplOffset) {
				bestrankedNode = selectnode
			}
		}
	}

	//If there are no slaves in the tiebreakernodes , then possiblity all nodes are masters without slots, then select the one in the first zone
	if bestrankedNode == nil {
		for _, zone := range zoneList {
			for _, selectnode := range tiebreaknodes {
				if selectnode.Zonename == zone {
					bestrankedNode = selectnode
					break
				}
			}
			if bestrankedNode != nil {
				break
			}
		}
	}

	if bestrankedNode == nil {
		log.Error(fmt.Errorf("node not found"), "TieBreakGetTopRankedNode", "No node found in tiebreakernodes")
	} else {

		if bestrankedNode.GetRole() == redisv1alpha1.RedisClusterNodeRoleMaster && bestrankedNode.TotalSlots() == 0 {
			log.Info("TieBreakGetTopRankedNode", "Possibility of selecting master with no slots", bestrankedNode.IP, "Statefulset", bestrankedNode.StatefulSet, "pod", bestrankedNode.PodName, "Zone", bestrankedNode.Zonename, "Rank", bestrankedNode.ZoneDistributionRank)
		} else if bestrankedNode.GetRole() == redisv1alpha1.RedisClusterNodeRoleSlave {
			log.Info("TieBreakGetTopRankedNode", "Selected Slave with least replication offset difference", bestrankedNode.StatefulSet, "pod", bestrankedNode.PodName, "Zone", bestrankedNode.Zonename, "Rank", bestrankedNode.ZoneDistributionRank, "MasterReplOffset", bestrankedNode.MasterReplOffset, "SlaveReplOffset", bestrankedNode.SlaveReplOffset)
		}
	}

	return bestrankedNode
}

// This function will return the best ranked node for the given STS within the zone
func getBestRankedNodeinZonePerSTS(stsname string, nodes redisutil.Nodes, zonename string, log logr.Logger) *redisutil.Node {

	var stsNodes []*redisutil.Node = make([]*redisutil.Node, 0)
	var bestrankedNode *redisutil.Node = nil
	for _, node := range nodes {
		if node.StatefulSet == stsname {
			stsNodes = append(stsNodes, node)
		}
	}
	//If there is only one node found in the zone for the given STS, That is the best ranked node
	if len(stsNodes) == 1 {
		return stsNodes[0]
	}
	//if there are no nodes found in the zone for the given STS, then return nil
	if len(stsNodes) == 0 {
		log.Error(fmt.Errorf("node not found"), "getSTSNode", "Statefulset", stsname, "Not found in Zone", zonename)
		return nil
	}
	//If there are more than one nodes beloning to a STS found in a zone, then there is a possibility of Zone failure
	//and more nodes are in same zone, In such case We will select the best ranked node based on the ZoneDistributionRank,
	// this will work out as we have already given higher rank to replicas with no master but connected to gossip
	//If all nodes in the zone have same rank then first one will be chosen
	log.Info("getSTSNode", "Possible Zone failure, number of nodes", len(stsNodes), "in zone", zonename)
	for _, selectnode := range stsNodes {
		if bestrankedNode == nil {
			bestrankedNode = selectnode
		} else if selectnode.ZoneDistributionRank > bestrankedNode.ZoneDistributionRank {
			bestrankedNode = selectnode
		}
	}
	if bestrankedNode != nil && (bestrankedNode.GetRole() == redisv1alpha1.RedisClusterNodeRoleMaster && bestrankedNode.TotalSlots() == 0) {
		log.Info("getSTSNode", "Possibility of selecting master with no slots", bestrankedNode.ID)
	}
	return bestrankedNode
}

func sortRedisNodeByStatefulSet(nodes redisutil.Nodes) (map[string]redisutil.Nodes, error) {

	nodesByStatefulSet := make(map[string]redisutil.Nodes)

	for _, rNode := range nodes {
		ssName := unknownVMName
		if rNode.StatefulSet != "" {
			ssName = rNode.StatefulSet
		} else {
			return nil, fmt.Errorf("unable to find statefulset for node %s", rNode.ID)
		}
		if _, ok := nodesByStatefulSet[ssName]; !ok {
			nodesByStatefulSet[ssName] = redisutil.Nodes{}
		}
		nodesByStatefulSet[ssName] = append(nodesByStatefulSet[ssName], rNode)
	}

	return nodesByStatefulSet, nil
}
