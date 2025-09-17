package k8sutil

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestIsRequestRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "ServiceUnavailable error should be retryable",
			err:      apierrors.NewServiceUnavailable("service unavailable"),
			expected: true,
		},
		{
			name:     "Timeout error should be retryable",
			err:      apierrors.NewTimeoutError("timeout", 5),
			expected: true,
		},
		{
			name:     "ServerTimeout error should be retryable",
			err:      apierrors.NewServerTimeout(schema.GroupResource{Group: "test", Resource: "test"}, "get", 5),
			expected: true,
		},
		{
			name:     "TooManyRequests error should be retryable",
			err:      apierrors.NewTooManyRequests("too many requests", 5),
			expected: true,
		},
		{
			name:     "NotFound error should not be retryable",
			err:      apierrors.NewNotFound(schema.GroupResource{Group: "test", Resource: "test"}, "test-resource"),
			expected: false,
		},
		{
			name:     "BadRequest error should not be retryable",
			err:      apierrors.NewBadRequest("bad request"),
			expected: false,
		},
		{
			name:     "Generic error should not be retryable",
			err:      errors.New("generic error"),
			expected: false,
		},
		{
			name:     "Nil error should not be retryable",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRequestRetryable(tt.err)
			if result != tt.expected {
				t.Errorf("IsRequestRetryable() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCreateSecret(t *testing.T) {
	logger := logr.Discard()

	tests := []struct {
		name         string
		secret       *corev1.Secret
		existingObjs []client.Object
		expectError  bool
		expectCreate bool
	}{
		{
			name: "Create new secret when it doesn't exist",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "test-namespace",
				},
				Data: map[string][]byte{
					"key": []byte("value"),
				},
			},
			existingObjs: []client.Object{},
			expectError:  false,
			expectCreate: true,
		},
		{
			name: "Don't create secret when it already exists",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-secret",
					Namespace: "test-namespace",
				},
				Data: map[string][]byte{
					"key": []byte("value"),
				},
			},
			existingObjs: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "existing-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"key": []byte("existing-value"),
					},
				},
			},
			expectError:  false,
			expectCreate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake client with existing objects
			fakeClient := fake.NewClientBuilder().
				WithObjects(tt.existingObjs...).
				Build()

			err := CreateSecret(fakeClient, tt.secret, logger)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			// Verify the secret exists in the client
			retrievedSecret := &corev1.Secret{}
			getErr := fakeClient.Get(context.TODO(), types.NamespacedName{
				Namespace: tt.secret.Namespace,
				Name:      tt.secret.Name,
			}, retrievedSecret)

			if getErr != nil {
				t.Errorf("Failed to retrieve secret after CreateSecret: %v", getErr)
			}

			// If we expected creation, verify the data matches what we tried to create
			if tt.expectCreate {
				if retrievedSecret.Data["key"] == nil {
					t.Error("Expected secret data to be set")
				}
				if string(retrievedSecret.Data["key"]) != string(tt.secret.Data["key"]) {
					t.Errorf("Expected secret data %s, got %s", string(tt.secret.Data["key"]), string(retrievedSecret.Data["key"]))
				}
			}
		})
	}
}
