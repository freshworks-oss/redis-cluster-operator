package heal

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	redisv1alpha1 "github.com/dinesh-murugiah/rediscluster-operator/api/v1alpha1"
	"github.com/dinesh-murugiah/rediscluster-operator/utils/redisutil"
)

func TestCheckAndHeal_FixUntrustedNodes_NoUntrustedNodes(t *testing.T) {
	// Test when there are no untrusted nodes
	checkAndHeal := &CheckAndHeal{
		Logger:        logr.Discard(),
		PodControl:    &MockPodControl{},
		Pods:          []*corev1.Pod{},
		DryRun:        false,
		MetricsClient: &MockMetrics{},
	}

	cluster := &redisv1alpha1.DistributedRedisCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster", Namespace: "test-ns"},
	}

	// Empty cluster infos - no untrusted nodes
	infos := &redisutil.ClusterInfos{
		Infos: map[string]*redisutil.NodeInfos{},
	}

	admin := &MockRedisAdmin{}
	ctx := context.Background()

	actionDone, err := checkAndHeal.FixUntrustedNodes(cluster, infos, admin, ctx)

	assert.NoError(t, err)
	assert.False(t, actionDone)

	admin.AssertExpectations(t)
}

func TestCheckAndHeal_FixUntrustedNodes_DryRun(t *testing.T) {
	// Test when DryRun is enabled - should detect untrusted nodes but not forget them
	pods := []*corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "untrusted-pod", Namespace: "test-ns"},
			Status:     corev1.PodStatus{PodIP: "10.0.0.1"},
		},
	}

	checkAndHeal := &CheckAndHeal{
		Logger:        logr.Discard(),
		PodControl:    &MockPodControl{},
		Pods:          pods,
		DryRun:        true, // DryRun enabled
		MetricsClient: &MockMetrics{},
	}

	cluster := &redisv1alpha1.DistributedRedisCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster", Namespace: "test-ns"},
	}

	// Create cluster infos with untrusted nodes
	infos := createClusterInfosWithUntrustedNodes()

	admin := &MockRedisAdmin{}
	ctx := context.Background()

	actionDone, err := checkAndHeal.FixUntrustedNodes(cluster, infos, admin, ctx)

	assert.NoError(t, err)
	assert.True(t, actionDone) // Action was detected but not performed due to DryRun

	// ForgetNode should not be called in DryRun mode
	admin.AssertNotCalled(t, "ForgetNode")
}

func TestCheckAndHeal_FixUntrustedNodes_Success(t *testing.T) {
	// Test successful fixing of untrusted nodes
	pods := []*corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "untrusted-pod", Namespace: "test-ns"},
			Status:     corev1.PodStatus{PodIP: "10.0.0.999"}, // Different IP so pod is not reused
		},
	}

	deleteCalled := false
	podControl := &MockPodControl{
		DeletePodByNameFunc: func(namespace, name string) error {
			deleteCalled = true
			assert.Equal(t, "test-ns", namespace)
			assert.Equal(t, "untrusted-pod", name)
			return nil
		},
	}

	checkAndHeal := &CheckAndHeal{
		Logger:        logr.Discard(),
		PodControl:    podControl,
		Pods:          pods,
		DryRun:        false,
		MetricsClient: &MockMetrics{},
	}

	cluster := &redisv1alpha1.DistributedRedisCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster", Namespace: "test-ns"},
	}

	// Create cluster infos with untrusted nodes
	infos := createClusterInfosWithUntrustedNodes()

	admin := &MockRedisAdmin{}
	ctx := context.Background()

	// Mock ForgetNode to succeed
	admin.On("ForgetNode", ctx, mock.AnythingOfType("string"), "test-ns", "test-cluster").Return(nil)

	actionDone, err := checkAndHeal.FixUntrustedNodes(cluster, infos, admin, ctx)

	assert.NoError(t, err)
	assert.True(t, actionDone)
	assert.True(t, deleteCalled)

	admin.AssertExpectations(t)
}

func TestCheckAndHeal_FixUntrustedNodes_PodReused(t *testing.T) {
	// Test when pod is reused by another node - should not delete pod
	pods := []*corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "reused-pod", Namespace: "test-ns"},
			Status:     corev1.PodStatus{PodIP: "10.0.0.1"}, // Same IP as untrusted node
		},
	}

	deleteCalled := false
	podControl := &MockPodControl{
		DeletePodByNameFunc: func(namespace, name string) error {
			deleteCalled = true
			return nil
		},
	}

	checkAndHeal := &CheckAndHeal{
		Logger:        logr.Discard(),
		PodControl:    podControl,
		Pods:          pods,
		DryRun:        false,
		MetricsClient: &MockMetrics{},
	}

	cluster := &redisv1alpha1.DistributedRedisCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster", Namespace: "test-ns"},
	}

	// Create cluster infos with untrusted node that has same IP as existing pod
	infos := &redisutil.ClusterInfos{
		Infos: map[string]*redisutil.NodeInfos{
			"node1": {
				Node: &redisutil.Node{
					ID:   "node1",
					IP:   "10.0.0.2",
					Port: "6379",
				},
				Friends: []*redisutil.Node{
					{
						ID:         "untrusted-node-123",
						IP:         "10.0.0.1", // Same IP as pod
						PodName:    "reused-pod",
						FailStatus: []string{redisutil.NodeStatusHandshake},
					},
				},
			},
		},
	}

	admin := &MockRedisAdmin{}
	ctx := context.Background()

	// Mock ForgetNode to succeed
	admin.On("ForgetNode", ctx, mock.AnythingOfType("string"), "test-ns", "test-cluster").Return(nil)

	actionDone, err := checkAndHeal.FixUntrustedNodes(cluster, infos, admin, ctx)

	assert.NoError(t, err)
	assert.True(t, actionDone)
	assert.False(t, deleteCalled) // Pod should not be deleted because it's reused

	admin.AssertExpectations(t)
}

func TestCheckAndHeal_FixUntrustedNodes_NodeIPConflict(t *testing.T) {
	// Test when there's another node with same IP but different ID
	checkAndHeal := &CheckAndHeal{
		Logger:        logr.Discard(),
		PodControl:    &MockPodControl{},
		Pods:          []*corev1.Pod{},
		DryRun:        false,
		MetricsClient: &MockMetrics{},
	}

	cluster := &redisv1alpha1.DistributedRedisCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster", Namespace: "test-ns"},
	}

	// Create cluster infos with IP conflict
	infos := &redisutil.ClusterInfos{
		Infos: map[string]*redisutil.NodeInfos{
			"node1": {
				Node: &redisutil.Node{
					ID:   "node1",
					IP:   "10.0.0.1",
					Port: "6379",
				},
				Friends: []*redisutil.Node{
					{
						ID:         "untrusted-node-123",
						IP:         "10.0.0.1", // Same IP as node1
						PodName:    "untrusted-pod",
						FailStatus: []string{redisutil.NodeStatusHandshake},
					},
				},
			},
		},
	}

	admin := &MockRedisAdmin{}
	ctx := context.Background()

	actionDone, err := checkAndHeal.FixUntrustedNodes(cluster, infos, admin, ctx)

	assert.NoError(t, err)
	assert.False(t, actionDone) // No action done because IP conflict causes continue

	// ForgetNode should not be called because continue statement is hit
	admin.AssertNotCalled(t, "ForgetNode")
}

func TestCheckAndHeal_FixUntrustedNodes_ForgetNodeError(t *testing.T) {
	// Test when ForgetNode returns an error
	checkAndHeal := &CheckAndHeal{
		Logger:        logr.Discard(),
		PodControl:    &MockPodControl{},
		Pods:          []*corev1.Pod{},
		DryRun:        false,
		MetricsClient: &MockMetrics{},
	}

	cluster := &redisv1alpha1.DistributedRedisCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster", Namespace: "test-ns"},
	}

	infos := createClusterInfosWithUntrustedNodes()

	admin := &MockRedisAdmin{}
	ctx := context.Background()

	expectedError := errors.New("failed to forget node")
	admin.On("ForgetNode", ctx, mock.AnythingOfType("string"), "test-ns", "test-cluster").Return(expectedError)

	actionDone, err := checkAndHeal.FixUntrustedNodes(cluster, infos, admin, ctx)

	assert.Error(t, err)
	assert.True(t, actionDone)
	assert.Contains(t, err.Error(), "failed to forget node")

	admin.AssertExpectations(t)
}

func TestListUntrustedNodes_NilInfos(t *testing.T) {
	// Test listUntrustedNodes with nil cluster infos
	untrustedNodes := listUntrustedNodes(nil)
	assert.Empty(t, untrustedNodes)
}

func TestListUntrustedNodes_EmptyInfos(t *testing.T) {
	// Test listUntrustedNodes with empty cluster infos
	infos := &redisutil.ClusterInfos{
		Infos: nil,
	}

	untrustedNodes := listUntrustedNodes(infos)
	assert.Empty(t, untrustedNodes)
}

func TestListUntrustedNodes_NoUntrustedNodes(t *testing.T) {
	// Test listUntrustedNodes when all nodes are trusted
	infos := &redisutil.ClusterInfos{
		Infos: map[string]*redisutil.NodeInfos{
			"node1": {
				Friends: []*redisutil.Node{
					{
						ID:         "trusted-node-1",
						FailStatus: []string{}, // No handshake status
					},
					{
						ID:         "trusted-node-2",
						FailStatus: []string{redisutil.NodeStatusFail}, // Different status
					},
				},
			},
		},
	}

	untrustedNodes := listUntrustedNodes(infos)
	assert.Empty(t, untrustedNodes)
}

func TestListUntrustedNodes_WithUntrustedNodes(t *testing.T) {
	// Test listUntrustedNodes when there are actual untrusted nodes
	untrustedNodeID := "untrusted-node-123"
	infos := &redisutil.ClusterInfos{
		Infos: map[string]*redisutil.NodeInfos{
			"node1": {
				Friends: []*redisutil.Node{
					{
						ID:         untrustedNodeID,
						IP:         "10.0.0.1",
						PodName:    "untrusted-pod",
						FailStatus: []string{redisutil.NodeStatusHandshake},
					},
				},
			},
		},
	}

	untrustedNodes := listUntrustedNodes(infos)

	assert.Len(t, untrustedNodes, 1)
	assert.Contains(t, untrustedNodes, untrustedNodeID)
	assert.Equal(t, "10.0.0.1", untrustedNodes[untrustedNodeID].IP)
	assert.Equal(t, "untrusted-pod", untrustedNodes[untrustedNodeID].PodName)
}

func TestListUntrustedNodes_DuplicateNodes(t *testing.T) {
	// Test listUntrustedNodes with same untrusted node reported by multiple nodes
	untrustedNodeID := "untrusted-node-123"
	infos := &redisutil.ClusterInfos{
		Infos: map[string]*redisutil.NodeInfos{
			"node1": {
				Friends: []*redisutil.Node{
					{
						ID:         untrustedNodeID,
						IP:         "10.0.0.1",
						FailStatus: []string{redisutil.NodeStatusHandshake},
					},
				},
			},
			"node2": {
				Friends: []*redisutil.Node{
					{
						ID:         untrustedNodeID, // Same ID
						IP:         "10.0.0.1",
						FailStatus: []string{redisutil.NodeStatusHandshake},
					},
				},
			},
		},
	}

	untrustedNodes := listUntrustedNodes(infos)

	// Should only have one entry despite being reported by multiple nodes
	assert.Len(t, untrustedNodes, 1)
	assert.Contains(t, untrustedNodes, untrustedNodeID)
}

func TestCheckIfPodNameExistAndIsReused_EmptyPodName(t *testing.T) {
	// Test checkIfPodNameExistAndIsReused with empty pod name
	node := &redisutil.Node{
		ID:      "node1",
		PodName: "", // Empty pod name
	}

	pods := []*corev1.Pod{
		{ObjectMeta: metav1.ObjectMeta{Name: "some-pod"}},
	}

	exist, reused := checkIfPodNameExistAndIsReused(node, pods)

	assert.False(t, exist)
	assert.False(t, reused)
}

func TestCheckIfPodNameExistAndIsReused_PodNotFound(t *testing.T) {
	// Test checkIfPodNameExistAndIsReused when pod doesn't exist
	node := &redisutil.Node{
		ID:      "node1",
		PodName: "missing-pod",
		IP:      "10.0.0.1",
	}

	pods := []*corev1.Pod{
		{ObjectMeta: metav1.ObjectMeta{Name: "other-pod"}},
	}

	exist, reused := checkIfPodNameExistAndIsReused(node, pods)

	assert.False(t, exist)
	assert.False(t, reused)
}

func TestCheckIfPodNameExistAndIsReused_PodExistsAndReused(t *testing.T) {
	// Test checkIfPodNameExistAndIsReused when pod exists and is reused (same IP)
	node := &redisutil.Node{
		ID:      "node1",
		PodName: "test-pod",
		IP:      "10.0.0.1",
	}

	pods := []*corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "test-pod"},
			Status:     corev1.PodStatus{PodIP: "10.0.0.1"}, // Same IP
		},
	}

	exist, reused := checkIfPodNameExistAndIsReused(node, pods)

	assert.True(t, exist)
	assert.True(t, reused)
}

func TestCheckIfPodNameExistAndIsReused_PodExistsButNotReused(t *testing.T) {
	// Test checkIfPodNameExistAndIsReused when pod exists but not reused (different IP)
	node := &redisutil.Node{
		ID:      "node1",
		PodName: "test-pod",
		IP:      "10.0.0.1",
	}

	pods := []*corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "test-pod"},
			Status:     corev1.PodStatus{PodIP: "10.0.0.2"}, // Different IP
		},
	}

	exist, reused := checkIfPodNameExistAndIsReused(node, pods)

	assert.True(t, exist)
	assert.False(t, reused)
}

func TestCheckIfPodNameExistAndIsReused_MultiplePods(t *testing.T) {
	// Test checkIfPodNameExistAndIsReused with multiple pods
	node := &redisutil.Node{
		ID:      "node1",
		PodName: "target-pod",
		IP:      "10.0.0.1",
	}

	pods := []*corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "other-pod"},
			Status:     corev1.PodStatus{PodIP: "10.0.0.2"},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "target-pod"},
			Status:     corev1.PodStatus{PodIP: "10.0.0.1"}, // Matching pod
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "another-pod"},
			Status:     corev1.PodStatus{PodIP: "10.0.0.3"},
		},
	}

	exist, reused := checkIfPodNameExistAndIsReused(node, pods)

	assert.True(t, exist)
	assert.True(t, reused)
}

// Helper function to create cluster infos with untrusted nodes for testing
func createClusterInfosWithUntrustedNodes() *redisutil.ClusterInfos {
	return &redisutil.ClusterInfos{
		Infos: map[string]*redisutil.NodeInfos{
			"node1": {
				Node: &redisutil.Node{
					ID:   "node1",
					IP:   "10.0.0.2",
					Port: "6379",
				},
				Friends: []*redisutil.Node{
					{
						ID:         "untrusted-node-123",
						IP:         "10.0.0.1",
						PodName:    "untrusted-pod",
						FailStatus: []string{redisutil.NodeStatusHandshake},
					},
				},
			},
		},
	}
}

// Additional edge case tests for better coverage

func TestCheckAndHeal_FixUntrustedNodes_EmptyPodName(t *testing.T) {
	// Test when untrusted node has empty PodName
	checkAndHeal := &CheckAndHeal{
		Logger:        logr.Discard(),
		PodControl:    &MockPodControl{},
		DryRun:        false,
		MetricsClient: &MockMetrics{},
		Pods:          []*corev1.Pod{},
	}

	cluster := &redisv1alpha1.DistributedRedisCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster", Namespace: "test-ns"},
	}

	infos := &redisutil.ClusterInfos{
		Infos: map[string]*redisutil.NodeInfos{
			"192.168.1.1:6379": {
				Node: &redisutil.Node{IP: "192.168.1.1", Port: "6379"},
				Friends: []*redisutil.Node{
					{
						ID:         "untrusted-node-1",
						IP:         "192.168.1.2",
						Port:       "6379",
						PodName:    "", // Empty PodName
						FailStatus: []string{redisutil.NodeStatusHandshake},
					},
				},
			},
		},
	}

	admin := &MockRedisAdmin{}
	ctx := context.Background()

	admin.On("ForgetNode", ctx, "untrusted-node-1", "test-ns", "test-cluster").Return(nil)

	actionDone, err := checkAndHeal.FixUntrustedNodes(cluster, infos, admin, ctx)

	assert.NoError(t, err)
	assert.True(t, actionDone)

	admin.AssertExpectations(t)
}

func TestCheckAndHeal_FixUntrustedNodes_DeletePodError(t *testing.T) {
	// Test when DeletePodByName returns an error
	deleteCalled := false
	checkAndHeal := &CheckAndHeal{
		Logger: logr.Discard(),
		PodControl: &MockPodControl{
			DeletePodByNameFunc: func(namespace, name string) error {
				deleteCalled = true
				return errors.New("failed to delete pod")
			},
		},
		DryRun:        false,
		MetricsClient: &MockMetrics{},
		Pods: []*corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "test-pod-1"},
				Status:     corev1.PodStatus{PodIP: "192.168.1.99"}, // Different IP to avoid reuse
			},
		},
	}

	cluster := &redisv1alpha1.DistributedRedisCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster", Namespace: "test-ns"},
	}

	infos := &redisutil.ClusterInfos{
		Infos: map[string]*redisutil.NodeInfos{
			"192.168.1.1:6379": {
				Node: &redisutil.Node{IP: "192.168.1.1", Port: "6379"},
				Friends: []*redisutil.Node{
					{
						ID:         "untrusted-node-1",
						IP:         "192.168.1.2",
						Port:       "6379",
						PodName:    "test-pod-1",
						FailStatus: []string{redisutil.NodeStatusHandshake},
					},
				},
			},
		},
	}

	admin := &MockRedisAdmin{}
	ctx := context.Background()

	admin.On("ForgetNode", ctx, "untrusted-node-1", "test-ns", "test-cluster").Return(nil)

	actionDone, err := checkAndHeal.FixUntrustedNodes(cluster, infos, admin, ctx)

	// Should return error due to pod deletion failure
	assert.Error(t, err)
	assert.True(t, actionDone)
	assert.True(t, deleteCalled)
	assert.Contains(t, err.Error(), "failed to delete pod")

	admin.AssertExpectations(t)
}

func TestListUntrustedNodes_NilInputs(t *testing.T) {
	// Test listUntrustedNodes with nil inputs
	result := listUntrustedNodes(nil)
	assert.NotNil(t, result)
	assert.Empty(t, result)

	// Test with ClusterInfos but nil Infos
	result = listUntrustedNodes(&redisutil.ClusterInfos{Infos: nil})
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestCheckIfPodNameExistAndIsReused_EdgeCases(t *testing.T) {
	// Test with empty PodName
	exist, reused := checkIfPodNameExistAndIsReused(&redisutil.Node{PodName: ""}, []*corev1.Pod{})
	assert.False(t, exist)
	assert.False(t, reused)

	// Test with nil pod list
	exist, reused = checkIfPodNameExistAndIsReused(&redisutil.Node{PodName: "test-pod"}, nil)
	assert.False(t, exist)
	assert.False(t, reused)

	// Test with empty pod list
	exist, reused = checkIfPodNameExistAndIsReused(&redisutil.Node{PodName: "test-pod"}, []*corev1.Pod{})
	assert.False(t, exist)
	assert.False(t, reused)
}
