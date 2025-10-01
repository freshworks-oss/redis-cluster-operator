package k8sutil

import (
	"context"

	"github.com/freshworks/redis-cluster-operator/metrics"
	utils "github.com/freshworks/redis-cluster-operator/utils/commonutils"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IDeploymentControl define the interface that used to create, update, and delete Deployments.
type IDeploymentControl interface {
	CreateDeployment(*appsv1.Deployment) error
	UpdateDeployment(*appsv1.Deployment) error
	DeleteDeployment(*appsv1.Deployment) error
	GetDeployment(namespace, name string) (*appsv1.Deployment, error)
	ListDeploymentByLabels(namespace string, labs client.MatchingLabels) (*appsv1.DeploymentList, error)
}

type DeploymentController struct {
	client  client.Client
	metrics metrics.Recorder
}

// NewRealDeploymentControl creates a concrete implementation of the
// IDeploymentControl.
func NewDeploymentController(client client.Client, metrics metrics.Recorder) IDeploymentControl {
	return &DeploymentController{client: client, metrics: metrics}
}

// CreateDeployment implement the IDeploymentControl.Interface.
func (d *DeploymentController) CreateDeployment(deployment *appsv1.Deployment) error {
	err := d.client.Create(context.TODO(), deployment)
	utils.RecordK8sMetrics(deployment.Namespace, "Deployment", deployment.Name, "CREATE", err, d.metrics)
	return err
}

// UpdateDeployment implement the IDeploymentControl.Interface.
func (d *DeploymentController) UpdateDeployment(deployment *appsv1.Deployment) error {
	err := d.client.Update(context.TODO(), deployment)
	utils.RecordK8sMetrics(deployment.Namespace, "Deployment", deployment.Name, "UPDATE", err, d.metrics)
	return err
}

// DeleteDeployment implement the IDeploymentControl.Interface.
func (d *DeploymentController) DeleteDeployment(deployment *appsv1.Deployment) error {
	err := d.client.Delete(context.TODO(), deployment)
	utils.RecordK8sMetrics(deployment.Namespace, "Deployment", deployment.Name, "DELETE", err, d.metrics)
	return err
}

// GetDeployment implement the IDeploymentControl.Interface.
func (d *DeploymentController) GetDeployment(namespace, name string) (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{}
	err := d.client.Get(context.TODO(), types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, deployment)
	utils.RecordK8sMetrics(namespace, "Deployment", name, "GET", err, d.metrics)
	return deployment, err
}

func (d *DeploymentController) ListDeploymentByLabels(namespace string, labs client.MatchingLabels) (*appsv1.DeploymentList, error) {
	deploymentList := &appsv1.DeploymentList{}
	opts := []client.ListOption{
		client.InNamespace(namespace),
		labs,
	}
	err := d.client.List(context.TODO(), deploymentList, opts...)
	utils.RecordK8sMetrics(namespace, "Deployment", "", "GETDEPLOYMENTBYLABELS", err, d.metrics)
	if err != nil {
		return nil, err
	}

	return deploymentList, nil
}
