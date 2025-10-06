package distributedrediscluster

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestErrorTypes(t *testing.T) {
	// Test all error type constants
	assert.Equal(t, ErrorType(0), NoType)
	assert.Equal(t, ErrorType(1), Requeue)
	assert.Equal(t, ErrorType(2), Kubernetes)
	assert.Equal(t, ErrorType(3), Redis)
	assert.Equal(t, ErrorType(4), Cluster)
	assert.Equal(t, ErrorType(5), StopRetry)
	assert.Equal(t, ErrorType(6), InvalidSpec)
	assert.Equal(t, ErrorType(7), EnsurePDB)
	assert.Equal(t, ErrorType(8), EnsureConfigMap)
	assert.Equal(t, ErrorType(9), EnsureSTS)
	assert.Equal(t, ErrorType(10), EnsureService)
}

func TestErrorTypeNew(t *testing.T) {
	tests := []struct {
		name        string
		errorType   ErrorType
		message     string
		expectedMsg string
	}{
		{
			name:        "Requeue error",
			errorType:   Requeue,
			message:     "need to requeue",
			expectedMsg: "need to requeue",
		},
		{
			name:        "Kubernetes error",
			errorType:   Kubernetes,
			message:     "k8s api error",
			expectedMsg: "k8s api error",
		},
		{
			name:        "Redis error",
			errorType:   Redis,
			message:     "redis connection failed",
			expectedMsg: "redis connection failed",
		},
		{
			name:        "Cluster error",
			errorType:   Cluster,
			message:     "cluster operation failed",
			expectedMsg: "cluster operation failed",
		},
		{
			name:        "StopRetry error",
			errorType:   StopRetry,
			message:     "fatal error, stop retrying",
			expectedMsg: "fatal error, stop retrying",
		},
		{
			name:        "InvalidSpec error",
			errorType:   InvalidSpec,
			message:     "invalid cluster spec",
			expectedMsg: "invalid cluster spec",
		},
		{
			name:        "EnsurePDB error",
			errorType:   EnsurePDB,
			message:     "failed to ensure PDB",
			expectedMsg: "failed to ensure PDB",
		},
		{
			name:        "EnsureConfigMap error",
			errorType:   EnsureConfigMap,
			message:     "failed to ensure configmap",
			expectedMsg: "failed to ensure configmap",
		},
		{
			name:        "EnsureSTS error",
			errorType:   EnsureSTS,
			message:     "failed to ensure statefulset",
			expectedMsg: "failed to ensure statefulset",
		},
		{
			name:        "EnsureService error",
			errorType:   EnsureService,
			message:     "failed to ensure service",
			expectedMsg: "failed to ensure service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.errorType.New(tt.message)

			assert.Error(t, err)
			assert.Equal(t, tt.expectedMsg, err.Error())
			assert.Equal(t, tt.errorType, GetType(err))
		})
	}
}

func TestErrorTypeNewf(t *testing.T) {
	tests := []struct {
		name        string
		errorType   ErrorType
		format      string
		args        []interface{}
		expectedMsg string
	}{
		{
			name:        "Formatted Requeue error",
			errorType:   Requeue,
			format:      "need to requeue after %d seconds",
			args:        []interface{}{30},
			expectedMsg: "need to requeue after 30 seconds",
		},
		{
			name:        "Formatted Kubernetes error",
			errorType:   Kubernetes,
			format:      "k8s api error for resource %s in namespace %s",
			args:        []interface{}{"pod", "default"},
			expectedMsg: "k8s api error for resource pod in namespace default",
		},
		{
			name:        "Formatted Redis error",
			errorType:   Redis,
			format:      "redis connection failed to %s:%d",
			args:        []interface{}{"10.0.0.1", 6379},
			expectedMsg: "redis connection failed to 10.0.0.1:6379",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.errorType.Newf(tt.format, tt.args...)

			assert.Error(t, err)
			assert.Equal(t, tt.expectedMsg, err.Error())
			assert.Equal(t, tt.errorType, GetType(err))
		})
	}
}

func TestErrorTypeWrap(t *testing.T) {
	originalErr := fmt.Errorf("original error")

	tests := []struct {
		name         string
		errorType    ErrorType
		originalErr  error
		message      string
		expectedType ErrorType
	}{
		{
			name:         "Wrap with Requeue",
			errorType:    Requeue,
			originalErr:  originalErr,
			message:      "wrapping with requeue",
			expectedType: Requeue,
		},
		{
			name:         "Wrap with Kubernetes",
			errorType:    Kubernetes,
			originalErr:  originalErr,
			message:      "wrapping with kubernetes",
			expectedType: Kubernetes,
		},
		{
			name:         "Wrap with Redis",
			errorType:    Redis,
			originalErr:  originalErr,
			message:      "wrapping with redis",
			expectedType: Redis,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.errorType.Wrap(tt.originalErr, tt.message)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.message)
			assert.Contains(t, err.Error(), "original error")
			assert.Equal(t, tt.expectedType, GetType(err))
		})
	}
}

func TestErrorTypeWrapf(t *testing.T) {
	originalErr := fmt.Errorf("original error")

	tests := []struct {
		name         string
		errorType    ErrorType
		originalErr  error
		format       string
		args         []interface{}
		expectedType ErrorType
	}{
		{
			name:         "Wrapf with Requeue",
			errorType:    Requeue,
			originalErr:  originalErr,
			format:       "wrapping with requeue after %d seconds",
			args:         []interface{}{30},
			expectedType: Requeue,
		},
		{
			name:         "Wrapf with Kubernetes",
			errorType:    Kubernetes,
			originalErr:  originalErr,
			format:       "wrapping with kubernetes for resource %s",
			args:         []interface{}{"pod"},
			expectedType: Kubernetes,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.errorType.Wrapf(tt.originalErr, tt.format, tt.args...)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "original error")
			assert.Equal(t, tt.expectedType, GetType(err))
		})
	}
}

func TestCustomErrorError(t *testing.T) {
	originalErr := fmt.Errorf("test error message")
	customErr := customError{
		errorType:     Requeue,
		originalError: originalErr,
	}

	assert.Equal(t, "test error message", customErr.Error())
}

func TestNew(t *testing.T) {
	message := "test no type error"
	err := New(message)

	assert.Error(t, err)
	assert.Equal(t, message, err.Error())
	assert.Equal(t, NoType, GetType(err))
}

func TestNewf(t *testing.T) {
	format := "test no type error with %s and %d"
	args := []interface{}{"string", 42}
	expectedMsg := "test no type error with string and 42"

	err := Newf(format, args...)

	assert.Error(t, err)
	assert.Equal(t, expectedMsg, err.Error())
	assert.Equal(t, NoType, GetType(err))
}

func TestWrap(t *testing.T) {
	originalErr := fmt.Errorf("original error")
	message := "wrapping message"

	err := Wrap(originalErr, message)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), message)
	assert.Contains(t, err.Error(), "original error")
	assert.Equal(t, NoType, GetType(err))
}

func TestWrapWithCustomError(t *testing.T) {
	// Create a custom error first
	originalCustomErr := Requeue.New("original requeue error")
	message := "wrapping custom error"

	err := Wrap(originalCustomErr, message)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), message)
	assert.Contains(t, err.Error(), "original requeue error")
	assert.Equal(t, Requeue, GetType(err)) // Should preserve the original error type
}

func TestWrapf(t *testing.T) {
	originalErr := fmt.Errorf("original error")
	format := "wrapping with %s and %d"
	args := []interface{}{"format", 123}

	err := Wrapf(originalErr, format, args...)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "wrapping with format and 123")
	assert.Contains(t, err.Error(), "original error")
	assert.Equal(t, NoType, GetType(err))
}

func TestWrapfWithCustomError(t *testing.T) {
	// Create a custom error first
	originalCustomErr := Kubernetes.New("k8s error")
	format := "wrapping k8s error with %s"
	args := []interface{}{"additional info"}

	err := Wrapf(originalCustomErr, format, args...)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "wrapping k8s error with additional info")
	assert.Contains(t, err.Error(), "k8s error")
	assert.Equal(t, Kubernetes, GetType(err)) // Should preserve the original error type
}

func TestCause(t *testing.T) {
	originalErr := fmt.Errorf("root cause error")
	wrappedErr := errors.Wrap(originalErr, "wrapped once")
	doubleWrappedErr := errors.Wrap(wrappedErr, "wrapped twice")

	cause := Cause(doubleWrappedErr)
	assert.Equal(t, originalErr, cause)
}

func TestGetType(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedType ErrorType
	}{
		{
			name:         "Custom error with Requeue type",
			err:          Requeue.New("requeue error"),
			expectedType: Requeue,
		},
		{
			name:         "Custom error with Kubernetes type",
			err:          Kubernetes.New("k8s error"),
			expectedType: Kubernetes,
		},
		{
			name:         "Custom error with Redis type",
			err:          Redis.New("redis error"),
			expectedType: Redis,
		},
		{
			name:         "Custom error with Cluster type",
			err:          Cluster.New("cluster error"),
			expectedType: Cluster,
		},
		{
			name:         "Custom error with StopRetry type",
			err:          StopRetry.New("stop retry error"),
			expectedType: StopRetry,
		},
		{
			name:         "Custom error with InvalidSpec type",
			err:          InvalidSpec.New("invalid spec error"),
			expectedType: InvalidSpec,
		},
		{
			name:         "Custom error with EnsurePDB type",
			err:          EnsurePDB.New("ensure pdb error"),
			expectedType: EnsurePDB,
		},
		{
			name:         "Custom error with EnsureConfigMap type",
			err:          EnsureConfigMap.New("ensure configmap error"),
			expectedType: EnsureConfigMap,
		},
		{
			name:         "Custom error with EnsureSTS type",
			err:          EnsureSTS.New("ensure sts error"),
			expectedType: EnsureSTS,
		},
		{
			name:         "Custom error with EnsureService type",
			err:          EnsureService.New("ensure service error"),
			expectedType: EnsureService,
		},
		{
			name:         "Standard error (not custom)",
			err:          fmt.Errorf("standard error"),
			expectedType: NoType,
		},
		{
			name:         "Wrapped custom error preserves type",
			err:          Requeue.Wrap(fmt.Errorf("base error"), "wrapped"),
			expectedType: Requeue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errorType := GetType(tt.err)
			assert.Equal(t, tt.expectedType, errorType)
		})
	}
}

func TestErrorChaining(t *testing.T) {
	// Test complex error chaining scenarios
	baseErr := fmt.Errorf("base error")

	// Chain different error types
	k8sErr := Kubernetes.Wrap(baseErr, "k8s wrapper")
	redisErr := Redis.Wrap(k8sErr, "redis wrapper")
	clusterErr := Cluster.Wrap(redisErr, "cluster wrapper")

	// The final error should have the type of the last wrapper
	assert.Equal(t, Cluster, GetType(clusterErr))
	assert.Contains(t, clusterErr.Error(), "cluster wrapper")
	assert.Contains(t, clusterErr.Error(), "redis wrapper")
	assert.Contains(t, clusterErr.Error(), "k8s wrapper")
	assert.Contains(t, clusterErr.Error(), "base error")
}

func TestErrorTypeStringRepresentation(t *testing.T) {
	// Test that error types can be used in switch statements
	err := Requeue.New("test error")

	switch GetType(err) {
	case NoType:
		t.Fatal("Expected Requeue type, got NoType")
	case Requeue:
		// Expected case
	case Kubernetes:
		t.Fatal("Expected Requeue type, got Kubernetes")
	default:
		t.Fatal("Unexpected error type")
	}
}

func TestNilErrorHandling(t *testing.T) {
	// Test GetType with nil error
	assert.Equal(t, NoType, GetType(nil))
}

func TestCustomErrorImplementsError(t *testing.T) {
	// Ensure customError implements the error interface
	var err error = customError{
		errorType:     Requeue,
		originalError: fmt.Errorf("test"),
	}

	assert.Equal(t, "test", err.Error())
}

func TestErrorTypeValues(t *testing.T) {
	// Test that error type values are as expected
	assert.Equal(t, ErrorType(0), NoType)
	assert.Equal(t, ErrorType(1), Requeue)
	assert.Equal(t, ErrorType(2), Kubernetes)
	assert.Equal(t, ErrorType(3), Redis)
	assert.Equal(t, ErrorType(4), Cluster)
	assert.Equal(t, ErrorType(5), StopRetry)
	assert.Equal(t, ErrorType(6), InvalidSpec)
	assert.Equal(t, ErrorType(7), EnsurePDB)
	assert.Equal(t, ErrorType(8), EnsureConfigMap)
	assert.Equal(t, ErrorType(9), EnsureSTS)
	assert.Equal(t, ErrorType(10), EnsureService)
}

func TestErrorTypesAreUnique(t *testing.T) {
	// Ensure all error types have unique values
	errorTypes := []ErrorType{
		NoType, Requeue, Kubernetes, Redis, Cluster,
		StopRetry, InvalidSpec, EnsurePDB, EnsureConfigMap,
		EnsureSTS, EnsureService,
	}

	seen := make(map[ErrorType]bool)
	for _, errorType := range errorTypes {
		assert.False(t, seen[errorType], "Duplicate error type found: %v", errorType)
		seen[errorType] = true
	}

	assert.Equal(t, len(errorTypes), len(seen))
}

func TestComplexErrorScenarios(t *testing.T) {
	// Test realistic error scenarios that might occur in the controller

	t.Run("Kubernetes API error wrapped as Requeue", func(t *testing.T) {
		apiErr := fmt.Errorf("connection refused")
		k8sErr := Kubernetes.Wrap(apiErr, "failed to get pod")
		requeueErr := Requeue.Wrap(k8sErr, "will retry later")

		assert.Equal(t, Requeue, GetType(requeueErr))
		assert.Contains(t, requeueErr.Error(), "will retry later")
		assert.Contains(t, requeueErr.Error(), "failed to get pod")
		assert.Contains(t, requeueErr.Error(), "connection refused")
	})

	t.Run("Redis connection error leading to cluster operation failure", func(t *testing.T) {
		connErr := fmt.Errorf("connection timeout")
		redisErr := Redis.Wrap(connErr, "failed to connect to redis node")
		clusterErr := Cluster.Wrap(redisErr, "unable to perform cluster operation")

		assert.Equal(t, Cluster, GetType(clusterErr))
		assert.Contains(t, clusterErr.Error(), "unable to perform cluster operation")
		assert.Contains(t, clusterErr.Error(), "failed to connect to redis node")
		assert.Contains(t, clusterErr.Error(), "connection timeout")
	})

	t.Run("Invalid spec error that should stop retry", func(t *testing.T) {
		specErr := InvalidSpec.New("master size must be at least 3")
		stopErr := StopRetry.Wrap(specErr, "invalid configuration detected")

		assert.Equal(t, StopRetry, GetType(stopErr))
		assert.Contains(t, stopErr.Error(), "invalid configuration detected")
		assert.Contains(t, stopErr.Error(), "master size must be at least 3")
	})
}

func TestErrorMessageFormatting(t *testing.T) {
	// Test that error messages are properly formatted
	baseErr := fmt.Errorf("base error")

	// Test simple wrap
	wrappedErr := Kubernetes.Wrap(baseErr, "kubernetes error")
	assert.Contains(t, wrappedErr.Error(), "kubernetes error")
	assert.Contains(t, wrappedErr.Error(), "base error")

	// Test formatted wrap
	formattedErr := Redis.Wrapf(baseErr, "redis error on node %s:%d", "10.0.0.1", 6379)
	assert.Contains(t, formattedErr.Error(), "redis error on node 10.0.0.1:6379")
	assert.Contains(t, formattedErr.Error(), "base error")
}

func TestErrorTypeConsistency(t *testing.T) {
	// Test that error types remain consistent through wrapping
	originalErr := Requeue.New("original error")
	wrappedErr := Wrap(originalErr, "wrapped")
	doubleWrappedErr := Wrapf(wrappedErr, "double wrapped with %s", "format")

	assert.Equal(t, Requeue, GetType(originalErr))
	assert.Equal(t, Requeue, GetType(wrappedErr))
	assert.Equal(t, Requeue, GetType(doubleWrappedErr))
}
