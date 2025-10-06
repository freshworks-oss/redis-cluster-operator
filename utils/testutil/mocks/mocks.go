// Package mocks provides consolidated mock implementations for all testing needs
// across the Redis Cluster Operator codebase.
//
// This package consolidates previously duplicated mocks from multiple files:
// - controllers/manager/mocks_test.go
// - controllers/heal/*.go test files
// - controllers/clustering/clustering_test.go
// - utils/testutil/controller_mocks.go
// - And others throughout the codebase
//
// Usage:
//
//	import "github.com/freshworks/redis-cluster-operator/utils/testutil/mocks"
//
//	mockSts := &mocks.MockStatefulSetController{}
//	mockRedis := &mocks.MockRedisAdmin{}
//	mockMetrics := &mocks.MockMetricsRecorder{}
//
// This eliminates duplication and provides a single source of truth for all mocks.
package mocks

// Re-export all mock types for convenient access
// This allows importing the mocks package and accessing all mocks directly

// Kubernetes Controller Mocks (from k8s_mocks.go)
type (
	StatefulSetController    = MockStatefulSetController
	ServiceController        = MockServiceController
	PodController            = MockPodController
	PDBController            = MockPodDisruptionBudgetController
	ConfigMapController      = MockConfigMapController
	PvcController            = MockPvcController
	CustomResourceController = MockCustomResourceController
	NodeController           = MockNodeController
	K8sClient                = MockClient
)

// Redis Mocks (from redis_mocks.go)
type (
	RedisAdmin       = MockRedisAdmin
	AdminConnections = MockAdminConnections
	RedisConnection  = MockRedisConnection
	StringCmd        = MockStringCmd
	Cmd              = MockCmd
)

// Metrics Mocks (from metrics_mocks.go)
type (
	MetricsRecorder = MockMetricsRecorder
	Logger          = MockLogger
)

// Common Utility Mocks (from common_mocks.go)
type (
	Execer  = MockExecer
	Ensurer = MockEnsurer
	Checker = MockChecker
	Healer  = MockHealer
)

// Constructor functions for commonly used mock combinations
func NewK8sControllerMocks() (*StatefulSetController, *ServiceController, *PodController, *ConfigMapController, *PDBController, *CustomResourceController, *NodeController, *PvcController, *K8sClient) {
	return &StatefulSetController{}, &ServiceController{}, &PodController{}, &ConfigMapController{}, &PDBController{}, &CustomResourceController{}, &NodeController{}, &PvcController{}, &K8sClient{}
}

func NewRedisMocks() (*RedisAdmin, *AdminConnections, *RedisConnection, *StringCmd, *Cmd) {
	return &RedisAdmin{}, &AdminConnections{}, &RedisConnection{}, &StringCmd{}, &Cmd{}
}

func NewMetricsMocks() (*MetricsRecorder, *Logger) {
	return &MetricsRecorder{}, &Logger{}
}

func NewCommonMocks() (*Execer, *Ensurer, *Checker, *Healer) {
	return &Execer{}, &Ensurer{}, &Checker{}, &Healer{}
}

func NewAllMocks() (*StatefulSetController, *ServiceController, *PodController, *ConfigMapController, *PDBController, *CustomResourceController, *NodeController, *PvcController, *K8sClient, *RedisAdmin, *AdminConnections, *RedisConnection, *StringCmd, *Cmd, *MetricsRecorder, *Logger, *Execer, *Ensurer, *Checker, *Healer) {
	return &StatefulSetController{}, &ServiceController{}, &PodController{}, &ConfigMapController{}, &PDBController{}, &CustomResourceController{}, &NodeController{}, &PvcController{}, &K8sClient{},
		&RedisAdmin{}, &AdminConnections{}, &RedisConnection{}, &StringCmd{}, &Cmd{},
		&MetricsRecorder{}, &Logger{},
		&Execer{}, &Ensurer{}, &Checker{}, &Healer{}
}
