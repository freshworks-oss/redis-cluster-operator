package distributedrediscluster

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	redisv1alpha1 "github.com/freshworks/redis-cluster-operator/api/v1alpha1"
	"github.com/freshworks/redis-cluster-operator/utils/redisutil"
	"github.com/freshworks/redis-cluster-operator/utils/testutil/mocks"
)

// TestVerifyBootstrapJoin covers Change 1: the join check requires both that the local pod IP is
// visible on each source node AND that the gossip link to it is "connected" — and that this holds
// for every remote, not just the first one checked.
func TestVerifyBootstrapJoin(t *testing.T) {
	ns, name := "default", "drc"
	instance := &redisv1alpha1.DistributedRedisCluster{ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name}}
	remote1 := &redisutil.Node{ID: "r0", IP: "10.8.8.8", Port: "6379", LinkState: redisutil.RedisLinkStateConnected}
	remote2 := &redisutil.Node{ID: "r1", IP: "10.8.8.9", Port: "6379", LinkState: redisutil.RedisLinkStateConnected}
	remoteNodes := redisutil.Nodes{remote1, remote2}
	local := []bootstrapPodRef{{ipPort: "10.0.0.11:6379"}}

	localView := func(self *redisutil.Node, linkState string) *redisutil.NodeInfos {
		return &redisutil.NodeInfos{
			Node:    self,
			Friends: redisutil.Nodes{{ID: "l0", IP: "10.0.0.11", Port: "6379", LinkState: linkState}},
		}
	}

	t.Run("all remotes report connected passes", func(t *testing.T) {
		admin := &mocks.MockRedisAdmin{}
		admin.On("GetClusterNodeInfos", mock.Anything, "10.8.8.8:6379", ns, name).Return(localView(remote1, redisutil.RedisLinkStateConnected), nil)
		admin.On("GetClusterNodeInfos", mock.Anything, "10.8.8.9:6379", ns, name).Return(localView(remote2, redisutil.RedisLinkStateConnected), nil)
		ctx := &syncContext{admin: admin}
		if err := verifyBootstrapJoin(context.Background(), ctx, instance, remoteNodes, local); err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})

	t.Run("one remote disconnected requeues even though another is connected", func(t *testing.T) {
		admin := &mocks.MockRedisAdmin{}
		admin.On("GetClusterNodeInfos", mock.Anything, "10.8.8.8:6379", ns, name).Return(localView(remote1, redisutil.RedisLinkStateConnected), nil)
		admin.On("GetClusterNodeInfos", mock.Anything, "10.8.8.9:6379", ns, name).Return(localView(remote2, redisutil.RedisLinkStateDisconnected), nil)
		ctx := &syncContext{admin: admin}
		err := verifyBootstrapJoin(context.Background(), ctx, instance, remoteNodes, local)
		if err == nil || GetType(err) != Requeue {
			t.Fatalf("expected Requeue, got %v", err)
		}
	})

	t.Run("one remote does not list the pod requeues even though another is connected", func(t *testing.T) {
		admin := &mocks.MockRedisAdmin{}
		admin.On("GetClusterNodeInfos", mock.Anything, "10.8.8.8:6379", ns, name).Return(localView(remote1, redisutil.RedisLinkStateConnected), nil)
		admin.On("GetClusterNodeInfos", mock.Anything, "10.8.8.9:6379", ns, name).Return(&redisutil.NodeInfos{Node: remote2}, nil)
		ctx := &syncContext{admin: admin}
		err := verifyBootstrapJoin(context.Background(), ctx, instance, remoteNodes, local)
		if err == nil || GetType(err) != Requeue {
			t.Fatalf("expected Requeue, got %v", err)
		}
	})

	// A remote the seed itself reports as disconnected (stale/ghost entry) must be skipped, not
	// queried — otherwise its failing GetClusterNodeInfos would wedge the join check forever.
	t.Run("disconnected remote is skipped and connected remote still verified", func(t *testing.T) {
		admin := &mocks.MockRedisAdmin{}
		deadRemote := &redisutil.Node{ID: "r2", IP: "10.8.8.10", Port: "6379", LinkState: redisutil.RedisLinkStateDisconnected}
		nodes := redisutil.Nodes{remote1, deadRemote}
		// Only remote1 is stubbed; deadRemote must never be queried (an unstubbed call panics the mock).
		admin.On("GetClusterNodeInfos", mock.Anything, "10.8.8.8:6379", ns, name).Return(localView(remote1, redisutil.RedisLinkStateConnected), nil)
		ctx := &syncContext{admin: admin}
		if err := verifyBootstrapJoin(context.Background(), ctx, instance, nodes, local); err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		admin.AssertNotCalled(t, "GetClusterNodeInfos", mock.Anything, "10.8.8.10:6379", ns, name)
	})
}

// TestReplicateBootstrapPods covers Change 2: one attach per shard is issued in a single pass, so
// shards proceed in parallel while pods within a shard are serialized.
func TestReplicateBootstrapPods(t *testing.T) {
	ns, name := "default", "drc"
	ctxbg := context.Background()
	instance := &redisv1alpha1.DistributedRedisCluster{ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name}}
	r := &DistributedRedisClusterReconciler{MetricsClient: &MockMetricsRecorder{}}

	remoteMasters := redisutil.Nodes{
		masterNode("m0", "10.8.8.1", 0),
		masterNode("m1", "10.8.8.2", 100),
	}
	localPods := []bootstrapPodRef{
		{pod: sourcePod("drc-shard0-0", "drc-shard0", "10.0.0.11"), shardIndex: 0, ordinal: 0, ipPort: "10.0.0.11:6379"},
		{pod: sourcePod("drc-shard0-1", "drc-shard0", "10.0.0.12"), shardIndex: 0, ordinal: 1, ipPort: "10.0.0.12:6379"},
		{pod: sourcePod("drc-shard1-0", "drc-shard1", "10.0.0.21"), shardIndex: 1, ordinal: 0, ipPort: "10.0.0.21:6379"},
	}

	admin := &mocks.MockRedisAdmin{}
	// shard0-ordinal0 and shard1-ordinal0 are freshly joined (empty masters, no slots) → both need
	// attaching. shard0-ordinal1 is deliberately left unstubbed: within-shard serialization means
	// its node info must never even be inspected while ordinal0 is still being attached this pass —
	// an unexpected call here would panic the mock and fail the test.
	admin.On("GetClusterNodeInfos", mock.Anything, "10.0.0.11:6379", ns, name).Return(&redisutil.NodeInfos{Node: emptyNode("l0", "10.0.0.11")}, nil)
	admin.On("GetClusterNodeInfos", mock.Anything, "10.0.0.21:6379", ns, name).Return(&redisutil.NodeInfos{Node: emptyNode("l1", "10.0.0.21")}, nil)
	admin.On("AttachSlaveToMaster", mock.Anything, mock.AnythingOfType("*redisutil.Node"), "m0", ns, name).Return(nil)
	admin.On("AttachSlaveToMaster", mock.Anything, mock.AnythingOfType("*redisutil.Node"), "m1", ns, name).Return(nil)

	ctx := &syncContext{admin: admin}
	err := r.replicateBootstrapPods(ctxbg, ctx, instance, remoteMasters, localPods)
	if err == nil || GetType(err) != Requeue {
		t.Fatalf("expected Requeue while replication in progress, got %v", err)
	}
	// Both shards advanced in the same pass (parallel across shards), but only one pod per shard —
	// shard0-ordinal1 is untouched this pass (serialized within its shard behind ordinal0, picked up
	// only on a later reconciliation pass).
	admin.AssertCalled(t, "AttachSlaveToMaster", mock.Anything, mock.AnythingOfType("*redisutil.Node"), "m0", ns, name)
	admin.AssertCalled(t, "AttachSlaveToMaster", mock.Anything, mock.AnythingOfType("*redisutil.Node"), "m1", ns, name)
	admin.AssertNumberOfCalls(t, "AttachSlaveToMaster", 2)
	admin.AssertNotCalled(t, "GetClusterNodeInfos", mock.Anything, "10.0.0.12:6379", ns, name)
}
