package clustering

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	redisv1alpha1 "github.com/freshworks/redis-cluster-operator/api/v1alpha1"
	"github.com/freshworks/redis-cluster-operator/utils/redisutil"
)

func TestNewRankingInfo(t *testing.T) {
	tests := []struct {
		name                   string
		haconf                 *redisv1alpha1.HAspec
		nodes                  map[string]redisutil.Nodes
		expectedMasterCount    int32
		expectedReplicaCount   int32
		clusterName            string
		expectedZoneList       []string
		expectedDesiredMaster  int
		expectedDesiredReplica int
	}{
		{
			name: "Basic ranking info creation",
			haconf: &redisv1alpha1.HAspec{
				HaEnabled: true,
				ZonesInfo: map[string]string{
					"zone1": "zone1-info",
					"zone2": "zone2-info",
				},
			},
			nodes: map[string]redisutil.Nodes{
				"sts-0": {
					createTestNode("node1", "1.1.1.1", "zone1", "sts-0", redisv1alpha1.RedisClusterNodeRoleMaster),
					createTestNode("node2", "1.1.1.2", "zone2", "sts-0", redisv1alpha1.RedisClusterNodeRoleSlave),
				},
			},
			expectedMasterCount:    3,
			expectedReplicaCount:   2,
			clusterName:            "test-cluster",
			expectedZoneList:       []string{"zone1", "zone2"},
			expectedDesiredMaster:  3,
			expectedDesiredReplica: 2,
		},
		{
			name: "Empty zones info",
			haconf: &redisv1alpha1.HAspec{
				HaEnabled: true,
				ZonesInfo: map[string]string{},
			},
			nodes: map[string]redisutil.Nodes{
				"sts-0": {
					createTestNode("node1", "1.1.1.1", "zone1", "sts-0", redisv1alpha1.RedisClusterNodeRoleMaster),
				},
			},
			expectedMasterCount:    1,
			expectedReplicaCount:   1,
			clusterName:            "test-cluster",
			expectedZoneList:       []string{},
			expectedDesiredMaster:  1,
			expectedDesiredReplica: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logr.Discard()
			rankingInfo := NewRankingInfo(tt.haconf, tt.nodes, tt.expectedMasterCount, tt.expectedReplicaCount, tt.clusterName, log)

			require.NotNil(t, rankingInfo)
			assert.Equal(t, tt.clusterName, rankingInfo.clusterName)
			assert.Equal(t, tt.expectedDesiredMaster, rankingInfo.desieredMasterCount)
			assert.Equal(t, tt.expectedDesiredReplica, rankingInfo.desieredReplicaCount)
			assert.Equal(t, tt.expectedZoneList, rankingInfo.zoneList)
			assert.NotNil(t, rankingInfo.nodesByZone)
			assert.NotNil(t, rankingInfo.NodesIndexedbyZoneAndSTS)
			assert.NotNil(t, rankingInfo.CurrentMastersbyZone)
			assert.NotNil(t, rankingInfo.replicaNodesBySTS)
		})
	}
}

func TestCreateRankingFromClusterInfo(t *testing.T) {
	tests := []struct {
		name                 string
		haconf               *redisv1alpha1.HAspec
		clusterInfo          *redisutil.ClusterInfos
		expectedMasterCount  int32
		expectedReplicaCount int32
		clusterName          string
		shouldReturnNil      bool
	}{
		{
			name: "Successful creation from cluster info",
			haconf: &redisv1alpha1.HAspec{
				HaEnabled: true,
				ZonesInfo: map[string]string{
					"zone1": "zone1-info",
				},
			},
			clusterInfo: func() *redisutil.ClusterInfos {
				ci := redisutil.NewClusterInfos()
				ci.Infos["node1"] = &redisutil.NodeInfos{
					Node:    createTestNode("node1", "1.1.1.1", "zone1", "sts-0", redisv1alpha1.RedisClusterNodeRoleMaster),
					Friends: redisutil.Nodes{},
				}
				ci.Infos["node2"] = &redisutil.NodeInfos{
					Node:    createTestNode("node2", "1.1.1.2", "zone1", "sts-0", redisv1alpha1.RedisClusterNodeRoleSlave),
					Friends: redisutil.Nodes{},
				}
				return ci
			}(),
			expectedMasterCount:  1,
			expectedReplicaCount: 1,
			clusterName:          "test-cluster",
			shouldReturnNil:      false,
		},
		{
			name: "Insufficient nodes",
			haconf: &redisv1alpha1.HAspec{
				HaEnabled: true,
				ZonesInfo: map[string]string{
					"zone1": "zone1-info",
				},
			},
			clusterInfo: func() *redisutil.ClusterInfos {
				ci := redisutil.NewClusterInfos()
				ci.Infos["node1"] = &redisutil.NodeInfos{
					Node:    createTestNode("node1", "1.1.1.1", "zone1", "sts-0", redisv1alpha1.RedisClusterNodeRoleMaster),
					Friends: redisutil.Nodes{},
				}
				return ci
			}(),
			expectedMasterCount:  2,
			expectedReplicaCount: 2,
			clusterName:          "test-cluster",
			shouldReturnNil:      true,
		},
		{
			name: "Empty cluster info",
			haconf: &redisv1alpha1.HAspec{
				HaEnabled: true,
				ZonesInfo: map[string]string{
					"zone1": "zone1-info",
				},
			},
			clusterInfo:          redisutil.NewClusterInfos(), // Empty cluster info
			expectedMasterCount:  1,
			expectedReplicaCount: 1,
			clusterName:          "test-cluster",
			shouldReturnNil:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logr.Discard()
			rankingInfo := CreateRankingFromClusterInfo(tt.haconf, tt.clusterInfo, tt.expectedMasterCount, tt.expectedReplicaCount, tt.clusterName, log)

			if tt.shouldReturnNil {
				assert.Nil(t, rankingInfo)
			} else {
				require.NotNil(t, rankingInfo)
				assert.Equal(t, tt.clusterName, rankingInfo.clusterName)
				assert.Equal(t, int(tt.expectedMasterCount), rankingInfo.desieredMasterCount)
				assert.Equal(t, int(tt.expectedReplicaCount), rankingInfo.desieredReplicaCount)
			}
		})
	}
}

func TestRankingInfo_InitRankingInfo(t *testing.T) {
	tests := []struct {
		name     string
		nodes    map[string]redisutil.Nodes
		expected map[string]redisutil.Nodes
	}{
		{
			name: "Initialize ranking info with mixed nodes",
			nodes: func() map[string]redisutil.Nodes {
				node1 := createTestNode("node1", "1.1.1.1", "zone1", "sts-0", redisv1alpha1.RedisClusterNodeRoleMaster)
				node1.Slots = []redisutil.Slot{1, 2, 3} // Add slots to make it an active master
				node3 := createTestNode("node3", "1.1.1.3", "zone2", "sts-1", redisv1alpha1.RedisClusterNodeRoleMaster)
				node3.Slots = []redisutil.Slot{4, 5, 6} // Add slots to make it an active master

				return map[string]redisutil.Nodes{
					"sts-0": {
						node1,
						createTestNode("node2", "1.1.1.2", "zone1", "sts-0", redisv1alpha1.RedisClusterNodeRoleSlave),
					},
					"sts-1": {
						node3,
						createTestNode("node4", "1.1.1.4", "zone2", "sts-1", redisv1alpha1.RedisClusterNodeRoleSlave),
					},
				}
			}(),
			expected: map[string]redisutil.Nodes{
				"zone1": {
					createTestNode("node1", "1.1.1.1", "zone1", "sts-0", redisv1alpha1.RedisClusterNodeRoleMaster),
					createTestNode("node2", "1.1.1.2", "zone1", "sts-0", redisv1alpha1.RedisClusterNodeRoleSlave),
				},
				"zone2": {
					createTestNode("node3", "1.1.1.3", "zone2", "sts-1", redisv1alpha1.RedisClusterNodeRoleMaster),
					createTestNode("node4", "1.1.1.4", "zone2", "sts-1", redisv1alpha1.RedisClusterNodeRoleSlave),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logr.Discard()
			rankingInfo := &RankingInfo{
				log:                      log,
				clusterName:              "test-cluster",
				desieredMasterCount:      2,
				desieredReplicaCount:     2,
				zoneList:                 []string{"zone1", "zone2"},
				masternodeBySTS:          make(map[string]*redisutil.Node),
				replicaNodesBySTS:        make(map[string]redisutil.Nodes),
				nodesByZone:              make(map[string]redisutil.Nodes),
				NodesIndexedbyZoneAndSTS: make(map[string]map[string][]*redisutil.Node),
				CurrentMastersbyZone:     make(map[string]redisutil.Nodes),
				BestRankedMasterBySTS:    make(map[string]*redisutil.Node),
				haconfig:                 &redisv1alpha1.HAspec{},
				ProxyAz:                  "",
			}

			rankingInfo.InitRankingInfo(tt.nodes)

			// Verify nodes are properly categorized by zone
			for zone, expectedNodes := range tt.expected {
				actualNodes, exists := rankingInfo.nodesByZone[zone]
				assert.True(t, exists, "Zone %s should exist", zone)
				assert.Equal(t, len(expectedNodes), len(actualNodes), "Zone %s should have correct number of nodes", zone)
			}

			// Verify master nodes are properly tracked
			assert.Equal(t, 2, len(rankingInfo.masternodeBySTS), "Should have 2 master nodes by STS")
			assert.Equal(t, 2, len(rankingInfo.replicaNodesBySTS), "Should have 2 replica node groups by STS")
		})
	}
}

func TestRankingInfo_ApplyZoneRankingPerNodeDistribution(t *testing.T) {
	tests := []struct {
		name          string
		nodesByZone   map[string]redisutil.Nodes
		expectedRanks map[string]int
	}{
		{
			name: "Apply zone ranking with multiple nodes per zone",
			nodesByZone: map[string]redisutil.Nodes{
				"zone1": {
					createTestNode("node1", "1.1.1.1", "zone1", "sts-0", redisv1alpha1.RedisClusterNodeRoleMaster),
					createTestNode("node2", "1.1.1.2", "zone1", "sts-0", redisv1alpha1.RedisClusterNodeRoleSlave),
				},
				"zone2": {
					createTestNode("node3", "1.1.1.3", "zone2", "sts-1", redisv1alpha1.RedisClusterNodeRoleMaster),
				},
			},
			expectedRanks: map[string]int{
				"node1": NodesCountsPerZoneRankingWeight / 2, // 100 / 2 = 50
				"node2": NodesCountsPerZoneRankingWeight / 2, // 100 / 2 = 50
				"node3": NodesCountsPerZoneRankingWeight / 1, // 100 / 1 = 100
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logr.Discard()
			rankingInfo := &RankingInfo{
				log:                      log,
				clusterName:              "test-cluster",
				desieredMasterCount:      2,
				desieredReplicaCount:     2,
				zoneList:                 []string{"zone1", "zone2"},
				masternodeBySTS:          make(map[string]*redisutil.Node),
				replicaNodesBySTS:        make(map[string]redisutil.Nodes),
				nodesByZone:              tt.nodesByZone,
				NodesIndexedbyZoneAndSTS: make(map[string]map[string][]*redisutil.Node),
				CurrentMastersbyZone:     make(map[string]redisutil.Nodes),
				BestRankedMasterBySTS:    make(map[string]*redisutil.Node),
				haconfig:                 &redisv1alpha1.HAspec{},
				ProxyAz:                  "",
			}

			rankingInfo.ApplyZoneRankingPerNodeDistribution()

			// Verify that zone distribution ranks are applied correctly
			for zone, nodes := range tt.nodesByZone {
				for _, node := range nodes {
					expectedRank := tt.expectedRanks[node.ID]
					assert.Equal(t, expectedRank, node.ZoneDistributionRank,
						"Node %s in zone %s should have correct zone distribution rank", node.ID, zone)
				}
			}
		})
	}
}

func TestRankingInfo_SetCurrentMastersZoneRank(t *testing.T) {
	tests := []struct {
		name                    string
		nodesByZone             map[string]redisutil.Nodes
		currentMastersByZone    map[string]redisutil.Nodes
		desiredMasterCount      int
		expectedRankAdjustments map[string]int
	}{
		{
			name: "Set zone rank for masters with quorum available",
			nodesByZone: map[string]redisutil.Nodes{
				"zone1": {
					createTestNode("node1", "1.1.1.1", "zone1", "sts-0", redisv1alpha1.RedisClusterNodeRoleMaster),
					createTestNode("node2", "1.1.1.2", "zone1", "sts-0", redisv1alpha1.RedisClusterNodeRoleSlave),
				},
				"zone2": {
					createTestNode("node3", "1.1.1.3", "zone2", "sts-1", redisv1alpha1.RedisClusterNodeRoleMaster),
				},
			},
			currentMastersByZone: map[string]redisutil.Nodes{
				"zone1": {
					createTestNode("node1", "1.1.1.1", "zone1", "sts-0", redisv1alpha1.RedisClusterNodeRoleMaster),
				},
				"zone2": {
					createTestNode("node3", "1.1.1.3", "zone2", "sts-1", redisv1alpha1.RedisClusterNodeRoleMaster),
				},
			},
			desiredMasterCount: 3,
			expectedRankAdjustments: map[string]int{
				"node1": HaplacedMasterRankThreshold, // Should get HA threshold bonus
				"node3": HaplacedMasterRankThreshold, // Should get HA threshold bonus
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logr.Discard()
			rankingInfo := &RankingInfo{
				log:                      log,
				clusterName:              "test-cluster",
				desieredMasterCount:      tt.desiredMasterCount,
				desieredReplicaCount:     2,
				zoneList:                 []string{"zone1", "zone2"},
				masternodeBySTS:          make(map[string]*redisutil.Node),
				replicaNodesBySTS:        make(map[string]redisutil.Nodes),
				nodesByZone:              tt.nodesByZone,
				NodesIndexedbyZoneAndSTS: make(map[string]map[string][]*redisutil.Node),
				CurrentMastersbyZone:     tt.currentMastersByZone,
				BestRankedMasterBySTS:    make(map[string]*redisutil.Node),
				haconfig:                 &redisv1alpha1.HAspec{},
				ProxyAz:                  "",
			}

			// First apply basic zone ranking
			rankingInfo.ApplyZoneRankingPerNodeDistribution()

			// Store initial ranks
			initialRanks := make(map[string]int)
			for _, nodes := range tt.currentMastersByZone {
				for _, node := range nodes {
					initialRanks[node.ID] = node.ZoneDistributionRank
				}
			}

			rankingInfo.SetCurrentMastersZoneRank()

			// Verify that master nodes get appropriate rank adjustments
			for zone, nodes := range tt.currentMastersByZone {
				for _, node := range nodes {
					expectedAdjustment := tt.expectedRankAdjustments[node.ID]
					actualRank := node.ZoneDistributionRank
					initialRank := initialRanks[node.ID]

					assert.GreaterOrEqual(t, actualRank, initialRank+expectedAdjustment,
						"Master node %s in zone %s should have rank adjustment", node.ID, zone)
				}
			}
		})
	}
}

func TestRankingInfo_GetHAPlacedNodeinSTS(t *testing.T) {
	tests := []struct {
		name                    string
		stsName                 string
		proxyAz                 string
		nodesByZone             map[string]redisutil.Nodes
		replicaNodesBySTS       map[string]redisutil.Nodes
		currentMastersByZone    map[string]redisutil.Nodes
		nodesIndexedByZoneAndSTS map[string]map[string][]*redisutil.Node
		zoneList                []string
		expectedNodeID          string
		expectError             bool
	}{
		{
			name:    "Get HA placed node with proxy zone",
			stsName: "sts-0",
			proxyAz: "zone1",
			nodesByZone: map[string]redisutil.Nodes{
				"zone1": {
					createTestNode("node1", "1.1.1.1", "zone1", "sts-0", redisv1alpha1.RedisClusterNodeRoleSlave),
				},
				"zone2": {
					createTestNode("node2", "1.1.1.2", "zone2", "sts-0", redisv1alpha1.RedisClusterNodeRoleSlave),
				},
			},
			replicaNodesBySTS: map[string]redisutil.Nodes{
				"sts-0": {
					createTestNode("node1", "1.1.1.1", "zone1", "sts-0", redisv1alpha1.RedisClusterNodeRoleSlave),
					createTestNode("node2", "1.1.1.2", "zone2", "sts-0", redisv1alpha1.RedisClusterNodeRoleSlave),
				},
			},
			currentMastersByZone: map[string]redisutil.Nodes{
				"zone1": {},
				"zone2": {},
			},
			expectedNodeID: "node1",
			expectError:    false,
		},
		{
			name:    "No nodes in proxy zone",
			stsName: "sts-0",
			proxyAz: "zone3",
			nodesByZone: map[string]redisutil.Nodes{
				"zone1": {
					createTestNode("node1", "1.1.1.1", "zone1", "sts-0", redisv1alpha1.RedisClusterNodeRoleSlave),
				},
			},
			replicaNodesBySTS: map[string]redisutil.Nodes{
				"sts-0": {
					createTestNode("node1", "1.1.1.1", "zone1", "sts-0", redisv1alpha1.RedisClusterNodeRoleSlave),
				},
			},
			currentMastersByZone: map[string]redisutil.Nodes{
				"zone1": {},
			},
			expectedNodeID: "",
			expectError:    true,
		},
		{
			name:    "Empty STS with proxy zone",
			stsName: "sts-empty",
			proxyAz: "zone1",
			replicaNodesBySTS: map[string]redisutil.Nodes{
				"sts-empty": {}, // Empty STS
			},
			expectedNodeID: "",
			expectError:    true,
		},
		{
			name:    "Single zone with least masters - no proxy",
			stsName: "sts-0",
			proxyAz: "", // No proxy zone
			zoneList: []string{"zone1", "zone2", "zone3"},
			nodesIndexedByZoneAndSTS: map[string]map[string][]*redisutil.Node{
				"zone1": {
					"sts-0": {createTestNode("node1", "1.1.1.1", "zone1", "sts-0", redisv1alpha1.RedisClusterNodeRoleSlave)},
				},
				"zone2": {
					"sts-0": {createTestNode("node2", "1.1.1.2", "zone2", "sts-0", redisv1alpha1.RedisClusterNodeRoleSlave)},
				},
				"zone3": {
					"sts-0": {createTestNode("node3", "1.1.1.3", "zone3", "sts-0", redisv1alpha1.RedisClusterNodeRoleSlave)},
				},
			},
			currentMastersByZone: map[string]redisutil.Nodes{
				"zone1": {}, // 0 masters
				"zone2": {createTestNode("master1", "1.1.1.10", "zone2", "sts-1", redisv1alpha1.RedisClusterNodeRoleMaster)}, // 1 master
				"zone3": {createTestNode("master2", "1.1.1.11", "zone3", "sts-2", redisv1alpha1.RedisClusterNodeRoleMaster)}, // 1 master
			},
			expectedNodeID: "node1", // Should pick zone1 as it has least masters (0)
			expectError:    false,
		},
		{
			name:    "Tie in master count - resolve by node count",
			stsName: "sts-0",
			proxyAz: "", // No proxy zone
			zoneList: []string{"zone1", "zone2", "zone3"},
			nodesIndexedByZoneAndSTS: map[string]map[string][]*redisutil.Node{
				"zone1": {
					"sts-0": {createTestNode("node1", "1.1.1.1", "zone1", "sts-0", redisv1alpha1.RedisClusterNodeRoleSlave)}, // 1 node
				},
				"zone2": {
					"sts-0": {
						createTestNode("node2", "1.1.1.2", "zone2", "sts-0", redisv1alpha1.RedisClusterNodeRoleSlave),
						createTestNode("node3", "1.1.1.3", "zone2", "sts-0", redisv1alpha1.RedisClusterNodeRoleSlave),
					}, // 2 nodes
				},
				"zone3": {
					"sts-0": {}, // No nodes in this STS
				},
			},
			currentMastersByZone: map[string]redisutil.Nodes{
				"zone1": {}, // 0 masters (tie)
				"zone2": {}, // 0 masters (tie)  
				"zone3": {}, // 0 masters (tie)
			},
			expectedNodeID: "node1", // Should pick zone1 as it has fewer nodes (1 vs 2)
			expectError:    false,
		},
		{
			name:    "Final tie-breaking - pick first zone",
			stsName: "sts-0",
			proxyAz: "", // No proxy zone
			zoneList: []string{"zone1", "zone2"},
			nodesIndexedByZoneAndSTS: map[string]map[string][]*redisutil.Node{
				"zone1": {
					"sts-0": {createTestNode("node1", "1.1.1.1", "zone1", "sts-0", redisv1alpha1.RedisClusterNodeRoleSlave)}, // 1 node
				},
				"zone2": {
					"sts-0": {createTestNode("node2", "1.1.1.2", "zone2", "sts-0", redisv1alpha1.RedisClusterNodeRoleSlave)}, // 1 node
				},
			},
			currentMastersByZone: map[string]redisutil.Nodes{
				"zone1": {}, // 0 masters (tie)
				"zone2": {}, // 0 masters (tie)
			},
			expectedNodeID: "node1", // Should pick zone1 as it's first in tie-breaking
			expectError:    false,
		},
		{
			name:    "Complex scenario - multiple zones with different master counts",
			stsName: "sts-0",
			proxyAz: "", // No proxy zone
			zoneList: []string{"zone1", "zone2", "zone3", "zone4"},
			nodesIndexedByZoneAndSTS: map[string]map[string][]*redisutil.Node{
				"zone1": {
					"sts-0": {createTestNode("node1", "1.1.1.1", "zone1", "sts-0", redisv1alpha1.RedisClusterNodeRoleSlave)},
				},
				"zone2": {
					"sts-0": {createTestNode("node2", "1.1.1.2", "zone2", "sts-0", redisv1alpha1.RedisClusterNodeRoleSlave)},
				},
				"zone3": {
					"sts-0": {createTestNode("node3", "1.1.1.3", "zone3", "sts-0", redisv1alpha1.RedisClusterNodeRoleSlave)},
				},
				"zone4": {
					"sts-0": {}, // No nodes in this STS for zone4
				},
			},
			currentMastersByZone: map[string]redisutil.Nodes{
				"zone1": {
					createTestNode("master1", "1.1.1.10", "zone1", "sts-1", redisv1alpha1.RedisClusterNodeRoleMaster),
					createTestNode("master2", "1.1.1.11", "zone1", "sts-2", redisv1alpha1.RedisClusterNodeRoleMaster),
				}, // 2 masters
				"zone2": {
					createTestNode("master3", "1.1.1.12", "zone2", "sts-3", redisv1alpha1.RedisClusterNodeRoleMaster),
				}, // 1 master
				"zone3": {
					createTestNode("master4", "1.1.1.13", "zone3", "sts-4", redisv1alpha1.RedisClusterNodeRoleMaster),
				}, // 1 master
				"zone4": {}, // 0 masters (but no nodes in STS)
			},
			expectedNodeID: "node2", // Should pick zone2 or zone3 (both have 1 master, least among zones with nodes)
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logr.Discard()
			
			// Use provided zoneList or default
			zoneList := tt.zoneList
			if len(zoneList) == 0 {
				zoneList = []string{"zone1", "zone2"}
			}
			
			// Use provided NodesIndexedbyZoneAndSTS or default
			nodesIndexedByZoneAndSTS := tt.nodesIndexedByZoneAndSTS
			if nodesIndexedByZoneAndSTS == nil {
				nodesIndexedByZoneAndSTS = make(map[string]map[string][]*redisutil.Node)
			}
			
			rankingInfo := &RankingInfo{
				log:                      log,
				clusterName:              "test-cluster",
				desieredMasterCount:      2,
				desieredReplicaCount:     2,
				zoneList:                 zoneList,
				masternodeBySTS:          make(map[string]*redisutil.Node),
				replicaNodesBySTS:        tt.replicaNodesBySTS,
				nodesByZone:              tt.nodesByZone,
				NodesIndexedbyZoneAndSTS: nodesIndexedByZoneAndSTS,
				CurrentMastersbyZone:     tt.currentMastersByZone,
				BestRankedMasterBySTS:    make(map[string]*redisutil.Node),
				haconfig:                 &redisv1alpha1.HAspec{},
				ProxyAz:                  tt.proxyAz,
			}

			node, err := rankingInfo.GetHAPlacedNodeinSTS(tt.stsName)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, node)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, node)
				assert.Equal(t, tt.expectedNodeID, node.ID)
			}
		})
	}
}

func TestRankingInfo_ZoneRankReducePostmasterSelect(t *testing.T) {
	tests := []struct {
		name         string
		masterNodeID string
		zoneName     string
		nodesByZone  map[string]redisutil.Nodes
		expectError  bool
	}{
		{
			name:         "Reduce zone rank after master selection",
			masterNodeID: "node1",
			zoneName:     "zone1",
			nodesByZone: map[string]redisutil.Nodes{
				"zone1": {
					createTestNode("node1", "1.1.1.1", "zone1", "sts-0", redisv1alpha1.RedisClusterNodeRoleMaster),
					createTestNode("node2", "1.1.1.2", "zone1", "sts-0", redisv1alpha1.RedisClusterNodeRoleSlave),
				},
			},
			expectError: false,
		},
		{
			name:         "Zone not found",
			masterNodeID: "node1",
			zoneName:     "nonexistent-zone",
			nodesByZone: map[string]redisutil.Nodes{
				"zone1": {
					createTestNode("node1", "1.1.1.1", "zone1", "sts-0", redisv1alpha1.RedisClusterNodeRoleMaster),
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logr.Discard()
			rankingInfo := &RankingInfo{
				log:                      log,
				clusterName:              "test-cluster",
				desieredMasterCount:      2,
				desieredReplicaCount:     2,
				zoneList:                 []string{"zone1"},
				masternodeBySTS:          make(map[string]*redisutil.Node),
				replicaNodesBySTS:        make(map[string]redisutil.Nodes),
				nodesByZone:              tt.nodesByZone,
				NodesIndexedbyZoneAndSTS: make(map[string]map[string][]*redisutil.Node),
				CurrentMastersbyZone:     make(map[string]redisutil.Nodes),
				BestRankedMasterBySTS:    make(map[string]*redisutil.Node),
				haconfig:                 &redisv1alpha1.HAspec{},
				ProxyAz:                  "",
			}

			// Set initial ranks
			for _, nodes := range tt.nodesByZone {
				for _, node := range nodes {
					node.ZoneDistributionRank = 100
				}
			}

			err := rankingInfo.ZoneRankReducePostmasterSelect(tt.masterNodeID, tt.zoneName)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify that other nodes in the zone have reduced ranks
				if nodes, exists := tt.nodesByZone[tt.zoneName]; exists {
					for _, node := range nodes {
						if node.ID != tt.masterNodeID {
							assert.Equal(t, 50, node.ZoneDistributionRank, // 100 / MasterPerZoneWeightDivision
								"Node %s should have reduced zone distribution rank", node.ID)
						}
					}
				}
			}
		})
	}
}

func TestRankingInfo_GetBestRankedNodeinAllZonesPerSTS(t *testing.T) {
	tests := []struct {
		name              string
		stsName           string
		proxyAz           string
		nodesByZone       map[string]redisutil.Nodes
		replicaNodesBySTS map[string]redisutil.Nodes
		masterNodeBySTS   map[string]*redisutil.Node
		expectedNodeID    string
		expectError       bool
	}{
		{
			name:    "Get best ranked node with proxy zone",
			stsName: "sts-0",
			proxyAz: "zone1",
			nodesByZone: map[string]redisutil.Nodes{
				"zone1": {
					createTestNode("node1", "1.1.1.1", "zone1", "sts-0", redisv1alpha1.RedisClusterNodeRoleSlave),
				},
			},
			replicaNodesBySTS: map[string]redisutil.Nodes{
				"sts-0": {
					createTestNode("node1", "1.1.1.1", "zone1", "sts-0", redisv1alpha1.RedisClusterNodeRoleSlave),
				},
			},
			masterNodeBySTS: map[string]*redisutil.Node{},
			expectedNodeID:  "node1",
			expectError:     false,
		},
		{
			name:    "No nodes found for STS",
			stsName: "sts-nonexistent",
			proxyAz: "",
			nodesByZone: map[string]redisutil.Nodes{
				"zone1": {
					createTestNode("node1", "1.1.1.1", "zone1", "sts-0", redisv1alpha1.RedisClusterNodeRoleSlave),
				},
			},
			replicaNodesBySTS: map[string]redisutil.Nodes{},
			masterNodeBySTS:   map[string]*redisutil.Node{},
			expectedNodeID:    "",
			expectError:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logr.Discard()
			rankingInfo := &RankingInfo{
				log:                      log,
				clusterName:              "test-cluster",
				desieredMasterCount:      2,
				desieredReplicaCount:     2,
				zoneList:                 []string{"zone1"},
				masternodeBySTS:          tt.masterNodeBySTS,
				replicaNodesBySTS:        tt.replicaNodesBySTS,
				nodesByZone:              tt.nodesByZone,
				NodesIndexedbyZoneAndSTS: make(map[string]map[string][]*redisutil.Node),
				CurrentMastersbyZone:     make(map[string]redisutil.Nodes),
				BestRankedMasterBySTS:    make(map[string]*redisutil.Node),
				haconfig:                 &redisv1alpha1.HAspec{},
				ProxyAz:                  tt.proxyAz,
			}

			node, err := rankingInfo.GetBestRankedNodeinAllZonesPerSTS(tt.stsName)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, node)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, node)
				assert.Equal(t, tt.expectedNodeID, node.ID)
			}
		})
	}
}

func TestGetTopRankedNodeorTieBreak(t *testing.T) {
	tests := []struct {
		name           string
		rankedNodesMap map[int][]*redisutil.Node
		zoneList       []string
		expectedNodeID string
	}{
		{
			name: "Single top ranked node",
			rankedNodesMap: map[int][]*redisutil.Node{
				100: {createTestNode("node1", "1.1.1.1", "zone1", "sts-0", redisv1alpha1.RedisClusterNodeRoleMaster)},
				50:  {createTestNode("node2", "1.1.1.2", "zone2", "sts-1", redisv1alpha1.RedisClusterNodeRoleSlave)},
			},
			zoneList:       []string{"zone1", "zone2"},
			expectedNodeID: "node1",
		},
		{
			name: "Tie break with multiple nodes",
			rankedNodesMap: map[int][]*redisutil.Node{
				100: {
					createTestNode("node1", "1.1.1.1", "zone1", "sts-0", redisv1alpha1.RedisClusterNodeRoleMaster),
					createTestNode("node2", "1.1.1.2", "zone2", "sts-1", redisv1alpha1.RedisClusterNodeRoleMaster),
				},
			},
			zoneList:       []string{"zone1", "zone2"},
			expectedNodeID: "node1", // Should select first zone
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logr.Discard()
			node := GetTopRankedNodeorTieBreak(tt.rankedNodesMap, tt.zoneList, log)

			assert.NotNil(t, node)
			assert.Equal(t, tt.expectedNodeID, node.ID)
		})
	}
}

func TestRankingInfo_AddWeightToBrokenClusterReplicas(t *testing.T) {
	tests := []struct {
		name            string
		replicaNodes    redisutil.Nodes
		masterNodeBySTS map[string]*redisutil.Node
		expectedRanks   map[string]int
	}{
		{
			name: "Add weight to broken cluster replicas",
			replicaNodes: redisutil.Nodes{
				createTestNode("node1", "1.1.1.1", "zone1", "sts-0", redisv1alpha1.RedisClusterNodeRoleSlave),
				createTestNode("node2", "1.1.1.2", "zone2", "sts-1", redisv1alpha1.RedisClusterNodeRoleSlave),
			},
			masterNodeBySTS: map[string]*redisutil.Node{
				"sts-0": nil,                                                                                              // No master for sts-0
				"sts-1": createTestNode("master1", "1.1.1.3", "zone1", "sts-1", redisv1alpha1.RedisClusterNodeRoleMaster), // Master with no slots
			},
			expectedRanks: map[string]int{
				"node1": 200, // 100 * ExistingReplicasWeightMultiplier (no master for sts-0)
				"node2": 200, // 100 * ExistingReplicasWeightMultiplier (master exists for sts-1 but has no slots)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logr.Discard()
			rankingInfo := &RankingInfo{
				log:                      log,
				clusterName:              "test-cluster",
				desieredMasterCount:      2,
				desieredReplicaCount:     2,
				zoneList:                 []string{"zone1", "zone2"},
				masternodeBySTS:          tt.masterNodeBySTS,
				replicaNodesBySTS:        make(map[string]redisutil.Nodes),
				nodesByZone:              make(map[string]redisutil.Nodes),
				NodesIndexedbyZoneAndSTS: make(map[string]map[string][]*redisutil.Node),
				CurrentMastersbyZone:     make(map[string]redisutil.Nodes),
				BestRankedMasterBySTS:    make(map[string]*redisutil.Node),
				haconfig:                 &redisv1alpha1.HAspec{},
				ProxyAz:                  "",
			}

			// Set initial ranks and replication offsets
			for _, node := range tt.replicaNodes {
				node.ZoneDistributionRank = 100
				node.LinkState = redisutil.RedisLinkStateConnected
				node.MasterReplOffset = 1000
				node.SlaveReplOffset = 990 // Small difference to trigger the additional weight
			}

			rankingInfo.AddWeightToBrokenClusterReplicas(tt.replicaNodes)

			// Verify that replica nodes get weight multipliers
			for _, node := range tt.replicaNodes {
				expectedRank := tt.expectedRanks[node.ID]
				assert.Equal(t, expectedRank, node.ZoneDistributionRank,
					"Replica node %s should have correct weight multiplier", node.ID)
			}
		})
	}
}

func TestSortRedisNodeByStatefulSet(t *testing.T) {
	tests := []struct {
		name           string
		nodes          redisutil.Nodes
		expectedResult map[string]redisutil.Nodes
		expectError    bool
	}{
		{
			name: "Sort nodes by statefulset successfully",
			nodes: redisutil.Nodes{
				createTestNode("node1", "1.1.1.1", "zone1", "sts-0", redisv1alpha1.RedisClusterNodeRoleMaster),
				createTestNode("node2", "1.1.1.2", "zone1", "sts-0", redisv1alpha1.RedisClusterNodeRoleSlave),
				createTestNode("node3", "1.1.1.3", "zone2", "sts-1", redisv1alpha1.RedisClusterNodeRoleMaster),
			},
			expectedResult: map[string]redisutil.Nodes{
				"sts-0": {
					createTestNode("node1", "1.1.1.1", "zone1", "sts-0", redisv1alpha1.RedisClusterNodeRoleMaster),
					createTestNode("node2", "1.1.1.2", "zone1", "sts-0", redisv1alpha1.RedisClusterNodeRoleSlave),
				},
				"sts-1": {
					createTestNode("node3", "1.1.1.3", "zone2", "sts-1", redisv1alpha1.RedisClusterNodeRoleMaster),
				},
			},
			expectError: false,
		},
		{
			name: "Node without statefulset should error",
			nodes: redisutil.Nodes{
				createTestNode("node1", "1.1.1.1", "zone1", "", redisv1alpha1.RedisClusterNodeRoleMaster),
			},
			expectedResult: nil,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logr.Discard()
			result, err := sortRedisNodeByStatefulSet(tt.nodes, log)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, len(tt.expectedResult), len(result))

				for stsName, expectedNodes := range tt.expectedResult {
					actualNodes, exists := result[stsName]
					assert.True(t, exists, "StatefulSet %s should exist", stsName)
					assert.Equal(t, len(expectedNodes), len(actualNodes),
						"StatefulSet %s should have correct number of nodes", stsName)
				}
			}
		})
	}
}

func TestRankingInfo_IsClusterNeedsRecovery(t *testing.T) {
	tests := []struct {
		name                  string
		desiredMasterCount    int
		masterNodeBySTS       map[string]*redisutil.Node
		replicaNodesBySTS     map[string]redisutil.Nodes
		expectedNeedsRecovery bool
		expectedSTSList       []string
	}{
		{
			name:               "Cluster needs recovery - no active masters",
			desiredMasterCount: 3,
			masterNodeBySTS: map[string]*redisutil.Node{
				"drc-test-cluster-shard0": nil,
				"drc-test-cluster-shard1": nil,
				"drc-test-cluster-shard2": nil,
			},
			replicaNodesBySTS: map[string]redisutil.Nodes{
				"drc-test-cluster-shard0": {
					createTestNode("node1", "1.1.1.1", "zone1", "drc-test-cluster-shard0", redisv1alpha1.RedisClusterNodeRoleSlave),
				},
				"drc-test-cluster-shard1": {
					createTestNode("node2", "1.1.1.2", "zone2", "drc-test-cluster-shard1", redisv1alpha1.RedisClusterNodeRoleSlave),
				},
			},
			expectedNeedsRecovery: true,
			expectedSTSList:       []string{"drc-test-cluster-shard0", "drc-test-cluster-shard1"},
		},
		{
			name:               "Cluster healthy - has active masters",
			desiredMasterCount: 3,
			masterNodeBySTS: map[string]*redisutil.Node{
				"drc-test-cluster-shard0": createTestNode("node1", "1.1.1.1", "zone1", "drc-test-cluster-shard0", redisv1alpha1.RedisClusterNodeRoleMaster),
				"drc-test-cluster-shard1": createTestNode("node2", "1.1.1.2", "zone2", "drc-test-cluster-shard1", redisv1alpha1.RedisClusterNodeRoleMaster),
				"drc-test-cluster-shard2": createTestNode("node3", "1.1.1.3", "zone3", "drc-test-cluster-shard2", redisv1alpha1.RedisClusterNodeRoleMaster),
			},
			replicaNodesBySTS:     map[string]redisutil.Nodes{},
			expectedNeedsRecovery: false,
			expectedSTSList:       nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logr.Discard()
			rankingInfo := &RankingInfo{
				log:                      log,
				clusterName:              "test-cluster",
				desieredMasterCount:      tt.desiredMasterCount,
				desieredReplicaCount:     2,
				zoneList:                 []string{"zone1", "zone2", "zone3"},
				masternodeBySTS:          tt.masterNodeBySTS,
				replicaNodesBySTS:        tt.replicaNodesBySTS,
				nodesByZone:              make(map[string]redisutil.Nodes),
				NodesIndexedbyZoneAndSTS: make(map[string]map[string][]*redisutil.Node),
				CurrentMastersbyZone:     make(map[string]redisutil.Nodes),
				BestRankedMasterBySTS:    make(map[string]*redisutil.Node),
				haconfig:                 &redisv1alpha1.HAspec{},
				ProxyAz:                  "",
			}

			// Set slots for master nodes to make them active
			for _, masterNode := range tt.masterNodeBySTS {
				if masterNode != nil {
					masterNode.Slots = []redisutil.Slot{1, 2, 3} // Add some slots
				}
			}

			needsRecovery, stsList := rankingInfo.IsClusterNeedsRecovery()

			assert.Equal(t, tt.expectedNeedsRecovery, needsRecovery)
			if tt.expectedSTSList != nil {
				assert.Equal(t, len(tt.expectedSTSList), len(stsList))
			}
		})
	}
}

// Helper function to create test nodes
func createTestNode(id, ip, zone, statefulSet string, role redisv1alpha1.RedisRole) *redisutil.Node {
	node := &redisutil.Node{
		ID:   id,
		IP:   ip,
		Port: "6379",
		Role: func() string {
			switch role {
			case redisv1alpha1.RedisClusterNodeRoleMaster:
				return redisutil.RedisMasterRole
			case redisv1alpha1.RedisClusterNodeRoleSlave:
				return redisutil.RedisSlaveRole
			default:
				return ""
			}
		}(),
		LinkState:        redisutil.RedisLinkStateConnected,
		MasterReferent:   "",
		LastIOSecs:       0,
		SlaveReplOffset:  0,
		MasterLinkStatus: "up",
		MasterReplOffset: 0,
		FailStatus:       []string{},
		PingSent:         0,
		PongRecv:         0,
		ConfigEpoch:      0,
		Slots:            []redisutil.Slot{},
		// balance field is unexported, so we can't set it directly
		MigratingSlots:        map[redisutil.Slot]string{},
		ImportingSlots:        map[redisutil.Slot]string{},
		NodeName:              "test-node",
		ZoneDistributionRank:  0,
		NodeDistributionRank:  0,
		ShardDistributionRank: 0,
		Zonename:              zone,
		PodName:               id + "-pod",
		StatefulSet:           statefulSet,
	}
	return node
}
