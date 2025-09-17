package k8sutil

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/dinesh-murugiah/rediscluster-operator/utils/testutil"
)

func TestNewPvcController(t *testing.T) {
	fakeClient := fake.NewClientBuilder().Build()
	metrics := &testutil.MockMetrics{}

	controller := NewPvcController(fakeClient, metrics)
	if controller == nil {
		t.Error("NewPvcController() returned nil")
	}

	// Verify it implements IPvcControl interface
	_, ok := controller.(IPvcControl)
	if !ok {
		t.Error("NewPvcController() does not implement IPvcControl interface")
	}
}

func TestPvcController_CreatePvc(t *testing.T) {
	tests := []struct {
		name        string
		pvc         *corev1.PersistentVolumeClaim
		expectError bool
	}{
		{
			name: "Create PVC successfully",
			pvc: &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pvc",
					Namespace: "test-namespace",
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteOnce,
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("1Gi"),
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
			controller := NewPvcController(fakeClient, &testutil.MockMetrics{})

			err := controller.CreatePvc(tt.pvc)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError {
				// Verify PVC was created
				retrievedPVC := &corev1.PersistentVolumeClaim{}
				getErr := fakeClient.Get(context.TODO(), types.NamespacedName{
					Namespace: tt.pvc.Namespace,
					Name:      tt.pvc.Name,
				}, retrievedPVC)
				if getErr != nil {
					t.Errorf("Failed to retrieve PVC after creation: %v", getErr)
				}
			}
		})
	}
}

func TestPvcController_DeletePvc(t *testing.T) {
	existingPVC := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pvc-to-delete",
			Namespace: "test-namespace",
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
		},
	}

	tests := []struct {
		name        string
		pvc         *corev1.PersistentVolumeClaim
		expectError bool
	}{
		{
			name:        "Delete existing PVC successfully",
			pvc:         existingPVC,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().WithObjects(existingPVC).Build()
			controller := NewPvcController(fakeClient, &testutil.MockMetrics{})

			err := controller.DeletePvc(tt.pvc)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestPvcController_GetPvc(t *testing.T) {
	existingPVC := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-pvc",
			Namespace: "test-namespace",
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
		},
	}

	tests := []struct {
		name        string
		namespace   string
		pvcName     string
		expectError bool
		expectFound bool
	}{
		{
			name:        "Get existing PVC successfully",
			namespace:   "test-namespace",
			pvcName:     "existing-pvc",
			expectError: false,
			expectFound: true,
		},
		{
			name:        "Get non-existing PVC returns error",
			namespace:   "test-namespace",
			pvcName:     "non-existing-pvc",
			expectError: true,
			expectFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fakeClient client.Client
			if tt.expectFound {
				fakeClient = fake.NewClientBuilder().WithObjects(existingPVC).Build()
			} else {
				fakeClient = fake.NewClientBuilder().Build()
			}

			controller := NewPvcController(fakeClient, &testutil.MockMetrics{})

			pvc, err := controller.GetPvc(tt.namespace, tt.pvcName)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if tt.expectFound && pvc == nil {
				t.Error("Expected PVC to be found but got nil")
			}
			if tt.expectFound && pvc.Name != tt.pvcName {
				t.Errorf("Expected PVC name %s, got %s", tt.pvcName, pvc.Name)
			}
		})
	}
}

func TestPvcController_DeletePvcByLabels(t *testing.T) {
	pvc1 := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pvc1",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"app": "test",
				"env": "dev",
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
		},
	}

	pvc2 := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pvc2",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"app": "test",
				"env": "prod",
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("2Gi"),
				},
			},
		},
	}

	pvc3 := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pvc3",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"app": "other",
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
		},
	}

	tests := []struct {
		name        string
		namespace   string
		labels      map[string]string
		expectError bool
	}{
		{
			name:      "Delete PVCs with matching app label",
			namespace: "test-namespace",
			labels: map[string]string{
				"app": "test",
			},
			expectError: false,
		},
		{
			name:      "Delete PVCs with specific env label",
			namespace: "test-namespace",
			labels: map[string]string{
				"env": "dev",
			},
			expectError: false,
		},
		{
			name:      "Delete PVCs with no matching labels",
			namespace: "test-namespace",
			labels: map[string]string{
				"nonexistent": "value",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().WithObjects(pvc1, pvc2, pvc3).Build()
			controller := NewPvcController(fakeClient, &testutil.MockMetrics{})

			err := controller.DeletePvcByLabels(tt.namespace, tt.labels)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}
