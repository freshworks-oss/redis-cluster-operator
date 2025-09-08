package testutil

import (
	"strings"
	"sync"
	"time"
)

// Int32Ptr creates a pointer to int32 value
// This replaces duplicate int32Ptr functions across multiple test files
func Int32Ptr(v int32) *int32 {
	return &v
}

// StringPtr creates a pointer to string value
func StringPtr(v string) *string {
	return &v
}

// BoolPtr creates a pointer to bool value
func BoolPtr(v bool) *bool {
	return &v
}

// ContainsString checks if a string contains a substring
// This replaces duplicate contains/containsString functions across test files
func ContainsString(s, substr string) bool {
	return strings.Contains(s, substr)
}

// ContainsAny checks if string contains any of the provided substrings
func ContainsAny(s string, substrings ...string) bool {
	for _, substr := range substrings {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

// ConcurrentRunner helps run concurrent test operations
type ConcurrentRunner struct {
	wg sync.WaitGroup
}

// NewConcurrentRunner creates a new concurrent test runner
func NewConcurrentRunner() *ConcurrentRunner {
	return &ConcurrentRunner{}
}

// Add adds a goroutine to run concurrently
func (c *ConcurrentRunner) Add(fn func()) {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		fn()
	}()
}

// Wait waits for all goroutines to complete
func (c *ConcurrentRunner) Wait() {
	c.wg.Wait()
}

// Timeout runs a function with a timeout
func Timeout(duration time.Duration, fn func()) bool {
	done := make(chan bool, 1)
	go func() {
		fn()
		done <- true
	}()

	select {
	case <-done:
		return true
	case <-time.After(duration):
		return false
	}
}

// RepeatUntilTimeout repeats a function until timeout or success
func RepeatUntilTimeout(duration time.Duration, interval time.Duration, condition func() bool) bool {
	timeout := time.After(duration)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return false
		case <-ticker.C:
			if condition() {
				return true
			}
		}
	}
}

// GenerateTestData generates test data of specified size (useful for performance tests)
func GenerateTestData(size int) []byte {
	data := make([]byte, size)
	for i := 0; i < size; i++ {
		data[i] = byte('a' + (i % 26))
	}
	return data
}

// SliceEqual compares two slices for equality (generic comparison)
func SliceEqual[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
