package redisutil

import (
	"strings"
	"testing"
	"time"

	"github.com/freshworks/redis-cluster-operator/utils/testutil"
)

// TestNewNodeInfos tests the NewNodeInfos constructor
func TestNewNodeInfos(t *testing.T) {
	nodeInfos := NewNodeInfos()

	if nodeInfos == nil {
		t.Fatal("NewNodeInfos() should return non-nil NodeInfos")
	}

	if nodeInfos.Node == nil {
		t.Error("NewNodeInfos() should initialize Node")
	}

	if nodeInfos.Friends == nil {
		t.Error("NewNodeInfos() should initialize Friends slice")
	}

	if len(nodeInfos.Friends) != 0 {
		t.Errorf("NewNodeInfos() should initialize empty Friends slice, got length %d", len(nodeInfos.Friends))
	}
}

// TestNewClusterInfos tests the NewClusterInfos constructor
func TestNewClusterInfos(t *testing.T) {
	clusterInfos := NewClusterInfos()

	if clusterInfos == nil {
		t.Fatal("NewClusterInfos() should return non-nil ClusterInfos")
	}

	if clusterInfos.Infos == nil {
		t.Error("NewClusterInfos() should initialize Infos map")
	}

	if len(clusterInfos.Infos) != 0 {
		t.Errorf("NewClusterInfos() should initialize empty Infos map, got length %d", len(clusterInfos.Infos))
	}

	if clusterInfos.Status != ClusterInfosUnset {
		t.Errorf("NewClusterInfos() should initialize Status as %s, got %s", ClusterInfosUnset, clusterInfos.Status)
	}

	if clusterInfos.ClusterState != ClusterStatusUnknown {
		t.Errorf("NewClusterInfos() should initialize ClusterState as %s, got %s", ClusterStatusUnknown, clusterInfos.ClusterState)
	}

	if clusterInfos.ClusterNokReason == nil {
		t.Error("NewClusterInfos() should initialize ClusterNokReason map")
	}

	if len(clusterInfos.ClusterNokReason) != 0 {
		t.Errorf("NewClusterInfos() should initialize empty ClusterNokReason map, got length %d", len(clusterInfos.ClusterNokReason))
	}
}

// TestDecodeNodeStartTime tests the DecodeNodeStartTime function
func TestDecodeNodeStartTime(t *testing.T) {
	logger := testutil.NewTestLogger()

	tests := []struct {
		name        string
		input       string
		expectError bool
		description string
	}{
		{
			name: "Valid uptime parsing",
			input: `# Server
redis_version:6.2.7
uptime_in_seconds:3600
uptime_in_days:0`,
			expectError: false,
			description: "Should parse valid uptime correctly",
		},
		{
			name: "Valid uptime with extra fields",
			input: `# Server
redis_version:6.2.7
redis_git_sha1:00000000
redis_git_dirty:0
uptime_in_seconds:86400
uptime_in_days:1
arch_bits:64`,
			expectError: false,
			description: "Should parse uptime from multi-line input",
		},
		{
			name: "Missing uptime field",
			input: `# Server
redis_version:6.2.7
redis_git_sha1:00000000
arch_bits:64`,
			expectError: true,
			description: "Should error when uptime field is missing",
		},
		{
			name: "Invalid uptime value",
			input: `# Server
redis_version:6.2.7
uptime_in_seconds:invalid_number
uptime_in_days:0`,
			expectError: true,
			description: "Should error when uptime value is not a number",
		},
		{
			name:        "Empty input",
			input:       ``,
			expectError: true,
			description: "Should error with empty input",
		},
		{
			name: "Uptime with whitespace",
			input: `# Server
redis_version:6.2.7
uptime_in_seconds:  7200  
uptime_in_days:0`,
			expectError: false,
			description: "Should handle uptime with whitespace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startTime, err := DecodeNodeStartTime(&tt.input, logger)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none for %s", tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v for %s", err, tt.description)
				}

				// Verify the start time makes sense (should be in the past)
				if startTime.After(time.Now()) {
					t.Error("Start time should be in the past")
				}

				// For the first test case (3600 seconds = 1 hour), verify approximate correctness
				if tt.name == "Valid uptime parsing" {
					expectedStartTime := time.Now().Add(-1 * time.Hour)
					timeDiff := startTime.Sub(expectedStartTime).Abs()
					if timeDiff > 10*time.Second {
						t.Errorf("Start time calculation seems incorrect, got %v, expected around %v", startTime, expectedStartTime)
					}
				}
			}
		})
	}
}

// TestDecodeNodeInfos tests the DecodeNodeInfos function
func TestDecodeNodeInfos(t *testing.T) {
	logger := testutil.NewTestLogger()
	addr := "127.0.0.1:6379"

	tests := []struct {
		name          string
		input         string
		expectedNodes int
		expectOwnNode bool
		description   string
	}{
		{
			name: "Single master node",
			input: `07c37dfeb235213a872192d90877d0cd55635b91 127.0.0.1:6379@16379 myself,master - 0 1518023237000 1 connected 0-16383
`,
			expectedNodes: 1,
			expectOwnNode: true,
			description:   "Should parse single master node correctly",
		},
		{
			name: "Master with slave",
			input: `07c37dfeb235213a872192d90877d0cd55635b91 127.0.0.1:6379@16379 myself,master - 0 1518023237000 1 connected 0-16383
07c37dfeb235213a872192d90877d0cd55635b92 127.0.0.1:6380@16380 slave 07c37dfeb235213a872192d90877d0cd55635b91 0 1518023238000 1 connected
`,
			expectedNodes: 2,
			expectOwnNode: true,
			description:   "Should parse master-slave configuration",
		},
		{
			name:          "Empty input",
			input:         ``,
			expectedNodes: 0,
			expectOwnNode: false,
			description:   "Should handle empty input gracefully",
		},
		{
			name: "Invalid line format",
			input: `invalid line format
07c37dfeb235213a872192d90877d0cd55635b91 127.0.0.1:6379@16379 myself,master - 0 1518023237000 1 connected 0-16383
`,
			expectedNodes: 1,
			expectOwnNode: true,
			description:   "Should skip invalid lines and continue parsing",
		},
		{
			name: "Three node cluster",
			input: `07c37dfeb235213a872192d90877d0cd55635b91 127.0.0.1:6379@16379 myself,master - 0 1518023237000 1 connected 0-5460
07c37dfeb235213a872192d90877d0cd55635b92 127.0.0.1:6380@16380 master - 0 1518023238000 2 connected 5461-10922
07c37dfeb235213a872192d90877d0cd55635b93 127.0.0.1:6381@16381 master - 0 1518023239000 3 connected 10923-16383
`,
			expectedNodes: 3,
			expectOwnNode: true,
			description:   "Should parse three node cluster correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodeInfos := DecodeNodeInfos(&tt.input, addr, logger)

			if nodeInfos == nil {
				t.Fatalf("DecodeNodeInfos should return non-nil result for %s", tt.description)
			}

			if nodeInfos.Node == nil {
				t.Errorf("Node should be initialized for %s", tt.description)
			}

			if nodeInfos.Friends == nil {
				t.Errorf("Friends should be initialized for %s", tt.description)
			}

			totalNodes := len(nodeInfos.Friends)
			if tt.expectOwnNode && nodeInfos.Node != nil {
				totalNodes++ // Count the own node
			}

			if totalNodes != tt.expectedNodes {
				t.Errorf("Expected %d total nodes, got %d for %s", tt.expectedNodes, totalNodes, tt.description)
			}

			// Verify that if we expect an own node, it should be marked as ourselves
			if tt.expectOwnNode && tt.expectedNodes > 0 {
				if nodeInfos.Node == nil {
					t.Errorf("Expected own node to be set for %s", tt.description)
				} else {
					// The own node should have been identified from the "myself" flag
					if !strings.Contains(tt.input, "myself") {
						t.Errorf("Test input should contain 'myself' flag for %s", tt.description)
					}
				}
			}
		})
	}
}

// TestClusterInfos_GetNodes tests the GetNodes method
func TestClusterInfos_GetNodes(t *testing.T) {
	clusterInfos := NewClusterInfos()

	t.Run("Empty cluster", func(t *testing.T) {
		nodes := clusterInfos.GetNodes()
		if nodes == nil {
			t.Error("GetNodes should return non-nil slice")
		}
		if len(nodes) != 0 {
			t.Errorf("Empty cluster should return 0 nodes, got %d", len(nodes))
		}
	})

	t.Run("Cluster with nodes", func(t *testing.T) {
		// Add some node infos
		nodeInfo1 := NewNodeInfos()
		nodeInfo1.Node.ID = "node1"
		nodeInfo1.Node.IP = "127.0.0.1"
		nodeInfo1.Node.Port = "6379"

		nodeInfo2 := NewNodeInfos()
		nodeInfo2.Node.ID = "node2"
		nodeInfo2.Node.IP = "127.0.0.1"
		nodeInfo2.Node.Port = "6380"

		clusterInfos.Infos["127.0.0.1:6379"] = nodeInfo1
		clusterInfos.Infos["127.0.0.1:6380"] = nodeInfo2

		nodes := clusterInfos.GetNodes()
		if len(nodes) != 2 {
			t.Errorf("Expected 2 nodes, got %d", len(nodes))
		}

		// Verify that the nodes are the correct ones
		foundIds := make(map[string]bool)
		for _, node := range nodes {
			foundIds[node.ID] = true
		}

		if !foundIds["node1"] || !foundIds["node2"] {
			t.Error("Should find both node1 and node2 in results")
		}
	})
}

// TestClusterInfos_ComputeStatus tests the ComputeStatus method
func TestClusterInfos_ComputeStatus(t *testing.T) {
	logger := testutil.NewTestLogger()

	tests := []struct {
		name           string
		setupFunc      func() *ClusterInfos
		expectedStatus string
		expectedResult bool
		description    string
	}{
		{
			name: "Already computed status",
			setupFunc: func() *ClusterInfos {
				ci := NewClusterInfos()
				ci.Status = ClusterInfosConsistent
				return ci
			},
			expectedStatus: ClusterInfosConsistent,
			expectedResult: false,
			description:    "Should not recompute if status already set",
		},
		{
			name: "Empty cluster",
			setupFunc: func() *ClusterInfos {
				return NewClusterInfos()
			},
			expectedStatus: ClusterInfosConsistent,
			expectedResult: true,
			description:    "Empty cluster should be consistent",
		},
		{
			name: "Single node consistent",
			setupFunc: func() *ClusterInfos {
				ci := NewClusterInfos()
				nodeInfo := NewNodeInfos()
				nodeInfo.Node.ID = "node1"
				nodeInfo.Node.IP = "127.0.0.1"
				nodeInfo.Node.Port = "6379"
				nodeInfo.Node.Role = RedisMasterRole
				ci.Infos["127.0.0.1:6379"] = nodeInfo
				return ci
			},
			expectedStatus: ClusterInfosConsistent,
			expectedResult: true,
			description:    "Single node should be consistent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clusterInfos := tt.setupFunc()
			result := clusterInfos.ComputeStatus(logger)

			if result != tt.expectedResult {
				t.Errorf("Expected result %v, got %v for %s", tt.expectedResult, result, tt.description)
			}

			if clusterInfos.Status != tt.expectedStatus {
				t.Errorf("Expected status %s, got %s for %s", tt.expectedStatus, clusterInfos.Status, tt.description)
			}
		})
	}
}

// TestConfigSignature_String tests the ConfigSignature String method
func TestConfigSignature_String(t *testing.T) {
	tests := []struct {
		name           string
		signature      ConfigSignature
		expectContains []string
		description    string
	}{
		{
			name:           "Empty signature",
			signature:      ConfigSignature{},
			expectContains: []string{"map[", "]"},
			description:    "Should handle empty signature",
		},
		{
			name: "Single node signature",
			signature: ConfigSignature{
				"127.0.0.1:6379": SlotSlice{Slot(0), Slot(5460), Slot(16383)},
			},
			expectContains: []string{"map[", "127.0.0.1:6379", "]"},
			description:    "Should format single node correctly",
		},
		{
			name: "Multiple nodes signature",
			signature: ConfigSignature{
				"127.0.0.1:6379": SlotSlice{Slot(0), Slot(5460)},
				"127.0.0.1:6380": SlotSlice{Slot(5461), Slot(10922)},
			},
			expectContains: []string{"map[", "127.0.0.1:6379", "127.0.0.1:6380", "]"},
			description:    "Should format multiple nodes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.signature.String()

			for _, expected := range tt.expectContains {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected result to contain '%s', got '%s' for %s", expected, result, tt.description)
				}
			}
		})
	}
}

// TestSplitHostPort tests the splitHostPort function
func TestSplitHostPort(t *testing.T) {
	tests := []struct {
		name         string
		address      string
		expectedHost string
		expectedPort string
		expectError  bool
	}{
		{
			name:         "Valid IPv4 address",
			address:      "127.0.0.1:6379",
			expectedHost: "127.0.0.1",
			expectedPort: "6379",
			expectError:  false,
		},
		{
			name:         "Valid IPv6 address",
			address:      "[::1]:6379",
			expectedHost: "[::1]",
			expectedPort: "6379",
			expectError:  false,
		},
		{
			name:         "Hostname with port",
			address:      "redis.example.com:6379",
			expectedHost: "redis.example.com",
			expectedPort: "6379",
			expectError:  false,
		},
		{
			name:         "Complex hostname with multiple colons",
			address:      "redis-cluster:my:complex:name:6379",
			expectedHost: "redis-cluster:my:complex:name",
			expectedPort: "6379",
			expectError:  false,
		},
		{
			name:        "Address without port",
			address:     "127.0.0.1",
			expectError: true,
		},
		{
			name:        "Empty address",
			address:     "",
			expectError: true,
		},
		{
			name:         "Only colon",
			address:      ":",
			expectError:  false,
			expectedHost: "",
			expectedPort: "",
		},
		{
			name:         "Port only",
			address:      ":6379",
			expectedHost: "",
			expectedPort: "6379",
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, port, err := splitHostPort(tt.address)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for address '%s' but got none", tt.address)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for address '%s' but got: %v", tt.address, err)
				}

				if host != tt.expectedHost {
					t.Errorf("Expected host '%s', got '%s' for address '%s'", tt.expectedHost, host, tt.address)
				}

				if port != tt.expectedPort {
					t.Errorf("Expected port '%s', got '%s' for address '%s'", tt.expectedPort, port, tt.address)
				}
			}
		})
	}
}

// TestClusterInfos_Integration tests integration scenarios
func TestClusterInfos_Integration(t *testing.T) {
	logger := testutil.NewTestLogger()

	t.Run("Complete workflow", func(t *testing.T) {
		// Create cluster infos
		clusterInfos := NewClusterInfos()

		// Verify initial state
		if clusterInfos.Status != ClusterInfosUnset {
			t.Error("New cluster infos should have unset status")
		}

		// Add some node information
		nodeInfo := NewNodeInfos()
		nodeInfo.Node.ID = "master-node-1"
		nodeInfo.Node.IP = "10.0.0.1"
		nodeInfo.Node.Port = "6379"
		nodeInfo.Node.Role = RedisMasterRole

		clusterInfos.Infos["10.0.0.1:6379"] = nodeInfo

		// Test GetNodes
		nodes := clusterInfos.GetNodes()
		if len(nodes) != 1 {
			t.Errorf("Expected 1 node, got %d", len(nodes))
		}

		// Compute status
		result := clusterInfos.ComputeStatus(logger)
		if !result {
			t.Error("Single node cluster should be consistent")
		}

		if clusterInfos.Status != ClusterInfosConsistent {
			t.Errorf("Expected status %s, got %s", ClusterInfosConsistent, clusterInfos.Status)
		}

		// Test that recomputing doesn't change result
		result2 := clusterInfos.ComputeStatus(logger)
		if result2 {
			t.Error("Should not recompute status when already set")
		}
	})

	t.Run("Status constants", func(t *testing.T) {
		// Verify constants are defined correctly
		if ClusterInfosUnset == "" {
			t.Error("ClusterInfosUnset should not be empty")
		}
		if ClusterInfosPartial == "" {
			t.Error("ClusterInfosPartial should not be empty")
		}
		if ClusterInfosInconsistent == "" {
			t.Error("ClusterInfosInconsistent should not be empty")
		}
		if ClusterInfosConsistent == "" {
			t.Error("ClusterInfosConsistent should not be empty")
		}

		// Verify they are all different
		statuses := []string{ClusterInfosUnset, ClusterInfosPartial, ClusterInfosInconsistent, ClusterInfosConsistent}
		for i, s1 := range statuses {
			for j, s2 := range statuses {
				if i != j && s1 == s2 {
					t.Errorf("Status constants should be unique, found duplicate: %s", s1)
				}
			}
		}
	})
}
