package k8sutil

import (
	"context"
	"testing"

	"github.com/dinesh-murugiah/rediscluster-operator/utils/testutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestNewSecretController(t *testing.T) {
	fakeClient := fake.NewClientBuilder().Build()
	metrics := &testutil.MockMetrics{}

	controller := NewSecretController(fakeClient, metrics)
	if controller == nil {
		t.Error("NewSecretController() returned nil")
	}

	// Verify it implements ISecretControl interface
	_, ok := controller.(ISecretControl)
	if !ok {
		t.Error("NewSecretController() does not implement ISecretControl interface")
	}
}

func TestSecretController_CreateSecret(t *testing.T) {
	tests := []struct {
		name        string
		secret      *corev1.Secret
		expectError bool
	}{
		{
			name: "Create secret successfully",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "test-namespace",
				},
				Data: map[string][]byte{
					"username": []byte("testuser"),
					"password": []byte("testpass"),
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().Build()
			controller := NewSecretController(fakeClient, &testutil.MockMetrics{})

			err := controller.CreateSecret(tt.secret)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError {
				// Verify secret was created
				retrievedSecret := &corev1.Secret{}
				getErr := fakeClient.Get(context.TODO(), types.NamespacedName{
					Namespace: tt.secret.Namespace,
					Name:      tt.secret.Name,
				}, retrievedSecret)
				if getErr != nil {
					t.Errorf("Failed to retrieve secret after creation: %v", getErr)
				}
				if string(retrievedSecret.Data["username"]) != "testuser" {
					t.Error("Secret data was not set correctly")
				}
			}
		})
	}
}

func TestSecretController_UpdateSecret(t *testing.T) {
	existingSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-secret",
			Namespace: "test-namespace",
		},
		Data: map[string][]byte{
			"username": []byte("olduser"),
			"password": []byte("oldpass"),
		},
	}

	tests := []struct {
		name        string
		secret      *corev1.Secret
		expectError bool
	}{
		{
			name: "Update existing secret successfully",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-secret",
					Namespace: "test-namespace",
				},
				Data: map[string][]byte{
					"username": []byte("newuser"),
					"password": []byte("newpass"),
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().WithObjects(existingSecret).Build()
			controller := NewSecretController(fakeClient, &testutil.MockMetrics{})

			err := controller.UpdateSecret(tt.secret)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestSecretController_DeleteSecret(t *testing.T) {
	existingSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret-to-delete",
			Namespace: "test-namespace",
		},
		Data: map[string][]byte{
			"key": []byte("value"),
		},
	}

	tests := []struct {
		name        string
		secret      *corev1.Secret
		expectError bool
	}{
		{
			name:        "Delete existing secret successfully",
			secret:      existingSecret,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().WithObjects(existingSecret).Build()
			controller := NewSecretController(fakeClient, &testutil.MockMetrics{})

			err := controller.DeleteSecret(tt.secret)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestSecretController_GetSecret(t *testing.T) {
	existingSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-secret",
			Namespace: "test-namespace",
		},
		Data: map[string][]byte{
			"key": []byte("value"),
		},
	}

	tests := []struct {
		name        string
		namespace   string
		secretName  string
		expectError bool
		expectFound bool
	}{
		{
			name:        "Get existing secret successfully",
			namespace:   "test-namespace",
			secretName:  "existing-secret",
			expectError: false,
			expectFound: true,
		},
		{
			name:        "Get non-existing secret returns error",
			namespace:   "test-namespace",
			secretName:  "non-existing-secret",
			expectError: true,
			expectFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fakeClient client.Client
			if tt.expectFound {
				fakeClient = fake.NewClientBuilder().WithObjects(existingSecret).Build()
			} else {
				fakeClient = fake.NewClientBuilder().Build()
			}

			controller := NewSecretController(fakeClient, &testutil.MockMetrics{})

			secret, err := controller.GetSecret(tt.namespace, tt.secretName)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if tt.expectFound && secret == nil {
				t.Error("Expected secret to be found but got nil")
			}
			if tt.expectFound && secret.Name != tt.secretName {
				t.Errorf("Expected secret name %s, got %s", tt.secretName, secret.Name)
			}
		})
	}
}
