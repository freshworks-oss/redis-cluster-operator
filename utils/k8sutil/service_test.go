package k8sutil

import (
	"context"
	"testing"

	"github.com/freshworks/redis-cluster-operator/utils/testutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestNewServiceController(t *testing.T) {
	fakeClient := fake.NewClientBuilder().Build()
	metrics := &testutil.MockMetrics{}

	controller := NewServiceController(fakeClient, metrics)
	if controller == nil {
		t.Error("NewServiceController() returned nil")
	}

	// Verify it implements IServiceControl interface
	_, ok := controller.(IServiceControl)
	if !ok {
		t.Error("NewServiceController() does not implement IServiceControl interface")
	}
}

func TestServiceController_CreateService(t *testing.T) {
	tests := []struct {
		name        string
		service     *corev1.Service
		expectError bool
	}{
		{
			name: "Create service successfully",
			service: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-service",
					Namespace: "test-namespace",
				},
				Spec: corev1.ServiceSpec{
					Selector: map[string]string{
						"app": "test",
					},
					Ports: []corev1.ServicePort{
						{
							Name:       "http",
							Port:       80,
							TargetPort: intstr.FromInt(8080),
							Protocol:   corev1.ProtocolTCP,
						},
					},
					Type: corev1.ServiceTypeClusterIP,
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().Build()
			controller := NewServiceController(fakeClient, &testutil.MockMetrics{})

			err := controller.CreateService(tt.service)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError {
				// Verify service was created
				retrievedService := &corev1.Service{}
				getErr := fakeClient.Get(context.TODO(), types.NamespacedName{
					Namespace: tt.service.Namespace,
					Name:      tt.service.Name,
				}, retrievedService)
				if getErr != nil {
					t.Errorf("Failed to retrieve service after creation: %v", getErr)
				}
			}
		})
	}
}

func TestServiceController_UpdateService(t *testing.T) {
	existingService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-service",
			Namespace: "test-namespace",
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "test",
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       80,
					TargetPort: intstr.FromInt(8080),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	tests := []struct {
		name        string
		service     *corev1.Service
		expectError bool
	}{
		{
			name: "Update existing service successfully",
			service: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-service",
					Namespace: "test-namespace",
				},
				Spec: corev1.ServiceSpec{
					Selector: map[string]string{
						"app": "updated",
					},
					Ports: []corev1.ServicePort{
						{
							Name:       "http",
							Port:       8080,
							TargetPort: intstr.FromInt(8080),
							Protocol:   corev1.ProtocolTCP,
						},
					},
					Type: corev1.ServiceTypeNodePort,
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().WithObjects(existingService).Build()
			controller := NewServiceController(fakeClient, &testutil.MockMetrics{})

			err := controller.UpdateService(tt.service)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestServiceController_DeleteService(t *testing.T) {
	existingService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "service-to-delete",
			Namespace: "test-namespace",
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "test",
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       80,
					TargetPort: intstr.FromInt(8080),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	tests := []struct {
		name        string
		service     *corev1.Service
		expectError bool
	}{
		{
			name:        "Delete existing service successfully",
			service:     existingService,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().WithObjects(existingService).Build()
			controller := NewServiceController(fakeClient, &testutil.MockMetrics{})

			err := controller.DeleteService(tt.service)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestServiceController_DeleteServiceByName(t *testing.T) {
	existingService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "service-to-delete",
			Namespace: "test-namespace",
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "test",
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       80,
					TargetPort: intstr.FromInt(8080),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	tests := []struct {
		name        string
		namespace   string
		serviceName string
		expectError bool
	}{
		{
			name:        "Delete existing service by name successfully",
			namespace:   "test-namespace",
			serviceName: "service-to-delete",
			expectError: false,
		},
		{
			name:        "Delete non-existing service by name returns error",
			namespace:   "test-namespace",
			serviceName: "non-existing-service",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fakeClient client.Client
			if tt.serviceName == "service-to-delete" {
				fakeClient = fake.NewClientBuilder().WithObjects(existingService).Build()
			} else {
				fakeClient = fake.NewClientBuilder().Build()
			}

			controller := NewServiceController(fakeClient, &testutil.MockMetrics{})

			err := controller.DeleteServiceByName(tt.namespace, tt.serviceName)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestServiceController_GetService(t *testing.T) {
	existingService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-service",
			Namespace: "test-namespace",
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "test",
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       80,
					TargetPort: intstr.FromInt(8080),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	tests := []struct {
		name        string
		namespace   string
		serviceName string
		expectError bool
		expectFound bool
	}{
		{
			name:        "Get existing service successfully",
			namespace:   "test-namespace",
			serviceName: "existing-service",
			expectError: false,
			expectFound: true,
		},
		{
			name:        "Get non-existing service returns error",
			namespace:   "test-namespace",
			serviceName: "non-existing-service",
			expectError: true,
			expectFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fakeClient client.Client
			if tt.expectFound {
				fakeClient = fake.NewClientBuilder().WithObjects(existingService).Build()
			} else {
				fakeClient = fake.NewClientBuilder().Build()
			}

			controller := NewServiceController(fakeClient, &testutil.MockMetrics{})

			service, err := controller.GetService(tt.namespace, tt.serviceName)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if tt.expectFound && service == nil {
				t.Error("Expected service to be found but got nil")
			}
			if tt.expectFound && service.Name != tt.serviceName {
				t.Errorf("Expected service name %s, got %s", tt.serviceName, service.Name)
			}
		})
	}
}
