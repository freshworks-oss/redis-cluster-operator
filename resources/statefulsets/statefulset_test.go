package statefulsets

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	redisv1alpha1 "github.com/freshworks/redis-cluster-operator/api/v1alpha1"
)

// Test helper functions from statefulset_extended_test.go
func createTestDistributedRedisCluster(name, namespace string) *redisv1alpha1.DistributedRedisCluster {
	return &redisv1alpha1.DistributedRedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: redisv1alpha1.DistributedRedisClusterSpec{
			MasterSize:      3,
			ClusterReplicas: 1,
			AdminSecret: &corev1.LocalObjectReference{
				Name: "test-admin-secret",
			},
			Image: "redis:7.0-alpine",
			// Add required fields to avoid nil pointer dereference
			Config: map[string]string{
				"maxmemory": "2gb",
			},
			// Add storage configuration to avoid nil pointer dereference
			ConfStorage: &redisv1alpha1.RedisStorage{
				Size:  resource.MustParse("1Gi"),
				Type:  redisv1alpha1.Ephemeral,
				Class: "standard",
			},
			BackupStorage: &redisv1alpha1.RedisStorage{
				Size:  resource.MustParse("1Gi"),
				Type:  redisv1alpha1.Ephemeral,
				Class: "standard",
			},
			// Add resources to avoid nil pointer dereference
			Resources: &corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("128Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("500m"),
					corev1.ResourceMemory: resource.MustParse("512Mi"),
				},
			},
		},
	}
}

func createTestDistributedRedisClusterWithStorage(name, namespace string) *redisv1alpha1.DistributedRedisCluster {
	return &redisv1alpha1.DistributedRedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: redisv1alpha1.DistributedRedisClusterSpec{
			MasterSize:      3,
			ClusterReplicas: 1,
			AdminSecret: &corev1.LocalObjectReference{
				Name: "test-admin-secret",
			},
			Image: "redis:7.0-alpine",
			Config: map[string]string{
				"maxmemory": "2gb",
			},
			ConfStorage: &redisv1alpha1.RedisStorage{
				Size:  resource.MustParse("1Gi"),
				Type:  redisv1alpha1.PersistentClaimEbs,
				Class: "standard",
			},
			BackupStorage: &redisv1alpha1.RedisStorage{
				Size:  resource.MustParse("1Gi"),
				Type:  redisv1alpha1.PersistentClaimEfs,
				Class: "standard",
			},
			// Add resources to avoid nil pointer dereference
			Resources: &corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("128Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("500m"),
					corev1.ResourceMemory: resource.MustParse("512Mi"),
				},
			},
		},
	}
}

func createTestLabels() map[string]string {
	return map[string]string{
		"app":       "redis-cluster",
		"component": "redis",
		"release":   "test-release",
	}
}

// Original tests from statefulset_test.go
func Test_mergeRenameCmds(t *testing.T) {
	type args struct {
		userCmds           []string
		systemRenameCmdMap map[string]string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "test No intersection",
			args: args{
				userCmds: []string{
					"--maxmemory 2gb",
					"--rename-command BGSAVE pp14qluk",
					"--rename-command CONFIG lni07z1p",
				},
				systemRenameCmdMap: map[string]string{
					"SAVE":  "6on30p6z",
					"DEBUG": "8a4insyv",
				},
			},
			want: []string{
				"--maxmemory 2gb",
				"--rename-command BGSAVE pp14qluk",
				"--rename-command CONFIG lni07z1p",
				"--rename-command DEBUG 8a4insyv",
				"--rename-command SAVE 6on30p6z",
			},
		},
		{
			name: "test intersection",
			args: args{
				userCmds: []string{
					"--rename-command BGSAVE pp14qluk",
					"--rename-command CONFIG lni07z1p",
				},
				systemRenameCmdMap: map[string]string{
					"BGSAVE": "fadfgad",
					"SAVE":   "6on30p6z",
					"DEBUG":  "8a4insyv",
				},
			},
			want: []string{
				"--rename-command CONFIG lni07z1p",
				"--rename-command BGSAVE fadfgad",
				"--rename-command DEBUG 8a4insyv",
				"--rename-command SAVE 6on30p6z",
			},
		},
		{
			name: "test complex",
			args: args{
				userCmds: []string{
					"--maxmemory 2gb",
					"--rename-command BGSAVE pp14qluk",
					"--rename-command CONFIG lni07z1p",
					`--rename-command FLUSHALL ""`,
				},
				systemRenameCmdMap: map[string]string{
					"BGSAVE": "fadfgad",
					"SAVE":   "6on30p6z",
					"DEBUG":  "8a4insyv",
				},
			},
			want: []string{
				"--maxmemory 2gb",
				"--rename-command CONFIG lni07z1p",
				`--rename-command FLUSHALL ""`,
				"--rename-command BGSAVE fadfgad",
				"--rename-command DEBUG 8a4insyv",
				"--rename-command SAVE 6on30p6z",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mergeRenameCmds(tt.args.userCmds, tt.args.systemRenameCmdMap); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mergeRenameCmds() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_customContainerEnv(t *testing.T) {
	type args struct {
		env       []corev1.EnvVar
		customEnv []corev1.EnvVar
	}
	tests := []struct {
		name string
		args args
		want []corev1.EnvVar
	}{
		{
			name: "nil all",
			args: args{
				env:       nil,
				customEnv: nil,
			},
			want: nil,
		},
		{
			name: "nil env",
			args: args{
				env: nil,
				customEnv: []corev1.EnvVar{{
					Name:      "foo",
					Value:     "",
					ValueFrom: nil,
				}},
			},
			want: []corev1.EnvVar{{
				Name:      "foo",
				Value:     "",
				ValueFrom: nil,
			}},
		},
		{
			name: "nil custom env",
			args: args{
				customEnv: nil,
				env: []corev1.EnvVar{{
					Name:      "foo",
					Value:     "",
					ValueFrom: nil,
				}},
			},
			want: []corev1.EnvVar{{
				Name:      "foo",
				Value:     "",
				ValueFrom: nil,
			}},
		},
		{
			name: "env for bar",
			args: args{
				env: []corev1.EnvVar{{
					Name:      "foo",
					Value:     "",
					ValueFrom: nil,
				}},
				customEnv: []corev1.EnvVar{{
					Name:      "bar",
					Value:     "",
					ValueFrom: nil,
				}},
			},
			want: []corev1.EnvVar{{
				Name:      "foo",
				Value:     "",
				ValueFrom: nil,
			}, {
				Name:      "bar",
				Value:     "",
				ValueFrom: nil,
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := customContainerEnv(tt.args.env, tt.args.customEnv); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("customContainerEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Tests from statefulset_extended_test.go
func TestNewStatefulSetForCR(t *testing.T) {
	cluster := createTestDistributedRedisCluster("test-cluster", "test-namespace")
	labels := createTestLabels()

	statefulset, err := NewStatefulSetForCR(cluster, "test-statefulset", "test-service", labels)

	assert.NoError(t, err)
	assert.NotNil(t, statefulset)
	assert.Equal(t, "test-statefulset", statefulset.Name)
	assert.Equal(t, "test-namespace", statefulset.Namespace)
	assert.Equal(t, "test-service", statefulset.Spec.ServiceName)
	assert.Equal(t, int32(2), *statefulset.Spec.Replicas) // MasterSize + ClusterReplicas
	assert.Equal(t, appsv1.OnDeleteStatefulSetStrategyType, statefulset.Spec.UpdateStrategy.Type)
}

func TestNewStatefulSetForCRNoAdminSecret(t *testing.T) {
	cluster := createTestDistributedRedisCluster("test-cluster", "test-namespace")
	cluster.Spec.AdminSecret = nil
	labels := createTestLabels()

	statefulset, err := NewStatefulSetForCR(cluster, "test-statefulset", "test-service", labels)

	assert.Error(t, err)
	assert.Nil(t, statefulset)
	assert.Contains(t, err.Error(), "missing admin secret")
}

func TestClusterStatefulSetName(t *testing.T) {
	tests := []struct {
		name        string
		clusterName string
		i           int
		expected    string
	}{
		{
			name:        "Basic naming",
			clusterName: "redis-cluster",
			i:           0,
			expected:    "drc-redis-cluster-shard0",
		},
		{
			name:        "Multiple instances",
			clusterName: "redis-cluster",
			i:           5,
			expected:    "drc-redis-cluster-shard5",
		},
		{
			name:        "Special characters in name",
			clusterName: "redis-cluster-test",
			i:           1,
			expected:    "drc-redis-cluster-test-shard1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClusterStatefulSetName(tt.clusterName, tt.i)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClusterHeadlessSvcName(t *testing.T) {
	tests := []struct {
		name     string
		svcName  string
		i        int
		expected string
	}{
		{
			name:     "Basic naming",
			svcName:  "redis-cluster-headless",
			i:        0,
			expected: "drc-redis-cluster-headless-svc-shard0",
		},
		{
			name:     "Multiple instances",
			svcName:  "redis-cluster-headless",
			i:        3,
			expected: "drc-redis-cluster-headless-svc-shard3",
		},
		{
			name:     "Special characters in name",
			svcName:  "redis-cluster-headless-test",
			i:        2,
			expected: "drc-redis-cluster-headless-test-svc-shard2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClusterHeadlessSvcName(tt.svcName, tt.i)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsPersistentStorageType(t *testing.T) {
	tests := []struct {
		name        string
		storageType redisv1alpha1.StorageType
		expected    bool
	}{
		{
			name:        "PersistentClaimEbs",
			storageType: redisv1alpha1.PersistentClaimEbs,
			expected:    true,
		},
		{
			name:        "PersistentClaimEfs",
			storageType: redisv1alpha1.PersistentClaimEfs,
			expected:    true,
		},
		{
			name:        "Ephemeral",
			storageType: redisv1alpha1.Ephemeral,
			expected:    false,
		},
		{
			name:        "Invalid storage type",
			storageType: "invalid",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPersistentStorageType(tt.storageType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsStorageTypeSupported(t *testing.T) {
	tests := []struct {
		name        string
		storageType redisv1alpha1.StorageType
		expected    bool
	}{
		{
			name:        "PersistentClaimEbs",
			storageType: redisv1alpha1.PersistentClaimEbs,
			expected:    false,
		},
		{
			name:        "PersistentClaimEfs",
			storageType: redisv1alpha1.PersistentClaimEfs,
			expected:    true,
		},
		{
			name:        "Ephemeral",
			storageType: redisv1alpha1.Ephemeral,
			expected:    true,
		},
		{
			name:        "Invalid storage type",
			storageType: "invalid",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsStorageTypeSupported(tt.storageType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsBothStorageTypePersistent(t *testing.T) {
	tests := []struct {
		name     string
		cluster  *redisv1alpha1.DistributedRedisCluster
		expected bool
	}{
		{
			name: "Both persistent",
			cluster: &redisv1alpha1.DistributedRedisCluster{
				Spec: redisv1alpha1.DistributedRedisClusterSpec{
					ConfStorage: &redisv1alpha1.RedisStorage{
						Type: redisv1alpha1.PersistentClaimEbs,
					},
					BackupStorage: &redisv1alpha1.RedisStorage{
						Type: redisv1alpha1.PersistentClaimEfs,
					},
				},
			},
			expected: true,
		},
		{
			name: "One persistent, one ephemeral",
			cluster: &redisv1alpha1.DistributedRedisCluster{
				Spec: redisv1alpha1.DistributedRedisClusterSpec{
					ConfStorage: &redisv1alpha1.RedisStorage{
						Type: redisv1alpha1.PersistentClaimEbs,
					},
					BackupStorage: &redisv1alpha1.RedisStorage{
						Type: redisv1alpha1.Ephemeral,
					},
				},
			},
			expected: false,
		},
		{
			name: "Both ephemeral",
			cluster: &redisv1alpha1.DistributedRedisCluster{
				Spec: redisv1alpha1.DistributedRedisClusterSpec{
					ConfStorage: &redisv1alpha1.RedisStorage{
						Type: redisv1alpha1.Ephemeral,
					},
					BackupStorage: &redisv1alpha1.RedisStorage{
						Type: redisv1alpha1.Ephemeral,
					},
				},
			},
			expected: false,
		},
		{
			name: "Nil storage",
			cluster: &redisv1alpha1.DistributedRedisCluster{
				Spec: redisv1alpha1.DistributedRedisClusterSpec{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsBothStorageTypePersistent(tt.cluster)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetStorageVolumeName(t *testing.T) {
	cluster := createTestDistributedRedisCluster("test-cluster", "test-namespace")

	tests := []struct {
		name     string
		dataType string
		expected string
	}{
		{
			name:     "Conf storage",
			dataType: "redisnodeconf",
			expected: "redis-data",
		},
		{
			name:     "Backup storage",
			dataType: "redisbackupdata",
			expected: "redis-data",
		},
		{
			name:     "Data storage",
			dataType: "data",
			expected: "redis-data",
		},
		{
			name:     "Unknown data type",
			dataType: "unknown",
			expected: "redis-data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetStorageVolumeName(cluster, tt.dataType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsPersistentStorageTypeSame(t *testing.T) {
	tests := []struct {
		name         string
		storageType1 redisv1alpha1.StorageType
		storageType2 redisv1alpha1.StorageType
		expected     bool
	}{
		{
			name:         "Both persistent",
			storageType1: redisv1alpha1.PersistentClaimEbs,
			storageType2: redisv1alpha1.PersistentClaimEfs,
			expected:     false,
		},
		{
			name:         "Both ephemeral",
			storageType1: redisv1alpha1.Ephemeral,
			storageType2: redisv1alpha1.Ephemeral,
			expected:     true,
		},
		{
			name:         "Mixed types",
			storageType1: redisv1alpha1.PersistentClaimEbs,
			storageType2: redisv1alpha1.Ephemeral,
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPersistentStorageTypeSame(tt.storageType1, tt.storageType2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRedisPassword(t *testing.T) {
	cluster := createTestDistributedRedisCluster("test-cluster", "test-namespace")

	password := redisPassword(cluster)

	assert.NotNil(t, password)
	assert.Equal(t, "REDIS_PASSWORD", password.Name)
	assert.NotNil(t, password.ValueFrom)
	assert.NotNil(t, password.ValueFrom.SecretKeyRef)
	assert.Equal(t, "test-admin-secret", password.ValueFrom.SecretKeyRef.Name)
	assert.Equal(t, "password", password.ValueFrom.SecretKeyRef.Key)
}

func TestGetRedisCommand(t *testing.T) {
	cluster := createTestDistributedRedisCluster("test-cluster", "test-namespace")
	password := &corev1.EnvVar{
		Name:  "REDIS_PASSWORD",
		Value: "testpassword",
	}

	commands := getRedisCommand(cluster, password)

	assert.NotEmpty(t, commands)
	assert.Contains(t, commands, "--requirepass no")
	assert.Contains(t, commands, "--masterauth '$(REDIS_PASSWORD)'")
}

func TestVolumeMounts(t *testing.T) {
	cluster := createTestDistributedRedisCluster("test-cluster", "test-namespace")

	mounts := volumeMounts(cluster)

	assert.NotEmpty(t, mounts)

	// Check for required volume mounts
	mountNames := make(map[string]bool)
	for _, mount := range mounts {
		mountNames[mount.Name] = true
	}

	assert.True(t, mountNames["redis-data"], "redis-data mount should be present")
	assert.True(t, mountNames["redis-modules"], "redis-modules mount should be present")
}

func TestEmptyVolume(t *testing.T) {
	volumeName := "test-volume"

	volume := emptyVolume(volumeName)

	assert.NotNil(t, volume)
	assert.Equal(t, volumeName, volume.Name)
	assert.NotNil(t, volume.EmptyDir)
}

func TestIsEphemeral(t *testing.T) {
	tests := []struct {
		name     string
		storage  *redisv1alpha1.RedisStorage
		expected bool
	}{
		{
			name: "Ephemeral storage",
			storage: &redisv1alpha1.RedisStorage{
				Type: redisv1alpha1.Ephemeral,
			},
			expected: true,
		},
		{
			name: "Persistent storage",
			storage: &redisv1alpha1.RedisStorage{
				Type: redisv1alpha1.PersistentClaimEbs,
			},
			expected: false,
		},
		{
			name:     "Nil storage",
			storage:  nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isEphemeral(tt.storage)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCreatePersistentVolumeClaim(t *testing.T) {
	claimName := "test-claim"
	labels := map[string]string{"app": "redis"}
	size := resource.MustParse("1Gi")
	class := "standard"

	pvc := createPersistentVolumeClaim(claimName, labels, size, class)

	assert.Equal(t, claimName, pvc.Name)
	assert.Equal(t, labels, pvc.Labels)
	assert.Equal(t, size, pvc.Spec.Resources.Requests[corev1.ResourceStorage])
	assert.Equal(t, class, *pvc.Spec.StorageClassName)
}

func TestEfsPersistentClaim(t *testing.T) {
	cluster := createTestDistributedRedisCluster("test-cluster", "test-namespace")
	labels := createTestLabels()
	claimName := "test-claim"
	storageFor := "backup"

	pvc := efsPersistentClaim(cluster, labels, claimName, storageFor)

	// Since the test cluster has ephemeral storage, this should return an empty PVC
	assert.Equal(t, "", pvc.Name)
	assert.Nil(t, pvc.Labels)
}

// Tests from statefulset_missing_tests.go
func TestGetTopolgyspreadConstraints(t *testing.T) {
	cluster := createTestDistributedRedisCluster("test-cluster", "test-namespace")
	cluster.Spec.HaConfig = &redisv1alpha1.HAspec{
		HaEnabled: true,
	}

	constraints := getTopolgyspreadConstraints(cluster, "test-statefulset")

	assert.NotEmpty(t, constraints)
	assert.Len(t, constraints, 2) // Should have zone and hostname constraints

	// Check zone constraint
	zoneConstraint := constraints[0]
	assert.Equal(t, "topology.kubernetes.io/zone", zoneConstraint.TopologyKey)
	assert.Equal(t, "test-statefulset", zoneConstraint.LabelSelector.MatchLabels["statefulSet"])

	// Check hostname constraint
	hostnameConstraint := constraints[1]
	assert.Equal(t, "kubernetes.io/hostname", hostnameConstraint.TopologyKey)
	assert.Equal(t, "test-statefulset", hostnameConstraint.LabelSelector.MatchLabels["statefulSet"])
}

func TestGetAffinity(t *testing.T) {
	cluster := createTestDistributedRedisCluster("test-cluster", "test-namespace")
	cluster.Spec.HaConfig = &redisv1alpha1.HAspec{
		HaEnabled: true,
	}
	labels := createTestLabels()

	affinity := getAffinity(cluster, labels)

	assert.NotNil(t, affinity)
	assert.NotNil(t, affinity.PodAntiAffinity)
	assert.NotEmpty(t, affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution)

	// Check anti-affinity rule
	rule := affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution[0]
	assert.Equal(t, "kubernetes.io/hostname", rule.TopologyKey)
	assert.Equal(t, "redis-cluster", rule.LabelSelector.MatchLabels["app"])
}

func TestRedisServerContainer(t *testing.T) {
	cluster := createTestDistributedRedisCluster("test-cluster", "test-namespace")
	password := &corev1.EnvVar{
		Name:  "REDIS_PASSWORD",
		Value: "testpassword",
	}

	container := redisServerContainer(cluster, password)

	assert.Equal(t, "redis", container.Name)
	assert.Equal(t, cluster.Spec.Image, container.Image)
	assert.NotEmpty(t, container.Command)
	// Note: container.Args is empty as the command is passed as a single slice
	assert.NotEmpty(t, container.Env)
	assert.NotEmpty(t, container.VolumeMounts)
	assert.NotNil(t, container.Resources)
}

func TestRedisMonitorContainers(t *testing.T) {
	cluster := createTestDistributedRedisCluster("test-cluster", "test-namespace")
	cluster.Spec.Monitor = &[]redisv1alpha1.AgentSpec{
		{
			Name:  "exporter",
			Image: "redis-exporter:latest",
			Prometheus: &redisv1alpha1.PrometheusSpec{
				Name: "metrics",
				Port: 9121,
			},
		},
	}
	password := &corev1.EnvVar{
		Name:  "REDIS_PASSWORD",
		Value: "testpassword",
	}

	containers := redisMonitorContainers(cluster, password)

	assert.NotEmpty(t, containers)

	// Check if exporter container is present
	var exporterContainer *corev1.Container
	for i := range containers {
		if containers[i].Name == "exporter" {
			exporterContainer = &containers[i]
			break
		}
	}

	assert.NotNil(t, exporterContainer)
	assert.Equal(t, "exporter", exporterContainer.Name)
	assert.NotEmpty(t, exporterContainer.Ports)
}

func TestRedisRestoreInitContainer(t *testing.T) {
	cluster := createTestDistributedRedisCluster("test-cluster", "test-namespace")
	cluster.Spec.BackupStorage = &redisv1alpha1.RedisStorage{
		Size:  resource.MustParse("1Gi"),
		Type:  redisv1alpha1.PersistentClaimEbs,
		Class: "standard",
	}
	password := &corev1.EnvVar{
		Name:  "REDIS_PASSWORD",
		Value: "testpassword",
	}

	container, err := redisRestoreInitContainer(cluster, password)

	assert.Error(t, err)
	assert.Equal(t, "restore Not supported yet", err.Error())
	assert.Equal(t, "", container.Name)
	assert.Empty(t, container.Command)
	assert.Empty(t, container.VolumeMounts)
}

func TestGetInitContainersWithRedisEnv(t *testing.T) {
	cluster := createTestDistributedRedisCluster("test-cluster", "test-namespace")
	password := &corev1.EnvVar{
		Name:  "REDIS_PASSWORD",
		Value: "testpassword",
	}

	containers := getInitContainersWithRedisEnv(cluster, password)

	// Test cluster has no init containers, so this should be empty
	assert.Empty(t, containers)
}

func TestGetContainersWithRedisEnv(t *testing.T) {
	cluster := createTestDistributedRedisCluster("test-cluster", "test-namespace")
	password := &corev1.EnvVar{
		Name:  "REDIS_PASSWORD",
		Value: "testpassword",
	}

	baseContainers := []corev1.Container{
		{
			Name:  "test-container",
			Image: "test-image",
		},
	}

	envVars := []corev1.EnvVar{
		{
			Name:  "TEST_VAR",
			Value: "test-value",
		},
	}

	containers := getContainersWithRedisEnv(cluster, baseContainers, envVars, password)

	assert.NotEmpty(t, containers)
	assert.Len(t, containers, 1)

	container := containers[0]
	assert.Equal(t, "test-container", container.Name)
	assert.Equal(t, "test-image", container.Image)

	// Check if all env vars are present
	envNames := make(map[string]bool)
	for _, env := range container.Env {
		envNames[env.Name] = true
	}

	assert.True(t, envNames["REDIS_PASSWORD"], "REDIS_PASSWORD should be present")
	assert.True(t, envNames["TEST_VAR"], "TEST_VAR should be present")
}

func TestRedisVolumes(t *testing.T) {
	cluster := createTestDistributedRedisCluster("test-cluster", "test-namespace")

	volumes := redisVolumes(cluster)

	assert.NotEmpty(t, volumes)

	// Check for required volumes
	volumeNames := make(map[string]bool)
	for _, volume := range volumes {
		volumeNames[volume.Name] = true
	}

	assert.True(t, volumeNames["redis-data"], "redis-data volume should be present")
	assert.True(t, volumeNames["redis-modules"], "redis-modules volume should be present")
}

func TestRedisVolumesWithPersistentStorage(t *testing.T) {
	cluster := createTestDistributedRedisClusterWithStorage("test-cluster", "test-namespace")

	volumes := redisVolumes(cluster)

	assert.NotEmpty(t, volumes)

	// Check for persistent storage volumes
	volumeNames := make(map[string]bool)
	for _, volume := range volumes {
		volumeNames[volume.Name] = true
	}

	// The test cluster has persistent storage, so we get redis-backup for EFS storage
	assert.True(t, volumeNames["redis-backup"], "redis-backup volume should be present")
	assert.True(t, volumeNames["redis-modules"], "redis-modules volume should be present")
}

func TestGetTopolgyspreadConstraintsDisabled(t *testing.T) {
	cluster := createTestDistributedRedisCluster("test-cluster", "test-namespace")
	// No HaConfig or HaEnabled false

	constraints := getTopolgyspreadConstraints(cluster, "test-statefulset")

	// The function always returns constraints regardless of HA configuration
	assert.NotEmpty(t, constraints)
	assert.Len(t, constraints, 2)
}

func TestGetAffinityWithCustomSettings(t *testing.T) {
	cluster := createTestDistributedRedisCluster("test-cluster", "test-namespace")
	cluster.Spec.Affinity = &corev1.Affinity{
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
	labels := createTestLabels()

	affinity := getAffinity(cluster, labels)

	// When HA is not enabled and custom affinity is provided, it should return the custom affinity
	assert.NotNil(t, affinity)
	assert.NotNil(t, affinity.NodeAffinity)
}

func TestRedisServerContainerWithCustomResources(t *testing.T) {
	cluster := createTestDistributedRedisCluster("test-cluster", "test-namespace")
	cluster.Spec.Resources = &corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("200m"),
			corev1.ResourceMemory: resource.MustParse("256Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("1000m"),
			corev1.ResourceMemory: resource.MustParse("1Gi"),
		},
	}
	password := &corev1.EnvVar{
		Name:  "REDIS_PASSWORD",
		Value: "testpassword",
	}

	container := redisServerContainer(cluster, password)

	assert.Equal(t, "redis", container.Name)
	assert.NotNil(t, container.Resources)
	assert.Equal(t, resource.MustParse("200m"), container.Resources.Requests[corev1.ResourceCPU])
	assert.Equal(t, resource.MustParse("256Mi"), container.Resources.Requests[corev1.ResourceMemory])
	assert.Equal(t, resource.MustParse("1000m"), container.Resources.Limits[corev1.ResourceCPU])
	assert.Equal(t, resource.MustParse("1Gi"), container.Resources.Limits[corev1.ResourceMemory])
}

func TestRedisMonitorContainersWithCustomImage(t *testing.T) {
	cluster := createTestDistributedRedisCluster("test-cluster", "test-namespace")
	cluster.Spec.Monitor = &[]redisv1alpha1.AgentSpec{
		{
			Name:  "exporter",
			Image: "redis-exporter:latest",
			Prometheus: &redisv1alpha1.PrometheusSpec{
				Name: "metrics",
				Port: 9121,
			},
		},
	}
	password := &corev1.EnvVar{
		Name:  "REDIS_PASSWORD",
		Value: "testpassword",
	}

	containers := redisMonitorContainers(cluster, password)

	assert.NotEmpty(t, containers)

	// Check if exporter container is present
	var exporterContainer *corev1.Container
	for i := range containers {
		if containers[i].Name == "exporter" {
			exporterContainer = &containers[i]
			break
		}
	}

	assert.NotNil(t, exporterContainer)
	assert.Equal(t, "exporter", exporterContainer.Name)
}

func TestRedisRestoreInitContainerError(t *testing.T) {
	cluster := createTestDistributedRedisCluster("test-cluster", "test-namespace")
	// No backup storage configured
	password := &corev1.EnvVar{
		Name:  "REDIS_PASSWORD",
		Value: "testpassword",
	}

	container, err := redisRestoreInitContainer(cluster, password)

	assert.Error(t, err)
	assert.Empty(t, container.Name)
}

func TestGetContainersWithRedisEnvEmptyInput(t *testing.T) {
	cluster := createTestDistributedRedisCluster("test-cluster", "test-namespace")
	password := &corev1.EnvVar{
		Name:  "REDIS_PASSWORD",
		Value: "testpassword",
	}

	// Empty containers and env vars
	containers := getContainersWithRedisEnv(cluster, []corev1.Container{}, []corev1.EnvVar{}, password)

	assert.Empty(t, containers)
}

func TestRedisVolumesWithEphemeralStorage(t *testing.T) {
	cluster := createTestDistributedRedisCluster("test-cluster", "test-namespace")
	// Use ephemeral storage
	cluster.Spec.ConfStorage = &redisv1alpha1.RedisStorage{
		Size:  resource.MustParse("1Gi"),
		Type:  redisv1alpha1.Ephemeral,
		Class: "standard",
	}
	cluster.Spec.BackupStorage = &redisv1alpha1.RedisStorage{
		Size:  resource.MustParse("1Gi"),
		Type:  redisv1alpha1.Ephemeral,
		Class: "standard",
	}

	volumes := redisVolumes(cluster)

	assert.NotEmpty(t, volumes)

	// Check for ephemeral storage volumes
	volumeNames := make(map[string]bool)
	for _, volume := range volumes {
		volumeNames[volume.Name] = true
	}

	assert.True(t, volumeNames["redis-data"], "redis-data volume should be present")
	assert.True(t, volumeNames["redis-modules"], "redis-modules volume should be present")
}
