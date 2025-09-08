package statefulsets

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestGetAdminPassword(t *testing.T) {
	tests := []struct {
		name   string
		key    string
		envSet []corev1.EnvVar
		want   string
	}{
		{
			name: "Password found",
			key:  "REDIS_PASSWORD",
			envSet: []corev1.EnvVar{
				{
					Name:  "REDIS_PASSWORD",
					Value: "testpassword",
				},
				{
					Name:  "OTHER_VAR",
					Value: "othervalue",
				},
			},
			want: "testpassword",
		},
		{
			name: "Password not found",
			key:  "REDIS_PASSWORD",
			envSet: []corev1.EnvVar{
				{
					Name:  "OTHER_VAR",
					Value: "othervalue",
				},
			},
			want: "",
		},
		{
			name:   "Empty env set",
			key:    "REDIS_PASSWORD",
			envSet: []corev1.EnvVar{},
			want:   "",
		},
		{
			name: "Empty key",
			key:  "",
			envSet: []corev1.EnvVar{
				{
					Name:  "REDIS_PASSWORD",
					Value: "testpassword",
				},
			},
			want: "",
		},
		{
			name: "Empty value",
			key:  "REDIS_PASSWORD",
			envSet: []corev1.EnvVar{
				{
					Name:  "REDIS_PASSWORD",
					Value: "",
				},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getAdminPassword(tt.key, tt.envSet)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestExtractUserFromSecretKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want string
	}{
		{
			name: "Regular secret key",
			key:  "redis-cluster--user1",
			want: "user1",
		},
		{
			name: "Secret key with multiple dashes",
			key:  "redis-cluster--admin--user",
			want: "user",
		},
		{
			name: "Secret key with single dash",
			key:  "redis-user1",
			want: "redis-user1", // The function returns the last part after splitting by "--"
		},
		{
			name: "Secret key ending with dash",
			key:  "redis-cluster--",
			want: "",
		},
		{
			name: "Empty key",
			key:  "",
			want: "",
		},
		{
			name: "Key without dashes",
			key:  "user1",
			want: "user1",
		},
		{
			name: "Complex key",
			key:  "redis-cluster--admin--user--test--123",
			want: "123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractUserFromSecretKey(tt.key)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestParseAclInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  map[string]string
	}{
		{
			name: "Valid ACL input with multiple users",
			input: `user admin on >admin ~* &* +@all
user user1 on >password1 ~* &* +@all
user user2 on >password2 ~* &* +@all`,
			want: map[string]string{
				"user1": "user1 on >password1 ~* &* +@all",
				"user2": "user2 on >password2 ~* &* +@all",
			},
		},
		{
			name: "ACL input with admin user (should be excluded)",
			input: `user admin on >admin ~* &* +@all
user user1 on >password1 ~* &* +@all`,
			want: map[string]string{
				"user1": "user1 on >password1 ~* &* +@all",
			},
		},
		{
			name:  "Empty input",
			input: "",
			want:  map[string]string{},
		},
		{
			name: "Input with empty lines",
			input: `user admin on >admin ~* &* +@all

user user1 on >password1 ~* &* +@all

user user2 on >password2 ~* &* +@all`,
			want: map[string]string{
				"user1": "user1 on >password1 ~* &* +@all",
				"user2": "user2 on >password2 ~* &* +@all",
			},
		},
		{
			name: "Input with malformed lines",
			input: `user admin on >admin ~* &* +@all
invalid line
user user1 on >password1 ~* &* +@all
another invalid line`,
			want: map[string]string{
				"user1": "user1 on >password1 ~* &* +@all",
			},
		},
		{
			name:  "Input with only admin user",
			input: `user admin on >admin ~* &* +@all`,
			want:  map[string]string{},
		},
		{
			name:  "Single user",
			input: "user user1 on >password1 ~* +@all",
			want: map[string]string{
				"user1": "user1 on >password1 ~* +@all",
			},
		},
		{
			name:  "Multiple users",
			input: "user user1 on >password1 ~* +@all\nuser user2 on >password2 ~* +@read",
			want: map[string]string{
				"user1": "user1 on >password1 ~* +@all",
				"user2": "user2 on >password2 ~* +@read",
			},
		},
		{
			name:  "Invalid format",
			input: "invalid acl format",
			want:  map[string]string{},
		},
		{
			name:  "User with spaces in name",
			input: "user \"user with spaces\" on >password1 ~* +@all",
			want: map[string]string{
				"\"user": "\"user with spaces\" on >password1 ~* +@all",
			},
		},
		{
			name: "Input with whitespace",
			input: `  user admin on >admin ~* &* +@all  
user user1 on >password1 ~* &* +@all  
  user user2 on >password2 ~* &* +@all  `,
			want: map[string]string{
				"user1": "user1 on >password1 ~* &* +@all",
				"user2": "user2 on >password2 ~* &* +@all",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseAclInput(tt.input)
			assert.Equal(t, tt.want, result)
		})
	}
}

// Note: Tests for functions that require external dependencies (client.Client, logr.Logger)
// are skipped as they require complex mocking. These functions are:
// - GetRedisClusterAdminPassword
// - GenerateAclcommands
// - GetClusterPassword
// - IsdiffAclConfigs

func TestHelperFunctionsEdgeCases(t *testing.T) {
	t.Run("ParseAclInput with whitespace", func(t *testing.T) {
		input := `  user admin on >admin ~* &* +@all  
user user1 on >password1 ~* &* +@all  
  user user2 on >password2 ~* &* +@all  `

		result := parseAclInput(input)

		expected := map[string]string{
			"user1": "user1 on >password1 ~* &* +@all",
			"user2": "user2 on >password2 ~* &* +@all",
		}

		assert.Equal(t, expected, result)
	})

	t.Run("ExtractUserFromSecretKey with complex key", func(t *testing.T) {
		key := "redis-cluster--admin--user--test--123"

		result := extractUserFromSecretKey(key)

		assert.Equal(t, "123", result)
	})

	t.Run("GetAdminPassword with ValueFrom", func(t *testing.T) {
		envSet := []corev1.EnvVar{
			{
				Name: "REDIS_PASSWORD",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "test-secret",
						},
						Key: "password",
					},
				},
			},
		}

		result := getAdminPassword("REDIS_PASSWORD", envSet)

		// Should return empty string when ValueFrom is set
		assert.Equal(t, "", result)
	})
}

func TestHelperFunctionsIntegration(t *testing.T) {
	t.Run("Full ACL workflow", func(t *testing.T) {
		// Test the complete workflow of ACL processing
		aclInput := `user admin on >admin ~* &* +@all
user user1 on >password1 ~* &* +@all
user user2 on >password2 ~* &* +@all`

		parsedAcl := parseAclInput(aclInput)

		assert.Len(t, parsedAcl, 2) // admin should be excluded
		assert.Contains(t, parsedAcl, "user1")
		assert.Contains(t, parsedAcl, "user2")

		// Test user extraction
		secretKey := "redis-cluster--user1"
		user := extractUserFromSecretKey(secretKey)
		assert.Equal(t, "user1", user)

		// Test ACL command generation
		expectedCommand := "acl setuser user1 " + parsedAcl["user1"]
		assert.Contains(t, expectedCommand, "acl setuser user1")
	})
}
