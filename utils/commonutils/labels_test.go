package utils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeLabels(t *testing.T) {
	tests := []struct {
		name           string
		inputLabels    []map[string]string
		expectedResult map[string]string
		description    string
	}{
		{
			name:           "No labels",
			inputLabels:    []map[string]string{},
			expectedResult: map[string]string{},
			description:    "Should return empty map when no labels provided",
		},
		{
			name:           "Single label map",
			inputLabels:    []map[string]string{{"app": "redis", "tier": "cache"}},
			expectedResult: map[string]string{"app": "redis", "tier": "cache"},
			description:    "Should return single label map unchanged",
		},
		{
			name: "Two label maps",
			inputLabels: []map[string]string{
				{"app": "redis", "tier": "cache"},
				{"version": "6.2", "environment": "production"},
			},
			expectedResult: map[string]string{
				"app":         "redis",
				"tier":        "cache",
				"version":     "6.2",
				"environment": "production",
			},
			description: "Should merge two label maps",
		},
		{
			name: "Multiple label maps",
			inputLabels: []map[string]string{
				{"app": "redis"},
				{"tier": "cache"},
				{"version": "6.2"},
				{"environment": "production"},
			},
			expectedResult: map[string]string{
				"app":         "redis",
				"tier":        "cache",
				"version":     "6.2",
				"environment": "production",
			},
			description: "Should merge multiple label maps",
		},
		{
			name: "Overlapping keys - later wins",
			inputLabels: []map[string]string{
				{"app": "redis", "version": "5.0"},
				{"version": "6.2", "tier": "cache"},
				{"version": "7.0", "environment": "production"},
			},
			expectedResult: map[string]string{
				"app":         "redis",
				"version":     "7.0", // Last value wins
				"tier":        "cache",
				"environment": "production",
			},
			description: "Should handle overlapping keys with later values taking precedence",
		},
		{
			name: "Empty maps in sequence",
			inputLabels: []map[string]string{
				{},
				{"app": "redis"},
				{},
				{"tier": "cache"},
				{},
			},
			expectedResult: map[string]string{
				"app":  "redis",
				"tier": "cache",
			},
			description: "Should handle empty maps in sequence",
		},
		{
			name: "Nil maps in sequence",
			inputLabels: []map[string]string{
				nil,
				{"app": "redis"},
				nil,
				{"tier": "cache"},
				nil,
			},
			expectedResult: map[string]string{
				"app":  "redis",
				"tier": "cache",
			},
			description: "Should handle nil maps in sequence",
		},
		{
			name: "Mixed nil and empty maps",
			inputLabels: []map[string]string{
				nil,
				{},
				{"app": "redis"},
				nil,
				{},
				{"tier": "cache"},
			},
			expectedResult: map[string]string{
				"app":  "redis",
				"tier": "cache",
			},
			description: "Should handle mixed nil and empty maps",
		},
		{
			name: "Complex label values",
			inputLabels: []map[string]string{
				{"app": "redis-cluster", "tier": "cache-layer"},
				{"version": "6.2.14-alpine", "environment": "prod-cluster-1"},
				{"region": "us-west-2", "zone": "us-west-2a"},
			},
			expectedResult: map[string]string{
				"app":         "redis-cluster",
				"tier":        "cache-layer",
				"version":     "6.2.14-alpine",
				"environment": "prod-cluster-1",
				"region":      "us-west-2",
				"zone":        "us-west-2a",
			},
			description: "Should handle complex label values with hyphens and dots",
		},
		{
			name: "Special characters in values",
			inputLabels: []map[string]string{
				{"app": "redis/cluster", "tier": "cache:layer"},
				{"version": "6.2.14_alpine", "environment": "prod.cluster.1"},
			},
			expectedResult: map[string]string{
				"app":         "redis/cluster",
				"tier":        "cache:layer",
				"version":     "6.2.14_alpine",
				"environment": "prod.cluster.1",
			},
			description: "Should handle special characters in label values",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergeLabels(tt.inputLabels...)

			assert.Equal(t, len(tt.expectedResult), len(result),
				"Expected %d labels, got %d", len(tt.expectedResult), len(result))

			for expectedKey, expectedValue := range tt.expectedResult {
				assert.Equal(t, expectedValue, result[expectedKey],
					"Expected key %s to have value %s, got %s", expectedKey, expectedValue, result[expectedKey])
			}

			// Verify no extra keys
			for resultKey := range result {
				assert.Contains(t, tt.expectedResult, resultKey,
					"Unexpected key %s in result", resultKey)
			}
		})
	}
}

func TestMergeLabelsEdgeCases(t *testing.T) {
	t.Run("All nil maps", func(t *testing.T) {
		result := MergeLabels(nil, nil, nil)
		assert.Empty(t, result, "Should return empty map when all inputs are nil")
	})

	t.Run("All empty maps", func(t *testing.T) {
		result := MergeLabels(map[string]string{}, map[string]string{}, map[string]string{})
		assert.Empty(t, result, "Should return empty map when all inputs are empty")
	})

	t.Run("Mixed nil and empty with actual labels", func(t *testing.T) {
		appLabels := map[string]string{"app": "redis"}
		tierLabels := map[string]string{"tier": "cache"}
		result := MergeLabels(nil, map[string]string{}, appLabels, nil, tierLabels)
		expected := map[string]string{"app": "redis", "tier": "cache"}
		assert.Equal(t, expected, result)
	})

	t.Run("Very large number of label maps", func(t *testing.T) {
		// Create many label maps
		labelMaps := make([]map[string]string, 1000)
		expected := make(map[string]string)

		for i := 0; i < 1000; i++ {
			key := fmt.Sprintf("key%d", i)
			value := fmt.Sprintf("value%d", i)
			labelMaps[i] = map[string]string{key: value}
			expected[key] = value
		}

		result := MergeLabels(labelMaps...)
		assert.Equal(t, expected, result)
	})
}

func TestMergeLabelsImmutability(t *testing.T) {
	t.Run("Original maps should not be modified", func(t *testing.T) {
		original1 := map[string]string{"app": "redis", "tier": "cache"}
		original2 := map[string]string{"version": "6.2"}

		// Create copies for comparison
		original1Copy := make(map[string]string)
		original2Copy := make(map[string]string)
		for k, v := range original1 {
			original1Copy[k] = v
		}
		for k, v := range original2 {
			original2Copy[k] = v
		}

		result := MergeLabels(original1, original2)

		// Verify original maps are unchanged
		assert.Equal(t, original1Copy, original1, "Original map 1 should not be modified")
		assert.Equal(t, original2Copy, original2, "Original map 2 should not be modified")

		// Verify result is correct
		expected := map[string]string{
			"app":     "redis",
			"tier":    "cache",
			"version": "6.2",
		}
		assert.Equal(t, expected, result)
	})
}

func TestMergeLabelsPerformance(t *testing.T) {
	t.Run("Performance with large label maps", func(t *testing.T) {
		// Create large label maps
		largeMap1 := make(map[string]string, 1000)
		largeMap2 := make(map[string]string, 1000)

		for i := 0; i < 1000; i++ {
			key := fmt.Sprintf("key1_%d", i)
			value := fmt.Sprintf("value1_%d", i)
			largeMap1[key] = value
		}

		for i := 0; i < 1000; i++ {
			key := fmt.Sprintf("key2_%d", i)
			value := fmt.Sprintf("value2_%d", i)
			largeMap2[key] = value
		}

		// This should complete quickly
		result := MergeLabels(largeMap1, largeMap2)

		assert.Equal(t, 2000, len(result), "Should merge all labels from both maps")
	})
}

func TestMergeLabelsRealWorld(t *testing.T) {
	t.Run("Kubernetes label merging scenario", func(t *testing.T) {
		// Simulate common Kubernetes label merging scenario
		appLabels := map[string]string{
			"app":     "redis-cluster",
			"tier":    "cache",
			"version": "6.2.14",
		}

		environmentLabels := map[string]string{
			"environment": "production",
			"region":      "us-west-2",
			"zone":        "us-west-2a",
		}

		operatorLabels := map[string]string{
			"managed-by": "redis-cluster-operator",
			"component":  "redis",
		}

		result := MergeLabels(appLabels, environmentLabels, operatorLabels)

		expected := map[string]string{
			"app":         "redis-cluster",
			"tier":        "cache",
			"version":     "6.2.14",
			"environment": "production",
			"region":      "us-west-2",
			"zone":        "us-west-2a",
			"managed-by":  "redis-cluster-operator",
			"component":   "redis",
		}

		assert.Equal(t, expected, result)
	})
}
