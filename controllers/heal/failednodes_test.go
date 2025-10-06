package heal

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	redisv1alpha1 "github.com/freshworks/redis-cluster-operator/api/v1alpha1"
	"github.com/freshworks/redis-cluster-operator/utils/redisutil"
)

// Use MockAdmin from manager package - simplified version for heal package
type MockRedisAdmin struct {
	mock.Mock
}

func (m *MockRedisAdmin) ForgetNode(ctx context.Context, id string, namespace string, name string) error {
	args := m.Called(ctx, id, namespace, name)
	return args.Error(0)
}

// Minimal required methods to satisfy the interface
func (m *MockRedisAdmin) Connections() redisutil.IAdminConnections { return nil }
func (m *MockRedisAdmin) Close(ctx context.Context)                {}
func (m *MockRedisAdmin) GetClusterInfos(ctx context.Context, namespace string, name string) (*redisutil.ClusterInfos, error) {
	return nil, nil
}
func (m *MockRedisAdmin) ClusterManagerNodeIsEmpty(ctx context.Context, namespace string, name string) (bool, error) {
	return false, nil
}
func (m *MockRedisAdmin) SetConfigEpoch(ctx context.Context, namespace string, name string) error {
	return nil
}
func (m *MockRedisAdmin) SetConfigIfNeed(ctx context.Context, newConfig map[string]string, namespace string, name string) error {
	return nil
}
func (m *MockRedisAdmin) GetAllConfig(ctx context.Context, c *redis.Client, addr string, namespace string, name string) (map[string]string, error) {
	return nil, nil
}
func (m *MockRedisAdmin) AttachNodeToCluster(ctx context.Context, addr string, namespace string, name string) error {
	return nil
}
func (m *MockRedisAdmin) AttachSlaveToMaster(ctx context.Context, slave *redisutil.Node, masterID string, namespace string, name string) error {
	return nil
}
func (m *MockRedisAdmin) DetachSlave(ctx context.Context, slave *redisutil.Node, namespace string, name string) error {
	return nil
}
func (m *MockRedisAdmin) SetSlots(ctx context.Context, addr string, action string, slots []redisutil.Slot, nodeID string, namespace string, name string) error {
	return nil
}
func (m *MockRedisAdmin) AddSlots(ctx context.Context, addr string, slots []redisutil.Slot) error {
	return nil
}
func (m *MockRedisAdmin) SetSlot(ctx context.Context, addr string, action string, slot redisutil.Slot, nodeID string, namespace string, name string) error {
	return nil
}
func (m *MockRedisAdmin) MigrateKeys(ctx context.Context, addr string, dest *redisutil.Node, slots []redisutil.Slot, batch, timeout int, replace bool, namespace string, name string) (int, error) {
	return 0, nil
}
func (m *MockRedisAdmin) MigrateKeysInSlot(ctx context.Context, addr string, dest *redisutil.Node, slot redisutil.Slot, batch int, timeout int, replace bool, namespace string, name string) (int, error) {
	return 0, nil
}
func (m *MockRedisAdmin) FlushAndReset(ctx context.Context, addr string, mode string, namespace string, name string) error {
	return nil
}
func (m *MockRedisAdmin) GetHashMaxSlot() redisutil.Slot { return 0 }
func (m *MockRedisAdmin) ResetPassword(ctx context.Context, newPassword string, namespace string, name string) error {
	return nil
}
func (m *MockRedisAdmin) FailoverSlaveToMaster(ctx context.Context, slave *redisutil.Node, option redisutil.RedisFailover, namespace string, name string) error {
	return nil
}
func (m *MockRedisAdmin) GetReplicationStatus(ip string, ctx context.Context, namespace string, name string) (string, redisutil.ReplicationInfo, error) {
	return "", redisutil.ReplicationInfo{}, nil
}
func (m *MockRedisAdmin) GetInfoReplication(ipPort string, ctx context.Context, namespace string, name string) (redisutil.ReplicationInfo, error) {
	return redisutil.ReplicationInfo{}, nil
}
func (m *MockRedisAdmin) DrainClientsInOldMaster(ctx context.Context, oldMaster *redisutil.Node, namespace string, name string) error {
	return nil
}

func TestCheckAndHeal_FixFailedNodes_NoGhostNodes(t *testing.T) {
	// Test when there are no ghost nodes to fix
	checkAndHeal := &CheckAndHeal{
		Logger:        logr.Discard(),
		PodControl:    &MockPodControl{},
		DryRun:        false,
		MetricsClient: &MockMetrics{},
	}

	cluster := &redisv1alpha1.DistributedRedisCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster", Namespace: "test-ns"},
		Spec: redisv1alpha1.DistributedRedisClusterSpec{
			MasterSize:      3,
			ClusterReplicas: 1,
		},
	}

	// Empty cluster infos - no ghost nodes
	infos := &redisutil.ClusterInfos{
		Infos: map[string]*redisutil.NodeInfos{},
	}

	admin := &MockRedisAdmin{}
	ctx := context.Background()

	actionDone, err := checkAndHeal.FixFailedNodes(cluster, infos, admin, ctx)

	assert.NoError(t, err)
	assert.False(t, actionDone)

	admin.AssertExpectations(t)
}

func TestCheckAndHeal_FixFailedNodes_DryRun(t *testing.T) {
	// Test when DryRun is enabled - should detect ghost nodes but not forget them
	checkAndHeal := &CheckAndHeal{
		Logger:        logr.Discard(),
		PodControl:    &MockPodControl{},
		DryRun:        true, // DryRun enabled
		MetricsClient: &MockMetrics{},
	}

	cluster := &redisv1alpha1.DistributedRedisCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster", Namespace: "test-ns"},
		Spec: redisv1alpha1.DistributedRedisClusterSpec{
			MasterSize:      3,
			ClusterReplicas: 1,
		},
	}

	// Create cluster infos with ghost nodes
	infos := createClusterInfosWithGhostNodes()

	admin := &MockRedisAdmin{}
	ctx := context.Background()

	actionDone, err := checkAndHeal.FixFailedNodes(cluster, infos, admin, ctx)

	assert.NoError(t, err)
	assert.True(t, actionDone) // Action was detected but not performed due to DryRun

	// ForgetNode should not be called in DryRun mode
	admin.AssertNotCalled(t, "ForgetNode")
}

func TestCheckAndHeal_FixFailedNodes_Success(t *testing.T) {
	// Test successful fixing of failed nodes
	checkAndHeal := &CheckAndHeal{
		Logger:        logr.Discard(),
		PodControl:    &MockPodControl{},
		DryRun:        false,
		MetricsClient: &MockMetrics{},
	}

	cluster := &redisv1alpha1.DistributedRedisCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster", Namespace: "test-ns"},
		Spec: redisv1alpha1.DistributedRedisClusterSpec{
			MasterSize:      3,
			ClusterReplicas: 1,
		},
	}

	// Create cluster infos with ghost nodes
	infos := createClusterInfosWithGhostNodes()

	admin := &MockRedisAdmin{}
	ctx := context.Background()

	// Mock ForgetNode to succeed
	admin.On("ForgetNode", ctx, mock.AnythingOfType("string"), "test-ns", "test-cluster").Return(nil)

	actionDone, err := checkAndHeal.FixFailedNodes(cluster, infos, admin, ctx)

	assert.NoError(t, err)
	assert.True(t, actionDone)

	admin.AssertExpectations(t)
}

func TestCheckAndHeal_FixFailedNodes_ForgetNodeError(t *testing.T) {
	// Test when ForgetNode returns an error
	checkAndHeal := &CheckAndHeal{
		Logger:        logr.Discard(),
		PodControl:    &MockPodControl{},
		DryRun:        false,
		MetricsClient: &MockMetrics{},
	}

	cluster := &redisv1alpha1.DistributedRedisCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster", Namespace: "test-ns"},
		Spec: redisv1alpha1.DistributedRedisClusterSpec{
			MasterSize:      3,
			ClusterReplicas: 1,
		},
	}

	// Create cluster infos with ghost nodes
	infos := createClusterInfosWithGhostNodes()

	admin := &MockRedisAdmin{}
	ctx := context.Background()

	expectedError := errors.New("failed to forget node")
	admin.On("ForgetNode", ctx, mock.AnythingOfType("string"), "test-ns", "test-cluster").Return(expectedError)

	actionDone, err := checkAndHeal.FixFailedNodes(cluster, infos, admin, ctx)

	assert.Error(t, err)
	assert.True(t, actionDone)
	assert.Contains(t, err.Error(), "failed to forget node")

	admin.AssertExpectations(t)
}

func TestListGhostNodes_NilInfos(t *testing.T) {
	// Test listGhostNodes with nil cluster infos
	cluster := &redisv1alpha1.DistributedRedisCluster{
		Spec: redisv1alpha1.DistributedRedisClusterSpec{
			MasterSize:      3,
			ClusterReplicas: 1,
		},
	}

	ghostNodes := listGhostNodes(cluster, nil, logr.Discard())

	assert.Empty(t, ghostNodes)
}

func TestListGhostNodes_EmptyInfos(t *testing.T) {
	// Test listGhostNodes with empty cluster infos
	cluster := &redisv1alpha1.DistributedRedisCluster{
		Spec: redisv1alpha1.DistributedRedisClusterSpec{
			MasterSize:      3,
			ClusterReplicas: 1,
		},
	}

	infos := &redisutil.ClusterInfos{
		Infos: nil,
	}

	ghostNodes := listGhostNodes(cluster, infos, logr.Discard())

	assert.Empty(t, ghostNodes)
}

func TestListGhostNodes_NoGhostNodes(t *testing.T) {
	// Test listGhostNodes when all nodes are healthy
	cluster := &redisv1alpha1.DistributedRedisCluster{
		Spec: redisv1alpha1.DistributedRedisClusterSpec{
			MasterSize:      3,
			ClusterReplicas: 1,
		},
	}

	infos := &redisutil.ClusterInfos{
		Infos: map[string]*redisutil.NodeInfos{
			"node1": {
				Friends: []*redisutil.Node{
					{
						ID:         "healthy-node-1",
						Role:       redisutil.RedisMasterRole,
						FailStatus: []string{}, // No failure status - healthy node
					},
					{
						ID:         "healthy-node-2",
						Role:       redisutil.RedisSlaveRole,
						FailStatus: []string{}, // No failure status - healthy node
					},
				},
			},
		},
	}

	ghostNodes := listGhostNodes(cluster, infos, logr.Discard())

	assert.Empty(t, ghostNodes)
}

func TestListGhostNodes_WithGhostNodes(t *testing.T) {
	// Test listGhostNodes when there are actual ghost nodes
	cluster := &redisv1alpha1.DistributedRedisCluster{
		Spec: redisv1alpha1.DistributedRedisClusterSpec{
			MasterSize:      3,
			ClusterReplicas: 1,
		},
	}

	// Create a scenario where multiple nodes report the same ghost node
	ghostNodeID := "ghost-node-123"
	infos := &redisutil.ClusterInfos{
		Infos: map[string]*redisutil.NodeInfos{
			"node1": {
				Friends: []*redisutil.Node{
					{
						ID:         ghostNodeID,
						FailStatus: []string{redisutil.NodeStatusFail},
					},
				},
			},
			"node2": {
				Friends: []*redisutil.Node{
					{
						ID:         ghostNodeID,
						FailStatus: []string{redisutil.NodeStatusNoAddr},
					},
				},
			},
			"node3": {
				Friends: []*redisutil.Node{
					{
						ID:         ghostNodeID,
						FailStatus: []string{redisutil.NodeStatusFail},
					},
				},
			},
		},
	}

	ghostNodes := listGhostNodes(cluster, infos, logr.Discard())

	// With MasterSize=3 and ClusterReplicas=1, we need (1+1)*3/2 = 3 reports
	// We have 3 reports (from node1, node2, node3), so it should be detected as ghost
	assert.Contains(t, ghostNodes, ghostNodeID)
	assert.True(t, ghostNodes[ghostNodeID])
}

func TestListGhostNodes_InsufficientReports(t *testing.T) {
	// Test when a node is reported as failed but not by enough nodes
	cluster := &redisv1alpha1.DistributedRedisCluster{
		Spec: redisv1alpha1.DistributedRedisClusterSpec{
			MasterSize:      3,
			ClusterReplicas: 1,
		},
	}

	// Only one node reports the ghost node (not enough for majority)
	ghostNodeID := "maybe-ghost-node"
	infos := &redisutil.ClusterInfos{
		Infos: map[string]*redisutil.NodeInfos{
			"node1": {
				Friends: []*redisutil.Node{
					{
						ID:         ghostNodeID,
						FailStatus: []string{redisutil.NodeStatusFail},
					},
				},
			},
		},
	}

	ghostNodes := listGhostNodes(cluster, infos, logr.Discard())

	// With MasterSize=3 and ClusterReplicas=1, we need (1+1)*3/2 = 3 reports
	// We only have 1 report, so it should NOT be detected as ghost
	assert.NotContains(t, ghostNodes, ghostNodeID)
}

// Helper function to create cluster infos with ghost nodes for testing
func createClusterInfosWithGhostNodes() *redisutil.ClusterInfos {
	ghostNodeID := "ghost-node-123"
	return &redisutil.ClusterInfos{
		Infos: map[string]*redisutil.NodeInfos{
			"node1": {
				Friends: []*redisutil.Node{
					{
						ID:         ghostNodeID,
						FailStatus: []string{redisutil.NodeStatusFail},
					},
				},
			},
			"node2": {
				Friends: []*redisutil.Node{
					{
						ID:         ghostNodeID,
						FailStatus: []string{redisutil.NodeStatusNoAddr},
					},
				},
			},
			"node3": {
				Friends: []*redisutil.Node{
					{
						ID:         ghostNodeID,
						FailStatus: []string{redisutil.NodeStatusFail},
					},
				},
			},
		},
	}
}
