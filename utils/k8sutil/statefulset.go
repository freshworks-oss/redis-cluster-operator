package k8sutil

import (
	"context"

	"github.com/freshworks/redis-cluster-operator/metrics"
	utils "github.com/freshworks/redis-cluster-operator/utils/commonutils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IStatefulSetControl defines the interface that uses to create, update, and delete StatefulSets.
type IStatefulSetControl interface {
	// CreateStatefulSet creates a StatefulSet in a DistributedRedisCluster.
	CreateStatefulSet(*appsv1.StatefulSet) error
	// UpdateStatefulSet updates a StatefulSet in a DistributedRedisCluster.
	UpdateStatefulSet(*appsv1.StatefulSet) error
	// DeleteStatefulSet deletes a StatefulSet in a DistributedRedisCluster.
	DeleteStatefulSet(*appsv1.StatefulSet) error
	DeleteStatefulSetByName(namespace, name string) error
	// GetStatefulSet get StatefulSet in a DistributedRedisCluster.
	GetStatefulSet(namespace, name string) (*appsv1.StatefulSet, error)
	ListStatefulSetByLabels(namespace string, labels map[string]string) (*appsv1.StatefulSetList, error)
	// GetStatefulSetPods will retrieve the pods managed by a given StatefulSet.
	GetStatefulSetPods(namespace, name string) (*corev1.PodList, error)
	GetStatefulSetPodsByLabels(namespace string, labels map[string]string) (*corev1.PodList, error)
}

type stateFulSetController struct {
	client  client.Client
	metrics metrics.Recorder
}

// NewRealStatefulSetControl creates a concrete implementation of the
// IStatefulSetControl.
func NewStatefulSetController(client client.Client, metrics metrics.Recorder) IStatefulSetControl {
	return &stateFulSetController{client: client, metrics: metrics}
}

// CreateStatefulSet implement the IStatefulSetControl.Interface.
func (s *stateFulSetController) CreateStatefulSet(ss *appsv1.StatefulSet) error {
	err := s.client.Create(context.TODO(), ss)
	utils.RecordK8sMetrics(ss.Namespace, "StatefulSet", ss.Name, "CREATE", err, s.metrics)
	return err
}

// UpdateStatefulSet implement the IStatefulSetControl.Interface.
func (s *stateFulSetController) UpdateStatefulSet(ss *appsv1.StatefulSet) error {
	err := s.client.Update(context.TODO(), ss)
	utils.RecordK8sMetrics(ss.Namespace, "StatefulSet", ss.Name, "UPDATE", err, s.metrics)
	return err
}

// DeleteStatefulSet implement the IStatefulSetControl.Interface.
func (s *stateFulSetController) DeleteStatefulSet(ss *appsv1.StatefulSet) error {
	err := s.client.Delete(context.TODO(), ss)
	utils.RecordK8sMetrics(ss.Namespace, "StatefulSet", ss.Name, "DELETE", err, s.metrics)
	return err
}

// GetStatefulSet implement the IStatefulSetControl.Interface.
func (s *stateFulSetController) GetStatefulSet(namespace, name string) (*appsv1.StatefulSet, error) {
	statefulSet := &appsv1.StatefulSet{}
	err := s.client.Get(context.TODO(), types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, statefulSet)
	utils.RecordK8sMetrics(namespace, "StatefulSet", name, "GET", err, s.metrics)
	return statefulSet, err
}

func (s *stateFulSetController) DeleteStatefulSetByName(namespace, name string) error {
	sts, err := s.GetStatefulSet(namespace, name)
	if err != nil {
		return err
	}
	return s.DeleteStatefulSet(sts)
}

// GetStatefulSetPods implement the IStatefulSetControl.Interface.
func (s *stateFulSetController) GetStatefulSetPods(namespace, name string) (*corev1.PodList, error) {
	statefulSet, err := s.GetStatefulSet(namespace, name)
	if err != nil {
		return nil, err
	}

	match := make(client.MatchingLabels)
	for k, v := range statefulSet.Spec.Selector.MatchLabels {
		match[k] = v
	}
	foundPods := &corev1.PodList{}
	err = s.client.List(context.TODO(), foundPods, client.InNamespace(namespace), match)
	utils.RecordK8sMetrics(namespace, "StatefulSet", name, "GETPODS", err, s.metrics)
	return foundPods, err
}

// GetStatefulSetPodsByLabels implement the IStatefulSetControl.Interface.
func (s *stateFulSetController) GetStatefulSetPodsByLabels(namespace string, labels map[string]string) (*corev1.PodList, error) {
	foundPods := &corev1.PodList{}
	err := s.client.List(context.TODO(), foundPods, client.InNamespace(namespace), client.MatchingLabels(labels))
	utils.RecordK8sMetrics(namespace, "StatefulSet", "NA", "GETPODSBYLABELS", err, s.metrics)
	return foundPods, err
}

func (s *stateFulSetController) ListStatefulSetByLabels(namespace string, labels map[string]string) (*appsv1.StatefulSetList, error) {
	foundSts := &appsv1.StatefulSetList{}
	err := s.client.List(context.TODO(), foundSts, client.InNamespace(namespace), client.MatchingLabels(labels))
	utils.RecordK8sMetrics(namespace, "StatefulSet", "NA", "GETSTSBYLABELS", err, s.metrics)
	return foundSts, err
}
