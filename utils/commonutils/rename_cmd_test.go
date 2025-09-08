package utils

import (
	"os"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCommandReplaceMapping(t *testing.T) {
	tests := []struct {
		name        string
		fileContent string
		expectedMap map[string]string
		expectError bool
		description string
	}{
		{
			name: "Valid rename commands",
			fileContent: `# Redis configuration
rename-command FLUSHDB FLUSHDB_CUSTOM
rename-command FLUSHALL FLUSHALL_CUSTOM
rename-command DEBUG DEBUG_CUSTOM`,
			expectedMap: map[string]string{
				"FLUSHDB":  "FLUSHDB_CUSTOM",
				"FLUSHALL": "FLUSHALL_CUSTOM",
				"DEBUG":    "DEBUG_CUSTOM",
			},
			expectError: false,
			description: "Should parse valid rename-command directives",
		},
		{
			name: "Case insensitive rename-command",
			fileContent: `# Redis configuration
RENAME-COMMAND FLUSHDB FLUSHDB_CUSTOM
Rename-Command FLUSHALL FLUSHALL_CUSTOM
rename-command DEBUG DEBUG_CUSTOM`,
			expectedMap: map[string]string{
				"FLUSHDB":  "FLUSHDB_CUSTOM",
				"FLUSHALL": "FLUSHALL_CUSTOM",
				"DEBUG":    "DEBUG_CUSTOM",
			},
			expectError: false,
			description: "Should handle case insensitive rename-command",
		},
		{
			name: "Mixed content with comments and empty lines",
			fileContent: `# Redis configuration file
# This is a comment

rename-command FLUSHDB FLUSHDB_CUSTOM

# Another comment
rename-command FLUSHALL FLUSHALL_CUSTOM

# Empty line above
rename-command DEBUG DEBUG_CUSTOM`,
			expectedMap: map[string]string{
				"FLUSHDB":  "FLUSHDB_CUSTOM",
				"FLUSHALL": "FLUSHALL_CUSTOM",
				"DEBUG":    "DEBUG_CUSTOM",
			},
			expectError: false,
			description: "Should handle mixed content with comments and empty lines",
		},
		{
			name: "Invalid rename-command format",
			fileContent: `# Redis configuration
rename-command FLUSHDB
rename-command FLUSHALL FLUSHALL_CUSTOM EXTRA
rename-command DEBUG DEBUG_CUSTOM`,
			expectedMap: map[string]string{
				"DEBUG": "DEBUG_CUSTOM",
			},
			expectError: false,
			description: "Should ignore invalid rename-command formats",
		},
		{
			name:        "Empty file",
			fileContent: "",
			expectedMap: map[string]string{},
			expectError: false,
			description: "Should handle empty file",
		},
		{
			name: "File with only comments",
			fileContent: `# This is a comment
# Another comment
# No rename commands here`,
			expectedMap: map[string]string{},
			expectError: false,
			description: "Should handle file with only comments",
		},
		{
			name: "File with whitespace around commands",
			fileContent: `  rename-command  FLUSHDB  FLUSHDB_CUSTOM  
rename-command	FLUSHALL	FLUSHALL_CUSTOM
rename-command DEBUG DEBUG_CUSTOM`,
			expectedMap: map[string]string{
				"FLUSHDB":  "FLUSHDB_CUSTOM",
				"FLUSHALL": "FLUSHALL_CUSTOM",
				"DEBUG":    "DEBUG_CUSTOM",
			},
			expectError: false,
			description: "Should handle whitespace around commands",
		},
		{
			name: "Complex command names",
			fileContent: `rename-command FLUSHDB_FLUSHALL_DEBUG FLUSHDB_FLUSHALL_DEBUG_CUSTOM
rename-command _INTERNAL_COMMAND _INTERNAL_COMMAND_CUSTOM`,
			expectedMap: map[string]string{
				"FLUSHDB_FLUSHALL_DEBUG": "FLUSHDB_FLUSHALL_DEBUG_CUSTOM",
				"_INTERNAL_COMMAND":      "_INTERNAL_COMMAND_CUSTOM",
			},
			expectError: false,
			description: "Should handle complex command names",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpFile, err := os.CreateTemp("", "redis-config-*.conf")
			require.NoError(t, err)
			defer os.Remove(tmpFile.Name())

			// Write content to file
			_, err = tmpFile.WriteString(tt.fileContent)
			require.NoError(t, err)
			tmpFile.Close()

			// Test the function
			logger := logr.Discard()
			result := BuildCommandReplaceMapping(tmpFile.Name(), logger)

			// Verify results
			if tt.expectError {
				assert.Empty(t, result)
			} else {
				assert.Equal(t, len(tt.expectedMap), len(result))
				for expectedKey, expectedValue := range tt.expectedMap {
					assert.Equal(t, expectedValue, result[expectedKey],
						"Expected key %s to have value %s, got %s", expectedKey, expectedValue, result[expectedKey])
				}
			}
		})
	}
}

func TestBuildCommandReplaceMappingEdgeCases(t *testing.T) {
	t.Run("Non-existent file", func(t *testing.T) {
		logger := logr.Discard()
		result := BuildCommandReplaceMapping("/non/existent/file.conf", logger)

		assert.Empty(t, result, "Should return empty map for non-existent file")
	})

	t.Run("File with malformed lines", func(t *testing.T) {
		fileContent := `# Redis configuration
rename-command
rename-command FLUSHDB
rename-command FLUSHALL FLUSHALL_CUSTOM EXTRA
rename-command DEBUG DEBUG_CUSTOM
invalid line
another invalid line`

		tmpFile, err := os.CreateTemp("", "redis-config-*.conf")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.WriteString(fileContent)
		require.NoError(t, err)
		tmpFile.Close()

		logger := logr.Discard()
		result := BuildCommandReplaceMapping(tmpFile.Name(), logger)

		// Should only parse the valid line
		expectedMap := map[string]string{
			"DEBUG": "DEBUG_CUSTOM",
		}
		assert.Equal(t, expectedMap, result)
	})

	t.Run("File with very long lines", func(t *testing.T) {
		// Create a very long command name
		longCommand := "A"
		for i := 0; i < 1000; i++ {
			longCommand += "A"
		}

		fileContent := `rename-command ` + longCommand + ` ` + longCommand + `_CUSTOM`

		tmpFile, err := os.CreateTemp("", "redis-config-*.conf")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.WriteString(fileContent)
		require.NoError(t, err)
		tmpFile.Close()

		logger := logr.Discard()
		result := BuildCommandReplaceMapping(tmpFile.Name(), logger)

		// Should handle long lines without issues
		assert.Equal(t, 1, len(result))
		assert.Contains(t, result, longCommand)
		assert.Equal(t, longCommand+"_CUSTOM", result[longCommand])
	})
}

func TestBuildCommandReplaceMappingRealWorld(t *testing.T) {
	t.Run("Real Redis configuration format", func(t *testing.T) {
		fileContent := `# Redis configuration file
# Generated by redis-cluster-operator

# Network
bind 0.0.0.0
port 6379
timeout 0
tcp-keepalive 0

# General
daemonize no
supervised no
pidfile /var/run/redis_6379.pid
loglevel notice
logfile ""
databases 16

# Snapshotting
save 900 1
save 300 10
save 60 10000
stop-writes-on-bgsave-error yes
rdbcompression yes
rdbchecksum yes
dbfilename dump.rdb
dir ./

# Replication
replica-serve-stale-data yes
replica-read-only yes
repl-diskless-sync no
repl-diskless-sync-delay 5
repl-ping-replica-period 10
repl-timeout 60
repl-disable-tcp-nodelay no
repl-backlog-size 1mb
repl-backlog-ttl 3600

# Security
rename-command FLUSHDB ""
rename-command FLUSHALL ""
rename-command DEBUG ""
rename-command CONFIG ""
rename-command SHUTDOWN ""
rename-command EVAL ""
rename-command EVALSHA ""

# Memory management
maxmemory 100mb
maxmemory-policy allkeys-lru

# Append only file
appendonly yes
appendfilename "appendonly.aof"
appendfsync everysec
no-appendfsync-on-rewrite no
auto-aof-rewrite-percentage 100
auto-aof-rewrite-min-size 64mb

# Cluster
cluster-enabled yes
cluster-config-file nodes.conf
cluster-node-timeout 15000
cluster-slave-validity-factor 10
cluster-migration-barrier 1
cluster-require-full-coverage yes`

		tmpFile, err := os.CreateTemp("", "redis-config-*.conf")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.WriteString(fileContent)
		require.NoError(t, err)
		tmpFile.Close()

		logger := logr.Discard()
		result := BuildCommandReplaceMapping(tmpFile.Name(), logger)

		// Should parse all the rename-command directives
		expectedMap := map[string]string{
			"FLUSHDB":  "\"\"",
			"FLUSHALL": "\"\"",
			"DEBUG":    "\"\"",
			"CONFIG":   "\"\"",
			"SHUTDOWN": "\"\"",
			"EVAL":     "\"\"",
			"EVALSHA":  "\"\"",
		}

		assert.Equal(t, expectedMap, result)
	})
}
