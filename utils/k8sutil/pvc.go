package k8sutil

import (
	"context"

	"github.com/dinesh-murugiah/rediscluster-operator/metrics"
	utils "github.com/dinesh-murugiah/rediscluster-operator/utils/commonutils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IPvcControl defines the interface that uses to create, update, and delete PersistentVolumeClaim.
type IPvcControl interface {
	DeletePvc(claim *corev1.PersistentVolumeClaim) error
	CreatePvc(claim *corev1.PersistentVolumeClaim) error
	DeletePvcByLabels(namespace string, labels map[string]string) error
	GetPvc(namespace, name string) (*corev1.PersistentVolumeClaim, error)
}

type pvcController struct {
	client  client.Client
	metrics metrics.Recorder
}

// NewPvcController creates a concrete implementation of the
// IPvcControl.
func NewPvcController(client client.Client, metrics metrics.Recorder) IPvcControl {
	return &pvcController{client: client, metrics: metrics}
}

// DeletePvc implement the IPvcControl.Interface.
func (s *pvcController) DeletePvc(pvc *corev1.PersistentVolumeClaim) error {
	err := s.client.Delete(context.TODO(), pvc)
	utils.RecordK8sMetrics(pvc.Namespace, "PersistentVolumeClaim", pvc.Name, "DELETE", err, s.metrics)
	return err
}

// CreatePvc implement the IPvcControl.Interface.
func (s *pvcController) CreatePvc(claim *corev1.PersistentVolumeClaim) error {
	err := s.client.Create(context.TODO(), claim)
	utils.RecordK8sMetrics(claim.Namespace, "PersistentVolumeClaim", claim.Name, "CREATE", err, s.metrics)
	return err
}

func (s *pvcController) DeletePvcByLabels(namespace string, labels map[string]string) error {
	foundPvcs := &corev1.PersistentVolumeClaimList{}
	err := s.client.List(context.TODO(), foundPvcs, client.InNamespace(namespace), client.MatchingLabels(labels))
	utils.RecordK8sMetrics(namespace, "PersistentVolumeClaim", "NA", "LISTPVCBYLABELS", err, s.metrics)
	if err != nil {
		return err
	}

	for _, pvc := range foundPvcs.Items {
		if err := s.client.Delete(context.TODO(), &pvc); err != nil {
			utils.RecordK8sMetrics(namespace, "PersistentVolumeClaim", pvc.Name, "DELETE", err, s.metrics)
			return err
		}
	}
	return nil
}

// GetPvc implement the IPvcControl.Interface.
func (s *pvcController) GetPvc(namespace, name string) (*corev1.PersistentVolumeClaim, error) {
	pvc := &corev1.PersistentVolumeClaim{}
	err := s.client.Get(context.TODO(), types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, pvc)
	utils.RecordK8sMetrics(namespace, "PersistentVolumeClaim", name, "GET", err, s.metrics)
	return pvc, err
}
