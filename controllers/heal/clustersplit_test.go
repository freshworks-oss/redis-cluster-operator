package heal

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	redisv1alpha1 "github.com/freshworks/redis-cluster-operator/api/v1alpha1"
	config "github.com/freshworks/redis-cluster-operator/redisconfig"
	"github.com/freshworks/redis-cluster-operator/utils/redisutil"
)

// Mock for IAdminConnections - specific to clustersplit tests
type MockAdminConnections struct {
	mock.Mock
}

func (m *MockAdminConnections) ReplaceAll(ctx context.Context, addrs []string) {
	m.Called(ctx, addrs)
}

func (m *MockAdminConnections) GetAll() map[string]interface{} {
	args := m.Called()
	return args.Get(0).(map[string]interface{})
}

func (m *MockAdminConnections) Add(ctx context.Context, addr string) error {
	args := m.Called(ctx, addr)
	return args.Error(0)
}

func (m *MockAdminConnections) Remove(addr string) {
	m.Called(addr)
}

func (m *MockAdminConnections) Get(addr string) interface{} {
	args := m.Called(addr)
	return args.Get(0)
}

func (m *MockAdminConnections) ValidateResp(ctx context.Context, resp interface{}, addr string, errMessage string) error {
	args := m.Called(ctx, resp, addr, errMessage)
	return args.Error(0)
}

// Tests for buildClustersLists helper function
func Test_buildClustersLists(t *testing.T) {
	// Initialize redisutil.Nodes variables for better test readability and reusability
	ip1 := redisutil.Nodes{{IP: "ip1", Port: "1234"}}
	ip2 := redisutil.Nodes{{IP: "ip2", Port: "1234"}}
	ip56 := redisutil.Nodes{{IP: "ip5", Port: "1234"}, {IP: "ip6", Port: "1234"}}
	ip64 := redisutil.Nodes{{IP: "ip6", Port: "1234"}, {IP: "ip4", Port: "1234"}}
	ip54 := redisutil.Nodes{{IP: "ip5", Port: "1234"}, {IP: "ip4", Port: "1234"}}

	testCases := []struct {
		input  *redisutil.ClusterInfos
		output []cluster
	}{ //several partilly different cannot happen, so not tested
		{ // empty
			input:  &redisutil.ClusterInfos{Infos: map[string]*redisutil.NodeInfos{}, Status: redisutil.ClusterInfosConsistent},
			output: []cluster{},
		},
		{ // one node
			input:  &redisutil.ClusterInfos{Infos: map[string]*redisutil.NodeInfos{"ip1:1234": {Node: &redisutil.Node{IP: "ip1", Port: "1234"}, Friends: redisutil.Nodes{}}}, Status: redisutil.ClusterInfosConsistent},
			output: []cluster{{"ip1:1234"}},
		},
		{ // no discrepency
			input: &redisutil.ClusterInfos{
				Infos: map[string]*redisutil.NodeInfos{
					"ip1:1234": {Node: &redisutil.Node{IP: "ip1", Port: "1234"}, Friends: ip2},
					"ip2:1234": {Node: &redisutil.Node{IP: "ip2", Port: "1234"}, Friends: ip1},
				},
				Status: redisutil.ClusterInfosConsistent,
			},
			output: []cluster{{"ip1:1234", "ip2:1234"}},
		},
		{ // several decorelated
			input: &redisutil.ClusterInfos{
				Infos: map[string]*redisutil.NodeInfos{
					"ip1:1234": {Node: &redisutil.Node{IP: "ip1", Port: "1234"}, Friends: ip2},
					"ip2:1234": {Node: &redisutil.Node{IP: "ip2", Port: "1234"}, Friends: ip1},
					"ip3:1234": {Node: &redisutil.Node{IP: "ip3", Port: "1234"}, Friends: redisutil.Nodes{}},
					"ip4:1234": {Node: &redisutil.Node{IP: "ip4", Port: "1234"}, Friends: ip56},
					"ip5:1234": {Node: &redisutil.Node{IP: "ip5", Port: "1234"}, Friends: ip64},
					"ip6:1234": {Node: &redisutil.Node{IP: "ip6", Port: "1234"}, Friends: ip54},
				},
				Status: redisutil.ClusterInfosInconsistent,
			},
			output: []cluster{{"ip1:1234", "ip2:1234"}, {"ip3:1234"}, {"ip4:1234", "ip5:1234", "ip6:1234"}},
		},
		{ // empty ignored
			input: &redisutil.ClusterInfos{
				Infos: map[string]*redisutil.NodeInfos{
					"ip1:1234": {Node: &redisutil.Node{IP: "ip1", Port: "1234"}, Friends: ip2},
					"ip2:1234": {Node: &redisutil.Node{IP: "ip2", Port: "1234"}, Friends: ip1},
					"ip3:1234": nil,
				},
				Status: redisutil.ClusterInfosInconsistent,
			},
			output: []cluster{{"ip1:1234", "ip2:1234"}},
		},
	}

	for i, tc := range testCases {
		output := buildClustersLists(tc.input)
		// because we work with map, order might not be conserved
		if !compareClusters(output, tc.output) {
			t.Errorf("[Case %d] Unexpected result for buildClustersLists, expected %v, got %v", i, tc.output, output)
		}
	}
}

// Tests for FixClusterSplit method
func TestCheckAndHeal_FixClusterSplit_NoSplit(t *testing.T) {
	// Test when no cluster split is detected
	checkAndHeal := &CheckAndHeal{
		Logger:        logr.Discard(),
		PodControl:    &MockPodControl{},
		DryRun:        false,
		MetricsClient: &MockMetrics{},
	}

	cluster := &redisv1alpha1.DistributedRedisCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster", Namespace: "test-ns"},
	}

	// Create cluster infos with no split (single cluster)
	infos := &redisutil.ClusterInfos{
		Infos: map[string]*redisutil.NodeInfos{
			"ip1:1234": {
				Node: &redisutil.Node{IP: "ip1", Port: "1234"},
				Friends: redisutil.Nodes{
					{IP: "ip2", Port: "1234"},
				},
			},
			"ip2:1234": {
				Node: &redisutil.Node{IP: "ip2", Port: "1234"},
				Friends: redisutil.Nodes{
					{IP: "ip1", Port: "1234"},
				},
			},
		},
	}

	admin := &MockRedisAdmin{}
	redisConfig := &config.Redis{}

	actionDone, err := checkAndHeal.FixClusterSplit(cluster, infos, admin, redisConfig)

	assert.NoError(t, err)
	assert.False(t, actionDone) // No action needed

	admin.AssertExpectations(t)
}

func TestCheckAndHeal_FixClusterSplit_DryRunMode(t *testing.T) {
	// Test when cluster split is detected but DryRun is enabled
	checkAndHeal := &CheckAndHeal{
		Logger:        logr.Discard(),
		PodControl:    &MockPodControl{},
		DryRun:        true, // DryRun enabled
		MetricsClient: &MockMetrics{},
	}

	cluster := &redisv1alpha1.DistributedRedisCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster", Namespace: "test-ns"},
	}

	// Create cluster infos with split (multiple clusters)
	infos := &redisutil.ClusterInfos{
		Infos: map[string]*redisutil.NodeInfos{
			"ip1:1234": {
				Node:    &redisutil.Node{IP: "ip1", Port: "1234"},
				Friends: redisutil.Nodes{}, // Isolated node
			},
			"ip2:1234": {
				Node:    &redisutil.Node{IP: "ip2", Port: "1234"},
				Friends: redisutil.Nodes{}, // Another isolated node
			},
		},
	}

	admin := &MockRedisAdmin{}
	redisConfig := &config.Redis{}

	actionDone, err := checkAndHeal.FixClusterSplit(cluster, infos, admin, redisConfig)

	assert.NoError(t, err)
	assert.True(t, actionDone) // Action detected but not performed due to DryRun

	// No actual fixing should be called in DryRun mode
	admin.AssertExpectations(t)
}

// Tests for splitMainCluster helper function
func TestSplitMainCluster(t *testing.T) {
	tests := []struct {
		name           string
		input          []cluster
		expectedMain   cluster
		expectedBadLen int
	}{
		{
			name:           "Empty clusters",
			input:          []cluster{},
			expectedMain:   cluster{},
			expectedBadLen: 0,
		},
		{
			name:           "Single cluster",
			input:          []cluster{{"ip1:1234", "ip2:1234"}},
			expectedMain:   cluster{"ip1:1234", "ip2:1234"},
			expectedBadLen: 0,
		},
		{
			name: "Multiple clusters - different sizes",
			input: []cluster{
				{"ip1:1234"},
				{"ip2:1234", "ip3:1234", "ip4:1234"}, // Largest
				{"ip5:1234", "ip6:1234"},
			},
			expectedMain:   cluster{"ip2:1234", "ip3:1234", "ip4:1234"},
			expectedBadLen: 2,
		},
		{
			name: "Multiple clusters - same size (first wins)",
			input: []cluster{
				{"ip1:1234", "ip2:1234"}, // First of equal size
				{"ip3:1234", "ip4:1234"}, // Same size
			},
			expectedMain:   cluster{"ip1:1234", "ip2:1234"},
			expectedBadLen: 1,
		},
		{
			name: "All clusters same size - first wins",
			input: []cluster{
				{"ip1:1234"},
				{"ip2:1234"},
				{"ip3:1234"},
			},
			expectedMain:   cluster{"ip1:1234"},
			expectedBadLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			main, bad := splitMainCluster(tt.input)

			assert.Equal(t, tt.expectedMain, main)
			assert.Len(t, bad, tt.expectedBadLen)

			// Verify that main cluster is not in bad clusters
			for _, badCluster := range bad {
				assert.NotEqual(t, main, badCluster)
			}
		})
	}
}

// Helper functions for cluster comparison
func compareClusters(c1, c2 []cluster) bool {
	if len(c1) != len(c2) {
		return false
	}

	for _, c1elem := range c2 {
		found := false
		for _, c2elem := range c1 {
			if compareCluster(c1elem, c2elem) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func compareCluster(c1, c2 cluster) bool {
	if len(c1) != len(c2) {
		return false
	}
	for _, c1elem := range c2 {
		found := false
		for _, c2elem := range c1 {
			if c1elem == c2elem {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// Note: Testing the full reassignClusters logic is complex because it creates
// new redisutil.NewAdmin instances which are hard to mock. The tests above cover
// the main entry points and helper functions. The reassignClusters method would
// require integration tests or more sophisticated mocking framework.
