package clustering

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorType_New(t *testing.T) {
	tests := []struct {
		name      string
		errorType ErrorType
		msg       string
		expected  string
	}{
		{
			name:      "NoType error",
			errorType: NoType,
			msg:       "test error",
			expected:  "test error",
		},
		{
			name:      "Requeue error",
			errorType: Requeue,
			msg:       "requeue error",
			expected:  "requeue error",
		},
		{
			name:      "StopRetry error",
			errorType: StopRetry,
			msg:       "stop retry error",
			expected:  "stop retry error",
		},
		{
			name:      "HealFailure error",
			errorType: HealFailure,
			msg:       "heal failure error",
			expected:  "heal failure error",
		},
		{
			name:      "HealReconcile error",
			errorType: HealReconcile,
			msg:       "heal reconcile error",
			expected:  "heal reconcile error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.errorType.New(tt.msg)
			assert.Error(t, err)
			assert.Equal(t, tt.expected, err.Error())

			// Check if it's a customError
			if customErr, ok := err.(customError); ok {
				assert.Equal(t, tt.errorType, customErr.errorType)
			}
		})
	}
}

func TestErrorType_Newf(t *testing.T) {
	tests := []struct {
		name      string
		errorType ErrorType
		msg       string
		args      []interface{}
		expected  string
	}{
		{
			name:      "NoType error with format",
			errorType: NoType,
			msg:       "test error: %s",
			args:      []interface{}{"value"},
			expected:  "test error: value",
		},
		{
			name:      "Requeue error with format",
			errorType: Requeue,
			msg:       "requeue error: %d",
			args:      []interface{}{42},
			expected:  "requeue error: 42",
		},
		{
			name:      "Multiple format args",
			errorType: StopRetry,
			msg:       "stop retry error: %s %d",
			args:      []interface{}{"test", 123},
			expected:  "stop retry error: test 123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.errorType.Newf(tt.msg, tt.args...)
			assert.Error(t, err)
			assert.Equal(t, tt.expected, err.Error())

			// Check if it's a customError
			if customErr, ok := err.(customError); ok {
				assert.Equal(t, tt.errorType, customErr.errorType)
			}
		})
	}
}

func TestErrorType_Wrap(t *testing.T) {
	tests := []struct {
		name      string
		errorType ErrorType
		original  error
		msg       string
		expected  string
	}{
		{
			name:      "Wrap NoType error",
			errorType: NoType,
			original:  errors.New("original error"),
			msg:       "wrapped error",
			expected:  "wrapped error: original error",
		},
		{
			name:      "Wrap Requeue error",
			errorType: Requeue,
			original:  errors.New("original error"),
			msg:       "wrapped error",
			expected:  "wrapped error: original error",
		},
		{
			name:      "Wrap nil error",
			errorType: NoType,
			original:  nil,
			msg:       "wrapped error",
			expected:  "wrapped error: <nil>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.errorType.Wrap(tt.original, tt.msg)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expected)

			// Check if it's a customError
			if customErr, ok := err.(customError); ok {
				assert.Equal(t, tt.errorType, customErr.errorType)
			}
		})
	}
}

func TestErrorType_Wrapf(t *testing.T) {
	tests := []struct {
		name      string
		errorType ErrorType
		original  error
		msg       string
		args      []interface{}
		expected  string
	}{
		{
			name:      "Wrapf NoType error",
			errorType: NoType,
			original:  errors.New("original error"),
			msg:       "wrapped error: %s",
			args:      []interface{}{"value"},
			expected:  "wrapped error: value: original error",
		},
		{
			name:      "Wrapf Requeue error",
			errorType: Requeue,
			original:  errors.New("original error"),
			msg:       "wrapped error: %d",
			args:      []interface{}{42},
			expected:  "wrapped error: 42: original error",
		},
		{
			name:      "Wrapf with multiple args",
			errorType: StopRetry,
			original:  errors.New("original error"),
			msg:       "wrapped error: %s %d",
			args:      []interface{}{"test", 123},
			expected:  "wrapped error: test 123: original error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.errorType.Wrapf(tt.original, tt.msg, tt.args...)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expected)

			// Check if it's a customError
			if customErr, ok := err.(customError); ok {
				assert.Equal(t, tt.errorType, customErr.errorType)
			}
		})
	}
}

func TestCustomError_Error(t *testing.T) {
	tests := []struct {
		name          string
		originalError error
		errorType     ErrorType
		expected      string
	}{
		{
			name:          "Normal error case",
			originalError: errors.New("original error"),
			errorType:     Requeue,
			expected:      "original error",
		},
		{
			name:          "Nil original error case",
			originalError: nil,
			errorType:     NoType,
			expected:      "<nil>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customErr := customError{
				errorType:     tt.errorType,
				originalError: tt.originalError,
			}

			assert.Equal(t, tt.expected, customErr.Error())
		})
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		msg      string
		expected string
	}{
		{
			name:     "Simple message",
			msg:      "test error",
			expected: "test error",
		},
		{
			name:     "Empty message",
			msg:      "",
			expected: "",
		},
		{
			name:     "Complex message",
			msg:      "complex error with special chars: !@#$%",
			expected: "complex error with special chars: !@#$%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(tt.msg)
			assert.Error(t, err)
			assert.Equal(t, tt.expected, err.Error())

			// Check if it's a customError with NoType
			if customErr, ok := err.(customError); ok {
				assert.Equal(t, NoType, customErr.errorType)
			}
		})
	}
}

func TestNewf(t *testing.T) {
	tests := []struct {
		name     string
		msg      string
		args     []interface{}
		expected string
	}{
		{
			name:     "Simple format",
			msg:      "test error: %s",
			args:     []interface{}{"value"},
			expected: "test error: value",
		},
		{
			name:     "Multiple format args",
			msg:      "test error: %s %d",
			args:     []interface{}{"value", 42},
			expected: "test error: value 42",
		},
		{
			name:     "No format args",
			msg:      "test error",
			args:     []interface{}{},
			expected: "test error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Newf(tt.msg, tt.args...)
			assert.Error(t, err)
			assert.Equal(t, tt.expected, err.Error())

			// Check if it's a customError with NoType
			if customErr, ok := err.(customError); ok {
				assert.Equal(t, NoType, customErr.errorType)
			}
		})
	}
}

func TestWrap(t *testing.T) {
	tests := []struct {
		name     string
		original error
		msg      string
		expected string
	}{
		{
			name:     "Wrap regular error",
			original: errors.New("original error"),
			msg:      "wrapped error",
			expected: "wrapped error: original error",
		},
		{
			name:     "Wrap nil error",
			original: nil,
			msg:      "wrapped error",
			expected: "wrapped error: <nil>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Wrap(tt.original, tt.msg)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expected)
		})
	}
}

func TestCause(t *testing.T) {
	tests := []struct {
		name     string
		original error
		expected error
	}{
		{
			name:     "Regular error",
			original: errors.New("original error"),
			expected: errors.New("original error"),
		},
		{
			name:     "Nil error",
			original: nil,
			expected: nil,
		},
		{
			name:     "Wrapped error",
			original: Wrap(errors.New("original error"), "wrapped"),
			expected: errors.New("original error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Cause(tt.original)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, tt.expected.Error(), result.Error())
			}
		})
	}
}

func TestWrapf(t *testing.T) {
	tests := []struct {
		name     string
		original error
		msg      string
		args     []interface{}
		expected string
	}{
		{
			name:     "Wrapf regular error",
			original: errors.New("original error"),
			msg:      "wrapped error: %s",
			args:     []interface{}{"value"},
			expected: "wrapped error: value: original error",
		},
		{
			name:     "Wrapf with customError",
			original: Requeue.New("requeue error"),
			msg:      "wrapped error: %s",
			args:     []interface{}{"value"},
			expected: "wrapped error: value: requeue error",
		},
		{
			name:     "Wrapf nil error",
			original: nil,
			msg:      "wrapped error: %s",
			args:     []interface{}{"value"},
			expected: "wrapped error: value: <nil>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Wrapf(tt.original, tt.msg, tt.args...)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expected)
		})
	}
}

func TestGetType(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected ErrorType
	}{
		{
			name:     "NoType error",
			err:      New("test error"),
			expected: NoType,
		},
		{
			name:     "Requeue error",
			err:      Requeue.New("requeue error"),
			expected: Requeue,
		},
		{
			name:     "StopRetry error",
			err:      StopRetry.New("stop retry error"),
			expected: StopRetry,
		},
		{
			name:     "HealFailure error",
			err:      HealFailure.New("heal failure error"),
			expected: HealFailure,
		},
		{
			name:     "HealReconcile error",
			err:      HealReconcile.New("heal reconcile error"),
			expected: HealReconcile,
		},
		{
			name:     "Regular error",
			err:      errors.New("regular error"),
			expected: NoType,
		},
		{
			name:     "Nil error",
			err:      nil,
			expected: NoType,
		},
		{
			name:     "Wrapped custom error",
			err:      Requeue.Wrap(errors.New("original"), "wrapped"),
			expected: Requeue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetType(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
