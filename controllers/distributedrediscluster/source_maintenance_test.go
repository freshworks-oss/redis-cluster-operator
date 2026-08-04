package distributedrediscluster

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	redisv1alpha1 "github.com/freshworks/redis-cluster-operator/api/v1alpha1"
	"github.com/freshworks/redis-cluster-operator/utils/redisutil"
	"github.com/freshworks/redis-cluster-operator/utils/testutil/mocks"
)

// sourcePod builds a pod owned by the given statefulset with the given IP.
func sourcePod(name, sts, ip string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			OwnerReferences: []metav1.OwnerReference{{Kind: "StatefulSet", Name: sts}},
		},
		Status: corev1.PodStatus{PodIP: ip},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{
			Name:  "redis",
			Ports: []corev1.ContainerPort{{Name: "client", ContainerPort: 6379}},
		}}},
	}
}

func clusterInfosFrom(nodes ...*redisutil.Node) *redisutil.ClusterInfos {
	ci := redisutil.NewClusterInfos()
	for _, n := range nodes {
		ci.Infos[n.IPPort()] = &redisutil.NodeInfos{Node: n}
	}
	return ci
}

func masterNode(id, ip string, slots ...redisutil.Slot) *redisutil.Node {
	return &redisutil.Node{ID: id, IP: ip, Port: "6379", Role: redisutil.RedisMasterRole, Slots: slots}
}

func slaveNode(id, ip, masterID string) *redisutil.Node {
	return &redisutil.Node{ID: id, IP: ip, Port: "6379", Role: redisutil.RedisSlaveRole, MasterReferent: masterID}
}

func emptyNode(id, ip string) *redisutil.Node {
	return &redisutil.Node{ID: id, IP: ip, Port: "6379", Role: redisutil.RedisMasterRole}
}

func TestShardCurrentMaster(t *testing.T) {
	orphan := sourcePod("drc-shard0-1", "drc-shard0", "10.0.0.12")

	t.Run("sibling is master with slots (pre-cutover)", func(t *testing.T) {
		pods := []*corev1.Pod{sourcePod("drc-shard0-0", "drc-shard0", "10.0.0.11"), orphan}
		infos := clusterInfosFrom(masterNode("m0", "10.0.0.11", 0, 100), emptyNode("x", "10.0.0.12"))
		master, err := shardCurrentMaster(infos, orphan, pods)
		if err != nil {
			t.Fatal(err)
		}
		if master.ID != "m0" {
			t.Fatalf("want master m0, got %s", master.ID)
		}
	})

	t.Run("sibling replica points at destination master (post-cutover)", func(t *testing.T) {
		pods := []*corev1.Pod{sourcePod("drc-shard0-0", "drc-shard0", "10.0.0.11"), orphan}
		infos := clusterInfosFrom(
			slaveNode("s0", "10.0.0.11", "dst0"),   // sibling now replicates the destination master
			masterNode("dst0", "10.9.9.9", 0, 100), // destination master owns the slots
			emptyNode("x", "10.0.0.12"),
		)
		master, err := shardCurrentMaster(infos, orphan, pods)
		if err != nil {
			t.Fatal(err)
		}
		if master.ID != "dst0" {
			t.Fatalf("want master dst0, got %s", master.ID)
		}
	})

	t.Run("no resolvable sibling returns error", func(t *testing.T) {
		pods := []*corev1.Pod{orphan}
		infos := clusterInfosFrom(emptyNode("x", "10.0.0.12"))
		if _, err := shardCurrentMaster(infos, orphan, pods); err == nil {
			t.Fatal("expected error when no sibling is a cluster member")
		}
	})

	t.Run("pod without statefulset owner returns error", func(t *testing.T) {
		noOwner := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "loose"}, Status: corev1.PodStatus{PodIP: "10.0.0.99"}}
		if _, err := shardCurrentMaster(clusterInfosFrom(), noOwner, []*corev1.Pod{noOwner}); err == nil {
			t.Fatal("expected error for pod with no statefulset owner")
		}
	})
}

func TestFindSourceNodeByIP(t *testing.T) {
	infos := clusterInfosFrom(masterNode("m0", "10.0.0.11", 0), emptyNode("x", "10.0.0.12"))
	if n := findSourceNodeByIP(infos, "10.0.0.11"); n == nil || n.ID != "m0" {
		t.Fatalf("expected to find m0, got %v", n)
	}
	if n := findSourceNodeByIP(infos, "10.0.0.99"); n != nil {
		t.Fatalf("expected nil for unknown ip, got %v", n)
	}
}

func TestMaintainSourceReplicas(t *testing.T) {
	ns, name := "default", "drc"
	ctxbg := context.Background()
	instance := &redisv1alpha1.DistributedRedisCluster{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name},
		Spec:       redisv1alpha1.DistributedRedisClusterSpec{MasterSize: 1},
	}

	t.Run("healthy pods are left untouched", func(t *testing.T) {
		admin := &mocks.MockRedisAdmin{}
		pods := []*corev1.Pod{
			sourcePod("drc-shard0-0", "drc-shard0", "10.0.0.11"),
			sourcePod("drc-shard0-1", "drc-shard0", "10.0.0.12"),
		}
		infos := clusterInfosFrom(masterNode("m0", "10.0.0.11", 0, 100), slaveNode("s0", "10.0.0.12", "m0"))
		r := &DistributedRedisClusterReconciler{MetricsClient: &MockMetricsRecorder{}}
		ctx := &syncContext{admin: admin, clusterInfos: infos, pods: pods, reqLogger: logr.Discard()}

		if err := r.maintainSourceReplicas(ctx, ctxbg, instance); err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		admin.AssertNotCalled(t, "AttachSlaveToMaster", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		admin.AssertNotCalled(t, "AttachNodeToCluster", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("orphan pod is rejoined and re-attached as replica", func(t *testing.T) {
		admin := &mocks.MockRedisAdmin{}
		pods := []*corev1.Pod{
			sourcePod("drc-shard0-0", "drc-shard0", "10.0.0.11"), // current master
			sourcePod("drc-shard0-1", "drc-shard0", "10.0.0.12"), // orphan (empty master)
		}
		infos := clusterInfosFrom(masterNode("m0", "10.0.0.11", 0, 100), emptyNode("orphan", "10.0.0.12"))
		admin.On("AttachNodeToCluster", mock.Anything, mock.Anything, ns, name).Return(nil)
		admin.On("AttachSlaveToMaster", mock.Anything, mock.AnythingOfType("*redisutil.Node"), "m0", ns, name).Return(nil)

		r := &DistributedRedisClusterReconciler{MetricsClient: &MockMetricsRecorder{}}
		ctx := &syncContext{admin: admin, clusterInfos: infos, pods: pods, reqLogger: logr.Discard()}

		err := r.maintainSourceReplicas(ctx, ctxbg, instance)
		if err == nil || GetType(err) != Requeue {
			t.Fatalf("expected Requeue error, got %v", err)
		}
		admin.AssertCalled(t, "AttachSlaveToMaster", mock.Anything, mock.AnythingOfType("*redisutil.Node"), "m0", ns, name)
		admin.AssertNumberOfCalls(t, "AttachSlaveToMaster", 1)
	})

	t.Run("all pods unreachable requeues without panicking", func(t *testing.T) {
		admin := &mocks.MockRedisAdmin{}
		pods := []*corev1.Pod{sourcePod("drc-shard0-0", "drc-shard0", "10.0.0.11")}
		infos := clusterInfosFrom() // empty Infos: no reachable node to meet against
		r := &DistributedRedisClusterReconciler{MetricsClient: &MockMetricsRecorder{}}
		ctx := &syncContext{admin: admin, clusterInfos: infos, pods: pods, reqLogger: logr.Discard()}

		err := r.maintainSourceReplicas(ctx, ctxbg, instance)
		if err == nil || GetType(err) != Requeue {
			t.Fatalf("expected Requeue (no panic), got %v", err)
		}
		// waitForClusterJoin bails on the nil firstNode before ever issuing a MEET.
		admin.AssertNotCalled(t, "AttachNodeToCluster", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("pod with no matching node triggers join with an empty attach set", func(t *testing.T) {
		admin := &mocks.MockRedisAdmin{}
		pods := []*corev1.Pod{
			sourcePod("drc-shard0-0", "drc-shard0", "10.0.0.11"), // healthy master with slots, left alone
			sourcePod("drc-shard0-1", "drc-shard0", "10.0.0.12"), // not yet visible in the cluster at all
		}
		// Only "10.0.0.11" has a corresponding node in clusterInfos; "10.0.0.12" has none, so
		// findSourceNodeByIP returns nil for it and the needJoin path runs — with toAttach empty,
		// since neither pod ends up queued for AttachSlaveToMaster.
		infos := clusterInfosFrom(masterNode("m0", "10.0.0.11", 0, 100))
		admin.On("AttachNodeToCluster", mock.Anything, "10.0.0.11:6379", ns, name).Return(nil)

		r := &DistributedRedisClusterReconciler{MetricsClient: &MockMetricsRecorder{}}
		ctx := &syncContext{admin: admin, clusterInfos: infos, pods: pods, reqLogger: logr.Discard()}

		err := r.maintainSourceReplicas(ctx, ctxbg, instance)
		if err == nil || GetType(err) != Requeue {
			t.Fatalf("expected Requeue error, got %v", err)
		}
		admin.AssertCalled(t, "AttachNodeToCluster", mock.Anything, "10.0.0.11:6379", ns, name)
		admin.AssertNotCalled(t, "AttachSlaveToMaster", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})
}
