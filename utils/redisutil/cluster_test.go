package redisutil

import (
	"testing"
)

// TestNewCluster tests the NewCluster constructor
func TestNewCluster(t *testing.T) {
	tests := []struct {
		name        string
		clusterName string
		namespace   string
	}{
		{
			name:        "Create cluster with valid name and namespace",
			clusterName: "test-cluster",
			namespace:   "test-namespace",
		},
		{
			name:        "Create cluster with empty name",
			clusterName: "",
			namespace:   "test-namespace",
		},
		{
			name:        "Create cluster with empty namespace",
			clusterName: "test-cluster",
			namespace:   "",
		},
		{
			name:        "Create cluster with both empty",
			clusterName: "",
			namespace:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cluster := NewCluster(tt.clusterName, tt.namespace)

			if cluster == nil {
				t.Fatal("NewCluster() returned nil")
			}

			if cluster.Name != tt.clusterName {
				t.Errorf("NewCluster() name = %v, expected %v", cluster.Name, tt.clusterName)
			}

			if cluster.Namespace != tt.namespace {
				t.Errorf("NewCluster() namespace = %v, expected %v", cluster.Namespace, tt.namespace)
			}

			if cluster.Nodes == nil {
				t.Error("NewCluster() should initialize Nodes map")
			}

			if len(cluster.Nodes) != 0 {
				t.Errorf("NewCluster() should initialize empty Nodes map, got length %d", len(cluster.Nodes))
			}
		})
	}
}

// TestCluster_AddNode tests the AddNode method
func TestCluster_AddNode(t *testing.T) {
	cluster := NewCluster("test-cluster", "test-namespace")

	// Create test nodes
	node1 := &Node{
		ID:      "node1",
		IP:      "192.168.1.10",
		Port:    "6379",
		Role:    "master",
		PodName: "redis-0",
	}

	node2 := &Node{
		ID:      "node2",
		IP:      "192.168.1.11",
		Port:    "6379",
		Role:    "slave",
		PodName: "redis-1",
	}

	node1Updated := &Node{
		ID:      "node1",        // Same ID as node1
		IP:      "192.168.1.12", // Different IP
		Port:    "6379",
		Role:    "master",
		PodName: "redis-0-updated",
	}

	t.Run("Add first node", func(t *testing.T) {
		cluster.AddNode(node1)

		if len(cluster.Nodes) != 1 {
			t.Errorf("Expected 1 node, got %d", len(cluster.Nodes))
		}

		storedNode, exists := cluster.Nodes[node1.ID]
		if !exists {
			t.Error("Node should exist in cluster")
		}

		if storedNode != node1 {
			t.Error("Stored node should be the same instance")
		}
	})

	t.Run("Add second node", func(t *testing.T) {
		cluster.AddNode(node2)

		if len(cluster.Nodes) != 2 {
			t.Errorf("Expected 2 nodes, got %d", len(cluster.Nodes))
		}

		if cluster.Nodes[node2.ID] != node2 {
			t.Error("Second node should be stored correctly")
		}
	})

	t.Run("Replace existing node with same ID", func(t *testing.T) {
		// This should call Clear() on the existing node and replace it
		cluster.AddNode(node1Updated)

		if len(cluster.Nodes) != 2 {
			t.Errorf("Expected 2 nodes after replacement, got %d", len(cluster.Nodes))
		}

		storedNode := cluster.Nodes[node1.ID]
		if storedNode.IP != node1Updated.IP {
			t.Errorf("Node should be updated, expected IP %v, got %v", node1Updated.IP, storedNode.IP)
		}

		if storedNode.PodName != node1Updated.PodName {
			t.Errorf("Node should be updated, expected PodName %v, got %v", node1Updated.PodName, storedNode.PodName)
		}
	})

	t.Run("Add nil node", func(t *testing.T) {
		// This will cause a panic, but we should handle it gracefully
		defer func() {
			if r := recover(); r == nil {
				t.Error("Adding nil node should panic")
			}
		}()

		cluster.AddNode(nil)
	})
}

// TestCluster_GetNodeByID tests the GetNodeByID method
func TestCluster_GetNodeByID(t *testing.T) {
	cluster := NewCluster("test-cluster", "test-namespace")

	node1 := &Node{
		ID:      "node1",
		IP:      "192.168.1.10",
		Port:    "6379",
		Role:    "master",
		PodName: "redis-0",
	}

	node2 := &Node{
		ID:      "node2",
		IP:      "192.168.1.11",
		Port:    "6379",
		Role:    "slave",
		PodName: "redis-1",
	}

	cluster.AddNode(node1)
	cluster.AddNode(node2)

	tests := []struct {
		name         string
		nodeID       string
		expectError  bool
		expectedNode *Node
	}{
		{
			name:         "Get existing node1",
			nodeID:       "node1",
			expectError:  false,
			expectedNode: node1,
		},
		{
			name:         "Get existing node2",
			nodeID:       "node2",
			expectError:  false,
			expectedNode: node2,
		},
		{
			name:        "Get non-existing node",
			nodeID:      "non-existing",
			expectError: true,
		},
		{
			name:        "Get node with empty ID",
			nodeID:      "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := cluster.GetNodeByID(tt.nodeID)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if !IsNodeNotFoundedError(err) {
					t.Errorf("Expected nodeNotFoundedError, got %v", err)
				}
				if node != nil {
					t.Error("Expected nil node when error occurs")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if node != tt.expectedNode {
					t.Errorf("Expected node %v, got %v", tt.expectedNode, node)
				}
			}
		})
	}
}

// TestCluster_GetNodeByIP tests the GetNodeByIP method
func TestCluster_GetNodeByIP(t *testing.T) {
	cluster := NewCluster("test-cluster", "test-namespace")

	node1 := &Node{
		ID:      "node1",
		IP:      "192.168.1.10",
		Port:    "6379",
		Role:    "master",
		PodName: "redis-0",
	}

	node2 := &Node{
		ID:      "node2",
		IP:      "192.168.1.11",
		Port:    "6379",
		Role:    "slave",
		PodName: "redis-1",
	}

	cluster.AddNode(node1)
	cluster.AddNode(node2)

	tests := []struct {
		name         string
		nodeIP       string
		expectError  bool
		expectedNode *Node
	}{
		{
			name:         "Get node by existing IP1",
			nodeIP:       "192.168.1.10",
			expectError:  false,
			expectedNode: node1,
		},
		{
			name:         "Get node by existing IP2",
			nodeIP:       "192.168.1.11",
			expectError:  false,
			expectedNode: node2,
		},
		{
			name:        "Get node by non-existing IP",
			nodeIP:      "192.168.1.99",
			expectError: true,
		},
		{
			name:        "Get node by empty IP",
			nodeIP:      "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := cluster.GetNodeByIP(tt.nodeIP)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if !IsNodeNotFoundedError(err) {
					t.Errorf("Expected nodeNotFoundedError, got %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if node != tt.expectedNode {
					t.Errorf("Expected node %v, got %v", tt.expectedNode, node)
				}
			}
		})
	}
}

// TestCluster_GetNodeByPodName tests the GetNodeByPodName method
func TestCluster_GetNodeByPodName(t *testing.T) {
	cluster := NewCluster("test-cluster", "test-namespace")

	node1 := &Node{
		ID:      "node1",
		IP:      "192.168.1.10",
		Port:    "6379",
		Role:    "master",
		PodName: "redis-0",
	}

	node2 := &Node{
		ID:      "node2",
		IP:      "192.168.1.11",
		Port:    "6379",
		Role:    "slave",
		PodName: "redis-1",
	}

	cluster.AddNode(node1)
	cluster.AddNode(node2)

	tests := []struct {
		name         string
		podName      string
		expectError  bool
		expectedNode *Node
	}{
		{
			name:         "Get node by existing PodName1",
			podName:      "redis-0",
			expectError:  false,
			expectedNode: node1,
		},
		{
			name:         "Get node by existing PodName2",
			podName:      "redis-1",
			expectError:  false,
			expectedNode: node2,
		},
		{
			name:        "Get node by non-existing PodName",
			podName:     "redis-99",
			expectError: true,
		},
		{
			name:        "Get node by empty PodName",
			podName:     "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := cluster.GetNodeByPodName(tt.podName)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if !IsNodeNotFoundedError(err) {
					t.Errorf("Expected nodeNotFoundedError, got %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if node != tt.expectedNode {
					t.Errorf("Expected node %v, got %v", tt.expectedNode, node)
				}
			}
		})
	}
}

// TestCluster_GetNodeByFunc tests the GetNodeByFunc method
func TestCluster_GetNodeByFunc(t *testing.T) {
	cluster := NewCluster("test-cluster", "test-namespace")

	node1 := &Node{
		ID:      "node1",
		IP:      "192.168.1.10",
		Port:    "6379",
		Role:    "master",
		PodName: "redis-0",
	}

	node2 := &Node{
		ID:      "node2",
		IP:      "192.168.1.11",
		Port:    "6379",
		Role:    "slave",
		PodName: "redis-1",
	}

	cluster.AddNode(node1)
	cluster.AddNode(node2)

	tests := []struct {
		name         string
		findFunc     FindNodeFunc
		expectError  bool
		expectedNode *Node
	}{
		{
			name: "Find node by Role master",
			findFunc: func(node *Node) bool {
				return node.Role == "master"
			},
			expectError:  false,
			expectedNode: node1,
		},
		{
			name: "Find node by Role slave",
			findFunc: func(node *Node) bool {
				return node.Role == "slave"
			},
			expectError:  false,
			expectedNode: node2,
		},
		{
			name: "Find node by port",
			findFunc: func(node *Node) bool {
				return node.Port == "6379"
			},
			expectError: false,
			// Will return one of the nodes (map iteration order is not guaranteed)
		},
		{
			name: "Find non-existing node",
			findFunc: func(node *Node) bool {
				return node.Role == "sentinel"
			},
			expectError: true,
		},
		{
			name: "Find with always false function",
			findFunc: func(node *Node) bool {
				return false
			},
			expectError: true,
		},
		{
			name: "Find with always true function",
			findFunc: func(node *Node) bool {
				return true
			},
			expectError: false,
			// Will return one of the nodes
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := cluster.GetNodeByFunc(tt.findFunc)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if !IsNodeNotFoundedError(err) {
					t.Errorf("Expected nodeNotFoundedError, got %v", err)
				}
				if node != nil {
					t.Error("Expected nil node when error occurs")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if node == nil {
					t.Error("Expected node but got nil")
				}
				// Verify the returned node matches the find function
				if !tt.findFunc(node) {
					t.Error("Returned node should match the find function criteria")
				}
				if tt.expectedNode != nil && node != tt.expectedNode {
					t.Errorf("Expected specific node %v, got %v", tt.expectedNode, node)
				}
			}
		})
	}
}

// TestCluster_GetNodesByFunc tests the GetNodesByFunc method
func TestCluster_GetNodesByFunc(t *testing.T) {
	cluster := NewCluster("test-cluster", "test-namespace")

	node1 := &Node{
		ID:      "node1",
		IP:      "192.168.1.10",
		Port:    "6379",
		Role:    "master",
		PodName: "redis-0",
	}

	node2 := &Node{
		ID:      "node2",
		IP:      "192.168.1.11",
		Port:    "6379",
		Role:    "slave",
		PodName: "redis-1",
	}

	node3 := &Node{
		ID:      "node3",
		IP:      "192.168.1.12",
		Port:    "6379",
		Role:    "slave",
		PodName: "redis-2",
	}

	cluster.AddNode(node1)
	cluster.AddNode(node2)
	cluster.AddNode(node3)

	tests := []struct {
		name          string
		findFunc      FindNodeFunc
		expectError   bool
		expectedCount int
		expectedNodes []*Node
	}{
		{
			name: "Find all nodes with port 6379",
			findFunc: func(node *Node) bool {
				return node.Port == "6379"
			},
			expectError:   false,
			expectedCount: 3,
			expectedNodes: []*Node{node1, node2, node3},
		},
		{
			name: "Find all slave nodes",
			findFunc: func(node *Node) bool {
				return node.Role == "slave"
			},
			expectError:   false,
			expectedCount: 2,
			expectedNodes: []*Node{node2, node3},
		},
		{
			name: "Find all master nodes",
			findFunc: func(node *Node) bool {
				return node.Role == "master"
			},
			expectError:   false,
			expectedCount: 1,
			expectedNodes: []*Node{node1},
		},
		{
			name: "Find nodes with non-existing criteria",
			findFunc: func(node *Node) bool {
				return node.Role == "sentinel"
			},
			expectError: true,
		},
		{
			name: "Find with always false function",
			findFunc: func(node *Node) bool {
				return false
			},
			expectError: true,
		},
		{
			name: "Find nodes by IP pattern",
			findFunc: func(node *Node) bool {
				return len(node.IP) > 10 // All our test IPs are longer than 10 chars
			},
			expectError:   false,
			expectedCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := cluster.GetNodesByFunc(tt.findFunc)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if !IsNodeNotFoundedError(err) {
					t.Errorf("Expected nodeNotFoundedError, got %v", err)
				}
				if len(nodes) != 0 {
					t.Error("Expected empty nodes slice when error occurs")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if len(nodes) != tt.expectedCount {
					t.Errorf("Expected %d nodes, got %d", tt.expectedCount, len(nodes))
				}

				// Verify all returned nodes match the find function
				for _, node := range nodes {
					if !tt.findFunc(node) {
						t.Error("All returned nodes should match the find function criteria")
					}
				}

				// If specific nodes are expected, verify them
				if tt.expectedNodes != nil {
					foundNodes := make(map[string]bool)
					for _, node := range nodes {
						foundNodes[node.ID] = true
					}
					for _, expectedNode := range tt.expectedNodes {
						if !foundNodes[expectedNode.ID] {
							t.Errorf("Expected node %s not found in results", expectedNode.ID)
						}
					}
				}
			}
		})
	}
}

// TestCluster_Integration tests integration scenarios
func TestCluster_Integration(t *testing.T) {
	t.Run("Complete cluster workflow", func(t *testing.T) {
		cluster := NewCluster("production-cluster", "redis-namespace")

		// Add multiple nodes
		masterNode := &Node{
			ID:      "master-1",
			IP:      "10.0.1.10",
			Port:    "6379",
			Role:    "master",
			PodName: "redis-master-0",
		}

		slaveNode1 := &Node{
			ID:      "slave-1",
			IP:      "10.0.1.11",
			Port:    "6379",
			Role:    "slave",
			PodName: "redis-slave-0",
		}

		slaveNode2 := &Node{
			ID:      "slave-2",
			IP:      "10.0.1.12",
			Port:    "6379",
			Role:    "slave",
			PodName: "redis-slave-1",
		}

		cluster.AddNode(masterNode)
		cluster.AddNode(slaveNode1)
		cluster.AddNode(slaveNode2)

		// Test various query methods
		if len(cluster.Nodes) != 3 {
			t.Errorf("Expected 3 nodes in cluster, got %d", len(cluster.Nodes))
		}

		// Test GetNodeByID
		node, err := cluster.GetNodeByID("master-1")
		if err != nil || node != masterNode {
			t.Error("Should find master node by ID")
		}

		// Test GetNodeByIP
		node, err = cluster.GetNodeByIP("10.0.1.11")
		if err != nil || node != slaveNode1 {
			t.Error("Should find slave node by IP")
		}

		// Test GetNodeByPodName
		node, err = cluster.GetNodeByPodName("redis-slave-1")
		if err != nil || node != slaveNode2 {
			t.Error("Should find slave node by pod name")
		}

		// Test GetNodesByFunc for all slaves
		slaves, err := cluster.GetNodesByFunc(func(node *Node) bool {
			return node.Role == "slave"
		})
		if err != nil || len(slaves) != 2 {
			t.Error("Should find 2 slave nodes")
		}

		// Test node replacement
		updatedMaster := &Node{
			ID:      "master-1",   // Same ID
			IP:      "10.0.1.100", // New IP
			Port:    "6379",
			Role:    "master",
			PodName: "redis-master-0-updated",
		}

		cluster.AddNode(updatedMaster)
		if len(cluster.Nodes) != 3 {
			t.Error("Node replacement should not change total count")
		}

		retrievedNode, _ := cluster.GetNodeByID("master-1")
		if retrievedNode.IP != "10.0.1.100" {
			t.Error("Node should be updated with new IP")
		}
	})

	t.Run("Empty cluster operations", func(t *testing.T) {
		emptyCluster := NewCluster("empty", "test")

		// All get operations should return nodeNotFoundedError
		_, err := emptyCluster.GetNodeByID("any-id")
		if !IsNodeNotFoundedError(err) {
			t.Error("Empty cluster should return nodeNotFoundedError for GetNodeByID")
		}

		_, err = emptyCluster.GetNodeByIP("any-ip")
		if !IsNodeNotFoundedError(err) {
			t.Error("Empty cluster should return nodeNotFoundedError for GetNodeByIP")
		}

		_, err = emptyCluster.GetNodeByPodName("any-pod")
		if !IsNodeNotFoundedError(err) {
			t.Error("Empty cluster should return nodeNotFoundedError for GetNodeByPodName")
		}

		_, err = emptyCluster.GetNodesByFunc(func(node *Node) bool { return true })
		if !IsNodeNotFoundedError(err) {
			t.Error("Empty cluster should return nodeNotFoundedError for GetNodesByFunc")
		}
	})
}

// TestClusterActionsInfo tests the ClusterActionsInfo struct
func TestClusterActionsInfo(t *testing.T) {
	t.Run("ClusterActionsInfo initialization", func(t *testing.T) {
		actionInfo := ClusterActionsInfo{
			NbslotsToMigrate: 100,
		}

		if actionInfo.NbslotsToMigrate != 100 {
			t.Errorf("Expected NbslotsToMigrate to be 100, got %d", actionInfo.NbslotsToMigrate)
		}
	})

	t.Run("ClusterActionsInfo in Cluster", func(t *testing.T) {
		cluster := NewCluster("test", "test")
		cluster.ActionsInfo.NbslotsToMigrate = 256

		if cluster.ActionsInfo.NbslotsToMigrate != 256 {
			t.Error("Should be able to set and get ActionsInfo")
		}
	})
}
