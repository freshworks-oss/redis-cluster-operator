package k8sutil

import (
	"context"

	"github.com/dinesh-murugiah/rediscluster-operator/metrics"
	utils "github.com/dinesh-murugiah/rediscluster-operator/utils/commonutils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IServiceControl defines the interface that uses to create, update, and delete Services.
type IServiceControl interface {
	// CreateService creates a Service in a DistributedRedisCluster.
	CreateService(*corev1.Service) error
	// UpdateService updates a Service in a DistributedRedisCluster.
	UpdateService(*corev1.Service) error
	// DeleteService deletes a Service in a DistributedRedisCluster.
	DeleteService(*corev1.Service) error
	DeleteServiceByName(namespace, name string) error
	// GetService get Service in a DistributedRedisCluster.
	GetService(namespace, name string) (*corev1.Service, error)
}

type serviceController struct {
	client  client.Client
	metrics metrics.Recorder
}

// NewRealServiceControl creates a concrete implementation of the
// IServiceControl.
func NewServiceController(client client.Client, metrics metrics.Recorder) IServiceControl {
	return &serviceController{client: client, metrics: metrics}
}

// CreateService implement the IServiceControl.Interface.
func (s *serviceController) CreateService(svc *corev1.Service) error {
	err := s.client.Create(context.TODO(), svc)
	utils.RecordK8sMetrics(svc.Namespace, "Service", svc.Name, "CREATE", err, s.metrics)
	return err
}

// UpdateService implement the IServiceControl.Interface.
func (s *serviceController) UpdateService(svc *corev1.Service) error {
	err := s.client.Update(context.TODO(), svc)
	utils.RecordK8sMetrics(svc.Namespace, "Service", svc.Name, "UPDATE", err, s.metrics)
	return err
}

// DeleteService implement the IServiceControl.Interface.
func (s *serviceController) DeleteService(svc *corev1.Service) error {
	err := s.client.Delete(context.TODO(), svc)
	utils.RecordK8sMetrics(svc.Namespace, "Service", svc.Name, "DELETE", err, s.metrics)
	return err
}

func (s *serviceController) DeleteServiceByName(namespace, name string) error {
	svc, err := s.GetService(namespace, name)
	if err != nil {
		return err
	}
	return s.DeleteService(svc)
}

// GetService implement the IServiceControl.Interface.
func (s *serviceController) GetService(namespace, name string) (*corev1.Service, error) {
	svc := &corev1.Service{}
	err := s.client.Get(context.TODO(), types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, svc)
	utils.RecordK8sMetrics(namespace, "Service", name, "GET", err, s.metrics)
	return svc, err
}
