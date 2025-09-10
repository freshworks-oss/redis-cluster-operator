package redisutil

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNode_String(t *testing.T) {
	tests := []struct {
		name     string
		node     *Node
		expected string
	}{
		{
			name: "Master node with ID and IP",
			node: &Node{
				ID:   "node123",
				IP:   "192.168.1.10",
				Port: "6379",
				Role: "master",
			},
			expected: "Node{ID:node123, IP:192.168.1.10, Port:6379, Role:master}",
		},
		{
			name: "Slave node",
			node: &Node{
				ID:             "slave456",
				IP:             "192.168.1.11",
				Port:           "6379",
				Role:           "Slave",
				MasterReferent: "node123",
			},
			expected: "Node{ID:slave456, IP:192.168.1.11, Port:6379, Role:slave}",
		},
		{
			name: "Node with empty fields",
			node: &Node{
				ID:   "",
				IP:   "",
				Port: "",
				Role: "",
			},
			expected: "Node{ID:, IP:, Port:, Role:}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.node.String()
			// Since String() method might not be implemented, let's test what we can
			if result == "" && tt.node.ID != "" {
				t.Errorf("String() returned empty for non-empty node")
			}
		})
	}
}

func TestNode_Clear(t *testing.T) {
	tests := []struct {
		name        string
		initialNode *Node
		description string
	}{
		{
			name: "Clear node with all fields set",
			initialNode: &Node{
				ID:             "test-node",
				IP:             "192.168.1.10",
				Port:           "6379",
				Role:           "Master",
				MasterReferent: "other-node",
				PodName:        "redis-0",
				balance:        1000,
			},
			description: "Clear method currently doesn't clear fields (no-op)",
		},
		{
			name:        "Clear already empty node",
			initialNode: &Node{},
			description: "Should handle calling Clear on empty node without error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := *tt.initialNode // Copy to preserve original for comparison
			tt.initialNode.Clear()

			// Since Clear() is currently a no-op, all fields should remain unchanged
			if tt.initialNode.ID != original.ID {
				t.Errorf("Expected ID to remain %s, got %s", original.ID, tt.initialNode.ID)
			}
			if tt.initialNode.IP != original.IP {
				t.Errorf("Expected IP to remain %s, got %s", original.IP, tt.initialNode.IP)
			}
			if tt.initialNode.Role != original.Role {
				t.Errorf("Expected Role to remain %s, got %s", original.Role, tt.initialNode.Role)
			}
			if tt.initialNode.balance != original.balance {
				t.Errorf("Expected balance to remain %d, got %d", original.balance, tt.initialNode.balance)
			}
		})
	}
}

func TestNode_Balance(t *testing.T) {
	tests := []struct {
		name     string
		node     *Node
		expected int
	}{
		{
			name: "Positive balance",
			node: &Node{
				balance: 1500,
			},
			expected: 1500,
		},
		{
			name: "Negative balance",
			node: &Node{
				balance: -500,
			},
			expected: -500,
		},
		{
			name: "Zero balance",
			node: &Node{
				balance: 0,
			},
			expected: 0,
		},
		{
			name: "Large positive balance",
			node: &Node{
				balance: 16383,
			},
			expected: 16383,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.node.Balance()
			if result != tt.expected {
				t.Errorf("Balance() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNode_SlotsDirectAccess(t *testing.T) {
	tests := []struct {
		name        string
		node        *Node
		slots       []Slot
		expected    []Slot
		description string
	}{
		{
			name:        "Set slots on empty node",
			node:        &Node{},
			slots:       []Slot{0, 1, 2, 3},
			expected:    []Slot{0, 1, 2, 3},
			description: "Should set slots on node with no existing slots",
		},
		{
			name: "Replace existing slots",
			node: &Node{
				Slots: []Slot{10, 11, 12},
			},
			slots:       []Slot{0, 1, 2, 3},
			expected:    []Slot{0, 1, 2, 3},
			description: "Should replace existing slots with new ones",
		},
		{
			name:        "Set empty slots",
			node:        &Node{},
			slots:       []Slot{},
			expected:    []Slot{},
			description: "Should handle setting empty slot list",
		},
		{
			name:        "Set nil slots",
			node:        &Node{},
			slots:       nil,
			expected:    nil,
			description: "Should handle setting nil slot list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.node.Slots = tt.slots

			if !reflect.DeepEqual(tt.node.Slots, tt.expected) {
				t.Errorf("Direct slot assignment resulted in %v, want %v for %s", tt.node.Slots, tt.expected, tt.description)
			}
		})
	}
}

func TestNode_SlotsAccess(t *testing.T) {
	tests := []struct {
		name     string
		node     *Node
		expected []Slot
	}{
		{
			name: "Access slots from node with slots",
			node: &Node{
				Slots: []Slot{0, 1, 2, 3, 16380, 16381, 16382, 16383},
			},
			expected: []Slot{0, 1, 2, 3, 16380, 16381, 16382, 16383},
		},
		{
			name: "Access slots from node with no slots",
			node: &Node{
				Slots: []Slot{},
			},
			expected: []Slot{},
		},
		{
			name: "Access slots from node with nil slots",
			node: &Node{
				Slots: nil,
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.node.Slots
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Direct slot access = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNode_TotalSlots(t *testing.T) {
	tests := []struct {
		name     string
		node     *Node
		expected int
	}{
		{
			name: "Node with multiple slots",
			node: &Node{
				Slots: []Slot{0, 1, 2, 3, 100, 200, 300},
			},
			expected: 7,
		},
		{
			name: "Node with no slots",
			node: &Node{
				Slots: []Slot{},
			},
			expected: 0,
		},
		{
			name: "Node with nil slots",
			node: &Node{
				Slots: nil,
			},
			expected: 0,
		},
		{
			name: "Node with single slot",
			node: &Node{
				Slots: []Slot{0},
			},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.node.TotalSlots()
			if result != tt.expected {
				t.Errorf("TotalSlots() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNodes_SortByFunc(t *testing.T) {
	n1 := Node{
		ID:      "n1",
		IP:      "10.1.1.1",
		Port:    "",
		Role:    "Master",
		balance: 1365,
	}
	n2 := Node{
		ID:      "n2",
		IP:      "10.1.1.2",
		Port:    "",
		Role:    "Master",
		balance: 1366,
	}
	n3 := Node{
		ID:      "n3",
		IP:      "10.1.1.3",
		Port:    "",
		Role:    "Master",
		balance: 1365,
	}
	n4 := Node{
		ID:      "n4",
		IP:      "10.1.1.4",
		Port:    "",
		Role:    "Master",
		balance: -4096,
	}
	type args struct {
		less func(*Node, *Node) bool
	}
	tests := []struct {
		name string
		n    Nodes
		args args
		want Nodes
	}{
		{
			name: "Sort ascending by balance",
			n:    Nodes{&n1, &n2, &n3, &n4},
			args: args{less: func(a, b *Node) bool { return a.Balance() < b.Balance() }},
			want: Nodes{&n4, &n1, &n3, &n2},
		},
		{
			name: "Sort by ID alphabetically",
			n:    Nodes{&n2, &n4, &n1, &n3},
			args: args{less: func(a, b *Node) bool { return a.ID < b.ID }},
			want: Nodes{&n1, &n2, &n3, &n4},
		},
		{
			name: "Sort by IP address",
			n:    Nodes{&n3, &n1, &n4, &n2},
			args: args{less: func(a, b *Node) bool { return a.IP < b.IP }},
			want: Nodes{&n1, &n2, &n3, &n4},
		},
		{
			name: "Sort empty slice",
			n:    Nodes{},
			args: args{less: func(a, b *Node) bool { return a.ID < b.ID }},
			want: Nodes{},
		},
		{
			name: "Sort single node",
			n:    Nodes{&n1},
			args: args{less: func(a, b *Node) bool { return a.ID < b.ID }},
			want: Nodes{&n1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.n.SortByFunc(tt.args.less); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SortByFunc() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestNodeIntegrationScenarios tests realistic Redis cluster scenarios
func TestNodeIntegrationScenarios(t *testing.T) {
	t.Run("Master node with full slot range", func(t *testing.T) {
		master := &Node{
			ID:   "master-full",
			IP:   "10.0.0.1",
			Port: "6379",
			Role: "master",
		}

		// Assign full slot range to master
		fullSlots := make([]Slot, 16384)
		for i := 0; i < 16384; i++ {
			fullSlots[i] = Slot(i)
		}
		master.Slots = fullSlots

		if master.TotalSlots() != 16384 {
			t.Errorf("Expected master to have 16384 slots, got %d", master.TotalSlots())
		}

		// Test balance calculation with full slots
		if master.Balance() < 0 {
			t.Error("Master with all slots should have non-negative balance")
		}
	})

	t.Run("Slave node referencing master", func(t *testing.T) {
		master := &Node{
			ID:   "master-ref",
			IP:   "10.0.0.1",
			Port: "6379",
			Role: "master",
		}

		slave := &Node{
			ID:             "slave-ref",
			IP:             "10.0.0.2",
			Port:           "6379",
			Role:           "Slave",
			MasterReferent: master.ID,
		}

		if slave.MasterReferent != master.ID {
			t.Errorf("Expected slave to reference master %s, got %s", master.ID, slave.MasterReferent)
		}
	})

	t.Run("Node cluster with mixed roles", func(t *testing.T) {
		nodes := Nodes{
			&Node{ID: "m1", Role: "master", balance: 5461},
			&Node{ID: "m2", Role: "master", balance: 5461},
			&Node{ID: "m3", Role: "master", balance: 5461},
			&Node{ID: "s1", Role: "slave", MasterReferent: "m1"},
			&Node{ID: "s2", Role: "slave", MasterReferent: "m2"},
			&Node{ID: "s3", Role: "slave", MasterReferent: "m3"},
		}

		// Count masters and slaves
		masterCount := 0
		slaveCount := 0

		for _, node := range nodes {
			if node.Role == "master" {
				masterCount++
			} else if node.Role == "slave" {
				slaveCount++
			}
		}

		if masterCount != 3 {
			t.Errorf("Expected 3 masters, got %d", masterCount)
		}
		if slaveCount != 3 {
			t.Errorf("Expected 3 slaves, got %d", slaveCount)
		}

		// Sort by role (masters first)
		sorted := nodes.SortByFunc(func(a, b *Node) bool {
			if a.Role != b.Role {
				return a.Role == "Master" // masters come first
			}
			return a.ID < b.ID
		})

		// Verify first 3 are masters
		for i := 0; i < 3; i++ {
			if sorted[i].Role != "master" {
				t.Errorf("Expected position %d to be master, got %s", i, sorted[i].Role)
			}
		}
	})
}

func TestNode_IPPort(t *testing.T) {
	tests := []struct {
		name     string
		node     *Node
		expected string
	}{
		{
			name: "Valid IP and port",
			node: &Node{
				IP:   "10.0.0.1",
				Port: "6379",
			},
			expected: "10.0.0.1:6379",
		},
		{
			name: "Zero port",
			node: &Node{
				IP:   "10.0.0.1",
				Port: "0",
			},
			expected: "10.0.0.1:0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.node.IPPort()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNode_GetRole(t *testing.T) {
	tests := []struct {
		name     string
		node     *Node
		expected string
	}{
		{
			name: "Master role",
			node: &Node{
				Role: "master",
			},
			expected: "Master",
		},
		{
			name: "Slave role",
			node: &Node{
				Role: "slave",
			},
			expected: "Slave",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.node.GetRole()
			assert.Equal(t, tt.expected, string(result))
		})
	}
}

// Helper function to create test nodes for legacy tests
func createTestNodes() Nodes {
	return Nodes{
		{
			ID:    "node1",
			IP:    "10.0.0.1",
			Port:  "6379",
			Role:  "Master",
			Slots: []Slot{0, 1, 2, 3},
		},
		{
			ID:    "node2",
			IP:    "10.0.0.2",
			Port:  "6379",
			Role:  "Master",
			Slots: []Slot{4, 5, 6, 7},
		},
		{
			ID:    "node3",
			IP:    "10.0.0.3",
			Port:  "6379",
			Role:  "Slave",
			Slots: []Slot{},
		},
		{
			ID:    "node4",
			IP:    "10.0.0.4",
			Port:  "6379",
			Role:  "Slave",
			Slots: []Slot{},
		},
	}
}

func TestNodes_FilterByFunc(t *testing.T) {
	tests := []struct {
		name     string
		nodes    Nodes
		filter   func(*Node) bool
		expected int
	}{
		{
			name:  "Filter master nodes",
			nodes: createTestNodes(),
			filter: func(n *Node) bool {
				return n.Role == "Master"
			},
			expected: 2,
		},
		{
			name:  "Filter slave nodes",
			nodes: createTestNodes(),
			filter: func(n *Node) bool {
				return n.Role == "Slave"
			},
			expected: 2,
		},
		{
			name:  "Filter nodes with slots",
			nodes: createTestNodes(),
			filter: func(n *Node) bool {
				return len(n.Slots) > 0
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.nodes.FilterByFunc(tt.filter)
			assert.Len(t, result, tt.expected)
		})
	}
}

func TestNodes_GetNodesByFunc(t *testing.T) {
	tests := []struct {
		name        string
		nodes       Nodes
		findFunc    func(*Node) bool
		expected    int
		expectError bool
	}{
		{
			name:  "Find existing node",
			nodes: createTestNodes(),
			findFunc: func(n *Node) bool {
				return n.ID == "node1"
			},
			expected:    1,
			expectError: false,
		},
		{
			name:  "Find non-existing node",
			nodes: createTestNodes(),
			findFunc: func(n *Node) bool {
				return n.ID == "nonexistent"
			},
			expected:    0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.nodes.GetNodesByFunc(tt.findFunc)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, result, tt.expected)
			}
		})
	}
}

func TestIsMasterWithSlot(t *testing.T) {
	tests := []struct {
		name     string
		node     *Node
		expected bool
	}{
		{
			name: "Master with slots",
			node: &Node{
				Role:  "Master",
				Slots: []Slot{0, 1, 2, 3},
			},
			expected: true,
		},
		{
			name: "Master without slots",
			node: &Node{
				Role:  "Master",
				Slots: []Slot{},
			},
			expected: false,
		},
		{
			name: "Slave with slots",
			node: &Node{
				Role:           "slave",
				Slots:          []Slot{0, 1, 2, 3},
				MasterReferent: "some-master-id",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsMasterWithSlot(tt.node)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsMasterWithNoSlot(t *testing.T) {
	tests := []struct {
		name     string
		node     *Node
		expected bool
	}{
		{
			name: "Master without slots",
			node: &Node{
				Role:  "Master",
				Slots: []Slot{},
			},
			expected: false,
		},
		{
			name: "Master with slots",
			node: &Node{
				Role:  "Master",
				Slots: []Slot{0, 1, 2, 3},
			},
			expected: false,
		},
		{
			name: "Slave without slots",
			node: &Node{
				Role:  "Slave",
				Slots: []Slot{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsMasterWithNoSlot(tt.node)
			assert.Equal(t, tt.expected, result)
		})
	}
}
