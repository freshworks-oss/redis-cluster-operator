package k8sutil

import (
	"context"
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/dinesh-murugiah/rediscluster-operator/utils/testutil"
)

func TestNewJobController(t *testing.T) {
	fakeClient := fake.NewClientBuilder().Build()
	metrics := &testutil.MockMetrics{}

	controller := NewJobController(fakeClient, metrics)
	if controller == nil {
		t.Error("NewJobController() returned nil")
	}

	// Verify it implements IJobControl interface
	_, ok := controller.(IJobControl)
	if !ok {
		t.Error("NewJobController() does not implement IJobControl interface")
	}
}

func TestJobController_CreateJob(t *testing.T) {
	tests := []struct {
		name        string
		job         *batchv1.Job
		expectError bool
	}{
		{
			name: "Create job successfully",
			job: &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job",
					Namespace: "test-namespace",
				},
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "test-container",
									Image: "test-image",
								},
							},
							RestartPolicy: corev1.RestartPolicyNever,
						},
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().Build()
			controller := NewJobController(fakeClient, &testutil.MockMetrics{})

			err := controller.CreateJob(tt.job)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError {
				// Verify job was created
				retrievedJob := &batchv1.Job{}
				getErr := fakeClient.Get(context.TODO(), types.NamespacedName{
					Namespace: tt.job.Namespace,
					Name:      tt.job.Name,
				}, retrievedJob)
				if getErr != nil {
					t.Errorf("Failed to retrieve job after creation: %v", getErr)
				}
			}
		})
	}
}

func TestJobController_UpdateJob(t *testing.T) {
	existingJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-job",
			Namespace: "test-namespace",
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "test-image",
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}

	tests := []struct {
		name        string
		job         *batchv1.Job
		expectError bool
	}{
		{
			name: "Update existing job successfully",
			job: &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-job",
					Namespace: "test-namespace",
				},
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "updated-container",
									Image: "updated-image",
								},
							},
							RestartPolicy: corev1.RestartPolicyNever,
						},
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().WithObjects(existingJob).Build()
			controller := NewJobController(fakeClient, &testutil.MockMetrics{})

			err := controller.UpdateJob(tt.job)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestJobController_DeleteJob(t *testing.T) {
	existingJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "job-to-delete",
			Namespace: "test-namespace",
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "test-image",
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}

	tests := []struct {
		name        string
		job         *batchv1.Job
		expectError bool
	}{
		{
			name:        "Delete existing job successfully",
			job:         existingJob,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().WithObjects(existingJob).Build()
			controller := NewJobController(fakeClient, &testutil.MockMetrics{})

			err := controller.DeleteJob(tt.job)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestJobController_GetJob(t *testing.T) {
	existingJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-job",
			Namespace: "test-namespace",
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "test-image",
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}

	tests := []struct {
		name        string
		namespace   string
		jobName     string
		expectError bool
		expectFound bool
	}{
		{
			name:        "Get existing job successfully",
			namespace:   "test-namespace",
			jobName:     "existing-job",
			expectError: false,
			expectFound: true,
		},
		{
			name:        "Get non-existing job returns error",
			namespace:   "test-namespace",
			jobName:     "non-existing-job",
			expectError: true,
			expectFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fakeClient client.Client
			if tt.expectFound {
				fakeClient = fake.NewClientBuilder().WithObjects(existingJob).Build()
			} else {
				fakeClient = fake.NewClientBuilder().Build()
			}

			controller := NewJobController(fakeClient, &testutil.MockMetrics{})

			job, err := controller.GetJob(tt.namespace, tt.jobName)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if tt.expectFound && job == nil {
				t.Error("Expected job to be found but got nil")
			}
			if tt.expectFound && job.Name != tt.jobName {
				t.Errorf("Expected job name %s, got %s", tt.jobName, job.Name)
			}
		})
	}
}

func TestJobController_ListJobByLabels(t *testing.T) {
	job1 := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "job1",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"app": "test",
				"env": "dev",
			},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "test-image",
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}

	job2 := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "job2",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"app": "test",
				"env": "prod",
			},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "test-image",
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}

	tests := []struct {
		name          string
		namespace     string
		labels        client.MatchingLabels
		expectedCount int
		expectError   bool
	}{
		{
			name:      "List jobs with matching labels",
			namespace: "test-namespace",
			labels: client.MatchingLabels{
				"app": "test",
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name:      "List jobs with specific env label",
			namespace: "test-namespace",
			labels: client.MatchingLabels{
				"env": "dev",
			},
			expectedCount: 1,
			expectError:   false,
		},
		{
			name:      "List jobs with no matching labels",
			namespace: "test-namespace",
			labels: client.MatchingLabels{
				"nonexistent": "value",
			},
			expectedCount: 0,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().WithObjects(job1, job2).Build()
			controller := NewJobController(fakeClient, &testutil.MockMetrics{})

			jobList, err := controller.ListJobByLabels(tt.namespace, tt.labels)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if !tt.expectError && len(jobList.Items) != tt.expectedCount {
				t.Errorf("Expected %d jobs, got %d", tt.expectedCount, len(jobList.Items))
			}
		})
	}
}
