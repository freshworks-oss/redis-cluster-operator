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
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	rediskunv1alpha1 "github.com/freshworks/redis-cluster-operator/api/v1alpha1"
	"github.com/freshworks/redis-cluster-operator/controllers/heal"
	"github.com/freshworks/redis-cluster-operator/utils/redisutil"
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
			mockAdmin := &MockRedisAdmin{}

			// Test failed node detection and recovery
			actionDone, err := healer.FixFailedNodes(drc, clusterInfos, mockAdmin, Ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(actionDone).To(BeTrue())

			// Verify that failed nodes are properly handled
			// This would typically involve checking that the failed nodes are removed
			// and cluster is rebalanced
		})

		It("should handle cluster split scenarios", func() {
			// Create DRC
			Expect(K8sClient.Create(Ctx, drc)).To(Succeed())

			// Create cluster infos representing a split cluster
			clusterInfos := createSplitClusterInfos()
			mockAdmin := &MockRedisAdmin{}

			// Test cluster split detection and healing
			hasSplit, err := healer.FixClusterSplit(Ctx, mockAdmin, clusterInfos, drc.Namespace, drc.Name)
			if hasSplit {
				Expect(err).NotTo(HaveOccurred())
			}

			// Verify cluster is reunified
			Eventually(func() bool {
				// Check that cluster is no longer split
				return !hasClusterSplit(clusterInfos)
			}, time.Second*30, time.Millisecond*100).Should(BeTrue())
		})

		It("should fix untrusted nodes", func() {
			// Create DRC
			Expect(K8sClient.Create(Ctx, drc)).To(Succeed())

			// Create cluster with untrusted nodes
			clusterInfos := createClusterWithUntrustedNodes()
			mockAdmin := &MockRedisAdmin{}

			// Test untrusted node fixing
			actionDone, err := healer.FixUntrustedNodes(drc, clusterInfos, mockAdmin, Ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(actionDone).To(BeTrue())

			// Verify untrusted nodes are fixed
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
		It("should recover cluster from master failures", func() {
			namespace := "test-recovery-" + RandomString(8)
			Expect(CreateNamespace(namespace)).To(Succeed())
			defer func() {
				Expect(DeleteNamespace(namespace)).To(Succeed())
			}()

			drc := CreateTestDRC("recovery-test", namespace)
			Expect(K8sClient.Create(Ctx, drc)).To(Succeed())

			healer := heal.NewCheckAndHeal(
				K8sClient,
				MetricsClient.Logger,
				MetricsClient,
			)

			// Create scenario where masters need recovery
			clusterInfos := createClusterNeedingRecovery()
			mockAdmin := &MockRedisAdmin{}

			// Test cluster recovery
			stopReconcile, actionDone, err := healer.CheckandRecoverCluster(drc, clusterInfos, mockAdmin, Ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(stopReconcile).To(BeFalse())

			if actionDone {
				// Verify cluster is recovered
				Eventually(func() bool {
					return isClusterHealthy(clusterInfos)
				}, time.Second*30, time.Millisecond*100).Should(BeTrue())
			}
		})
	})
})

// Mock Redis Admin for testing
type MockRedisAdmin struct {
	// Add fields as needed for testing
}

func (m *MockRedisAdmin) FailoverSlaveToMaster(ctx context.Context, node *redisutil.Node, mode redisutil.FailoverMode, namespace, name string) error {
	// Mock implementation
	return nil
}

func (m *MockRedisAdmin) FlushAndReset(ctx context.Context, addr string, mode redisutil.ResetMode, namespace, name string) error {
	// Mock implementation
	return nil
}

func (m *MockRedisAdmin) AttachNodeToCluster(ctx context.Context, addr, namespace, name string) error {
	// Mock implementation
	return nil
}

func (m *MockRedisAdmin) ForgetNode(ctx context.Context, id, namespace, name string) error {
	// Mock implementation
	return nil
}

// Helper functions for creating test scenarios

func createClusterWithFailedNodes() *redisutil.ClusterInfos {
	return &redisutil.ClusterInfos{
		Infos: map[string]*redisutil.NodeInfos{
			"10.0.0.1:6379": {
				Node: &redisutil.Node{
					ID:          "node1",
					IP:          "10.0.0.1",
					Port:        "6379",
					Role:        rediskunv1alpha1.RedisClusterNodeRoleMaster,
					PodName:     "redis-cluster-0-0",
					StatefulSet: "redis-cluster-0",
					Zonename:    "zone-a",
					Flags:       []string{"fail"},
				},
			},
			"10.0.0.2:6379": {
				Node: &redisutil.Node{
					ID:             "node2",
					IP:             "10.0.0.2",
					Port:           "6379",
					Role:           rediskunv1alpha1.RedisClusterNodeRoleSlave,
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
					Role:        rediskunv1alpha1.RedisClusterNodeRoleMaster,
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
					Role:        rediskunv1alpha1.RedisClusterNodeRoleMaster,
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
					Role:        rediskunv1alpha1.RedisClusterNodeRoleMaster,
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
					Role:        rediskunv1alpha1.RedisClusterNodeRoleSlave,
					PodName:     "redis-cluster-0-1",
					StatefulSet: "redis-cluster-0",
					Zonename:    "zone-a",
					Flags:       []string{"noaddr"}, // Untrusted node
				},
			},
		},
	}
}

func createClusterNeedingRecovery() *redisutil.ClusterInfos {
	return &redisutil.ClusterInfos{
		Infos: map[string]*redisutil.NodeInfos{
			"10.0.0.1:6379": {
				Node: &redisutil.Node{
					ID:          "node1",
					IP:          "10.0.0.1",
					Port:        "6379",
					Role:        rediskunv1alpha1.RedisClusterNodeRoleMaster,
					PodName:     "redis-cluster-0-0",
					StatefulSet: "redis-cluster-0",
					Zonename:    "zone-a",
					Flags:       []string{"fail"},
					Slots:       []redisutil.Slot{0, 1, 2, 3, 4},
				},
			},
			"10.0.0.2:6379": {
				Node: &redisutil.Node{
					ID:             "node2",
					IP:             "10.0.0.2",
					Port:           "6379",
					Role:           rediskunv1alpha1.RedisClusterNodeRoleSlave,
					PodName:        "redis-cluster-0-1",
					StatefulSet:    "redis-cluster-0",
					Zonename:       "zone-a",
					MasterReferent: "node1",
				},
			},
		},
	}
}

func hasClusterSplit(infos *redisutil.ClusterInfos) bool {
	return infos.Status == redisutil.ClusterInfosPartial
}

func isClusterHealthy(infos *redisutil.ClusterInfos) bool {
	for _, info := range infos.Infos {
		if info.Node.HasStatus("fail") || info.Node.HasStatus("noaddr") {
			return false
		}
	}
	return true
}
