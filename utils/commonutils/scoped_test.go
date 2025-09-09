package utils

import (
	"os"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// MockObject implements metav1.Object for testing
type MockObject struct {
	namespace string
}

func (m *MockObject) GetNamespace() string {
	return m.namespace
}

func (m *MockObject) SetNamespace(namespace string) {
	m.namespace = namespace
}

func (m *MockObject) GetName() string {
	return "test-object"
}

func (m *MockObject) SetName(name string) {}

func (m *MockObject) GetGenerateName() string {
	return ""
}

func (m *MockObject) SetGenerateName(name string) {}

func (m *MockObject) GetUID() types.UID {
	return "test-uid"
}

func (m *MockObject) SetUID(uid types.UID) {}

func (m *MockObject) GetResourceVersion() string {
	return "1"
}

func (m *MockObject) SetResourceVersion(version string) {}

func (m *MockObject) GetGeneration() int64 {
	return 1
}

func (m *MockObject) SetGeneration(generation int64) {}

func (m *MockObject) GetSelfLink() string {
	return ""
}

func (m *MockObject) SetSelfLink(selfLink string) {}

func (m *MockObject) GetCreationTimestamp() metav1.Time {
	return metav1.Time{}
}

func (m *MockObject) SetCreationTimestamp(timestamp metav1.Time) {}

func (m *MockObject) GetDeletionTimestamp() *metav1.Time {
	return nil
}

func (m *MockObject) SetDeletionTimestamp(timestamp *metav1.Time) {}

func (m *MockObject) GetDeletionGracePeriodSeconds() *int64 {
	return nil
}

func (m *MockObject) SetDeletionGracePeriodSeconds(gracePeriodSeconds *int64) {}

func (m *MockObject) GetLabels() map[string]string {
	return nil
}

func (m *MockObject) SetLabels(labels map[string]string) {}

func (m *MockObject) GetAnnotations() map[string]string {
	return nil
}

func (m *MockObject) SetAnnotations(annotations map[string]string) {}

func (m *MockObject) GetFinalizers() []string {
	return nil
}

func (m *MockObject) SetFinalizers(finalizers []string) {}

func (m *MockObject) GetOwnerReferences() []metav1.OwnerReference {
	return nil
}

func (m *MockObject) SetOwnerReferences(references []metav1.OwnerReference) {}

func (m *MockObject) GetClusterName() string {
	return ""
}

func (m *MockObject) SetClusterName(name string) {}

func (m *MockObject) GetManagedFields() []metav1.ManagedFieldsEntry {
	return nil
}

func (m *MockObject) SetManagedFields(fields []metav1.ManagedFieldsEntry) {}

func TestIsClusterScoped(t *testing.T) {
	// Test default behavior
	result := IsClusterScoped()
	assert.True(t, result, "Default should be cluster scoped")
}

func TestGetNamespacesList(t *testing.T) {
	// Save original environment variables
	originalWatch := os.Getenv("WATCH_NAMESPACE")
	originalUnwatch := os.Getenv("UNWATCH_NAMESPACE")
	defer func() {
		os.Setenv("WATCH_NAMESPACE", originalWatch)
		os.Setenv("UNWATCH_NAMESPACE", originalUnwatch)
	}()

	tests := []struct {
		name                 string
		watchNamespace       string
		unwatchNamespace     string
		expectError          bool
		expectedWatchCount   int
		expectedUnwatchCount int
	}{
		{
			name:                 "Valid namespaces",
			watchNamespace:       "ns1,ns2,ns3",
			unwatchNamespace:     "ns4,ns5",
			expectError:          false,
			expectedWatchCount:   3,
			expectedUnwatchCount: 2,
		},
		{
			name:                 "Single namespace",
			watchNamespace:       "default",
			unwatchNamespace:     "kube-system",
			expectError:          false,
			expectedWatchCount:   1,
			expectedUnwatchCount: 1,
		},
		{
			name:                 "Empty namespaces",
			watchNamespace:       "",
			unwatchNamespace:     "",
			expectError:          true, // Should expect error when both are empty
			expectedWatchCount:   0,
			expectedUnwatchCount: 0,
		},
		{
			name:                 "Missing WATCH_NAMESPACE",
			watchNamespace:       "",
			unwatchNamespace:     "ns1",
			expectError:          true,
			expectedWatchCount:   0,
			expectedUnwatchCount: 0,
		},
		{
			name:                 "Missing UNWATCH_NAMESPACE",
			watchNamespace:       "ns1",
			unwatchNamespace:     "",
			expectError:          true,
			expectedWatchCount:   0,
			expectedUnwatchCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			if tt.watchNamespace != "" {
				os.Setenv("WATCH_NAMESPACE", tt.watchNamespace)
			} else {
				os.Unsetenv("WATCH_NAMESPACE")
			}
			if tt.unwatchNamespace != "" {
				os.Setenv("UNWATCH_NAMESPACE", tt.unwatchNamespace)
			} else {
				os.Unsetenv("UNWATCH_NAMESPACE")
			}

			watchList, unwatchList, err := GetNamespacesList()

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, watchList)
				assert.Nil(t, unwatchList)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, watchList)
				assert.NotNil(t, unwatchList)
				assert.Equal(t, tt.expectedWatchCount, len(watchList))
				assert.Equal(t, tt.expectedUnwatchCount, len(unwatchList))
			}
		})
	}
}

func TestGenerateNamespaceList(t *testing.T) {
	// Save original environment variables
	originalWatch := os.Getenv("WATCH_NAMESPACE")
	originalUnwatch := os.Getenv("UNWATCH_NAMESPACE")
	defer func() {
		os.Setenv("WATCH_NAMESPACE", originalWatch)
		os.Setenv("UNWATCH_NAMESPACE", originalUnwatch)
	}()

	t.Run("Valid namespaces", func(t *testing.T) {
		os.Setenv("WATCH_NAMESPACE", "ns1,ns2")
		os.Setenv("UNWATCH_NAMESPACE", "ns3")

		err := GenerateNamespaceList(logr.Discard()) // Passing discard logger
		assert.NoError(t, err)

		watchList, unwatchList := GetNamespaceList()
		assert.Equal(t, 2, len(watchList))
		assert.Equal(t, 1, len(unwatchList))
		assert.Contains(t, watchList, "ns1")
		assert.Contains(t, watchList, "ns2")
		assert.Contains(t, unwatchList, "ns3")
	})

	t.Run("Invalid namespaces", func(t *testing.T) {
		os.Unsetenv("WATCH_NAMESPACE")
		os.Setenv("UNWATCH_NAMESPACE", "ns1")

		err := GenerateNamespaceList(logr.Discard())
		assert.Error(t, err)
	})
}

func TestSetOperatorScope(t *testing.T) {
	// Save original environment variable
	originalWatch := os.Getenv("WATCH_NAMESPACE")
	defer func() {
		os.Setenv("WATCH_NAMESPACE", originalWatch)
	}()

	t.Run("Cluster scoped - no WATCH_NAMESPACE", func(t *testing.T) {
		os.Unsetenv("WATCH_NAMESPACE")

		SetOperatorScope()

		assert.True(t, IsClusterScoped(), "Should be cluster scoped when WATCH_NAMESPACE is not set")
	})

	t.Run("Cluster scoped - empty WATCH_NAMESPACE", func(t *testing.T) {
		os.Setenv("WATCH_NAMESPACE", "")

		SetOperatorScope()

		assert.True(t, IsClusterScoped(), "Should be cluster scoped when WATCH_NAMESPACE is empty")
	})

	t.Run("Namespace scoped - with WATCH_NAMESPACE", func(t *testing.T) {
		os.Setenv("WATCH_NAMESPACE", "default")

		SetOperatorScope()

		assert.False(t, IsClusterScoped(), "Should be namespace scoped when WATCH_NAMESPACE is set")
	})
}

func TestShoudManage(t *testing.T) {
	// Save original environment variables
	originalWatch := os.Getenv("WATCH_NAMESPACE")
	originalUnwatch := os.Getenv("UNWATCH_NAMESPACE")
	defer func() {
		os.Setenv("WATCH_NAMESPACE", originalWatch)
		os.Setenv("UNWATCH_NAMESPACE", originalUnwatch)
	}()

	t.Run("Cluster scoped - should manage all except unwatch list", func(t *testing.T) {
		// For cluster-scoped mode, set WATCH_NAMESPACE to empty but still set it
		os.Setenv("WATCH_NAMESPACE", "")
		os.Setenv("UNWATCH_NAMESPACE", "ns1,ns2")

		SetOperatorScope()
		GenerateNamespaceList(logr.Discard())

		// Test namespaces in unwatch list
		obj1 := &MockObject{namespace: "ns1"}
		assert.False(t, ShoudManage(obj1), "Should not manage namespace in unwatch list")

		obj2 := &MockObject{namespace: "ns2"}
		assert.False(t, ShoudManage(obj2), "Should not manage namespace in unwatch list")

		// Test namespaces not in unwatch list
		obj3 := &MockObject{namespace: "ns3"}
		assert.True(t, ShoudManage(obj3), "Should manage namespace not in unwatch list")

		obj4 := &MockObject{namespace: "default"}
		assert.True(t, ShoudManage(obj4), "Should manage namespace not in unwatch list")
	})

	t.Run("Namespace scoped - should only manage watch list", func(t *testing.T) {
		os.Setenv("WATCH_NAMESPACE", "ns1,ns2")
		os.Setenv("UNWATCH_NAMESPACE", "ns3")

		SetOperatorScope()
		GenerateNamespaceList(logr.Discard())

		// Test namespaces in watch list
		obj1 := &MockObject{namespace: "ns1"}
		assert.True(t, ShoudManage(obj1), "Should manage namespace in watch list")

		obj2 := &MockObject{namespace: "ns2"}
		assert.True(t, ShoudManage(obj2), "Should manage namespace in watch list")

		// Test namespaces not in watch list
		obj3 := &MockObject{namespace: "ns3"}
		assert.False(t, ShoudManage(obj3), "Should not manage namespace not in watch list")

		obj4 := &MockObject{namespace: "default"}
		assert.False(t, ShoudManage(obj4), "Should not manage namespace not in watch list")
	})
}

func TestNamespaceListPersistence(t *testing.T) {
	// Save original environment variables
	originalWatch := os.Getenv("WATCH_NAMESPACE")
	originalUnwatch := os.Getenv("UNWATCH_NAMESPACE")
	defer func() {
		os.Setenv("WATCH_NAMESPACE", originalWatch)
		os.Setenv("UNWATCH_NAMESPACE", originalUnwatch)
	}()

	// Set initial values
	os.Setenv("WATCH_NAMESPACE", "ns1,ns2")
	os.Setenv("UNWATCH_NAMESPACE", "ns3")

	SetOperatorScope()
	err := GenerateNamespaceList(logr.Discard())
	require.NoError(t, err)

	// Verify initial state
	watchList, unwatchList := GetNamespaceList()
	assert.Equal(t, 2, len(watchList))
	assert.Equal(t, 1, len(unwatchList))

	// Change environment variables
	os.Setenv("WATCH_NAMESPACE", "ns4,ns5")
	os.Setenv("UNWATCH_NAMESPACE", "ns6")

	// Verify that the lists are still the same (they should persist)
	watchList2, unwatchList2 := GetNamespaceList()
	assert.Equal(t, watchList, watchList2, "Watch list should persist")
	assert.Equal(t, unwatchList, unwatchList2, "Unwatch list should persist")
}
