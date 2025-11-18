package heal

import (
	"errors"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	redisv1alpha1 "github.com/freshworks/redis-cluster-operator/api/v1alpha1"
)

func TestCheckAndHeal_FixTerminatingPods_ZeroDuration(t *testing.T) {
	// Test when maxDuration is 0 - should return immediately
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

	actionDone, err := checkAndHeal.FixTerminatingPods(cluster, time.Duration(0))

	assert.NoError(t, err)
	assert.False(t, actionDone)
}

func TestCheckAndHeal_FixTerminatingPods_NoTerminatingPods(t *testing.T) {
	// Test when no pods are terminating
	pods := []*corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "running-pod",
				Namespace: "test-ns",
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
			},
			// No DeletionTimestamp - not terminating
		},
	}

	checkAndHeal := &CheckAndHeal{
		Logger:        logr.Discard(),
		PodControl:    &MockPodControl{},
		Pods:          pods,
		DryRun:        false,
		MetricsClient: &MockMetrics{},
	}

	cluster := &redisv1alpha1.DistributedRedisCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster", Namespace: "test-ns"},
	}

	actionDone, err := checkAndHeal.FixTerminatingPods(cluster, 5*time.Minute)

	assert.NoError(t, err)
	assert.False(t, actionDone)
}

func TestCheckAndHeal_FixTerminatingPods_RecentlyTerminating(t *testing.T) {
	// Test when pods are terminating but haven't exceeded maxDuration
	now := time.Now()
	recentDeletionTime := metav1.NewTime(now.Add(-2 * time.Minute)) // 2 minutes ago

	pods := []*corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "terminating-pod",
				Namespace:         "test-ns",
				DeletionTimestamp: &recentDeletionTime,
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
			},
		},
	}

	checkAndHeal := &CheckAndHeal{
		Logger:        logr.Discard(),
		PodControl:    &MockPodControl{},
		Pods:          pods,
		DryRun:        false,
		MetricsClient: &MockMetrics{},
	}

	cluster := &redisv1alpha1.DistributedRedisCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster", Namespace: "test-ns"},
	}

	// Max duration is 5 minutes, pod was deleted 2 minutes ago, so should not be deleted
	actionDone, err := checkAndHeal.FixTerminatingPods(cluster, 5*time.Minute)

	assert.NoError(t, err)
	assert.False(t, actionDone)
}

func TestCheckAndHeal_FixTerminatingPods_OldTerminatingPods_DryRun(t *testing.T) {
	// Test when pods are terminating and have exceeded maxDuration, but DryRun is enabled
	now := time.Now()
	oldDeletionTime := metav1.NewTime(now.Add(-10 * time.Minute)) // 10 minutes ago

	pods := []*corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "stuck-terminating-pod",
				Namespace:         "test-ns",
				DeletionTimestamp: &oldDeletionTime,
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
			},
		},
	}

	checkAndHeal := &CheckAndHeal{
		Logger:        logr.Discard(),
		PodControl:    &MockPodControl{},
		Pods:          pods,
		DryRun:        true, // DryRun mode - should not actually delete
		MetricsClient: &MockMetrics{},
	}

	cluster := &redisv1alpha1.DistributedRedisCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster", Namespace: "test-ns"},
	}

	// Max duration is 5 minutes, pod was deleted 10 minutes ago, should be marked for deletion
	actionDone, err := checkAndHeal.FixTerminatingPods(cluster, 5*time.Minute)

	assert.NoError(t, err)
	assert.True(t, actionDone) // Action was identified but not performed due to DryRun
}

func TestCheckAndHeal_FixTerminatingPods_OldTerminatingPods_Success(t *testing.T) {
	// Test when pods are terminating and have exceeded maxDuration - should delete
	now := time.Now()
	oldDeletionTime := metav1.NewTime(now.Add(-10 * time.Minute)) // 10 minutes ago

	pods := []*corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "stuck-terminating-pod",
				Namespace:         "test-ns",
				DeletionTimestamp: &oldDeletionTime,
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
			},
		},
	}

	deleteCalled := false
	podControl := &MockPodControl{
		DeletePodByNameFunc: func(namespace, name string) error {
			deleteCalled = true
			assert.Equal(t, "test-ns", namespace)
			assert.Equal(t, "stuck-terminating-pod", name)
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

	actionDone, err := checkAndHeal.FixTerminatingPods(cluster, 5*time.Minute)

	assert.NoError(t, err)
	assert.True(t, actionDone)
	assert.True(t, deleteCalled)
}

func TestCheckAndHeal_FixTerminatingPods_DeleteError(t *testing.T) {
	// Test when pod deletion fails
	now := time.Now()
	oldDeletionTime := metav1.NewTime(now.Add(-10 * time.Minute)) // 10 minutes ago

	pods := []*corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "stuck-terminating-pod",
				Namespace:         "test-ns",
				DeletionTimestamp: &oldDeletionTime,
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
			},
		},
	}

	expectedError := errors.New("failed to delete pod")
	podControl := &MockPodControl{
		DeletePodByNameFunc: func(namespace, name string) error {
			return expectedError
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

	actionDone, err := checkAndHeal.FixTerminatingPods(cluster, 5*time.Minute)

	assert.Error(t, err)
	assert.True(t, actionDone)
	assert.Contains(t, err.Error(), "failed to delete pod")
}

func TestCheckAndHeal_FixTerminatingPods_MultiplePods(t *testing.T) {
	// Test with multiple pods in different states
	now := time.Now()
	oldDeletionTime := metav1.NewTime(now.Add(-10 * time.Minute))   // 10 minutes ago
	recentDeletionTime := metav1.NewTime(now.Add(-2 * time.Minute)) // 2 minutes ago

	pods := []*corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "stuck-terminating-pod",
				Namespace:         "test-ns",
				DeletionTimestamp: &oldDeletionTime,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "recent-terminating-pod",
				Namespace:         "test-ns",
				DeletionTimestamp: &recentDeletionTime,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "running-pod",
				Namespace: "test-ns",
				// No DeletionTimestamp
			},
		},
	}

	deleteCallCount := 0
	podControl := &MockPodControl{
		DeletePodByNameFunc: func(namespace, name string) error {
			deleteCallCount++
			assert.Equal(t, "test-ns", namespace)
			assert.Equal(t, "stuck-terminating-pod", name) // Only this pod should be deleted
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

	actionDone, err := checkAndHeal.FixTerminatingPods(cluster, 5*time.Minute)

	assert.NoError(t, err)
	assert.True(t, actionDone)
	assert.Equal(t, 1, deleteCallCount) // Only one pod should be deleted
}
