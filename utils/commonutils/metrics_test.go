package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/dinesh-murugiah/rediscluster-operator/metrics"
)

// MockRecorder is a mock implementation of metrics.Recorder
type MockRecorder struct {
	mock.Mock
}

func (m *MockRecorder) SetClusterOK(namespace string, name string, val float64) {
	m.Called(namespace, name, val)
}

func (m *MockRecorder) ReconcileClusterGauge(namespace string, name string, function string, val float64) {
	m.Called(namespace, name, function, val)
}

func (m *MockRecorder) RecordK8sOperation(namespace, kind, object, operation, status, errorType string) {
	m.Called(namespace, kind, object, operation, status, errorType)
}

func (m *MockRecorder) RecordRedisOperation(namespace, name, operation, ip, status, errorMsg string) {
	m.Called(namespace, name, operation, ip, status, errorMsg)
}

func (m *MockRecorder) ReconcileCluster(namespace, name, operation, function, status, err string) {
	m.Called(namespace, name, operation, function, status, err)
}

func TestRecordK8sMetrics(t *testing.T) {
	tests := []struct {
		name           string
		namespace      string
		kind           string
		object         string
		operation      string
		err            error
		expectedStatus string
		expectedError  string
	}{
		{
			name:           "Success case - no error",
			namespace:      "test-ns",
			kind:           "Pod",
			object:         "test-pod",
			operation:      "CREATE",
			err:            nil,
			expectedStatus: metrics.SUCCESS,
			expectedError:  metrics.NOT_APPLICABLE,
		},
		{
			name:           "Forbidden error",
			namespace:      "test-ns",
			kind:           "Pod",
			object:         "test-pod",
			operation:      "CREATE",
			err:            &errors.StatusError{ErrStatus: metav1.Status{Code: 403}},
			expectedStatus: metrics.FAIL,
			expectedError:  metrics.K8S_FORBIDDEN_ERR,
		},
		{
			name:           "Unauthorized error",
			namespace:      "test-ns",
			kind:           "Pod",
			object:         "test-pod",
			operation:      "CREATE",
			err:            &errors.StatusError{ErrStatus: metav1.Status{Code: 401}},
			expectedStatus: metrics.FAIL,
			expectedError:  metrics.K8S_UNAUTH,
		},
		{
			name:           "Not found error",
			namespace:      "test-ns",
			kind:           "Pod",
			object:         "test-pod",
			operation:      "CREATE",
			err:            &errors.StatusError{ErrStatus: metav1.Status{Code: 404}},
			expectedStatus: metrics.FAIL,
			expectedError:  metrics.K8S_NOT_FOUND,
		},
		{
			name:           "Other error",
			namespace:      "test-ns",
			kind:           "Pod",
			object:         "test-pod",
			operation:      "CREATE",
			err:            &errors.StatusError{ErrStatus: metav1.Status{Code: 500}},
			expectedStatus: metrics.FAIL,
			expectedError:  metrics.K8S_MISC,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRecorder := &MockRecorder{}
			mockRecorder.On("RecordK8sOperation", tt.namespace, tt.kind, tt.object, tt.operation, tt.expectedStatus, tt.expectedError).Return()

			RecordK8sMetrics(tt.namespace, tt.kind, tt.object, tt.operation, tt.err, mockRecorder)

			mockRecorder.AssertExpectations(t)
		})
	}
}

func TestRecordRedisMetrics(t *testing.T) {
	tests := []struct {
		name           string
		namespace      string
		name2          string
		operation      string
		ip             string
		err            error
		expectedStatus string
		expectedError  string
	}{
		{
			name:           "Success case - no error",
			namespace:      "test-ns",
			name2:          "test-cluster",
			operation:      "PING",
			ip:             "192.168.1.1",
			err:            nil,
			expectedStatus: metrics.SUCCESS,
			expectedError:  "nil",
		},
		{
			name:           "Failure case - with error",
			namespace:      "test-ns",
			name2:          "test-cluster",
			operation:      "PING",
			ip:             "192.168.1.1",
			err:            assert.AnError,
			expectedStatus: metrics.FAIL,
			expectedError:  assert.AnError.Error(),
		},
		{
			name:           "Nil recorder - should skip recording",
			namespace:      "test-ns",
			name2:          "test-cluster",
			operation:      "PING",
			ip:             "192.168.1.1",
			err:            nil,
			expectedStatus: metrics.SUCCESS,
			expectedError:  "nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "Nil recorder - should skip recording" {
				// Test with nil recorder - should not panic
				RecordRedisMetrics(tt.namespace, tt.name2, tt.operation, tt.ip, tt.err, nil)
				return
			}

			mockRecorder := &MockRecorder{}
			mockRecorder.On("RecordRedisOperation", tt.namespace, tt.name2, tt.operation, tt.ip, tt.expectedStatus, tt.expectedError).Return()

			RecordRedisMetrics(tt.namespace, tt.name2, tt.operation, tt.ip, tt.err, mockRecorder)

			mockRecorder.AssertExpectations(t)
		})
	}
}

func TestRecordK8sMetricsWithRealErrors(t *testing.T) {
	mockRecorder := &MockRecorder{}

	// Test with real k8s errors
	t.Run("Forbidden error", func(t *testing.T) {
		mockRecorder.On("RecordK8sOperation", "test-ns", "Pod", "test-pod", "CREATE", metrics.FAIL, metrics.K8S_FORBIDDEN_ERR).Return()

		err := errors.NewForbidden(schema.GroupResource{Group: "v1", Resource: "pods"}, "test-pod", assert.AnError)
		RecordK8sMetrics("test-ns", "Pod", "test-pod", "CREATE", err, mockRecorder)

		mockRecorder.AssertExpectations(t)
	})

	t.Run("Unauthorized error", func(t *testing.T) {
		mockRecorder.On("RecordK8sOperation", "test-ns", "Pod", "test-pod", "CREATE", metrics.FAIL, metrics.K8S_UNAUTH).Return()

		err := errors.NewUnauthorized("unauthorized")
		RecordK8sMetrics("test-ns", "Pod", "test-pod", "CREATE", err, mockRecorder)

		mockRecorder.AssertExpectations(t)
	})

	t.Run("Not found error", func(t *testing.T) {
		mockRecorder.On("RecordK8sOperation", "test-ns", "Pod", "test-pod", "CREATE", metrics.FAIL, metrics.K8S_NOT_FOUND).Return()

		err := errors.NewNotFound(schema.GroupResource{Group: "v1", Resource: "pods"}, "test-pod")
		RecordK8sMetrics("test-ns", "Pod", "test-pod", "CREATE", err, mockRecorder)

		mockRecorder.AssertExpectations(t)
	})
}
