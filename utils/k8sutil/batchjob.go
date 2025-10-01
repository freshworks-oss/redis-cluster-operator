package k8sutil

import (
	"context"

	"github.com/freshworks/redis-cluster-operator/metrics"
	utils "github.com/freshworks/redis-cluster-operator/utils/commonutils"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IJobControl definej the interface that usej to create, update, and delete Jobs.
type IJobControl interface {
	CreateJob(*batchv1.Job) error
	UpdateJob(*batchv1.Job) error
	DeleteJob(*batchv1.Job) error
	GetJob(namespace, name string) (*batchv1.Job, error)
	ListJobByLabels(namespace string, labs client.MatchingLabels) (*batchv1.JobList, error)
}

type JobController struct {
	client  client.Client
	metrics metrics.Recorder
}

// NewRealJobControl creates a concrete implementation of the
// IJobControl.
func NewJobController(client client.Client, metrics metrics.Recorder) IJobControl {
	return &JobController{client: client, metrics: metrics}
}

// CreateJob implement the IJobControl.Interface.
func (j *JobController) CreateJob(job *batchv1.Job) error {
	err := j.client.Create(context.TODO(), job)
	utils.RecordK8sMetrics(job.Namespace, "Job", job.Name, "CREATE", err, j.metrics)
	return err
}

// UpdateJob implement the IJobControl.Interface.
func (j *JobController) UpdateJob(job *batchv1.Job) error {
	err := j.client.Update(context.TODO(), job)
	utils.RecordK8sMetrics(job.Namespace, "Job", job.Name, "UPDATE", err, j.metrics)
	return err
}

// DeleteJob implement the IJobControl.Interface.
func (j *JobController) DeleteJob(job *batchv1.Job) error {
	err := j.client.Delete(context.TODO(), job)
	utils.RecordK8sMetrics(job.Namespace, "Job", job.Name, "DELETE", err, j.metrics)
	return err
}

// GetJob implement the IJobControl.Interface.
func (j *JobController) GetJob(namespace, name string) (*batchv1.Job, error) {
	job := &batchv1.Job{}
	err := j.client.Get(context.TODO(), types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, job)
	utils.RecordK8sMetrics(namespace, "Job", name, "GET", err, j.metrics)
	return job, err
}

func (j *JobController) ListJobByLabels(namespace string, labs client.MatchingLabels) (*batchv1.JobList, error) {
	jobList := &batchv1.JobList{}
	opts := []client.ListOption{
		client.InNamespace(namespace),
		labs,
	}
	err := j.client.List(context.TODO(), jobList, opts...)
	utils.RecordK8sMetrics(namespace, "Job", "NA", "GETJOBBYLABELS", err, j.metrics)
	if err != nil {
		return nil, err
	}

	return jobList, nil
}
