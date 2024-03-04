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
				c.replicaNodesBySTS[rNode.MasterReferent] = append(c.replicaNodesBySTS[rNode.MasterReferent], rNode)
			} else if rNode.GetRole() == redisv1alpha1.RedisClusterNodeRoleSlave && rNode.MasterReferent == "" {
				c.log.Info("InitRankingInfo", "Slave without master", rNode.ID)
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
			// This is needed to rank this as a potential master on multi zone failures
			if (node.GetRole() == redisv1alpha1.RedisClusterNodeRoleSlave) && node.MasterReferent == "" && node.LinkState == redisutil.RedisLinkStateConnected {
				node.ZoneDistributionRank = node.ZoneDistributionRank * ExistingReplicasWeightMultiplier
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
	var tieBreakerRank int = 0
	var tieBreakNodes []*redisutil.Node = []*redisutil.Node{}

	for _, zone := range c.zoneList {
		nodes, ok := c.nodesByZone[zone]
		if !ok {
			continue
		}
		checkRankNode := getBestRankedNodeinZonePerSTS(stsname, nodes, zone, c.log)
		if checkRankNode == nil {
			c.log.Info("GetZoneRankedNodePerSTS", "Statefulset", stsname, "Not found in Zone", zone)
			continue
		}

		if bestRankedNode == nil {
			bestRankedNode = checkRankNode
			tieBreakerRank = 0
		} else if checkRankNode.ZoneDistributionRank > bestRankedNode.ZoneDistributionRank {
			bestRankedNode = checkRankNode
			tieBreakerRank = 0
		} else if checkRankNode.ZoneDistributionRank == bestRankedNode.ZoneDistributionRank {
			//If there is a tie, we will store it and break the tie later after checking all the zones
			tieBreakerRank = checkRankNode.ZoneDistributionRank
			tieBreakNodes = append(tieBreakNodes, bestRankedNode, checkRankNode)
		}
	}
	//If there is a tie, we will break the tie by adding weight to the node in lowest zone in ascending order for now
	//This can be enhanced later to consider other factors like node distribution, zone with lowest master etc
	if tieBreakerRank != 0 {
		for _, zone := range c.zoneList {
			for _, node := range tieBreakNodes {
				if node.Zonename == zone {
					node.ZoneDistributionRank += ZoneRankingTieBreakerWeightAdd
					return node, nil
				}
			}
		}
		for _, node := range tieBreakNodes {
			c.log.Error(fmt.Errorf("unable to break tie"), "GetBestRankedNodeinAllZonesPerSTS", "unable to break tie for nodes", node.PodName, "rank", node.ZoneDistributionRank, "zone", node.Zonename)
		}
		return nil, fmt.Errorf("unable to break the tie")
	}
	return bestRankedNode, nil
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
