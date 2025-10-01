package k8sutil

import (
	"testing"

	"github.com/freshworks/redis-cluster-operator/utils/testutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestNewNodeController(t *testing.T) {
	fakeClient := fake.NewClientBuilder().Build()
	metrics := &testutil.MockMetrics{}

	controller := NewNodeController(fakeClient, metrics)
	if controller == nil {
		t.Error("NewNodeController() returned nil")
	}

	// Verify it implements INodeControl interface
	_, ok := controller.(INodeControl)
	if !ok {
		t.Error("NewNodeController() does not implement INodeControl interface")
	}
}

func TestNodeController_GetNode(t *testing.T) {
	existingNode := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "existing-node",
			Labels: map[string]string{
				"topology.kubernetes.io/zone": "us-west-2a",
			},
		},
		Spec: corev1.NodeSpec{},
	}

	tests := []struct {
		name        string
		nodeName    string
		expectError bool
		expectFound bool
	}{
		{
			name:        "Get existing node successfully",
			nodeName:    "existing-node",
			expectError: false,
			expectFound: true,
		},
		{
			name:        "Get non-existing node returns error",
			nodeName:    "non-existing-node",
			expectError: true,
			expectFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fakeClient client.Client
			if tt.expectFound {
				fakeClient = fake.NewClientBuilder().WithObjects(existingNode).Build()
			} else {
				fakeClient = fake.NewClientBuilder().Build()
			}

			controller := NewNodeController(fakeClient, &testutil.MockMetrics{})

			node, err := controller.GetNode(tt.nodeName)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if tt.expectFound && node == nil {
				t.Error("Expected node to be found but got nil")
			}
			if tt.expectFound && node.Name != tt.nodeName {
				t.Errorf("Expected node name %s, got %s", tt.nodeName, node.Name)
			}
		})
	}
}

func TestNodeController_GetZoneLabel(t *testing.T) {
	tests := []struct {
		name         string
		node         *corev1.Node
		expectedZone string
	}{
		{
			name: "Get zone from node with zone label",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node-with-zone",
					Labels: map[string]string{
						"topology.kubernetes.io/zone": "us-west-2a",
						"other-label":                 "other-value",
					},
				},
			},
			expectedZone: "us-west-2a",
		},
		{
			name: "Return unknown for node without zone label",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node-without-zone",
					Labels: map[string]string{
						"other-label": "other-value",
					},
				},
			},
			expectedZone: "unknown",
		},
		{
			name: "Return unknown for node with no labels",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node-no-labels",
				},
			},
			expectedZone: "unknown",
		},
		{
			name:         "Return unknown for nil node",
			node:         nil,
			expectedZone: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().Build()
			controller := NewNodeController(fakeClient, &testutil.MockMetrics{})

			zone := controller.GetZoneLabel(tt.node)

			if zone != tt.expectedZone {
				t.Errorf("Expected zone %s, got %s", tt.expectedZone, zone)
			}
		})
	}
}

func TestNodeController_CheckZoneAvailable(t *testing.T) {
	nodeInZone := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node-in-zone",
			Labels: map[string]string{
				"topology.kubernetes.io/zone": "us-west-2a",
			},
		},
	}

	nodeInDifferentZone := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node-in-different-zone",
			Labels: map[string]string{
				"topology.kubernetes.io/zone": "us-west-2b",
			},
		},
	}

	tests := []struct {
		name         string
		zoneName     string
		nodes        []client.Object
		expectError  bool
		expectResult bool
	}{
		{
			name:         "Zone is available with existing node",
			zoneName:     "us-west-2a",
			nodes:        []client.Object{nodeInZone, nodeInDifferentZone},
			expectError:  false,
			expectResult: true,
		},
		{
			name:         "Zone is not available with no matching nodes",
			zoneName:     "us-east-1a",
			nodes:        []client.Object{nodeInZone, nodeInDifferentZone},
			expectError:  false,
			expectResult: false,
		},
		{
			name:         "Zone is not available with no nodes at all",
			zoneName:     "us-west-2a",
			nodes:        []client.Object{},
			expectError:  false,
			expectResult: false,
		},
		{
			name:         "Return error for empty zone name",
			zoneName:     "",
			nodes:        []client.Object{nodeInZone},
			expectError:  true,
			expectResult: false,
		},
		{
			name:         "Return false for zone with only whitespace",
			zoneName:     "   ",
			nodes:        []client.Object{nodeInZone},
			expectError:  false,
			expectResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().WithObjects(tt.nodes...).Build()
			controller := NewNodeController(fakeClient, &testutil.MockMetrics{})

			result, err := controller.CheckZoneAvailable(tt.zoneName)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if result != tt.expectResult {
				t.Errorf("Expected result %v, got %v", tt.expectResult, result)
			}
		})
	}
}
