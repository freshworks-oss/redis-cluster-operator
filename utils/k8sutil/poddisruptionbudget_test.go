package k8sutil

import (
	"context"
	"testing"

	"github.com/dinesh-murugiah/rediscluster-operator/utils/testutil"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestNewPodDisruptionBudgetController(t *testing.T) {
	fakeClient := fake.NewClientBuilder().Build()
	metrics := &testutil.MockMetrics{}

	controller := NewPodDisruptionBudgetController(fakeClient, metrics)
	if controller == nil {
		t.Error("NewPodDisruptionBudgetController() returned nil")
	}

	// Verify it implements IPodDisruptionBudgetControl interface
	_, ok := controller.(IPodDisruptionBudgetControl)
	if !ok {
		t.Error("NewPodDisruptionBudgetController() does not implement IPodDisruptionBudgetControl interface")
	}
}

func TestPodDisruptionBudgetController_CreatePodDisruptionBudget(t *testing.T) {
	minAvailable := intstr.FromInt(1)

	tests := []struct {
		name        string
		pdb         *policyv1.PodDisruptionBudget
		expectError bool
	}{
		{
			name: "Create PodDisruptionBudget successfully",
			pdb: &policyv1.PodDisruptionBudget{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pdb",
					Namespace: "test-namespace",
				},
				Spec: policyv1.PodDisruptionBudgetSpec{
					MinAvailable: &minAvailable,
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "test",
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
			controller := NewPodDisruptionBudgetController(fakeClient, &testutil.MockMetrics{})

			err := controller.CreatePodDisruptionBudget(tt.pdb)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError {
				// Verify PDB was created
				retrievedPDB := &policyv1.PodDisruptionBudget{}
				getErr := fakeClient.Get(context.TODO(), types.NamespacedName{
					Namespace: tt.pdb.Namespace,
					Name:      tt.pdb.Name,
				}, retrievedPDB)
				if getErr != nil {
					t.Errorf("Failed to retrieve PodDisruptionBudget after creation: %v", getErr)
				}
			}
		})
	}
}

func TestPodDisruptionBudgetController_UpdatePodDisruptionBudget(t *testing.T) {
	minAvailable := intstr.FromInt(1)
	existingPDB := &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-pdb",
			Namespace: "test-namespace",
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			MinAvailable: &minAvailable,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
			},
		},
	}

	updatedMinAvailable := intstr.FromInt(2)
	tests := []struct {
		name        string
		expectError bool
	}{
		{
			name:        "Update existing PodDisruptionBudget successfully",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().WithObjects(existingPDB).Build()
			controller := NewPodDisruptionBudgetController(fakeClient, &testutil.MockMetrics{})

			// Get the existing PDB to have the correct resource version
			retrievedPDB, err := controller.GetPodDisruptionBudget("test-namespace", "existing-pdb")
			if err != nil {
				t.Fatalf("Failed to get existing PDB: %v", err)
			}

			// Update the spec
			retrievedPDB.Spec.MinAvailable = &updatedMinAvailable

			err = controller.UpdatePodDisruptionBudget(retrievedPDB)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestPodDisruptionBudgetController_DeletePodDisruptionBudget(t *testing.T) {
	minAvailable := intstr.FromInt(1)
	existingPDB := &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pdb-to-delete",
			Namespace: "test-namespace",
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			MinAvailable: &minAvailable,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
			},
		},
	}

	tests := []struct {
		name        string
		pdb         *policyv1.PodDisruptionBudget
		expectError bool
	}{
		{
			name:        "Delete existing PodDisruptionBudget successfully",
			pdb:         existingPDB,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().WithObjects(existingPDB).Build()
			controller := NewPodDisruptionBudgetController(fakeClient, &testutil.MockMetrics{})

			err := controller.DeletePodDisruptionBudget(tt.pdb)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestPodDisruptionBudgetController_DeletePodDisruptionBudgetByName(t *testing.T) {
	minAvailable := intstr.FromInt(1)
	existingPDB := &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pdb-to-delete",
			Namespace: "test-namespace",
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			MinAvailable: &minAvailable,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
			},
		},
	}

	tests := []struct {
		name        string
		namespace   string
		pdbName     string
		expectError bool
	}{
		{
			name:        "Delete existing PDB by name successfully",
			namespace:   "test-namespace",
			pdbName:     "pdb-to-delete",
			expectError: false,
		},
		{
			name:        "Delete non-existing PDB by name returns error",
			namespace:   "test-namespace",
			pdbName:     "non-existing-pdb",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fakeClient client.Client
			if tt.pdbName == "pdb-to-delete" {
				fakeClient = fake.NewClientBuilder().WithObjects(existingPDB).Build()
			} else {
				fakeClient = fake.NewClientBuilder().Build()
			}

			controller := NewPodDisruptionBudgetController(fakeClient, &testutil.MockMetrics{})

			err := controller.DeletePodDisruptionBudgetByName(tt.namespace, tt.pdbName)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestPodDisruptionBudgetController_GetPodDisruptionBudget(t *testing.T) {
	minAvailable := intstr.FromInt(1)
	existingPDB := &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-pdb",
			Namespace: "test-namespace",
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			MinAvailable: &minAvailable,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
			},
		},
	}

	tests := []struct {
		name        string
		namespace   string
		pdbName     string
		expectError bool
		expectFound bool
	}{
		{
			name:        "Get existing PodDisruptionBudget successfully",
			namespace:   "test-namespace",
			pdbName:     "existing-pdb",
			expectError: false,
			expectFound: true,
		},
		{
			name:        "Get non-existing PodDisruptionBudget returns error",
			namespace:   "test-namespace",
			pdbName:     "non-existing-pdb",
			expectError: true,
			expectFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fakeClient client.Client
			if tt.expectFound {
				fakeClient = fake.NewClientBuilder().WithObjects(existingPDB).Build()
			} else {
				fakeClient = fake.NewClientBuilder().Build()
			}

			controller := NewPodDisruptionBudgetController(fakeClient, &testutil.MockMetrics{})

			pdb, err := controller.GetPodDisruptionBudget(tt.namespace, tt.pdbName)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if tt.expectFound && pdb == nil {
				t.Error("Expected PodDisruptionBudget to be found but got nil")
			}
			if tt.expectFound && pdb.Name != tt.pdbName {
				t.Errorf("Expected PodDisruptionBudget name %s, got %s", tt.pdbName, pdb.Name)
			}
		})
	}
}
