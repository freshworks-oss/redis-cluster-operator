package v1alpha1

import (
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestValidateEngineAndImage(t *testing.T) {
	tests := []struct {
		name    string
		cluster *DistributedRedisCluster
		wantErr string
	}{
		{
			name: "ok redis engine redis image",
			cluster: &DistributedRedisCluster{
				ObjectMeta: metav1.ObjectMeta{Name: "x"},
				Spec: DistributedRedisClusterSpec{
					Engine: DatabaseEngineRedis,
					Image:  "redis:7.2.4-alpine",
				},
			},
		},
		{
			name: "ok empty engine redis image",
			cluster: &DistributedRedisCluster{
				ObjectMeta: metav1.ObjectMeta{Name: "x"},
				Spec: DistributedRedisClusterSpec{
					Image: "redis:7.2.4-alpine",
				},
			},
		},
		{
			name: "ok valkey engine valkey image",
			cluster: &DistributedRedisCluster{
				ObjectMeta: metav1.ObjectMeta{Name: "x"},
				Spec: DistributedRedisClusterSpec{
					Engine: DatabaseEngineValkey,
					Image:  "valkey/valkey:7.2.11-alpine",
				},
			},
		},
		{
			name: "ok valkey engine image with both substrings",
			cluster: &DistributedRedisCluster{
				ObjectMeta: metav1.ObjectMeta{Name: "x"},
				Spec: DistributedRedisClusterSpec{
					Engine: DatabaseEngineValkey,
					Image:  "registry.example/valkey-redis-compat:1",
				},
			},
		},
		{
			name: "ok redis engine image with both substrings",
			cluster: &DistributedRedisCluster{
				ObjectMeta: metav1.ObjectMeta{Name: "x"},
				Spec: DistributedRedisClusterSpec{
					Engine: DatabaseEngineRedis,
					Image:  "myredis/valkey-compatible-redis:1",
				},
			},
		},
		{
			name: "empty image skipped",
			cluster: &DistributedRedisCluster{
				ObjectMeta: metav1.ObjectMeta{Name: "x"},
				Spec: DistributedRedisClusterSpec{
					Engine: DatabaseEngineValkey,
					Image:  "",
				},
			},
		},
		{
			name: "bad engine",
			cluster: &DistributedRedisCluster{
				ObjectMeta: metav1.ObjectMeta{Name: "x"},
				Spec: DistributedRedisClusterSpec{
					Engine: "memcached",
					Image:  "redis:7",
				},
			},
			wantErr: "spec.engine must be empty",
		},
		{
			name: "redis engine image without redis substring",
			cluster: &DistributedRedisCluster{
				ObjectMeta: metav1.ObjectMeta{Name: "x"},
				Spec: DistributedRedisClusterSpec{
					Image: "valkey/valkey:7",
				},
			},
			wantErr: "must contain",
		},
		{
			name: "redis engine missing redis in image",
			cluster: &DistributedRedisCluster{
				ObjectMeta: metav1.ObjectMeta{Name: "x"},
				Spec: DistributedRedisClusterSpec{
					Image: "bitnami/valkey:7",
				},
			},
			wantErr: "must contain",
		},
		{
			name: "valkey engine image without valkey substring",
			cluster: &DistributedRedisCluster{
				ObjectMeta: metav1.ObjectMeta{Name: "x"},
				Spec: DistributedRedisClusterSpec{
					Engine: DatabaseEngineValkey,
					Image:  "redis:7",
				},
			},
			wantErr: "must contain",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cluster.ValidateEngineAndImage()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected err: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err %q should contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}
