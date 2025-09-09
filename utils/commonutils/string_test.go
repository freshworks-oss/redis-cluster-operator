package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestStringer implements Stringer interface for testing
type TestStringer struct {
	value string
}

func (t TestStringer) String() string {
	return t.value
}

// CustomStringer implements Stringer interface for testing
type CustomStringer struct {
	id   int
	name string
}

func (c CustomStringer) String() string {
	return c.name
}

func TestSliceJoin(t *testing.T) {
	tests := []struct {
		name     string
		slice    []Stringer
		sep      string
		expected string
	}{
		{
			name:     "Empty slice",
			slice:    []Stringer{},
			sep:      ",",
			expected: "",
		},
		{
			name:     "Single element",
			slice:    []Stringer{TestStringer{value: "hello"}},
			sep:      ",",
			expected: "hello",
		},
		{
			name: "Two elements",
			slice: []Stringer{
				TestStringer{value: "hello"},
				TestStringer{value: "world"},
			},
			sep:      ",",
			expected: "hello,world",
		},
		{
			name: "Three elements",
			slice: []Stringer{
				TestStringer{value: "hello"},
				TestStringer{value: "world"},
				TestStringer{value: "test"},
			},
			sep:      ",",
			expected: "hello,world,test",
		},
		{
			name: "Four elements",
			slice: []Stringer{
				TestStringer{value: "a"},
				TestStringer{value: "b"},
				TestStringer{value: "c"},
				TestStringer{value: "d"},
			},
			sep:      "-",
			expected: "a-b-c-d",
		},
		{
			name: "Empty separator",
			slice: []Stringer{
				TestStringer{value: "hello"},
				TestStringer{value: "world"},
			},
			sep:      "",
			expected: "helloworld",
		},
		{
			name: "Complex separator",
			slice: []Stringer{
				TestStringer{value: "part1"},
				TestStringer{value: "part2"},
				TestStringer{value: "part3"},
			},
			sep:      " -> ",
			expected: "part1 -> part2 -> part3",
		},
		{
			name: "Empty strings in slice",
			slice: []Stringer{
				TestStringer{value: ""},
				TestStringer{value: "hello"},
				TestStringer{value: ""},
				TestStringer{value: "world"},
			},
			sep:      ",",
			expected: ",hello,,world",
		},
		{
			name: "Large slice",
			slice: []Stringer{
				TestStringer{value: "1"},
				TestStringer{value: "2"},
				TestStringer{value: "3"},
				TestStringer{value: "4"},
				TestStringer{value: "5"},
				TestStringer{value: "6"},
				TestStringer{value: "7"},
				TestStringer{value: "8"},
				TestStringer{value: "9"},
				TestStringer{value: "10"},
			},
			sep:      "|",
			expected: "1|2|3|4|5|6|7|8|9|10",
		},
		{
			name: "Special characters in separator",
			slice: []Stringer{
				TestStringer{value: "hello"},
				TestStringer{value: "world"},
			},
			sep:      "\n\t",
			expected: "hello\n\tworld",
		},
		{
			name: "Unicode characters",
			slice: []Stringer{
				TestStringer{value: "こんにちは"},
				TestStringer{value: "世界"},
				TestStringer{value: "test"},
			},
			sep:      " - ",
			expected: "こんにちは - 世界 - test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SliceJoin(tt.slice, tt.sep)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSliceJoinEdgeCases(t *testing.T) {
	t.Run("Nil slice", func(t *testing.T) {
		var slice []Stringer
		result := SliceJoin(slice, ",")
		assert.Equal(t, "", result)
	})

	t.Run("Very long strings", func(t *testing.T) {
		longString := "a"
		for i := 0; i < 1000; i++ {
			longString += "a"
		}

		slice := []Stringer{
			TestStringer{value: longString},
			TestStringer{value: "short"},
			TestStringer{value: longString},
		}

		result := SliceJoin(slice, "|")
		expected := longString + "|short|" + longString
		assert.Equal(t, expected, result)
	})

	t.Run("Mixed string lengths", func(t *testing.T) {
		slice := []Stringer{
			TestStringer{value: "a"},
			TestStringer{value: "very long string that should work correctly"},
			TestStringer{value: "b"},
			TestStringer{value: "another long string to test the implementation"},
			TestStringer{value: "c"},
		}

		result := SliceJoin(slice, " -> ")
		expected := "a -> very long string that should work correctly -> b -> another long string to test the implementation -> c"
		assert.Equal(t, expected, result)
	})
}

func TestSliceJoinPerformance(t *testing.T) {
	// Test with a reasonably large slice to ensure performance is acceptable
	slice := make([]Stringer, 1000)
	for i := 0; i < 1000; i++ {
		slice[i] = TestStringer{value: "test"}
	}

	// This should complete quickly
	result := SliceJoin(slice, ",")
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "test")
}

func TestSliceJoinWithCustomStringer(t *testing.T) {
	// Test with a custom Stringer implementation
	slice := []Stringer{
		CustomStringer{id: 1, name: "first"},
		CustomStringer{id: 2, name: "second"},
		CustomStringer{id: 3, name: "third"},
	}

	result := SliceJoin(slice, "|")
	expected := "first|second|third"
	assert.Equal(t, expected, result)
}
