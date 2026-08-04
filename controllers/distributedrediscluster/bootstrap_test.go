package distributedrediscluster

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	redisv1alpha1 "github.com/freshworks/redis-cluster-operator/api/v1alpha1"
	"github.com/freshworks/redis-cluster-operator/utils/redisutil"
)

func TestApplyBootstrapRedisConfig(t *testing.T) {
	t.Run("reconcile enabled defaults no-failover yes, allow-migration no", func(t *testing.T) {
		cluster := &redisv1alpha1.DistributedRedisCluster{
			Spec: redisv1alpha1.DistributedRedisClusterSpec{
				BootstrapNode: &redisv1alpha1.BootstrapNodeSpec{Host: "10.0.0.1"},
			},
		}
		applyBootstrapRedisConfig(cluster)
		assertBootstrapRedisConfig(t, cluster, bootstrapReplicaNoFailoverValue, bootstrapAllowReplicaMigrationValue)
	})

	t.Run("reconcile explicitly true sets no-failover yes, allow-migration no", func(t *testing.T) {
		reconcileTrue := true
		cluster := &redisv1alpha1.DistributedRedisCluster{
			Spec: redisv1alpha1.DistributedRedisClusterSpec{
				BootstrapNode: &redisv1alpha1.BootstrapNodeSpec{
					Host:      "10.0.0.1",
					Reconcile: &reconcileTrue,
				},
			},
		}
		applyBootstrapRedisConfig(cluster)
		assertBootstrapRedisConfig(t, cluster, bootstrapReplicaNoFailoverValue, bootstrapAllowReplicaMigrationValue)
	})

	t.Run("reconcile false sets no-failover no, allow-migration yes", func(t *testing.T) {
		reconcileFalse := false
		cluster := &redisv1alpha1.DistributedRedisCluster{
			Spec: redisv1alpha1.DistributedRedisClusterSpec{
				BootstrapNode: &redisv1alpha1.BootstrapNodeSpec{
					Host:      "10.0.0.1",
					Reconcile: &reconcileFalse,
				},
			},
		}
		applyBootstrapRedisConfig(cluster)
		assertBootstrapRedisConfig(t, cluster, bootstrapReplicaNoFailoverDisabled, bootstrapAllowReplicaMigrationEnabled)
	})
}

func assertBootstrapRedisConfig(t *testing.T, cluster *redisv1alpha1.DistributedRedisCluster, wantNoFailover, wantAllowMigration string) {
	t.Helper()
	if cluster.Spec.Config[bootstrapReplicaNoFailover] != wantNoFailover {
		t.Fatalf("expected cluster-replica-no-failover=%s, got %q", wantNoFailover, cluster.Spec.Config[bootstrapReplicaNoFailover])
	}
	if cluster.Spec.Config[bootstrapAllowReplicaMigration] != wantAllowMigration {
		t.Fatalf("expected cluster-allow-replica-migration=%s, got %q", wantAllowMigration, cluster.Spec.Config[bootstrapAllowReplicaMigration])
	}
}

func TestSortRemoteMasters(t *testing.T) {
	masters := redisutil.Nodes{
		{ID: "b", Role: redisutil.RedisMasterRole, Slots: []redisutil.Slot{5000, 5001}},
		{ID: "a", Role: redisutil.RedisMasterRole, Slots: []redisutil.Slot{0, 1, 2}},
		{ID: "c", Role: redisutil.RedisSlaveRole, Slots: nil},
	}
	sorted, err := sortRemoteMasters(masters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sorted) != 2 {
		t.Fatalf("expected 2 masters, got %d", len(sorted))
	}
	if sorted[0].ID != "a" || sorted[1].ID != "b" {
		t.Fatalf("unexpected master order: %s, %s", sorted[0].ID, sorted[1].ID)
	}
}

func TestOrderedBootstrapPods(t *testing.T) {
	pods := []*corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "drc-test-shard1-0"},
			Status:     corev1.PodStatus{PodIP: "10.0.0.2"},
			Spec: corev1.PodSpec{Containers: []corev1.Container{{
				Name:  "redis",
				Ports: []corev1.ContainerPort{{Name: "client", ContainerPort: 6379}},
			}}},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "drc-test-shard0-0"},
			Status:     corev1.PodStatus{PodIP: "10.0.0.1"},
			Spec: corev1.PodSpec{Containers: []corev1.Container{{
				Name:  "redis",
				Ports: []corev1.ContainerPort{{Name: "client", ContainerPort: 6379}},
			}}},
		},
	}
	ordered, err := orderedBootstrapPods(pods)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ordered) != 2 {
		t.Fatalf("expected 2 pods, got %d", len(ordered))
	}
	if ordered[0].shardIndex != 0 || ordered[1].shardIndex != 1 {
		t.Fatalf("unexpected shard order: %d, %d", ordered[0].shardIndex, ordered[1].shardIndex)
	}
}

func TestBootstrapModeHelpers(t *testing.T) {
	reconcileFalse := false
	cluster := &redisv1alpha1.DistributedRedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{redisv1alpha1.BootstrapAnnotation: "true"},
		},
		Spec: redisv1alpha1.DistributedRedisClusterSpec{
			BootstrapNode: &redisv1alpha1.BootstrapNodeSpec{
				Host:      "1.2.3.4",
				Port:      "6380",
				Reconcile: &reconcileFalse,
			},
		},
	}
	if !cluster.BootstrapMode() {
		t.Fatal("expected bootstrap mode")
	}
	if cluster.BootstrapReconcileEnabled() {
		t.Fatal("expected reconcile disabled")
	}
	if !cluster.HasBootstrapAnnotation() {
		t.Fatal("expected bootstrap annotation")
	}
	if cluster.BootstrapSeedPort() != "6380" {
		t.Fatalf("expected port 6380, got %s", cluster.BootstrapSeedPort())
	}
}
