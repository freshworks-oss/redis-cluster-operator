package k8sutil

import (
	"context"

	"github.com/freshworks/redis-cluster-operator/metrics"
	utils "github.com/freshworks/redis-cluster-operator/utils/commonutils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IConfigMapControl defines the interface that uses to create, update, and delete ConfigMaps.
type IConfigMapControl interface {
	// CreateConfigMap creates a ConfigMap in a DistributedRedisCluster.
	CreateConfigMap(*corev1.ConfigMap) error
	// UpdateConfigMap updates a ConfigMap in a DistributedRedisCluster.
	UpdateConfigMap(*corev1.ConfigMap) error
	// DeleteConfigMap deletes a ConfigMap in a DistributedRedisCluster.
	DeleteConfigMap(*corev1.ConfigMap) error
	// GetConfigMap get ConfigMap in a DistributedRedisCluster.
	GetConfigMap(namespace, name string) (*corev1.ConfigMap, error)
}

type ConfigMapController struct {
	client  client.Client
	metrics metrics.Recorder
}

// NewRealConfigMapControl creates a concrete implementation of the
// IConfigMapControl.
func NewConfigMapController(client client.Client, metrics metrics.Recorder) IConfigMapControl {
	return &ConfigMapController{client: client, metrics: metrics}
}

// CreateConfigMap implement the IConfigMapControl.Interface.
func (s *ConfigMapController) CreateConfigMap(cm *corev1.ConfigMap) error {
	err := s.client.Create(context.TODO(), cm)
	utils.RecordK8sMetrics(cm.Namespace, "ConfigMap", cm.Name, "CREATE", err, s.metrics)
	return err
}

// UpdateConfigMap implement the IConfigMapControl.Interface.
func (s *ConfigMapController) UpdateConfigMap(cm *corev1.ConfigMap) error {
	err := s.client.Update(context.TODO(), cm)
	utils.RecordK8sMetrics(cm.Namespace, "ConfigMap", cm.Name, "UPDATE", err, s.metrics)
	return err
}

// DeleteConfigMap implement the IConfigMapControl.Interface.
func (s *ConfigMapController) DeleteConfigMap(cm *corev1.ConfigMap) error {
	err := s.client.Delete(context.TODO(), cm)
	utils.RecordK8sMetrics(cm.Namespace, "ConfigMap", cm.Name, "DELETE", err, s.metrics)
	return err
}

// GetConfigMap implement the IConfigMapControl.Interface.
func (s *ConfigMapController) GetConfigMap(namespace, name string) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{}
	err := s.client.Get(context.TODO(), types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, cm)
	utils.RecordK8sMetrics(namespace, "ConfigMap", name, "GET", err, s.metrics)
	return cm, err
}
