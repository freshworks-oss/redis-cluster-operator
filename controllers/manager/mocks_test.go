package manager

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/mock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	redisv1alpha1 "github.com/freshworks/redis-cluster-operator/api/v1alpha1"
	"github.com/freshworks/redis-cluster-operator/utils/redisutil"
)

// Shared mock implementations for all manager tests

type MockStatefulSetControl struct {
	mock.Mock
}

func (m *MockStatefulSetControl) GetStatefulSet(namespace, name string) (*appsv1.StatefulSet, error) {
	args := m.Called(namespace, name)
	return args.Get(0).(*appsv1.StatefulSet), args.Error(1)
}

func (m *MockStatefulSetControl) CreateStatefulSet(statefulset *appsv1.StatefulSet) error {
	args := m.Called(statefulset)
	return args.Error(0)
}

func (m *MockStatefulSetControl) UpdateStatefulSet(statefulset *appsv1.StatefulSet) error {
	args := m.Called(statefulset)
	return args.Error(0)
}

func (m *MockStatefulSetControl) DeleteStatefulSet(statefulset *appsv1.StatefulSet) error {
	args := m.Called(statefulset)
	return args.Error(0)
}

func (m *MockStatefulSetControl) DeleteStatefulSetByName(namespace, name string) error {
	args := m.Called(namespace, name)
	return args.Error(0)
}

func (m *MockStatefulSetControl) ListStatefulSetByLabels(namespace string, labels map[string]string) (*appsv1.StatefulSetList, error) {
	args := m.Called(namespace, labels)
	return args.Get(0).(*appsv1.StatefulSetList), args.Error(1)
}

func (m *MockStatefulSetControl) GetStatefulSetPods(namespace, name string) (*corev1.PodList, error) {
	args := m.Called(namespace, name)
	return args.Get(0).(*corev1.PodList), args.Error(1)
}

func (m *MockStatefulSetControl) GetStatefulSetPodsByLabels(namespace string, labels map[string]string) (*corev1.PodList, error) {
	args := m.Called(namespace, labels)
	return args.Get(0).(*corev1.PodList), args.Error(1)
}

type MockServiceControl struct {
	mock.Mock
}

func (m *MockServiceControl) GetService(namespace, name string) (*corev1.Service, error) {
	args := m.Called(namespace, name)
	return args.Get(0).(*corev1.Service), args.Error(1)
}

func (m *MockServiceControl) CreateService(service *corev1.Service) error {
	args := m.Called(service)
	return args.Error(0)
}

func (m *MockServiceControl) UpdateService(service *corev1.Service) error {
	args := m.Called(service)
	return args.Error(0)
}

func (m *MockServiceControl) DeleteService(service *corev1.Service) error {
	args := m.Called(service)
	return args.Error(0)
}

func (m *MockServiceControl) DeleteServiceByName(namespace, name string) error {
	args := m.Called(namespace, name)
	return args.Error(0)
}

type MockConfigMapControl struct {
	mock.Mock
}

func (m *MockConfigMapControl) GetConfigMap(namespace, name string) (*corev1.ConfigMap, error) {
	args := m.Called(namespace, name)
	return args.Get(0).(*corev1.ConfigMap), args.Error(1)
}

func (m *MockConfigMapControl) CreateConfigMap(configmap *corev1.ConfigMap) error {
	args := m.Called(configmap)
	return args.Error(0)
}

func (m *MockConfigMapControl) UpdateConfigMap(configmap *corev1.ConfigMap) error {
	args := m.Called(configmap)
	return args.Error(0)
}

func (m *MockConfigMapControl) DeleteConfigMap(configmap *corev1.ConfigMap) error {
	args := m.Called(configmap)
	return args.Error(0)
}

type MockPodDisruptionBudgetControl struct {
	mock.Mock
}

func (m *MockPodDisruptionBudgetControl) GetPodDisruptionBudget(namespace, name string) (*policyv1.PodDisruptionBudget, error) {
	args := m.Called(namespace, name)
	return args.Get(0).(*policyv1.PodDisruptionBudget), args.Error(1)
}

func (m *MockPodDisruptionBudgetControl) CreatePodDisruptionBudget(pdb *policyv1.PodDisruptionBudget) error {
	args := m.Called(pdb)
	return args.Error(0)
}

func (m *MockPodDisruptionBudgetControl) UpdatePodDisruptionBudget(pdb *policyv1.PodDisruptionBudget) error {
	args := m.Called(pdb)
	return args.Error(0)
}

func (m *MockPodDisruptionBudgetControl) DeletePodDisruptionBudget(pdb *policyv1.PodDisruptionBudget) error {
	args := m.Called(pdb)
	return args.Error(0)
}

func (m *MockPodDisruptionBudgetControl) DeletePodDisruptionBudgetByName(namespace, name string) error {
	args := m.Called(namespace, name)
	return args.Error(0)
}

type MockCustomResource struct {
	mock.Mock
}

func (m *MockCustomResource) UpdateCRStatus(obj runtime.Object) error {
	args := m.Called(obj)
	return args.Error(0)
}

func (m *MockCustomResource) UpdateCR(obj runtime.Object) error {
	args := m.Called(obj)
	return args.Error(0)
}

func (m *MockCustomResource) GetRedisClusterBackup(namespace, name string) (*redisv1alpha1.RedisClusterBackup, error) {
	args := m.Called(namespace, name)
	return args.Get(0).(*redisv1alpha1.RedisClusterBackup), args.Error(1)
}

func (m *MockCustomResource) GetDistributedRedisCluster(namespace, name string) (*redisv1alpha1.DistributedRedisCluster, error) {
	args := m.Called(namespace, name)
	return args.Get(0).(*redisv1alpha1.DistributedRedisCluster), args.Error(1)
}

func (m *MockCustomResource) UpdateCrdAnnotations(crd *redisv1alpha1.DistributedRedisCluster, annotationKey string, annotationvalue string) error {
	args := m.Called(crd, annotationKey, annotationvalue)
	return args.Error(0)
}

// MockAdmin provides a minimal implementation of IAdmin for testing
// We only implement the methods we actually need for our tests
type MockAdmin struct {
	mock.Mock
}

// Implement all required methods from IAdmin interface
func (m *MockAdmin) Connections() redisutil.IAdminConnections {
	args := m.Called()
	return args.Get(0).(redisutil.IAdminConnections)
}

func (m *MockAdmin) Close(ctx context.Context) {
	m.Called(ctx)
}

func (m *MockAdmin) GetClusterInfos(ctx context.Context, namespace string, name string) (*redisutil.ClusterInfos, error) {
	args := m.Called(ctx, namespace, name)
	return args.Get(0).(*redisutil.ClusterInfos), args.Error(1)
}

func (m *MockAdmin) ClusterManagerNodeIsEmpty(ctx context.Context, namespace string, name string) (bool, error) {
	args := m.Called(ctx, namespace, name)
	return args.Bool(0), args.Error(1)
}

func (m *MockAdmin) SetConfigEpoch(ctx context.Context, namespace string, name string) error {
	args := m.Called(ctx, namespace, name)
	return args.Error(0)
}

func (m *MockAdmin) SetConfigIfNeed(ctx context.Context, newConfig map[string]string, namespace string, name string) error {
	args := m.Called(ctx, newConfig, namespace, name)
	return args.Error(0)
}

func (m *MockAdmin) GetAllConfig(ctx context.Context, c *redis.Client, addr string, namespace string, name string) (map[string]string, error) {
	args := m.Called(ctx, c, addr, namespace, name)
	return args.Get(0).(map[string]string), args.Error(1)
}

func (m *MockAdmin) AttachNodeToCluster(ctx context.Context, addr string, namespace string, name string) error {
	args := m.Called(ctx, addr, namespace, name)
	return args.Error(0)
}

func (m *MockAdmin) AttachSlaveToMaster(ctx context.Context, slave *redisutil.Node, masterID string, namespace string, name string) error {
	args := m.Called(ctx, slave, masterID, namespace, name)
	return args.Error(0)
}

func (m *MockAdmin) DetachSlave(ctx context.Context, slave *redisutil.Node, namespace string, name string) error {
	args := m.Called(ctx, slave, namespace, name)
	return args.Error(0)
}

func (m *MockAdmin) ForgetNode(ctx context.Context, id string, namespace string, name string) error {
	args := m.Called(ctx, id, namespace, name)
	return args.Error(0)
}

func (m *MockAdmin) SetSlots(ctx context.Context, addr string, action string, slots []redisutil.Slot, nodeID string, namespace string, name string) error {
	args := m.Called(ctx, addr, action, slots, nodeID, namespace, name)
	return args.Error(0)
}

func (m *MockAdmin) AddSlots(ctx context.Context, addr string, slots []redisutil.Slot) error {
	args := m.Called(ctx, addr, slots)
	return args.Error(0)
}

func (m *MockAdmin) SetSlot(ctx context.Context, addr string, action string, slot redisutil.Slot, nodeID string, namespace string, name string) error {
	args := m.Called(ctx, addr, action, slot, nodeID, namespace, name)
	return args.Error(0)
}

func (m *MockAdmin) MigrateKeys(ctx context.Context, addr string, dest *redisutil.Node, slots []redisutil.Slot, batch, timeout int, replace bool, namespace string, name string) (int, error) {
	args := m.Called(ctx, addr, dest, slots, batch, timeout, replace, namespace, name)
	return args.Int(0), args.Error(1)
}

func (m *MockAdmin) MigrateKeysInSlot(ctx context.Context, addr string, dest *redisutil.Node, slot redisutil.Slot, batch int, timeout int, replace bool, namespace string, name string) (int, error) {
	args := m.Called(ctx, addr, dest, slot, batch, timeout, replace, namespace, name)
	return args.Int(0), args.Error(1)
}

func (m *MockAdmin) FlushAndReset(ctx context.Context, addr string, mode string, namespace string, name string) error {
	args := m.Called(ctx, addr, mode, namespace, name)
	return args.Error(0)
}

func (m *MockAdmin) GetHashMaxSlot() redisutil.Slot {
	args := m.Called()
	return args.Get(0).(redisutil.Slot)
}

func (m *MockAdmin) ResetPassword(ctx context.Context, newPassword string, namespace string, name string) error {
	args := m.Called(ctx, newPassword, namespace, name)
	return args.Error(0)
}

func (m *MockAdmin) FailoverSlaveToMaster(ctx context.Context, slave *redisutil.Node, option redisutil.RedisFailover, namespace string, name string) error {
	args := m.Called(ctx, slave, option, namespace, name)
	return args.Error(0)
}

func (m *MockAdmin) GetReplicationStatus(ip string, ctx context.Context, namespace string, name string) (string, redisutil.ReplicationInfo, error) {
	args := m.Called(ip, ctx, namespace, name)
	return args.String(0), args.Get(1).(redisutil.ReplicationInfo), args.Error(2)
}

func (m *MockAdmin) GetInfoReplication(ipPort string, ctx context.Context, namespace string, name string) (redisutil.ReplicationInfo, error) {
	args := m.Called(ipPort, ctx, namespace, name)
	return args.Get(0).(redisutil.ReplicationInfo), args.Error(1)
}

func (m *MockAdmin) DrainClientsInOldMaster(ctx context.Context, oldMaster *redisutil.Node, namespace string, name string) error {
	args := m.Called(ctx, oldMaster, namespace, name)
	return args.Error(0)
}

type MockPodControl struct {
	mock.Mock
}

func (m *MockPodControl) DeletePodByName(namespace, name string) error {
	args := m.Called(namespace, name)
	return args.Error(0)
}

type MockMetricsRecorder struct {
	mock.Mock
}

func (m *MockMetricsRecorder) RecordK8sOperation(namespace, kind, object, operation, status, err string) {
	m.Called(namespace, kind, object, operation, status, err)
}

func (m *MockMetricsRecorder) RecordRedisOperation(namespace, name, operation, ip, status, err string) {
	m.Called(namespace, name, operation, ip, status, err)
}

func (m *MockMetricsRecorder) ReconcileCluster(namespace, name, operation, function, status, err string) {
	m.Called(namespace, name, operation, function, status, err)
}

func (m *MockMetricsRecorder) ReconcileClusterGauge(namespace, name, function string, value float64) {
	m.Called(namespace, name, function, value)
}

func (m *MockMetricsRecorder) SetClusterOK(namespace, name string, val float64) {
	m.Called(namespace, name, val)
}

// Note: We don't mock client.Client directly as it's complex and not essential for our unit tests.
// Instead, we focus on mocking the k8sutil interfaces that our code actually uses.

// Enhanced mock builders for easier test setup

// MockBuilder provides a fluent interface for setting up mocks
type MockBuilder struct {
	mockSTS       *MockStatefulSetControl
	mockSvc       *MockServiceControl
	mockConfigMap *MockConfigMapControl
	mockPDB       *MockPodDisruptionBudgetControl
	mockCR        *MockCustomResource
	mockMetrics   *MockMetricsRecorder
}

// NewMockBuilder creates a new mock builder with all mocks initialized
func NewMockBuilder() *MockBuilder {
	return &MockBuilder{
		mockSTS:       &MockStatefulSetControl{},
		mockSvc:       &MockServiceControl{},
		mockConfigMap: &MockConfigMapControl{},
		mockPDB:       &MockPodDisruptionBudgetControl{},
		mockCR:        &MockCustomResource{},
		mockMetrics:   &MockMetricsRecorder{},
	}
}

// WithStatefulSetNotFound sets up mocks for StatefulSets that don't exist
func (mb *MockBuilder) WithStatefulSetNotFound(namespace, name string) *MockBuilder {
	mb.mockSTS.On("GetStatefulSet", namespace, name).Return(
		(*appsv1.StatefulSet)(nil),
		errors.NewNotFound(appsv1.Resource("statefulset"), name))
	return mb
}

// WithStatefulSetExists sets up mocks for StatefulSets that exist
func (mb *MockBuilder) WithStatefulSetExists(namespace, name string, replicas int32, readyReplicas int32) *MockBuilder {
	mb.mockSTS.On("GetStatefulSet", namespace, name).Return(
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: &replicas,
			},
			Status: appsv1.StatefulSetStatus{
				ReadyReplicas: readyReplicas,
			},
		}, nil)
	return mb
}

// WithStatefulSetCreateSuccess sets up mocks for successful StatefulSet creation
func (mb *MockBuilder) WithStatefulSetCreateSuccess() *MockBuilder {
	mb.mockSTS.On("CreateStatefulSet", mock.AnythingOfType("*v1.StatefulSet")).Return(nil)
	return mb
}

// WithStatefulSetCreateError sets up mocks for failed StatefulSet creation
func (mb *MockBuilder) WithStatefulSetCreateError(err error) *MockBuilder {
	mb.mockSTS.On("CreateStatefulSet", mock.AnythingOfType("*v1.StatefulSet")).Return(err)
	return mb
}

// WithStatefulSetUpdateSuccess sets up mocks for successful StatefulSet updates
func (mb *MockBuilder) WithStatefulSetUpdateSuccess() *MockBuilder {
	mb.mockSTS.On("UpdateStatefulSet", mock.AnythingOfType("*v1.StatefulSet")).Return(nil)
	return mb
}

// WithStatefulSetUpdateError sets up mocks for failed StatefulSet updates
func (mb *MockBuilder) WithStatefulSetUpdateError(err error) *MockBuilder {
	mb.mockSTS.On("UpdateStatefulSet", mock.AnythingOfType("*v1.StatefulSet")).Return(err)
	return mb
}

// WithServiceNotFound sets up mocks for Services that don't exist
func (mb *MockBuilder) WithServiceNotFound(namespace, name string) *MockBuilder {
	mb.mockSvc.On("GetService", namespace, name).Return(
		(*corev1.Service)(nil),
		errors.NewNotFound(corev1.Resource("service"), name))
	return mb
}

// WithServiceExists sets up mocks for Services that exist
func (mb *MockBuilder) WithServiceExists(namespace, name string) *MockBuilder {
	mb.mockSvc.On("GetService", namespace, name).Return(
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}, nil)
	return mb
}

// WithServiceCreateSuccess sets up mocks for successful Service creation
func (mb *MockBuilder) WithServiceCreateSuccess() *MockBuilder {
	mb.mockSvc.On("CreateService", mock.AnythingOfType("*v1.Service")).Return(nil)
	return mb
}

// WithServiceCreateError sets up mocks for failed Service creation
func (mb *MockBuilder) WithServiceCreateError(err error) *MockBuilder {
	mb.mockSvc.On("CreateService", mock.AnythingOfType("*v1.Service")).Return(err)
	return mb
}

// WithConfigMapNotFound sets up mocks for ConfigMaps that don't exist
func (mb *MockBuilder) WithConfigMapNotFound(namespace, name string) *MockBuilder {
	mb.mockConfigMap.On("GetConfigMap", namespace, name).Return(
		(*corev1.ConfigMap)(nil),
		errors.NewNotFound(corev1.Resource("configmap"), name))
	return mb
}

// WithConfigMapExists sets up mocks for ConfigMaps that exist
func (mb *MockBuilder) WithConfigMapExists(namespace, name string, data map[string]string) *MockBuilder {
	mb.mockConfigMap.On("GetConfigMap", namespace, name).Return(
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Data: data,
		}, nil)
	return mb
}

// WithConfigMapCreateSuccess sets up mocks for successful ConfigMap creation
func (mb *MockBuilder) WithConfigMapCreateSuccess() *MockBuilder {
	mb.mockConfigMap.On("CreateConfigMap", mock.AnythingOfType("*v1.ConfigMap")).Return(nil)
	return mb
}

// WithConfigMapUpdateSuccess sets up mocks for successful ConfigMap updates
func (mb *MockBuilder) WithConfigMapUpdateSuccess() *MockBuilder {
	mb.mockConfigMap.On("UpdateConfigMap", mock.AnythingOfType("*v1.ConfigMap")).Return(nil)
	return mb
}

// WithPDBNotFound sets up mocks for PodDisruptionBudgets that don't exist
func (mb *MockBuilder) WithPDBNotFound(namespace, name string) *MockBuilder {
	mb.mockPDB.On("GetPodDisruptionBudget", namespace, name).Return(
		(*policyv1.PodDisruptionBudget)(nil),
		errors.NewNotFound(policyv1.Resource("poddisruptionbudget"), name))
	return mb
}

// WithPDBExists sets up mocks for PodDisruptionBudgets that exist
func (mb *MockBuilder) WithPDBExists(namespace, name string) *MockBuilder {
	mb.mockPDB.On("GetPodDisruptionBudget", namespace, name).Return(
		&policyv1.PodDisruptionBudget{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}, nil)
	return mb
}

// WithPDBCreateSuccess sets up mocks for successful PodDisruptionBudget creation
func (mb *MockBuilder) WithPDBCreateSuccess() *MockBuilder {
	mb.mockPDB.On("CreatePodDisruptionBudget", mock.AnythingOfType("*v1.PodDisruptionBudget")).Return(nil)
	return mb
}

// WithMetricsRecording sets up mocks for metrics recording
func (mb *MockBuilder) WithMetricsRecording() *MockBuilder {
	mb.mockMetrics.On("RecordK8sOperation", mock.AnythingOfType("string"), mock.AnythingOfType("string"),
		mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"),
		mock.AnythingOfType("string")).Maybe()
	mb.mockMetrics.On("ReconcileCluster", mock.AnythingOfType("string"), mock.AnythingOfType("string"),
		mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"),
		mock.AnythingOfType("string")).Maybe()
	return mb
}

// GetMocks returns all the mock objects
func (mb *MockBuilder) GetMocks() (*MockStatefulSetControl, *MockServiceControl, *MockConfigMapControl,
	*MockPodDisruptionBudgetControl, *MockCustomResource, *MockMetricsRecorder) {
	return mb.mockSTS, mb.mockSvc, mb.mockConfigMap, mb.mockPDB, mb.mockCR, mb.mockMetrics
}

// AssertAllExpectations asserts expectations on all mocks
func (mb *MockBuilder) AssertAllExpectations(t *testing.T) {
	mb.mockSTS.AssertExpectations(t)
	mb.mockSvc.AssertExpectations(t)
	mb.mockConfigMap.AssertExpectations(t)
	mb.mockPDB.AssertExpectations(t)
	mb.mockCR.AssertExpectations(t)
	mb.mockMetrics.AssertExpectations(t)
}

// TestClusterBuilder provides a fluent interface for creating test clusters
type TestClusterBuilder struct {
	cluster *redisv1alpha1.DistributedRedisCluster
}

// NewTestClusterBuilder creates a new test cluster builder with default values
func NewTestClusterBuilder(name, namespace string) *TestClusterBuilder {
	return &TestClusterBuilder{
		cluster: &redisv1alpha1.DistributedRedisCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: redisv1alpha1.DistributedRedisClusterSpec{
				MasterSize:      3,
				ClusterReplicas: 1,
				Image:           "redis:7.4.4-alpine",
				ServiceName:     name + "-service",
				PdbEnabled:      true,
				Config: map[string]string{
					"appendonly":                "yes",
					"auto-aof-rewrite-min-size": "67108864",
				},
			},
			Status: redisv1alpha1.DistributedRedisClusterStatus{
				NumberOfMaster: 3,
				SecretsVer: map[string]string{
					"secret1": "v1",
				},
			},
		},
	}
}

// WithMasterSize sets the master size
func (tcb *TestClusterBuilder) WithMasterSize(size int32) *TestClusterBuilder {
	tcb.cluster.Spec.MasterSize = size
	tcb.cluster.Status.NumberOfMaster = size
	return tcb
}

// WithReplicas sets the number of replicas per master
func (tcb *TestClusterBuilder) WithReplicas(replicas int32) *TestClusterBuilder {
	tcb.cluster.Spec.ClusterReplicas = replicas
	return tcb
}

// WithImage sets the Redis image
func (tcb *TestClusterBuilder) WithImage(image string) *TestClusterBuilder {
	tcb.cluster.Spec.Image = image
	return tcb
}

// WithPDBEnabled sets whether PDB is enabled
func (tcb *TestClusterBuilder) WithPDBEnabled(enabled bool) *TestClusterBuilder {
	tcb.cluster.Spec.PdbEnabled = enabled
	return tcb
}

// WithConfig sets the Redis configuration
func (tcb *TestClusterBuilder) WithConfig(config map[string]string) *TestClusterBuilder {
	tcb.cluster.Spec.Config = config
	return tcb
}

// WithSecretsVersion sets the secrets version
func (tcb *TestClusterBuilder) WithSecretsVersion(versions map[string]string) *TestClusterBuilder {
	tcb.cluster.Status.SecretsVer = versions
	return tcb
}

// WithServiceName sets the service name
func (tcb *TestClusterBuilder) WithServiceName(serviceName string) *TestClusterBuilder {
	tcb.cluster.Spec.ServiceName = serviceName
	return tcb
}

// WithStatusMasterCount sets the status master count (can be different from spec for error testing)
func (tcb *TestClusterBuilder) WithStatusMasterCount(count int32) *TestClusterBuilder {
	tcb.cluster.Status.NumberOfMaster = count
	return tcb
}

// Build returns the constructed cluster
func (tcb *TestClusterBuilder) Build() *redisv1alpha1.DistributedRedisCluster {
	return tcb.cluster
}

// MockClient is a simple mock for client.Client interface
type MockClient struct {
	mock.Mock
}

func (m *MockClient) Get(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) error {
	args := m.Called(ctx, key, obj)
	return args.Error(0)
}

func (m *MockClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	args := m.Called(ctx, list, opts)
	return args.Error(0)
}

func (m *MockClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	args := m.Called(ctx, obj, patch, opts)
	return args.Error(0)
}

func (m *MockClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) Status() client.StatusWriter {
	args := m.Called()
	return args.Get(0).(client.StatusWriter)
}

func (m *MockClient) Scheme() *runtime.Scheme {
	args := m.Called()
	return args.Get(0).(*runtime.Scheme)
}

func (m *MockClient) RESTMapper() meta.RESTMapper {
	args := m.Called()
	return args.Get(0).(meta.RESTMapper)
}

func (m *MockClient) GroupVersionKindFor(obj runtime.Object) (schema.GroupVersionKind, error) {
	args := m.Called(obj)
	return args.Get(0).(schema.GroupVersionKind), args.Error(1)
}

func (m *MockClient) IsObjectNamespaced(obj runtime.Object) (bool, error) {
	args := m.Called(obj)
	return args.Bool(0), args.Error(1)
}
