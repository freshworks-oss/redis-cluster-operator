package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Helper function to create int32 pointers for testing (local to this file)
func newInt32Ptr(v int32) *int32 {
	return &v
}

func TestInt32(t *testing.T) {

	tests := []struct {
		name     string
		input    *int32
		expected int32
	}{
		{
			name:     "Nil pointer",
			input:    nil,
			expected: 0,
		},
		{
			name:     "Zero value",
			input:    newInt32Ptr(0),
			expected: 0,
		},
		{
			name:     "Positive value",
			input:    newInt32Ptr(42),
			expected: 42,
		},
		{
			name:     "Negative value",
			input:    newInt32Ptr(-123),
			expected: -123,
		},
		{
			name:     "Maximum int32 value",
			input:    newInt32Ptr(2147483647),
			expected: 2147483647,
		},
		{
			name:     "Minimum int32 value",
			input:    newInt32Ptr(-2147483648),
			expected: -2147483648,
		},
		{
			name:     "Large positive value",
			input:    newInt32Ptr(1000000),
			expected: 1000000,
		},
		{
			name:     "Large negative value",
			input:    newInt32Ptr(-1000000),
			expected: -1000000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Int32(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInt32EdgeCases(t *testing.T) {
	t.Run("Multiple calls with same pointer", func(t *testing.T) {
		value := int32(42)
		ptr := &value

		// Call multiple times to ensure consistency
		result1 := Int32(ptr)
		result2 := Int32(ptr)
		result3 := Int32(ptr)

		assert.Equal(t, int32(42), result1)
		assert.Equal(t, int32(42), result2)
		assert.Equal(t, int32(42), result3)
		assert.Equal(t, result1, result2)
		assert.Equal(t, result2, result3)
	})

	t.Run("Pointer to zero after modification", func(t *testing.T) {
		value := int32(100)
		ptr := &value

		// First call
		result1 := Int32(ptr)
		assert.Equal(t, int32(100), result1)

		// Modify the value
		*ptr = 200

		// Second call should reflect the new value
		result2 := Int32(ptr)
		assert.Equal(t, int32(200), result2)
	})

	t.Run("Nil pointer consistency", func(t *testing.T) {
		// Multiple calls with nil should always return 0
		result1 := Int32(nil)
		result2 := Int32(nil)
		result3 := Int32(nil)

		assert.Equal(t, int32(0), result1)
		assert.Equal(t, int32(0), result2)
		assert.Equal(t, int32(0), result3)
		assert.Equal(t, result1, result2)
		assert.Equal(t, result2, result3)
	})
}

func TestInt32WithRealUsage(t *testing.T) {
	t.Run("Simulating real-world usage", func(t *testing.T) {
		// Simulate a scenario where we might have optional int32 fields
		type TestStruct struct {
			RequiredField int32
			OptionalField *int32
		}

		// Case 1: Optional field is set
		obj1 := TestStruct{
			RequiredField: 100,
			OptionalField: newInt32Ptr(50),
		}

		// Use Int32 to safely get the optional field value
		actualOptional := Int32(obj1.OptionalField)
		assert.Equal(t, int32(50), actualOptional)

		// Case 2: Optional field is nil
		obj2 := TestStruct{
			RequiredField: 200,
			OptionalField: nil,
		}

		// Use Int32 to safely get the optional field value (should return 0)
		actualOptional2 := Int32(obj2.OptionalField)
		assert.Equal(t, int32(0), actualOptional2)
	})
}
