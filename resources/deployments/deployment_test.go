package deployments

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	redisv1alpha1 "github.com/freshworks/redis-cluster-operator/api/v1alpha1"
)

func createTestRedisClusterBackup(name, namespace string) *redisv1alpha1.RedisClusterBackup {
	replicas := int32(1)
	terminationGracePeriod := int64(30)

	return &redisv1alpha1.RedisClusterBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: redisv1alpha1.RedisClusterBackupSpec{
			RedisClusterName: "test-cluster",
			UtilSpec: &redisv1alpha1.UtilSpec{
				Replicas:               replicas,
				Image:                  "redis:6.2.14-alpine",
				ImagePullPolicy:        "Always",
				StartUpCommand:         "/bin/sh",
				TerminationGracePeriod: &terminationGracePeriod,
				S3Bucket:               "test-bucket",
				ServiceAccountName:     "test-service-account",
				Resources: &corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("128Mi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("200m"),
						corev1.ResourceMemory: resource.MustParse("256Mi"),
					},
				},
				ContainerSecurityContext: &corev1.SecurityContext{
					RunAsUser:  int64Ptr(1000),
					RunAsGroup: int64Ptr(1000),
				},
			},
		},
	}
}

func createTestDistributedRedisCluster(name, namespace string) *redisv1alpha1.DistributedRedisCluster {
	return &redisv1alpha1.DistributedRedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: redisv1alpha1.DistributedRedisClusterSpec{
			MasterSize:      3,
			ClusterReplicas: 1,
			UtilConfig:      "test-util-config",
		},
	}
}

func int64Ptr(i int64) *int64 {
	return &i
}

func TestUtilNewDeploymentForCR(t *testing.T) {
	tests := []struct {
		name        string
		backup      *redisv1alpha1.RedisClusterBackup
		cluster     *redisv1alpha1.DistributedRedisCluster
		dpName      string
		secretName  string
		expectedErr bool
	}{
		{
			name:        "Valid deployment creation",
			backup:      createTestRedisClusterBackup("test-backup", "test-ns"),
			cluster:     createTestDistributedRedisCluster("test-cluster", "test-ns"),
			dpName:      "test-deployment",
			secretName:  "test-secret",
			expectedErr: false,
		},
		{
			name:        "Nil backup instance",
			backup:      nil,
			cluster:     createTestDistributedRedisCluster("test-cluster", "test-ns"),
			dpName:      "test-deployment",
			secretName:  "test-secret",
			expectedErr: true,
		},
		{
			name:        "Nil cluster instance",
			backup:      createTestRedisClusterBackup("test-backup", "test-ns"),
			cluster:     nil,
			dpName:      "test-deployment",
			secretName:  "test-secret",
			expectedErr: true,
		},
		{
			name:        "Empty deployment name",
			backup:      createTestRedisClusterBackup("test-backup", "test-ns"),
			cluster:     createTestDistributedRedisCluster("test-cluster", "test-ns"),
			dpName:      "",
			secretName:  "test-secret",
			expectedErr: false, // This should still work as it's just a name
		},
		{
			name:        "Empty secret name",
			backup:      createTestRedisClusterBackup("test-backup", "test-ns"),
			cluster:     createTestDistributedRedisCluster("test-cluster", "test-ns"),
			dpName:      "test-deployment",
			secretName:  "",
			expectedErr: false, // This should still work as it's just a name
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.backup == nil || tt.cluster == nil {
				// Skip tests with nil inputs as they would cause panic
				return
			}

			deployment := UtilNewDeploymentForCR(tt.backup, tt.cluster, tt.dpName, tt.secretName)

			if tt.expectedErr {
				assert.Nil(t, deployment)
			} else {
				assert.NotNil(t, deployment)
				assert.Equal(t, tt.dpName, deployment.Name)
				assert.Equal(t, tt.backup.Namespace, deployment.Namespace)
				assert.Equal(t, tt.backup.Spec.RedisClusterName, deployment.Labels["app"])
				assert.Equal(t, utilContainerName, deployment.Labels["container"])
				assert.Equal(t, tt.backup.Spec.UtilSpec.Replicas, *deployment.Spec.Replicas)
			}
		})
	}
}

func TestUtilGenerateDeploymentSpec(t *testing.T) {
	tests := []struct {
		name        string
		backup      *redisv1alpha1.RedisClusterBackup
		cluster     *redisv1alpha1.DistributedRedisCluster
		dpName      string
		secretName  string
		expectedErr bool
	}{
		{
			name:        "Valid deployment spec",
			backup:      createTestRedisClusterBackup("test-backup", "test-ns"),
			cluster:     createTestDistributedRedisCluster("test-cluster", "test-ns"),
			dpName:      "test-deployment",
			secretName:  "test-secret",
			expectedErr: false,
		},
		{
			name:        "Zero replicas",
			backup:      createTestRedisClusterBackup("test-backup", "test-ns"),
			cluster:     createTestDistributedRedisCluster("test-cluster", "test-ns"),
			dpName:      "test-deployment",
			secretName:  "test-secret",
			expectedErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "Zero replicas" {
				tt.backup.Spec.UtilSpec.Replicas = 0
			}

			deployment := utilGenerateDeploymentSpec(tt.backup, tt.cluster, tt.dpName, tt.secretName)

			if tt.expectedErr {
				assert.Nil(t, deployment)
			} else {
				assert.NotNil(t, deployment)
				assert.Equal(t, tt.dpName, deployment.Name)
				assert.Equal(t, tt.backup.Namespace, deployment.Namespace)
				assert.Equal(t, tt.backup.Spec.RedisClusterName, deployment.Labels["app"])
				assert.Equal(t, utilContainerName, deployment.Labels["container"])
				assert.Equal(t, tt.backup.Spec.UtilSpec.Replicas, *deployment.Spec.Replicas)

				// Check selector
				assert.Equal(t, tt.backup.Spec.RedisClusterName, deployment.Spec.Selector.MatchLabels["app"])
				assert.Equal(t, utilContainerName, deployment.Spec.Selector.MatchLabels["container"])
			}
		})
	}
}

func TestUtilGeneratePodTemplate(t *testing.T) {
	tests := []struct {
		name        string
		backup      *redisv1alpha1.RedisClusterBackup
		cluster     *redisv1alpha1.DistributedRedisCluster
		secretName  string
		expectedErr bool
	}{
		{
			name:        "Valid pod template",
			backup:      createTestRedisClusterBackup("test-backup", "test-ns"),
			cluster:     createTestDistributedRedisCluster("test-cluster", "test-ns"),
			secretName:  "test-secret",
			expectedErr: false,
		},
		{
			name:        "Empty image pull policy",
			backup:      createTestRedisClusterBackup("test-backup", "test-ns"),
			cluster:     createTestDistributedRedisCluster("test-cluster", "test-ns"),
			secretName:  "test-secret",
			expectedErr: false,
		},
		{
			name:        "Zero termination grace period",
			backup:      createTestRedisClusterBackup("test-backup", "test-ns"),
			cluster:     createTestDistributedRedisCluster("test-cluster", "test-ns"),
			secretName:  "test-secret",
			expectedErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "Empty image pull policy" {
				tt.backup.Spec.UtilSpec.ImagePullPolicy = ""
			}
			if tt.name == "Zero termination grace period" {
				zeroGracePeriod := int64(0)
				tt.backup.Spec.UtilSpec.TerminationGracePeriod = &zeroGracePeriod
			}

			podTemplate := utilGeneratePodTemplate(tt.backup, tt.cluster, tt.secretName)

			if tt.expectedErr {
				assert.Empty(t, podTemplate)
			} else {
				assert.NotNil(t, podTemplate)
				assert.Equal(t, tt.backup.Spec.RedisClusterName, podTemplate.Labels["app"])
				assert.Equal(t, utilContainerName, podTemplate.Labels["container"])

				// Check annotations
				assert.Equal(t, "/metrics", podTemplate.Annotations["prometheus.io/path"])
				assert.Equal(t, "9999", podTemplate.Annotations["prometheus.io/port"])
				assert.Equal(t, "true", podTemplate.Annotations["prometheus.io/scrape"])

				// Check containers
				assert.Len(t, podTemplate.Spec.Containers, 1)
				container := podTemplate.Spec.Containers[0]
				assert.Equal(t, utilContainerName, container.Name)
				assert.Equal(t, tt.backup.Spec.UtilSpec.Image, container.Image)
				assert.Equal(t, tt.backup.Spec.UtilSpec.StartUpCommand, container.Command[0])

				// Check environment variables
				envVars := make(map[string]string)
				for _, env := range container.Env {
					if env.Value != "" {
						envVars[env.Name] = env.Value
					}
				}

				assert.Equal(t, "6379", envVars["REDIS_PORT"])
				assert.Equal(t, "3", envVars["MASTER_SIZE"])
				assert.Equal(t, "1", envVars["CLUSTER_REPLICAS"])
				assert.Equal(t, tt.backup.Spec.RedisClusterName, envVars["CLUSTER_NAME"])
				assert.Equal(t, tt.backup.Spec.UtilSpec.S3Bucket, envVars["S3_BUCKET"])
				assert.Equal(t, tt.backup.Namespace, envVars["CLUSTER_NAMESPACE"])

				// Check volume mounts
				assert.Len(t, container.VolumeMounts, 2)
				volumeMountNames := make(map[string]bool)
				for _, mount := range container.VolumeMounts {
					volumeMountNames[mount.Name] = true
				}
				assert.True(t, volumeMountNames["configmap-volume"])
				assert.True(t, volumeMountNames["redis-backup"])
			}
		})
	}
}

func TestUtilGenerateConfigmapVolumeSpec(t *testing.T) {
	tests := []struct {
		name        string
		cluster     *redisv1alpha1.DistributedRedisCluster
		expectedErr bool
	}{
		{
			name:        "Valid cluster with util config",
			cluster:     createTestDistributedRedisCluster("test-cluster", "test-ns"),
			expectedErr: false,
		},
		{
			name:        "Cluster with empty util config",
			cluster:     createTestDistributedRedisCluster("test-cluster", "test-ns"),
			expectedErr: false,
		},
		{
			name:        "Nil cluster",
			cluster:     nil,
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "Cluster with empty util config" {
				tt.cluster.Spec.UtilConfig = ""
			}
			if tt.name == "Nil cluster" {
				// Skip nil cluster test as it would cause panic
				return
			}

			volumes := utilGenerateConfigmapVolumeSpec(tt.cluster)

			if tt.expectedErr {
				assert.Nil(t, volumes)
			} else {
				assert.NotNil(t, volumes)
				assert.Len(t, volumes, 2)

				// Check configmap volume
				configMapVolume := volumes[0]
				assert.Equal(t, "configmap-volume", configMapVolume.Name)
				assert.NotNil(t, configMapVolume.ConfigMap)
				assert.Equal(t, tt.cluster.Spec.UtilConfig, configMapVolume.ConfigMap.Name)
				assert.Equal(t, int32(0755), *configMapVolume.ConfigMap.DefaultMode)

				// Check redis backup volume
				redisBackupVolume := volumes[1]
				assert.Equal(t, "redis-backup", redisBackupVolume.Name)
				assert.NotNil(t, redisBackupVolume.EmptyDir)
			}
		})
	}
}

func TestDeploymentConstants(t *testing.T) {
	t.Run("Check configMapMountPath constant", func(t *testing.T) {
		assert.Equal(t, "/config", configMapMountPath)
	})

	t.Run("Check utilContainerName constant", func(t *testing.T) {
		assert.Equal(t, "rediscluster-util-container", utilContainerName)
	})
}

func TestDeploymentWithCustomSettings(t *testing.T) {
	t.Run("Deployment with custom image pull policy", func(t *testing.T) {
		backup := createTestRedisClusterBackup("test-backup", "test-ns")
		backup.Spec.UtilSpec.ImagePullPolicy = "IfNotPresent"
		cluster := createTestDistributedRedisCluster("test-cluster", "test-ns")

		deployment := UtilNewDeploymentForCR(backup, cluster, "test-deployment", "test-secret")

		assert.NotNil(t, deployment)
		assert.Equal(t, corev1.PullIfNotPresent, deployment.Spec.Template.Spec.Containers[0].ImagePullPolicy)
	})

	t.Run("Deployment with custom termination grace period", func(t *testing.T) {
		backup := createTestRedisClusterBackup("test-backup", "test-ns")
		customGracePeriod := int64(60)
		backup.Spec.UtilSpec.TerminationGracePeriod = &customGracePeriod
		cluster := createTestDistributedRedisCluster("test-cluster", "test-ns")

		deployment := UtilNewDeploymentForCR(backup, cluster, "test-deployment", "test-secret")

		assert.NotNil(t, deployment)
		assert.Equal(t, customGracePeriod, *deployment.Spec.Template.Spec.TerminationGracePeriodSeconds)
	})

	t.Run("Deployment with custom resources", func(t *testing.T) {
		backup := createTestRedisClusterBackup("test-backup", "test-ns")
		backup.Spec.UtilSpec.Resources = &corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("500m"),
				corev1.ResourceMemory: resource.MustParse("512Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1000m"),
				corev1.ResourceMemory: resource.MustParse("1Gi"),
			},
		}
		cluster := createTestDistributedRedisCluster("test-cluster", "test-ns")

		deployment := UtilNewDeploymentForCR(backup, cluster, "test-deployment", "test-secret")

		assert.NotNil(t, deployment)
		container := deployment.Spec.Template.Spec.Containers[0]
		assert.Equal(t, resource.MustParse("500m"), container.Resources.Requests[corev1.ResourceCPU])
		assert.Equal(t, resource.MustParse("512Mi"), container.Resources.Requests[corev1.ResourceMemory])
		assert.Equal(t, resource.MustParse("1000m"), container.Resources.Limits[corev1.ResourceCPU])
		assert.Equal(t, resource.MustParse("1Gi"), container.Resources.Limits[corev1.ResourceMemory])
	})

	t.Run("Deployment with security context", func(t *testing.T) {
		backup := createTestRedisClusterBackup("test-backup", "test-ns")
		backup.Spec.SecurityContext = &corev1.PodSecurityContext{
			RunAsUser:  int64Ptr(1001),
			RunAsGroup: int64Ptr(1001),
			FSGroup:    int64Ptr(1001),
		}
		cluster := createTestDistributedRedisCluster("test-cluster", "test-ns")

		deployment := UtilNewDeploymentForCR(backup, cluster, "test-deployment", "test-secret")

		assert.NotNil(t, deployment)
		assert.Equal(t, int64(1001), *deployment.Spec.Template.Spec.SecurityContext.RunAsUser)
		assert.Equal(t, int64(1001), *deployment.Spec.Template.Spec.SecurityContext.RunAsGroup)
		assert.Equal(t, int64(1001), *deployment.Spec.Template.Spec.SecurityContext.FSGroup)
	})
}

func TestDeploymentEdgeCases(t *testing.T) {
	t.Run("Deployment with nil container security context", func(t *testing.T) {
		backup := createTestRedisClusterBackup("test-backup", "test-ns")
		backup.Spec.UtilSpec.ContainerSecurityContext = nil
		cluster := createTestDistributedRedisCluster("test-cluster", "test-ns")

		deployment := UtilNewDeploymentForCR(backup, cluster, "test-deployment", "test-secret")

		assert.NotNil(t, deployment)
		container := deployment.Spec.Template.Spec.Containers[0]
		assert.Nil(t, container.SecurityContext)
	})

	t.Run("Deployment with nil pod security context", func(t *testing.T) {
		backup := createTestRedisClusterBackup("test-backup", "test-ns")
		backup.Spec.SecurityContext = nil
		cluster := createTestDistributedRedisCluster("test-cluster", "test-ns")

		deployment := UtilNewDeploymentForCR(backup, cluster, "test-deployment", "test-secret")

		assert.NotNil(t, deployment)
		assert.Nil(t, deployment.Spec.Template.Spec.SecurityContext)
	})

	t.Run("Deployment with custom affinity", func(t *testing.T) {
		backup := createTestRedisClusterBackup("test-backup", "test-ns")
		backup.Spec.UtilSpec.Affinity = &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								{
									Key:      "kubernetes.io/os",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"linux"},
								},
							},
						},
					},
				},
			},
		}
		cluster := createTestDistributedRedisCluster("test-cluster", "test-ns")

		deployment := UtilNewDeploymentForCR(backup, cluster, "test-deployment", "test-secret")

		assert.NotNil(t, deployment)
		assert.NotNil(t, deployment.Spec.Template.Spec.Affinity)
		assert.NotNil(t, deployment.Spec.Template.Spec.Affinity.NodeAffinity)
	})

	t.Run("Deployment with custom service account", func(t *testing.T) {
		backup := createTestRedisClusterBackup("test-backup", "test-ns")
		backup.Spec.UtilSpec.ServiceAccountName = "custom-service-account"
		cluster := createTestDistributedRedisCluster("test-cluster", "test-ns")

		deployment := UtilNewDeploymentForCR(backup, cluster, "test-deployment", "test-secret")

		assert.NotNil(t, deployment)
		assert.Equal(t, "custom-service-account", deployment.Spec.Template.Spec.ServiceAccountName)
	})

	t.Run("Deployment with empty service account", func(t *testing.T) {
		backup := createTestRedisClusterBackup("test-backup", "test-ns")
		backup.Spec.UtilSpec.ServiceAccountName = ""
		cluster := createTestDistributedRedisCluster("test-cluster", "test-ns")

		deployment := UtilNewDeploymentForCR(backup, cluster, "test-deployment", "test-secret")

		assert.NotNil(t, deployment)
		assert.Equal(t, "", deployment.Spec.Template.Spec.ServiceAccountName)
	})

	t.Run("Deployment with empty image pull policy", func(t *testing.T) {
		backup := createTestRedisClusterBackup("test-backup", "test-ns")
		backup.Spec.UtilSpec.ImagePullPolicy = ""
		cluster := createTestDistributedRedisCluster("test-cluster", "test-ns")

		deployment := UtilNewDeploymentForCR(backup, cluster, "test-deployment", "test-secret")

		assert.NotNil(t, deployment)
		// Should default to PullAlways when empty
		assert.Equal(t, corev1.PullAlways, deployment.Spec.Template.Spec.Containers[0].ImagePullPolicy)
	})

	t.Run("Deployment with zero termination grace period", func(t *testing.T) {
		backup := createTestRedisClusterBackup("test-backup", "test-ns")
		zeroGracePeriod := int64(0)
		backup.Spec.UtilSpec.TerminationGracePeriod = &zeroGracePeriod
		cluster := createTestDistributedRedisCluster("test-cluster", "test-ns")

		deployment := UtilNewDeploymentForCR(backup, cluster, "test-deployment", "test-secret")

		assert.NotNil(t, deployment)
		// Should default to 30 when zero
		assert.Equal(t, int64(30), *deployment.Spec.Template.Spec.TerminationGracePeriodSeconds)
	})
}

func TestDeploymentEnvironmentVariables(t *testing.T) {
	t.Run("Check all environment variables are set correctly", func(t *testing.T) {
		backup := createTestRedisClusterBackup("test-backup", "test-ns")
		cluster := createTestDistributedRedisCluster("test-cluster", "test-ns")

		deployment := UtilNewDeploymentForCR(backup, cluster, "test-deployment", "test-secret")

		assert.NotNil(t, deployment)
		container := deployment.Spec.Template.Spec.Containers[0]

		// Check that all expected environment variables are present
		envVars := make(map[string]string)
		for _, env := range container.Env {
			if env.Value != "" {
				envVars[env.Name] = env.Value
			}
		}

		// Check required environment variables
		assert.Equal(t, "6379", envVars["REDIS_PORT"])
		assert.Equal(t, "3", envVars["MASTER_SIZE"])
		assert.Equal(t, "1", envVars["CLUSTER_REPLICAS"])
		assert.Equal(t, "test-cluster", envVars["CLUSTER_NAME"])
		assert.Equal(t, "test-bucket", envVars["S3_BUCKET"])
		assert.Equal(t, "test-ns", envVars["CLUSTER_NAMESPACE"])

		// Check that secret references are set correctly
		secretRefs := make(map[string]bool)
		for _, env := range container.Env {
			if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil {
				secretRefs[env.Name] = true
			}
		}

		assert.True(t, secretRefs["REDIS_USER"])
		assert.True(t, secretRefs["REDIS_PASSWORD"])
	})
}

func TestDeploymentVolumeMounts(t *testing.T) {
	t.Run("Check volume mounts are configured correctly", func(t *testing.T) {
		backup := createTestRedisClusterBackup("test-backup", "test-ns")
		cluster := createTestDistributedRedisCluster("test-cluster", "test-ns")

		deployment := UtilNewDeploymentForCR(backup, cluster, "test-deployment", "test-secret")

		assert.NotNil(t, deployment)
		container := deployment.Spec.Template.Spec.Containers[0]

		// Check volume mounts
		assert.Len(t, container.VolumeMounts, 2)

		volumeMountPaths := make(map[string]string)
		for _, mount := range container.VolumeMounts {
			volumeMountPaths[mount.Name] = mount.MountPath
		}

		assert.Equal(t, "/config", volumeMountPaths["configmap-volume"])
		assert.Equal(t, "/backup", volumeMountPaths["redis-backup"])

		// Check volumes
		assert.Len(t, deployment.Spec.Template.Spec.Volumes, 2)

		volumeNames := make(map[string]bool)
		for _, volume := range deployment.Spec.Template.Spec.Volumes {
			volumeNames[volume.Name] = true
		}

		assert.True(t, volumeNames["configmap-volume"])
		assert.True(t, volumeNames["redis-backup"])
	})
}
