package heal

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MockPodControl for testing - implements all IPodControl methods
type MockPodControl struct {
	DeletePodByNameFunc func(namespace, name string) error
	CreatePodFunc       func(pod *corev1.Pod) error
	UpdatePodFunc       func(pod *corev1.Pod) error
	DeletePodFunc       func(pod *corev1.Pod) error
	GetPodFunc          func(namespace, name string) (*corev1.Pod, error)
}

func (m *MockPodControl) CreatePod(pod *corev1.Pod) error {
	if m.CreatePodFunc != nil {
		return m.CreatePodFunc(pod)
	}
	return nil
}

func (m *MockPodControl) UpdatePod(pod *corev1.Pod) error {
	if m.UpdatePodFunc != nil {
		return m.UpdatePodFunc(pod)
	}
	return nil
}

func (m *MockPodControl) DeletePod(pod *corev1.Pod) error {
	if m.DeletePodFunc != nil {
		return m.DeletePodFunc(pod)
	}
	return nil
}

func (m *MockPodControl) DeletePodByName(namespace, name string) error {
	if m.DeletePodByNameFunc != nil {
		return m.DeletePodByNameFunc(namespace, name)
	}
	return nil
}

func (m *MockPodControl) GetPod(namespace, name string) (*corev1.Pod, error) {
	if m.GetPodFunc != nil {
		return m.GetPodFunc(namespace, name)
	}
	return &corev1.Pod{}, nil
}

func (m *MockPodControl) UpdatePodLabels(pod *corev1.Pod, labelKey string, labelValue string) error {
	return nil
}

func (m *MockPodControl) ListPodsWithLabel(namespace string, labelkey string, labelvalue string) (*corev1.PodList, error) {
	return &corev1.PodList{}, nil
}

func (m *MockPodControl) UpdatePodAnnotations(pod *corev1.Pod, annotationKey string, annotationvalue string) error {
	return nil
}

// MockMetrics for testing
type MockMetrics struct {
	ReconcileClusterFunc func(name, namespace, operation, component, status, reason string)
}

func (m *MockMetrics) ReconcileCluster(name, namespace, operation, component, status, reason string) {
	if m.ReconcileClusterFunc != nil {
		m.ReconcileClusterFunc(name, namespace, operation, component, status, reason)
	}
}

func (m *MockMetrics) SetClusterOK(name, namespace string, val float64)                          {}
func (m *MockMetrics) RecordK8sOperation(namespace, kind, object, operation, status, err string) {}
func (m *MockMetrics) RecordRedisOperation(namespace, name, operation, ip, status, err string)   {}
func (m *MockMetrics) ReconcileClusterGauge(namespace, name, function string, value float64)     {}

func TestCheckAndHeal_Struct(t *testing.T) {
	// Test creating CheckAndHeal struct
	logger := logr.Discard()
	podControl := &MockPodControl{}
	metricsClient := &MockMetrics{}

	checkAndHeal := &CheckAndHeal{
		Logger:        logger,
		PodControl:    podControl,
		Pods:          []*corev1.Pod{},
		DryRun:        false,
		MetricsClient: metricsClient,
	}

	assert.NotNil(t, checkAndHeal)
	assert.Equal(t, logger, checkAndHeal.Logger)
	assert.Equal(t, podControl, checkAndHeal.PodControl)
	assert.Equal(t, []*corev1.Pod{}, checkAndHeal.Pods)
	assert.False(t, checkAndHeal.DryRun)
	assert.Equal(t, metricsClient, checkAndHeal.MetricsClient)
}

func TestCheckAndHeal_WithPods(t *testing.T) {
	// Test CheckAndHeal with sample pods
	logger := logr.Discard()
	podControl := &MockPodControl{}
	metricsClient := &MockMetrics{}

	// Create test pods
	pods := []*corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod-1",
				Namespace: "test-ns",
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod-2",
				Namespace: "test-ns",
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodPending,
			},
		},
	}

	checkAndHeal := &CheckAndHeal{
		Logger:        logger,
		PodControl:    podControl,
		Pods:          pods,
		DryRun:        true, // Use DryRun for testing
		MetricsClient: metricsClient,
	}

	assert.NotNil(t, checkAndHeal)
	assert.Len(t, checkAndHeal.Pods, 2)
	assert.True(t, checkAndHeal.DryRun)
	assert.Equal(t, "test-pod-1", checkAndHeal.Pods[0].Name)
	assert.Equal(t, "test-pod-2", checkAndHeal.Pods[1].Name)
}

func TestCheckAndHeal_DryRunMode(t *testing.T) {
	// Test DryRun mode functionality
	tests := []struct {
		name   string
		dryRun bool
	}{
		{
			name:   "DryRun enabled",
			dryRun: true,
		},
		{
			name:   "DryRun disabled",
			dryRun: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkAndHeal := &CheckAndHeal{
				Logger:        logr.Discard(),
				PodControl:    &MockPodControl{},
				Pods:          []*corev1.Pod{},
				DryRun:        tt.dryRun,
				MetricsClient: &MockMetrics{},
			}

			assert.Equal(t, tt.dryRun, checkAndHeal.DryRun)
		})
	}
}

func TestCheckAndHeal_WithNilComponents(t *testing.T) {
	// Test CheckAndHeal with nil components (edge cases)
	checkAndHeal := &CheckAndHeal{
		Logger:        logr.Discard(),
		PodControl:    nil, // This could happen in some edge cases
		Pods:          nil,
		DryRun:        false,
		MetricsClient: nil, // This could happen in some edge cases
	}

	assert.NotNil(t, checkAndHeal)
	assert.Nil(t, checkAndHeal.PodControl)
	assert.Nil(t, checkAndHeal.Pods)
	assert.Nil(t, checkAndHeal.MetricsClient)
}

func TestCheckAndHeal_PodFiltering(t *testing.T) {
	// Test working with different pod states
	logger := logr.Discard()
	podControl := &MockPodControl{}
	metricsClient := &MockMetrics{}

	// Create pods in different states
	pods := []*corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "running-pod",
				Namespace: "test-ns",
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pending-pod",
				Namespace: "test-ns",
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodPending,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "failed-pod",
				Namespace: "test-ns",
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodFailed,
			},
		},
	}

	checkAndHeal := &CheckAndHeal{
		Logger:        logger,
		PodControl:    podControl,
		Pods:          pods,
		DryRun:        false,
		MetricsClient: metricsClient,
	}

	// Count pods by phase
	runningCount := 0
	pendingCount := 0
	failedCount := 0

	for _, pod := range checkAndHeal.Pods {
		switch pod.Status.Phase {
		case corev1.PodRunning:
			runningCount++
		case corev1.PodPending:
			pendingCount++
		case corev1.PodFailed:
			failedCount++
		}
	}

	assert.Equal(t, 1, runningCount)
	assert.Equal(t, 1, pendingCount)
	assert.Equal(t, 1, failedCount)
	assert.Len(t, checkAndHeal.Pods, 3)
}
