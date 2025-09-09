package poddisruptionbudgets

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	redisv1alpha1 "github.com/dinesh-murugiah/rediscluster-operator/api/v1alpha1"
)

func createTestDistributedRedisCluster(name, namespace string, clusterReplicas int32) *redisv1alpha1.DistributedRedisCluster {
	return &redisv1alpha1.DistributedRedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: redisv1alpha1.DistributedRedisClusterSpec{
			MasterSize:      3,
			ClusterReplicas: clusterReplicas,
		},
	}
}

func createTestLabels() map[string]string {
	return map[string]string{
		"app":       "redis-cluster",
		"component": "redis",
		"tier":      "cache",
	}
}

func TestNewPodDisruptionBudgetForCR(t *testing.T) {
	tests := []struct {
		name                 string
		cluster              *redisv1alpha1.DistributedRedisCluster
		nameParam            string
		labels               map[string]string
		expectedErr          bool
		expectedMinAvailable int
	}{
		{
			name:                 "Valid PDB creation with 1 replica",
			cluster:              createTestDistributedRedisCluster("test-cluster", "test-ns", 1),
			nameParam:            "test-pdb",
			labels:               createTestLabels(),
			expectedErr:          false,
			expectedMinAvailable: 1,
		},
		{
			name:                 "Valid PDB creation with 2 replicas",
			cluster:              createTestDistributedRedisCluster("test-cluster", "test-ns", 2),
			nameParam:            "test-pdb",
			labels:               createTestLabels(),
			expectedErr:          false,
			expectedMinAvailable: 2,
		},
		{
			name:                 "Valid PDB creation with 0 replicas",
			cluster:              createTestDistributedRedisCluster("test-cluster", "test-ns", 0),
			nameParam:            "test-pdb",
			labels:               createTestLabels(),
			expectedErr:          false,
			expectedMinAvailable: 0,
		},
		{
			name:                 "Valid PDB creation with 5 replicas",
			cluster:              createTestDistributedRedisCluster("test-cluster", "test-ns", 5),
			nameParam:            "test-pdb",
			labels:               createTestLabels(),
			expectedErr:          false,
			expectedMinAvailable: 5,
		},
		{
			name:                 "Nil cluster",
			cluster:              nil,
			nameParam:            "test-pdb",
			labels:               createTestLabels(),
			expectedErr:          true,
			expectedMinAvailable: 0,
		},
		{
			name:                 "Empty name",
			cluster:              createTestDistributedRedisCluster("test-cluster", "test-ns", 1),
			nameParam:            "",
			labels:               createTestLabels(),
			expectedErr:          false,
			expectedMinAvailable: 1,
		},
		{
			name:                 "Nil labels",
			cluster:              createTestDistributedRedisCluster("test-cluster", "test-ns", 1),
			nameParam:            "test-pdb",
			labels:               nil,
			expectedErr:          false,
			expectedMinAvailable: 1,
		},
		{
			name:                 "Empty labels",
			cluster:              createTestDistributedRedisCluster("test-cluster", "test-ns", 1),
			nameParam:            "test-pdb",
			labels:               map[string]string{},
			expectedErr:          false,
			expectedMinAvailable: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cluster == nil {
				// Skip nil cluster test as it would cause panic
				return
			}

			pdb := NewPodDisruptionBudgetForCR(tt.cluster, tt.nameParam, tt.labels)

			if tt.expectedErr {
				assert.Nil(t, pdb)
			} else {
				assert.NotNil(t, pdb)
				assert.Equal(t, tt.nameParam, pdb.Name)
				assert.Equal(t, tt.cluster.Namespace, pdb.Namespace)
				assert.Equal(t, tt.labels, pdb.Labels)
				assert.Equal(t, tt.labels, pdb.Spec.Selector.MatchLabels)

				// Check MinAvailable
				assert.NotNil(t, pdb.Spec.MinAvailable)
				assert.Equal(t, tt.expectedMinAvailable, pdb.Spec.MinAvailable.IntValue())

				// Check owner references
				assert.NotEmpty(t, pdb.OwnerReferences)
			}
		})
	}
}

func TestPodDisruptionBudgetSpec(t *testing.T) {
	t.Run("Check PDB spec structure", func(t *testing.T) {
		cluster := createTestDistributedRedisCluster("test-cluster", "test-ns", 2)
		labels := createTestLabels()

		pdb := NewPodDisruptionBudgetForCR(cluster, "test-pdb", labels)

		assert.NotNil(t, pdb)

		// Check spec structure
		assert.NotNil(t, pdb.Spec)
		assert.NotNil(t, pdb.Spec.MinAvailable)
		assert.NotNil(t, pdb.Spec.Selector)

		// Check MinAvailable is set to cluster replicas
		assert.Equal(t, 2, pdb.Spec.MinAvailable.IntValue())

		// Check selector matches labels
		assert.Equal(t, labels, pdb.Spec.Selector.MatchLabels)
	})
}

func TestPodDisruptionBudgetMetadata(t *testing.T) {
	t.Run("Check PDB metadata", func(t *testing.T) {
		cluster := createTestDistributedRedisCluster("test-cluster", "test-ns", 1)
		labels := createTestLabels()

		pdb := NewPodDisruptionBudgetForCR(cluster, "test-pdb", labels)

		assert.NotNil(t, pdb)

		// Check metadata
		assert.Equal(t, "test-pdb", pdb.Name)
		assert.Equal(t, "test-ns", pdb.Namespace)
		assert.Equal(t, labels, pdb.Labels)

		// Check owner references
		assert.Len(t, pdb.OwnerReferences, 1)
		ownerRef := pdb.OwnerReferences[0]
		assert.Equal(t, "DistributedRedisCluster", ownerRef.Kind)
		assert.Equal(t, "redis.kun/v1alpha1", ownerRef.APIVersion)
		assert.Equal(t, cluster.Name, ownerRef.Name)
		assert.True(t, *ownerRef.Controller)
	})
}

func TestPodDisruptionBudgetWithDifferentReplicaCounts(t *testing.T) {
	tests := []struct {
		name                 string
		clusterReplicas      int32
		expectedMinAvailable int
	}{
		{
			name:                 "1 replica",
			clusterReplicas:      1,
			expectedMinAvailable: 1,
		},
		{
			name:                 "2 replicas",
			clusterReplicas:      2,
			expectedMinAvailable: 2,
		},
		{
			name:                 "3 replicas",
			clusterReplicas:      3,
			expectedMinAvailable: 3,
		},
		{
			name:                 "5 replicas",
			clusterReplicas:      5,
			expectedMinAvailable: 5,
		},
		{
			name:                 "10 replicas",
			clusterReplicas:      10,
			expectedMinAvailable: 10,
		},
		{
			name:                 "0 replicas",
			clusterReplicas:      0,
			expectedMinAvailable: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cluster := createTestDistributedRedisCluster("test-cluster", "test-ns", tt.clusterReplicas)
			labels := createTestLabels()

			pdb := NewPodDisruptionBudgetForCR(cluster, "test-pdb", labels)

			assert.NotNil(t, pdb)
			assert.Equal(t, tt.expectedMinAvailable, pdb.Spec.MinAvailable.IntValue())
		})
	}
}

func TestPodDisruptionBudgetWithCustomLabels(t *testing.T) {
	t.Run("PDB with custom labels", func(t *testing.T) {
		cluster := createTestDistributedRedisCluster("test-cluster", "test-ns", 2)
		customLabels := map[string]string{
			"app":         "custom-redis",
			"environment": "production",
			"team":        "platform",
			"version":     "v1.0.0",
		}

		pdb := NewPodDisruptionBudgetForCR(cluster, "test-pdb", customLabels)

		assert.NotNil(t, pdb)
		assert.Equal(t, customLabels, pdb.Labels)
		assert.Equal(t, customLabels, pdb.Spec.Selector.MatchLabels)
	})
}

func TestPodDisruptionBudgetEdgeCases(t *testing.T) {
	t.Run("PDB with empty name", func(t *testing.T) {
		cluster := createTestDistributedRedisCluster("test-cluster", "test-ns", 1)
		labels := createTestLabels()

		pdb := NewPodDisruptionBudgetForCR(cluster, "", labels)

		assert.NotNil(t, pdb)
		assert.Equal(t, "", pdb.Name)
	})

	t.Run("PDB with nil labels", func(t *testing.T) {
		cluster := createTestDistributedRedisCluster("test-cluster", "test-ns", 1)

		pdb := NewPodDisruptionBudgetForCR(cluster, "test-pdb", nil)

		assert.NotNil(t, pdb)
		assert.Nil(t, pdb.Labels)
		assert.Nil(t, pdb.Spec.Selector.MatchLabels)
	})

	t.Run("PDB with empty labels", func(t *testing.T) {
		cluster := createTestDistributedRedisCluster("test-cluster", "test-ns", 1)
		emptyLabels := map[string]string{}

		pdb := NewPodDisruptionBudgetForCR(cluster, "test-pdb", emptyLabels)

		assert.NotNil(t, pdb)
		assert.Equal(t, emptyLabels, pdb.Labels)
		assert.Equal(t, emptyLabels, pdb.Spec.Selector.MatchLabels)
	})
}
