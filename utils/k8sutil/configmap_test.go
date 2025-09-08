package k8sutil

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/dinesh-murugiah/rediscluster-operator/utils/testutil"
)

func TestNewConfigMapController(t *testing.T) {
	fakeClient := fake.NewClientBuilder().Build()
	metrics := &testutil.MockMetrics{}

	controller := NewConfigMapController(fakeClient, metrics)
	if controller == nil {
		t.Error("NewConfigMapController() returned nil")
	}

	// Verify it implements IConfigMapControl interface
	_, ok := controller.(IConfigMapControl)
	if !ok {
		t.Error("NewConfigMapController() does not implement IConfigMapControl interface")
	}
}

func TestConfigMapController_CreateConfigMap(t *testing.T) {
	tests := []struct {
		name        string
		configMap   *corev1.ConfigMap
		expectError bool
	}{
		{
			name: "Create configmap successfully",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-configmap",
					Namespace: "test-namespace",
				},
				Data: map[string]string{
					"key1": "value1",
					"key2": "value2",
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().Build()
			controller := NewConfigMapController(fakeClient, &testutil.MockMetrics{})

			err := controller.CreateConfigMap(tt.configMap)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError {
				// Verify configmap was created
				retrievedCM := &corev1.ConfigMap{}
				getErr := fakeClient.Get(context.TODO(), types.NamespacedName{
					Namespace: tt.configMap.Namespace,
					Name:      tt.configMap.Name,
				}, retrievedCM)
				if getErr != nil {
					t.Errorf("Failed to retrieve configmap after creation: %v", getErr)
				}
				if retrievedCM.Data["key1"] != "value1" {
					t.Error("ConfigMap data was not set correctly")
				}
			}
		})
	}
}

func TestConfigMapController_UpdateConfigMap(t *testing.T) {
	existingCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-configmap",
			Namespace: "test-namespace",
		},
		Data: map[string]string{
			"key1": "old-value",
		},
	}

	tests := []struct {
		name        string
		configMap   *corev1.ConfigMap
		expectError bool
	}{
		{
			name: "Update existing configmap successfully",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-configmap",
					Namespace: "test-namespace",
				},
				Data: map[string]string{
					"key1": "new-value",
					"key2": "added-value",
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().WithObjects(existingCM).Build()
			controller := NewConfigMapController(fakeClient, &testutil.MockMetrics{})

			err := controller.UpdateConfigMap(tt.configMap)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestConfigMapController_DeleteConfigMap(t *testing.T) {
	existingCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "configmap-to-delete",
			Namespace: "test-namespace",
		},
		Data: map[string]string{
			"key1": "value1",
		},
	}

	tests := []struct {
		name        string
		configMap   *corev1.ConfigMap
		expectError bool
	}{
		{
			name:        "Delete existing configmap successfully",
			configMap:   existingCM,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().WithObjects(existingCM).Build()
			controller := NewConfigMapController(fakeClient, &testutil.MockMetrics{})

			err := controller.DeleteConfigMap(tt.configMap)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestConfigMapController_GetConfigMap(t *testing.T) {
	existingCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-configmap",
			Namespace: "test-namespace",
		},
		Data: map[string]string{
			"key1": "value1",
		},
	}

	tests := []struct {
		name        string
		namespace   string
		cmName      string
		expectError bool
		expectFound bool
	}{
		{
			name:        "Get existing configmap successfully",
			namespace:   "test-namespace",
			cmName:      "existing-configmap",
			expectError: false,
			expectFound: true,
		},
		{
			name:        "Get non-existing configmap returns error",
			namespace:   "test-namespace",
			cmName:      "non-existing-configmap",
			expectError: true,
			expectFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fakeClient client.Client
			if tt.expectFound {
				fakeClient = fake.NewClientBuilder().WithObjects(existingCM).Build()
			} else {
				fakeClient = fake.NewClientBuilder().Build()
			}

			controller := NewConfigMapController(fakeClient, &testutil.MockMetrics{})

			cm, err := controller.GetConfigMap(tt.namespace, tt.cmName)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if tt.expectFound && cm == nil {
				t.Error("Expected configmap to be found but got nil")
			}
			if tt.expectFound && cm.Name != tt.cmName {
				t.Errorf("Expected configmap name %s, got %s", tt.cmName, cm.Name)
			}
		})
	}
}
