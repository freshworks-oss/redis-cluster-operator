package redisutil

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/freshworks/redis-cluster-operator/utils/testutil"
	redis "github.com/redis/go-redis/v9"
)

// TestNewAdminConnections tests the NewAdminConnections constructor
func TestNewAdminConnections(t *testing.T) {
	logger := testutil.NewTestLogger()
	ctx := context.Background()

	tests := []struct {
		name        string
		addrs       []string
		options     *AdminOptions
		expectError bool
	}{
		{
			name:        "Create with empty addresses",
			addrs:       []string{},
			options:     nil,
			expectError: false,
		},
		{
			name:        "Create with nil options",
			addrs:       []string{"127.0.0.1:6379"},
			options:     nil,
			expectError: false,
		},
		{
			name:  "Create with custom options",
			addrs: []string{"127.0.0.1:6379", "127.0.0.1:6380"},
			options: &AdminOptions{
				ConnectionTimeout: 5 * time.Second,
				ClientName:        "test-client",
				Password:          "test-password",
			},
			expectError: false,
		},
		{
			name:  "Create with zero timeout in options",
			addrs: []string{"127.0.0.1:6379"},
			options: &AdminOptions{
				ConnectionTimeout: 0, // Should use default
				ClientName:        "zero-timeout-client",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: This will try to actually connect, which may fail in test environment
			// In a real test, we'd need to mock the redis connection
			conn := NewAdminConnections(tt.addrs, tt.options, logger, ctx)

			if conn == nil && !tt.expectError {
				t.Error("Expected non-nil AdminConnections")
			}

			// Test interface compliance
			var _ IAdminConnections = conn

			// Test that we can call methods without panicking
			allClients := conn.GetAll()
			if allClients == nil {
				t.Error("GetAll should return a non-nil map")
			}

			// Test password retrieval
			if tt.options != nil && tt.options.Password != "" {
				password, hasAuth := conn.GetAUTH()
				if !hasAuth {
					t.Error("Should have auth when password is set")
				}
				if password != tt.options.Password {
					t.Errorf("Expected password %s, got %s", tt.options.Password, password)
				}
			} else {
				_, hasAuth := conn.GetAUTH()
				if hasAuth {
					t.Error("Should not have auth when no password is set")
				}
			}
		})
	}
}

// TestAdminConnections_GetAUTH tests the GetAUTH method
func TestAdminConnections_GetAUTH(t *testing.T) {
	logger := testutil.NewTestLogger()
	ctx := context.Background()

	tests := []struct {
		name             string
		password         string
		expectedAuth     bool
		expectedPassword string
	}{
		{
			name:             "No password set",
			password:         "",
			expectedAuth:     false,
			expectedPassword: "",
		},
		{
			name:             "Password set",
			password:         "secret123",
			expectedAuth:     true,
			expectedPassword: "secret123",
		},
		{
			name:             "Long password",
			password:         "very-long-password-with-special-chars!@#$%^&*()",
			expectedAuth:     true,
			expectedPassword: "very-long-password-with-special-chars!@#$%^&*()",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &AdminOptions{
				Password: tt.password,
			}

			conn := NewAdminConnections([]string{}, options, logger, ctx)
			password, hasAuth := conn.GetAUTH()

			if hasAuth != tt.expectedAuth {
				t.Errorf("Expected hasAuth %v, got %v", tt.expectedAuth, hasAuth)
			}

			if password != tt.expectedPassword {
				t.Errorf("Expected password %s, got %s", tt.expectedPassword, password)
			}
		})
	}
}

// TestAdminConnections_GetAll tests the GetAll method
func TestAdminConnections_GetAll(t *testing.T) {
	logger := testutil.NewTestLogger()
	ctx := context.Background()

	conn := NewAdminConnections([]string{}, nil, logger, ctx)

	t.Run("Empty connections", func(t *testing.T) {
		clients := conn.GetAll()
		if clients == nil {
			t.Error("GetAll should return non-nil map")
		}
		if len(clients) != 0 {
			t.Errorf("Expected empty map, got %d clients", len(clients))
		}
	})

	// Note: Testing with actual connections would require running Redis instances
	// In a production test suite, we'd use testcontainers or similar
}

// TestAdminConnections_GetSelected tests the GetSelected method
func TestAdminConnections_GetSelected(t *testing.T) {
	logger := testutil.NewTestLogger()
	ctx := context.Background()

	conn := NewAdminConnections([]string{}, nil, logger, ctx)

	tests := []struct {
		name          string
		requestAddrs  []string
		expectedCount int
	}{
		{
			name:          "Empty address list",
			requestAddrs:  []string{},
			expectedCount: 0,
		},
		{
			name:          "Non-existing addresses",
			requestAddrs:  []string{"127.0.0.1:6379", "127.0.0.1:6380"},
			expectedCount: 0, // No connections exist
		},
		{
			name:          "Nil address list",
			requestAddrs:  nil,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selected := conn.GetSelected(tt.requestAddrs)
			if selected == nil {
				t.Error("GetSelected should return non-nil map")
			}
			if len(selected) != tt.expectedCount {
				t.Errorf("Expected %d clients, got %d", tt.expectedCount, len(selected))
			}
		})
	}
}

// TestAdminConnections_Reset tests the Reset method
func TestAdminConnections_Reset(t *testing.T) {
	logger := testutil.NewTestLogger()
	ctx := context.Background()

	conn := NewAdminConnections([]string{}, nil, logger, ctx)

	t.Run("Reset empty connections", func(t *testing.T) {
		// Should not panic
		conn.Reset(ctx)

		clients := conn.GetAll()
		if len(clients) != 0 {
			t.Error("After reset, should have no clients")
		}
	})
}

// TestAdminConnections_Remove tests the Remove method
func TestAdminConnections_Remove(t *testing.T) {
	logger := testutil.NewTestLogger()
	ctx := context.Background()

	conn := NewAdminConnections([]string{}, nil, logger, ctx)

	t.Run("Remove non-existing connection", func(t *testing.T) {
		// Should not panic
		conn.Remove(ctx, "non-existing:6379")

		clients := conn.GetAll()
		if len(clients) != 0 {
			t.Error("Should still have no clients after removing non-existing")
		}
	})
}

// TestAdminConnections_GetRandom tests the GetRandom method
func TestAdminConnections_GetRandom(t *testing.T) {
	logger := testutil.NewTestLogger()
	ctx := context.Background()

	conn := NewAdminConnections([]string{}, nil, logger, ctx)

	t.Run("GetRandom with no connections", func(t *testing.T) {
		client, err := conn.GetRandom()

		if err == nil {
			t.Error("Expected error when no connections exist")
		}

		if client != nil {
			t.Error("Expected nil client when error occurs")
		}

		if err.Error() != ErrNotFound {
			t.Errorf("Expected ErrNotFound, got %v", err)
		}
	})
}

// TestAdminConnections_GetDifferentFrom tests the GetDifferentFrom method
func TestAdminConnections_GetDifferentFrom(t *testing.T) {
	logger := testutil.NewTestLogger()
	ctx := context.Background()

	conn := NewAdminConnections([]string{}, nil, logger, ctx)

	t.Run("GetDifferentFrom with no connections", func(t *testing.T) {
		client, err := conn.GetDifferentFrom("any-addr:6379")

		if err == nil {
			t.Error("Expected error when no connections exist")
		}

		if client != nil {
			t.Error("Expected nil client when error occurs")
		}

		if err.Error() != ErrNotFound {
			t.Errorf("Expected ErrNotFound, got %v", err)
		}
	})
}

// TestAdminConnections_ValidateResp tests the ValidateResp method
func TestAdminConnections_ValidateResp(t *testing.T) {
	logger := testutil.NewTestLogger()
	ctx := context.Background()

	conn := NewAdminConnections([]string{}, nil, logger, ctx)
	addr := "127.0.0.1:6379"
	errMessage := "Test command failed"

	tests := []struct {
		name          string
		resp          *redis.Cmd
		expectError   bool
		errorContains string
	}{
		{
			name:          "Nil response",
			resp:          nil,
			expectError:   true,
			errorContains: "unable to connect to node",
		},
		{
			name: "Response with error",
			resp: func() *redis.Cmd {
				cmd := redis.NewCmd(ctx)
				cmd.SetErr(errors.New("redis error"))
				return cmd
			}(),
			expectError:   true,
			errorContains: "unexpected error on node",
		},
		{
			name: "Successful response",
			resp: func() *redis.Cmd {
				cmd := redis.NewCmd(ctx)
				cmd.SetVal("OK")
				return cmd
			}(),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := conn.ValidateResp(ctx, tt.resp, addr, errMessage)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// TestAdminConnections_ValidatePipeResp tests the ValidatePipeResp method
func TestAdminConnections_ValidatePipeResp(t *testing.T) {
	logger := testutil.NewTestLogger()
	ctx := context.Background()

	conn := NewAdminConnections([]string{}, nil, logger, ctx)
	addr := "127.0.0.1:6379"
	errMessage := "Test pipe command failed"

	tests := []struct {
		name     string
		resp     []redis.Cmder
		err      error
		expected bool
	}{
		{
			name:     "Nil response",
			resp:     nil,
			err:      nil,
			expected: false,
		},
		{
			name:     "Valid response with no error",
			resp:     []redis.Cmder{redis.NewCmd(ctx)},
			err:      nil,
			expected: true,
		},
		{
			name:     "Valid response with error",
			resp:     []redis.Cmder{redis.NewCmd(ctx)},
			err:      errors.New("pipe error"),
			expected: true, // Still returns true if not a network error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := conn.ValidatePipeResp(ctx, tt.resp, addr, tt.err, errMessage)

			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestAdminConnections_AddAll tests the AddAll method
func TestAdminConnections_AddAll(t *testing.T) {
	logger := testutil.NewTestLogger()
	ctx := context.Background()

	conn := NewAdminConnections([]string{}, nil, logger, ctx)

	tests := []struct {
		name  string
		addrs []string
	}{
		{
			name:  "Add empty list",
			addrs: []string{},
		},
		{
			name:  "Add nil list",
			addrs: nil,
		},
		{
			name:  "Add single address",
			addrs: []string{"127.0.0.1:6379"},
		},
		{
			name:  "Add multiple addresses",
			addrs: []string{"127.0.0.1:6379", "127.0.0.1:6380", "127.0.0.1:6381"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			conn.AddAll(ctx, tt.addrs)

			// Note: In a real test environment, we'd verify the connections were attempted
			// But since we're not running actual Redis instances, we just verify no panic
		})
	}
}

// TestAdminConnections_ReplaceAll tests the ReplaceAll method
func TestAdminConnections_ReplaceAll(t *testing.T) {
	logger := testutil.NewTestLogger()
	ctx := context.Background()

	conn := NewAdminConnections([]string{}, nil, logger, ctx)

	t.Run("Replace with new addresses", func(t *testing.T) {
		newAddrs := []string{"127.0.0.1:7000", "127.0.0.1:7001"}

		// Should not panic
		conn.ReplaceAll(ctx, newAddrs)

		// Verify connections were reset
		clients := conn.GetAll()
		if clients == nil {
			t.Error("GetAll should return non-nil map after ReplaceAll")
		}
	})
}

// TestAdminConnections_Integration tests integration scenarios
func TestAdminConnections_Integration(t *testing.T) {
	logger := testutil.NewTestLogger()
	ctx := context.Background()

	t.Run("Complete workflow", func(t *testing.T) {
		options := &AdminOptions{
			ConnectionTimeout: 1 * time.Second,
			ClientName:        "test-integration",
			Password:          "integration-password",
		}

		conn := NewAdminConnections([]string{}, options, logger, ctx)

		// Test password functionality
		password, hasAuth := conn.GetAUTH()
		if !hasAuth || password != "integration-password" {
			t.Error("Password should be set correctly")
		}

		// Test empty state
		if len(conn.GetAll()) != 0 {
			t.Error("Should start with no connections")
		}

		// Test GetRandom on empty
		_, err := conn.GetRandom()
		if err == nil {
			t.Error("Should error when getting random from empty connections")
		}

		// Test GetDifferentFrom on empty
		_, err = conn.GetDifferentFrom("any")
		if err == nil {
			t.Error("Should error when getting different from empty connections")
		}

		// Test operations that should not panic
		conn.Reset(ctx)
		conn.Remove(ctx, "non-existing")
		conn.AddAll(ctx, []string{})
		conn.ReplaceAll(ctx, []string{})

		// Test GetSelected
		selected := conn.GetSelected([]string{"addr1", "addr2"})
		if selected == nil || len(selected) != 0 {
			t.Error("GetSelected should return empty map for non-existing addresses")
		}
	})
}

// TestNewClient tests the NewClient function
func TestNewClient(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		addr        string
		password    string
		timeout     time.Duration
		expectError bool
	}{
		{
			name:        "Valid parameters",
			addr:        "127.0.0.1:6379",
			password:    "test",
			timeout:     2 * time.Second,
			expectError: true, // Will fail due to no Redis server
		},
		{
			name:        "Empty address",
			addr:        "",
			password:    "",
			timeout:     2 * time.Second,
			expectError: true,
		},
		{
			name:        "Invalid address format",
			addr:        "invalid-address",
			password:    "",
			timeout:     2 * time.Second,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(ctx, tt.addr, tt.password, tt.timeout, map[string]string{})

			if tt.expectError {
				// We expect this to fail since there's no Redis server running
				// Just verify we don't panic and handle the error gracefully
				if err == nil && client == nil {
					t.Error("Expected either error or client, got both nil")
				}
			}

			// Clean up if we got a client
			if client != nil {
				client.Close()
			}
		})
	}
}

// Helper function to check if string contains substring
func containsString(s, substr string) bool {
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
