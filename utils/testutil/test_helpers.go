package testutil

import (
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	redisv1alpha1 "github.com/dinesh-murugiah/rediscluster-operator/api/v1alpha1"
)

// CreateTestCluster creates a test DistributedRedisCluster with common defaults
func CreateTestCluster(name, namespace string) *redisv1alpha1.DistributedRedisCluster {
	// Ensure name has the required prefix
	if !strings.HasPrefix(name, redisv1alpha1.CrResourcesprefix) {
		name = redisv1alpha1.CrResourcesprefix + name
	}

	return &redisv1alpha1.DistributedRedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				redisv1alpha1.LabelManagedByKey: redisv1alpha1.OperatorName,
			},
		},
		Spec: redisv1alpha1.DistributedRedisClusterSpec{
			MasterSize:      3,
			ClusterReplicas: 1,
			Image:           "redis:7.4.4-alpine",
			ServiceName:     name,
			Resources: &corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("2Gi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("1000m"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
				},
			},
			HaConfig: &redisv1alpha1.HAspec{
				HaEnabled: true,
				ZonesInfo: map[string]string{
					"us-west-2a": "zone1",
					"us-west-2b": "zone2",
					"us-west-2c": "zone3",
				},
			},
		},
		Status: redisv1alpha1.DistributedRedisClusterStatus{
			Status: redisv1alpha1.ClusterStatusOK,
		},
	}
}

// CreateTestClusterWithHAConfig creates a test cluster with specific HA configuration
func CreateTestClusterWithHAConfig(name, namespace string, haEnabled bool, zones map[string]string) *redisv1alpha1.DistributedRedisCluster {
	cluster := CreateTestCluster(name, namespace)
	cluster.Spec.HaConfig = &redisv1alpha1.HAspec{
		HaEnabled: haEnabled,
		ZonesInfo: zones,
	}
	return cluster
}

// CreateTestClusterWithStorage creates a test cluster with storage configuration
func CreateTestClusterWithStorage(name, namespace string, confStorageType, backupStorageType redisv1alpha1.StorageType) *redisv1alpha1.DistributedRedisCluster {
	cluster := CreateTestCluster(name, namespace)

	if confStorageType != "" {
		cluster.Spec.ConfStorage = &redisv1alpha1.RedisStorage{
			Type:  confStorageType,
			Class: "test-storage-class",
			Size:  resource.MustParse("1Gi"),
		}
	}

	if backupStorageType != "" {
		cluster.Spec.BackupStorage = &redisv1alpha1.RedisStorage{
			Type:  backupStorageType,
			Class: "test-storage-class",
			Size:  resource.MustParse("5Gi"),
		}
	}

	return cluster
}

// CreateTestPod creates a test pod with common defaults
func CreateTestPod(name, namespace, ip, nodeName string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app": "redis",
			},
		},
		Spec: corev1.PodSpec{
			NodeName: nodeName,
			Containers: []corev1.Container{
				{
					Name: "redis",
					Ports: []corev1.ContainerPort{
						{
							Name:          "client",
							ContainerPort: 6379,
						},
					},
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			PodIP: ip,
		},
	}
}

// CreateTestPodWithOwner creates a test pod with owner reference
func CreateTestPodWithOwner(name, namespace, ip, nodeName, ownerKind, ownerName string) *corev1.Pod {
	pod := CreateTestPod(name, namespace, ip, nodeName)
	pod.OwnerReferences = []metav1.OwnerReference{
		{
			Kind: ownerKind,
			Name: ownerName,
		},
	}
	return pod
}

// CreateTestNode creates a test Kubernetes node
func CreateTestNode(name, zone string, ready bool) *corev1.Node {
	status := corev1.ConditionTrue
	if !ready {
		status = corev1.ConditionFalse
	}

	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"topology.kubernetes.io/zone": zone,
			},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: status,
				},
			},
		},
	}
}

// Note: Redis-related helper functions have been moved to avoid circular imports
// These functions should be defined in the redisutil package test files directly

// CreateTestStatus creates a test cluster status
func CreateTestStatus(status redisv1alpha1.ClusterStatus, reason string, masters int32, minRep, maxRep int32) *redisv1alpha1.DistributedRedisClusterStatus {
	return &redisv1alpha1.DistributedRedisClusterStatus{
		Status:               status,
		Reason:               reason,
		NumberOfMaster:       masters,
		MinReplicationFactor: minRep,
		MaxReplicationFactor: maxRep,
		HAStatus:             redisv1alpha1.HaStatusHealthy,
		SecretStatus:         "ACLintact",
		SecretsVer:           map[string]string{"secret1": "v1"},
	}
}

// CreateReadyPod creates a test pod that is explicitly ready
func CreateReadyPod(name, namespace, ip, nodeName string) *corev1.Pod {
	pod := CreateTestPod(name, namespace, ip, nodeName)
	pod.Status.Conditions = []corev1.PodCondition{
		{
			Type:   corev1.PodReady,
			Status: corev1.ConditionTrue,
		},
	}
	return pod
}

// ParseQuantity is a helper to parse resource quantities for tests
func ParseQuantity(value string) resource.Quantity {
	q, _ := resource.ParseQuantity(value)
	return q
}

// CreateStatefulSetList creates a StatefulSetList with the specified number of items
func CreateStatefulSetList(count int) *appsv1.StatefulSetList {
	items := make([]appsv1.StatefulSet, count)
	for i := 0; i < count; i++ {
		items[i] = *TestStatefulSet(fmt.Sprintf("test-sts-%d", i), "default", 1)
	}
	return &appsv1.StatefulSetList{
		Items: items,
	}
}
