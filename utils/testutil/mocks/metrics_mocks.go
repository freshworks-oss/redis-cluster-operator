package mocks

import (
	"github.com/stretchr/testify/mock"
)

// MockMetricsRecorder provides a unified mock for all metrics recording operations
// Consolidates MockMetricsRecorder, MockMetrics, MockRecorder across multiple files
type MockMetricsRecorder struct {
	mock.Mock
}

func (m *MockMetricsRecorder) ReconcileCluster(namespace, name, operation, phase, status, reason string) {
	m.Called(namespace, name, operation, phase, status, reason)
}

func (m *MockMetricsRecorder) ReconcileClusterGauge(namespace, name, operation string, value float64) {
	m.Called(namespace, name, operation, value)
}

func (m *MockMetricsRecorder) RecordK8sOperation(namespace, resource, name, operation, status, reason string) {
	m.Called(namespace, resource, name, operation, status, reason)
}

func (m *MockMetricsRecorder) RecordRedisOperation(namespace, name, operation, address, status, reason string) {
	m.Called(namespace, name, operation, address, status, reason)
}

func (m *MockMetricsRecorder) SetClusterOK(namespace, name string, value float64) {
	m.Called(namespace, name, value)
}

// Alternative method signatures for different interfaces
func (m *MockMetricsRecorder) RecordK8sOperationAlt(namespace, kind, object, operation, status, err string) {
	m.Called(namespace, kind, object, operation, status, err)
}

func (m *MockMetricsRecorder) ReconcileClusterAlt(name, namespace, operation, component, status, reason string) {
	m.Called(name, namespace, operation, component, status, reason)
}

// MockLogger provides a unified mock for logging operations
type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Info(msg string, keysAndValues ...interface{}) {
	args := []interface{}{msg}
	args = append(args, keysAndValues...)
	m.Called(args...)
}

func (m *MockLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	args := []interface{}{err, msg}
	args = append(args, keysAndValues...)
	m.Called(args...)
}

func (m *MockLogger) Debug(msg string, keysAndValues ...interface{}) {
	args := []interface{}{msg}
	args = append(args, keysAndValues...)
	m.Called(args...)
}

func (m *MockLogger) V(level int) *MockLogger {
	m.Called(level)
	return m
}

func (m *MockLogger) WithValues(keysAndValues ...interface{}) *MockLogger {
	m.Called(keysAndValues...)
	return m
}

func (m *MockLogger) WithName(name string) *MockLogger {
	m.Called(name)
	return m
}

func (m *MockLogger) Enabled() bool {
	args := m.Called()
	return args.Bool(0)
}
