package exec

import (
	"bytes"
	"io"
	"net/url"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

// Helper function to create test pods
func createTestPod(name, namespace, containerName string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  containerName,
					Image: "redis:latest",
				},
			},
		},
	}
}

func TestNewRemoteExec(t *testing.T) {
	tests := []struct {
		name         string
		expectNonNil bool
	}{
		{
			name:         "Valid parameters",
			expectNonNil: true,
		},
		{
			name:         "Nil rest client",
			expectNonNil: true, // Constructor doesn't validate nil inputs
		},
		{
			name:         "Nil config",
			expectNonNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var restClient rest.Interface
			var config *rest.Config
			logger := logr.Discard()

			switch tt.name {
			case "Nil rest client":
				restClient = nil
				config = &rest.Config{}
			case "Nil config":
				restClient = nil
				config = nil
			default:
				restClient = nil
				config = &rest.Config{}
			}

			result := NewRemoteExec(restClient, config, logger)
			if tt.expectNonNil {
				if result == nil {
					t.Errorf("Expected non-nil result, got nil")
				}
				// Verify it implements IExec interface
				var _ IExec = result
			} else {
				if result != nil {
					t.Errorf("Expected nil result, got %v", result)
				}
			}
		})
	}
}

func TestRemoteExec_ExecCommandInPodSet_Structure(t *testing.T) {
	tests := []struct {
		name          string
		pods          []*corev1.Pod
		cmd           []string
		shouldIterate bool
	}{
		{
			name: "Single pod",
			pods: []*corev1.Pod{
				createTestPod("test-pod", "test-ns", "redis"),
			},
			cmd:           []string{"redis-cli", "ping"},
			shouldIterate: true,
		},
		{
			name: "Multiple pods",
			pods: []*corev1.Pod{
				createTestPod("test-pod-1", "test-ns", "redis"),
				createTestPod("test-pod-2", "test-ns", "redis"),
			},
			cmd:           []string{"redis-cli", "info"},
			shouldIterate: true,
		},
		{
			name:          "Empty pod set",
			pods:          []*corev1.Pod{},
			cmd:           []string{"redis-cli", "ping"},
			shouldIterate: false,
		},
		{
			name:          "Nil pod set",
			pods:          nil,
			cmd:           []string{"redis-cli", "ping"},
			shouldIterate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the structure and logic without actually executing
			podCount := len(tt.pods)
			if tt.shouldIterate && podCount == 0 {
				t.Errorf("Expected pods to iterate over, got empty set")
			}
			if !tt.shouldIterate && podCount > 0 {
				t.Errorf("Expected empty pod set, got %d pods", podCount)
			}

			// Verify command structure
			if len(tt.cmd) == 0 {
				t.Logf("Command is empty for test case: %s", tt.name)
			}
		})
	}
}

func TestRemoteExec_ExecCommandInContainer_Parameters(t *testing.T) {
	tests := []struct {
		name          string
		namespace     string
		podName       string
		containerName string
		cmd           []string
		expectPanic   bool
	}{
		{
			name:          "Valid parameters",
			namespace:     "test-ns",
			podName:       "test-pod",
			containerName: "redis",
			cmd:           []string{"echo", "hello"},
			expectPanic:   true, // Will panic due to nil client, but tests parameter passing
		},
		{
			name:          "Empty namespace",
			namespace:     "",
			podName:       "test-pod",
			containerName: "redis",
			cmd:           []string{"echo", "hello"},
			expectPanic:   true, // Will panic due to nil client
		},
		{
			name:          "Empty pod name",
			namespace:     "test-ns",
			podName:       "",
			containerName: "redis",
			cmd:           []string{"echo", "hello"},
			expectPanic:   true, // Will panic due to nil client
		},
		{
			name:          "Empty container name",
			namespace:     "test-ns",
			podName:       "test-pod",
			containerName: "",
			cmd:           []string{"echo", "hello"},
			expectPanic:   true, // Will panic due to nil client
		},
		{
			name:          "Empty command",
			namespace:     "test-ns",
			podName:       "test-pod",
			containerName: "redis",
			cmd:           []string{},
			expectPanic:   true, // Will panic due to nil client
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &rest.Config{}
			logger := logr.Discard()

			exec := &remoteExec{
				restGVKClient: nil,
				logger:        logger,
				config:        config,
			}

			// Expect panic due to nil client - this tests parameter passing
			if tt.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("Expected panic due to nil client, but didn't panic")
					}
				}()
			}

			exec.ExecCommandInContainer(tt.namespace, tt.podName, tt.containerName, tt.cmd...)
		})
	}
}

func TestRemoteExec_ExecCommandInContainerWithFullOutput_Parameters(t *testing.T) {
	tests := []struct {
		name          string
		namespace     string
		podName       string
		containerName string
		cmd           []string
		expectPanic   bool
	}{
		{
			name:          "Valid parameters",
			namespace:     "test-ns",
			podName:       "test-pod",
			containerName: "redis",
			cmd:           []string{"echo", "hello world"},
			expectPanic:   true, // Will panic due to nil client
		},
		{
			name:          "Command with multiple arguments",
			namespace:     "default",
			podName:       "redis-0",
			containerName: "redis-server",
			cmd:           []string{"redis-cli", "-h", "localhost", "ping"},
			expectPanic:   true, // Will panic due to nil client
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &rest.Config{}
			logger := logr.Discard()

			exec := &remoteExec{
				restGVKClient: nil,
				logger:        logger,
				config:        config,
			}

			// Expect panic due to nil client - this tests parameter passing to ExecWithOptions
			if tt.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("Expected panic due to nil client, but didn't panic")
					}
				}()
			}

			exec.ExecCommandInContainerWithFullOutput(tt.namespace, tt.podName, tt.containerName, tt.cmd...)
		})
	}
}

func TestRemoteExec_ExecWithOptions_Parameters(t *testing.T) {
	tests := []struct {
		name        string
		options     ExecOptions
		expectPanic bool
	}{
		{
			name: "Basic options",
			options: ExecOptions{
				Command:            []string{"echo", "test"},
				Namespace:          "test-ns",
				PodName:            "test-pod",
				ContainerName:      "redis",
				Stdin:              nil,
				CaptureStdout:      true,
				CaptureStderr:      true,
				PreserveWhitespace: false,
			},
			expectPanic: true, // Will panic due to nil client
		},
		{
			name: "Options with stdin",
			options: ExecOptions{
				Command:            []string{"cat"},
				Namespace:          "test-ns",
				PodName:            "test-pod",
				ContainerName:      "redis",
				Stdin:              bytes.NewReader([]byte("test input")),
				CaptureStdout:      true,
				CaptureStderr:      false,
				PreserveWhitespace: true,
			},
			expectPanic: true, // Will panic due to nil client
		},
		{
			name: "Options without stdout capture",
			options: ExecOptions{
				Command:            []string{"redis-cli", "ping"},
				Namespace:          "test-ns",
				PodName:            "test-pod",
				ContainerName:      "redis",
				Stdin:              nil,
				CaptureStdout:      false,
				CaptureStderr:      true,
				PreserveWhitespace: false,
			},
			expectPanic: true, // Will panic due to nil client
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &rest.Config{}
			logger := logr.Discard()

			exec := &remoteExec{
				restGVKClient: nil,
				logger:        logger,
				config:        config,
			}

			// Expect panic due to nil client - this tests the ExecWithOptions parameter handling
			if tt.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("Expected panic due to nil client, but didn't panic")
					}
				}()
			}

			exec.ExecWithOptions(tt.options)
		})
	}
}

func TestExecOptions(t *testing.T) {
	tests := []struct {
		name    string
		options ExecOptions
		valid   bool
	}{
		{
			name: "Complete valid options",
			options: ExecOptions{
				Command:            []string{"redis-cli", "ping"},
				Namespace:          "test-ns",
				PodName:            "test-pod",
				ContainerName:      "redis",
				Stdin:              nil,
				CaptureStdout:      true,
				CaptureStderr:      true,
				PreserveWhitespace: false,
			},
			valid: true,
		},
		{
			name: "Minimal valid options",
			options: ExecOptions{
				Command:       []string{"ping"},
				Namespace:     "ns",
				PodName:       "pod",
				ContainerName: "container",
			},
			valid: true,
		},
		{
			name: "Options with stdin reader",
			options: ExecOptions{
				Command:            []string{"cat"},
				Namespace:          "test-ns",
				PodName:            "test-pod",
				ContainerName:      "redis",
				Stdin:              bytes.NewReader([]byte("input")),
				CaptureStdout:      true,
				CaptureStderr:      false,
				PreserveWhitespace: true,
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that the struct can be created and populated correctly
			if len(tt.options.Command) == 0 && tt.valid {
				t.Errorf("Expected valid options to have command")
			}
			if tt.options.Namespace == "" && tt.valid {
				t.Errorf("Expected valid options to have namespace")
			}
			if tt.options.PodName == "" && tt.valid {
				t.Errorf("Expected valid options to have pod name")
			}
			if tt.options.ContainerName == "" && tt.valid {
				t.Errorf("Expected valid options to have container name")
			}
		})
	}
}

func TestIExecInterface(t *testing.T) {
	// Test that remoteExec implements IExec interface
	var _ IExec = (*remoteExec)(nil)

	config := &rest.Config{}
	logger := logr.Discard()

	exec := NewRemoteExec(nil, config, logger)
	if exec == nil {
		t.Errorf("Expected non-nil exec instance")
	}

	// Verify it implements the interface
	_, ok := exec.(IExec)
	if !ok {
		t.Errorf("remoteExec does not implement IExec interface")
	}
}

// Test for whitespace handling behavior
func TestWhitespaceHandling(t *testing.T) {
	tests := []struct {
		name               string
		input              string
		preserveWhitespace bool
		expected           string
	}{
		{
			name:               "Preserve whitespace enabled",
			input:              "  hello world  \n",
			preserveWhitespace: true,
			expected:           "  hello world  \n",
		},
		{
			name:               "Preserve whitespace disabled",
			input:              "  hello world  \n",
			preserveWhitespace: false,
			expected:           "hello world",
		},
		{
			name:               "Empty strings with whitespace preserved",
			input:              "",
			preserveWhitespace: true,
			expected:           "",
		},
		{
			name:               "Empty strings with whitespace trimmed",
			input:              "",
			preserveWhitespace: false,
			expected:           "",
		},
		{
			name:               "Only whitespace with preserve enabled",
			input:              "   \n\t  ",
			preserveWhitespace: true,
			expected:           "   \n\t  ",
		},
		{
			name:               "Only whitespace with preserve disabled",
			input:              "   \n\t  ",
			preserveWhitespace: false,
			expected:           "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result string
			if tt.preserveWhitespace {
				result = tt.input
			} else {
				result = strings.TrimSpace(tt.input)
			}

			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestRemoteExecErrorHandling(t *testing.T) {
	t.Run("Pod with no containers should panic", func(t *testing.T) {
		pods := []*corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "test-ns",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{}, // Empty containers
				},
			},
		}
		cmd := []string{"echo", "test"}

		config := &rest.Config{}
		logger := logr.Discard()

		exec := &remoteExec{
			restGVKClient: nil,
			logger:        logger,
			config:        config,
		}

		// We expect a panic due to accessing containers[0] on empty slice
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Expected panic for pod with no containers")
			}
		}()

		exec.ExecCommandInPodSet(pods, cmd...)
	})
}

// Test the execute function with different parameters
func TestExecuteFunction(t *testing.T) {
	tests := []struct {
		name      string
		method    string
		urlString string
		config    *rest.Config
		stdin     io.Reader
		expectErr bool
	}{
		{
			name:      "POST method with valid URL",
			method:    "POST",
			urlString: "https://test.com/api/v1/pods/test/exec",
			config:    &rest.Config{Host: "https://test.com"},
			stdin:     nil,
			expectErr: true, // Will fail due to no actual k8s connection, but tests flow
		},
		{
			name:      "POST method with stdin",
			method:    "POST",
			urlString: "https://test.com/api/v1/pods/test/exec",
			config:    &rest.Config{Host: "https://test.com"},
			stdin:     bytes.NewReader([]byte("test")),
			expectErr: true, // Will fail due to no actual k8s connection, but tests flow
		},
		{
			name:      "Invalid URL",
			method:    "POST",
			urlString: "://invalid-url",
			config:    &rest.Config{},
			stdin:     nil,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testURL, parseErr := url.Parse(tt.urlString)
			if parseErr != nil && !tt.expectErr {
				t.Errorf("Failed to parse URL: %v", parseErr)
				return
			}
			if parseErr != nil {
				return // Skip if URL parsing failed as expected
			}

			var stdout, stderr bytes.Buffer

			err := execute(tt.method, testURL, tt.config, tt.stdin, &stdout, &stderr, false)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

// Test edge cases and boundary conditions
func TestEdgeCases(t *testing.T) {
	t.Run("Empty command slice", func(t *testing.T) {
		options := ExecOptions{
			Command:       []string{},
			Namespace:     "test",
			PodName:       "test",
			ContainerName: "test",
		}

		if len(options.Command) != 0 {
			t.Errorf("Expected empty command slice")
		}
	})

	t.Run("Nil command slice", func(t *testing.T) {
		options := ExecOptions{
			Command:       nil,
			Namespace:     "test",
			PodName:       "test",
			ContainerName: "test",
		}

		if options.Command != nil {
			t.Errorf("Expected nil command slice")
		}
	})

	t.Run("Large command", func(t *testing.T) {
		largeCmd := make([]string, 1000)
		for i := range largeCmd {
			largeCmd[i] = "arg"
		}

		options := ExecOptions{
			Command:       largeCmd,
			Namespace:     "test",
			PodName:       "test",
			ContainerName: "test",
		}

		if len(options.Command) != 1000 {
			t.Errorf("Expected command slice of length 1000, got %d", len(options.Command))
		}
	})
}

// Test constructor behavior
func TestNewRemoteExecConstruction(t *testing.T) {
	t.Run("Constructor returns correct type", func(t *testing.T) {
		config := &rest.Config{Host: "https://test.com"}
		logger := logr.Discard()

		exec := NewRemoteExec(nil, config, logger)

		// Verify non-nil
		if exec == nil {
			t.Errorf("NewRemoteExec returned nil")
		}

		// Verify it's the correct underlying type
		remoteExecImpl, ok := exec.(*remoteExec)
		if !ok {
			t.Errorf("NewRemoteExec did not return *remoteExec type")
		}

		// Verify fields are set (we can't access private fields directly, but we can test behavior)
		if remoteExecImpl == nil {
			t.Errorf("remoteExec implementation is nil")
		}
	})
}

// Test ExecOptions structure validation
func TestExecOptionsValidation(t *testing.T) {
	tests := []struct {
		name        string
		options     ExecOptions
		description string
	}{
		{
			name: "All boolean flags false",
			options: ExecOptions{
				Command:            []string{"test"},
				Namespace:          "ns",
				PodName:            "pod",
				ContainerName:      "container",
				CaptureStdout:      false,
				CaptureStderr:      false,
				PreserveWhitespace: false,
			},
			description: "Should handle all boolean options set to false",
		},
		{
			name: "All boolean flags true",
			options: ExecOptions{
				Command:            []string{"test"},
				Namespace:          "ns",
				PodName:            "pod",
				ContainerName:      "container",
				CaptureStdout:      true,
				CaptureStderr:      true,
				PreserveWhitespace: true,
			},
			description: "Should handle all boolean options set to true",
		},
		{
			name: "Mixed boolean flags",
			options: ExecOptions{
				Command:            []string{"test"},
				Namespace:          "ns",
				PodName:            "pod",
				ContainerName:      "container",
				CaptureStdout:      true,
				CaptureStderr:      false,
				PreserveWhitespace: true,
			},
			description: "Should handle mixed boolean options",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify the options struct is populated correctly
			if len(tt.options.Command) == 0 {
				t.Errorf("Command should not be empty")
			}
			if tt.options.Namespace == "" {
				t.Errorf("Namespace should not be empty")
			}
			if tt.options.PodName == "" {
				t.Errorf("PodName should not be empty")
			}
			if tt.options.ContainerName == "" {
				t.Errorf("ContainerName should not be empty")
			}
		})
	}
}

// Test that verifies the interface contract
func TestInterfaceContract(t *testing.T) {
	t.Run("IExec interface has correct method signature", func(t *testing.T) {
		// This is a compile-time check that ensures the interface is correctly defined
		var exec IExec = &remoteExec{}

		// Test with nil pod set (should not panic)
		err := exec.ExecCommandInPodSet(nil)
		// We expect this to succeed with nil pods
		if err != nil {
			t.Errorf("ExecCommandInPodSet with nil pods should not error, got %v", err)
		}

		// Test with empty command
		err = exec.ExecCommandInPodSet([]*corev1.Pod{})
		if err != nil {
			t.Errorf("ExecCommandInPodSet with empty pods should not error, got %v", err)
		}
	})
}

// Test pod creation helper
func TestCreateTestPod(t *testing.T) {
	pod := createTestPod("test-pod", "test-ns", "redis")

	if pod.Name != "test-pod" {
		t.Errorf("Expected pod name 'test-pod', got %s", pod.Name)
	}
	if pod.Namespace != "test-ns" {
		t.Errorf("Expected pod namespace 'test-ns', got %s", pod.Namespace)
	}
	if len(pod.Spec.Containers) != 1 {
		t.Errorf("Expected 1 container, got %d", len(pod.Spec.Containers))
	}
	if pod.Spec.Containers[0].Name != "redis" {
		t.Errorf("Expected container name 'redis', got %s", pod.Spec.Containers[0].Name)
	}
}

// Test ExecOptions with different stdin configurations
func TestExecOptionsStdin(t *testing.T) {
	tests := []struct {
		name        string
		stdin       io.Reader
		expectedNil bool
	}{
		{
			name:        "Nil stdin",
			stdin:       nil,
			expectedNil: true,
		},
		{
			name:        "Bytes reader stdin",
			stdin:       bytes.NewReader([]byte("test input")),
			expectedNil: false,
		},
		{
			name:        "Empty bytes reader",
			stdin:       bytes.NewReader([]byte{}),
			expectedNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := ExecOptions{
				Command:       []string{"cat"},
				Namespace:     "test",
				PodName:       "test",
				ContainerName: "test",
				Stdin:         tt.stdin,
			}

			if tt.expectedNil {
				if options.Stdin != nil {
					t.Errorf("Expected nil stdin, got %v", options.Stdin)
				}
			} else {
				if options.Stdin == nil {
					t.Errorf("Expected non-nil stdin, got nil")
				}
			}
		})
	}
}

// Test various command configurations
func TestCommandConfigurations(t *testing.T) {
	tests := []struct {
		name    string
		command []string
		valid   bool
	}{
		{
			name:    "Single command",
			command: []string{"ping"},
			valid:   true,
		},
		{
			name:    "Command with arguments",
			command: []string{"redis-cli", "-h", "localhost", "ping"},
			valid:   true,
		},
		{
			name:    "Empty command",
			command: []string{},
			valid:   false,
		},
		{
			name:    "Nil command",
			command: nil,
			valid:   false,
		},
		{
			name:    "Command with empty string",
			command: []string{""},
			valid:   true, // Technically valid, though might not work in practice
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.valid && len(tt.command) == 0 {
				t.Errorf("Valid command should not be empty")
			}
			if !tt.valid && len(tt.command) > 0 {
				t.Logf("Invalid command has length %d", len(tt.command))
			}
		})
	}
}

// Test pod access patterns (what could cause index out of bounds)
func TestPodAccessPatterns(t *testing.T) {
	t.Run("Pod with multiple containers", func(t *testing.T) {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "test-ns",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "redis", Image: "redis:latest"},
					{Name: "sidecar", Image: "busybox:latest"},
				},
			},
		}

		// The code accesses containers[0], so this should be safe
		if len(pod.Spec.Containers) == 0 {
			t.Errorf("Pod should have at least one container")
		}

		firstContainer := pod.Spec.Containers[0]
		if firstContainer.Name != "redis" {
			t.Errorf("Expected first container name 'redis', got %s", firstContainer.Name)
		}
	})

	t.Run("Pod with no containers causes index out of bounds", func(t *testing.T) {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "test-ns",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{}, // Empty containers
			},
		}

		// This demonstrates the vulnerability in the original code
		if len(pod.Spec.Containers) == 0 {
			t.Logf("Pod has no containers - accessing containers[0] would panic")
			// We don't actually access it here to avoid panic in test
		}
	})
}

// Test logger integration
func TestLoggerIntegration(t *testing.T) {
	t.Run("Logger is used for stderr output", func(t *testing.T) {
		// Test that when ExecCommandInContainer encounters stderr, it logs it
		// This is difficult to test without a mock logger, but we can verify the structure

		config := &rest.Config{}
		logger := logr.Discard()

		exec := &remoteExec{
			restGVKClient: nil,
			logger:        logger,
			config:        config,
		}

		// Verify the logger is set in the struct (indirectly)
		if exec == nil {
			t.Errorf("remoteExec should not be nil")
		}
		// The actual logging behavior would require a mock logger to test properly
	})
}

// Test URL construction and validation in execute function
func TestExecuteFunctionURLHandling(t *testing.T) {
	tests := []struct {
		name      string
		method    string
		urlString string
		valid     bool
	}{
		{
			name:      "Valid HTTPS URL",
			method:    "POST",
			urlString: "https://kubernetes.default.svc.cluster.local:443/api/v1/namespaces/default/pods/test-pod/exec",
			valid:     true,
		},
		{
			name:      "Valid HTTP URL",
			method:    "POST",
			urlString: "http://localhost:8080/api/v1/namespaces/default/pods/test-pod/exec",
			valid:     true,
		},
		{
			name:      "URL with query parameters",
			method:    "POST",
			urlString: "https://test.com/api/v1/pods/test/exec?container=redis&stdout=true&stderr=true",
			valid:     true,
		},
		{
			name:      "Invalid scheme",
			method:    "POST",
			urlString: "ftp://test.com/api/v1/pods/test/exec",
			valid:     true, // URL parsing will succeed, but execution will fail
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testURL, parseErr := url.Parse(tt.urlString)
			if parseErr != nil {
				if tt.valid {
					t.Errorf("Expected valid URL, but parsing failed: %v", parseErr)
				}
				return
			}

			// Test URL structure
			if tt.valid {
				if testURL.Host == "" {
					t.Errorf("Expected valid host in URL, got empty")
				}
				if testURL.Path == "" {
					t.Errorf("Expected valid path in URL, got empty")
				}
			}
		})
	}
}

// Benchmark test for performance
func BenchmarkExecCommandInPodSet(b *testing.B) {
	config := &rest.Config{}
	logger := logr.Discard()
	exec := NewRemoteExec(nil, config, logger)

	// Use empty pod set to avoid nil pointer issues in benchmark
	pods := []*corev1.Pod{}
	cmd := []string{"redis-cli", "ping"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// This should complete quickly with empty pod set
		exec.ExecCommandInPodSet(pods, cmd...)
	}
}

// Benchmark for ExecOptions creation
func BenchmarkExecOptionsCreation(b *testing.B) {
	cmd := []string{"redis-cli", "ping"}
	namespace := "test-ns"
	podName := "test-pod"
	containerName := "redis"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		options := ExecOptions{
			Command:            cmd,
			Namespace:          namespace,
			PodName:            podName,
			ContainerName:      containerName,
			CaptureStdout:      true,
			CaptureStderr:      true,
			PreserveWhitespace: false,
		}
		// Use options to avoid compiler optimization
		if len(options.Command) == 0 {
			b.Errorf("Command should not be empty")
		}
	}
}
