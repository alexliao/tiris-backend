package test

import (
	"testing"
	"time"

	"tiris-backend/internal/services"
	"tiris-backend/pkg/security"
	"tiris-backend/test/config"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestSecurityService_UserAPIKey_SimpleMethods tests UserAPIKey helper methods
func TestSecurityService_UserAPIKey_SimpleMethods(t *testing.T) {
	// Setup test config
	_ = config.GetProfileConfig(config.ProfileQuick)
	
	// Test MaskedAPIKey method
	t.Run("masked_api_key", func(t *testing.T) {
		plaintextKey := "usr_test1234567890abcdef1234567890abcdef"
		apiKey := &services.UserAPIKey{
			PlaintextKey: &plaintextKey,
		}
		
		masked := apiKey.MaskedAPIKey()
		assert.NotEqual(t, plaintextKey, masked)
		assert.Contains(t, masked, "****")
		
		// Check that the masked version is shorter than original
		assert.True(t, len(masked) <= len(plaintextKey), "Masked key should not be longer than original")
	})
	
	// Test MaskedAPIKey with nil plaintext key
	t.Run("masked_api_key_nil", func(t *testing.T) {
		apiKey := &services.UserAPIKey{
			PlaintextKey: nil,
		}
		
		masked := apiKey.MaskedAPIKey()
		assert.Equal(t, "****", masked)
	})
	
	// Test HasPermission method
	t.Run("has_permission", func(t *testing.T) {
		apiKey := &services.UserAPIKey{
			Permissions: []string{"read", "write"},
		}
		
		assert.True(t, apiKey.HasPermission("read"))
		assert.True(t, apiKey.HasPermission("write"))
		assert.False(t, apiKey.HasPermission("delete"))
		assert.False(t, apiKey.HasPermission("admin"))
	})
	
	// Test HasPermission with wildcard
	t.Run("has_permission_wildcard", func(t *testing.T) {
		apiKey := &services.UserAPIKey{
			Permissions: []string{"*"},
		}
		
		assert.True(t, apiKey.HasPermission("read"))
		assert.True(t, apiKey.HasPermission("write"))
		assert.True(t, apiKey.HasPermission("delete"))
		assert.True(t, apiKey.HasPermission("admin"))
		assert.True(t, apiKey.HasPermission("anything"))
	})
	
	// Test HasPermission with empty permissions
	t.Run("has_permission_empty", func(t *testing.T) {
		apiKey := &services.UserAPIKey{
			Permissions: []string{},
		}
		
		assert.False(t, apiKey.HasPermission("read"))
		assert.False(t, apiKey.HasPermission("write"))
	})
}

// TestSecurityService_APIKeyValidationResult tests APIKeyValidationResult struct
func TestSecurityService_APIKeyValidationResult(t *testing.T) {
	t.Run("valid_result", func(t *testing.T) {
		userID := uuid.New()
		apiKeyID := uuid.New()
		
		result := &services.APIKeyValidationResult{
			Valid:       true,
			UserID:      userID,
			APIKeyID:    apiKeyID,
			Permissions: []string{"read", "write"},
			Error:       "",
		}
		
		assert.True(t, result.Valid)
		assert.Equal(t, userID, result.UserID)
		assert.Equal(t, apiKeyID, result.APIKeyID)
		assert.Len(t, result.Permissions, 2)
		assert.Empty(t, result.Error)
	})
	
	t.Run("invalid_result", func(t *testing.T) {
		result := &services.APIKeyValidationResult{
			Valid: false,
			Error: "API key not found",
		}
		
		assert.False(t, result.Valid)
		assert.Equal(t, uuid.Nil, result.UserID)
		assert.Equal(t, uuid.Nil, result.APIKeyID)
		assert.Nil(t, result.Permissions)
		assert.Contains(t, result.Error, "not found")
	})
}

// TestSecurityService_TableName tests UserAPIKey table name method
func TestSecurityService_TableName(t *testing.T) {
	apiKey := &services.UserAPIKey{}
	tableName := apiKey.TableName()
	
	assert.Equal(t, "user_api_keys", tableName)
}

// TestSecurityService_UserAPIKey_Fields tests UserAPIKey field validation
func TestSecurityService_UserAPIKey_Fields(t *testing.T) {
	t.Run("create_user_api_key", func(t *testing.T) {
		userID := uuid.New()
		apiKeyID := uuid.New()
		plaintextKey := "usr_test1234567890abcdef"
		
		apiKey := &services.UserAPIKey{
			ID:           apiKeyID,
			UserID:       userID,
			Name:         "Test API Key",
			EncryptedKey: "encrypted_data_here",
			KeyHash:      "hash_of_key",
			Permissions:  []string{"read", "write"},
			IsActive:     true,
			PlaintextKey: &plaintextKey,
		}
		
		// Verify fields are set correctly
		assert.Equal(t, apiKeyID, apiKey.ID)
		assert.Equal(t, userID, apiKey.UserID)
		assert.Equal(t, "Test API Key", apiKey.Name)
		assert.NotEmpty(t, apiKey.EncryptedKey)
		assert.NotEmpty(t, apiKey.KeyHash)
		assert.Len(t, apiKey.Permissions, 2)
		assert.True(t, apiKey.IsActive)
		assert.NotNil(t, apiKey.PlaintextKey)
		assert.Equal(t, plaintextKey, *apiKey.PlaintextKey)
	})
	
	t.Run("api_key_without_plaintext", func(t *testing.T) {
		userID := uuid.New()
		
		apiKey := &services.UserAPIKey{
			UserID:       userID,
			Name:         "Stored API Key",
			EncryptedKey: "encrypted_data_here",
			KeyHash:      "hash_of_key",
			Permissions:  []string{"admin"},
			IsActive:     false,
			PlaintextKey: nil, // No plaintext key (normal for stored keys)
		}
		
		// Verify fields are set correctly
		assert.Equal(t, userID, apiKey.UserID)
		assert.Equal(t, "Stored API Key", apiKey.Name)
		assert.Len(t, apiKey.Permissions, 1)
		assert.Equal(t, "admin", apiKey.Permissions[0])
		assert.False(t, apiKey.IsActive)
		assert.Nil(t, apiKey.PlaintextKey)
		
		// Test masked key returns default when no plaintext
		masked := apiKey.MaskedAPIKey()
		assert.Equal(t, "****", masked)
	})
}

// TestSecurityService_PermissionLogic tests various permission scenarios
func TestSecurityService_PermissionLogic(t *testing.T) {
	t.Run("multiple_permissions", func(t *testing.T) {
		apiKey := &services.UserAPIKey{
			Permissions: []string{"read", "write", "delete"},
		}
		
		assert.True(t, apiKey.HasPermission("read"))
		assert.True(t, apiKey.HasPermission("write"))
		assert.True(t, apiKey.HasPermission("delete"))
		assert.False(t, apiKey.HasPermission("admin"))
		assert.False(t, apiKey.HasPermission("nonexistent"))
	})
	
	t.Run("admin_permission", func(t *testing.T) {
		apiKey := &services.UserAPIKey{
			Permissions: []string{"admin"},
		}
		
		assert.True(t, apiKey.HasPermission("admin"))
		assert.False(t, apiKey.HasPermission("read"))
		assert.False(t, apiKey.HasPermission("write"))
	})
	
	t.Run("wildcard_and_specific", func(t *testing.T) {
		apiKey := &services.UserAPIKey{
			Permissions: []string{"*", "read"},
		}
		
		// Wildcard should grant all permissions
		assert.True(t, apiKey.HasPermission("read"))
		assert.True(t, apiKey.HasPermission("write"))
		assert.True(t, apiKey.HasPermission("delete"))
		assert.True(t, apiKey.HasPermission("admin"))
		assert.True(t, apiKey.HasPermission("anything"))
	})
	
	t.Run("case_sensitive_permissions", func(t *testing.T) {
		apiKey := &services.UserAPIKey{
			Permissions: []string{"Read", "WRITE"},
		}
		
		// Permissions should be case-sensitive
		assert.True(t, apiKey.HasPermission("Read"))
		assert.True(t, apiKey.HasPermission("WRITE"))
		assert.False(t, apiKey.HasPermission("read"))
		assert.False(t, apiKey.HasPermission("write"))
	})
}

// TestSecurityService_UtilityFunctions tests various utility functions
func TestSecurityService_UtilityFunctions(t *testing.T) {
	t.Run("mask_sensitive_data_behavior", func(t *testing.T) {
		// Test the masking behavior without relying on the exact implementation
		plaintext := "usr_very_long_api_key_1234567890abcdef"
		
		// This tests the behavior that's expected in the UserAPIKey.MaskedAPIKey method
		masked := security.MaskSensitiveData(plaintext, 4)
		
		// Basic assertions about masking behavior
		assert.NotEqual(t, plaintext, masked, "Masked version should be different from original")
		assert.Contains(t, masked, "****", "Masked version should contain asterisks")
		
		// For very short strings, might return the original or partial
		shortText := "usr"
		maskedShort := security.MaskSensitiveData(shortText, 4)
		assert.NotEmpty(t, maskedShort, "Should handle short strings gracefully")
	})
	
	t.Run("empty_string_masking", func(t *testing.T) {
		empty := ""
		masked := security.MaskSensitiveData(empty, 4)
		assert.Equal(t, "", masked, "Empty string should remain empty")
	})
}

// Performance test for permission checking
func TestSecurityService_PermissionPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}
	
	t.Run("permission_check_performance", func(t *testing.T) {
		// Create API key with many permissions
		permissions := make([]string, 100)
		for i := 0; i < 100; i++ {
			permissions[i] = "permission_" + string(rune(i))
		}
		
		apiKey := &services.UserAPIKey{
			Permissions: permissions,
		}
		
		// Test permission checking performance
		startTime := time.Now()
		
		for i := 0; i < 1000; i++ {
			// Check existing permission
			apiKey.HasPermission("permission_50")
			// Check non-existing permission
			apiKey.HasPermission("non_existent_permission")
		}
		
		duration := time.Since(startTime)
		
		// Should complete quickly
		assert.Less(t, duration.Milliseconds(), int64(100),
			"1000 permission checks should complete within 100ms")
	})
	
	t.Run("wildcard_permission_performance", func(t *testing.T) {
		apiKey := &services.UserAPIKey{
			Permissions: []string{"*"},
		}
		
		startTime := time.Now()
		
		for i := 0; i < 1000; i++ {
			apiKey.HasPermission("any_permission")
		}
		
		duration := time.Since(startTime)
		
		// Wildcard checks should be very fast
		assert.Less(t, duration.Milliseconds(), int64(50),
			"1000 wildcard permission checks should complete within 50ms")
	})
}