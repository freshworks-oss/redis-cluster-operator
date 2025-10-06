package distributedrediscluster

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	redisv1alpha1 "github.com/freshworks/redis-cluster-operator/api/v1alpha1"
	config "github.com/freshworks/redis-cluster-operator/redisconfig"
	"github.com/freshworks/redis-cluster-operator/utils/redisutil"
	"github.com/freshworks/redis-cluster-operator/utils/testutil"
	"github.com/freshworks/redis-cluster-operator/utils/testutil/mocks"
)

// Local helper functions to avoid circular imports
func createTestRedisNode(id, ip, port, role string, slots []redisutil.Slot) *redisutil.Node {
	return &redisutil.Node{
		ID:    id,
		IP:    ip,
		Port:  port,
		Role:  role,
		Slots: slots,
	}
}

func createTestRedisNodeWithZone(id, ip, port, role, zone string, slots []redisutil.Slot) *redisutil.Node {
	return &redisutil.Node{
		ID:       id,
		IP:       ip,
		Port:     port,
		Role:     role,
		Slots:    slots,
		Zonename: zone,
	}
}

func createTestClusterInfos(nodes []*redisutil.Node) *redisutil.ClusterInfos {
	infos := make(map[string]*redisutil.NodeInfos)
	for _, node := range nodes {
		infos[node.ID] = &redisutil.NodeInfos{
			Node: node,
		}
	}
	return &redisutil.ClusterInfos{
		Infos:  infos,
		Status: redisutil.ClusterInfosConsistent,
	}
}

func createMasterRedisNode(id, ip, zone string, slotStart, slotEnd int) *redisutil.Node {
	slots := make([]redisutil.Slot, slotEnd-slotStart+1)
	for i := slotStart; i <= slotEnd; i++ {
		slots[i-slotStart] = redisutil.Slot(i)
	}
	return createTestRedisNodeWithZone(id, ip, "6379", redisutil.RedisMasterRole, zone, slots)
}

func TestGetLabels(t *testing.T) {
	cluster := testutil.CreateTestCluster("test-cluster", "test-ns")

	labels := getLabels(cluster)

	assert.NotNil(t, labels)
	assert.Equal(t, redisv1alpha1.OperatorName, labels[redisv1alpha1.LabelManagedByKey])
	assert.Equal(t, "distributedrediscluster-test-cluster", labels[redisv1alpha1.LabelClusterName])
}

func TestGetLabelsWithCustomLabels(t *testing.T) {
	cluster := testutil.CreateTestCluster("test-cluster", "test-ns")
	cluster.Labels = map[string]string{
		"custom-label": "custom-value",
		"env":          "test",
	}

	labels := getLabels(cluster)

	assert.NotNil(t, labels)
	assert.Equal(t, redisv1alpha1.OperatorName, labels[redisv1alpha1.LabelManagedByKey])
	assert.Equal(t, "distributedrediscluster-test-cluster", labels[redisv1alpha1.LabelClusterName])
	assert.Equal(t, "custom-value", labels["custom-label"])
	assert.Equal(t, "test", labels["env"])
}

func TestGetpodFromlist(t *testing.T) {
	pods := []*corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "pod1"},
			Status:     corev1.PodStatus{PodIP: "10.0.0.1"},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "pod2"},
			Status:     corev1.PodStatus{PodIP: "10.0.0.2"},
		},
	}

	// Test finding existing pod
	result := getpodFromlist(pods, "10.0.0.1")
	assert.NotNil(t, result)
	assert.Equal(t, "pod1", result.Name)

	// Test not finding pod
	result = getpodFromlist(pods, "10.0.0.3")
	assert.Nil(t, result)

	// Test empty list
	result = getpodFromlist([]*corev1.Pod{}, "10.0.0.1")
	assert.Nil(t, result)
}

func TestClusterPods(t *testing.T) {
	pods := []corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "running-pod"},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "pending-pod"},
			Status:     corev1.PodStatus{Phase: corev1.PodPending},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "failed-pod"},
			Status:     corev1.PodStatus{Phase: corev1.PodFailed},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "succeeded-pod"},
			Status:     corev1.PodStatus{Phase: corev1.PodSucceeded},
		},
	}

	result := clusterPods(pods)

	// Should only return running pods (not succeeded)
	assert.Len(t, result, 1)

	names := make([]string, len(result))
	for i, pod := range result {
		names[i] = pod.Name
	}

	assert.Contains(t, names, "running-pod")
	assert.NotContains(t, names, "succeeded-pod")
	assert.NotContains(t, names, "pending-pod")
	assert.NotContains(t, names, "failed-pod")
}

func TestVerifyIfSingleAzDeployment(t *testing.T) {
	tests := []struct {
		name     string
		cluster  *redisv1alpha1.DistributedRedisCluster
		expected bool
	}{
		{
			name: "Single AZ deployment",
			cluster: testutil.CreateTestClusterWithHAConfig("test-cluster", "test-ns", true, map[string]string{
				"us-west-2a": "RedisMasterSingleAz",
			}),
			expected: true,
		},
		{
			name:     "Multi AZ deployment",
			cluster:  testutil.CreateTestCluster("test-cluster", "test-ns"),
			expected: false,
		},
		{
			name: "No HA config",
			cluster: func() *redisv1alpha1.DistributedRedisCluster {
				cluster := testutil.CreateTestCluster("test-cluster", "test-ns")
				cluster.Spec.HaConfig = nil
				return cluster
			}(),
			expected: false,
		},
		{
			name: "HA disabled",
			cluster: testutil.CreateTestClusterWithHAConfig("test-cluster", "test-ns", false, map[string]string{
				"us-west-2a": "zone1",
			}),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := verifyIfSingleAzDeployment(tt.cluster)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNeedClusterOperation(t *testing.T) {
	tests := []struct {
		name     string
		cluster  *redisv1alpha1.DistributedRedisCluster
		expected bool
	}{
		{
			name: "Master size mismatch",
			cluster: func() *redisv1alpha1.DistributedRedisCluster {
				cluster := testutil.CreateTestCluster("test-cluster", "test-ns")
				cluster.Status.NumberOfMaster = 2 // Expected is 3
				cluster.Status.MinReplicationFactor = 1
				cluster.Status.MaxReplicationFactor = 1
				return cluster
			}(),
			expected: true,
		},
		{
			name: "Min replication factor mismatch",
			cluster: func() *redisv1alpha1.DistributedRedisCluster {
				cluster := testutil.CreateTestCluster("test-cluster", "test-ns")
				cluster.Spec.ClusterReplicas = 2
				cluster.Status.NumberOfMaster = 3
				cluster.Status.MinReplicationFactor = 1 // Expected is 2
				cluster.Status.MaxReplicationFactor = 1
				return cluster
			}(),
			expected: true,
		},
		{
			name: "Max replication factor mismatch",
			cluster: func() *redisv1alpha1.DistributedRedisCluster {
				cluster := testutil.CreateTestCluster("test-cluster", "test-ns")
				cluster.Status.NumberOfMaster = 3
				cluster.Status.MinReplicationFactor = 1
				cluster.Status.MaxReplicationFactor = 2 // Expected is 1
				return cluster
			}(),
			expected: true,
		},
		{
			name: "No cluster operation needed",
			cluster: func() *redisv1alpha1.DistributedRedisCluster {
				cluster := testutil.CreateTestCluster("test-cluster", "test-ns")
				cluster.Status.NumberOfMaster = 3
				cluster.Status.MinReplicationFactor = 1
				cluster.Status.MaxReplicationFactor = 1
				return cluster
			}(),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := needClusterOperation(tt.cluster, testutil.NewTestLogger())
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUpdateClusterInfos(t *testing.T) {
	tests := []struct {
		name          string
		clusterInfos  *redisutil.ClusterInfos
		pods          []*corev1.Pod
		nodes         []*corev1.Node
		expectedError bool
	}{
		{
			name: "Successful update with zone information",
			clusterInfos: createTestClusterInfos([]*redisutil.Node{
				createTestRedisNode("node1", "10.0.0.1", "6379", "master", []redisutil.Slot{}),
			}),
			pods: []*corev1.Pod{
				testutil.CreateTestPodWithOwner("redis-pod-1", "test-ns", "10.0.0.1", "k8s-node-1", "StatefulSet", "redis-cluster-0"),
			},
			nodes: []*corev1.Node{
				testutil.CreateTestNode("k8s-node-1", "us-west-2a", true),
			},
			expectedError: false,
		},
		{
			name: "Pod not found for node",
			clusterInfos: createTestClusterInfos([]*redisutil.Node{
				createTestRedisNode("node1", "10.0.0.1", "6379", "master", []redisutil.Slot{}),
			}),
			pods:          []*corev1.Pod{}, // No pods
			nodes:         []*corev1.Node{},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup scheme and client
			s := scheme.Scheme
			var objs []client.Object
			for i := range tt.nodes {
				objs = append(objs, tt.nodes[i])
			}
			fakeClient := fake.NewClientBuilder().WithScheme(s).WithObjects(objs...).Build()

			mockMetrics := &testutil.MockMetrics{}

			err := UpdateClusterInfos(tt.clusterInfos, tt.pods, fakeClient, testutil.NewTestLogger(), mockMetrics)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewRedisAdmin(t *testing.T) {
	tests := []struct {
		name          string
		pods          []*corev1.Pod
		password      string
		expectedError bool
	}{
		{
			name: "Successful admin creation with pods",
			pods: []*corev1.Pod{
				testutil.CreateTestPod("redis-0", "test-ns", "10.0.0.1", "node1"),
				testutil.CreateTestPod("redis-1", "test-ns", "10.0.0.2", "node2"),
			},
			password:      "test-password",
			expectedError: false,
		},
		{
			name:          "Empty pod list",
			pods:          []*corev1.Pod{},
			password:      "test-password",
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMetrics := &testutil.MockMetrics{}
			cfg := &config.Redis{}

			admin, err := newRedisAdmin(tt.pods, tt.password, cfg, testutil.NewTestLogger(), context.Background(), mockMetrics)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, admin)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, admin)
			}
		})
	}
}

func TestNewRedisCluster(t *testing.T) {
	tests := []struct {
		name          string
		clusterInfos  *redisutil.ClusterInfos
		cluster       *redisv1alpha1.DistributedRedisCluster
		expectedError bool
	}{
		{
			name: "Successful cluster creation with HA enabled",
			clusterInfos: createTestClusterInfos([]*redisutil.Node{
				createMasterRedisNode("master1", "10.0.0.1", "us-west-2a", 0, 5461),
			}),
			cluster:       testutil.CreateTestCluster("test-cluster", "test-ns"),
			expectedError: false,
		},
		{
			name: "Non-HA cluster",
			clusterInfos: createTestClusterInfos([]*redisutil.Node{
				createTestRedisNode("node1", "10.0.0.1", "6379", "master", []redisutil.Slot{}),
			}),
			cluster: func() *redisv1alpha1.DistributedRedisCluster {
				cluster := testutil.CreateTestCluster("test-cluster", "test-ns")
				cluster.Spec.HaConfig = nil
				return cluster
			}(),
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rCluster, nodes, err := newRedisCluster(tt.clusterInfos, tt.cluster, testutil.NewTestLogger())

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, rCluster)
				assert.Nil(t, nodes)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, rCluster)
				assert.NotNil(t, nodes)
			}
		})
	}
}

// Test proxyPods function (0% coverage) - Additional tests
func TestProxyPodsAdditional(t *testing.T) {
	logger := logr.Discard()

	t.Run("No proxy pods for non-single AZ deployment", func(t *testing.T) {
		cluster := testutil.CreateTestCluster("drc-test", "default")

		ctx := &syncContext{
			cluster:              cluster,
			reqLogger:            logger,
			isSingleAzDeployment: false, // Not single AZ
		}

		reconciler := &DistributedRedisClusterReconciler{}
		req := ctrl.Request{NamespacedName: types.NamespacedName{Name: cluster.Name, Namespace: cluster.Namespace}}

		err := proxyPods(ctx, reconciler, req)
		assert.NoError(t, err)
		assert.Nil(t, ctx.proxyPods) // Should remain nil for non-single AZ
	})

	t.Run("Handles error from StatefulSetController", func(t *testing.T) {
		cluster := testutil.CreateTestCluster("drc-test", "default")

		ctx := &syncContext{
			cluster:              cluster,
			reqLogger:            logger,
			isSingleAzDeployment: true, // Single AZ deployment
		}

		mockStsController := &mocks.MockStatefulSetController{}
		mockStsController.On("GetStatefulSetPodsByLabels", cluster.Namespace, mock.AnythingOfType("map[string]string")).Return(&corev1.PodList{}, assert.AnError)

		reconciler := &DistributedRedisClusterReconciler{
			StatefulSetController: mockStsController,
		}
		req := ctrl.Request{NamespacedName: types.NamespacedName{Name: cluster.Name, Namespace: cluster.Namespace}}

		err := proxyPods(ctx, reconciler, req)
		assert.Error(t, err)
		mockStsController.AssertExpectations(t)
	})

	t.Run("Successfully sets proxy pods for single AZ deployment", func(t *testing.T) {
		cluster := testutil.CreateTestCluster("drc-test", "default")

		ctx := &syncContext{
			cluster:              cluster,
			reqLogger:            logger,
			isSingleAzDeployment: true, // Single AZ deployment
		}

		// Create test pods with Running status (required by clusterPods function)
		pod1 := testutil.TestPod("proxy-1", cluster.Namespace)
		pod1.Status.Phase = corev1.PodRunning
		pod2 := testutil.TestPod("proxy-2", cluster.Namespace)
		pod2.Status.Phase = corev1.PodRunning

		pods := &corev1.PodList{
			Items: []corev1.Pod{*pod1, *pod2},
		}

		mockStsController := &mocks.MockStatefulSetController{}
		mockStsController.On("GetStatefulSetPodsByLabels", cluster.Namespace, mock.AnythingOfType("map[string]string")).Return(pods, nil)

		reconciler := &DistributedRedisClusterReconciler{
			StatefulSetController: mockStsController,
		}
		req := ctrl.Request{NamespacedName: types.NamespacedName{Name: cluster.Name, Namespace: cluster.Namespace}}

		err := proxyPods(ctx, reconciler, req)
		assert.NoError(t, err)
		assert.NotNil(t, ctx.proxyPods)
		assert.Len(t, ctx.proxyPods, 2) // Should have 2 running proxy pods
		mockStsController.AssertExpectations(t)
	})
}

// Test findProxyAz function (0% coverage) - Additional tests
func TestFindProxyAzAdditional(t *testing.T) {
	logger := logr.Discard()

	t.Run("Returns empty string for non-single AZ deployment", func(t *testing.T) {
		cluster := testutil.CreateTestCluster("drc-test", "default")

		ctx := &syncContext{
			cluster:              cluster,
			reqLogger:            logger,
			isSingleAzDeployment: false, // Not single AZ
		}

		reconciler := &DistributedRedisClusterReconciler{}

		az, err := findProxyAz(reconciler, ctx)
		assert.NoError(t, err)
		assert.Equal(t, "", az) // Should return empty string
	})
}

// Test interface methods for waitPodTerminating (currently 0% coverage)
func TestWaitPodTerminatingInterfaceMethods(t *testing.T) {
	cluster := testutil.CreateTestCluster("drc-test", "default")

	waitPod := &waitPodTerminating{
		name:        "test-wait-pod",
		statefulSet: "test-sts",
		timeout:     time.Minute * 5,
		tick:        time.Second * 10,
		cluster:     cluster,
	}

	t.Run("Name returns correct name", func(t *testing.T) {
		assert.Equal(t, "test-wait-pod", waitPod.Name())
	})

	t.Run("Tick returns correct tick duration", func(t *testing.T) {
		assert.Equal(t, time.Second*10, waitPod.Tick())
	})

	t.Run("Timeout returns correct timeout duration", func(t *testing.T) {
		assert.Equal(t, time.Minute*5, waitPod.Timeout())
	})

	t.Run("Handler returns error when StatefulSetController fails", func(t *testing.T) {
		mockStsController := &mocks.MockStatefulSetController{}
		mockStsController.On("GetStatefulSetPodsByLabels", cluster.Namespace, mock.AnythingOfType("map[string]string")).Return(&corev1.PodList{}, assert.AnError)

		waitPod.statefulSetController = mockStsController

		err := waitPod.Handler()
		assert.Error(t, err)
		mockStsController.AssertExpectations(t)
	})
}

// Test interface methods for waitStatefulSetUpdating (currently 0% coverage)
func TestWaitStatefulSetUpdatingInterfaceMethods(t *testing.T) {
	cluster := testutil.CreateTestCluster("drc-test", "default")

	waitSts := &waitStatefulSetUpdating{
		name:    "test-wait-sts",
		timeout: time.Minute * 3,
		tick:    time.Second * 5,
		cluster: cluster,
	}

	t.Run("Name returns correct name", func(t *testing.T) {
		assert.Equal(t, "test-wait-sts", waitSts.Name())
	})

	t.Run("Tick returns correct tick duration", func(t *testing.T) {
		assert.Equal(t, time.Second*5, waitSts.Tick())
	})

	t.Run("Timeout returns correct timeout duration", func(t *testing.T) {
		assert.Equal(t, time.Minute*3, waitSts.Timeout())
	})

	t.Run("Handler returns error when StatefulSetController fails", func(t *testing.T) {
		mockStsController := &mocks.MockStatefulSetController{}
		mockStsController.On("ListStatefulSetByLabels", cluster.Namespace, mock.AnythingOfType("map[string]string")).Return(&appsv1.StatefulSetList{}, assert.AnError)

		waitSts.statefulSetController = mockStsController

		err := waitSts.Handler()
		assert.Error(t, err)
		mockStsController.AssertExpectations(t)
	})
}
