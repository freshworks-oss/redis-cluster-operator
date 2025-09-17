package k8sutil

import (
	"testing"

	redisv1alpha1 "github.com/dinesh-murugiah/rediscluster-operator/api/v1alpha1"
	"github.com/dinesh-murugiah/rediscluster-operator/utils/testutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestNewCRControl(t *testing.T) {
	fakeClient := fake.NewClientBuilder().Build()
	metrics := &testutil.MockMetrics{}

	controller := NewCRControl(fakeClient, metrics)
	if controller == nil {
		t.Error("NewCRControl() returned nil")
	}

	// Verify it implements ICustomResource interface
	_, ok := controller.(ICustomResource)
	if !ok {
		t.Error("NewCRControl() does not implement ICustomResource interface")
	}
}

func TestClusterControl_UpdateCRStatus(t *testing.T) {
	existingDRC := &redisv1alpha1.DistributedRedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-drc",
			Namespace: "test-namespace",
		},
		Spec: redisv1alpha1.DistributedRedisClusterSpec{
			MasterSize: 3,
		},
	}

	tests := []struct {
		name        string
		obj         runtime.Object
		expectError bool
	}{
		{
			name:        "Update CR status successfully",
			obj:         existingDRC,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			_ = redisv1alpha1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(existingDRC).
				WithStatusSubresource(&redisv1alpha1.DistributedRedisCluster{}).
				Build()

			controller := NewCRControl(fakeClient, &testutil.MockMetrics{})

			err := controller.UpdateCRStatus(tt.obj)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestClusterControl_UpdateCR(t *testing.T) {
	existingDRC := &redisv1alpha1.DistributedRedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-drc",
			Namespace: "test-namespace",
		},
		Spec: redisv1alpha1.DistributedRedisClusterSpec{
			MasterSize: 3,
		},
	}

	tests := []struct {
		name        string
		obj         runtime.Object
		expectError bool
	}{
		{
			name:        "Update CR successfully",
			obj:         existingDRC,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			_ = redisv1alpha1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(existingDRC).
				Build()

			controller := NewCRControl(fakeClient, &testutil.MockMetrics{})

			err := controller.UpdateCR(tt.obj)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestClusterControl_GetRedisClusterBackup(t *testing.T) {
	existingBackup := &redisv1alpha1.RedisClusterBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-backup",
			Namespace: "test-namespace",
		},
		Spec: redisv1alpha1.RedisClusterBackupSpec{
			RedisClusterName: "test-cluster",
		},
	}

	tests := []struct {
		name        string
		namespace   string
		backupName  string
		expectError bool
		expectFound bool
	}{
		{
			name:        "Get existing backup successfully",
			namespace:   "test-namespace",
			backupName:  "existing-backup",
			expectError: false,
			expectFound: true,
		},
		{
			name:        "Get non-existing backup returns error",
			namespace:   "test-namespace",
			backupName:  "non-existing-backup",
			expectError: true,
			expectFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			_ = redisv1alpha1.AddToScheme(scheme)

			var fakeClient client.Client
			if tt.expectFound {
				fakeClient = fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(existingBackup).
					Build()
			} else {
				fakeClient = fake.NewClientBuilder().
					WithScheme(scheme).
					Build()
			}

			controller := NewCRControl(fakeClient, &testutil.MockMetrics{})

			backup, err := controller.GetRedisClusterBackup(tt.namespace, tt.backupName)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if tt.expectFound && backup == nil {
				t.Error("Expected backup to be found but got nil")
			}
			if tt.expectFound && backup != nil && backup.Name != tt.backupName {
				t.Errorf("Expected backup name %s, got %s", tt.backupName, backup.Name)
			}
		})
	}
}

func TestClusterControl_GetDistributedRedisCluster(t *testing.T) {
	existingDRC := &redisv1alpha1.DistributedRedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-drc",
			Namespace: "test-namespace",
		},
		Spec: redisv1alpha1.DistributedRedisClusterSpec{
			MasterSize: 3,
		},
	}

	tests := []struct {
		name        string
		namespace   string
		drcName     string
		expectError bool
		expectFound bool
	}{
		{
			name:        "Get existing DRC successfully",
			namespace:   "test-namespace",
			drcName:     "existing-drc",
			expectError: false,
			expectFound: true,
		},
		{
			name:        "Get non-existing DRC returns error",
			namespace:   "test-namespace",
			drcName:     "non-existing-drc",
			expectError: true,
			expectFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			_ = redisv1alpha1.AddToScheme(scheme)

			var fakeClient client.Client
			if tt.expectFound {
				fakeClient = fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(existingDRC).
					Build()
			} else {
				fakeClient = fake.NewClientBuilder().
					WithScheme(scheme).
					Build()
			}

			controller := NewCRControl(fakeClient, &testutil.MockMetrics{})

			drc, err := controller.GetDistributedRedisCluster(tt.namespace, tt.drcName)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if tt.expectFound && drc == nil {
				t.Error("Expected DRC to be found but got nil")
			}
			if tt.expectFound && drc != nil && drc.Name != tt.drcName {
				t.Errorf("Expected DRC name %s, got %s", tt.drcName, drc.Name)
			}
		})
	}
}

func TestClusterControl_UpdateCrdAnnotations(t *testing.T) {
	tests := []struct {
		name            string
		drc             *redisv1alpha1.DistributedRedisCluster
		annotationKey   string
		annotationValue string
		expectError     bool
		expectUpdate    bool
	}{
		{
			name: "Update annotation when key exists and value differs",
			drc: &redisv1alpha1.DistributedRedisCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drc",
					Namespace: "test-namespace",
					Annotations: map[string]string{
						"test-key": "old-value",
					},
				},
			},
			annotationKey:   "test-key",
			annotationValue: "new-value",
			expectError:     false,
			expectUpdate:    true,
		},
		{
			name: "Don't update when annotation value is the same",
			drc: &redisv1alpha1.DistributedRedisCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drc",
					Namespace: "test-namespace",
					Annotations: map[string]string{
						"test-key": "same-value",
					},
				},
			},
			annotationKey:   "test-key",
			annotationValue: "same-value",
			expectError:     false,
			expectUpdate:    false,
		},
		{
			name: "Return error when annotation key doesn't exist",
			drc: &redisv1alpha1.DistributedRedisCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drc",
					Namespace: "test-namespace",
					Annotations: map[string]string{
						"other-key": "value",
					},
				},
			},
			annotationKey:   "nonexistent-key",
			annotationValue: "value",
			expectError:     true,
			expectUpdate:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			_ = redisv1alpha1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.drc).
				Build()

			controller := NewCRControl(fakeClient, &testutil.MockMetrics{})

			err := controller.UpdateCrdAnnotations(tt.drc, tt.annotationKey, tt.annotationValue)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError && !tt.expectUpdate {
				// Verify annotation value is as expected
				if tt.drc.ObjectMeta.Annotations[tt.annotationKey] != tt.annotationValue {
					t.Errorf("Expected annotation value %s, got %s", tt.annotationValue, tt.drc.ObjectMeta.Annotations[tt.annotationKey])
				}
			}
		})
	}
}
