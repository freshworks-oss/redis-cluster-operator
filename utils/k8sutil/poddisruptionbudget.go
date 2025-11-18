package k8sutil

import (
	"context"

	"github.com/freshworks/redis-cluster-operator/metrics"
	utils "github.com/freshworks/redis-cluster-operator/utils/commonutils"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IPodDisruptionBudgetControl defines the interface that uses to create, update, and delete PodDisruptionBudgets.
type IPodDisruptionBudgetControl interface {
	// CreatePodDisruptionBudget creates a PodDisruptionBudget in a DistributedRedisCluster.
	CreatePodDisruptionBudget(*policyv1.PodDisruptionBudget) error
	// UpdatePodDisruptionBudget updates a PodDisruptionBudget in a DistributedRedisCluster.
	UpdatePodDisruptionBudget(*policyv1.PodDisruptionBudget) error
	// DeletePodDisruptionBudget deletes a PodDisruptionBudget in a DistributedRedisCluster.
	DeletePodDisruptionBudget(*policyv1.PodDisruptionBudget) error
	DeletePodDisruptionBudgetByName(namespace, name string) error
	// GetPodDisruptionBudget get PodDisruptionBudget in a DistributedRedisCluster.
	GetPodDisruptionBudget(namespace, name string) (*policyv1.PodDisruptionBudget, error)
}

type PodDisruptionBudgetController struct {
	client  client.Client
	metrics metrics.Recorder
}

// NewRealPodDisruptionBudgetControl creates a concrete implementation of the
// IPodDisruptionBudgetControl.
func NewPodDisruptionBudgetController(client client.Client, metrics metrics.Recorder) IPodDisruptionBudgetControl {
	return &PodDisruptionBudgetController{client: client, metrics: metrics}
}

// CreatePodDisruptionBudget implement the IPodDisruptionBudgetControl.Interface.
func (s *PodDisruptionBudgetController) CreatePodDisruptionBudget(pb *policyv1.PodDisruptionBudget) error {
	err := s.client.Create(context.TODO(), pb)
	utils.RecordK8sMetrics(pb.Namespace, "PodDisruptionBudget", pb.Name, "CREATE", err, s.metrics)
	return err
}

// UpdatePodDisruptionBudget implement the IPodDisruptionBudgetControl.Interface.
func (s *PodDisruptionBudgetController) UpdatePodDisruptionBudget(pb *policyv1.PodDisruptionBudget) error {
	err := s.client.Update(context.TODO(), pb)
	utils.RecordK8sMetrics(pb.Namespace, "PodDisruptionBudget", pb.Name, "UPDATE", err, s.metrics)
	return err
}

// DeletePodDisruptionBudget implement the IPodDisruptionBudgetControl.Interface.
func (s *PodDisruptionBudgetController) DeletePodDisruptionBudget(pb *policyv1.PodDisruptionBudget) error {
	err := s.client.Delete(context.TODO(), pb)
	utils.RecordK8sMetrics(pb.Namespace, "PodDisruptionBudget", pb.Name, "DELETE", err, s.metrics)
	return err
}

func (s *PodDisruptionBudgetController) DeletePodDisruptionBudgetByName(namespace, name string) error {
	pdb, err := s.GetPodDisruptionBudget(namespace, name)
	if err != nil {
		return err
	}
	return s.DeletePodDisruptionBudget(pdb)
}

// GetPodDisruptionBudget implement the IPodDisruptionBudgetControl.Interface.
func (s *PodDisruptionBudgetController) GetPodDisruptionBudget(namespace, name string) (*policyv1.PodDisruptionBudget, error) {
	pb := &policyv1.PodDisruptionBudget{}
	err := s.client.Get(context.TODO(), types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, pb)
	utils.RecordK8sMetrics(namespace, "PodDisruptionBudget", name, "GET", err, s.metrics)
	return pb, err
}
