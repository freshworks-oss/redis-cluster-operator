package k8sutil

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"

	"github.com/dinesh-murugiah/rediscluster-operator/metrics"
	utils "github.com/dinesh-murugiah/rediscluster-operator/utils/commonutils"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	redisv1alpha1 "github.com/dinesh-murugiah/rediscluster-operator/api/v1alpha1"
)

// ICustomResource defines the interface that uses to update cr status
type ICustomResource interface {
	// UpdateCRStatus update the RedisCluster status
	UpdateCRStatus(runtime.Object) error
	UpdateCR(runtime.Object) error
	GetRedisClusterBackup(namespace, name string) (*redisv1alpha1.RedisClusterBackup, error)
	GetDistributedRedisCluster(namespace, name string) (*redisv1alpha1.DistributedRedisCluster, error)
	UpdateCrdAnnotations(crd *redisv1alpha1.DistributedRedisCluster, annotationKey string, annotationvalue string) error
}

type clusterControl struct {
	client  client.Client
	metrics metrics.Recorder
}

// NewCRControl creates a concrete implementation of the
// ICustomResource.
func NewCRControl(client client.Client, metrics metrics.Recorder) ICustomResource {
	return &clusterControl{client: client, metrics: metrics}
}

func (c *clusterControl) UpdateCRStatus(obj runtime.Object) error {
	err := c.client.Status().Update(context.Background(), obj.(client.Object))
	utils.RecordK8sMetrics(obj.(client.Object).GetNamespace(), "RedisClusterCR", obj.(client.Object).GetName(), "UPDATECRSTATUS", err, c.metrics)
	return err
}

func (c *clusterControl) UpdateCR(obj runtime.Object) error {
	err := c.client.Update(context.Background(), obj.(client.Object))
	utils.RecordK8sMetrics(obj.(client.Object).GetNamespace(), "RedisClusterCR", obj.(client.Object).GetName(), "UPDATECR", err, c.metrics)
	return err
}

func (c *clusterControl) GetRedisClusterBackup(namespace, name string) (*redisv1alpha1.RedisClusterBackup, error) {
	backup := &redisv1alpha1.RedisClusterBackup{}
	err := c.client.Get(context.TODO(), types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, backup)
	utils.RecordK8sMetrics(namespace, "RedisClusterBackup", name, "GET", err, c.metrics)
	if err != nil {
		return nil, err
	}
	return backup, nil
}

func (c *clusterControl) GetDistributedRedisCluster(namespace, name string) (*redisv1alpha1.DistributedRedisCluster, error) {
	drc := &redisv1alpha1.DistributedRedisCluster{}
	err := c.client.Get(context.TODO(), types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, drc)
	utils.RecordK8sMetrics(namespace, "DistributedRedisCluster", name, "GET", err, c.metrics)
	if err != nil {
		return nil, err
	}
	return drc, nil
}

func (c *clusterControl) UpdateCrdAnnotations(crd *redisv1alpha1.DistributedRedisCluster, annotationKey string, annotationvalue string) error {

	if _, ok := crd.ObjectMeta.Annotations[annotationKey]; !ok {
		return fmt.Errorf("annotation %s Notfound", annotationKey)
	}

	if crd.ObjectMeta.Annotations[annotationKey] == annotationvalue {
		return nil
	} else {
		crd.ObjectMeta.Annotations[annotationKey] = annotationvalue
		return c.UpdateCR(crd)
	}
}
