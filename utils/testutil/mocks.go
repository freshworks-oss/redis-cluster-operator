package testutil

import (
	"github.com/go-logr/logr"
)

// MockMetrics provides a shared mock implementation of metrics.Recorder
// This replaces the duplicate mockMetrics structs across multiple test files
type MockMetrics struct{}

func (m *MockMetrics) SetClusterOK(namespace string, name string, val float64) {}

func (m *MockMetrics) ReconcileClusterGauge(namespace string, name string, function string, val float64) {
}

func (m *MockMetrics) RecordK8sOperation(namespace, kind, object, operation, status, errorType string) {
}

func (m *MockMetrics) RecordRedisOperation(namespace, name, operation, ip, status, errorMsg string) {}

func (m *MockMetrics) ReconcileCluster(namespace, name, operation, function, status, err string) {}

// MockLogger provides a shared mock implementation of logr.LogSink
// This replaces the duplicate MockLogger structs across redisutil test files
type MockLogger struct{}

func (m MockLogger) Init(runtimeInfo logr.RuntimeInfo)                         {}
func (m MockLogger) Info(level int, msg string, keysAndValues ...interface{})  {}
func (m MockLogger) Error(err error, msg string, keysAndValues ...interface{}) {}
func (m MockLogger) Enabled(level int) bool                                    { return true }
func (m MockLogger) WithValues(keysAndValues ...interface{}) logr.LogSink      { return m }
func (m MockLogger) WithName(name string) logr.LogSink                         { return m }

// NewTestLogger creates a proper logr.Logger for testing
func NewTestLogger() logr.Logger {
	return logr.New(MockLogger{})
}
