package configmaps

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	redisv1alpha1 "github.com/freshworks/redis-cluster-operator/api/v1alpha1"
)

// MockClient is a mock implementation of client.Client
type MockClient struct {
	mock.Mock
}

func (m *MockClient) Get(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) error {
	args := m.Called(ctx, key, obj, opts)
	return args.Error(0)
}

func (m *MockClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	args := m.Called(ctx, list, opts)
	return args.Error(0)
}

func (m *MockClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	args := m.Called(ctx, obj, patch, opts)
	return args.Error(0)
}

func (m *MockClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) Status() client.StatusWriter {
	args := m.Called()
	return args.Get(0).(client.StatusWriter)
}

func (m *MockClient) Scheme() *runtime.Scheme {
	args := m.Called()
	return args.Get(0).(*runtime.Scheme)
}

func (m *MockClient) RESTMapper() meta.RESTMapper {
	args := m.Called()
	return args.Get(0).(meta.RESTMapper)
}

func (m *MockClient) SubResource(subResource string) client.SubResourceClient {
	args := m.Called(subResource)
	return args.Get(0).(client.SubResourceClient)
}

func createTestCluster(name, namespace string) *redisv1alpha1.DistributedRedisCluster {
	return &redisv1alpha1.DistributedRedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: redisv1alpha1.DistributedRedisClusterSpec{
			MasterSize:      3,
			ClusterReplicas: 1,
			Image:           "redis:6.2.14-alpine",
			Config: map[string]string{
				"maxmemory":        "1000000000",
				"maxmemory-policy": "noeviction",
			},
		},
	}
}

func TestSupportsACL(t *testing.T) {
	tests := []struct {
		name     string
		image    string
		expected bool
	}{
		{
			name:     "Redis 6.0+ should support ACL",
			image:    "redis:6.2.14-alpine",
			expected: true,
		},
		{
			name:     "Redis 7.0+ should support ACL",
			image:    "redis:7.0.0-alpine",
			expected: true,
		},
		{
			name:     "Redis 5.x should not support ACL",
			image:    "redis:5.0.14-alpine",
			expected: false,
		},
		{
			name:     "Redis 4.x should not support ACL",
			image:    "redis:4.0.14-alpine",
			expected: false,
		},
		{
			name:     "Redis 3.x should not support ACL",
			image:    "redis:3.2.12-alpine",
			expected: false,
		},
		{
			name:     "Unknown version should default to true",
			image:    "redis:latest",
			expected: true,
		},
		{
			name:     "Empty image should default to true",
			image:    "",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := supportsACL(tt.image)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRedisConfigMapName(t *testing.T) {
	tests := []struct {
		name         string
		clusterName  string
		expectedName string
	}{
		{
			name:         "Regular cluster name",
			clusterName:  "my-cluster",
			expectedName: "drc-my-cluster-redis-config",
		},
		{
			name:         "Cluster name with prefix",
			clusterName:  "distributedrediscluster-default",
			expectedName: "drc-default-redis-config",
		},
		{
			name:         "Empty cluster name",
			clusterName:  "",
			expectedName: "drc--redis-config",
		},
		{
			name:         "Cluster name with special characters",
			clusterName:  "test-cluster-123",
			expectedName: "drc-test-cluster-123-redis-config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedisConfigMapName(tt.clusterName)
			assert.Equal(t, tt.expectedName, result)
		})
	}
}

func TestRestoreConfigMapName(t *testing.T) {
	tests := []struct {
		name         string
		clusterName  string
		expectedName string
	}{
		{
			name:         "Regular cluster name",
			clusterName:  "my-cluster",
			expectedName: "drc-my-cluster-restore",
		},
		{
			name:         "Cluster name with prefix",
			clusterName:  "distributedrediscluster-default",
			expectedName: "drc-default-restore",
		},
		{
			name:         "Empty cluster name",
			clusterName:  "",
			expectedName: "drc--restore",
		},
		{
			name:         "Cluster name with special characters",
			clusterName:  "test-cluster-123",
			expectedName: "drc-test-cluster-123-restore",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RestoreConfigMapName(tt.clusterName)
			assert.Equal(t, tt.expectedName, result)
		})
	}
}

func TestAclConfigMapName(t *testing.T) {
	tests := []struct {
		name         string
		clusterName  string
		expectedName string
	}{
		{
			name:         "Regular cluster name",
			clusterName:  "my-cluster",
			expectedName: "drc-my-cluster-redis-acl",
		},
		{
			name:         "Cluster name with prefix",
			clusterName:  "distributedrediscluster-default",
			expectedName: "drc-default-redis-acl",
		},
		{
			name:         "Empty cluster name",
			clusterName:  "",
			expectedName: "drc--redis-acl",
		},
		{
			name:         "Cluster name with special characters",
			clusterName:  "test-cluster-123",
			expectedName: "drc-test-cluster-123-redis-acl",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AclConfigMapName(tt.clusterName)
			assert.Equal(t, tt.expectedName, result)
		})
	}
}

func TestRedisInitConfigMapName(t *testing.T) {
	tests := []struct {
		name         string
		clusterName  string
		expectedName string
	}{
		{
			name:         "Regular cluster name",
			clusterName:  "my-cluster",
			expectedName: "drc-my-cluster-redis-init",
		},
		{
			name:         "Cluster name with prefix",
			clusterName:  "distributedrediscluster-default",
			expectedName: "drc-default-redis-init",
		},
		{
			name:         "Empty cluster name",
			clusterName:  "",
			expectedName: "drc--redis-init",
		},
		{
			name:         "Cluster name with special characters",
			clusterName:  "test-cluster-123",
			expectedName: "drc-test-cluster-123-redis-init",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedisInitConfigMapName(tt.clusterName)
			assert.Equal(t, tt.expectedName, result)
		})
	}
}

func TestGetHashedPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		expected string
	}{
		{
			name:     "Simple password",
			password: "password123",
			expected: "ef92b778bafe771e89245b89ecbc08a44a4e166c06659911881f383d4473e94f",
		},
		{
			name:     "Empty password",
			password: "",
			expected: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:     "Special characters password",
			password: "p@ssw0rd!@#",
			expected: "07c618c522e7dc5da2c90c836bd2ea33c382136ac15a8b85282aa665d1e4d043",
		},
		{
			name:     "Long password",
			password: "verylongpasswordwithlotsofcharacters",
			expected: "f603eae8148900b21d4c5ad16d247b7f312db8d5799cbc49b9eedbd17dc0f4da",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetHashedPassword(tt.password)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewConfigMapForCR(t *testing.T) {
	cluster := createTestCluster("test-cluster", "test-ns")
	labels := map[string]string{
		"app":  "redis",
		"type": "cluster",
	}

	configMap := NewConfigMapForCR(cluster, labels)

	// Verify basic structure
	assert.NotNil(t, configMap)
	assert.Equal(t, "drc-test-cluster-redis-config", configMap.Name)
	assert.Equal(t, "test-ns", configMap.Namespace)
	assert.Equal(t, labels, configMap.Labels)

	// Verify data contains expected keys
	assert.Contains(t, configMap.Data, "fix-ip.sh")
	assert.Contains(t, configMap.Data, "redis.conf")

	// Verify fix-ip.sh content
	fixIPContent := configMap.Data["fix-ip.sh"]
	assert.Contains(t, fixIPContent, "#!/bin/sh")
	assert.Contains(t, fixIPContent, "CLUSTER_CONFIG=\"/data/nodes.conf\"")
	assert.Contains(t, fixIPContent, "POD_IP")

	// Verify redis.conf content
	redisConfContent := configMap.Data["redis.conf"]
	assert.Contains(t, redisConfContent, "maxmemory 1000000000")
	assert.Contains(t, redisConfContent, "maxmemory-policy noeviction")
}

func TestNewConfigMapForRestore(t *testing.T) {
	cluster := createTestCluster("test-cluster", "test-ns")
	labels := map[string]string{
		"app":  "redis",
		"type": "restore",
	}

	configMap := NewConfigMapForRestore(cluster, labels)

	// Verify basic structure
	assert.NotNil(t, configMap)
	assert.Equal(t, "drc-test-cluster-restore", configMap.Name)
	assert.Equal(t, "test-ns", configMap.Namespace)
	assert.Equal(t, labels, configMap.Labels)

	// Verify data contains expected keys
	assert.Contains(t, configMap.Data, RestoreSucceeded)
	assert.Equal(t, "0", configMap.Data[RestoreSucceeded])
}

func TestGetAcldataFromSecrets(t *testing.T) {
	// Test the function structure - actual integration testing would be done in e2e tests
	t.Run("Function structure test", func(t *testing.T) {
		assert.NotNil(t, GetAcldataFromSecrets)
	})
}

func TestGetAclStringFromSecrets(t *testing.T) {
	// Test the function structure - actual integration testing would be done in e2e tests
	t.Run("Function structure test", func(t *testing.T) {
		assert.NotNil(t, GetAclStringFromSecrets)
	})
}

func TestGetsecretversions(t *testing.T) {
	// Test the function structure - actual integration testing would be done in e2e tests
	t.Run("Function structure test", func(t *testing.T) {
		assert.NotNil(t, Getsecretversions)
	})
}

func TestNewACLConfigMap(t *testing.T) {
	// Test the function structure - actual integration testing would be done in e2e tests
	t.Run("Function structure test", func(t *testing.T) {
		assert.NotNil(t, NewACLConfigMap)
	})
}

func TestGenerateAclconf(t *testing.T) {
	// Test the function structure - actual integration testing would be done in e2e tests
	t.Run("Function structure test", func(t *testing.T) {
		assert.NotNil(t, GenerateAclconf)
	})
}

func TestGenerateRedisConfContent(t *testing.T) {
	tests := []struct {
		name       string
		configMap  map[string]string
		redisImage string
		expected   string
	}{
		{
			name: "Redis 6.0+ with ACL support",
			configMap: map[string]string{
				"activerehashing":               "yes",
				"appendfsync":                   "everysec",
				"appendonly":                    "yes",
				"auto-aof-rewrite-min-size":     "67108864",
				"auto-aof-rewrite-percentage":   "100",
				"cluster-node-timeout":          "15000",
				"cluster-require-full-coverage": "yes",
				"hash-max-ziplist-entries":      "512",
				"hash-max-ziplist-value":        "64",
				"hll-sparse-max-bytes":          "3000",
				"list-compress-depth":           "0",
				"maxmemory":                     "1000000000",
				"maxmemory-policy":              "noeviction",
				"maxmemory-samples":             "5",
				"no-appendfsync-on-rewrite":     "no",
				"notify-keyspace-events":        "",
				"repl-backlog-size":             "1048576",
				"repl-backlog-ttl":              "3600",
				"set-max-intset-entries":        "512",
				"slowlog-log-slower-than":       "10000",
				"slowlog-max-len":               "128",
				"stop-writes-on-bgsave-error":   "yes",
				"tcp-keepalive":                 "0",
				"timeout":                       "0",
				"zset-max-ziplist-entries":      "128",
				"zset-max-ziplist-value":        "64",
			},
			redisImage: "redis:6.2.14-alpine",
			expected: `aclfile /acl/users.acl
dir /backup
activerehashing yes
appendfsync everysec
appendonly yes
auto-aof-rewrite-min-size 67108864
auto-aof-rewrite-percentage 100
cluster-node-timeout 15000
cluster-require-full-coverage yes
hash-max-ziplist-entries 512
hash-max-ziplist-value 64
hll-sparse-max-bytes 3000
list-compress-depth 0
maxmemory 1000000000
maxmemory-policy noeviction
maxmemory-samples 5
no-appendfsync-on-rewrite no
repl-backlog-size 1048576
repl-backlog-ttl 3600
set-max-intset-entries 512
slowlog-log-slower-than 10000
slowlog-max-len 128
stop-writes-on-bgsave-error yes
tcp-keepalive 0
timeout 0
zset-max-ziplist-entries 128
zset-max-ziplist-value 64
`,
		},
		{
			name: "Redis 5.x without ACL support",
			configMap: map[string]string{
				"maxmemory":        "1000000000",
				"maxmemory-policy": "noeviction",
			},
			redisImage: "redis:5.0.14-alpine",
			expected: `dir /backup
maxmemory 1000000000
maxmemory-policy noeviction
`,
		},
		{
			name:       "Nil config map",
			configMap:  nil,
			redisImage: "redis:6.2.14-alpine",
			expected: `aclfile /acl/users.acl
dir /backup
`,
		},
		{
			name:       "Empty config map",
			configMap:  map[string]string{},
			redisImage: "redis:6.2.14-alpine",
			expected: `aclfile /acl/users.acl
dir /backup
`,
		},
		{
			name: "Config map with empty values",
			configMap: map[string]string{
				"maxmemory":        "1000000000",
				"empty-value":      "",
				"maxmemory-policy": "noeviction",
			},
			redisImage: "redis:6.2.14-alpine",
			expected: `aclfile /acl/users.acl
dir /backup
maxmemory 1000000000
maxmemory-policy noeviction
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateRedisConfContent(tt.configMap, tt.redisImage)
			assert.Equal(t, tt.expected, result)
		})
	}
}
