package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	redisv1alpha1 "github.com/freshworks/redis-cluster-operator/api/v1alpha1"
)

// MockStatefulSetController provides a unified mock for all StatefulSet operations
// Consolidates MockStatefulSetControl and MockStatefulSetController
type MockStatefulSetController struct {
	mock.Mock
}

func (m *MockStatefulSetController) CreateStatefulSet(sts *appsv1.StatefulSet) error {
	args := m.Called(sts)
	return args.Error(0)
}

func (m *MockStatefulSetController) UpdateStatefulSet(sts *appsv1.StatefulSet) error {
	args := m.Called(sts)
	return args.Error(0)
}

func (m *MockStatefulSetController) DeleteStatefulSet(sts *appsv1.StatefulSet) error {
	args := m.Called(sts)
	return args.Error(0)
}

func (m *MockStatefulSetController) GetStatefulSet(namespace, name string) (*appsv1.StatefulSet, error) {
	args := m.Called(namespace, name)
	return args.Get(0).(*appsv1.StatefulSet), args.Error(1)
}

func (m *MockStatefulSetController) DeleteStatefulSetByName(namespace, name string) error {
	args := m.Called(namespace, name)
	return args.Error(0)
}

func (m *MockStatefulSetController) GetStatefulSetPodsByLabels(namespace string, labels map[string]string) (*corev1.PodList, error) {
	args := m.Called(namespace, labels)
	return args.Get(0).(*corev1.PodList), args.Error(1)
}

func (m *MockStatefulSetController) ListStatefulSetByLabels(namespace string, labels map[string]string) (*appsv1.StatefulSetList, error) {
	args := m.Called(namespace, labels)
	return args.Get(0).(*appsv1.StatefulSetList), args.Error(1)
}

func (m *MockStatefulSetController) GetStatefulSetPods(namespace, stsName string) (*corev1.PodList, error) {
	args := m.Called(namespace, stsName)
	return args.Get(0).(*corev1.PodList), args.Error(1)
}

// MockServiceController provides a unified mock for all Service operations
type MockServiceController struct {
	mock.Mock
}

func (m *MockServiceController) CreateService(svc *corev1.Service) error {
	args := m.Called(svc)
	return args.Error(0)
}

func (m *MockServiceController) UpdateService(svc *corev1.Service) error {
	args := m.Called(svc)
	return args.Error(0)
}

func (m *MockServiceController) DeleteService(svc *corev1.Service) error {
	args := m.Called(svc)
	return args.Error(0)
}

func (m *MockServiceController) GetService(namespace, name string) (*corev1.Service, error) {
	args := m.Called(namespace, name)
	return args.Get(0).(*corev1.Service), args.Error(1)
}

func (m *MockServiceController) DeleteServiceByName(namespace, name string) error {
	args := m.Called(namespace, name)
	return args.Error(0)
}

func (m *MockServiceController) ListServiceByLabels(namespace string, labels map[string]string) (*corev1.ServiceList, error) {
	args := m.Called(namespace, labels)
	return args.Get(0).(*corev1.ServiceList), args.Error(1)
}

// MockPodController provides a unified mock for all Pod operations
// Consolidates MockPodControl and MockPodController
type MockPodController struct {
	mock.Mock
}

func (m *MockPodController) DeletePodByName(namespace, name string) error {
	args := m.Called(namespace, name)
	return args.Error(0)
}

func (m *MockPodController) CreatePod(pod *corev1.Pod) error {
	args := m.Called(pod)
	return args.Error(0)
}

func (m *MockPodController) UpdatePod(pod *corev1.Pod) error {
	args := m.Called(pod)
	return args.Error(0)
}

func (m *MockPodController) DeletePod(pod *corev1.Pod) error {
	args := m.Called(pod)
	return args.Error(0)
}

func (m *MockPodController) GetPod(namespace, name string) (*corev1.Pod, error) {
	args := m.Called(namespace, name)
	return args.Get(0).(*corev1.Pod), args.Error(1)
}

func (m *MockPodController) UpdatePodLabels(pod *corev1.Pod, labelKey, labelValue string) error {
	args := m.Called(pod, labelKey, labelValue)
	return args.Error(0)
}

func (m *MockPodController) UpdatePodAnnotations(pod *corev1.Pod, annotationKey, annotationValue string) error {
	args := m.Called(pod, annotationKey, annotationValue)
	return args.Error(0)
}

func (m *MockPodController) ListPodByLabels(namespace string, labels map[string]string) (*corev1.PodList, error) {
	args := m.Called(namespace, labels)
	return args.Get(0).(*corev1.PodList), args.Error(1)
}

func (m *MockPodController) ListPodsWithLabel(namespace string, labelkey string, labelvalue string) (*corev1.PodList, error) {
	args := m.Called(namespace, labelkey, labelvalue)
	return args.Get(0).(*corev1.PodList), args.Error(1)
}

// MockPodDisruptionBudgetController provides a unified mock for PDB operations
type MockPodDisruptionBudgetController struct {
	mock.Mock
}

func (m *MockPodDisruptionBudgetController) GetPodDisruptionBudget(namespace, name string) (*policyv1.PodDisruptionBudget, error) {
	args := m.Called(namespace, name)
	return args.Get(0).(*policyv1.PodDisruptionBudget), args.Error(1)
}

func (m *MockPodDisruptionBudgetController) CreatePodDisruptionBudget(pdb *policyv1.PodDisruptionBudget) error {
	args := m.Called(pdb)
	return args.Error(0)
}

func (m *MockPodDisruptionBudgetController) UpdatePodDisruptionBudget(pdb *policyv1.PodDisruptionBudget) error {
	args := m.Called(pdb)
	return args.Error(0)
}

func (m *MockPodDisruptionBudgetController) DeletePodDisruptionBudgetByName(namespace, name string) error {
	args := m.Called(namespace, name)
	return args.Error(0)
}

// MockConfigMapController provides a unified mock for ConfigMap operations
type MockConfigMapController struct {
	mock.Mock
}

func (m *MockConfigMapController) GetConfigMap(namespace, name string) (*corev1.ConfigMap, error) {
	args := m.Called(namespace, name)
	return args.Get(0).(*corev1.ConfigMap), args.Error(1)
}

func (m *MockConfigMapController) CreateConfigMap(cm *corev1.ConfigMap) error {
	args := m.Called(cm)
	return args.Error(0)
}

func (m *MockConfigMapController) UpdateConfigMap(cm *corev1.ConfigMap) error {
	args := m.Called(cm)
	return args.Error(0)
}

func (m *MockConfigMapController) DeleteConfigMapByName(namespace, name string) error {
	args := m.Called(namespace, name)
	return args.Error(0)
}

// MockPvcController provides a unified mock for PVC operations
type MockPvcController struct {
	mock.Mock
}

func (m *MockPvcController) GetPvc(namespace, name string) (*corev1.PersistentVolumeClaim, error) {
	args := m.Called(namespace, name)
	return args.Get(0).(*corev1.PersistentVolumeClaim), args.Error(1)
}

func (m *MockPvcController) CreatePvc(pvc *corev1.PersistentVolumeClaim) error {
	args := m.Called(pvc)
	return args.Error(0)
}

func (m *MockPvcController) DeletePvc(pvc *corev1.PersistentVolumeClaim) error {
	args := m.Called(pvc)
	return args.Error(0)
}

func (m *MockPvcController) UpdatePvc(pvc *corev1.PersistentVolumeClaim) error {
	args := m.Called(pvc)
	return args.Error(0)
}

func (m *MockPvcController) DeletePvcByLabels(namespace string, labels map[string]string) error {
	args := m.Called(namespace, labels)
	return args.Error(0)
}

// MockCustomResourceController provides a unified mock for CR operations
type MockCustomResourceController struct {
	mock.Mock
}

func (m *MockCustomResourceController) UpdateCR(obj runtime.Object) error {
	args := m.Called(obj)
	return args.Error(0)
}

func (m *MockCustomResourceController) UpdateCRStatus(obj runtime.Object) error {
	args := m.Called(obj)
	return args.Error(0)
}

func (m *MockCustomResourceController) GetRedisClusterBackup(namespace, name string) (*redisv1alpha1.RedisClusterBackup, error) {
	args := m.Called(namespace, name)
	return args.Get(0).(*redisv1alpha1.RedisClusterBackup), args.Error(1)
}

func (m *MockCustomResourceController) GetDistributedRedisCluster(namespace, name string) (*redisv1alpha1.DistributedRedisCluster, error) {
	args := m.Called(namespace, name)
	return args.Get(0).(*redisv1alpha1.DistributedRedisCluster), args.Error(1)
}

func (m *MockCustomResourceController) UpdateCrdAnnotations(crd *redisv1alpha1.DistributedRedisCluster, annotationKey string, annotationValue string) error {
	args := m.Called(crd, annotationKey, annotationValue)
	return args.Error(0)
}

// MockNodeController provides a unified mock for Node operations
type MockNodeController struct {
	mock.Mock
}

func (m *MockNodeController) GetNode(name string) (*corev1.Node, error) {
	args := m.Called(name)
	return args.Get(0).(*corev1.Node), args.Error(1)
}

func (m *MockNodeController) ListNodes() (*corev1.NodeList, error) {
	args := m.Called()
	return args.Get(0).(*corev1.NodeList), args.Error(1)
}

// MockClient provides a unified mock for controller-runtime client operations
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
