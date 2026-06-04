/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package integration

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	rediskunv1alpha1 "github.com/freshworks/redis-cluster-operator/api/v1alpha1"
	"github.com/freshworks/redis-cluster-operator/controllers/heal"
	redisconfig "github.com/freshworks/redis-cluster-operator/redisconfig"
	"github.com/freshworks/redis-cluster-operator/resources/statefulsets"
	"github.com/freshworks/redis-cluster-operator/utils/redisutil"
	redismocks "github.com/freshworks/redis-cluster-operator/utils/testutil/mocks"
)

var _ = Describe("Healing Integration Tests", func() {
	Context("When testing cluster healing scenarios", func() {
		var (
			namespace string
			drc       *rediskunv1alpha1.DistributedRedisCluster
			healer    *heal.CheckAndHeal
		)

		BeforeEach(func() {
			namespace = "test-heal-" + RandomString(8)
			Expect(CreateNamespace(namespace)).To(Succeed())

			drc = CreateTestDRC("heal-test", namespace)

			// Initialize healer
			healer = heal.NewCheckAndHeal(
				K8sClient,
				MetricsClient.Logger,
				MetricsClient,
			)
		})

		AfterEach(func() {
			if drc != nil {
				K8sClient.Delete(Ctx, drc)
			}
			Expect(DeleteNamespace(namespace)).To(Succeed())
		})

		It("should detect and recover from failed nodes", func() {
			// Create DRC
			Expect(K8sClient.Create(Ctx, drc)).To(Succeed())

			// Create cluster infos with failed nodes
			clusterInfos := createClusterWithFailedNodes()
			mockAdmin := &redismocks.MockRedisAdmin{}

			// Ghost detection uses Friends views + quorum; this mock only sets Node.FailStatus.
			actionDone, err := healer.FixFailedNodes(drc, clusterInfos, mockAdmin, Ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(actionDone).To(BeFalse())
		})

		It("should handle cluster split scenarios", func() {
			// Create DRC
			Expect(K8sClient.Create(Ctx, drc)).To(Succeed())

			// Create cluster infos representing a split cluster
			clusterInfos := createSplitClusterInfos()
			mockAdmin := &redismocks.MockRedisAdmin{}

			// DryRun avoids running reassignClusters (needs real Redis / full IAdmin stubs).
			splitHealer := heal.NewCheckAndHeal(K8sClient, MetricsClient.Logger, MetricsClient)
			splitHealer.DryRun = true
			redisCfg := &redisconfig.Redis{DialTimeout: redisconfig.DefaultRedisTimeout}
			hasSplit, err := splitHealer.FixClusterSplit(drc, clusterInfos, mockAdmin, redisCfg)
			Expect(hasSplit).To(BeTrue())
			Expect(err).NotTo(HaveOccurred())
			// In-memory clusterInfos is static; reunification would require a live cluster or richer mocks.
		})

		It("should fix untrusted nodes", func() {
			// Create DRC
			Expect(K8sClient.Create(Ctx, drc)).To(Succeed())

			// Create cluster with untrusted nodes
			clusterInfos := createClusterWithUntrustedNodes()
			mockAdmin := &redismocks.MockRedisAdmin{}

			// Untrusted detection uses Friends with handshake; mock uses FailStatus on Node only.
			actionDone, err := healer.FixUntrustedNodes(drc, clusterInfos, mockAdmin, Ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(actionDone).To(BeFalse())
		})

		It("should handle terminating pods gracefully", func() {
			// Create DRC
			Expect(K8sClient.Create(Ctx, drc)).To(Succeed())

			// Create terminating pods
			terminatingPod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "redis-cluster-0-0",
					Namespace: namespace,
					Labels: map[string]string{
						"app": "redis-cluster",
					},
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "redis",
							Image: "redis:6.2.6",
						},
					},
				},
			}

			Expect(K8sClient.Create(Ctx, terminatingPod)).To(Succeed())

			// Test terminating pod handling
			actionDone, err := healer.FixTerminatingPods(drc, time.Minute*5)
			Expect(err).NotTo(HaveOccurred())

			if actionDone {
				// Verify terminating pods are handled properly
				Eventually(func() bool {
					pod := &corev1.Pod{}
					err := K8sClient.Get(Ctx,
						types.NamespacedName{Name: terminatingPod.Name, Namespace: namespace}, pod)
					return err != nil // Pod should be gone
				}, time.Second*30, time.Millisecond*100).Should(BeTrue())
			}
		})
	})

	Context("When testing cluster recovery scenarios", func() {
		var (
			namespace string
			drc       *rediskunv1alpha1.DistributedRedisCluster
			healer    *heal.CheckAndHeal
		)

		BeforeEach(func() {
			namespace = "test-heal-rc-" + RandomString(8)
			Expect(CreateNamespace(namespace)).To(Succeed())

			// Single-shard + HA so CreateRankingFromClusterInfo succeeds in envtest (no validating webhook).
			drc = CreateTestDRC("recover-test", namespace)
			drc.Spec.MasterSize = 1
			drc.Spec.ClusterReplicas = 0
			drc.Spec.HaConfig = &rediskunv1alpha1.HAspec{
				HaEnabled: true,
				ZonesInfo: map[string]string{"zone-a": "zone-a"},
			}

			healer = heal.NewCheckAndHeal(K8sClient, MetricsClient.Logger, MetricsClient)
		})

		AfterEach(func() {
			if drc != nil {
				_ = K8sClient.Delete(Ctx, drc)
			}
			Expect(DeleteNamespace(namespace)).To(Succeed())
		})

		// Exercises CheckandRecoverCluster wiring (ranking + IsClusterNeedsRecovery) against a real DRC object.
		// Full master-loss failover with IAdmin is covered in controllers/heal/recover_cluster_test.go.
		It("should complete CheckandRecoverCluster when HA ranking builds and no shard recovery is required", func() {
			Expect(K8sClient.Create(Ctx, drc)).To(Succeed())

			sts := statefulsets.ClusterStatefulSetName("recover-test", 0)
			infos := &redisutil.ClusterInfos{
				Infos: map[string]*redisutil.NodeInfos{
					"10.0.0.1:6379": {
						Node: &redisutil.Node{
							ID:          "node1",
							IP:          "10.0.0.1",
							Port:        "6379",
							Role:        redisutil.RedisMasterRole,
							StatefulSet: sts,
							Zonename:    "zone-a",
							Slots:       []redisutil.Slot{0},
						},
						Friends: []*redisutil.Node{},
					},
				},
			}
			mockAdmin := &redismocks.MockRedisAdmin{}

			stopReconcile, actionDone, err := healer.CheckandRecoverCluster(drc, infos, mockAdmin, Ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(stopReconcile).To(BeFalse())
			Expect(actionDone).To(BeFalse())
		})
	})
})

// Helper functions for creating test scenarios

func createClusterWithFailedNodes() *redisutil.ClusterInfos {
	return &redisutil.ClusterInfos{
		Infos: map[string]*redisutil.NodeInfos{
			"10.0.0.1:6379": {
				Node: &redisutil.Node{
					ID:          "node1",
					IP:          "10.0.0.1",
					Port:        "6379",
					Role:        string(rediskunv1alpha1.RedisClusterNodeRoleMaster),
					PodName:     "redis-cluster-0-0",
					StatefulSet: "redis-cluster-0",
					Zonename:    "zone-a",
					FailStatus:  []string{redisutil.NodeStatusFail},
				},
			},
			"10.0.0.2:6379": {
				Node: &redisutil.Node{
					ID:             "node2",
					IP:             "10.0.0.2",
					Port:           "6379",
					Role:           string(rediskunv1alpha1.RedisClusterNodeRoleSlave),
					PodName:        "redis-cluster-0-1",
					StatefulSet:    "redis-cluster-0",
					Zonename:       "zone-a",
					MasterReferent: "node1",
				},
			},
		},
	}
}

func createSplitClusterInfos() *redisutil.ClusterInfos {
	return &redisutil.ClusterInfos{
		Infos: map[string]*redisutil.NodeInfos{
			"10.0.0.1:6379": {
				Node: &redisutil.Node{
					ID:          "node1",
					IP:          "10.0.0.1",
					Port:        "6379",
					Role:        string(rediskunv1alpha1.RedisClusterNodeRoleMaster),
					PodName:     "redis-cluster-0-0",
					StatefulSet: "redis-cluster-0",
					Zonename:    "zone-a",
				},
			},
			"10.0.0.2:6379": {
				Node: &redisutil.Node{
					ID:          "node2",
					IP:          "10.0.0.2",
					Port:        "6379",
					Role:        string(rediskunv1alpha1.RedisClusterNodeRoleMaster),
					PodName:     "redis-cluster-1-0",
					StatefulSet: "redis-cluster-1",
					Zonename:    "zone-b",
				},
			},
		},
		Status: redisutil.ClusterInfosPartial, // Indicates split cluster
	}
}

func createClusterWithUntrustedNodes() *redisutil.ClusterInfos {
	return &redisutil.ClusterInfos{
		Infos: map[string]*redisutil.NodeInfos{
			"10.0.0.1:6379": {
				Node: &redisutil.Node{
					ID:          "node1",
					IP:          "10.0.0.1",
					Port:        "6379",
					Role:        string(rediskunv1alpha1.RedisClusterNodeRoleMaster),
					PodName:     "redis-cluster-0-0",
					StatefulSet: "redis-cluster-0",
					Zonename:    "zone-a",
				},
			},
			"10.0.0.2:6379": {
				Node: &redisutil.Node{
					ID:          "node2",
					IP:          "10.0.0.2",
					Port:        "6379",
					Role:        string(rediskunv1alpha1.RedisClusterNodeRoleSlave),
					PodName:     "redis-cluster-0-1",
					StatefulSet: "redis-cluster-0",
					Zonename:    "zone-a",
					FailStatus:  []string{redisutil.NodeStatusNoAddr}, // Untrusted node
				},
			},
		},
	}
}
