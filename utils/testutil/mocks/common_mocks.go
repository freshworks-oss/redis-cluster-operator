package mocks

import (
	"context"
	"time"

	redisv1alpha1 "github.com/dinesh-murugiah/rediscluster-operator/api/v1alpha1"
	"github.com/dinesh-murugiah/rediscluster-operator/utils/redisutil"
	"github.com/stretchr/testify/mock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// MockExecer provides a unified mock for command execution operations
type MockExecer struct {
	mock.Mock
}

func (m *MockExecer) ExecCommandInPodSet(podSet []*corev1.Pod, cmd ...string) error {
	args := m.Called(podSet, cmd)
	return args.Error(0)
}

func (m *MockExecer) ExecCommand(ctx context.Context, podName, namespace, container string, command []string) (string, string, error) {
	args := m.Called(ctx, podName, namespace, container, command)
	return args.String(0), args.String(1), args.Error(2)
}

// MockBuilder provides a unified mock for building test scenarios
type MockBuilder struct {
	mock.Mock
	MockStatefulSet *MockStatefulSetController
	MockService     *MockServiceController
	MockConfigMap   *MockConfigMapController
	MockPDB         *MockPodDisruptionBudgetController
	MockCR          *MockCustomResourceController
	MockPod         *MockPodController
	MockNode        *MockNodeController
	MockPvc         *MockPvcController
}

func NewMockBuilder() *MockBuilder {
	return &MockBuilder{
		MockStatefulSet: &MockStatefulSetController{},
		MockService:     &MockServiceController{},
		MockConfigMap:   &MockConfigMapController{},
		MockPDB:         &MockPodDisruptionBudgetController{},
		MockCR:          &MockCustomResourceController{},
		MockPod:         &MockPodController{},
		MockNode:        &MockNodeController{},
		MockPvc:         &MockPvcController{},
	}
}

func (m *MockBuilder) WithStatefulSetNotFound(namespace, name string) *MockBuilder {
	m.MockStatefulSet.On("GetStatefulSet", namespace, name).Return(
		(*appsv1.StatefulSet)(nil),
		errors.NewNotFound(schema.GroupResource{Resource: "statefulset"}, name))
	return m
}

func (m *MockBuilder) WithStatefulSetFound(namespace, name string, sts *appsv1.StatefulSet) *MockBuilder {
	m.MockStatefulSet.On("GetStatefulSet", namespace, name).Return(sts, nil)
	return m
}

func (m *MockBuilder) WithServiceNotFound(namespace, name string) *MockBuilder {
	m.MockService.On("GetService", namespace, name).Return(
		(*corev1.Service)(nil),
		errors.NewNotFound(schema.GroupResource{Resource: "service"}, name))
	return m
}

func (m *MockBuilder) WithServiceFound(namespace, name string, svc *corev1.Service) *MockBuilder {
	m.MockService.On("GetService", namespace, name).Return(svc, nil)
	return m
}

func (m *MockBuilder) WithConfigMapNotFound(namespace, name string) *MockBuilder {
	m.MockConfigMap.On("GetConfigMap", namespace, name).Return(
		(*corev1.ConfigMap)(nil),
		errors.NewNotFound(schema.GroupResource{Resource: "configmap"}, name))
	return m
}

func (m *MockBuilder) WithConfigMapFound(namespace, name string, cm *corev1.ConfigMap) *MockBuilder {
	m.MockConfigMap.On("GetConfigMap", namespace, name).Return(cm, nil)
	return m
}

func (m *MockBuilder) WithPDBNotFound(namespace, name string) *MockBuilder {
	m.MockPDB.On("GetPodDisruptionBudget", namespace, name).Return(
		(*policyv1.PodDisruptionBudget)(nil),
		errors.NewNotFound(schema.GroupResource{Resource: "poddisruptionbudget"}, name))
	return m
}

func (m *MockBuilder) WithPDBFound(namespace, name string, pdb *policyv1.PodDisruptionBudget) *MockBuilder {
	m.MockPDB.On("GetPodDisruptionBudget", namespace, name).Return(pdb, nil)
	return m
}

// MockEnsurer provides a unified mock for resource ensuring operations
type MockEnsurer struct {
	mock.Mock
}

func (m *MockEnsurer) EnsureRedisStatefulsets(cluster *redisv1alpha1.DistributedRedisCluster, labels map[string]string, update bool) (redisv1alpha1.StsEnsureStatus, error) {
	args := m.Called(cluster, labels, update)
	return args.Get(0).(redisv1alpha1.StsEnsureStatus), args.Error(1)
}

func (m *MockEnsurer) EnsureRedisPDB(cluster *redisv1alpha1.DistributedRedisCluster, labels map[string]string) error {
	args := m.Called(cluster, labels)
	return args.Error(0)
}

func (m *MockEnsurer) EnsureClusterShardsHeadLessSvcs(cluster *redisv1alpha1.DistributedRedisCluster, labels map[string]string) error {
	args := m.Called(cluster, labels)
	return args.Error(0)
}

func (m *MockEnsurer) EnsureClusterHeadLessSvc(cluster *redisv1alpha1.DistributedRedisCluster, labels map[string]string) error {
	args := m.Called(cluster, labels)
	return args.Error(0)
}

func (m *MockEnsurer) EnsureRedisConfigMap(cluster *redisv1alpha1.DistributedRedisCluster, labels map[string]string) error {
	args := m.Called(cluster, labels)
	return args.Error(0)
}

func (m *MockEnsurer) EnsureRedisRCloneSecret(cluster *redisv1alpha1.DistributedRedisCluster, labels map[string]string) error {
	args := m.Called(cluster, labels)
	return args.Error(0)
}

func (m *MockEnsurer) UpdateRedisStatefulsets(cluster *redisv1alpha1.DistributedRedisCluster, labels map[string]string) error {
	args := m.Called(cluster, labels)
	return args.Error(0)
}

// MockChecker provides a unified mock for checking operations
type MockChecker struct {
	mock.Mock
}

func (m *MockChecker) CheckRedisNodeNum(cluster *redisv1alpha1.DistributedRedisCluster) error {
	args := m.Called(cluster)
	return args.Error(0)
}

func (m *MockChecker) CheckPasswordChanged(cluster *redisv1alpha1.DistributedRedisCluster) (bool, string, map[string]string, error) {
	args := m.Called(cluster)
	return args.Bool(0), args.String(1), args.Get(2).(map[string]string), args.Error(3)
}

// MockHealer provides a unified mock for healing operations
type MockHealer struct {
	mock.Mock
}

func (m *MockHealer) Heal(cluster *redisv1alpha1.DistributedRedisCluster, clusterInfos *redisutil.ClusterInfos, admin redisutil.IAdmin, ctx context.Context) (bool, bool, error) {
	args := m.Called(cluster, clusterInfos, admin, ctx)
	return args.Bool(0), args.Bool(1), args.Error(2)
}

func (m *MockHealer) FixTerminatingPods(cluster *redisv1alpha1.DistributedRedisCluster, terminationGracePeriod time.Duration) (bool, error) {
	args := m.Called(cluster, terminationGracePeriod)
	return args.Bool(0), args.Error(1)
}
