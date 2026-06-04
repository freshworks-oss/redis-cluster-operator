package redisutil

import (
	"context"
	"testing"
	"time"

	"github.com/freshworks/redis-cluster-operator/utils/testutil"
)

// TestNewAdmin tests the NewAdmin constructor
func TestNewAdmin(t *testing.T) {
	logger := testutil.NewTestLogger()
	ctx := context.Background()
	mockMetrics := &testutil.MockMetrics{}

	tests := []struct {
		name        string
		addrs       []string
		options     *AdminOptions
		expectError bool
	}{
		{
			name:        "Create admin with empty addresses",
			addrs:       []string{},
			options:     nil,
			expectError: false,
		},
		{
			name:        "Create admin with single address",
			addrs:       []string{"127.0.0.1:6379"},
			options:     nil,
			expectError: false,
		},
		{
			name:  "Create admin with options",
			addrs: []string{"127.0.0.1:6379", "127.0.0.1:6380"},
			options: &AdminOptions{
				ConnectionTimeout: 5 * time.Second,
				ClientName:        "test-admin",
				Password:          "admin-password",
			},
			expectError: false,
		},
		{
			name:  "Create admin with custom options",
			addrs: []string{"127.0.0.1:6379"},
			options: &AdminOptions{
				ConnectionTimeout:  10 * time.Second,
				ClientName:         "custom-client",
				RenameCommandsFile: "/nonexistent/file", // Should handle gracefully
				Password:           "secure-password",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			admin := NewAdmin(tt.addrs, tt.options, logger, ctx, mockMetrics)

			if admin == nil && !tt.expectError {
				t.Error("Expected non-nil admin interface")
			}

			// Test interface compliance
			var _ IAdmin = admin

			// Test that admin has connections
			connections := admin.Connections()
			if connections == nil {
				t.Error("Admin should have non-nil connections")
			}

			// Test hash max slot value
			hashMaxSlot := admin.GetHashMaxSlot()
			if hashMaxSlot != DefaultHashMaxSlots {
				t.Errorf("Expected hash max slot %d, got %d", DefaultHashMaxSlots, hashMaxSlot)
			}

			// Clean up
			admin.Close(ctx)
		})
	}
}

// TestAdmin_Connections tests the Connections method
func TestAdmin_Connections(t *testing.T) {
	logger := testutil.NewTestLogger()
	ctx := context.Background()
	mockMetrics := &testutil.MockMetrics{}

	admin := NewAdmin([]string{}, nil, logger, ctx, mockMetrics)
	defer admin.Close(ctx)

	connections := admin.Connections()
	if connections == nil {
		t.Error("Connections should return non-nil IAdminConnections")
	}

	// Test that we can call methods on the connections
	allClients := connections.GetAll()
	if allClients == nil {
		t.Error("GetAll should return non-nil map")
	}
}

// TestAdmin_Close tests the Close method
func TestAdmin_Close(t *testing.T) {
	logger := testutil.NewTestLogger()
	ctx := context.Background()
	mockMetrics := &testutil.MockMetrics{}

	admin := NewAdmin([]string{}, nil, logger, ctx, mockMetrics)

	// Should not panic
	admin.Close(ctx)

	// Should be safe to call multiple times
	admin.Close(ctx)

	// Connections should still be accessible after close
	connections := admin.Connections()
	if connections == nil {
		t.Error("Connections should still be accessible after close")
	}
}

// TestAdmin_GetHashMaxSlot tests the GetHashMaxSlot method
func TestAdmin_GetHashMaxSlot(t *testing.T) {
	logger := testutil.NewTestLogger()
	ctx := context.Background()
	mockMetrics := &testutil.MockMetrics{}

	admin := NewAdmin([]string{}, nil, logger, ctx, mockMetrics)
	defer admin.Close(ctx)

	hashMaxSlot := admin.GetHashMaxSlot()

	if hashMaxSlot != DefaultHashMaxSlots {
		t.Errorf("Expected hash max slot %d, got %d", DefaultHashMaxSlots, hashMaxSlot)
	}

	// Test the constant value
	if DefaultHashMaxSlots != 16383 {
		t.Errorf("Expected DefaultHashMaxSlots to be 16383, got %d", DefaultHashMaxSlots)
	}
}

// TestAdminOptions tests the AdminOptions struct
func TestAdminOptions(t *testing.T) {
	tests := []struct {
		name        string
		options     *AdminOptions
		description string
	}{
		{
			name:        "Nil options",
			options:     nil,
			description: "Should handle nil options gracefully",
		},
		{
			name: "Complete options",
			options: &AdminOptions{
				ConnectionTimeout:  30 * time.Second,
				ClientName:         "test-client",
				RenameCommandsFile: "/path/to/rename.conf",
				Password:           "super-secret",
			},
			description: "Should handle all option fields",
		},
		{
			name: "Partial options",
			options: &AdminOptions{
				ClientName: "partial-client",
				Password:   "partial-password",
			},
			description: "Should handle partially filled options",
		},
		{
			name: "Zero timeout options",
			options: &AdminOptions{
				ConnectionTimeout: 0, // Should use default
				ClientName:        "zero-timeout",
			},
			description: "Should handle zero timeout gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := testutil.NewTestLogger()
			ctx := context.Background()
			mockMetrics := &testutil.MockMetrics{}

			// Should not panic with any options
			admin := NewAdmin([]string{}, tt.options, logger, ctx, mockMetrics)
			if admin == nil {
				t.Errorf("NewAdmin should not return nil for %s", tt.description)
			}

			// Test that connections work with the options
			connections := admin.Connections()
			if connections == nil {
				t.Errorf("Connections should work for %s", tt.description)
			}

			admin.Close(ctx)
		})
	}
}

// TestConstants tests package constants
func TestConstants(t *testing.T) {
	// Test hash slots constant
	if DefaultHashMaxSlots != 16383 {
		t.Errorf("DefaultHashMaxSlots should be 16383, got %d", DefaultHashMaxSlots)
	}

	// Test redis failover constants
	if RedisFailoverForce != "FORCE" {
		t.Errorf("RedisFailoverForce should be 'FORCE', got %s", RedisFailoverForce)
	}

	if RedisFailoverTakeover != "TAKEOVER" {
		t.Errorf("RedisFailoverTakeover should be 'TAKEOVER', got %s", RedisFailoverTakeover)
	}

	if RedisFailoverDefault != "" {
		t.Errorf("RedisFailoverDefault should be empty, got %s", RedisFailoverDefault)
	}

	// Test reset constants
	if ResetHard != "HARD" {
		t.Errorf("ResetHard should be 'HARD', got %s", ResetHard)
	}

	if ResetSoft != "SOFT" {
		t.Errorf("ResetSoft should be 'SOFT', got %s", ResetSoft)
	}

	// Test timeout constants
	if RoleChangeTimeout != 20 {
		t.Errorf("RoleChangeTimeout should be 20, got %d", RoleChangeTimeout)
	}

	if RoleChangeInterval != 1 {
		t.Errorf("RoleChangeInterval should be 1, got %d", RoleChangeInterval)
	}
}

// TestReplicationInfo tests the ReplicationInfo struct
func TestReplicationInfo(t *testing.T) {
	tests := []struct {
		name string
		info ReplicationInfo
	}{
		{
			name: "Master replication info",
			info: ReplicationInfo{
				Role:                    "master",
				MasterLinkStatus:        "up",
				MasterHost:              "",
				LastIoSecondsAgo:        0,
				SlaveReplicationOffset:  0,
				MasterReplicationOffset: 1000,
				MasterSyncInProgress:    false,
			},
		},
		{
			name: "Slave replication info",
			info: ReplicationInfo{
				Role:                    "slave",
				MasterLinkStatus:        "up",
				MasterHost:              "127.0.0.1",
				LastIoSecondsAgo:        5,
				SlaveReplicationOffset:  995,
				MasterReplicationOffset: 1000,
				MasterSyncInProgress:    false,
			},
		},
		{
			name: "Syncing slave info",
			info: ReplicationInfo{
				Role:                    "slave",
				MasterLinkStatus:        "up",
				MasterHost:              "127.0.0.1",
				LastIoSecondsAgo:        1,
				SlaveReplicationOffset:  500,
				MasterReplicationOffset: 1000,
				MasterSyncInProgress:    true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that we can create and access ReplicationInfo
			info := tt.info

			if info.Role == "" {
				t.Error("Role should not be empty")
			}

			// Test that offset values are accessible
			if info.Role == "slave" {
				if info.MasterHost == "" {
					t.Error("Slave should have master host")
				}

				if info.SlaveReplicationOffset > info.MasterReplicationOffset {
					t.Error("Slave offset should not be greater than master offset")
				}
			}

			if info.Role == "master" {
				if info.MasterReplicationOffset < 0 {
					t.Error("Master replication offset should not be negative")
				}
			}
		})
	}
}

// TestAdmin_Integration tests integration scenarios
func TestAdmin_Integration(t *testing.T) {
	logger := testutil.NewTestLogger()
	ctx := context.Background()
	mockMetrics := &testutil.MockMetrics{}

	t.Run("Complete admin workflow", func(t *testing.T) {
		options := &AdminOptions{
			ConnectionTimeout: 2 * time.Second,
			ClientName:        "integration-test",
			Password:          "integration-password",
		}

		// Create admin
		admin := NewAdmin([]string{}, options, logger, ctx, mockMetrics)
		if admin == nil {
			t.Fatal("Admin should be created successfully")
		}

		// Test connections
		connections := admin.Connections()
		if connections == nil {
			t.Error("Should have connections")
		}

		// Test auth retrieval through connections
		password, hasAuth := connections.GetAUTH()
		if !hasAuth {
			t.Error("Should have auth when password is set")
		}
		if password != "integration-password" {
			t.Errorf("Expected password 'integration-password', got '%s'", password)
		}

		// Test hash max slot
		maxSlot := admin.GetHashMaxSlot()
		if maxSlot != DefaultHashMaxSlots {
			t.Errorf("Expected max slot %d, got %d", DefaultHashMaxSlots, maxSlot)
		}

		// Clean up
		admin.Close(ctx)

		// Should still be accessible after close
		if admin.Connections() == nil {
			t.Error("Connections should still be accessible after close")
		}
	})

	t.Run("Multiple admins", func(t *testing.T) {
		// Test that we can create multiple admin instances
		admin1 := NewAdmin([]string{}, nil, logger, ctx, mockMetrics)
		admin2 := NewAdmin([]string{"127.0.0.1:6379"}, &AdminOptions{ClientName: "test2"}, logger, ctx, mockMetrics)

		if admin1 == nil || admin2 == nil {
			t.Fatal("Both admins should be created successfully")
		}

		// Both should have independent hash max slots
		if admin1.GetHashMaxSlot() != admin2.GetHashMaxSlot() {
			t.Error("Both admins should have same max slot value")
		}

		// Both should have independent connections
		conn1 := admin1.Connections()
		conn2 := admin2.Connections()

		if conn1 == conn2 {
			t.Error("Admins should have independent connections")
		}

		// Clean up
		admin1.Close(ctx)
		admin2.Close(ctx)
	})

	t.Run("Concurrent access", func(t *testing.T) {
		admin := NewAdmin([]string{}, nil, logger, ctx, mockMetrics)
		defer admin.Close(ctx)

		// Test concurrent access to admin methods
		done := make(chan bool, 3)

		go func() {
			for i := 0; i < 10; i++ {
				_ = admin.GetHashMaxSlot()
			}
			done <- true
		}()

		go func() {
			for i := 0; i < 10; i++ {
				_ = admin.Connections()
			}
			done <- true
		}()

		go func() {
			for i := 0; i < 10; i++ {
				connections := admin.Connections()
				_ = connections.GetAll()
			}
			done <- true
		}()

		// Wait for all goroutines to complete
		for i := 0; i < 3; i++ {
			<-done
		}
		// If we reach here without deadlock, the test passes
	})
}

// TestRedisReplStatus tests the RedisReplStatus type
func TestRedisReplStatus(t *testing.T) {
	tests := []struct {
		name   string
		status RedisReplStatus
	}{
		{
			name:   "Redis syncing status",
			status: RedisSyncing,
		},
		{
			name:   "Redis master still pending status",
			status: RedisMasterSillPending,
		},
		{
			name:   "Redis link up status",
			status: RedisLinkUp,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that status is not empty
			if string(tt.status) == "" {
				t.Error("Redis status should not be empty")
			}

			// Test that we can convert to string
			statusStr := string(tt.status)
			if len(statusStr) == 0 {
				t.Error("Status string should not be empty")
			}
		})
	}
}

// TestAdmin_GetClusterInfos tests the GetClusterInfos method
func TestAdmin_GetClusterInfos(t *testing.T) {
	logger := testutil.NewTestLogger()
	ctx := context.Background()
	mockMetrics := &testutil.MockMetrics{}

	admin := NewAdmin([]string{}, nil, logger, ctx, mockMetrics)
	defer admin.Close(ctx)

	// Test with empty connections
	infos, err := admin.GetClusterInfos(ctx, "test-ns", "test-cluster")
	if err != nil {
		t.Errorf("Expected no error for empty connections, got %v", err)
	}
	if infos == nil {
		t.Error("Expected non-nil ClusterInfos")
	}
	// With empty connections, the status should be consistent since there are no errors
	if infos.Status != ClusterInfosConsistent {
		t.Errorf("Expected status ClusterInfosConsistent, got %s", infos.Status)
	}
}

// TestAdmin_ClusterManagerNodeIsEmpty tests the ClusterManagerNodeIsEmpty method
func TestAdmin_ClusterManagerNodeIsEmpty(t *testing.T) {
	logger := testutil.NewTestLogger()
	ctx := context.Background()
	mockMetrics := &testutil.MockMetrics{}

	admin := NewAdmin([]string{}, nil, logger, ctx, mockMetrics)
	defer admin.Close(ctx)

	// Test with empty connections
	isEmpty, err := admin.ClusterManagerNodeIsEmpty(ctx, "test-ns", "test-cluster")
	if err != nil {
		t.Errorf("Expected no error for empty connections, got %v", err)
	}
	if !isEmpty {
		t.Error("Expected node to be empty with no connections")
	}
}

// TestAdmin_SetConfigEpoch tests the SetConfigEpoch method
func TestAdmin_SetConfigEpoch(t *testing.T) {
	logger := testutil.NewTestLogger()
	ctx := context.Background()
	mockMetrics := &testutil.MockMetrics{}

	admin := NewAdmin([]string{}, nil, logger, ctx, mockMetrics)
	defer admin.Close(ctx)

	// Test with empty connections
	err := admin.SetConfigEpoch(ctx, "test-ns", "test-cluster")
	if err != nil {
		t.Errorf("Expected no error for empty connections, got %v", err)
	}
}

// TestAdmin_SetConfigIfNeed tests the SetConfigIfNeed method
func TestAdmin_SetConfigIfNeed(t *testing.T) {
	logger := testutil.NewTestLogger()
	ctx := context.Background()
	mockMetrics := &testutil.MockMetrics{}

	admin := NewAdmin([]string{}, nil, logger, ctx, mockMetrics)
	defer admin.Close(ctx)

	// Test with empty connections
	config := map[string]string{"maxmemory": "100mb"}
	err := admin.SetConfigIfNeed(ctx, config, "test-ns", "test-cluster")
	if err != nil {
		t.Errorf("Expected no error for empty connections, got %v", err)
	}
}

// TestAdmin_GetAllConfig tests the GetAllConfig method
func TestAdmin_GetAllConfig(t *testing.T) {
	logger := testutil.NewTestLogger()
	ctx := context.Background()
	mockMetrics := &testutil.MockMetrics{}

	admin := NewAdmin([]string{}, nil, logger, ctx, mockMetrics)
	defer admin.Close(ctx)

	// Test with nil client - this should handle gracefully
	config, err := admin.GetAllConfig(ctx, nil, "127.0.0.1:6379", "test-ns", "test-cluster")
	if err == nil {
		t.Error("Expected error for nil client")
	}
	if config != nil {
		t.Error("Expected nil config for nil client")
	}
}

// TestAdmin_AttachNodeToCluster tests the AttachNodeToCluster method
func TestAdmin_AttachNodeToCluster(t *testing.T) {
	logger := testutil.NewTestLogger()
	ctx := context.Background()
	mockMetrics := &testutil.MockMetrics{}

	admin := NewAdmin([]string{}, nil, logger, ctx, mockMetrics)
	defer admin.Close(ctx)

	// Test with empty connections - should return error
	err := admin.AttachNodeToCluster(ctx, "127.0.0.1:6379", "test-ns", "test-cluster")
	if err == nil {
		t.Error("Expected error for empty connections")
	}
}

// TestAdmin_AttachSlaveToMaster tests the AttachSlaveToMaster method
func TestAdmin_AttachSlaveToMaster(t *testing.T) {
	logger := testutil.NewTestLogger()
	ctx := context.Background()
	mockMetrics := &testutil.MockMetrics{}

	admin := NewAdmin([]string{}, nil, logger, ctx, mockMetrics)
	defer admin.Close(ctx)

	slave := &Node{ID: "slave1", IP: "127.0.0.1", Port: "6379"}
	masterID := "master1"

	// Test with empty connections - should return error
	err := admin.AttachSlaveToMaster(ctx, slave, masterID, "test-ns", "test-cluster")
	if err == nil {
		t.Error("Expected error for empty connections")
	}
}

// TestAdmin_DetachSlave tests the DetachSlave method
func TestAdmin_DetachSlave(t *testing.T) {
	logger := testutil.NewTestLogger()
	ctx := context.Background()
	mockMetrics := &testutil.MockMetrics{}

	admin := NewAdmin([]string{}, nil, logger, ctx, mockMetrics)
	defer admin.Close(ctx)

	slave := &Node{ID: "slave1", IP: "127.0.0.1", Port: "6379"}

	// Test with empty connections - should return error
	err := admin.DetachSlave(ctx, slave, "test-ns", "test-cluster")
	if err == nil {
		t.Error("Expected error for empty connections")
	}
}

// TestAdmin_ForgetNode tests the ForgetNode method
func TestAdmin_ForgetNode(t *testing.T) {
	logger := testutil.NewTestLogger()
	ctx := context.Background()
	mockMetrics := &testutil.MockMetrics{}

	admin := NewAdmin([]string{}, nil, logger, ctx, mockMetrics)
	defer admin.Close(ctx)

	// Test with empty connections
	err := admin.ForgetNode(ctx, "node1", "test-ns", "test-cluster")
	if err != nil {
		t.Errorf("Expected no error for empty connections, got %v", err)
	}
}

// TestAdmin_SetSlots tests the SetSlots method
func TestAdmin_SetSlots(t *testing.T) {
	logger := testutil.NewTestLogger()
	ctx := context.Background()
	mockMetrics := &testutil.MockMetrics{}

	admin := NewAdmin([]string{}, nil, logger, ctx, mockMetrics)
	defer admin.Close(ctx)

	slots := []Slot{1, 2, 3}

	// Test with empty connections - should return error
	err := admin.SetSlots(ctx, "127.0.0.1:6379", "IMPORTING", slots, "node1", "test-ns", "test-cluster")
	if err == nil {
		t.Error("Expected error for empty connections")
	}
}

// TestAdmin_AddSlots tests the AddSlots method
func TestAdmin_AddSlots(t *testing.T) {
	logger := testutil.NewTestLogger()
	ctx := context.Background()
	mockMetrics := &testutil.MockMetrics{}

	admin := NewAdmin([]string{}, nil, logger, ctx, mockMetrics)
	defer admin.Close(ctx)

	slots := []Slot{1, 2, 3}

	// Test with empty connections - should return error
	err := admin.AddSlots(ctx, "127.0.0.1:6379", slots)
	if err == nil {
		t.Error("Expected error for empty connections")
	}
}

// TestAdmin_SetSlot tests the SetSlot method
func TestAdmin_SetSlot(t *testing.T) {
	logger := testutil.NewTestLogger()
	ctx := context.Background()
	mockMetrics := &testutil.MockMetrics{}

	admin := NewAdmin([]string{}, nil, logger, ctx, mockMetrics)
	defer admin.Close(ctx)

	// Empty address list: no live Redis — expect pipeline/connection failure
	err := admin.SetSlot(ctx, "127.0.0.1:6379", "IMPORTING", 1, "node1", "test-ns", "test-cluster")
	if err == nil {
		t.Error("Expected error when no Redis connections are configured")
	}
}

// TestAdmin_MigrateKeys tests the MigrateKeys method
func TestAdmin_MigrateKeys(t *testing.T) {
	logger := testutil.NewTestLogger()
	ctx := context.Background()
	mockMetrics := &testutil.MockMetrics{}

	admin := NewAdmin([]string{}, nil, logger, ctx, mockMetrics)
	defer admin.Close(ctx)

	dest := &Node{ID: "dest1", IP: "127.0.0.1", Port: "6380"}
	slots := []Slot{1, 2, 3}

	// Test with empty connections - should return error
	count, err := admin.MigrateKeys(ctx, "127.0.0.1:6379", dest, slots, 10, 1000, false, "test-ns", "test-cluster")
	if err == nil {
		t.Error("Expected error for empty connections")
	}
	if count != 0 {
		t.Errorf("Expected 0 migrated keys, got %d", count)
	}
}

// TestAdmin_MigrateKeysInSlot tests the MigrateKeysInSlot method
func TestAdmin_MigrateKeysInSlot(t *testing.T) {
	logger := testutil.NewTestLogger()
	ctx := context.Background()
	mockMetrics := &testutil.MockMetrics{}

	admin := NewAdmin([]string{}, nil, logger, ctx, mockMetrics)
	defer admin.Close(ctx)

	dest := &Node{ID: "dest1", IP: "127.0.0.1", Port: "6380"}

	// Test with empty connections - should return error
	count, err := admin.MigrateKeysInSlot(ctx, "127.0.0.1:6379", dest, 1, 10, 1000, false, "test-ns", "test-cluster")
	if err == nil {
		t.Error("Expected error for empty connections")
	}
	if count != 0 {
		t.Errorf("Expected 0 migrated keys, got %d", count)
	}
}

// TestAdmin_FlushAndReset tests the FlushAndReset method
func TestAdmin_FlushAndReset(t *testing.T) {
	logger := testutil.NewTestLogger()
	ctx := context.Background()
	mockMetrics := &testutil.MockMetrics{}

	admin := NewAdmin([]string{}, nil, logger, ctx, mockMetrics)
	defer admin.Close(ctx)

	err := admin.FlushAndReset(ctx, "127.0.0.1:6379", ResetSoft, "test-ns", "test-cluster")
	if err == nil {
		t.Error("Expected error when no Redis connections are configured")
	}
}

// TestAdmin_ResetPassword tests the ResetPassword method
func TestAdmin_ResetPassword(t *testing.T) {
	logger := testutil.NewTestLogger()
	ctx := context.Background()
	mockMetrics := &testutil.MockMetrics{}

	admin := NewAdmin([]string{}, nil, logger, ctx, mockMetrics)
	defer admin.Close(ctx)

	// Test with empty connections - should return error
	err := admin.ResetPassword(ctx, "newpassword", "test-ns", "test-cluster")
	if err == nil {
		t.Error("Expected error for empty connections")
	}
}

// TestAdmin_FailoverSlaveToMaster tests the FailoverSlaveToMaster method
func TestAdmin_FailoverSlaveToMaster(t *testing.T) {
	logger := testutil.NewTestLogger()
	ctx := context.Background()
	mockMetrics := &testutil.MockMetrics{}

	admin := NewAdmin([]string{}, nil, logger, ctx, mockMetrics)
	defer admin.Close(ctx)

	slave := &Node{ID: "slave1", IP: "127.0.0.1", Port: "6379"}

	// Lazy Get() uses NewClient, which currently returns a client without a failed PING check, so this
	// often returns nil here even without Redis in Addrs; errors show up later if role polling runs.
	err := admin.FailoverSlaveToMaster(ctx, slave, RedisFailoverDefault, "test-ns", "test-cluster")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestAdmin_GetReplicationStatus tests the GetReplicationStatus method
func TestAdmin_GetReplicationStatus(t *testing.T) {
	logger := testutil.NewTestLogger()
	ctx := context.Background()
	mockMetrics := &testutil.MockMetrics{}

	admin := NewAdmin([]string{}, nil, logger, ctx, mockMetrics)
	defer admin.Close(ctx)

	// Test with empty connections - should return error due to invalid address format
	status, info, err := admin.GetReplicationStatus("127.0.0.1", ctx, "test-ns", "test-cluster")
	if err == nil {
		t.Error("Expected error for invalid address format")
	}
	// The function returns ReplicaStatusUnknown as default even on error
	if status != ReplicaStatusUnknown {
		t.Errorf("Expected status %s for error case, got %s", ReplicaStatusUnknown, status)
	}
	if info.Role != "" {
		t.Errorf("Expected empty role for error case, got %s", info.Role)
	}
}

// TestAdmin_GetInfoReplication tests the GetInfoReplication method
func TestAdmin_GetInfoReplication(t *testing.T) {
	logger := testutil.NewTestLogger()
	ctx := context.Background()
	mockMetrics := &testutil.MockMetrics{}

	admin := NewAdmin([]string{}, nil, logger, ctx, mockMetrics)
	defer admin.Close(ctx)

	info, err := admin.GetInfoReplication("127.0.0.1:6379", ctx, "test-ns", "test-cluster")
	if err == nil {
		t.Error("Expected error when no Redis connections are configured")
	}
	if info.Role != "" {
		t.Error("Expected empty replication info on error")
	}
}

// TestAdmin_DrainClientsInOldMaster tests the DrainClientsInOldMaster method
func TestAdmin_DrainClientsInOldMaster(t *testing.T) {
	logger := testutil.NewTestLogger()
	ctx := context.Background()
	mockMetrics := &testutil.MockMetrics{}

	admin := NewAdmin([]string{}, nil, logger, ctx, mockMetrics)
	defer admin.Close(ctx)

	node := &Node{ID: "node1", IP: "127.0.0.1", Port: "6379"}

	err := admin.DrainClientsInOldMaster(ctx, node, "test-ns", "test-cluster")
	if err == nil {
		t.Error("Expected error when no Redis connections are configured")
	}
}

// TestAdmin_ReplicationInfo tests the ReplicationInfo struct and constants
func TestAdmin_ReplicationInfo(t *testing.T) {
	// Test ReplicationInfo struct
	info := ReplicationInfo{
		Role:                    "master",
		MasterLinkStatus:        "up",
		MasterHost:              "127.0.0.1",
		LastIoSecondsAgo:        10,
		SlaveReplicationOffset:  1000,
		MasterReplicationOffset: 1000,
		MasterSyncInProgress:    false,
	}

	if info.Role != "master" {
		t.Errorf("Expected role 'master', got %s", info.Role)
	}

	// Test RedisReplStatus constants
	if RedisSyncing != "master_sync_in_progress:1" {
		t.Errorf("Expected RedisSyncing to be 'master_sync_in_progress:1', got %s", RedisSyncing)
	}

	if RedisMasterSillPending != "master_host:127.0.0.1" {
		t.Errorf("Expected RedisMasterSillPending to be 'master_host:127.0.0.1', got %s", RedisMasterSillPending)
	}

	if RedisLinkUp != "master_link_status:up" {
		t.Errorf("Expected RedisLinkUp to be 'master_link_status:up', got %s", RedisLinkUp)
	}

	// Test RedisFailover constants
	if RedisFailoverForce != "FORCE" {
		t.Errorf("Expected RedisFailoverForce to be 'FORCE', got %s", RedisFailoverForce)
	}

	if RedisFailoverTakeover != "TAKEOVER" {
		t.Errorf("Expected RedisFailoverTakeover to be 'TAKEOVER', got %s", RedisFailoverTakeover)
	}

	if RedisFailoverDefault != "" {
		t.Errorf("Expected RedisFailoverDefault to be empty string, got %s", RedisFailoverDefault)
	}

	// Test other constants
	if ReplicaStatusUnknown != "ReplicaStatusunknown" {
		t.Errorf("Expected ReplicaStatusUnknown to be 'ReplicaStatusunknown', got %s", ReplicaStatusUnknown)
	}

	if ReplicaStatusReady != "ReplicaSatusready" {
		t.Errorf("Expected ReplicaStatusReady to be 'ReplicaSatusready', got %s", ReplicaStatusReady)
	}

	if ReplicaLinkDown != "MasterLinkDown" {
		t.Errorf("Expected ReplicaLinkDown to be 'MasterLinkDown', got %s", ReplicaLinkDown)
	}

	if ReplicaStatusSyncing != "ReplicaSyncing" {
		t.Errorf("Expected ReplicaStatusSyncing to be 'ReplicaSyncing', got %s", ReplicaStatusSyncing)
	}

	if ReplicaStatusMasterNil != "ReplicaMasterNil" {
		t.Errorf("Expected ReplicaStatusMasterNil to be 'ReplicaMasterNil', got %s", ReplicaStatusMasterNil)
	}

	if ClusterStatusMigrating != "SlotsMigrating" {
		t.Errorf("Expected ClusterStatusMigrating to be 'SlotsMigrating', got %s", ClusterStatusMigrating)
	}
}

// TestAdmin_RegexPatterns tests the regex patterns
func TestAdmin_RegexPatterns(t *testing.T) {
	// Test clusterKnownNodesRE
	clusterInfo := "cluster_known_nodes:5"
	matches := clusterKnownNodesRE.FindStringSubmatch(clusterInfo)
	if len(matches) != 2 || matches[1] != "5" {
		t.Errorf("Expected clusterKnownNodesRE to match '5', got %v", matches)
	}

	// Test forgetnodeUnknownRE
	forgetInfo := "ERR Unknown node abc123"
	matches = forgetnodeUnknownRE.FindStringSubmatch(forgetInfo)
	if len(matches) != 2 || matches[1] != "abc123" {
		t.Errorf("Expected forgetnodeUnknownRE to match 'abc123', got %v", matches)
	}

	// Test clusterStateRE
	stateInfo := "cluster_state:ok"
	matches = clusterStateRE.FindStringSubmatch(stateInfo)
	if len(matches) != 2 || matches[1] != "ok" {
		t.Errorf("Expected clusterStateRE to match 'ok', got %v", matches)
	}
}

// TestAdmin_Constants tests the admin constants
func TestAdmin_Constants(t *testing.T) {
	if DefaultHashMaxSlots != 16383 {
		t.Errorf("Expected DefaultHashMaxSlots to be 16383, got %d", DefaultHashMaxSlots)
	}

	if ResetHard != "HARD" {
		t.Errorf("Expected ResetHard to be 'HARD', got %s", ResetHard)
	}

	if ResetSoft != "SOFT" {
		t.Errorf("Expected ResetSoft to be 'SOFT', got %s", ResetSoft)
	}

	if RoleChangeTimeout != 20 {
		t.Errorf("Expected RoleChangeTimeout to be 20, got %d", RoleChangeTimeout)
	}

	if RoleChangeInterval != 1 {
		t.Errorf("Expected RoleChangeInterval to be 1, got %d", RoleChangeInterval)
	}
}

// TestAdmin_HelperFunctions tests the internal helper functions
func TestAdmin_HelperFunctions(t *testing.T) {
	logger := testutil.NewTestLogger()
	ctx := context.Background()
	mockMetrics := &testutil.MockMetrics{}

	admin := NewAdmin([]string{}, nil, logger, ctx, mockMetrics)
	defer admin.Close(ctx)

	// Test getInfos helper function behavior with empty connections
	// This tests the internal getInfos function indirectly
	infos, err := admin.GetClusterInfos(ctx, "test-ns", "test-cluster")
	if err != nil {
		t.Errorf("Expected no error for empty connections, got %v", err)
	}
	if infos == nil {
		t.Error("Expected non-nil ClusterInfos")
	}

	// Test clusterKnowNodes helper function behavior
	// This tests the internal clusterKnowNodes function indirectly
	// When there are no connections, it should return empty results
	if len(infos.Infos) != 0 {
		t.Errorf("Expected 0 cluster infos for empty connections, got %d", len(infos.Infos))
	}

	// Test CheckandUpdateClusterState helper function behavior
	// This tests the internal CheckandUpdateClusterState function indirectly
	if infos.Status != ClusterInfosConsistent {
		t.Errorf("Expected status ClusterInfosConsistent for empty connections, got %s", infos.Status)
	}

	// Test clusterGetState helper function behavior
	// This tests the internal clusterGetState function indirectly
	// The status should be consistent when there are no errors
	if infos.Status != ClusterInfosConsistent {
		t.Errorf("Expected consistent status for empty connections, got %s", infos.Status)
	}
}

func TestParseConfigGetStar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   interface{}
		want    map[string]string
		wantErr bool
	}{
		{
			name: "map interface keys string values string",
			input: map[interface{}]interface{}{
				"maxmemory": "0",
				"port":      "6379",
			},
			want: map[string]string{"maxmemory": "0", "port": "6379"},
		},
		{
			// []byte cannot be a map key (unhashable); RESP bulk keys/values arrive as alternating slices.
			name:  "slice with []byte and string pairs from RESP bulk",
			input: []interface{}{[]byte("timeout"), []byte("0"), "tcp-keepalive", "300"},
			want:  map[string]string{"timeout": "0", "tcp-keepalive": "300"},
		},
		{
			name:  "slice alternating string pairs",
			input: []interface{}{"a", "1", "b", "2"},
			want:  map[string]string{"a": "1", "b": "2"},
		},
		{
			name:  "slice with []byte pairs",
			input: []interface{}{[]byte("x"), []byte("y")},
			want:  map[string]string{"x": "y"},
		},
		{
			name:    "slice odd length rejects malformed reply",
			input:   []interface{}{"orphan-key"},
			wantErr: true,
		},
		{
			name:  "nil",
			input: nil,
			want:  map[string]string{},
		},
		{
			name:    "unsupported type",
			input:   int64(42),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseConfigGetStar(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseConfigGetStar: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("len got %d want %d: got=%v want=%v", len(got), len(tt.want), got, tt.want)
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("key %q: got %q want %q", k, got[k], v)
				}
			}
		})
	}
}
