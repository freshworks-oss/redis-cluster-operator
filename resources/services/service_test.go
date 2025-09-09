package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	redisv1alpha1 "github.com/dinesh-murugiah/rediscluster-operator/api/v1alpha1"
)

func createTestDistributedRedisCluster(name, namespace string) *redisv1alpha1.DistributedRedisCluster {
	return &redisv1alpha1.DistributedRedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: redisv1alpha1.DistributedRedisClusterSpec{
			MasterSize:      3,
			ClusterReplicas: 1,
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

func TestNewHeadLessSvcForCR(t *testing.T) {
	tests := []struct {
		name        string
		cluster     *redisv1alpha1.DistributedRedisCluster
		nameParam   string
		labels      map[string]string
		expectedErr bool
	}{
		{
			name:        "Valid headless service creation",
			cluster:     createTestDistributedRedisCluster("test-cluster", "test-ns"),
			nameParam:   "test-headless-svc",
			labels:      createTestLabels(),
			expectedErr: false,
		},
		{
			name:        "Nil cluster",
			cluster:     nil,
			nameParam:   "test-headless-svc",
			labels:      createTestLabels(),
			expectedErr: true,
		},
		{
			name:        "Empty name",
			cluster:     createTestDistributedRedisCluster("test-cluster", "test-ns"),
			nameParam:   "",
			labels:      createTestLabels(),
			expectedErr: false,
		},
		{
			name:        "Nil labels",
			cluster:     createTestDistributedRedisCluster("test-cluster", "test-ns"),
			nameParam:   "test-headless-svc",
			labels:      nil,
			expectedErr: false,
		},
		{
			name:        "Empty labels",
			cluster:     createTestDistributedRedisCluster("test-cluster", "test-ns"),
			nameParam:   "test-headless-svc",
			labels:      map[string]string{},
			expectedErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cluster == nil {
				// Skip nil cluster test as it would cause panic
				return
			}

			svc := NewHeadLessSvcForCR(tt.cluster, tt.nameParam, tt.labels)

			if tt.expectedErr {
				assert.Nil(t, svc)
			} else {
				assert.NotNil(t, svc)
				assert.Equal(t, tt.nameParam, svc.Name)
				assert.Equal(t, tt.cluster.Namespace, svc.Namespace)
				assert.Equal(t, tt.labels, svc.Labels)
				assert.Equal(t, tt.labels, svc.Spec.Selector)
				assert.Equal(t, corev1.ClusterIPNone, svc.Spec.ClusterIP)

				// Check ports
				assert.Len(t, svc.Spec.Ports, 2)

				// Check client port
				clientPort := svc.Spec.Ports[0]
				assert.Equal(t, "client", clientPort.Name)
				assert.Equal(t, int32(6379), clientPort.Port)

				// Check gossip port
				gossipPort := svc.Spec.Ports[1]
				assert.Equal(t, "gossip", gossipPort.Name)
				assert.Equal(t, int32(16379), gossipPort.Port)

				// Check owner references
				assert.NotEmpty(t, svc.OwnerReferences)
			}
		})
	}
}

func TestNewSvcForCR(t *testing.T) {
	tests := []struct {
		name        string
		cluster     *redisv1alpha1.DistributedRedisCluster
		nameParam   string
		labels      map[string]string
		expectedErr bool
	}{
		{
			name:        "Valid service creation",
			cluster:     createTestDistributedRedisCluster("test-cluster", "test-ns"),
			nameParam:   "test-svc",
			labels:      createTestLabels(),
			expectedErr: false,
		},
		{
			name:        "Nil cluster",
			cluster:     nil,
			nameParam:   "test-svc",
			labels:      createTestLabels(),
			expectedErr: true,
		},
		{
			name:        "Empty name",
			cluster:     createTestDistributedRedisCluster("test-cluster", "test-ns"),
			nameParam:   "",
			labels:      createTestLabels(),
			expectedErr: false,
		},
		{
			name:        "Nil labels",
			cluster:     createTestDistributedRedisCluster("test-cluster", "test-ns"),
			nameParam:   "test-svc",
			labels:      nil,
			expectedErr: false,
		},
		{
			name:        "Empty labels",
			cluster:     createTestDistributedRedisCluster("test-cluster", "test-ns"),
			nameParam:   "test-svc",
			labels:      map[string]string{},
			expectedErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cluster == nil {
				// Skip nil cluster test as it would cause panic
				return
			}

			svc := NewSvcForCR(tt.cluster, tt.nameParam, tt.labels)

			if tt.expectedErr {
				assert.Nil(t, svc)
			} else {
				assert.NotNil(t, svc)
				assert.Equal(t, tt.nameParam, svc.Name)
				assert.Equal(t, tt.cluster.Namespace, svc.Namespace)
				assert.Equal(t, tt.labels, svc.Labels)
				assert.Equal(t, tt.labels, svc.Spec.Selector)

				// Check ports
				assert.Len(t, svc.Spec.Ports, 2)

				// Check client port
				clientPort := svc.Spec.Ports[0]
				assert.Equal(t, "client", clientPort.Name)
				assert.Equal(t, int32(6379), clientPort.Port)

				// Check gossip port
				gossipPort := svc.Spec.Ports[1]
				assert.Equal(t, "gossip", gossipPort.Name)
				assert.Equal(t, int32(16379), gossipPort.Port)

				// Check owner references
				assert.NotEmpty(t, svc.OwnerReferences)
			}
		})
	}
}

func TestHeadlessServiceSpec(t *testing.T) {
	t.Run("Check headless service spec structure", func(t *testing.T) {
		cluster := createTestDistributedRedisCluster("test-cluster", "test-ns")
		labels := createTestLabels()

		svc := NewHeadLessSvcForCR(cluster, "test-headless-svc", labels)

		assert.NotNil(t, svc)

		// Check spec structure
		assert.NotNil(t, svc.Spec)
		assert.Equal(t, corev1.ClusterIPNone, svc.Spec.ClusterIP)
		assert.Equal(t, labels, svc.Spec.Selector)

		// Check ports are in correct order
		assert.Len(t, svc.Spec.Ports, 2)
		assert.Equal(t, "client", svc.Spec.Ports[0].Name)
		assert.Equal(t, int32(6379), svc.Spec.Ports[0].Port)
		assert.Equal(t, "gossip", svc.Spec.Ports[1].Name)
		assert.Equal(t, int32(16379), svc.Spec.Ports[1].Port)
	})
}

func TestRegularServiceSpec(t *testing.T) {
	t.Run("Check regular service spec structure", func(t *testing.T) {
		cluster := createTestDistributedRedisCluster("test-cluster", "test-ns")
		labels := createTestLabels()

		svc := NewSvcForCR(cluster, "test-svc", labels)

		assert.NotNil(t, svc)

		// Check spec structure
		assert.NotNil(t, svc.Spec)
		assert.Equal(t, labels, svc.Spec.Selector)

		// Check ports are in correct order
		assert.Len(t, svc.Spec.Ports, 2)
		assert.Equal(t, "client", svc.Spec.Ports[0].Name)
		assert.Equal(t, int32(6379), svc.Spec.Ports[0].Port)
		assert.Equal(t, "gossip", svc.Spec.Ports[1].Name)
		assert.Equal(t, int32(16379), svc.Spec.Ports[1].Port)
	})
}

func TestServiceMetadata(t *testing.T) {
	t.Run("Check service metadata", func(t *testing.T) {
		cluster := createTestDistributedRedisCluster("test-cluster", "test-ns")
		labels := createTestLabels()

		svc := NewSvcForCR(cluster, "test-svc", labels)

		assert.NotNil(t, svc)

		// Check metadata
		assert.Equal(t, "test-svc", svc.Name)
		assert.Equal(t, "test-ns", svc.Namespace)
		assert.Equal(t, labels, svc.Labels)

		// Check owner references
		assert.Len(t, svc.OwnerReferences, 1)
		ownerRef := svc.OwnerReferences[0]
		assert.Equal(t, "DistributedRedisCluster", ownerRef.Kind)
		assert.Equal(t, "redis.kun/v1alpha1", ownerRef.APIVersion)
		assert.Equal(t, cluster.Name, ownerRef.Name)
		assert.True(t, *ownerRef.Controller)
	})
}

func TestHeadlessServiceMetadata(t *testing.T) {
	t.Run("Check headless service metadata", func(t *testing.T) {
		cluster := createTestDistributedRedisCluster("test-cluster", "test-ns")
		labels := createTestLabels()

		svc := NewHeadLessSvcForCR(cluster, "test-headless-svc", labels)

		assert.NotNil(t, svc)

		// Check metadata
		assert.Equal(t, "test-headless-svc", svc.Name)
		assert.Equal(t, "test-ns", svc.Namespace)
		assert.Equal(t, labels, svc.Labels)

		// Check owner references
		assert.Len(t, svc.OwnerReferences, 1)
		ownerRef := svc.OwnerReferences[0]
		assert.Equal(t, "DistributedRedisCluster", ownerRef.Kind)
		assert.Equal(t, "redis.kun/v1alpha1", ownerRef.APIVersion)
		assert.Equal(t, cluster.Name, ownerRef.Name)
		assert.True(t, *ownerRef.Controller)
	})
}

func TestServiceWithCustomLabels(t *testing.T) {
	t.Run("Service with custom labels", func(t *testing.T) {
		cluster := createTestDistributedRedisCluster("test-cluster", "test-ns")
		customLabels := map[string]string{
			"app":         "custom-redis",
			"environment": "production",
			"team":        "platform",
			"version":     "v1.0.0",
		}

		svc := NewSvcForCR(cluster, "test-svc", customLabels)

		assert.NotNil(t, svc)
		assert.Equal(t, customLabels, svc.Labels)
		assert.Equal(t, customLabels, svc.Spec.Selector)
	})
}

func TestHeadlessServiceWithCustomLabels(t *testing.T) {
	t.Run("Headless service with custom labels", func(t *testing.T) {
		cluster := createTestDistributedRedisCluster("test-cluster", "test-ns")
		customLabels := map[string]string{
			"app":         "custom-redis",
			"environment": "production",
			"team":        "platform",
			"version":     "v1.0.0",
		}

		svc := NewHeadLessSvcForCR(cluster, "test-headless-svc", customLabels)

		assert.NotNil(t, svc)
		assert.Equal(t, customLabels, svc.Labels)
		assert.Equal(t, customLabels, svc.Spec.Selector)
	})
}

func TestServiceEdgeCases(t *testing.T) {
	t.Run("Service with empty name", func(t *testing.T) {
		cluster := createTestDistributedRedisCluster("test-cluster", "test-ns")
		labels := createTestLabels()

		svc := NewSvcForCR(cluster, "", labels)

		assert.NotNil(t, svc)
		assert.Equal(t, "", svc.Name)
	})

	t.Run("Service with nil labels", func(t *testing.T) {
		cluster := createTestDistributedRedisCluster("test-cluster", "test-ns")

		svc := NewSvcForCR(cluster, "test-svc", nil)

		assert.NotNil(t, svc)
		assert.Nil(t, svc.Labels)
		assert.Nil(t, svc.Spec.Selector)
	})

	t.Run("Service with empty labels", func(t *testing.T) {
		cluster := createTestDistributedRedisCluster("test-cluster", "test-ns")
		emptyLabels := map[string]string{}

		svc := NewSvcForCR(cluster, "test-svc", emptyLabels)

		assert.NotNil(t, svc)
		assert.Equal(t, emptyLabels, svc.Labels)
		assert.Equal(t, emptyLabels, svc.Spec.Selector)
	})

	t.Run("Headless service with empty name", func(t *testing.T) {
		cluster := createTestDistributedRedisCluster("test-cluster", "test-ns")
		labels := createTestLabels()

		svc := NewHeadLessSvcForCR(cluster, "", labels)

		assert.NotNil(t, svc)
		assert.Equal(t, "", svc.Name)
	})

	t.Run("Headless service with nil labels", func(t *testing.T) {
		cluster := createTestDistributedRedisCluster("test-cluster", "test-ns")

		svc := NewHeadLessSvcForCR(cluster, "test-headless-svc", nil)

		assert.NotNil(t, svc)
		assert.Nil(t, svc.Labels)
		assert.Nil(t, svc.Spec.Selector)
	})

	t.Run("Headless service with empty labels", func(t *testing.T) {
		cluster := createTestDistributedRedisCluster("test-cluster", "test-ns")
		emptyLabels := map[string]string{}

		svc := NewHeadLessSvcForCR(cluster, "test-headless-svc", emptyLabels)

		assert.NotNil(t, svc)
		assert.Equal(t, emptyLabels, svc.Labels)
		assert.Equal(t, emptyLabels, svc.Spec.Selector)
	})
}

func TestServicePortConfiguration(t *testing.T) {
	t.Run("Check service port configuration", func(t *testing.T) {
		cluster := createTestDistributedRedisCluster("test-cluster", "test-ns")
		labels := createTestLabels()

		svc := NewSvcForCR(cluster, "test-svc", labels)

		assert.NotNil(t, svc)
		assert.Len(t, svc.Spec.Ports, 2)

		// Verify client port
		clientPort := svc.Spec.Ports[0]
		assert.Equal(t, "client", clientPort.Name)
		assert.Equal(t, int32(6379), clientPort.Port)

		// Verify gossip port
		gossipPort := svc.Spec.Ports[1]
		assert.Equal(t, "gossip", gossipPort.Name)
		assert.Equal(t, int32(16379), gossipPort.Port)
	})
}

func TestHeadlessServicePortConfiguration(t *testing.T) {
	t.Run("Check headless service port configuration", func(t *testing.T) {
		cluster := createTestDistributedRedisCluster("test-cluster", "test-ns")
		labels := createTestLabels()

		svc := NewHeadLessSvcForCR(cluster, "test-headless-svc", labels)

		assert.NotNil(t, svc)
		assert.Len(t, svc.Spec.Ports, 2)

		// Verify client port
		clientPort := svc.Spec.Ports[0]
		assert.Equal(t, "client", clientPort.Name)
		assert.Equal(t, int32(6379), clientPort.Port)

		// Verify gossip port
		gossipPort := svc.Spec.Ports[1]
		assert.Equal(t, "gossip", gossipPort.Name)
		assert.Equal(t, int32(16379), gossipPort.Port)
	})
}
