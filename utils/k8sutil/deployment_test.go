package k8sutil

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/dinesh-murugiah/rediscluster-operator/utils/testutil"
)

func TestNewDeploymentController(t *testing.T) {
	fakeClient := fake.NewClientBuilder().Build()
	metrics := &testutil.MockMetrics{}

	controller := NewDeploymentController(fakeClient, metrics)
	if controller == nil {
		t.Error("NewDeploymentController() returned nil")
	}

	// Verify it implements IDeploymentControl interface
	_, ok := controller.(IDeploymentControl)
	if !ok {
		t.Error("NewDeploymentController() does not implement IDeploymentControl interface")
	}
}

func TestDeploymentController_CreateDeployment(t *testing.T) {
	tests := []struct {
		name        string
		deployment  *appsv1.Deployment
		expectError bool
	}{
		{
			name: "Create deployment successfully",
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "test-namespace",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: int32Ptr(3),
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "test",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app": "test",
							},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "test-container",
									Image: "test-image",
								},
							},
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
			controller := NewDeploymentController(fakeClient, &testutil.MockMetrics{})

			err := controller.CreateDeployment(tt.deployment)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError {
				// Verify deployment was created
				retrievedDeployment := &appsv1.Deployment{}
				getErr := fakeClient.Get(context.TODO(), types.NamespacedName{
					Namespace: tt.deployment.Namespace,
					Name:      tt.deployment.Name,
				}, retrievedDeployment)
				if getErr != nil {
					t.Errorf("Failed to retrieve deployment after creation: %v", getErr)
				}
			}
		})
	}
}

func TestDeploymentController_UpdateDeployment(t *testing.T) {
	existingDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-deployment",
			Namespace: "test-namespace",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(3),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "test",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "test-image",
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name        string
		deployment  *appsv1.Deployment
		expectError bool
	}{
		{
			name: "Update existing deployment successfully",
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-deployment",
					Namespace: "test-namespace",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: int32Ptr(5),
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "test",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app": "test",
							},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "updated-container",
									Image: "updated-image",
								},
							},
						},
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().WithObjects(existingDeployment).Build()
			controller := NewDeploymentController(fakeClient, &testutil.MockMetrics{})

			err := controller.UpdateDeployment(tt.deployment)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestDeploymentController_DeleteDeployment(t *testing.T) {
	existingDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-to-delete",
			Namespace: "test-namespace",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(3),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "test",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "test-image",
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name        string
		deployment  *appsv1.Deployment
		expectError bool
	}{
		{
			name:        "Delete existing deployment successfully",
			deployment:  existingDeployment,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().WithObjects(existingDeployment).Build()
			controller := NewDeploymentController(fakeClient, &testutil.MockMetrics{})

			err := controller.DeleteDeployment(tt.deployment)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestDeploymentController_GetDeployment(t *testing.T) {
	existingDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-deployment",
			Namespace: "test-namespace",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(3),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "test",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "test-image",
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name           string
		namespace      string
		deploymentName string
		expectError    bool
		expectFound    bool
	}{
		{
			name:           "Get existing deployment successfully",
			namespace:      "test-namespace",
			deploymentName: "existing-deployment",
			expectError:    false,
			expectFound:    true,
		},
		{
			name:           "Get non-existing deployment returns error",
			namespace:      "test-namespace",
			deploymentName: "non-existing-deployment",
			expectError:    true,
			expectFound:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fakeClient client.Client
			if tt.expectFound {
				fakeClient = fake.NewClientBuilder().WithObjects(existingDeployment).Build()
			} else {
				fakeClient = fake.NewClientBuilder().Build()
			}

			controller := NewDeploymentController(fakeClient, &testutil.MockMetrics{})

			deployment, err := controller.GetDeployment(tt.namespace, tt.deploymentName)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if tt.expectFound && deployment == nil {
				t.Error("Expected deployment to be found but got nil")
			}
			if tt.expectFound && deployment.Name != tt.deploymentName {
				t.Errorf("Expected deployment name %s, got %s", tt.deploymentName, deployment.Name)
			}
		})
	}
}

func TestDeploymentController_ListDeploymentByLabels(t *testing.T) {
	deployment1 := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment1",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"app": "test",
				"env": "dev",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(3),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "test",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "test-image",
						},
					},
				},
			},
		},
	}

	deployment2 := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment2",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"app": "test",
				"env": "prod",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(5),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "test",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "test-image",
						},
					},
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
			name:      "List deployments with matching labels",
			namespace: "test-namespace",
			labels: client.MatchingLabels{
				"app": "test",
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name:      "List deployments with specific env label",
			namespace: "test-namespace",
			labels: client.MatchingLabels{
				"env": "dev",
			},
			expectedCount: 1,
			expectError:   false,
		},
		{
			name:      "List deployments with no matching labels",
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
			fakeClient := fake.NewClientBuilder().WithObjects(deployment1, deployment2).Build()
			controller := NewDeploymentController(fakeClient, &testutil.MockMetrics{})

			deploymentList, err := controller.ListDeploymentByLabels(tt.namespace, tt.labels)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if !tt.expectError && len(deploymentList.Items) != tt.expectedCount {
				t.Errorf("Expected %d deployments, got %d", tt.expectedCount, len(deploymentList.Items))
			}
		})
	}
}

// Helper function to create int32 pointer
func int32Ptr(i int32) *int32 {
	return &i
}
