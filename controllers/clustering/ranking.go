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
	NodesCountsPerZoneRankingWeight  = 100
	ZoneRankingTieBreakerWeightAdd   = 2
	MasterPerZoneWeightDivision      = 2
	ExistingReplicasWeightMultiplier = 2
	HaplacedMasterRankThreshold      = 50
)

type RankingInfo struct {
	log                      logr.Logger
	clusterName              string
	desieredMasterCount      int
	desieredReplicaCount     int
	zoneList                 []string
	masternodeBySTS          map[string]*redisutil.Node
	replicaNodesBySTS        map[string]redisutil.Nodes
	singleAzMasters          map[string]*redisutil.Node
	singleAzReplicas         map[string]redisutil.Nodes
	nodesByZone              map[string]redisutil.Nodes
	NodesIndexedbyZoneAndSTS map[string]map[string][]*redisutil.Node
	CurrentMastersbyZone     map[string]redisutil.Nodes
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
		log.Error(fmt.Errorf("error in clusterinfo"), "CreateRankingFromClusterInfo", "Info for Nodes not found ", "or less than expected")
		return nil
	}

	stsnodes, err = sortRedisNodeByStatefulSet(nodes, log)
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
		CurrentMastersbyZone:     make(map[string]redisutil.Nodes),
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

			//Update the master nodes per STS and zone
			if _, ok := c.CurrentMastersbyZone[rNode.Zonename]; !ok {
				c.CurrentMastersbyZone[rNode.Zonename] = redisutil.Nodes{}
			}
			if (rNode.GetRole() == redisv1alpha1.RedisClusterNodeRoleMaster) && rNode.TotalSlots() > 0 {

				c.masternodeBySTS[rNode.StatefulSet] = rNode
				c.CurrentMastersbyZone[rNode.Zonename] = append(c.CurrentMastersbyZone[rNode.Zonename], rNode)

			}
			//Update the replica nodes per STS
			if _, ok := c.replicaNodesBySTS[rNode.StatefulSet]; !ok {
				c.replicaNodesBySTS[rNode.StatefulSet] = redisutil.Nodes{}
			}

			if rNode.GetRole() == redisv1alpha1.RedisClusterNodeRoleSlave {
				c.replicaNodesBySTS[rNode.StatefulSet] = append(c.replicaNodesBySTS[rNode.StatefulSet], rNode)
				if rNode.MasterReferent == "" {
					c.log.Info("InitRankingInfo", "Slave without master", rNode.ID, "Statefulset", rNode.StatefulSet, "IP", rNode.IP)
				}
			}

		}
	}

}

func (c *RankingInfo) ApplyZoneRankingPerNodeDistribution() {
	for _, nodes := range c.nodesByZone {
		if len(nodes) == 0 {
			continue
		}
		//For all the nodes in the zone, we will add the weight based on the number of nodes in the zone
		for _, node := range nodes {
			node.ZoneDistributionRank = node.ZoneDistributionRank + (NodesCountsPerZoneRankingWeight / len(nodes))

		}

	}
}

func (c *RankingInfo) SetCurrentMastersZoneRank() {

	//Reset the Zone Distribution Rank for the masters if any set earlier
	for _, zone := range c.zoneList {
		for _, masterNode := range c.CurrentMastersbyZone[zone] {
			masterNode.ZoneDistributionRank = 0
		}
	}

	//First apply generic Zone Ranking Per node distribution
	c.ApplyZoneRankingPerNodeDistribution()

	//Iterate through the zones and set the weights for the masters based on the zone placement
	for _, zone := range c.zoneList {

		//If there are valid masters in this zone , then add weights to the masters based on its zone placement
		if len(c.CurrentMastersbyZone[zone]) > 0 {
			//If the master count in this zone is such that if this zone is lost , then rest of the masters will be able to reach quorum, then add more weight to the masters in this zone else reduce weight to the masters in this zone
			zoneMastersCount := len(c.CurrentMastersbyZone[zone])
			if (c.desieredMasterCount - zoneMastersCount) > (c.desieredMasterCount / 2) {
				for _, masterNode := range c.CurrentMastersbyZone[zone] {
					masterNode.ZoneDistributionRank = masterNode.ZoneDistributionRank + HaplacedMasterRankThreshold
				}
			} else {

				for _, masterNode := range c.CurrentMastersbyZone[zone] {
					if masterNode.ZoneDistributionRank > HaplacedMasterRankThreshold {
						masterNode.ZoneDistributionRank = masterNode.ZoneDistributionRank - HaplacedMasterRankThreshold
					} else {
						masterNode.ZoneDistributionRank = masterNode.ZoneDistributionRank / MasterPerZoneWeightDivision
					}
				}
			}

		}
	}
}

func (c *RankingInfo) GetHAPlacedNodeinSTS(stsname string) (*redisutil.Node, error) {

	var bestRankedNode *redisutil.Node = nil

	var leastMastersZone = make(map[int][]string)
	var leastNodesZone = make(map[int][]string)
	var masterCount int = 0
	var leastNodeCount int = 0

	masterCountSlice := make([]int, 0)
	//For a given Statefulset, find the zone with least number of masters
	//If there are more than one zone with least number of masters, then select the zone with least number of redis nodes
	for _, zone := range c.zoneList {
		if len(c.NodesIndexedbyZoneAndSTS[zone][stsname]) == 0 {
			continue
		}
		masterCount = len(c.CurrentMastersbyZone[zone])
		// Extract the counts from the leastMastersZone map to get a slice of counts so that it can be sorted.
		masterCountSlice = append(masterCountSlice, masterCount)
		leastMastersZone[masterCount] = append(leastMastersZone[masterCount], zone)
	}

	// Sort the slice of counts in ascending order.
	sort.SliceStable(masterCountSlice, func(i, j int) bool {
		return masterCountSlice[i] < masterCountSlice[j]
	})

	leastmasterCnt := masterCountSlice[0]

	if len(leastMastersZone[leastmasterCnt]) == 1 {
		bestRankedNode = c.NodesIndexedbyZoneAndSTS[leastMastersZone[leastmasterCnt][0]][stsname][0]
		return bestRankedNode, nil
	}
	//If there is a tie in the number of masters in the zone, then select the zone with least number of nodes
	ZonesList := leastMastersZone[leastmasterCnt]

	for _, zone := range ZonesList {
		if len(c.NodesIndexedbyZoneAndSTS[zone][stsname]) == 0 {
			continue
		}
		leastNodeCount = len(c.NodesIndexedbyZoneAndSTS[zone][stsname])
		leastNodesZone[leastNodeCount] = append(leastNodesZone[leastNodeCount], zone)
	}

	// Extract the counts from the leastMastersZone map to get a slice of counts so that it can be sorted.
	nodecounts := make([]int, 0, len(leastNodesZone))
	for nodecount := range leastNodesZone {
		nodecounts = append(nodecounts, nodecount)
	}
	// Sort the slice of counts in ascending order.
	sort.SliceStable(nodecounts, func(i, j int) bool {
		return nodecounts[i] < nodecounts[j]
	})
	leastNodeCount = nodecounts[0]

	if len(leastNodesZone[leastNodeCount]) == 1 {
		bestRankedNode = c.NodesIndexedbyZoneAndSTS[leastNodesZone[leastNodeCount][0]][stsname][0]
		return bestRankedNode, nil
	}

	//if still there is a tie , select a node from the first zone
	bestRankedNode = c.NodesIndexedbyZoneAndSTS[leastNodesZone[leastNodeCount][0]][stsname][0]
	return bestRankedNode, nil

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
// This function will also add weights to the nodes based on the cluster state and the node state
// This function will also break the tie if there are more than one node with same rank
// This function is expected to be called only when creating or recovering a shard of a cluster
func (c *RankingInfo) GetBestRankedNodeinAllZonesPerSTS(stsname string) (*redisutil.Node, error) {

	var bestRankedNode *redisutil.Node = nil
	var rankedNodes = make(map[int][]*redisutil.Node)

	//If there are replica(s) in the STS,and if those replica(s) donot have master
	// then select the best ranked node from the replicas
	if len(c.replicaNodesBySTS[stsname]) > 0 && (c.masternodeBySTS[stsname] == nil || c.masternodeBySTS[stsname].TotalSlots() == 0) {
		replicas := c.replicaNodesBySTS[stsname]
		c.AddWeightToBrokenClusterReplicas(replicas)
		for _, node := range replicas {
			rankedNodes[node.ZoneDistributionRank] = append(rankedNodes[node.ZoneDistributionRank], node)
		}
	} else {
		// For each zone, find the best-ranked node for the STS and group nodes by their ranks.
		for _, zone := range c.zoneList {
			if checkRankNode := c.getBestRankedNodeinZonePerSTS(stsname, zone); checkRankNode != nil {
				rankedNodes[checkRankNode.ZoneDistributionRank] = append(rankedNodes[checkRankNode.ZoneDistributionRank], checkRankNode)
			} else {
				c.log.Info("No node found in zone for STS", "Statefulset", stsname, "Zone", zone)
			}
		}
	}

	if len(rankedNodes) == 0 {
		return nil, fmt.Errorf("no nodes found for the statefulset %s", stsname)
	}

	bestRankedNode = GetTopRankedNodeorTieBreak(rankedNodes, c.zoneList, c.log)
	if bestRankedNode == nil {
		return nil, fmt.Errorf("failed to determine the best ranked node for statefulset %s", stsname)
	} else if bestRankedNode.GetRole() == redisv1alpha1.RedisClusterNodeRoleMaster && bestRankedNode.TotalSlots() == 0 {
		c.log.Info("GetBestRankedNodeinAllZonesPerSTS", "Master with no slots", bestRankedNode.ID, "Statefulset", bestRankedNode.StatefulSet, "IP", bestRankedNode.IP)
	} else if bestRankedNode.GetRole() == redisv1alpha1.RedisClusterNodeRoleSlave {
		c.log.Info("GetBestRankedNodeinAllZonesPerSTS", "Selected Slave with least replication offset difference", bestRankedNode.ID, "Statefulset", bestRankedNode.StatefulSet, "IP", bestRankedNode.IP, "MasterReplOffset", bestRankedNode.MasterReplOffset, "SlaveReplOffset", bestRankedNode.SlaveReplOffset)
	}

	return bestRankedNode, nil
}

func GetTopRankedNodeorTieBreak(rankedNodesMap map[int][]*redisutil.Node, zoneList []string, log logr.Logger) *redisutil.Node {
	var bestrankedNode *redisutil.Node = nil

	// Extract the Ranks from the rankedNodes map to get a slice of rank so that it can be sorted.
	ranks := make([]int, 0, len(rankedNodesMap))
	for rank := range rankedNodesMap {
		ranks = append(ranks, rank)
	}

	// Sort the slice of ranks in descending order.
	sort.Slice(ranks, func(i, j int) bool {
		return ranks[i] > ranks[j]
	})

	topRankedNodes := rankedNodesMap[ranks[0]]
	if len(topRankedNodes) == 1 {
		bestrankedNode = topRankedNodes[0]
		return bestrankedNode
	}

	//If there are slaves in the tiebreaknodes, then select the slave with the least replication offset difference with the master only if the shard needs to be recovered
	for _, selectnode := range topRankedNodes {
		if selectnode.GetRole() == redisv1alpha1.RedisClusterNodeRoleSlave && selectnode.LinkState == redisutil.RedisLinkStateConnected && selectnode.MasterLinkStatus != "up" {
			if bestrankedNode == nil {
				bestrankedNode = selectnode
			} else if selectnode.SlaveReplOffset > bestrankedNode.SlaveReplOffset {
				bestrankedNode = selectnode
			} else if selectnode.SlaveReplOffset == bestrankedNode.SlaveReplOffset {
				if (selectnode.MasterReplOffset - selectnode.SlaveReplOffset) < (bestrankedNode.MasterReplOffset - bestrankedNode.SlaveReplOffset) {
					bestrankedNode = selectnode
				}
			}
		}
	}

	//If there are no slaves in the tiebreakernodes , then possiblity all nodes are masters without slots, then select the one in the first zone
	if bestrankedNode == nil {
		for _, zone := range zoneList {
			for _, selectnode := range topRankedNodes {
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
		log.Error(fmt.Errorf("node not found"), "GetTopRankedNodeorTieBreak", "No node found in tiebreakernodes", "this is ideally not expected")
	}
	/*
		 else {

			if bestrankedNode.GetRole() == redisv1alpha1.RedisClusterNodeRoleMaster && bestrankedNode.TotalSlots() == 0 {
				log.Info("GetTopRankedNodeorTieBreak", "Possibility of selecting master with no slots", bestrankedNode.IP, "Statefulset", bestrankedNode.StatefulSet, "pod", bestrankedNode.PodName, "Zone", bestrankedNode.Zonename, "Rank", bestrankedNode.ZoneDistributionRank)
			} else if bestrankedNode.GetRole() == redisv1alpha1.RedisClusterNodeRoleSlave {
				log.Info("GetTopRankedNodeorTieBreak", "Selected Slave with least replication offset difference", bestrankedNode.StatefulSet, "pod", bestrankedNode.PodName, "Zone", bestrankedNode.Zonename, "Rank", bestrankedNode.ZoneDistributionRank, "MasterReplOffset", bestrankedNode.MasterReplOffset, "SlaveReplOffset", bestrankedNode.SlaveReplOffset)
			}
		}
	*/
	return bestrankedNode
}

// This function will return the best ranked node for the given STS within the zone
// This function will also add weights to the nodes based on the cluster state and the node state
func (c *RankingInfo) getBestRankedNodeinZonePerSTS(stsName string, zoneName string) *redisutil.Node {

	// Group nodes by their ZoneDistributionRank.
	rankedNodes := make(map[int][]*redisutil.Node)
	for _, nodes := range c.nodesByZone {
		for _, node := range nodes {
			if node.StatefulSet == stsName {
				rankedNodes[node.ZoneDistributionRank] = append(rankedNodes[node.ZoneDistributionRank], node)
			}
		}
	}

	if len(rankedNodes) == 0 {
		c.log.Info("No nodes found in zone for STS", "StatefulSet", stsName, "Zone", zoneName)
		return nil
	}

	//If there one node in the zone, then return that node
	//If there are more than one node in the zone, then select the best ranked node from the zone
	//If there are more than one node with same rank, then select the node with least replication offset difference with the master
	bestNode := GetTopRankedNodeorTieBreak(rankedNodes, c.zoneList, log)

	/*
		if bestNode != nil && bestNode.GetRole() == redisv1alpha1.RedisClusterNodeRoleMaster && bestNode.TotalSlots() == 0 {
			c.log.Info("Selected master node with no slots", "NodeID", bestNode.ID, "Zone", zoneName, "StatefulSet", stsName)
		}
	*/
	return bestNode
}

// If the node is a slave without a master, and gossip is connected, then we will add weight to it
// This is needed to rank this as a potential master on multi zone failures or when there are no masters for a shard and we need to promote a slave
func (c *RankingInfo) AddWeightToBrokenClusterReplicas(replicaNodes redisutil.Nodes) {

	for _, node := range replicaNodes {
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

func sortRedisNodeByStatefulSet(nodes redisutil.Nodes, log logr.Logger) (map[string]redisutil.Nodes, error) {

	nodesByStatefulSet := make(map[string]redisutil.Nodes)

	for _, rNode := range nodes {
		if len(rNode.FailStatus) > 0 {
			log.Info("sortRedisNodeByStatefulSet From InitHAranking", "Node has fail status", rNode.ID, "FailStatus", rNode.FailStatus)
			continue
		}
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

func (c *RankingInfo) IsClusterNeedsRecovery() (bool, []string) {

	var shardRecoverycount int = 0
	var activeMasters int = 0
	var stsList []string = []string{}

	for i := 0; i < c.desieredMasterCount; i++ {

		stsName := statefulsets.ClusterStatefulSetName(c.clusterName, i)
		if c.masternodeBySTS[stsName] != nil {
			c.log.Info("IsClusterNeedsRecovery", "Checking STS", stsName, "MasterNode IP", c.masternodeBySTS[stsName].IP, "MasterSlots", c.masternodeBySTS[stsName].TotalSlots())
		}

		if c.masternodeBySTS[stsName] != nil && c.masternodeBySTS[stsName].TotalSlots() > 0 {
			activeMasters = activeMasters + 1
			continue
		} else if c.masternodeBySTS[stsName] == nil && (c.replicaNodesBySTS[stsName] != nil && len(c.replicaNodesBySTS[stsName]) > 0) {
			shardRecoverycount = shardRecoverycount + 1
			stsList = append(stsList, stsName)
		}
	}
	if shardRecoverycount > 0 {

		if activeMasters < ((c.desieredMasterCount / 2) + 1) {
			minStsToRecover := ((c.desieredMasterCount / 2) + 1) - activeMasters
			minStsRecoverList := make([]string, minStsToRecover)
			copy(minStsRecoverList, stsList[:minStsToRecover])
			c.log.Info("IsClusterNeedsRecovery", "Cluster need recovery,Minimum required shard recovery count-", minStsToRecover, "ActiveMasters", activeMasters, "MinSTSList", minStsRecoverList)
			return true, minStsRecoverList
		} else {
			c.log.Info("IsClusterNeedsRecovery", "Quorum Available waiting for cluster to healitself", shardRecoverycount, "ActiveMasters", activeMasters, "STSList", stsList)
			return true, nil

		}
	}
	c.log.Info("IsClusterNeedsRecovery", "Cluster does not need recovery,shard recovery count-", shardRecoverycount, "ActiveMasters", activeMasters, "STSList", stsList)
	return false, nil
}
