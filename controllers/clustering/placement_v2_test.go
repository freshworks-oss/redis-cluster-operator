package clustering

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	redisv1alpha1 "github.com/dinesh-murugiah/rediscluster-operator/api/v1alpha1"
	"github.com/dinesh-murugiah/rediscluster-operator/utils/redisutil"
)

func TestNewCtx(t *testing.T) {
	tests := []struct {
		name            string
		haconf          *redisv1alpha1.HAspec
		cluster         *redisutil.Cluster
		nodes           redisutil.Nodes
		masterNum       int32
		replicaCount    int32
		clusterName     string
		expectedMaster  int
		expectedReplica int
	}{
		{
			name: "Create context with HA enabled",
			haconf: &redisv1alpha1.HAspec{
				HaEnabled: true,
				ZonesInfo: map[string]string{
					"zone1": "zone1-info",
					"zone2": "zone2-info",
				},
			},
			cluster: &redisutil.Cluster{
				HaStatus: redisv1alpha1.HaStatusCreating,
			},
			nodes: redisutil.Nodes{
				createTestNode("node1", "1.1.1.1", "zone1", "drc-test-cluster-shard0", redisv1alpha1.RedisClusterNodeRoleMaster),
				createTestNode("node2", "1.1.1.2", "zone2", "drc-test-cluster-shard0", redisv1alpha1.RedisClusterNodeRoleSlave),
			},
			masterNum:       3,
			replicaCount:    2,
			clusterName:     "test-cluster",
			expectedMaster:  3,
			expectedReplica: 2,
		},
		{
			name: "Create context with HA disabled",
			haconf: &redisv1alpha1.HAspec{
				HaEnabled: false,
				ZonesInfo: map[string]string{},
			},
			cluster: &redisutil.Cluster{
				HaStatus: redisv1alpha1.HaStatusDisabled,
			},
			nodes: redisutil.Nodes{
				createTestNode("node1", "1.1.1.1", "zone1", "drc-test-cluster-shard0", redisv1alpha1.RedisClusterNodeRoleMaster),
			},
			masterNum:       1,
			replicaCount:    1,
			clusterName:     "test-cluster",
			expectedMaster:  1,
			expectedReplica: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logr.Discard()
			ctx := NewCtx(tt.haconf, tt.cluster, tt.nodes, tt.masterNum, tt.replicaCount, tt.clusterName, log)

			require.NotNil(t, ctx)
			assert.Equal(t, tt.expectedMaster, ctx.expectedMasterNum)
			assert.Equal(t, tt.expectedReplica, ctx.expectedReplicaCount)
			assert.Equal(t, tt.clusterName, ctx.clusterName)
			assert.Equal(t, tt.haconf, ctx.haconfig)
			assert.Equal(t, tt.cluster.HaStatus, ctx.hastatus)
			assert.NotNil(t, ctx.nodes)
			assert.NotNil(t, ctx.slavesByMaster)
			assert.NotNil(t, ctx.newMastersBySts)
		})
	}
}

func TestCtx_SetProxyAz(t *testing.T) {
	tests := []struct {
		name       string
		proxyPodAz string
	}{
		{
			name:       "Set proxy AZ",
			proxyPodAz: "zone1",
		},
		{
			name:       "Set empty proxy AZ",
			proxyPodAz: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &Ctx{
				proxyPodAz: "",
			}

			ctx.SetProxyAz(tt.proxyPodAz)

			assert.Equal(t, tt.proxyPodAz, ctx.proxyPodAz)
		})
	}
}

func TestCtx_PlaceMasters(t *testing.T) {
	tests := []struct {
		name             string
		ssName           string
		currentMasters   redisutil.Nodes
		newMastersBySts  map[string]*redisutil.Node
		nodes            map[string]redisutil.Nodes
		expectedNodeID   string
		expectBestEffort bool
	}{
		{
			name:   "Place master with available nodes - no conflicts",
			ssName: "drc-test-cluster-shard0",
			currentMasters: redisutil.Nodes{
				createTestNodeWithNodeName("master1", "1.1.1.10", "zone1", "drc-test-cluster-shard1", redisv1alpha1.RedisClusterNodeRoleMaster, "node-master1"),
			},
			newMastersBySts: map[string]*redisutil.Node{
				"drc-test-cluster-shard1": createTestNodeWithNodeName("master1", "1.1.1.10", "zone1", "drc-test-cluster-shard1", redisv1alpha1.RedisClusterNodeRoleMaster, "node-master1"),
			},
			nodes: map[string]redisutil.Nodes{
				"drc-test-cluster-shard0": {
					createTestNodeWithNodeName("node1", "1.1.1.1", "zone1", "drc-test-cluster-shard0", redisv1alpha1.RedisClusterNodeRoleMaster, "node-node1"),
					createTestNodeWithNodeName("node2", "1.1.1.2", "zone2", "drc-test-cluster-shard0", redisv1alpha1.RedisClusterNodeRoleSlave, "node-node2"),
				},
			},
			expectedNodeID:   "node1",
			expectBestEffort: false,
		},
		{
			name:   "Place master with best effort - all nodes conflict",
			ssName: "drc-test-cluster-shard0",
			currentMasters: redisutil.Nodes{
				createTestNodeWithNodeName("master1", "1.1.1.1", "zone1", "drc-test-cluster-shard1", redisv1alpha1.RedisClusterNodeRoleMaster, "node-conflict"),
			},
			newMastersBySts: map[string]*redisutil.Node{
				"drc-test-cluster-shard1": createTestNodeWithNodeName("master1", "1.1.1.1", "zone1", "drc-test-cluster-shard1", redisv1alpha1.RedisClusterNodeRoleMaster, "node-conflict"),
			},
			nodes: map[string]redisutil.Nodes{
				"drc-test-cluster-shard0": {
					createTestNodeWithNodeName("node1", "1.1.1.1", "zone1", "drc-test-cluster-shard0", redisv1alpha1.RedisClusterNodeRoleMaster, "node-conflict"),
				},
			},
			expectedNodeID:   "node1",
			expectBestEffort: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logr.Discard()
			ctx := &Ctx{
				log:             log,
				currentMasters:  tt.currentMasters,
				newMastersBySts: tt.newMastersBySts,
				nodes:           tt.nodes,
				bestEffort:      false,
			}

			result := ctx.PlaceMasters(tt.ssName)

			assert.NotNil(t, result)
			assert.Equal(t, tt.expectedNodeID, result.ID)
			assert.Equal(t, tt.expectBestEffort, ctx.bestEffort)
		})
	}
}

func TestCtx_PlaceMasterSingleAz(t *testing.T) {
	tests := []struct {
		name           string
		ssName         string
		proxyPodAz     string
		nodes          map[string]redisutil.Nodes
		expectedNodeID string
		expectError    bool
	}{
		{
			name:       "Place master in specific AZ",
			ssName:     "drc-test-cluster-shard0",
			proxyPodAz: "zone1",
			nodes: map[string]redisutil.Nodes{
				"drc-test-cluster-shard0": {
					createTestNode("node1", "1.1.1.1", "zone1", "drc-test-cluster-shard0", redisv1alpha1.RedisClusterNodeRoleMaster),
					createTestNode("node2", "1.1.1.2", "zone2", "drc-test-cluster-shard0", redisv1alpha1.RedisClusterNodeRoleSlave),
				},
			},
			expectedNodeID: "node1",
			expectError:    false,
		},
		{
			name:       "No nodes in specified AZ",
			ssName:     "drc-test-cluster-shard0",
			proxyPodAz: "zone3",
			nodes: map[string]redisutil.Nodes{
				"drc-test-cluster-shard0": {
					createTestNode("node1", "1.1.1.1", "zone1", "drc-test-cluster-shard0", redisv1alpha1.RedisClusterNodeRoleMaster),
					createTestNode("node2", "1.1.1.2", "zone2", "drc-test-cluster-shard0", redisv1alpha1.RedisClusterNodeRoleSlave),
				},
			},
			expectedNodeID: "",
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logr.Discard()
			ctx := &Ctx{
				log:        log,
				proxyPodAz: tt.proxyPodAz,
				nodes:      tt.nodes,
			}

			result, err := ctx.PlaceMasterSingleAz(tt.ssName)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectedNodeID, result.ID)
			}
		})
	}
}

func TestCtx_PlaceSlavesV2(t *testing.T) {
	tests := []struct {
		name                   string
		nodes                  map[string]redisutil.Nodes
		newMastersBySts        map[string]*redisutil.Node
		expectedSlavesByMaster map[string]int
	}{
		{
			name: "Place slaves with mixed roles",
			nodes: map[string]redisutil.Nodes{
				"drc-test-cluster-shard0": {
					createTestNode("master1", "1.1.1.1", "zone1", "drc-test-cluster-shard0", redisv1alpha1.RedisClusterNodeRoleMaster),
					createTestNode("slave1", "1.1.1.2", "zone2", "drc-test-cluster-shard0", redisv1alpha1.RedisClusterNodeRoleSlave),
					createTestNode("slave2", "1.1.1.3", "zone1", "drc-test-cluster-shard0", redisv1alpha1.RedisClusterNodeRoleSlave),
				},
			},
			newMastersBySts: map[string]*redisutil.Node{
				"drc-test-cluster-shard0": createTestNode("master1", "1.1.1.1", "zone1", "drc-test-cluster-shard0", redisv1alpha1.RedisClusterNodeRoleMaster),
			},
			expectedSlavesByMaster: map[string]int{
				"master1": 2, // slave1 and slave2
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logr.Discard()
			ctx := &Ctx{
				log:             log,
				nodes:           tt.nodes,
				newMastersBySts: tt.newMastersBySts,
				slavesByMaster:  make(map[string]redisutil.Nodes),
				bestEffort:      false,
			}

			err := ctx.PlaceSlaves()

			assert.NoError(t, err)

			// Verify slaves are properly assigned
			for masterID, expectedCount := range tt.expectedSlavesByMaster {
				slaves, exists := ctx.slavesByMaster[masterID]
				assert.True(t, exists, "Master %s should have slaves", masterID)
				assert.Equal(t, expectedCount, len(slaves), "Master %s should have correct number of slaves", masterID)
			}
		})
	}
}

func TestCtx_GetCurrentMasters(t *testing.T) {
	tests := []struct {
		name           string
		currentMasters redisutil.Nodes
		expectedCount  int
	}{
		{
			name: "Get current masters",
			currentMasters: redisutil.Nodes{
				createTestNode("master1", "1.1.1.1", "zone1", "drc-test-cluster-shard0", redisv1alpha1.RedisClusterNodeRoleMaster),
				createTestNode("master2", "1.1.1.2", "zone2", "drc-test-cluster-shard1", redisv1alpha1.RedisClusterNodeRoleMaster),
			},
			expectedCount: 2,
		},
		{
			name:           "Get empty current masters",
			currentMasters: redisutil.Nodes{},
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &Ctx{
				currentMasters: tt.currentMasters,
			}

			result := ctx.GetCurrentMasters()

			assert.Equal(t, tt.expectedCount, len(result))
			assert.Equal(t, tt.currentMasters, result)
		})
	}
}

func TestCtx_GetNewMasters(t *testing.T) {
	tests := []struct {
		name            string
		newMastersBySts map[string]*redisutil.Node
		expectedCount   int
	}{
		{
			name: "Get new masters",
			newMastersBySts: map[string]*redisutil.Node{
				"drc-test-cluster-shard0": createTestNode("master1", "1.1.1.1", "zone1", "drc-test-cluster-shard0", redisv1alpha1.RedisClusterNodeRoleMaster),
				"drc-test-cluster-shard1": createTestNode("master2", "1.1.1.2", "zone2", "drc-test-cluster-shard1", redisv1alpha1.RedisClusterNodeRoleMaster),
			},
			expectedCount: 2,
		},
		{
			name:            "Get empty new masters",
			newMastersBySts: map[string]*redisutil.Node{},
			expectedCount:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &Ctx{
				newMastersBySts: tt.newMastersBySts,
			}

			result := ctx.GetNewMasters()

			assert.Equal(t, tt.expectedCount, len(result))
		})
	}
}

func TestCtx_GetSlaves(t *testing.T) {
	tests := []struct {
		name           string
		slavesByMaster map[string]redisutil.Nodes
		expectedCount  int
	}{
		{
			name: "Get slaves by master",
			slavesByMaster: map[string]redisutil.Nodes{
				"master1": {
					createTestNode("slave1", "1.1.1.2", "zone2", "drc-test-cluster-shard0", redisv1alpha1.RedisClusterNodeRoleSlave),
					createTestNode("slave2", "1.1.1.3", "zone1", "drc-test-cluster-shard0", redisv1alpha1.RedisClusterNodeRoleSlave),
				},
				"master2": {
					createTestNode("slave3", "1.1.1.4", "zone2", "drc-test-cluster-shard1", redisv1alpha1.RedisClusterNodeRoleSlave),
				},
			},
			expectedCount: 2, // 2 masters
		},
		{
			name:           "Get empty slaves",
			slavesByMaster: map[string]redisutil.Nodes{},
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &Ctx{
				slavesByMaster: tt.slavesByMaster,
			}

			result := ctx.GetSlaves()

			assert.Equal(t, tt.expectedCount, len(result))
			assert.Equal(t, tt.slavesByMaster, result)
		})
	}
}

func TestCtx_GetAllSlaves(t *testing.T) {
	tests := []struct {
		name          string
		nodes         map[string]redisutil.Nodes
		expectedCount int
	}{
		{
			name: "Get all slaves from nodes",
			nodes: map[string]redisutil.Nodes{
				"drc-test-cluster-shard0": {
					createTestNode("master1", "1.1.1.1", "zone1", "drc-test-cluster-shard0", redisv1alpha1.RedisClusterNodeRoleMaster),
					createTestNode("slave1", "1.1.1.2", "zone2", "drc-test-cluster-shard0", redisv1alpha1.RedisClusterNodeRoleSlave),
					createTestNode("slave2", "1.1.1.3", "zone1", "drc-test-cluster-shard0", redisv1alpha1.RedisClusterNodeRoleSlave),
				},
				"drc-test-cluster-shard1": {
					createTestNode("master2", "1.1.1.4", "zone1", "drc-test-cluster-shard1", redisv1alpha1.RedisClusterNodeRoleMaster),
					createTestNode("slave3", "1.1.1.5", "zone2", "drc-test-cluster-shard1", redisv1alpha1.RedisClusterNodeRoleSlave),
				},
			},
			expectedCount: 3, // slave1, slave2, slave3
		},
		{
			name:          "Get slaves from empty nodes",
			nodes:         map[string]redisutil.Nodes{},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &Ctx{
				nodes: tt.nodes,
			}

			result := ctx.GetAllSlaves()

			assert.Equal(t, tt.expectedCount, len(result))
		})
	}
}

func TestCtx_GetStatefulsetNodes(t *testing.T) {
	tests := []struct {
		name          string
		nodes         map[string]redisutil.Nodes
		expectedCount int
	}{
		{
			name: "Get statefulset nodes",
			nodes: map[string]redisutil.Nodes{
				"drc-test-cluster-shard0": {
					createTestNode("node1", "1.1.1.1", "zone1", "drc-test-cluster-shard0", redisv1alpha1.RedisClusterNodeRoleMaster),
					createTestNode("node2", "1.1.1.2", "zone2", "drc-test-cluster-shard0", redisv1alpha1.RedisClusterNodeRoleSlave),
				},
				"drc-test-cluster-shard1": {
					createTestNode("node3", "1.1.1.3", "zone1", "drc-test-cluster-shard1", redisv1alpha1.RedisClusterNodeRoleMaster),
				},
			},
			expectedCount: 2, // 2 StatefulSets
		},
		{
			name:          "Get empty statefulset nodes",
			nodes:         map[string]redisutil.Nodes{},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &Ctx{
				nodes: tt.nodes,
			}

			result := ctx.GetStatefulsetNodes()

			assert.Equal(t, tt.expectedCount, len(result))
			assert.Equal(t, tt.nodes, result)
		})
	}
}

func TestCtx_GetNodestoScaledandRecover(t *testing.T) {
	tests := []struct {
		name            string
		newMasters      redisutil.Nodes
		curMasters      redisutil.Nodes
		nodes           map[string]redisutil.Nodes
		expectedScaleUp int
		expectedRecover int
	}{
		{
			name: "Scale up new shard",
			newMasters: redisutil.Nodes{
				createTestNode("new-master1", "1.1.1.10", "zone1", "drc-test-cluster-shard2", redisv1alpha1.RedisClusterNodeRoleMaster),
			},
			curMasters: redisutil.Nodes{
				createTestNode("cur-master1", "1.1.1.1", "zone1", "drc-test-cluster-shard0", redisv1alpha1.RedisClusterNodeRoleMaster),
				createTestNode("cur-master2", "1.1.1.2", "zone2", "drc-test-cluster-shard1", redisv1alpha1.RedisClusterNodeRoleMaster),
			},
			nodes: map[string]redisutil.Nodes{
				"drc-test-cluster-shard2": {
					createTestNode("new-master1", "1.1.1.10", "zone1", "drc-test-cluster-shard2", redisv1alpha1.RedisClusterNodeRoleMaster),
				},
			},
			expectedScaleUp: 1,
			expectedRecover: 0,
		},
		{
			name: "Recover existing shard with slaves",
			newMasters: redisutil.Nodes{
				createTestNode("new-master1", "1.1.1.10", "zone1", "drc-test-cluster-shard0", redisv1alpha1.RedisClusterNodeRoleMaster),
			},
			curMasters: redisutil.Nodes{
				createTestNode("cur-master2", "1.1.1.2", "zone2", "drc-test-cluster-shard1", redisv1alpha1.RedisClusterNodeRoleMaster),
			},
			nodes: map[string]redisutil.Nodes{
				"drc-test-cluster-shard0": {
					createTestNode("new-master1", "1.1.1.10", "zone1", "drc-test-cluster-shard0", redisv1alpha1.RedisClusterNodeRoleMaster),
					createTestNode("slave1", "1.1.1.11", "zone2", "drc-test-cluster-shard0", redisv1alpha1.RedisClusterNodeRoleSlave),
				},
			},
			expectedScaleUp: 0,
			expectedRecover: 1,
		},
		{
			name: "Master already exists in current masters",
			newMasters: redisutil.Nodes{
				createTestNode("existing-master", "1.1.1.1", "zone1", "drc-test-cluster-shard0", redisv1alpha1.RedisClusterNodeRoleMaster),
			},
			curMasters: redisutil.Nodes{
				createTestNode("existing-master", "1.1.1.1", "zone1", "drc-test-cluster-shard0", redisv1alpha1.RedisClusterNodeRoleMaster),
			},
			nodes: map[string]redisutil.Nodes{
				"drc-test-cluster-shard0": {
					createTestNode("existing-master", "1.1.1.1", "zone1", "drc-test-cluster-shard0", redisv1alpha1.RedisClusterNodeRoleMaster),
				},
			},
			expectedScaleUp: 0,
			expectedRecover: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logr.Discard()
			ctx := &Ctx{
				log:   log,
				nodes: tt.nodes,
			}

			scaleUpMasters, recoverMasters := ctx.GetNodestoScaledandRecover(tt.newMasters, tt.curMasters)

			assert.Equal(t, tt.expectedScaleUp, len(scaleUpMasters), "Should have correct number of scale up masters")
			assert.Equal(t, tt.expectedRecover, len(recoverMasters), "Should have correct number of recover masters")
		})
	}
}

// Helper function to create test nodes with custom NodeName
func createTestNodeWithNodeName(id, ip, zone, statefulSet string, role redisv1alpha1.RedisRole, nodeName string) *redisutil.Node {
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
		NodeName:              nodeName,
		ZoneDistributionRank:  0,
		NodeDistributionRank:  0,
		ShardDistributionRank: 0,
		Zonename:              zone,
		PodName:               id + "-pod",
		StatefulSet:           statefulSet,
	}
	return node
}

func TestCtx_PlaceMastersHA(t *testing.T) {
	log := logr.Discard()

	tests := []struct {
		name           string
		ssName         string
		haconf         *redisv1alpha1.HAspec
		nodes          redisutil.Nodes
		expectedResult *redisutil.Node
		expectNil      bool
	}{
		{
			name:   "Successful master placement with HA",
			ssName: "drc-test-cluster-shard0",
			haconf: &redisv1alpha1.HAspec{
				HaEnabled: true,
				ZonesInfo: map[string]string{
					"zone1": "zone1-info",
					"zone2": "zone2-info",
				},
			},
			nodes: redisutil.Nodes{
				createTestNode("node1", "1.1.1.1", "zone1", "drc-test-cluster-shard0", redisv1alpha1.RedisClusterNodeRoleSlave),
				createTestNode("node2", "1.1.1.2", "zone2", "drc-test-cluster-shard0", redisv1alpha1.RedisClusterNodeRoleSlave),
				createTestNode("node3", "1.1.1.3", "zone1", "drc-test-cluster-shard1", redisv1alpha1.RedisClusterNodeRoleSlave),
			},
			expectedResult: &redisutil.Node{ID: "node1"}, // Should get best ranked node
			expectNil:      false,
		},
		{
			name:   "No suitable node found - empty statefulset",
			ssName: "nonexistent-shard",
			haconf: &redisv1alpha1.HAspec{
				HaEnabled: true,
				ZonesInfo: map[string]string{
					"zone1": "zone1-info",
				},
			},
			nodes: redisutil.Nodes{
				createTestNode("node1", "1.1.1.1", "zone1", "drc-test-cluster-shard0", redisv1alpha1.RedisClusterNodeRoleSlave),
			},
			expectedResult: nil,
			expectNil:      true,
		},
		{
			name:   "No suitable node found - all nodes in different statefulset",
			ssName: "drc-test-cluster-shard2",
			haconf: &redisv1alpha1.HAspec{
				HaEnabled: true,
				ZonesInfo: map[string]string{
					"zone1": "zone1-info",
					"zone2": "zone2-info",
				},
			},
			nodes: redisutil.Nodes{
				createTestNode("node1", "1.1.1.1", "zone1", "drc-test-cluster-shard0", redisv1alpha1.RedisClusterNodeRoleSlave),
				createTestNode("node2", "1.1.1.2", "zone2", "drc-test-cluster-shard1", redisv1alpha1.RedisClusterNodeRoleSlave),
			},
			expectedResult: nil,
			expectNil:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create context
			cluster := &redisutil.Cluster{
				HaStatus: redisv1alpha1.HaStatusCreating,
			}
			ctx := NewCtx(tt.haconf, cluster, tt.nodes, 3, 2, "test-cluster", log)

			// Create ranking info
			nodesByStatefulSet := make(map[string]redisutil.Nodes)
			for _, node := range tt.nodes {
				nodesByStatefulSet[node.StatefulSet] = append(nodesByStatefulSet[node.StatefulSet], node)
			}
			zoneRankingInfo := NewRankingInfo(tt.haconf, nodesByStatefulSet, 3, 2, "test-cluster", log)
			zoneRankingInfo.InitRankingInfo(nodesByStatefulSet)
			zoneRankingInfo.ApplyZoneRankingPerNodeDistribution()

			// Test the function
			result := ctx.PlaceMastersHA(tt.ssName, zoneRankingInfo)

			if tt.expectNil {
				assert.Nil(t, result, "Expected nil result")
			} else {
				assert.NotNil(t, result, "Expected non-nil result")
				if result != nil && tt.expectedResult != nil {
					// Verify we got a node from the correct statefulset
					assert.Equal(t, tt.ssName, result.StatefulSet, "Node should be from the requested statefulset")
				}
			}
		})
	}
}

// Note: createTestNode function is already defined in ranking_test.go
