package clustering

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	rediskunv1alpha1 "github.com/dinesh-murugiah/rediscluster-operator/api/v1alpha1"
	"github.com/dinesh-murugiah/rediscluster-operator/utils/redisutil"
	"github.com/redis/go-redis/v9"
)

// Mock implementations
type MockAdmin struct {
	mock.Mock
}

func (m *MockAdmin) Connections() redisutil.IAdminConnections {
	args := m.Called()
	return args.Get(0).(redisutil.IAdminConnections)
}

func (m *MockAdmin) GetHashMaxSlot() redisutil.Slot {
	args := m.Called()
	return redisutil.Slot(args.Int(0))
}

func (m *MockAdmin) AddSlots(ctx context.Context, addr string, slots []redisutil.Slot) error {
	args := m.Called(ctx, addr, slots)
	return args.Error(0)
}

func (m *MockAdmin) SetSlots(ctx context.Context, addr, state string, slots []redisutil.Slot, nodeID string, namespace, name string) error {
	args := m.Called(ctx, addr, state, slots, nodeID, namespace, name)
	return args.Error(0)
}

func (m *MockAdmin) MigrateKeys(ctx context.Context, addr string, dest *redisutil.Node, slots []redisutil.Slot, batch int, timeout int, replace bool, namespace, name string) (int, error) {
	args := m.Called(ctx, addr, dest, slots, batch, timeout, replace, namespace, name)
	return args.Int(0), args.Error(1)
}

func (m *MockAdmin) ForgetNode(ctx context.Context, nodeID, namespace, name string) error {
	args := m.Called(ctx, nodeID, namespace, name)
	return args.Error(0)
}

func (m *MockAdmin) FailoverSlaveToMaster(ctx context.Context, slave *redisutil.Node, takeoverType redisutil.RedisFailover, namespace, name string) error {
	args := m.Called(ctx, slave, takeoverType, namespace, name)
	return args.Error(0)
}

func (m *MockAdmin) AttachNodeToCluster(ctx context.Context, addr, namespace, name string) error {
	args := m.Called(ctx, addr, namespace, name)
	return args.Error(0)
}

func (m *MockAdmin) FlushAndReset(ctx context.Context, addr string, resetType string, namespace, name string) error {
	args := m.Called(ctx, addr, resetType, namespace, name)
	return args.Error(0)
}

func (m *MockAdmin) AttachSlaveToMaster(ctx context.Context, slave *redisutil.Node, masterID string, namespace, name string) error {
	args := m.Called(ctx, slave, masterID, namespace, name)
	return args.Error(0)
}

func (m *MockAdmin) Close(ctx context.Context) {
	m.Called(ctx)
}

func (m *MockAdmin) GetClusterInfos(ctx context.Context, namespace, name string) (*redisutil.ClusterInfos, error) {
	args := m.Called(ctx, namespace, name)
	return args.Get(0).(*redisutil.ClusterInfos), args.Error(1)
}

func (m *MockAdmin) ClusterManagerNodeIsEmpty(ctx context.Context, namespace, name string) (bool, error) {
	args := m.Called(ctx, namespace, name)
	return args.Bool(0), args.Error(1)
}

func (m *MockAdmin) SetConfigEpoch(ctx context.Context, namespace, name string) error {
	args := m.Called(ctx, namespace, name)
	return args.Error(0)
}

func (m *MockAdmin) SetConfigIfNeed(ctx context.Context, newConfig map[string]string, namespace, name string) error {
	args := m.Called(ctx, newConfig, namespace, name)
	return args.Error(0)
}

func (m *MockAdmin) GetAllConfig(ctx context.Context, c *redis.Client, addr, namespace, name string) (map[string]string, error) {
	args := m.Called(ctx, c, addr, namespace, name)
	return args.Get(0).(map[string]string), args.Error(1)
}

func (m *MockAdmin) DetachSlave(ctx context.Context, slave *redisutil.Node, namespace, name string) error {
	args := m.Called(ctx, slave, namespace, name)
	return args.Error(0)
}

func (m *MockAdmin) SetSlot(ctx context.Context, addr, action string, slot redisutil.Slot, nodeID, namespace, name string) error {
	args := m.Called(ctx, addr, action, slot, nodeID, namespace, name)
	return args.Error(0)
}

func (m *MockAdmin) MigrateKeysInSlot(ctx context.Context, addr string, dest *redisutil.Node, slot redisutil.Slot, batch, timeout int, replace bool, namespace, name string) (int, error) {
	args := m.Called(ctx, addr, dest, slot, batch, timeout, replace, namespace, name)
	return args.Int(0), args.Error(1)
}

func (m *MockAdmin) ResetPassword(ctx context.Context, newPassword, namespace, name string) error {
	args := m.Called(ctx, newPassword, namespace, name)
	return args.Error(0)
}

func (m *MockAdmin) GetReplicationStatus(ip string, ctx context.Context, namespace, name string) (string, redisutil.ReplicationInfo, error) {
	args := m.Called(ip, ctx, namespace, name)
	return args.String(0), args.Get(1).(redisutil.ReplicationInfo), args.Error(2)
}

func (m *MockAdmin) GetInfoReplication(ipPort string, ctx context.Context, namespace, name string) (redisutil.ReplicationInfo, error) {
	args := m.Called(ipPort, ctx, namespace, name)
	return args.Get(0).(redisutil.ReplicationInfo), args.Error(1)
}

func (m *MockAdmin) DrainClientsInOldMaster(ctx context.Context, node *redisutil.Node, namespace, name string) error {
	args := m.Called(ctx, node, namespace, name)
	return args.Error(0)
}

type MockAdminConnections struct {
	mock.Mock
}

func (m *MockAdminConnections) Add(ctx context.Context, addr string) error {
	args := m.Called(ctx, addr)
	return args.Error(0)
}

func (m *MockAdminConnections) Remove(ctx context.Context, addr string) {
	m.Called(ctx, addr)
}

func (m *MockAdminConnections) Get(ctx context.Context, addr string) (*redis.Client, error) {
	args := m.Called(ctx, addr)
	return args.Get(0).(*redis.Client), args.Error(1)
}

func (m *MockAdminConnections) GetRandom() (*redis.Client, error) {
	args := m.Called()
	return args.Get(0).(*redis.Client), args.Error(1)
}

func (m *MockAdminConnections) GetDifferentFrom(addr string) (*redis.Client, error) {
	args := m.Called(addr)
	return args.Get(0).(*redis.Client), args.Error(1)
}

func (m *MockAdminConnections) GetAll() map[string]*redis.Client {
	args := m.Called()
	return args.Get(0).(map[string]*redis.Client)
}

func (m *MockAdminConnections) GetSelected(addrs []string) map[string]*redis.Client {
	args := m.Called(addrs)
	return args.Get(0).(map[string]*redis.Client)
}

func (m *MockAdminConnections) Reconnect(ctx context.Context, addr string) error {
	args := m.Called(ctx, addr)
	return args.Error(0)
}

func (m *MockAdminConnections) AddAll(ctx context.Context, addrs []string) {
	m.Called(ctx, addrs)
}

func (m *MockAdminConnections) ReplaceAll(ctx context.Context, addrs []string) {
	m.Called(ctx, addrs)
}

func (m *MockAdminConnections) ValidateResp(ctx context.Context, resp *redis.Cmd, addr, errMessage string) error {
	args := m.Called(ctx, resp, addr, errMessage)
	return args.Error(0)
}

// Helper function to create test nodes
func createTestNodes() redisutil.Nodes {
	return redisutil.Nodes{
		{
			ID:       "node1",
			IP:       "10.0.0.1",
			Port:     "6379",
			Role:     "master",
			NodeName: "node-1",
			Zonename: "zone-a",
			Slots:    redisutil.BuildSlotSlice(0, 5461), // Assign some slots
		},
		{
			ID:       "node2",
			IP:       "10.0.0.2",
			Port:     "6379",
			Role:     "master",
			NodeName: "node-2",
			Zonename: "zone-b",
			Slots:    redisutil.BuildSlotSlice(5462, 10923), // Assign some slots
		},
		{
			ID:       "node3",
			IP:       "10.0.0.3",
			Port:     "6379",
			Role:     "master",
			NodeName: "node-3",
			Zonename: "zone-c",
			Slots:    redisutil.BuildSlotSlice(10924, 16383), // Assign some slots
		},
		{
			ID:       "node4",
			IP:       "10.0.0.4",
			Port:     "6379",
			Role:     "slave",
			NodeName: "node-4",
			Zonename: "zone-a",
		},
		{
			ID:       "node5",
			IP:       "10.0.0.5",
			Port:     "6379",
			Role:     "slave",
			NodeName: "node-5",
			Zonename: "zone-b",
		},
		{
			ID:       "node6",
			IP:       "10.0.0.6",
			Port:     "6379",
			Role:     "slave",
			NodeName: "node-6",
			Zonename: "zone-c",
		},
	}
}

// Helper function to create test cluster
func createTestCluster() *redisutil.Cluster {
	nodes := createTestNodes()
	// Convert to map[string]*redisutil.Node
	nodesMap := make(map[string]*redisutil.Node)
	for _, node := range nodes {
		nodesMap[node.ID] = node
	}
	return &redisutil.Cluster{
		Status: rediskunv1alpha1.ClusterStatusCreating,
		Nodes:  nodesMap,
	}
}

func TestDispatchMasters(t *testing.T) {
	tests := []struct {
		name              string
		cluster           *redisutil.Cluster
		nodes             redisutil.Nodes
		nbMaster          int32
		expectedMasters   int
		expectedError     bool
		expectedStatus    rediskunv1alpha1.ClusterStatus
		expectedPlacement rediskunv1alpha1.NodesPlacementInfo
	}{
		{
			name:              "Successful master dispatch - optimal placement",
			cluster:           createTestCluster(),
			nodes:             createTestNodes(), // Creates mixed master/slave nodes
			nbMaster:          3,
			expectedMasters:   3,
			expectedError:     false,
			expectedStatus:    rediskunv1alpha1.ClusterStatusCalculatingRebalancing,
			expectedPlacement: rediskunv1alpha1.NodesPlacementInfoOptimal,
		},
		{
			name: "Masters with slots only - should succeed",
			cluster: &redisutil.Cluster{
				Status: rediskunv1alpha1.ClusterStatusCreating,
			},
			nodes: redisutil.Nodes{
				// Masters with slots - should be sufficient
				func() *redisutil.Node {
					node := createTestNode("master1", "1.1.1.1", "zone1", "sts-0", rediskunv1alpha1.RedisClusterNodeRoleMaster)
					node.Slots = redisutil.BuildSlotSlice(0, 100)
					return node
				}(),
				func() *redisutil.Node {
					node := createTestNode("master2", "1.1.1.2", "zone2", "sts-1", rediskunv1alpha1.RedisClusterNodeRoleMaster)
					node.Slots = redisutil.BuildSlotSlice(101, 200)
					return node
				}(),
				func() *redisutil.Node {
					node := createTestNode("master3", "1.1.1.3", "zone3", "sts-2", rediskunv1alpha1.RedisClusterNodeRoleMaster)
					node.Slots = redisutil.BuildSlotSlice(201, 300)
					return node
				}(),
				// Some slaves
				createTestNode("slave1", "1.1.1.5", "zone1", "sts-0", rediskunv1alpha1.RedisClusterNodeRoleSlave),
			},
			nbMaster:          3,
			expectedMasters:   3,
			expectedError:     false,
			expectedStatus:    rediskunv1alpha1.ClusterStatusCalculatingRebalancing,
			expectedPlacement: rediskunv1alpha1.NodesPlacementInfoOptimal,
		},
		{
			name:            "Not enough master nodes",
			cluster:         createTestCluster(),
			nodes:           createTestNodes(),
			nbMaster:        10, // More than available
			expectedMasters: 0,
			expectedError:   true,
		},
		{
			name:            "Zero master nodes",
			cluster:         createTestCluster(),
			nodes:           redisutil.Nodes{},
			nbMaster:        3,
			expectedMasters: 0,
			expectedError:   true,
		},
		{
			name: "Only slaves available - should fail",
			cluster: &redisutil.Cluster{
				Status: rediskunv1alpha1.ClusterStatusCreating,
			},
			nodes: redisutil.Nodes{
				createTestNode("slave1", "1.1.1.1", "zone1", "sts-0", rediskunv1alpha1.RedisClusterNodeRoleSlave),
				createTestNode("slave2", "1.1.1.2", "zone2", "sts-1", rediskunv1alpha1.RedisClusterNodeRoleSlave),
				createTestNode("slave3", "1.1.1.3", "zone3", "sts-2", rediskunv1alpha1.RedisClusterNodeRoleSlave),
			},
			nbMaster:      3,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute DispatchMasters
			newMasters, currentMasters, allMasters, err := DispatchMasters(tt.cluster, tt.nodes, tt.nbMaster)

			// Assertions
			if tt.expectedError {
				assert.Error(t, err, "Should return error")
				assert.Contains(t, err.Error(), "not enough master available", "Error should mention insufficient masters")
			} else {
				assert.NoError(t, err, "Should not return error")
				assert.Len(t, newMasters, tt.expectedMasters, "Should have correct number of new masters")
				
				// Verify cluster status and placement
				assert.Equal(t, tt.expectedStatus, tt.cluster.Status, "Cluster status should be set correctly")
				assert.Equal(t, tt.expectedPlacement, tt.cluster.NodesPlacement, "Nodes placement should be set correctly")
				
				// Verify sorting - newMasters should be sorted by ID
				if len(newMasters) > 1 {
					for i := 1; i < len(newMasters); i++ {
						assert.True(t, newMasters[i-1].ID < newMasters[i].ID, "New masters should be sorted by ID")
					}
				}
				
				// Verify currentMasters contains only masters with slots
				for _, master := range currentMasters {
					assert.True(t, master.TotalSlots() > 0, "Current masters should have slots")
					assert.Equal(t, rediskunv1alpha1.RedisClusterNodeRoleMaster, master.GetRole(), "Should be master role")
				}
				
				// Verify allMasters contains all master nodes
				expectedAllMasters := tt.nodes.FilterByFunc(func(node *redisutil.Node) bool {
					return node.GetRole() == rediskunv1alpha1.RedisClusterNodeRoleMaster
				})
				assert.Len(t, allMasters, len(expectedAllMasters), "All masters should include all master nodes from input")
			}
		})
	}
}

func TestCtx_DispatchMasters(t *testing.T) {
	tests := []struct {
		name          string
		expectedError bool
		setupCtx      func() *Ctx
	}{
		{
			name:          "Successful master dispatch - no HA",
			expectedError: false,
			setupCtx: func() *Ctx {
				nodes := createTestNodes()
				nodesBySts := map[string]redisutil.Nodes{
					"drc-redis-cluster-shard0": {nodes[0], nodes[3]},
					"drc-redis-cluster-shard1": {nodes[1], nodes[4]},
					"drc-redis-cluster-shard2": {nodes[2], nodes[5]},
				}
				return &Ctx{
					log:                  logr.Discard(),
					expectedMasterNum:    3,
					expectedReplicaCount: 1,
					clusterName:          "redis-cluster",
					cluster:              createTestCluster(),
					nodes:                nodesBySts,
					newMastersBySts:      make(map[string]*redisutil.Node),
					haconfig:             &rediskunv1alpha1.HAspec{HaEnabled: false},
				}
			},
		},
		{
			name:          "Successful master dispatch - HA enabled with creating status",
			expectedError: false,
			setupCtx: func() *Ctx {
				nodes := createTestNodes()
				for i, node := range nodes {
					node.Zonename = fmt.Sprintf("zone%d", i%3) // Distribute across zones
				}
				nodesBySts := map[string]redisutil.Nodes{
					"drc-redis-cluster-shard0": {nodes[0], nodes[3]},
					"drc-redis-cluster-shard1": {nodes[1], nodes[4]},
					"drc-redis-cluster-shard2": {nodes[2], nodes[5]},
				}
				return &Ctx{
					log:                  logr.Discard(),
					expectedMasterNum:    3,
					expectedReplicaCount: 1,
					clusterName:          "redis-cluster",
					cluster:              createTestCluster(),
					nodes:                nodesBySts,
					newMastersBySts:      make(map[string]*redisutil.Node),
					haconfig: &rediskunv1alpha1.HAspec{
						HaEnabled: true,
						ZonesInfo: map[string]string{
							"zone0": "zone0-info",
							"zone1": "zone1-info",
							"zone2": "zone2-info",
						},
					},
					hastatus: rediskunv1alpha1.HaStatusCreating,
				}
			},
		},
		{
			name:          "HA enabled with single AZ proxy - successful placement",
			expectedError: false,
			setupCtx: func() *Ctx {
				nodes := createTestNodes()
				nodes[0].Zonename = "proxy-zone"
				nodes[3].Zonename = "proxy-zone"
				nodesBySts := map[string]redisutil.Nodes{
					"drc-redis-cluster-shard0": {nodes[0], nodes[3]},
					"drc-redis-cluster-shard1": {nodes[1], nodes[4]},
					"drc-redis-cluster-shard2": {nodes[2], nodes[5]},
				}
				return &Ctx{
					log:                  logr.Discard(),
					expectedMasterNum:    3,
					expectedReplicaCount: 1,
					clusterName:          "redis-cluster",
					cluster:              createTestCluster(),
					nodes:                nodesBySts,
					newMastersBySts:      make(map[string]*redisutil.Node),
					haconfig:             &rediskunv1alpha1.HAspec{HaEnabled: true},
					proxyPodAz:           "proxy-zone",
				}
			},
		},
		{
			name:          "Missing statefulset",
			expectedError: true,
			setupCtx: func() *Ctx {
				nodes := createTestNodes()
				nodesBySts := map[string]redisutil.Nodes{
					"drc-redis-cluster-shard0": {nodes[0], nodes[3]},
					// Missing drc-redis-cluster-shard1 and drc-redis-cluster-shard2
				}
				return &Ctx{
					log:                  logr.Discard(),
					expectedMasterNum:    3,
					expectedReplicaCount: 1,
					clusterName:          "redis-cluster",
					cluster:              createTestCluster(),
					nodes:                nodesBySts,
					newMastersBySts:      make(map[string]*redisutil.Node),
					haconfig:             &rediskunv1alpha1.HAspec{},
				}
			},
		},
		{
			name:          "Single existing master - should use it",
			expectedError: false,
			setupCtx: func() *Ctx {
				nodes := createTestNodes()
				// Set first node in each STS as master with slots
				nodes[0].Role = redisutil.RedisMasterRole
				nodes[0].Slots = redisutil.BuildSlotSlice(0, 100)
				nodes[1].Role = redisutil.RedisMasterRole
				nodes[1].Slots = redisutil.BuildSlotSlice(101, 200)
				nodes[2].Role = redisutil.RedisMasterRole
				nodes[2].Slots = redisutil.BuildSlotSlice(201, 300)

				nodesBySts := map[string]redisutil.Nodes{
					"drc-redis-cluster-shard0": {nodes[0], nodes[3]},
					"drc-redis-cluster-shard1": {nodes[1], nodes[4]},
					"drc-redis-cluster-shard2": {nodes[2], nodes[5]},
				}
				return &Ctx{
					log:                  logr.Discard(),
					expectedMasterNum:    3,
					expectedReplicaCount: 1,
					clusterName:          "redis-cluster",
					cluster:              createTestCluster(),
					nodes:                nodesBySts,
					newMastersBySts:      make(map[string]*redisutil.Node),
					haconfig:             &rediskunv1alpha1.HAspec{},
				}
			},
		},
		{
			name:          "Split brain scenario - multiple masters in STS",
			expectedError: true,
			setupCtx: func() *Ctx {
				nodes := createTestNodes()
				// Set both nodes in first STS as masters with slots (split brain)
				nodes[0].Role = redisutil.RedisMasterRole
				nodes[0].Slots = redisutil.BuildSlotSlice(0, 100)
				nodes[3].Role = redisutil.RedisMasterRole
				nodes[3].Slots = redisutil.BuildSlotSlice(101, 200)

				nodesBySts := map[string]redisutil.Nodes{
					"drc-redis-cluster-shard0": {nodes[0], nodes[3]}, // Both masters - split brain
					"drc-redis-cluster-shard1": {nodes[1], nodes[4]},
					"drc-redis-cluster-shard2": {nodes[2], nodes[5]},
				}
				return &Ctx{
					log:                  logr.Discard(),
					expectedMasterNum:    3,
					expectedReplicaCount: 1,
					clusterName:          "redis-cluster",
					cluster:              createTestCluster(),
					nodes:                nodesBySts,
					newMastersBySts:      make(map[string]*redisutil.Node),
					haconfig:             &rediskunv1alpha1.HAspec{},
				}
			},
		},
		{
			name:          "HA enabled with non-creating status - should log info",
			expectedError: false,
			setupCtx: func() *Ctx {
				nodes := createTestNodes()
				for i, node := range nodes {
					node.Zonename = fmt.Sprintf("zone%d", i%3) // Distribute across zones
				}
				nodesBySts := map[string]redisutil.Nodes{
					"drc-redis-cluster-shard0": {nodes[0], nodes[3]},
					"drc-redis-cluster-shard1": {nodes[1], nodes[4]},
					"drc-redis-cluster-shard2": {nodes[2], nodes[5]},
				}
				return &Ctx{
					log:                  logr.Discard(),
					expectedMasterNum:    3,
					expectedReplicaCount: 1,
					clusterName:          "redis-cluster",
					cluster:              createTestCluster(),
					nodes:                nodesBySts,
					newMastersBySts:      make(map[string]*redisutil.Node),
					haconfig: &rediskunv1alpha1.HAspec{
						HaEnabled: true,
						ZonesInfo: map[string]string{
							"zone0": "zone0-info",
							"zone1": "zone1-info",
							"zone2": "zone2-info",
						},
					},
					hastatus: rediskunv1alpha1.HaStatusHealthy, // Non-creating status
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup context
			ctx := tt.setupCtx()

			// Execute DispatchMasters
			err := ctx.DispatchMasters()

			// Assertions
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, ctx.newMastersBySts, ctx.expectedMasterNum)
			}
		})
	}
}

func TestCtx_PlaceSlaves(t *testing.T) {
	tests := []struct {
		name          string
		expectedError bool
		setupCtx      func() *Ctx
	}{
		{
			name:          "Successful slave placement",
			expectedError: false,
			setupCtx: func() *Ctx {
				nodes := createTestNodes()
				masters := redisutil.Nodes{nodes[0], nodes[1], nodes[2]}
				_ = redisutil.Nodes{nodes[3], nodes[4], nodes[5]} // slaves not used in this test

				return &Ctx{
					log:                  logr.Discard(),
					expectedMasterNum:    3,
					expectedReplicaCount: 1,
					clusterName:          "redis-cluster",
					cluster:              createTestCluster(),
					currentMasters:       masters,
					slavesByMaster:       make(map[string]redisutil.Nodes),
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup context
			ctx := tt.setupCtx()

			// Execute PlaceSlaves
			err := ctx.PlaceSlaves()

			// Assertions
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCtx_sortRedisNodeByStatefulSet(t *testing.T) {
	tests := []struct {
		name                   string
		inputNodes             redisutil.Nodes
		clusterNodes           map[string]*redisutil.Node
		initialCurrentMasters  redisutil.Nodes
		expectedGroups         map[string]int // StatefulSet name -> expected node count
		expectedCurrentMasters int
	}{
		{
			name: "Normal case - nodes with StatefulSet names",
			inputNodes: redisutil.Nodes{
				createTestNode("node1", "1.1.1.1", "zone1", "sts-0", rediskunv1alpha1.RedisClusterNodeRoleMaster),
				createTestNode("node2", "1.1.1.2", "zone1", "sts-0", rediskunv1alpha1.RedisClusterNodeRoleSlave),
				createTestNode("node3", "1.1.1.3", "zone2", "sts-1", rediskunv1alpha1.RedisClusterNodeRoleMaster),
			},
			clusterNodes: map[string]*redisutil.Node{
				"node1": createTestNode("node1", "1.1.1.1", "zone1", "sts-0", rediskunv1alpha1.RedisClusterNodeRoleMaster),
				"node2": createTestNode("node2", "1.1.1.2", "zone1", "sts-0", rediskunv1alpha1.RedisClusterNodeRoleSlave),
				"node3": createTestNode("node3", "1.1.1.3", "zone2", "sts-1", rediskunv1alpha1.RedisClusterNodeRoleMaster),
			},
			expectedGroups: map[string]int{
				"sts-0": 2,
				"sts-1": 1,
			},
			expectedCurrentMasters: 0, // No masters with slots
		},
		{
			name: "Masters with slots - should be added to currentMasters",
			inputNodes: redisutil.Nodes{
				func() *redisutil.Node {
					node := createTestNode("master1", "1.1.1.1", "zone1", "sts-0", rediskunv1alpha1.RedisClusterNodeRoleMaster)
					node.Slots = redisutil.BuildSlotSlice(0, 100) // Has slots
					return node
				}(),
				func() *redisutil.Node {
					node := createTestNode("master2", "1.1.1.2", "zone2", "sts-1", rediskunv1alpha1.RedisClusterNodeRoleMaster)
					node.Slots = redisutil.BuildSlotSlice(101, 200) // Has slots
					return node
				}(),
				createTestNode("slave1", "1.1.1.3", "zone1", "sts-0", rediskunv1alpha1.RedisClusterNodeRoleSlave),
			},
			clusterNodes: map[string]*redisutil.Node{
				"master1": func() *redisutil.Node {
					node := createTestNode("master1", "1.1.1.1", "zone1", "sts-0", rediskunv1alpha1.RedisClusterNodeRoleMaster)
					node.Slots = redisutil.BuildSlotSlice(0, 100)
					return node
				}(),
				"master2": func() *redisutil.Node {
					node := createTestNode("master2", "1.1.1.2", "zone2", "sts-1", rediskunv1alpha1.RedisClusterNodeRoleMaster)
					node.Slots = redisutil.BuildSlotSlice(101, 200)
					return node
				}(),
				"slave1": createTestNode("slave1", "1.1.1.3", "zone1", "sts-0", rediskunv1alpha1.RedisClusterNodeRoleSlave),
			},
			expectedGroups: map[string]int{
				"sts-0": 2,
				"sts-1": 1,
			},
			expectedCurrentMasters: 2, // Both masters have slots
		},
		{
			name: "Node lookup error - should skip node",
			inputNodes: redisutil.Nodes{
				createTestNode("existing-node", "1.1.1.1", "zone1", "sts-0", rediskunv1alpha1.RedisClusterNodeRoleMaster),
				createTestNode("missing-node", "1.1.1.2", "zone1", "sts-0", rediskunv1alpha1.RedisClusterNodeRoleSlave),
			},
			clusterNodes: map[string]*redisutil.Node{
				"existing-node": createTestNode("existing-node", "1.1.1.1", "zone1", "sts-0", rediskunv1alpha1.RedisClusterNodeRoleMaster),
				// "missing-node" not in cluster - should cause lookup error
			},
			expectedGroups: map[string]int{
				"sts-0": 1, // Only existing node should be grouped
			},
			expectedCurrentMasters: 0,
		},
		{
			name: "Empty StatefulSet name - should use unknownVMName",
			inputNodes: redisutil.Nodes{
				createTestNode("node1", "1.1.1.1", "zone1", "", rediskunv1alpha1.RedisClusterNodeRoleMaster), // Empty StatefulSet
			},
			clusterNodes: map[string]*redisutil.Node{
				"node1": createTestNode("node1", "1.1.1.1", "zone1", "", rediskunv1alpha1.RedisClusterNodeRoleMaster),
			},
			expectedGroups: map[string]int{
				unknownVMName: 1, // Should use unknownVMName constant
			},
			expectedCurrentMasters: 0,
		},
		{
			name: "Nil currentMasters initialization",
			inputNodes: redisutil.Nodes{
				createTestNode("node1", "1.1.1.1", "zone1", "sts-0", rediskunv1alpha1.RedisClusterNodeRoleMaster),
			},
			clusterNodes: map[string]*redisutil.Node{
				"node1": createTestNode("node1", "1.1.1.1", "zone1", "sts-0", rediskunv1alpha1.RedisClusterNodeRoleMaster),
			},
			initialCurrentMasters: nil, // Should be initialized to empty slice
			expectedGroups: map[string]int{
				"sts-0": 1,
			},
			expectedCurrentMasters: 0,
		},
		{
			name: "Sorting current masters by StatefulSet name",
			inputNodes: redisutil.Nodes{
				func() *redisutil.Node {
					node := createTestNode("master-z", "1.1.1.1", "zone1", "sts-z", rediskunv1alpha1.RedisClusterNodeRoleMaster)
					node.Slots = redisutil.BuildSlotSlice(0, 100)
					return node
				}(),
				func() *redisutil.Node {
					node := createTestNode("master-a", "1.1.1.2", "zone2", "sts-a", rediskunv1alpha1.RedisClusterNodeRoleMaster)
					node.Slots = redisutil.BuildSlotSlice(101, 200)
					return node
				}(),
				func() *redisutil.Node {
					node := createTestNode("master-m", "1.1.1.3", "zone3", "sts-m", rediskunv1alpha1.RedisClusterNodeRoleMaster)
					node.Slots = redisutil.BuildSlotSlice(201, 300)
					return node
				}(),
			},
			clusterNodes: map[string]*redisutil.Node{
				"master-z": func() *redisutil.Node {
					node := createTestNode("master-z", "1.1.1.1", "zone1", "sts-z", rediskunv1alpha1.RedisClusterNodeRoleMaster)
					node.Slots = redisutil.BuildSlotSlice(0, 100)
					return node
				}(),
				"master-a": func() *redisutil.Node {
					node := createTestNode("master-a", "1.1.1.2", "zone2", "sts-a", rediskunv1alpha1.RedisClusterNodeRoleMaster)
					node.Slots = redisutil.BuildSlotSlice(101, 200)
					return node
				}(),
				"master-m": func() *redisutil.Node {
					node := createTestNode("master-m", "1.1.1.3", "zone3", "sts-m", rediskunv1alpha1.RedisClusterNodeRoleMaster)
					node.Slots = redisutil.BuildSlotSlice(201, 300)
					return node
				}(),
			},
			expectedGroups: map[string]int{
				"sts-a": 1,
				"sts-m": 1,
				"sts-z": 1,
			},
			expectedCurrentMasters: 3, // Should be sorted: sts-a, sts-m, sts-z
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cluster := &redisutil.Cluster{
				Nodes: tt.clusterNodes,
			}

			ctx := &Ctx{
				log:            logr.Discard(),
				cluster:        cluster,
				currentMasters: tt.initialCurrentMasters,
			}

			result := ctx.sortRedisNodeByStatefulSet(tt.inputNodes)

			// Verify grouping
			assert.Len(t, result, len(tt.expectedGroups), "Should have correct number of StatefulSet groups")
			for stsName, expectedCount := range tt.expectedGroups {
				assert.Len(t, result[stsName], expectedCount, "StatefulSet %s should have %d nodes", stsName, expectedCount)
			}

			// Verify currentMasters
			assert.Len(t, ctx.currentMasters, tt.expectedCurrentMasters, "Should have correct number of current masters")
			assert.NotNil(t, ctx.currentMasters, "currentMasters should not be nil")

			// Verify sorting of currentMasters (if more than one)
			if len(ctx.currentMasters) > 1 {
				for i := 1; i < len(ctx.currentMasters); i++ {
					assert.True(t, ctx.currentMasters[i-1].StatefulSet <= ctx.currentMasters[i].StatefulSet,
						"Current masters should be sorted by StatefulSet name")
				}
			}
		})
	}
}

func TestCtx_retrieveSlotToMigrateFrom(t *testing.T) {
	tests := []struct {
		name              string
		oldMasterNodes    redisutil.Nodes
		nbSlotByNode      int
		expectedSlots     map[string]int // NodeID -> expected number of slots to migrate
	}{
		{
			name: "Normal case - nodes with excess slots",
			oldMasterNodes: redisutil.Nodes{
				func() *redisutil.Node {
					node := createTestNode("master1", "1.1.1.1", "zone1", "sts-0", rediskunv1alpha1.RedisClusterNodeRoleMaster)
					node.Slots = redisutil.BuildSlotSlice(0, 199) // 200 slots
					return node
				}(),
				func() *redisutil.Node {
					node := createTestNode("master2", "1.1.1.2", "zone2", "sts-1", rediskunv1alpha1.RedisClusterNodeRoleMaster)
					node.Slots = redisutil.BuildSlotSlice(200, 349) // 150 slots
					return node
				}(),
			},
			nbSlotByNode: 100,
			expectedSlots: map[string]int{
				"master1": 100, // 200 - 100 = 100 slots to migrate
				"master2": 50,  // 150 - 100 = 50 slots to migrate
			},
		},
		{
			name: "Exact slot count - no migration needed",
			oldMasterNodes: redisutil.Nodes{
				func() *redisutil.Node {
					node := createTestNode("master1", "1.1.1.1", "zone1", "sts-0", rediskunv1alpha1.RedisClusterNodeRoleMaster)
					node.Slots = redisutil.BuildSlotSlice(0, 99) // Exactly 100 slots
					return node
				}(),
			},
			nbSlotByNode: 100,
			expectedSlots: map[string]int{}, // No slots to migrate
		},
		{
			name: "Insufficient slots - no migration needed",
			oldMasterNodes: redisutil.Nodes{
				func() *redisutil.Node {
					node := createTestNode("master1", "1.1.1.1", "zone1", "sts-0", rediskunv1alpha1.RedisClusterNodeRoleMaster)
					node.Slots = redisutil.BuildSlotSlice(0, 49) // Only 50 slots
					return node
				}(),
			},
			nbSlotByNode: 100,
			expectedSlots: map[string]int{}, // No slots to migrate
		},
		{
			name: "Mixed scenario - some nodes migrate, others don't",
			oldMasterNodes: redisutil.Nodes{
				func() *redisutil.Node {
					node := createTestNode("master1", "1.1.1.1", "zone1", "sts-0", rediskunv1alpha1.RedisClusterNodeRoleMaster)
					node.Slots = redisutil.BuildSlotSlice(0, 299) // 300 slots
					return node
				}(),
				func() *redisutil.Node {
					node := createTestNode("master2", "1.1.1.2", "zone2", "sts-1", rediskunv1alpha1.RedisClusterNodeRoleMaster)
					node.Slots = redisutil.BuildSlotSlice(300, 399) // Exactly 100 slots
					return node
				}(),
				func() *redisutil.Node {
					node := createTestNode("master3", "1.1.1.3", "zone3", "sts-2", rediskunv1alpha1.RedisClusterNodeRoleMaster)
					node.Slots = redisutil.BuildSlotSlice(400, 449) // Only 50 slots
					return node
				}(),
			},
			nbSlotByNode: 100,
			expectedSlots: map[string]int{
				"master1": 200, // 300 - 100 = 200 slots to migrate
				// master2 and master3 don't migrate (exact/insufficient slots)
			},
		},
		{
			name: "Empty nodes list",
			oldMasterNodes: redisutil.Nodes{},
			nbSlotByNode: 100,
			expectedSlots: map[string]int{}, // No slots to migrate
		},
		{
			name: "Single slot excess",
			oldMasterNodes: redisutil.Nodes{
				func() *redisutil.Node {
					node := createTestNode("master1", "1.1.1.1", "zone1", "sts-0", rediskunv1alpha1.RedisClusterNodeRoleMaster)
					node.Slots = redisutil.BuildSlotSlice(0, 100) // 101 slots (0-100 inclusive)
					return node
				}(),
			},
			nbSlotByNode: 100,
			expectedSlots: map[string]int{
				"master1": 1, // Only 1 slot to migrate
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &Ctx{
				log: logr.Discard(),
			}

			result := ctx.retrieveSlotToMigrateFrom(tt.oldMasterNodes, tt.nbSlotByNode)

			// Verify the number of nodes with slots to migrate
			assert.Len(t, result, len(tt.expectedSlots), "Should have correct number of nodes with slots to migrate")

			// Verify each node's slot migration count
			for nodeID, expectedCount := range tt.expectedSlots {
				slots, exists := result[nodeID]
				assert.True(t, exists, "Node %s should have slots to migrate", nodeID)
				assert.Len(t, slots, expectedCount, "Node %s should have %d slots to migrate", nodeID, expectedCount)
				
				// Verify slots are unique and in order
				if len(slots) > 1 {
					for i := 1; i < len(slots); i++ {
						assert.True(t, slots[i-1] < slots[i], "Slots should be in ascending order")
					}
				}
			}

			// Verify no unexpected nodes have slots to migrate
			for nodeID := range result {
				_, expected := tt.expectedSlots[nodeID]
				assert.True(t, expected, "Node %s should not have slots to migrate", nodeID)
			}
		})
	}
}

func TestCtx_AttachingSlavesToMaster(t *testing.T) {
	tests := []struct {
		name           string
		expectedError  bool
		slavesByMaster map[string]redisutil.Nodes
		clusterNodes   map[string]*redisutil.Node
		setupMocks     func(*MockAdmin)
	}{
		{
			name:          "Successful slave attachment",
			expectedError: false,
			slavesByMaster: map[string]redisutil.Nodes{
				"master1": {
					{ID: "slave1", IP: "10.0.0.4", Port: "6379", Role: "slave"},
				},
			},
			clusterNodes: map[string]*redisutil.Node{
				"master1": {ID: "master1", IP: "10.0.0.1", Port: "6379", Role: "master"},
				"slave1":  {ID: "slave1", IP: "10.0.0.4", Port: "6379", Role: "slave"},
			},
			setupMocks: func(admin *MockAdmin) {
				admin.On("AttachSlaveToMaster", mock.Anything, mock.AnythingOfType("*redisutil.Node"), "master1", "test-ns", "test-cluster").Return(nil)
			},
		},
		{
			name:          "Multiple slaves to multiple masters",
			expectedError: false,
			slavesByMaster: map[string]redisutil.Nodes{
				"master1": {
					{ID: "slave1", IP: "10.0.0.4", Port: "6379", Role: "slave"},
					{ID: "slave2", IP: "10.0.0.5", Port: "6379", Role: "slave"},
				},
				"master2": {
					{ID: "slave3", IP: "10.0.0.6", Port: "6379", Role: "slave"},
				},
			},
			clusterNodes: map[string]*redisutil.Node{
				"master1": {ID: "master1", IP: "10.0.0.1", Port: "6379", Role: "master"},
				"master2": {ID: "master2", IP: "10.0.0.2", Port: "6379", Role: "master"},
				"slave1":  {ID: "slave1", IP: "10.0.0.4", Port: "6379", Role: "slave"},
				"slave2":  {ID: "slave2", IP: "10.0.0.5", Port: "6379", Role: "slave"},
				"slave3":  {ID: "slave3", IP: "10.0.0.6", Port: "6379", Role: "slave"},
			},
			setupMocks: func(admin *MockAdmin) {
				admin.On("AttachSlaveToMaster", mock.Anything, mock.AnythingOfType("*redisutil.Node"), "master1", "test-ns", "test-cluster").Return(nil).Times(2)
				admin.On("AttachSlaveToMaster", mock.Anything, mock.AnythingOfType("*redisutil.Node"), "master2", "test-ns", "test-cluster").Return(nil)
			},
		},
		{
			name:          "Attachment failure - should return error",
			expectedError: true,
			slavesByMaster: map[string]redisutil.Nodes{
				"master1": {
					{ID: "slave1", IP: "10.0.0.4", Port: "6379", Role: "slave"},
				},
			},
			clusterNodes: map[string]*redisutil.Node{
				"master1": {ID: "master1", IP: "10.0.0.1", Port: "6379", Role: "master"},
				"slave1":  {ID: "slave1", IP: "10.0.0.4", Port: "6379", Role: "slave"},
			},
			setupMocks: func(admin *MockAdmin) {
				admin.On("AttachSlaveToMaster", mock.Anything, mock.AnythingOfType("*redisutil.Node"), "master1", "test-ns", "test-cluster").Return(fmt.Errorf("attachment failed"))
			},
		},
		{
			name:          "Master not found - continues processing",
			expectedError: false,
			slavesByMaster: map[string]redisutil.Nodes{
				"nonexistent": {
					{ID: "slave1", IP: "10.0.0.4", Port: "6379", Role: "slave"},
				},
				"master1": {
					{ID: "slave2", IP: "10.0.0.5", Port: "6379", Role: "slave"},
				},
			},
			clusterNodes: map[string]*redisutil.Node{
				"master1": {ID: "master1", IP: "10.0.0.1", Port: "6379", Role: "master"},
				"slave1":  {ID: "slave1", IP: "10.0.0.4", Port: "6379", Role: "slave"},
				"slave2":  {ID: "slave2", IP: "10.0.0.5", Port: "6379", Role: "slave"},
			},
			setupMocks: func(admin *MockAdmin) {
				// Should only be called for the existing master
				admin.On("AttachSlaveToMaster", mock.Anything, mock.AnythingOfType("*redisutil.Node"), "master1", "test-ns", "test-cluster").Return(nil)
			},
		},
		{
			name:           "No slaves to attach - no error",
			expectedError:  false,
			slavesByMaster: map[string]redisutil.Nodes{},
			clusterNodes: map[string]*redisutil.Node{
				"master1": {ID: "master1", IP: "10.0.0.1", Port: "6379", Role: "master"},
			},
			setupMocks: func(admin *MockAdmin) {
				// No calls should be made
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup context
			cluster := &redisutil.Cluster{
				Status: rediskunv1alpha1.ClusterStatusCreating,
				Nodes:  tt.clusterNodes,
			}

			ctx := &Ctx{
				log:                  logr.Discard(),
				expectedMasterNum:    3,
				expectedReplicaCount: 1,
				clusterName:          "redis-cluster",
				cluster:              cluster,
				slavesByMaster:       tt.slavesByMaster,
			}

			// Setup mocks
			admin := &MockAdmin{}
			if tt.setupMocks != nil {
				tt.setupMocks(admin)
			}

			// Execute AttachingSlavesToMaster
			err := ctx.AttachingSlavesToMaster(admin, context.Background(), "test-ns", "test-cluster")

			// Assertions
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Verify all mock expectations were met
			admin.AssertExpectations(t)
		})
	}
}

func TestCtx_AllocSlots(t *testing.T) {
	tests := []struct {
		name          string
		masters       redisutil.Nodes
		expectedError bool
		setupMocks    func(*MockAdmin)
	}{
		{
			name:          "Successful slot allocation",
			masters:       createTestNodes()[:3], // First 3 nodes as masters
			expectedError: false,
			setupMocks: func(admin *MockAdmin) {
				admin.On("GetHashMaxSlot").Return(16383)
				admin.On("AddSlots", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			name:          "AddSlots fails",
			masters:       createTestNodes()[:3],
			expectedError: true,
			setupMocks: func(admin *MockAdmin) {
				admin.On("GetHashMaxSlot").Return(16383)
				admin.On("AddSlots", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("add slots failed"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup context
			ctx := &Ctx{
				log:                  logr.Discard(),
				expectedMasterNum:    3,
				expectedReplicaCount: 1,
				clusterName:          "redis-cluster",
				cluster:              createTestCluster(),
			}

			// Setup mocks
			admin := &MockAdmin{}
			if tt.setupMocks != nil {
				tt.setupMocks(admin)
			}

			// Execute AllocSlots
			err := ctx.AllocSlots(admin, tt.masters, context.Background())

			// Assertions
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCtx_RebalancedCluster(t *testing.T) {
	tests := []struct {
		name          string
		masters       redisutil.Nodes
		expectedError bool
		setupMocks    func(*MockAdmin)
	}{
		{
			name:          "Successful cluster rebalancing",
			masters:       createTestNodes()[:3],
			expectedError: false,
			setupMocks: func(admin *MockAdmin) {
				admin.On("GetHashMaxSlot").Return(16383)
				admin.On("SetSlot", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
				admin.On("MigrateKeysInSlot", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(0, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup context
			ctx := &Ctx{
				log:                  logr.Discard(),
				expectedMasterNum:    3,
				expectedReplicaCount: 1,
				clusterName:          "redis-cluster",
				cluster:              createTestCluster(),
			}

			// Setup mocks
			admin := &MockAdmin{}
			if tt.setupMocks != nil {
				tt.setupMocks(admin)
			}

			// Execute RebalancedCluster
			err := ctx.RebalancedCluster(admin, tt.masters, context.Background(), "test-ns", "test-cluster")

			// Assertions
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCtx_DispatchSlotToNewMasters(t *testing.T) {
	tests := []struct {
		name           string
		newMasters     redisutil.Nodes
		currentMasters redisutil.Nodes
		allMasters     redisutil.Nodes
		expectedError  bool
		setupMocks     func(*MockAdmin)
	}{
		{
			name:           "Successful slot dispatch",
			newMasters:     createTestNodes()[:2],
			currentMasters: createTestNodes()[2:3],
			allMasters:     createTestNodes()[:3],
			expectedError:  false,
			setupMocks: func(admin *MockAdmin) {
				admin.On("GetHashMaxSlot").Return(16383)
				admin.On("AddSlots", mock.Anything, mock.Anything, mock.Anything).Return(nil)
				admin.On("SetSlots", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
				admin.On("MigrateKeys", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(0, nil)
			},
		},
		{
			name:           "Error during AddSlots",
			newMasters:     createTestNodes()[:2],
			currentMasters: createTestNodes()[2:3],
			allMasters:     createTestNodes()[:3],
			expectedError:  true,
			setupMocks: func(admin *MockAdmin) {
				admin.On("GetHashMaxSlot").Return(16383)
				admin.On("AddSlots", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("addslots failed"))
			},
		},
		{
			name:           "Error during IMPORTING SetSlots",
			newMasters:     createTestNodes()[:2],
			currentMasters: createTestNodes()[2:3],
			allMasters:     createTestNodes()[:3],
			expectedError:  true,
			setupMocks: func(admin *MockAdmin) {
				admin.On("GetHashMaxSlot").Return(16383)
				admin.On("AddSlots", mock.Anything, mock.Anything, mock.Anything).Return(nil)
				admin.On("SetSlots", mock.Anything, mock.Anything, "IMPORTING", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("importing failed"))
			},
		},
		{
			name:           "Error during MIGRATING SetSlots",
			newMasters:     createTestNodes()[:2],
			currentMasters: createTestNodes()[2:3],
			allMasters:     createTestNodes()[:3],
			expectedError:  true,
			setupMocks: func(admin *MockAdmin) {
				admin.On("GetHashMaxSlot").Return(16383)
				admin.On("AddSlots", mock.Anything, mock.Anything, mock.Anything).Return(nil)
				admin.On("SetSlots", mock.Anything, mock.Anything, "IMPORTING", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
				admin.On("SetSlots", mock.Anything, mock.Anything, "MIGRATING", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("migrating failed"))
			},
		},
		{
			name:           "Successful migration with key migration error (non-blocking)",
			newMasters:     createTestNodes()[:2],
			currentMasters: createTestNodes()[2:3],
			allMasters:     createTestNodes()[:3],
			expectedError:  false, // Key migration errors don't block the process
			setupMocks: func(admin *MockAdmin) {
				admin.On("GetHashMaxSlot").Return(16383)
				admin.On("AddSlots", mock.Anything, mock.Anything, mock.Anything).Return(nil)
				admin.On("SetSlots", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
				admin.On("MigrateKeys", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(0, fmt.Errorf("key migration failed"))
			},
		},
		{
			name:           "Successful with NODE SetSlots warnings (non-blocking)",
			newMasters:     createTestNodes()[:2],
			currentMasters: createTestNodes()[2:3],
			allMasters:     createTestNodes()[:3],
			expectedError:  false, // NODE SetSlots errors are warnings, don't block
			setupMocks: func(admin *MockAdmin) {
				admin.On("GetHashMaxSlot").Return(16383)
				admin.On("AddSlots", mock.Anything, mock.Anything, mock.Anything).Return(nil)
				admin.On("SetSlots", mock.Anything, mock.Anything, "IMPORTING", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
				admin.On("SetSlots", mock.Anything, mock.Anything, "MIGRATING", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
				admin.On("MigrateKeys", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(5, nil)
				// NODE SetSlots calls return errors but are treated as warnings
				admin.On("SetSlots", mock.Anything, mock.Anything, "NODE", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("node setslot warning"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup context
			ctx := &Ctx{
				log:                  logr.Discard(),
				expectedMasterNum:    3,
				expectedReplicaCount: 1,
				clusterName:          "redis-cluster",
				cluster:              createTestCluster(),
			}

			// Setup mocks
			admin := &MockAdmin{}
			if tt.setupMocks != nil {
				tt.setupMocks(admin)
			}

			// Execute DispatchSlotToNewMasters
			err := ctx.DispatchSlotToNewMasters(admin, tt.newMasters, tt.currentMasters, tt.allMasters, "test-ns", "test-cluster", context.Background())

			// Assertions
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCtx_CheckAndHealCluster(t *testing.T) {
	tests := []struct {
		name              string
		expectedError     bool
		expectedErrorType ErrorType
		setupCtx          func() *Ctx
		setupMocks        func(*MockAdmin)
	}{
		{
			name:          "No healing needed - current masters equal expected",
			expectedError: false,
			setupCtx: func() *Ctx {
				nodes := createTestNodes()
				nodesMap := make(map[string]redisutil.Nodes)
				nodesMap["sts-1"] = nodes[:3]

				return &Ctx{
					log:                  logr.Discard(),
					expectedMasterNum:    3,
					expectedReplicaCount: 1,
					clusterName:          "redis-cluster",
					cluster:              createTestCluster(),
					currentMasters:       nodes[:3], // 3 masters as expected
					nodes:                nodesMap,
				}
			},
			setupMocks: func(admin *MockAdmin) {
				// No calls should be made
			},
		},
		{
			name:          "First time cluster creation - no healing needed",
			expectedError: false, // Should return nil for first time creation
			setupCtx: func() *Ctx {
				nodes := createTestNodes()
				nodesMap := make(map[string]redisutil.Nodes)
				nodesMap["sts-1"] = nodes[:3]

				// Create new masters map to simulate first time creation
				newMasters := make(map[string]*redisutil.Node)
				newMasters["sts-1"] = nodes[0]
				newMasters["sts-2"] = nodes[1]
				newMasters["sts-3"] = nodes[2]

				return &Ctx{
					log:                  logr.Discard(),
					expectedMasterNum:    3,
					expectedReplicaCount: 1,
					clusterName:          "redis-cluster",
					cluster:              createTestCluster(),
					currentMasters:       redisutil.Nodes{},                // No current masters
					newMastersBySts:      newMasters,                       // 3 new masters
					slavesByMaster:       make(map[string]redisutil.Nodes), // No slaves
					nodes:                nodesMap,
				}
			},
			setupMocks: func(admin *MockAdmin) {
				// No calls should be made for first time creation
			},
		},
		{
			name:          "Scale up scenario - no recovery needed",
			expectedError: false, // Should return nil when no recovery needed
			setupCtx: func() *Ctx {
				nodes := createTestNodes()
				nodesMap := make(map[string]redisutil.Nodes)
				nodesMap["sts-1"] = nodes[:2]  // Only 2 nodes available
				nodesMap["sts-2"] = nodes[2:3] // 1 more node

				// Create scenario where curMasters < newMasters but no recovery needed
				// This happens when GetNodestoScaledandRecover returns empty recoverMasters
				newMasters := make(map[string]*redisutil.Node)
				newMasters["sts-1"] = nodes[0] // Already a current master
				newMasters["sts-2"] = nodes[1] // Already a current master
				newMasters["sts-3"] = nodes[2] // New master (scale up)

				return &Ctx{
					log:                  logr.Discard(),
					expectedMasterNum:    3,
					expectedReplicaCount: 1,
					clusterName:          "redis-cluster",
					cluster:              createTestCluster(),
					currentMasters:       nodes[:2],  // 2 current masters
					newMastersBySts:      newMasters, // 3 new masters
					nodes:                nodesMap,
				}
			},
			setupMocks: func(admin *MockAdmin) {
				// No calls should be made
			},
		},
		{
			name:              "Quorum available - no healing",
			expectedError:     true,
			expectedErrorType: HealReconcile,
			setupCtx: func() *Ctx {
				nodes := createTestNodes()
				// Set up roles correctly - need slaves in the STS to trigger recovery
				nodes[0].Role = redisutil.RedisMasterRole // Current master
				nodes[1].Role = redisutil.RedisMasterRole // Current master
				nodes[2].Role = redisutil.RedisSlaveRole  // Slave in sts-recover
				nodes[2].StatefulSet = "sts-recover"

				nodesMap := make(map[string]redisutil.Nodes)
				nodesMap["sts-1"] = redisutil.Nodes{nodes[0]}
				nodesMap["sts-2"] = redisutil.Nodes{nodes[1]}
				nodesMap["sts-recover"] = redisutil.Nodes{nodes[2]} // STS with slave - triggers recovery

				// Create recovery scenario: 5 expected masters, 4 current (quorum available)
				newMasters := make(map[string]*redisutil.Node)
				newMasters["sts-1"] = nodes[0]       // Existing master
				newMasters["sts-2"] = nodes[1]       // Existing master
				newMasters["sts-3"] = nodes[3]       // Existing master
				newMasters["sts-4"] = nodes[4]       // Existing master
				newMasters["sts-recover"] = nodes[2] // New master in STS with slave - needs recovery

				return &Ctx{
					log:                  logr.Discard(),
					expectedMasterNum:    5, // Need 5 expected masters
					expectedReplicaCount: 1,
					clusterName:          "redis-cluster",
					cluster:              createTestCluster(),
					currentMasters:       redisutil.Nodes{nodes[0], nodes[1], nodes[3], nodes[4]}, // 4 current masters (quorum: 4 > (5/2)+1 = 4 > 3 = true)
					newMastersBySts:      newMasters,
					nodes:                nodesMap,
					haconfig:             &rediskunv1alpha1.HAspec{HaEnabled: false},
				}
			},
			setupMocks: func(admin *MockAdmin) {
				// No failover should be called due to quorum
			},
		},
		{
			name:              "Successful healing with failover",
			expectedError:     true,
			expectedErrorType: HealReconcile,
			setupCtx: func() *Ctx {
				nodes := createTestNodes()
				// Set up proper roles for the test
				nodes[0].Role = redisutil.RedisMasterRole // Current master
				nodes[1].Role = redisutil.RedisSlaveRole  // Slave that will be promoted
				nodes[2].Role = redisutil.RedisSlaveRole  // Another slave
				nodes[1].StatefulSet = "sts-recover"      // The statefulset that needs recovery

				nodesMap := make(map[string]redisutil.Nodes)
				nodesMap["sts-1"] = redisutil.Nodes{nodes[0]}
				nodesMap["sts-recover"] = redisutil.Nodes{nodes[1], nodes[2]}

				// Create recovery scenario: need to recover one master
				newMasters := make(map[string]*redisutil.Node)
				newMasters["sts-1"] = nodes[0]
				newMasters["sts-recover"] = nodes[1] // This triggers recovery

				return &Ctx{
					log:                  logr.Discard(),
					expectedMasterNum:    3,
					expectedReplicaCount: 1,
					clusterName:          "redis-cluster",
					cluster:              createTestCluster(),
					currentMasters:       redisutil.Nodes{nodes[0]}, // Only 1 master (less than quorum of 2)
					newMastersBySts:      newMasters,
					nodes:                nodesMap,
					haconfig:             &rediskunv1alpha1.HAspec{HaEnabled: false},
				}
			},
			setupMocks: func(admin *MockAdmin) {
				admin.On("FailoverSlaveToMaster", mock.Anything, mock.AnythingOfType("*redisutil.Node"), redisutil.RedisFailoverTakeover, "test-ns", "test-cluster").Return(nil)
			},
		},
		{
			name:              "Healing failure - failover error",
			expectedError:     true,
			expectedErrorType: HealFailure,
			setupCtx: func() *Ctx {
				nodes := createTestNodes()
				nodes[0].Role = redisutil.RedisMasterRole
				nodes[1].Role = redisutil.RedisSlaveRole
				nodes[1].StatefulSet = "sts-recover"

				nodesMap := make(map[string]redisutil.Nodes)
				nodesMap["sts-1"] = redisutil.Nodes{nodes[0]}
				nodesMap["sts-recover"] = redisutil.Nodes{nodes[1]}

				newMasters := make(map[string]*redisutil.Node)
				newMasters["sts-1"] = nodes[0]
				newMasters["sts-recover"] = nodes[1]

				return &Ctx{
					log:                  logr.Discard(),
					expectedMasterNum:    3,
					expectedReplicaCount: 1,
					clusterName:          "redis-cluster",
					cluster:              createTestCluster(),
					currentMasters:       redisutil.Nodes{nodes[0]},
					newMastersBySts:      newMasters,
					nodes:                nodesMap,
					haconfig:             &rediskunv1alpha1.HAspec{HaEnabled: false},
				}
			},
			setupMocks: func(admin *MockAdmin) {
				admin.On("FailoverSlaveToMaster", mock.Anything, mock.AnythingOfType("*redisutil.Node"), redisutil.RedisFailoverTakeover, "test-ns", "test-cluster").Return(fmt.Errorf("failover failed"))
			},
		},
		{
			name:              "Healing failure - node is not a slave",
			expectedError:     true,
			expectedErrorType: HealFailure,
			setupCtx: func() *Ctx {
				nodes := createTestNodes()
				nodes[0].Role = redisutil.RedisMasterRole
				nodes[1].Role = redisutil.RedisMasterRole // This should be a slave but is a master - will cause validation failure
				nodes[1].StatefulSet = "sts-recover"
				nodes[2].Role = redisutil.RedisSlaveRole // Add a slave in the STS to trigger recovery logic
				nodes[2].StatefulSet = "sts-recover"

				nodesMap := make(map[string]redisutil.Nodes)
				nodesMap["sts-1"] = redisutil.Nodes{nodes[0]}
				nodesMap["sts-recover"] = redisutil.Nodes{nodes[1], nodes[2]} // STS with master+slave

				newMasters := make(map[string]*redisutil.Node)
				newMasters["sts-1"] = nodes[0]
				newMasters["sts-recover"] = nodes[1] // This will be the recovery target but it's a master (validation fails)

				return &Ctx{
					log:                  logr.Discard(),
					expectedMasterNum:    3,
					expectedReplicaCount: 1,
					clusterName:          "redis-cluster",
					cluster:              createTestCluster(),
					currentMasters:       redisutil.Nodes{nodes[0]}, // Only 1 master (less than quorum)
					newMastersBySts:      newMasters,
					nodes:                nodesMap,
					haconfig:             &rediskunv1alpha1.HAspec{HaEnabled: false},
				}
			},
			setupMocks: func(admin *MockAdmin) {
				// No failover should be called due to validation failure
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup context
			ctx := tt.setupCtx()

			// Setup mocks
			admin := &MockAdmin{}
			if tt.setupMocks != nil {
				tt.setupMocks(admin)
			}

			// Execute CheckAndHealCluster
			err := ctx.CheckAndHealCluster(admin, context.Background(), "test-ns", "test-cluster")

			// Assertions
			if tt.expectedError {
				assert.Error(t, err)
				if tt.expectedErrorType != NoType {
					assert.Equal(t, tt.expectedErrorType, GetType(err), "Expected specific error type")
				}
			} else {
				assert.NoError(t, err)
			}

			// Verify all mock expectations were met
			admin.AssertExpectations(t)
		})
	}
}

func TestPlaceMasters(t *testing.T) {
	tests := []struct {
		name               string
		cluster            *redisutil.Cluster
		currentMaster      redisutil.Nodes
		allPossibleMasters redisutil.Nodes
		nbMaster           int32
		expectedMasters    int
		expectedBestEffort bool
		expectedError      bool
	}{
		{
			name:               "Successful master placement",
			cluster:            createTestCluster(),
			currentMaster:      redisutil.Nodes{},
			allPossibleMasters: createTestNodes(),
			nbMaster:           3,
			expectedMasters:    3,
			expectedBestEffort: false,
			expectedError:      false,
		},
		{
			name:               "Not enough masters available",
			cluster:            createTestCluster(),
			currentMaster:      redisutil.Nodes{},
			allPossibleMasters: createTestNodes()[:2], // Only 2 available
			nbMaster:           3,
			expectedMasters:    0,
			expectedBestEffort: false,
			expectedError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute PlaceMasters
			masters, bestEffort, err := PlaceMasters(tt.cluster, tt.currentMaster, tt.allPossibleMasters, tt.nbMaster)

			// Assertions
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, masters, tt.expectedMasters)
				assert.Equal(t, tt.expectedBestEffort, bestEffort)
			}
		})
	}
}

func TestPlaceSlaves(t *testing.T) {
	tests := []struct {
		name               string
		cluster            *redisutil.Cluster
		masters            redisutil.Nodes
		oldSlaves          redisutil.Nodes
		newSlaves          redisutil.Nodes
		replicationFactor  int32
		expectedBestEffort bool
		expectedSlaveCount int
	}{
		{
			name:               "Successful slave placement",
			cluster:            createTestCluster(),
			masters:            createTestNodes()[:3],
			oldSlaves:          redisutil.Nodes{},
			newSlaves:          createTestNodes()[3:],
			replicationFactor:  1,
			expectedBestEffort: false,
			expectedSlaveCount: 3, // 3 masters, 1 slave each
		},
		{
			name:               "Remove duplicate slaves from old and new",
			cluster:            createTestCluster(),
			masters:            createTestNodes()[:2],                                       // 2 masters
			oldSlaves:          redisutil.Nodes{createTestNodes()[3]},                       // 1 old slave
			newSlaves:          redisutil.Nodes{createTestNodes()[3], createTestNodes()[4]}, // Duplicate + new slave
			replicationFactor:  1,
			expectedBestEffort: false,
			expectedSlaveCount: 2, // Should place 2 slaves (duplicate removed)
		},
		{
			name:    "Old slaves with existing master referent",
			cluster: createTestCluster(),
			masters: createTestNodes()[:2],
			oldSlaves: redisutil.Nodes{
				func() *redisutil.Node {
					node := createTestNodes()[3]
					node.MasterReferent = createTestNodes()[0].ID // Assign to first master
					return node
				}(),
			},
			newSlaves:          redisutil.Nodes{createTestNodes()[4]},
			replicationFactor:  1,
			expectedBestEffort: false,
			expectedSlaveCount: 2, // 1 old slave + 1 new slave
		},
		{
			name:    "Replication factor limit - old slave overflow",
			cluster: createTestCluster(),
			masters: createTestNodes()[:1], // Only 1 master
			oldSlaves: redisutil.Nodes{
				func() *redisutil.Node {
					node := createTestNodes()[3]
					node.MasterReferent = createTestNodes()[0].ID
					return node
				}(),
				func() *redisutil.Node {
					node := createTestNodes()[4]
					node.MasterReferent = createTestNodes()[0].ID
					return node
				}(),
			},
			newSlaves:          redisutil.Nodes{},
			replicationFactor:  1, // Only 1 slave per master
			expectedBestEffort: false,
			expectedSlaveCount: 1, // Only 1 slave should be assigned due to replication factor
		},
		{
			name: "VM conflict avoidance",
			cluster: func() *redisutil.Cluster {
				cluster := createTestCluster()
				// Set up masters and slaves on same VM to test conflict detection
				masters := createTestNodes()[:2]
				slaves := createTestNodes()[3:5]

				// Master and slave on same VM (should be avoided)
				masters[0].NodeName = "vm-1"
				slaves[0].NodeName = "vm-1"
				slaves[1].NodeName = "vm-2"

				// Add nodes to cluster
				for _, node := range masters {
					cluster.Nodes[node.ID] = node
				}
				for _, node := range slaves {
					cluster.Nodes[node.ID] = node
				}

				return cluster
			}(),
			masters:            createTestNodes()[:2],
			oldSlaves:          redisutil.Nodes{},
			newSlaves:          createTestNodes()[3:5],
			replicationFactor:  1,
			expectedBestEffort: false,
			expectedSlaveCount: 2,
		},
		{
			name: "Best effort when VM conflicts force suboptimal placement",
			cluster: func() *redisutil.Cluster {
				cluster := createTestCluster()
				masters := createTestNodes()[:2]
				slaves := createTestNodes()[3:5]

				// All nodes on same VM - forces best effort
				for _, node := range masters {
					node.NodeName = "vm-same"
					cluster.Nodes[node.ID] = node
				}
				for _, node := range slaves {
					node.NodeName = "vm-same"
					cluster.Nodes[node.ID] = node
				}

				return cluster
			}(),
			masters:            createTestNodes()[:2],
			oldSlaves:          redisutil.Nodes{},
			newSlaves:          createTestNodes()[3:5],
			replicationFactor:  1,
			expectedBestEffort: true, // Should trigger best effort due to VM conflicts
			expectedSlaveCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute PlaceSlaves
			slavesByMaster, bestEffort := PlaceSlaves(tt.cluster, tt.masters, tt.oldSlaves, tt.newSlaves, tt.replicationFactor)

			// Assertions
			assert.Equal(t, tt.expectedBestEffort, bestEffort, "Best effort flag should match expected")
			assert.NotNil(t, slavesByMaster, "SlavesByMaster should not be nil")

			// Count total slaves assigned
			totalSlaves := 0
			for _, slaves := range slavesByMaster {
				totalSlaves += len(slaves)
			}
			assert.Equal(t, tt.expectedSlaveCount, totalSlaves, "Total slave count should match expected")

			// Verify each master has a slot in the map
			for _, master := range tt.masters {
				_, exists := slavesByMaster[master.ID]
				assert.True(t, exists, "Each master should have an entry in slavesByMaster map")
			}
		})
	}
}

func TestRebalancedCluster(t *testing.T) {
	tests := []struct {
		name          string
		masters       redisutil.Nodes
		expectedError bool
		setupMocks    func(*MockAdmin)
	}{
		{
			name:          "Successful cluster rebalancing",
			masters:       createTestNodes()[:3],
			expectedError: false,
			setupMocks: func(admin *MockAdmin) {
				admin.On("GetHashMaxSlot").Return(16383)
				admin.On("SetSlot", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
				admin.On("MigrateKeysInSlot", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(0, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup context
			ctx := &Ctx{
				log:                  logr.Discard(),
				expectedMasterNum:    3,
				expectedReplicaCount: 1,
				clusterName:          "redis-cluster",
				cluster:              createTestCluster(),
			}

			// Setup mocks
			admin := &MockAdmin{}
			if tt.setupMocks != nil {
				tt.setupMocks(admin)
			}

			// Execute RebalancedCluster
			err := ctx.RebalancedCluster(admin, tt.masters, context.Background(), "test-ns", "test-cluster")

			// Assertions
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestComputeReshardTable(t *testing.T) {
	tests := []struct {
		name     string
		src      redisutil.Nodes
		numSlots int
		expected int
	}{
		{
			name:     "Compute reshard table for 10 slots",
			src:      createTestNodes()[:3],
			numSlots: 10,
			expected: 10,
		},
		{
			name:     "Compute reshard table for 0 slots",
			src:      createTestNodes()[:3],
			numSlots: 0,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute computeReshardTable
			result := computeReshardTable(tt.src, tt.numSlots)

			// Assertions
			assert.Len(t, result, tt.expected)
		})
	}
}

func TestCtx_moveSlot(t *testing.T) {
	tests := []struct {
		name          string
		source        *MovedNode
		target        *redisutil.Node
		expectedError bool
		setupMocks    func(*MockAdmin)
	}{
		{
			name: "Successful slot move",
			source: &MovedNode{
				Source: &redisutil.Node{
					ID: "node1", IP: "10.0.0.1", Port: "6379",
					Slots: redisutil.BuildSlotSlice(1000, 1000), // Include slot 1000
				},
				Slot: redisutil.Slot(1000),
			},
			target:        &redisutil.Node{ID: "node2", IP: "10.0.0.2", Port: "6379"},
			expectedError: false,
			setupMocks: func(admin *MockAdmin) {
				admin.On("SetSlot", mock.Anything, "10.0.0.2:6379", "IMPORTING", redisutil.Slot(1000), "node2", mock.Anything, mock.Anything).Return(nil)
				admin.On("SetSlot", mock.Anything, "10.0.0.1:6379", "MIGRATING", redisutil.Slot(1000), "node1", mock.Anything, mock.Anything).Return(nil)
				admin.On("MigrateKeysInSlot", mock.Anything, "10.0.0.1:6379", mock.AnythingOfType("*redisutil.Node"), redisutil.Slot(1000), 10, 30000, true, mock.Anything, mock.Anything).Return(0, nil)
				admin.On("SetSlot", mock.Anything, "10.0.0.2:6379", "NODE", redisutil.Slot(1000), "node2", mock.Anything, mock.Anything).Return(nil)
				admin.On("SetSlot", mock.Anything, "10.0.0.1:6379", "NODE", redisutil.Slot(1000), "node2", mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			name: "Error on target IMPORTING setup",
			source: &MovedNode{
				Source: &redisutil.Node{ID: "node1", IP: "10.0.0.1", Port: "6379"},
				Slot:   redisutil.Slot(1000),
			},
			target:        &redisutil.Node{ID: "node2", IP: "10.0.0.2", Port: "6379"},
			expectedError: true,
			setupMocks: func(admin *MockAdmin) {
				admin.On("SetSlot", mock.Anything, "10.0.0.2:6379", "IMPORTING", redisutil.Slot(1000), "node2", mock.Anything, mock.Anything).Return(fmt.Errorf("importing failed"))
			},
		},
		{
			name: "Error on source MIGRATING setup",
			source: &MovedNode{
				Source: &redisutil.Node{ID: "node1", IP: "10.0.0.1", Port: "6379"},
				Slot:   redisutil.Slot(1000),
			},
			target:        &redisutil.Node{ID: "node2", IP: "10.0.0.2", Port: "6379"},
			expectedError: true,
			setupMocks: func(admin *MockAdmin) {
				admin.On("SetSlot", mock.Anything, "10.0.0.2:6379", "IMPORTING", redisutil.Slot(1000), "node2", mock.Anything, mock.Anything).Return(nil)
				admin.On("SetSlot", mock.Anything, "10.0.0.1:6379", "MIGRATING", redisutil.Slot(1000), "node1", mock.Anything, mock.Anything).Return(fmt.Errorf("migrating failed"))
			},
		},
		{
			name: "Error on key migration",
			source: &MovedNode{
				Source: &redisutil.Node{ID: "node1", IP: "10.0.0.1", Port: "6379"},
				Slot:   redisutil.Slot(1000),
			},
			target:        &redisutil.Node{ID: "node2", IP: "10.0.0.2", Port: "6379"},
			expectedError: true,
			setupMocks: func(admin *MockAdmin) {
				admin.On("SetSlot", mock.Anything, "10.0.0.2:6379", "IMPORTING", redisutil.Slot(1000), "node2", mock.Anything, mock.Anything).Return(nil)
				admin.On("SetSlot", mock.Anything, "10.0.0.1:6379", "MIGRATING", redisutil.Slot(1000), "node1", mock.Anything, mock.Anything).Return(nil)
				admin.On("MigrateKeysInSlot", mock.Anything, "10.0.0.1:6379", mock.AnythingOfType("*redisutil.Node"), redisutil.Slot(1000), 10, 30000, true, mock.Anything, mock.Anything).Return(0, fmt.Errorf("migration failed"))
			},
		},
		{
			name: "Error on target NODE finalization - should not fail overall",
			source: &MovedNode{
				Source: &redisutil.Node{
					ID: "node1", IP: "10.0.0.1", Port: "6379",
					Slots: redisutil.BuildSlotSlice(1000, 1000), // Include slot 1000
				},
				Slot: redisutil.Slot(1000),
			},
			target:        &redisutil.Node{ID: "node2", IP: "10.0.0.2", Port: "6379"},
			expectedError: false, // Should not fail overall, just logs error
			setupMocks: func(admin *MockAdmin) {
				admin.On("SetSlot", mock.Anything, "10.0.0.2:6379", "IMPORTING", redisutil.Slot(1000), "node2", mock.Anything, mock.Anything).Return(nil)
				admin.On("SetSlot", mock.Anything, "10.0.0.1:6379", "MIGRATING", redisutil.Slot(1000), "node1", mock.Anything, mock.Anything).Return(nil)
				admin.On("MigrateKeysInSlot", mock.Anything, "10.0.0.1:6379", mock.AnythingOfType("*redisutil.Node"), redisutil.Slot(1000), 10, 30000, true, mock.Anything, mock.Anything).Return(0, nil)
				admin.On("SetSlot", mock.Anything, "10.0.0.2:6379", "NODE", redisutil.Slot(1000), "node2", mock.Anything, mock.Anything).Return(fmt.Errorf("target node finalization failed"))
				admin.On("SetSlot", mock.Anything, "10.0.0.1:6379", "NODE", redisutil.Slot(1000), "node2", mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			name: "Error on source NODE finalization - should not fail overall",
			source: &MovedNode{
				Source: &redisutil.Node{
					ID: "node1", IP: "10.0.0.1", Port: "6379",
					Slots: redisutil.BuildSlotSlice(1000, 1000), // Include slot 1000
				},
				Slot: redisutil.Slot(1000),
			},
			target:        &redisutil.Node{ID: "node2", IP: "10.0.0.2", Port: "6379"},
			expectedError: false, // Should not fail overall, just logs error
			setupMocks: func(admin *MockAdmin) {
				admin.On("SetSlot", mock.Anything, "10.0.0.2:6379", "IMPORTING", redisutil.Slot(1000), "node2", mock.Anything, mock.Anything).Return(nil)
				admin.On("SetSlot", mock.Anything, "10.0.0.1:6379", "MIGRATING", redisutil.Slot(1000), "node1", mock.Anything, mock.Anything).Return(nil)
				admin.On("MigrateKeysInSlot", mock.Anything, "10.0.0.1:6379", mock.AnythingOfType("*redisutil.Node"), redisutil.Slot(1000), 10, 30000, true, mock.Anything, mock.Anything).Return(0, nil)
				admin.On("SetSlot", mock.Anything, "10.0.0.2:6379", "NODE", redisutil.Slot(1000), "node2", mock.Anything, mock.Anything).Return(nil)
				admin.On("SetSlot", mock.Anything, "10.0.0.1:6379", "NODE", redisutil.Slot(1000), "node2", mock.Anything, mock.Anything).Return(fmt.Errorf("source node finalization failed"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup context
			ctx := &Ctx{
				log:                  logr.Discard(),
				expectedMasterNum:    3,
				expectedReplicaCount: 1,
				clusterName:          "redis-cluster",
				cluster:              createTestCluster(),
			}

			// Setup mocks
			admin := &MockAdmin{}
			if tt.setupMocks != nil {
				tt.setupMocks(admin)
			}

			// Execute moveSlot
			err := ctx.moveSlot(tt.source, tt.target, admin, context.Background(), "test-ns", "test-cluster")

			// Assertions
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
