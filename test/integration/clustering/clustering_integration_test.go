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

	rediskunv1alpha1 "github.com/freshworks/redis-cluster-operator/api/v1alpha1"
	"github.com/freshworks/redis-cluster-operator/controllers/clustering"
	"github.com/freshworks/redis-cluster-operator/test/integration"
	"github.com/freshworks/redis-cluster-operator/utils/redisutil"
)

var _ = Describe("Clustering Integration Tests", func() {
	Context("When testing master placement logic", func() {
		var (
			namespace string
			drc       *rediskunv1alpha1.DistributedRedisCluster
		)

		BeforeEach(func() {
			namespace = "test-clustering-" + integration.RandomString(8)
			Expect(integration.CreateNamespace(namespace)).To(Succeed())

			drc = integration.CreateTestDRC("clustering-test", namespace)
			drc.Spec.MasterSize = 3
			drc.Spec.ClusterReplicas = 1
		})

		AfterEach(func() {
			if drc != nil {
				integration.K8sClient.Delete(integration.Ctx, drc)
			}
			Expect(integration.DeleteNamespace(namespace)).To(Succeed())
		})

		It("should create proper cluster context with correct master placement", func() {
			// Create DRC
			Expect(integration.K8sClient.Create(integration.Ctx, drc)).To(Succeed())

			// Create mock cluster infos for testing
			clusterInfos := &redisutil.ClusterInfos{
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
					"10.0.0.3:6379": {
						Node: &redisutil.Node{
							ID:          "node3",
							IP:          "10.0.0.3",
							Port:        "6379",
							Role:        rediskunv1alpha1.RedisClusterNodeRoleMaster,
							PodName:     "redis-cluster-2-0",
							StatefulSet: "redis-cluster-2",
							Zonename:    "zone-c",
						},
					},
				},
			}

			// Create cluster from infos
			rCluster, nodes, err := newRedisClusterFromInfos(clusterInfos, drc)
			Expect(err).NotTo(HaveOccurred())
			Expect(rCluster).NotTo(BeNil())
			Expect(len(nodes)).To(Equal(3))

			// Create clustering context
			clusterCtx := clustering.NewCtx(
				drc.Spec.HaConfig,
				rCluster,
				nodes,
				drc.Spec.MasterSize,
				drc.Spec.ClusterReplicas,
				drc.Name,
				integration.MetricsClient.Logger,
			)

			Expect(clusterCtx).NotTo(BeNil())

			// Test master dispatch
			err = clusterCtx.DispatchMasters()
			Expect(err).NotTo(HaveOccurred())

			// Verify masters are properly distributed
			currentMasters := clusterCtx.GetCurrentMasters()
			Expect(len(currentMasters)).To(Equal(int(drc.Spec.MasterSize)))

			// Verify zone distribution
			zones := make(map[string]int)
			for _, master := range currentMasters {
				zones[master.Zonename]++
			}
			Expect(len(zones)).To(BeNumerically(">=", 2)) // Should distribute across zones
		})

		It("should handle cluster rebalancing scenarios", func() {
			// Create DRC
			Expect(integration.K8sClient.Create(integration.Ctx, drc)).To(Succeed())

			// Create imbalanced cluster scenario
			clusterInfos := createImbalancedClusterInfos()
			rCluster, nodes, err := newRedisClusterFromInfos(clusterInfos, drc)
			Expect(err).NotTo(HaveOccurred())

			clusterCtx := clustering.NewCtx(
				drc.Spec.HaConfig,
				rCluster,
				nodes,
				drc.Spec.MasterSize,
				drc.Spec.ClusterReplicas,
				drc.Name,
				integration.MetricsClient.Logger,
			)

			// Test rebalancing logic
			needsRebalance := clusterCtx.NeedsRebalance()
			if needsRebalance {
				err = clusterCtx.RebalanceCluster()
				Expect(err).NotTo(HaveOccurred())
			}

			// Verify cluster is balanced after rebalancing
			Eventually(func() bool {
				return !clusterCtx.NeedsRebalance()
			}, time.Second*30, time.Millisecond*100).Should(BeTrue())
		})

		It("should handle master failure scenarios", func() {
			// Create DRC
			Expect(integration.K8sClient.Create(integration.Ctx, drc)).To(Succeed())

			// Create cluster with failed master
			clusterInfos := createClusterWithFailedMaster()
			rCluster, nodes, err := newRedisClusterFromInfos(clusterInfos, drc)
			Expect(err).NotTo(HaveOccurred())

			clusterCtx := clustering.NewCtx(
				drc.Spec.HaConfig,
				rCluster,
				nodes,
				drc.Spec.MasterSize,
				drc.Spec.ClusterReplicas,
				drc.Name,
				integration.MetricsClient.Logger,
			)

			// Check if cluster needs recovery
			needsRecovery := clusterCtx.NeedsRecovery()
			Expect(needsRecovery).To(BeTrue())

			// Test recovery logic would be implemented here
			// For now, just verify the detection works
		})
	})

	Context("When testing slot management", func() {
		It("should properly distribute slots across masters", func() {
			namespace := "test-slots-" + integration.RandomString(8)
			Expect(integration.CreateNamespace(namespace)).To(Succeed())
			defer func() {
				Expect(integration.DeleteNamespace(namespace)).To(Succeed())
			}()

			drc := integration.CreateTestDRC("slot-test", namespace)
			drc.Spec.MasterSize = 3

			// Create cluster with proper slot distribution
			clusterInfos := createClusterWithSlots()
			rCluster, nodes, err := newRedisClusterFromInfos(clusterInfos, drc)
			Expect(err).NotTo(HaveOccurred())

			clusterCtx := clustering.NewCtx(
				drc.Spec.HaConfig,
				rCluster,
				nodes,
				drc.Spec.MasterSize,
				drc.Spec.ClusterReplicas,
				drc.Name,
				integration.MetricsClient.Logger,
			)

			// Verify slot distribution
			masters := clusterCtx.GetCurrentMasters()
			totalSlots := 0
			for _, master := range masters {
				totalSlots += master.TotalSlots()
			}

			Expect(totalSlots).To(Equal(16384)) // Redis cluster has 16384 slots
		})
	})
})

// Helper functions for creating test cluster scenarios

func newRedisClusterFromInfos(infos *redisutil.ClusterInfos, drc *rediskunv1alpha1.DistributedRedisCluster) (*redisutil.Cluster, redisutil.Nodes, error) {
	// Mock implementation for testing
	nodes := make(redisutil.Nodes, 0, len(infos.Infos))
	for _, info := range infos.Infos {
		nodes = append(nodes, info.Node)
	}

	cluster := &redisutil.Cluster{
		Name:      drc.Name,
		Namespace: drc.Namespace,
		Nodes:     nodes,
	}

	return cluster, nodes, nil
}

func createImbalancedClusterInfos() *redisutil.ClusterInfos {
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
					Slots:       []redisutil.Slot{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, // Too many slots
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
					Slots:       []redisutil.Slot{10, 11}, // Too few slots
				},
			},
		},
	}
}

func createClusterWithFailedMaster() *redisutil.ClusterInfos {
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
					Slots:       []redisutil.Slot{0, 1, 2, 3, 4},
					Flags:       []string{"fail"}, // Failed master
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

func createClusterWithSlots() *redisutil.ClusterInfos {
	// Create a cluster with proper slot distribution
	slotsPerMaster := 16384 / 3
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
					Slots:       generateSlots(0, slotsPerMaster-1),
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
					Slots:       generateSlots(slotsPerMaster, 2*slotsPerMaster-1),
				},
			},
			"10.0.0.3:6379": {
				Node: &redisutil.Node{
					ID:          "node3",
					IP:          "10.0.0.3",
					Port:        "6379",
					Role:        rediskunv1alpha1.RedisClusterNodeRoleMaster,
					PodName:     "redis-cluster-2-0",
					StatefulSet: "redis-cluster-2",
					Zonename:    "zone-c",
					Slots:       generateSlots(2*slotsPerMaster, 16383),
				},
			},
		},
	}
}

func generateSlots(start, end int) []redisutil.Slot {
	slots := make([]redisutil.Slot, 0, end-start+1)
	for i := start; i <= end; i++ {
		slots = append(slots, redisutil.Slot(i))
	}
	return slots
}
