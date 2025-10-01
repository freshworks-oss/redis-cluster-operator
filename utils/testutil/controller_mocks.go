package testutil

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/runtime"

	redisv1alpha1 "github.com/dinesh-murugiah/rediscluster-operator/api/v1alpha1"
)

// MockEnsurer provides a testable implementation of clustermanger.IEnsureResource
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

// MockChecker provides a testable implementation of clustermanger.ICheck
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

// MockHealer provides a testable implementation of manager.IHeal
type MockHealer struct {
	mock.Mock
}

func (m *MockHealer) FixTerminatingPods(cluster *redisv1alpha1.DistributedRedisCluster, gracePeriod time.Duration) (bool, error) {
	args := m.Called(cluster, gracePeriod)
	return args.Bool(0), args.Error(1)
}

func (m *MockHealer) Heal(cluster *redisv1alpha1.DistributedRedisCluster, infos interface{}, admin interface{}, ctx context.Context) (bool, bool, error) {
	args := m.Called(cluster, infos, admin, ctx)
	return args.Bool(0), args.Bool(1), args.Error(2)
}

// MockStatefulSetController provides a testable implementation of k8sutil.IStatefulSetControl
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
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*appsv1.StatefulSet), args.Error(1)
}

func (m *MockStatefulSetController) DeleteStatefulSetByName(namespace, name string) error {
	args := m.Called(namespace, name)
	return args.Error(0)
}

func (m *MockStatefulSetController) GetStatefulSetPodsByLabels(namespace string, labels map[string]string) (*corev1.PodList, error) {
	args := m.Called(namespace, labels)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*corev1.PodList), args.Error(1)
}

func (m *MockStatefulSetController) ListStatefulSetByLabels(namespace string, labels map[string]string) (*appsv1.StatefulSetList, error) {
	args := m.Called(namespace, labels)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*appsv1.StatefulSetList), args.Error(1)
}

func (m *MockStatefulSetController) GetStatefulSetPods(namespace, stsName string) (*corev1.PodList, error) {
	args := m.Called(namespace, stsName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*corev1.PodList), args.Error(1)
}

// MockServiceController provides a testable implementation of k8sutil.IServiceControl
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
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*corev1.Service), args.Error(1)
}

func (m *MockServiceController) DeleteServiceByName(namespace, name string) error {
	args := m.Called(namespace, name)
	return args.Error(0)
}

// MockPdbController provides a testable implementation of k8sutil.IPodDisruptionBudgetControl
type MockPdbController struct {
	mock.Mock
}

func (m *MockPdbController) CreatePodDisruptionBudget(pdb *policyv1.PodDisruptionBudget) error {
	args := m.Called(pdb)
	return args.Error(0)
}

func (m *MockPdbController) UpdatePodDisruptionBudget(pdb *policyv1.PodDisruptionBudget) error {
	args := m.Called(pdb)
	return args.Error(0)
}

func (m *MockPdbController) DeletePodDisruptionBudget(pdb *policyv1.PodDisruptionBudget) error {
	args := m.Called(pdb)
	return args.Error(0)
}

func (m *MockPdbController) GetPodDisruptionBudget(namespace, name string) (*policyv1.PodDisruptionBudget, error) {
	args := m.Called(namespace, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*policyv1.PodDisruptionBudget), args.Error(1)
}

func (m *MockPdbController) DeletePodDisruptionBudgetByName(namespace, name string) error {
	args := m.Called(namespace, name)
	return args.Error(0)
}

// MockPvcController provides a testable implementation of k8sutil.IPvcControl
type MockPvcController struct {
	mock.Mock
}

func (m *MockPvcController) CreatePvc(pvc *corev1.PersistentVolumeClaim) error {
	args := m.Called(pvc)
	return args.Error(0)
}

func (m *MockPvcController) UpdatePvc(pvc *corev1.PersistentVolumeClaim) error {
	args := m.Called(pvc)
	return args.Error(0)
}

func (m *MockPvcController) DeletePvc(pvc *corev1.PersistentVolumeClaim) error {
	args := m.Called(pvc)
	return args.Error(0)
}

func (m *MockPvcController) GetPvc(namespace, name string) (*corev1.PersistentVolumeClaim, error) {
	args := m.Called(namespace, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*corev1.PersistentVolumeClaim), args.Error(1)
}

func (m *MockPvcController) DeletePvcByLabels(namespace string, labels map[string]string) error {
	args := m.Called(namespace, labels)
	return args.Error(0)
}

// MockCrController provides a testable implementation of k8sutil.ICustomResource
type MockCrController struct {
	mock.Mock
}

func (m *MockCrController) UpdateCRStatus(obj runtime.Object) error {
	args := m.Called(obj)
	return args.Error(0)
}

func (m *MockCrController) UpdateCR(obj runtime.Object) error {
	args := m.Called(obj)
	return args.Error(0)
}

func (m *MockCrController) GetRedisClusterBackup(namespace, name string) (*redisv1alpha1.RedisClusterBackup, error) {
	args := m.Called(namespace, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*redisv1alpha1.RedisClusterBackup), args.Error(1)
}

func (m *MockCrController) GetDistributedRedisCluster(namespace, name string) (*redisv1alpha1.DistributedRedisCluster, error) {
	args := m.Called(namespace, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*redisv1alpha1.DistributedRedisCluster), args.Error(1)
}

func (m *MockCrController) UpdateCrdAnnotations(crd *redisv1alpha1.DistributedRedisCluster, annotationKey string, annotationvalue string) error {
	args := m.Called(crd, annotationKey, annotationvalue)
	return args.Error(0)
}

// MockPodController provides a testable implementation of k8sutil.IPodControl
type MockPodController struct {
	mock.Mock
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
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*corev1.Pod), args.Error(1)
}

func (m *MockPodController) DeletePodByName(namespace, name string) error {
	args := m.Called(namespace, name)
	return args.Error(0)
}

func (m *MockPodController) UpdatePodLabels(pod *corev1.Pod, key, value string) error {
	args := m.Called(pod, key, value)
	return args.Error(0)
}

func (m *MockPodController) UpdatePodAnnotations(pod *corev1.Pod, key, value string) error {
	args := m.Called(pod, key, value)
	return args.Error(0)
}

func (m *MockPodController) ListPodsWithLabel(namespace string, labelkey string, labelvalue string) (*corev1.PodList, error) {
	args := m.Called(namespace, labelkey, labelvalue)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*corev1.PodList), args.Error(1)
}

// MockNodeController provides a testable implementation of k8sutil.INodeControl
type MockNodeController struct {
	mock.Mock
}

func (m *MockNodeController) GetNode(name string) (*corev1.Node, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*corev1.Node), args.Error(1)
}

func (m *MockNodeController) ListNodes() (*corev1.NodeList, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*corev1.NodeList), args.Error(1)
}

func (m *MockNodeController) CheckZoneAvailable(zone string) (bool, error) {
	args := m.Called(zone)
	return args.Bool(0), args.Error(1)
}

func (m *MockNodeController) GetZoneLabel(node *corev1.Node) string {
	args := m.Called(node)
	return args.String(0)
}

// MockExecer provides a testable implementation of exec.IExec
type MockExecer struct {
	mock.Mock
}

func (m *MockExecer) ExecCommandInPodSet(podSet []*corev1.Pod, cmd ...string) error {
	args := m.Called(podSet, cmd)
	return args.Error(0)
}

// Note: MockRedisAdmin has been moved to avoid circular imports
// Redis-related mocks should be defined in the redisutil package test files directly
