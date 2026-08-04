package distributedrediscluster

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	redisv1alpha1 "github.com/freshworks/redis-cluster-operator/api/v1alpha1"
)

func TestCheckBootstrapMigrationStatus(t *testing.T) {
	reconcileFalse := false
	cluster := &redisv1alpha1.DistributedRedisCluster{
		Spec: redisv1alpha1.DistributedRedisClusterSpec{
			MasterSize: 3,
			BootstrapNode: &redisv1alpha1.BootstrapNodeSpec{
				Host:      "10.0.0.1",
				Reconcile: &reconcileFalse,
			},
		},
	}
	allNoFailover := replicaNoFailoverStatus{Ready: 12, Total: 12}

	tests := []struct {
		name           string
		numberOfMaster int32
		noFailover     replicaNoFailoverStatus
		wantPhase      redisv1alpha1.BootstrapPhase
		wantProgress   string
		wantHealthy    bool
		wantClusterOK  bool
	}{
		{
			name:           "no local masters yet",
			numberOfMaster: 0,
			noFailover:     replicaNoFailoverStatus{},
			wantPhase:      redisv1alpha1.BootstrapPhaseMigrating,
			wantProgress:   "masters: 0/3",
			wantHealthy:    false,
		},
		{
			name:           "partial cutover",
			numberOfMaster: 2,
			noFailover:     allNoFailover,
			wantPhase:      redisv1alpha1.BootstrapPhaseMigrating,
			wantProgress:   "masters: 2/3; nofailover: 12/12",
			wantHealthy:    false,
		},
		{
			name:           "cutover complete",
			numberOfMaster: 3,
			noFailover:     allNoFailover,
			wantPhase:      redisv1alpha1.BootstrapPhaseMigrated,
			wantProgress:   "masters: 3/3; nofailover: 12/12",
			wantHealthy:    true,
			wantClusterOK:  true,
		},
		{
			name:           "too many local masters",
			numberOfMaster: 4,
			noFailover:     allNoFailover,
			wantPhase:      redisv1alpha1.BootstrapPhaseFailed,
			wantProgress:   "masters: 4/3; nofailover: 12/12",
			wantHealthy:    false,
		},
		{
			name:           "nofailover pending does not block migrated phase",
			numberOfMaster: 3,
			noFailover: replicaNoFailoverStatus{
				Ready:   10,
				Total:   12,
				Pending: []string{"shard0-1", "shard1-0"},
			},
			wantPhase:     redisv1alpha1.BootstrapPhaseMigrated,
			wantProgress:  "masters: 3/3; nofailover: 10/12",
			wantHealthy:   true,
			wantClusterOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := &redisv1alpha1.DistributedRedisClusterStatus{}
			status.NumberOfMaster = tt.numberOfMaster
			got := checkBootstrapMigrationStatus(status, cluster, tt.noFailover)
			if got != tt.wantHealthy {
				t.Fatalf("healthy: got %v want %v", got, tt.wantHealthy)
			}
			if status.Bootstrap.Phase != tt.wantPhase {
				t.Fatalf("phase: got %q want %q", status.Bootstrap.Phase, tt.wantPhase)
			}
			if status.Bootstrap.Progress != tt.wantProgress {
				t.Fatalf("progress: got %q want %q", status.Bootstrap.Progress, tt.wantProgress)
			}
			if tt.noFailover.Total > 0 && len(tt.noFailover.Pending) > 0 {
				for _, pod := range tt.noFailover.Pending {
					if !strings.Contains(status.Bootstrap.Reason, pod) {
						t.Fatalf("reason should list pending pod %q: %q", pod, status.Bootstrap.Reason)
					}
				}
			}
			if tt.wantClusterOK && status.Status != redisv1alpha1.ClusterStatusOK {
				t.Fatalf("expected cluster OK, got %q", status.Status)
			}
			if !tt.wantHealthy && tt.wantPhase != redisv1alpha1.BootstrapPhaseFailed && status.Status != redisv1alpha1.ClusterStatusNOK {
				t.Fatalf("expected cluster NOK, got %q", status.Status)
			}
		})
	}
}

func TestPodsForReplicaNoFailoverCheck(t *testing.T) {
	makePod := func(name, sts string) *corev1.Pod {
		return &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
				OwnerReferences: []metav1.OwnerReference{{
					Kind: "StatefulSet",
					Name: sts,
				}},
			},
			Status: corev1.PodStatus{PodIP: "10.0.0.1"},
			Spec: corev1.PodSpec{Containers: []corev1.Container{{
				Name:  "redis",
				Ports: []corev1.ContainerPort{{Name: "client", ContainerPort: 6379}},
			}}},
		}
	}
	ordered, err := orderedBootstrapPods([]*corev1.Pod{
		makePod("drc-test-shard0-0", "drc-test-shard0"),
		makePod("drc-test-shard0-1", "drc-test-shard0"),
		makePod("drc-test-shard1-0", "drc-test-shard1"),
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("no local masters skips all pods", func(t *testing.T) {
		status := &redisv1alpha1.DistributedRedisClusterStatus{NumberOfMaster: 0}
		if got := podsForReplicaNoFailoverCheck(status, ordered); len(got) != 0 {
			t.Fatalf("expected no pods, got %d", len(got))
		}
	})

	t.Run("only statefulsets with a local master", func(t *testing.T) {
		status := &redisv1alpha1.DistributedRedisClusterStatus{
			NumberOfMaster: 1,
			Nodes: []redisv1alpha1.RedisClusterNode{{
				PodName:     "drc-test-shard0-0",
				Role:        redisv1alpha1.RedisClusterNodeRoleMaster,
				Slots:       []string{"0-5460"},
				StatefulSet: "drc-test-shard0",
			}},
		}
		got := podsForReplicaNoFailoverCheck(status, ordered)
		if len(got) != 2 {
			t.Fatalf("expected 2 pods on shard0 sts, got %d", len(got))
		}
		for _, ref := range got {
			if podStatefulSetName(ref.pod) != "drc-test-shard0" {
				t.Fatalf("unexpected sts pod %s", ref.pod.Name)
			}
		}
	})
}

func TestBootstrapMigrationProgress(t *testing.T) {
	got := bootstrapMigrationProgress(2, 3, replicaNoFailoverStatus{Ready: 5, Total: 12})
	if got != "masters: 2/3; nofailover: 5/12" {
		t.Fatalf("unexpected progress: %q", got)
	}
}

func TestBootstrapMigrationReason(t *testing.T) {
	reason := bootstrapMigrationReason(1, 3, replicaNoFailoverStatus{
		Ready:   1,
		Total:   2,
		Pending: []string{"shard0-1"},
	}, "cutover in progress: %d of %d expected shard masters are local", 1, 3)
	if reason != "cutover in progress: 1 of 3 expected shard masters are local; cluster-replica-no-failover=no on 1/2 pods on statefulsets with a local master (pending: shard0-1)" {
		t.Fatalf("unexpected reason: %q", reason)
	}
}
