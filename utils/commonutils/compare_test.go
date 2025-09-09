package utils

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
)

// Helper function to create a pointer to int32
func int32Ptr(v int32) *int32 {
	return &v
}

func TestCompareIntValue(t *testing.T) {
	tests := []struct {
		name           string
		old            *int32
		new            *int32
		expectedResult bool
		description    string
	}{
		{
			name:           "Both nil",
			old:            nil,
			new:            nil,
			expectedResult: true,
			description:    "Should return true when both values are nil",
		},
		{
			name:           "Old nil, new not nil",
			old:            nil,
			new:            int32Ptr(42),
			expectedResult: false,
			description:    "Should return false when old is nil and new is not",
		},
		{
			name:           "Old not nil, new nil",
			old:            int32Ptr(42),
			new:            nil,
			expectedResult: false,
			description:    "Should return false when old is not nil and new is nil",
		},
		{
			name:           "Same values",
			old:            int32Ptr(42),
			new:            int32Ptr(42),
			expectedResult: false,
			description:    "Should return false when values are the same",
		},
		{
			name:           "Different values - old less than new",
			old:            int32Ptr(10),
			new:            int32Ptr(42),
			expectedResult: true,
			description:    "Should return true when values are different (old < new)",
		},
		{
			name:           "Different values - old greater than new",
			old:            int32Ptr(100),
			new:            int32Ptr(42),
			expectedResult: true,
			description:    "Should return true when values are different (old > new)",
		},
		{
			name:           "Zero values",
			old:            int32Ptr(0),
			new:            int32Ptr(0),
			expectedResult: false,
			description:    "Should return false when both values are zero",
		},
		{
			name:           "Old zero, new non-zero",
			old:            int32Ptr(0),
			new:            int32Ptr(42),
			expectedResult: true,
			description:    "Should return true when old is zero and new is non-zero",
		},
		{
			name:           "Old non-zero, new zero",
			old:            int32Ptr(42),
			new:            int32Ptr(0),
			expectedResult: true,
			description:    "Should return true when old is non-zero and new is zero",
		},
		{
			name:           "Negative values",
			old:            int32Ptr(-10),
			new:            int32Ptr(-5),
			expectedResult: true,
			description:    "Should return true when values are different negative numbers",
		},
		{
			name:           "Mixed positive and negative",
			old:            int32Ptr(-10),
			new:            int32Ptr(10),
			expectedResult: true,
			description:    "Should return true when values have different signs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logr.Discard()
			result := CompareIntValue("testField", tt.old, tt.new, logger)
			assert.Equal(t, tt.expectedResult, result, tt.description)
		})
	}
}

func TestCompareInt32(t *testing.T) {
	tests := []struct {
		name           string
		old            int32
		new            int32
		expectedResult bool
		description    string
	}{
		{
			name:           "Same values",
			old:            42,
			new:            42,
			expectedResult: false,
			description:    "Should return false when values are the same",
		},
		{
			name:           "Different values - old less than new",
			old:            10,
			new:            42,
			expectedResult: true,
			description:    "Should return true when values are different (old < new)",
		},
		{
			name:           "Different values - old greater than new",
			old:            100,
			new:            42,
			expectedResult: true,
			description:    "Should return true when values are different (old > new)",
		},
		{
			name:           "Zero values",
			old:            0,
			new:            0,
			expectedResult: false,
			description:    "Should return false when both values are zero",
		},
		{
			name:           "Old zero, new non-zero",
			old:            0,
			new:            42,
			expectedResult: true,
			description:    "Should return true when old is zero and new is non-zero",
		},
		{
			name:           "Old non-zero, new zero",
			old:            42,
			new:            0,
			expectedResult: true,
			description:    "Should return true when old is non-zero and new is zero",
		},
		{
			name:           "Negative values",
			old:            -10,
			new:            -5,
			expectedResult: true,
			description:    "Should return true when values are different negative numbers",
		},
		{
			name:           "Mixed positive and negative",
			old:            -10,
			new:            10,
			expectedResult: true,
			description:    "Should return true when values have different signs",
		},
		{
			name:           "Large values",
			old:            1000000,
			new:            2000000,
			expectedResult: true,
			description:    "Should handle large values correctly",
		},
		{
			name:           "Edge case - max int32",
			old:            2147483647,
			new:            2147483646,
			expectedResult: true,
			description:    "Should handle max int32 values correctly",
		},
		{
			name:           "Edge case - min int32",
			old:            -2147483648,
			new:            -2147483647,
			expectedResult: true,
			description:    "Should handle min int32 values correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logr.Discard()
			result := CompareInt32("testField", tt.old, tt.new, logger)
			assert.Equal(t, tt.expectedResult, result, tt.description)
		})
	}
}

func TestCompareStringValue(t *testing.T) {
	tests := []struct {
		name           string
		old            string
		new            string
		expectedResult bool
		description    string
	}{
		{
			name:           "Same values",
			old:            "test",
			new:            "test",
			expectedResult: false,
			description:    "Should return false when values are the same",
		},
		{
			name:           "Different values",
			old:            "oldValue",
			new:            "newValue",
			expectedResult: true,
			description:    "Should return true when values are different",
		},
		{
			name:           "Empty strings",
			old:            "",
			new:            "",
			expectedResult: false,
			description:    "Should return false when both values are empty",
		},
		{
			name:           "Old empty, new not empty",
			old:            "",
			new:            "newValue",
			expectedResult: true,
			description:    "Should return true when old is empty and new is not",
		},
		{
			name:           "Old not empty, new empty",
			old:            "oldValue",
			new:            "",
			expectedResult: true,
			description:    "Should return true when old is not empty and new is empty",
		},
		{
			name:           "Case sensitive comparison",
			old:            "Test",
			new:            "test",
			expectedResult: true,
			description:    "Should be case sensitive",
		},
		{
			name:           "Whitespace differences",
			old:            "test",
			new:            " test ",
			expectedResult: true,
			description:    "Should detect whitespace differences",
		},
		{
			name:           "Special characters",
			old:            "test-value",
			new:            "test_value",
			expectedResult: true,
			description:    "Should detect special character differences",
		},
		{
			name:           "Long strings",
			old:            "very long string with many characters",
			new:            "very long string with many characters",
			expectedResult: false,
			description:    "Should handle long strings correctly",
		},
		{
			name:           "Unicode characters",
			old:            "こんにちは",
			new:            "世界",
			expectedResult: true,
			description:    "Should handle unicode characters correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logr.Discard()
			result := CompareStringValue("testField", tt.old, tt.new, logger)
			assert.Equal(t, tt.expectedResult, result, tt.description)
		})
	}
}

func TestCompareFunctionsLogging(t *testing.T) {
	t.Run("Logging when values differ", func(t *testing.T) {
		logger := logr.Discard()

		// Test Int32 comparison
		result := CompareInt32("cpu", 2, 4, logger)
		assert.True(t, result)
	})

	t.Run("No logging when values are the same", func(t *testing.T) {
		logger := logr.Discard()

		// Test Int32 comparison
		result := CompareInt32("cpu", 4, 4, logger)
		assert.False(t, result)
	})
}

func TestCompareFunctionsEdgeCases(t *testing.T) {
	t.Run("Nil logger handling", func(t *testing.T) {
		// Create a no-op logger for nil cases
		noOpLogger := logr.Discard()

		// These should not panic even with no-op logger
		assert.NotPanics(t, func() {
			CompareIntValue("test", int32Ptr(1), int32Ptr(2), noOpLogger)
		})

		assert.NotPanics(t, func() {
			CompareInt32("test", 1, 2, noOpLogger)
		})

		assert.NotPanics(t, func() {
			CompareStringValue("test", "old", "new", noOpLogger)
		})
	})

	t.Run("Extreme values", func(t *testing.T) {
		logger := logr.Discard()

		// Test with extreme int32 values
		result := CompareInt32("extreme", 2147483647, -2147483648, logger)
		assert.True(t, result)
	})
}
