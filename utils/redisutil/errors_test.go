package redisutil

import (
	"errors"
	"fmt"
	"testing"
)

// TestError_Error tests the Error type's Error method
func TestError_Error(t *testing.T) {
	tests := []struct {
		name     string
		e        Error
		expected string
	}{
		{
			name:     "Simple error message",
			e:        Error("test error"),
			expected: "test error",
		},
		{
			name:     "Empty error message",
			e:        Error(""),
			expected: "",
		},
		{
			name:     "Error with special characters",
			e:        Error("error: node not found! @#$%"),
			expected: "error: node not found! @#$%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.e.Error(); got != tt.expected {
				t.Errorf("Error.Error() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

// TestNodeNotFoundedError tests the nodeNotFoundedError constant
func TestNodeNotFoundedError(t *testing.T) {
	err := nodeNotFoundedError
	expectedMessage := "node not founded"

	if err.Error() != expectedMessage {
		t.Errorf("nodeNotFoundedError.Error() = %v, expected %v", err.Error(), expectedMessage)
	}
}

// TestIsNodeNotFoundedError tests the IsNodeNotFoundedError function
func TestIsNodeNotFoundedError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "Correct nodeNotFoundedError",
			err:      nodeNotFoundedError,
			expected: true,
		},
		{
			name:     "Different Error type with same message",
			err:      Error("node not founded"),
			expected: true, // Same underlying string value
		},
		{
			name:     "Standard error with same message",
			err:      errors.New("node not founded"),
			expected: false,
		},
		{
			name:     "Different error message",
			err:      errors.New("different error"),
			expected: false,
		},
		{
			name:     "Nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNodeNotFoundedError(tt.err); got != tt.expected {
				t.Errorf("IsNodeNotFoundedError() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

// TestNewClusterInfosError tests the NewClusterInfosError constructor
func TestNewClusterInfosError(t *testing.T) {
	err := NewClusterInfosError()

	if err.errs == nil {
		t.Error("NewClusterInfosError() should initialize errs map")
	}

	if err.partial {
		t.Error("NewClusterInfosError() should initialize partial as false")
	}

	if err.inconsistent {
		t.Error("NewClusterInfosError() should initialize inconsistent as false")
	}

	if len(err.errs) != 0 {
		t.Errorf("NewClusterInfosError() should initialize empty errs map, got length %d", len(err.errs))
	}
}

// TestClusterInfosError_Error tests the ClusterInfosError Error method
func TestClusterInfosError_Error(t *testing.T) {
	tests := []struct {
		name     string
		setupErr func() ClusterInfosError
		expected string
	}{
		{
			name: "New error with no flags set",
			setupErr: func() ClusterInfosError {
				return NewClusterInfosError()
			},
			expected: "",
		},
		{
			name: "Error with inconsistent flag",
			setupErr: func() ClusterInfosError {
				err := NewClusterInfosError()
				err.inconsistent = true
				return err
			},
			expected: "Cluster view is inconsistent",
		},
		{
			name: "Error with partial flag and errors",
			setupErr: func() ClusterInfosError {
				err := NewClusterInfosError()
				err.partial = true
				err.errs["127.0.0.1:6379"] = errors.New("connection refused")
				err.errs["127.0.0.1:6380"] = errors.New("timeout")
				return err
			},
			expected: "Cluster infos partial: 127.0.0.1:6379: 'connection refused'127.0.0.1:6380: 'timeout'",
		},
		{
			name: "Error with partial flag but no errors",
			setupErr: func() ClusterInfosError {
				err := NewClusterInfosError()
				err.partial = true
				return err
			},
			expected: "Cluster infos partial: ",
		},
		{
			name: "Error with both flags set (partial takes precedence)",
			setupErr: func() ClusterInfosError {
				err := NewClusterInfosError()
				err.partial = true
				err.inconsistent = true
				err.errs["127.0.0.1:6379"] = errors.New("test error")
				return err
			},
			expected: "Cluster infos partial: 127.0.0.1:6379: 'test error'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.setupErr()
			got := err.Error()

			// For partial errors, we need to handle the map iteration order
			if err.partial && len(err.errs) > 1 {
				// Check that the string contains expected parts
				if len(got) < len("Cluster infos partial: ") {
					t.Errorf("ClusterInfosError.Error() = %v, should contain partial error prefix", got)
				}
				// Check that all error addresses are present
				for addr := range err.errs {
					if !contains(got, addr) {
						t.Errorf("ClusterInfosError.Error() = %v, should contain address %v", got, addr)
					}
				}
			} else {
				if got != tt.expected {
					t.Errorf("ClusterInfosError.Error() = %v, expected %v", got, tt.expected)
				}
			}
		})
	}
}

// TestClusterInfosError_Partial tests the Partial method
func TestClusterInfosError_Partial(t *testing.T) {
	tests := []struct {
		name     string
		setupErr func() ClusterInfosError
		expected bool
	}{
		{
			name: "New error (not partial)",
			setupErr: func() ClusterInfosError {
				return NewClusterInfosError()
			},
			expected: false,
		},
		{
			name: "Error with partial flag set",
			setupErr: func() ClusterInfosError {
				err := NewClusterInfosError()
				err.partial = true
				return err
			},
			expected: true,
		},
		{
			name: "Error with inconsistent flag (not partial)",
			setupErr: func() ClusterInfosError {
				err := NewClusterInfosError()
				err.inconsistent = true
				return err
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.setupErr()
			if got := err.Partial(); got != tt.expected {
				t.Errorf("ClusterInfosError.Partial() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

// TestClusterInfosError_Inconsistent tests the Inconsistent method
func TestClusterInfosError_Inconsistent(t *testing.T) {
	tests := []struct {
		name     string
		setupErr func() ClusterInfosError
		expected bool
	}{
		{
			name: "New error (not inconsistent)",
			setupErr: func() ClusterInfosError {
				return NewClusterInfosError()
			},
			expected: false,
		},
		{
			name: "Error with inconsistent flag set",
			setupErr: func() ClusterInfosError {
				err := NewClusterInfosError()
				err.inconsistent = true
				return err
			},
			expected: true,
		},
		{
			name: "Error with partial flag (not inconsistent)",
			setupErr: func() ClusterInfosError {
				err := NewClusterInfosError()
				err.partial = true
				return err
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.setupErr()
			if got := err.Inconsistent(); got != tt.expected {
				t.Errorf("ClusterInfosError.Inconsistent() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

// TestIsPartialError tests the IsPartialError function
func TestIsPartialError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name: "ClusterInfosError with partial flag",
			err: func() error {
				err := NewClusterInfosError()
				err.partial = true
				return err
			}(),
			expected: true,
		},
		{
			name: "ClusterInfosError without partial flag",
			err: func() error {
				err := NewClusterInfosError()
				err.inconsistent = true
				return err
			}(),
			expected: false,
		},
		{
			name:     "Standard error",
			err:      errors.New("standard error"),
			expected: false,
		},
		{
			name:     "Error type",
			err:      Error("custom error"),
			expected: false,
		},
		{
			name:     "Nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsPartialError(tt.err); got != tt.expected {
				t.Errorf("IsPartialError() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

// TestIsInconsistentError tests the IsInconsistentError function
func TestIsInconsistentError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name: "ClusterInfosError with inconsistent flag",
			err: func() error {
				err := NewClusterInfosError()
				err.inconsistent = true
				return err
			}(),
			expected: true,
		},
		{
			name: "ClusterInfosError without inconsistent flag",
			err: func() error {
				err := NewClusterInfosError()
				err.partial = true
				return err
			}(),
			expected: false,
		},
		{
			name:     "Standard error",
			err:      errors.New("standard error"),
			expected: false,
		},
		{
			name:     "Error type",
			err:      Error("custom error"),
			expected: false,
		},
		{
			name:     "Nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsInconsistentError(tt.err); got != tt.expected {
				t.Errorf("IsInconsistentError() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

// TestClusterInfosError_Integration tests integration scenarios
func TestClusterInfosError_Integration(t *testing.T) {
	t.Run("Complete workflow", func(t *testing.T) {
		// Create new error
		err := NewClusterInfosError()

		// Should not be partial or inconsistent initially
		if IsPartialError(err) {
			t.Error("New error should not be partial")
		}
		if IsInconsistentError(err) {
			t.Error("New error should not be inconsistent")
		}

		// Set partial and add errors
		err.partial = true
		err.errs["node1"] = errors.New("connection failed")
		err.errs["node2"] = fmt.Errorf("timeout after %d seconds", 30)

		// Should now be partial
		if !IsPartialError(err) {
			t.Error("Error should be partial after setting flag")
		}
		if IsInconsistentError(err) {
			t.Error("Error should not be inconsistent")
		}

		// Error message should contain addresses and error details
		errMsg := err.Error()
		if !contains(errMsg, "Cluster infos partial:") {
			t.Error("Error message should contain partial prefix")
		}
		if !contains(errMsg, "node1") || !contains(errMsg, "node2") {
			t.Error("Error message should contain node addresses")
		}
	})

	t.Run("Type assertion edge cases", func(t *testing.T) {
		var err error = NewClusterInfosError()

		// Should work with interface{}
		if clusterErr, ok := err.(ClusterInfosError); !ok {
			t.Error("Should be able to type assert to ClusterInfosError")
		} else {
			if clusterErr.Partial() {
				t.Error("Fresh error should not be partial")
			}
		}
	})
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(len(substr) == 0 || func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}
