package manager

import (
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	redisv1alpha1 "github.com/freshworks/redis-cluster-operator/api/v1alpha1"
	"github.com/freshworks/redis-cluster-operator/controllers/heal"
)

func TestNewHealer_Simple(t *testing.T) {
	// Test creating a new healer - this is the main function we can test
	checkAndHeal := &heal.CheckAndHeal{
		Logger: logr.Discard(),
	}

	healer := NewHealer(checkAndHeal)

	assert.NotNil(t, healer)
	assert.Implements(t, (*IHeal)(nil), healer)

	// Verify it's the correct type
	realHealer, ok := healer.(*realHeal)
	assert.True(t, ok)
	assert.Equal(t, checkAndHeal, realHealer.CheckAndHeal)
}

func TestNewHealer_WithNilCheckAndHeal(t *testing.T) {
	// Test creating healer with nil (should still work)
	healer := NewHealer(nil)

	assert.NotNil(t, healer)
	assert.Implements(t, (*IHeal)(nil), healer)

	// Verify it's the correct type
	realHealer, ok := healer.(*realHeal)
	assert.True(t, ok)
	assert.Nil(t, realHealer.CheckAndHeal)
}

func TestRealHeal_ImplementsInterface(t *testing.T) {
	// Test that realHeal implements the IHeal interface
	var _ IHeal = &realHeal{}

	// Test interface methods exist
	checkAndHeal := &heal.CheckAndHeal{
		Logger: logr.Discard(),
	}

	healer := NewHealer(checkAndHeal)

	// Verify interface methods are callable (though we won't call them with real data)
	assert.NotNil(t, healer.Heal)
	assert.NotNil(t, healer.FixTerminatingPods)
}

func TestRealHeal_StructureAndEmbedding(t *testing.T) {
	// Test the structure and embedding behavior
	checkAndHeal := &heal.CheckAndHeal{
		Logger: logr.Discard(),
		DryRun: true, // Use DryRun to avoid actual operations
	}

	healer := &realHeal{CheckAndHeal: checkAndHeal}

	// Verify embedding works correctly
	assert.Equal(t, checkAndHeal, healer.CheckAndHeal)
	assert.Equal(t, checkAndHeal.Logger, healer.CheckAndHeal.Logger)
	assert.Equal(t, checkAndHeal.DryRun, healer.CheckAndHeal.DryRun)
}

func TestRealHeal_BasicFunctionality(t *testing.T) {
	// Test basic functionality without actual Redis operations
	checkAndHeal := &heal.CheckAndHeal{
		Logger: logr.Discard(),
		DryRun: true, // Use DryRun to avoid actual operations
	}

	healer := NewHealer(checkAndHeal)

	// Test that we can create a cluster object for testing
	cluster := &redisv1alpha1.DistributedRedisCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster", Namespace: "test-ns"},
		Spec: redisv1alpha1.DistributedRedisClusterSpec{
			MasterSize:      3,
			ClusterReplicas: 1,
		},
	}

	// Test FixTerminatingPods with zero duration (should return immediately)
	actionDone, err := healer.FixTerminatingPods(cluster, time.Duration(0))
	assert.NoError(t, err)
	assert.False(t, actionDone)
}

func TestIHeal_InterfaceDefinition(t *testing.T) {
	// Test the IHeal interface definition
	tests := []struct {
		name         string
		methodName   string
		methodExists bool
	}{
		{
			name:         "Heal method exists",
			methodName:   "Heal",
			methodExists: true,
		},
		{
			name:         "FixTerminatingPods method exists",
			methodName:   "FixTerminatingPods",
			methodExists: true,
		},
	}

	checkAndHeal := &heal.CheckAndHeal{
		Logger: logr.Discard(),
	}
	healer := NewHealer(checkAndHeal)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test verifies the interface methods exist
			switch tt.methodName {
			case "Heal":
				assert.NotNil(t, healer.Heal)
			case "FixTerminatingPods":
				assert.NotNil(t, healer.FixTerminatingPods)
			}
		})
	}
}
