package v1alpha1

import (
	"strings"
	"testing"
)

func TestValidateBootstrapNode(t *testing.T) {
	tests := []struct {
		name    string
		cluster *DistributedRedisCluster
		wantErr string
	}{
		{
			name: "unset ok",
			cluster: &DistributedRedisCluster{
				Spec: DistributedRedisClusterSpec{},
			},
		},
		{
			name: "valid ip and port",
			cluster: &DistributedRedisCluster{
				Spec: DistributedRedisClusterSpec{
					BootstrapNode: &BootstrapNodeSpec{
						Host: "10.0.0.5",
						Port: "6379",
					},
				},
			},
		},
		{
			name: "valid hostname",
			cluster: &DistributedRedisCluster{
				Spec: DistributedRedisClusterSpec{
					BootstrapNode: &BootstrapNodeSpec{
						Host: "redis-seed.example.com",
					},
				},
			},
		},
		{
			name: "missing host",
			cluster: &DistributedRedisCluster{
				Spec: DistributedRedisClusterSpec{
					BootstrapNode: &BootstrapNodeSpec{},
				},
			},
			wantErr: "host is required",
		},
		{
			name: "invalid port",
			cluster: &DistributedRedisCluster{
				Spec: DistributedRedisClusterSpec{
					BootstrapNode: &BootstrapNodeSpec{
						Host: "10.0.0.1",
						Port: "bad",
					},
				},
			},
			wantErr: "port",
		},
		{
			name: "with restore",
			cluster: &DistributedRedisCluster{
				Spec: DistributedRedisClusterSpec{
					BootstrapNode: &BootstrapNodeSpec{Host: "10.0.0.1"},
					Restore:       &RestoreSpec{BackupSource: &BackupSourceSpec{Namespace: "ns", Name: "bk"}},
				},
			},
			wantErr: "restore",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cluster.ValidateBootstrapNode()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestBootstrapReconcileEnabled(t *testing.T) {
	cluster := &DistributedRedisCluster{}
	if !cluster.BootstrapReconcileEnabled() {
		t.Fatal("expected default reconcile enabled when bootstrapNode is nil")
	}

	reconcileFalse := false
	cluster = &DistributedRedisCluster{
		Spec: DistributedRedisClusterSpec{
			BootstrapNode: &BootstrapNodeSpec{
				Host:      "10.0.0.1",
				Reconcile: &reconcileFalse,
			},
		},
	}
	if cluster.BootstrapReconcileEnabled() {
		t.Fatal("expected reconcile disabled")
	}
	cluster.Spec.BootstrapNode.Reconcile = nil
	if !cluster.BootstrapReconcileEnabled() {
		t.Fatal("expected default reconcile enabled")
	}
}
