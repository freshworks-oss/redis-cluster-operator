package utils

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRound(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected int
	}{
		{
			name:     "Positive integer",
			input:    42.0,
			expected: 42,
		},
		{
			name:     "Negative integer",
			input:    -42.0,
			expected: -42,
		},
		{
			name:     "Zero",
			input:    0.0,
			expected: 0,
		},
		{
			name:     "Positive decimal less than 0.5",
			input:    42.3,
			expected: 42,
		},
		{
			name:     "Positive decimal exactly 0.5",
			input:    42.5,
			expected: 43,
		},
		{
			name:     "Positive decimal greater than 0.5",
			input:    42.7,
			expected: 43,
		},
		{
			name:     "Negative decimal less than 0.5",
			input:    -42.3,
			expected: -42,
		},
		{
			name:     "Negative decimal exactly 0.5",
			input:    -42.5,
			expected: -43, // Rounds away from zero
		},
		{
			name:     "Negative decimal greater than 0.5",
			input:    -42.7,
			expected: -43,
		},
		{
			name:     "Very small positive decimal",
			input:    42.1,
			expected: 42,
		},
		{
			name:     "Very small negative decimal",
			input:    -42.1,
			expected: -42,
		},
		{
			name:     "Very large positive decimal",
			input:    42.9,
			expected: 43,
		},
		{
			name:     "Very large negative decimal",
			input:    -42.9,
			expected: -43,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Round(tt.input)
			assert.Equal(t, tt.expected, result,
				"Round(%.1f) = %d, expected %d", tt.input, result, tt.expected)
		})
	}
}

func TestRoundEdgeCases(t *testing.T) {
	t.Run("Positive infinity", func(t *testing.T) {
		result := Round(math.Inf(1))
		assert.Equal(t, math.MaxInt64, result, "Should handle positive infinity") // Returns int64 max, not int32 max
	})

	t.Run("Negative infinity", func(t *testing.T) {
		result := Round(math.Inf(-1))
		assert.Equal(t, math.MinInt64, result, "Should handle negative infinity") // Returns int64 min, not int32 min
	})

	t.Run("NaN", func(t *testing.T) {
		// NaN behavior is undefined, but should not panic
		// We just verify the function doesn't crash
		assert.NotPanics(t, func() {
			Round(math.NaN())
		}, "Should not panic on NaN")
	})

	t.Run("Very large positive number", func(t *testing.T) {
		result := Round(1e15)
		assert.Equal(t, 1000000000000000, result, "Should handle very large positive numbers")
	})

	t.Run("Very large negative number", func(t *testing.T) {
		result := Round(-1e15)
		assert.Equal(t, -1000000000000000, result, "Should handle very large negative numbers")
	})

	t.Run("Very small positive number", func(t *testing.T) {
		result := Round(1e-15)
		assert.Equal(t, 0, result, "Should handle very small positive numbers")
	})

	t.Run("Very small negative number", func(t *testing.T) {
		result := Round(-1e-15)
		assert.Equal(t, 0, result, "Should handle very small negative numbers")
	})
}

func TestRoundPrecision(t *testing.T) {
	t.Run("Precision around 0.5 boundary", func(t *testing.T) {
		// Test values very close to 0.5
		assert.Equal(t, 0, Round(0.4999999999999999))
		assert.Equal(t, 1, Round(0.5))
		assert.Equal(t, 1, Round(0.5000000000000001))

		assert.Equal(t, 0, Round(-0.4999999999999999))
		assert.Equal(t, -1, Round(-0.5)) // Rounds away from zero for negative 0.5
		assert.Equal(t, -1, Round(-0.5000000000000001))
	})

	t.Run("Precision around integer boundaries", func(t *testing.T) {
		// Test values very close to integers
		assert.Equal(t, 42, Round(41.999999999999999))
		assert.Equal(t, 42, Round(42.0))
		assert.Equal(t, 42, Round(42.000000000000001))

		assert.Equal(t, -42, Round(-41.999999999999999))
		assert.Equal(t, -42, Round(-42.0))
		assert.Equal(t, -42, Round(-42.000000000000001))
	})
}

func TestRoundMathematicalProperties(t *testing.T) {
	t.Run("Symmetry around zero", func(t *testing.T) {
		// The Round function uses math.Copysign which means it rounds away from zero
		// So Round(x) + Round(-x) = 0 for most values, not 1
		testValues := []float64{0.1, 0.5, 0.9, 1.1, 1.5, 1.9, 42.3, 42.5, 42.7}

		for _, x := range testValues {
			posResult := Round(x)
			negResult := Round(-x)

			// For the Round function's behavior, symmetric rounding away from zero
			// means Round(x) + Round(-x) should be 0 for most cases
			expectedSum := 0

			assert.Equal(t, expectedSum, posResult+negResult,
				"Round(%.1f) + Round(%.1f) = %d + %d = %d, expected %d",
				x, -x, posResult, negResult, posResult+negResult, expectedSum)
		}
	})

	t.Run("Monotonicity", func(t *testing.T) {
		// Round should be monotonically increasing
		values := []float64{-10.5, -10.0, -9.5, -9.0, -8.5, -8.0, -7.5, -7.0, -6.5, -6.0}
		results := make([]int, len(values))

		for i, v := range values {
			results[i] = Round(v)
		}

		// Verify results are monotonically increasing
		for i := 1; i < len(results); i++ {
			assert.LessOrEqual(t, results[i-1], results[i],
				"Round should be monotonically increasing: Round(%.1f) = %d <= Round(%.1f) = %d",
				values[i-1], results[i-1], values[i], results[i])
		}
	})
}

func TestRoundPerformance(t *testing.T) {
	t.Run("Performance with many calls", func(t *testing.T) {
		// Test performance with many calls
		testValues := []float64{0.1, 0.5, 0.9, 1.1, 1.5, 1.9, 42.3, 42.5, 42.7, -42.3, -42.5, -42.7}

		// This should complete quickly
		for i := 0; i < 10000; i++ {
			for _, v := range testValues {
				result := Round(v)
				// Just verify the result is reasonable
				assert.GreaterOrEqual(t, result, -1000)
				assert.LessOrEqual(t, result, 1000)
			}
		}
	})
}

func TestRoundRealWorld(t *testing.T) {
	t.Run("Percentage calculations", func(t *testing.T) {
		// Simulate percentage calculations
		percentage := 85.7
		roundedPercentage := Round(percentage)
		assert.Equal(t, 86, roundedPercentage, "85.7% should round to 86%")

		percentage2 := 92.3
		roundedPercentage2 := Round(percentage2)
		assert.Equal(t, 92, roundedPercentage2, "92.3% should round to 92%")
	})

	t.Run("Resource allocation", func(t *testing.T) {
		// Simulate resource allocation scenarios
		cpuCores := 3.7
		roundedCores := Round(cpuCores)
		assert.Equal(t, 4, roundedCores, "3.7 CPU cores should round to 4")

		memoryGB := 8.2
		roundedMemory := Round(memoryGB)
		assert.Equal(t, 8, roundedMemory, "8.2 GB should round to 8")
	})

	t.Run("Time calculations", func(t *testing.T) {
		// Simulate time calculations
		hours := 12.5
		roundedHours := Round(hours)
		assert.Equal(t, 13, roundedHours, "12.5 hours should round to 13")

		minutes := 30.0
		roundedMinutes := Round(minutes)
		assert.Equal(t, 30, roundedMinutes, "30.0 minutes should round to 30")
	})
}
