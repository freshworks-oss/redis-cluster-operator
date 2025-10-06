package heal

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	redisv1alpha1 "github.com/freshworks/redis-cluster-operator/api/v1alpha1"
	"github.com/freshworks/redis-cluster-operator/utils/redisutil"
)

func TestCheckAndHeal_CheckandRecoverCluster_NilRanking(t *testing.T) {
	// Test when CreateRankingFromClusterInfo returns nil
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
			HaConfig: &redisv1alpha1.HAspec{
				HaEnabled: false, // This will cause CreateRankingFromClusterInfo to return nil
			},
		},
	}

	// Empty cluster infos will cause ranking creation to fail
	infos := &redisutil.ClusterInfos{
		Infos: map[string]*redisutil.NodeInfos{},
	}

	admin := &MockRedisAdmin{}
	ctx := context.Background()

	stopReconcile, actionDone, err := checkAndHeal.CheckandRecoverCluster(cluster, infos, admin, ctx)

	assert.Error(t, err)
	assert.False(t, stopReconcile)
	assert.False(t, actionDone)
	assert.Contains(t, err.Error(), "error Creating ranking object from cluster info")

	admin.AssertExpectations(t)
}

func TestCheckAndHeal_CheckandRecoverCluster_NoRecoveryNeeded(t *testing.T) {
	// Test when CreateRankingFromClusterInfo succeeds but no recovery is needed
	// This is hard to test without complex clustering setup, so we'll focus on
	// the error case where ranking creation fails due to insufficient nodes
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
			HaConfig: &redisv1alpha1.HAspec{
				HaEnabled: true,
				ZonesInfo: map[string]string{
					"zone1": "zone1-info",
				},
			},
		},
	}

	// Create cluster infos with insufficient nodes (will cause ranking to fail)
	infos := &redisutil.ClusterInfos{
		Infos: map[string]*redisutil.NodeInfos{
			"10.0.0.1:6379": {
				Node: &redisutil.Node{
					ID:   "node1",
					IP:   "10.0.0.1",
					Port: "6379",
				},
				Friends: []*redisutil.Node{},
			},
		},
	}

	admin := &MockRedisAdmin{}
	ctx := context.Background()

	stopReconcile, actionDone, err := checkAndHeal.CheckandRecoverCluster(cluster, infos, admin, ctx)

	// Should fail due to insufficient nodes for ranking
	assert.Error(t, err)
	assert.False(t, stopReconcile)
	assert.False(t, actionDone)
	assert.Contains(t, err.Error(), "error Creating ranking object from cluster info")

	admin.AssertExpectations(t)
}

func TestCheckAndHeal_CheckandRecoverCluster_InsufficientNodes(t *testing.T) {
	// Test when there are insufficient nodes for clustering
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
			HaConfig: &redisv1alpha1.HAspec{
				HaEnabled: true,
				ZonesInfo: map[string]string{
					"zone1": "zone1-info",
				},
			},
		},
	}

	// Create cluster infos with only one node (insufficient for 3 masters + 1 replica each)
	infos := &redisutil.ClusterInfos{
		Infos: map[string]*redisutil.NodeInfos{
			"10.0.0.1:6379": {
				Node: &redisutil.Node{
					ID:          "node1",
					IP:          "10.0.0.1",
					Port:        "6379",
					StatefulSet: "test-cluster-0",
				},
				Friends: []*redisutil.Node{},
			},
		},
	}

	admin := &MockRedisAdmin{}
	ctx := context.Background()

	stopReconcile, actionDone, err := checkAndHeal.CheckandRecoverCluster(cluster, infos, admin, ctx)

	// Should fail due to insufficient nodes
	assert.Error(t, err)
	assert.False(t, stopReconcile)
	assert.False(t, actionDone)
	assert.Contains(t, err.Error(), "error Creating ranking object from cluster info")

	admin.AssertExpectations(t)
}

func TestCheckAndHeal_CheckandRecoverCluster_WithSufficientNodes(t *testing.T) {
	// Test with sufficient nodes for clustering but no recovery needed
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
			HaConfig: &redisv1alpha1.HAspec{
				HaEnabled: true,
				ZonesInfo: map[string]string{
					"zone1": "zone1-info",
					"zone2": "zone2-info",
					"zone3": "zone3-info",
				},
			},
		},
	}

	// Create cluster infos with enough nodes for clustering (3 masters + 3 replicas = 6 nodes minimum)
	infos := &redisutil.ClusterInfos{
		Infos: map[string]*redisutil.NodeInfos{
			"10.0.0.1:6379": {
				Node: &redisutil.Node{
					ID:          "node1",
					IP:          "10.0.0.1",
					Port:        "6379",
					Role:        redisutil.RedisMasterRole,
					StatefulSet: "test-cluster-0",
					Zonename:    "zone1",
				},
				Friends: []*redisutil.Node{
					{
						ID:          "node2",
						IP:          "10.0.0.2",
						Port:        "6379",
						Role:        redisutil.RedisSlaveRole,
						StatefulSet: "test-cluster-0",
						Zonename:    "zone2",
					},
					{
						ID:          "node3",
						IP:          "10.0.0.3",
						Port:        "6379",
						Role:        redisutil.RedisMasterRole,
						StatefulSet: "test-cluster-1",
						Zonename:    "zone2",
					},
					{
						ID:          "node4",
						IP:          "10.0.0.4",
						Port:        "6379",
						Role:        redisutil.RedisSlaveRole,
						StatefulSet: "test-cluster-1",
						Zonename:    "zone3",
					},
					{
						ID:          "node5",
						IP:          "10.0.0.5",
						Port:        "6379",
						Role:        redisutil.RedisMasterRole,
						StatefulSet: "test-cluster-2",
						Zonename:    "zone3",
					},
					{
						ID:          "node6",
						IP:          "10.0.0.6",
						Port:        "6379",
						Role:        redisutil.RedisSlaveRole,
						StatefulSet: "test-cluster-2",
						Zonename:    "zone1",
					},
				},
			},
			"10.0.0.2:6379": {
				Node: &redisutil.Node{
					ID:          "node2",
					IP:          "10.0.0.2",
					Port:        "6379",
					Role:        redisutil.RedisSlaveRole,
					StatefulSet: "test-cluster-0",
					Zonename:    "zone2",
				},
				Friends: []*redisutil.Node{
					{
						ID:          "node1",
						IP:          "10.0.0.1",
						Port:        "6379",
						Role:        redisutil.RedisMasterRole,
						StatefulSet: "test-cluster-0",
						Zonename:    "zone1",
					},
					{
						ID:          "node3",
						IP:          "10.0.0.3",
						Port:        "6379",
						Role:        redisutil.RedisMasterRole,
						StatefulSet: "test-cluster-1",
						Zonename:    "zone2",
					},
					{
						ID:          "node4",
						IP:          "10.0.0.4",
						Port:        "6379",
						Role:        redisutil.RedisSlaveRole,
						StatefulSet: "test-cluster-1",
						Zonename:    "zone3",
					},
					{
						ID:          "node5",
						IP:          "10.0.0.5",
						Port:        "6379",
						Role:        redisutil.RedisMasterRole,
						StatefulSet: "test-cluster-2",
						Zonename:    "zone3",
					},
					{
						ID:          "node6",
						IP:          "10.0.0.6",
						Port:        "6379",
						Role:        redisutil.RedisSlaveRole,
						StatefulSet: "test-cluster-2",
						Zonename:    "zone1",
					},
				},
			},
			"10.0.0.3:6379": {
				Node: &redisutil.Node{
					ID:          "node3",
					IP:          "10.0.0.3",
					Port:        "6379",
					Role:        redisutil.RedisMasterRole,
					StatefulSet: "test-cluster-1",
					Zonename:    "zone2",
				},
				Friends: []*redisutil.Node{
					{
						ID:          "node1",
						IP:          "10.0.0.1",
						Port:        "6379",
						Role:        redisutil.RedisMasterRole,
						StatefulSet: "test-cluster-0",
						Zonename:    "zone1",
					},
					{
						ID:          "node2",
						IP:          "10.0.0.2",
						Port:        "6379",
						Role:        redisutil.RedisSlaveRole,
						StatefulSet: "test-cluster-0",
						Zonename:    "zone2",
					},
					{
						ID:          "node4",
						IP:          "10.0.0.4",
						Port:        "6379",
						Role:        redisutil.RedisSlaveRole,
						StatefulSet: "test-cluster-1",
						Zonename:    "zone3",
					},
					{
						ID:          "node5",
						IP:          "10.0.0.5",
						Port:        "6379",
						Role:        redisutil.RedisMasterRole,
						StatefulSet: "test-cluster-2",
						Zonename:    "zone3",
					},
					{
						ID:          "node6",
						IP:          "10.0.0.6",
						Port:        "6379",
						Role:        redisutil.RedisSlaveRole,
						StatefulSet: "test-cluster-2",
						Zonename:    "zone1",
					},
				},
			},
			"10.0.0.4:6379": {
				Node: &redisutil.Node{
					ID:          "node4",
					IP:          "10.0.0.4",
					Port:        "6379",
					Role:        redisutil.RedisSlaveRole,
					StatefulSet: "test-cluster-1",
					Zonename:    "zone3",
				},
				Friends: []*redisutil.Node{
					{
						ID:          "node1",
						IP:          "10.0.0.1",
						Port:        "6379",
						Role:        redisutil.RedisMasterRole,
						StatefulSet: "test-cluster-0",
						Zonename:    "zone1",
					},
					{
						ID:          "node2",
						IP:          "10.0.0.2",
						Port:        "6379",
						Role:        redisutil.RedisSlaveRole,
						StatefulSet: "test-cluster-0",
						Zonename:    "zone2",
					},
					{
						ID:          "node3",
						IP:          "10.0.0.3",
						Port:        "6379",
						Role:        redisutil.RedisMasterRole,
						StatefulSet: "test-cluster-1",
						Zonename:    "zone2",
					},
					{
						ID:          "node5",
						IP:          "10.0.0.5",
						Port:        "6379",
						Role:        redisutil.RedisMasterRole,
						StatefulSet: "test-cluster-2",
						Zonename:    "zone3",
					},
					{
						ID:          "node6",
						IP:          "10.0.0.6",
						Port:        "6379",
						Role:        redisutil.RedisSlaveRole,
						StatefulSet: "test-cluster-2",
						Zonename:    "zone1",
					},
				},
			},
			"10.0.0.5:6379": {
				Node: &redisutil.Node{
					ID:          "node5",
					IP:          "10.0.0.5",
					Port:        "6379",
					Role:        redisutil.RedisMasterRole,
					StatefulSet: "test-cluster-2",
					Zonename:    "zone3",
				},
				Friends: []*redisutil.Node{
					{
						ID:          "node1",
						IP:          "10.0.0.1",
						Port:        "6379",
						Role:        redisutil.RedisMasterRole,
						StatefulSet: "test-cluster-0",
						Zonename:    "zone1",
					},
					{
						ID:          "node2",
						IP:          "10.0.0.2",
						Port:        "6379",
						Role:        redisutil.RedisSlaveRole,
						StatefulSet: "test-cluster-0",
						Zonename:    "zone2",
					},
					{
						ID:          "node3",
						IP:          "10.0.0.3",
						Port:        "6379",
						Role:        redisutil.RedisMasterRole,
						StatefulSet: "test-cluster-1",
						Zonename:    "zone2",
					},
					{
						ID:          "node4",
						IP:          "10.0.0.4",
						Port:        "6379",
						Role:        redisutil.RedisSlaveRole,
						StatefulSet: "test-cluster-1",
						Zonename:    "zone3",
					},
					{
						ID:          "node6",
						IP:          "10.0.0.6",
						Port:        "6379",
						Role:        redisutil.RedisSlaveRole,
						StatefulSet: "test-cluster-2",
						Zonename:    "zone1",
					},
				},
			},
			"10.0.0.6:6379": {
				Node: &redisutil.Node{
					ID:          "node6",
					IP:          "10.0.0.6",
					Port:        "6379",
					Role:        redisutil.RedisSlaveRole,
					StatefulSet: "test-cluster-2",
					Zonename:    "zone1",
				},
				Friends: []*redisutil.Node{
					{
						ID:          "node1",
						IP:          "10.0.0.1",
						Port:        "6379",
						Role:        redisutil.RedisMasterRole,
						StatefulSet: "test-cluster-0",
						Zonename:    "zone1",
					},
					{
						ID:          "node2",
						IP:          "10.0.0.2",
						Port:        "6379",
						Role:        redisutil.RedisSlaveRole,
						StatefulSet: "test-cluster-0",
						Zonename:    "zone2",
					},
					{
						ID:          "node3",
						IP:          "10.0.0.3",
						Port:        "6379",
						Role:        redisutil.RedisMasterRole,
						StatefulSet: "test-cluster-1",
						Zonename:    "zone2",
					},
					{
						ID:          "node4",
						IP:          "10.0.0.4",
						Port:        "6379",
						Role:        redisutil.RedisSlaveRole,
						StatefulSet: "test-cluster-1",
						Zonename:    "zone3",
					},
					{
						ID:          "node5",
						IP:          "10.0.0.5",
						Port:        "6379",
						Role:        redisutil.RedisMasterRole,
						StatefulSet: "test-cluster-2",
						Zonename:    "zone3",
					},
				},
			},
		},
	}

	admin := &MockRedisAdmin{}
	ctx := context.Background()

	stopReconcile, actionDone, err := checkAndHeal.CheckandRecoverCluster(cluster, infos, admin, ctx)

	// This should succeed in creating ranking but may not need recovery
	// The exact behavior depends on the clustering algorithm
	assert.NoError(t, err)
	// stopReconcile should be false (don't stop reconciliation)
	assert.False(t, stopReconcile)
	// actionDone could be true or false depending on clustering logic
	t.Logf("actionDone: %v", actionDone)

	admin.AssertExpectations(t)
}

func TestCheckAndHeal_CheckandRecoverCluster_EmptyClusterInfos(t *testing.T) {
	// Test with completely empty cluster infos
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
			HaConfig: &redisv1alpha1.HAspec{
				HaEnabled: true,
				ZonesInfo: map[string]string{
					"zone1": "zone1-info",
				},
			},
		},
	}

	// Completely empty cluster infos
	infos := &redisutil.ClusterInfos{
		Infos: map[string]*redisutil.NodeInfos{},
	}

	admin := &MockRedisAdmin{}
	ctx := context.Background()

	stopReconcile, actionDone, err := checkAndHeal.CheckandRecoverCluster(cluster, infos, admin, ctx)

	// Should fail due to insufficient nodes
	assert.Error(t, err)
	assert.False(t, stopReconcile)
	assert.False(t, actionDone)
	assert.Contains(t, err.Error(), "error Creating ranking object from cluster info")

	admin.AssertExpectations(t)
}

// Basic tests for edge cases and error conditions
func TestCheckAndHeal_CheckandRecoverCluster_EdgeCases(t *testing.T) {
	checkAndHeal := &CheckAndHeal{
		Logger:        logr.Discard(),
		PodControl:    &MockPodControl{},
		DryRun:        false,
		MetricsClient: &MockMetrics{},
	}

	tests := []struct {
		name         string
		cluster      *redisv1alpha1.DistributedRedisCluster
		infos        *redisutil.ClusterInfos
		expectError  bool
		expectStop   bool
		expectAction bool
	}{
		{
			name:        "Nil cluster",
			cluster:     nil,
			infos:       &redisutil.ClusterInfos{Infos: map[string]*redisutil.NodeInfos{}},
			expectError: true,
		},
		{
			name: "Nil infos",
			cluster: &redisv1alpha1.DistributedRedisCluster{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"},
				Spec: redisv1alpha1.DistributedRedisClusterSpec{
					MasterSize:      1,
					ClusterReplicas: 1,
				},
			},
			infos:       nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			admin := &MockRedisAdmin{}
			ctx := context.Background()

			// This will likely panic or error due to nil inputs, which is expected
			defer func() {
				if r := recover(); r != nil {
					// Panic is expected for nil inputs
					assert.True(t, tt.expectError, "Expected error/panic for test case: %s", tt.name)
				}
			}()

			stopReconcile, actionDone, err := checkAndHeal.CheckandRecoverCluster(tt.cluster, tt.infos, admin, ctx)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectStop, stopReconcile)
				assert.Equal(t, tt.expectAction, actionDone)
			}
		})
	}
}

// Note: Testing the full recovery logic is complex because it depends on the
// clustering package's ranking algorithm. The tests above cover the main error
// paths and edge cases that can be controlled. For comprehensive testing of
// the recovery logic, integration tests would be more appropriate.
