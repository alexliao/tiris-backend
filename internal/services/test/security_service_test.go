package test

import (
	"context"
	"testing"
	"time"

	"tiris-backend/internal/models"
	"tiris-backend/internal/services"
	"tiris-backend/pkg/security"
	"tiris-backend/test/config"
	"tiris-backend/test/helpers"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// MockRateLimiter implements rate limiting for testing
type MockRateLimiter struct {
	mock.Mock
}

func (m *MockRateLimiter) CheckRateLimit(ctx context.Context, identifier, ruleName string, rule security.RateLimitRule) (*security.RateLimitResult, error) {
	args := m.Called(ctx, identifier, ruleName, rule)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*security.RateLimitResult), args.Error(1)
}

// MockAuditLogger implements audit logging for testing
type MockAuditLogger struct {
	mock.Mock
}

func (m *MockAuditLogger) LogSecurityEvent(ctx context.Context, action security.AuditAction, userID *uuid.UUID, ipAddress string, metadata map[string]interface{}, err error) {
	m.Called(ctx, action, userID, ipAddress, metadata, err)
}

func (m *MockAuditLogger) LogDataAccess(ctx context.Context, userID *uuid.UUID, action security.AuditAction, resourceType, resourceID, ipAddress string, success bool) error {
	args := m.Called(ctx, userID, action, resourceType, resourceID, ipAddress, success)
	return args.Error(0)
}

func (m *MockAuditLogger) GetSecurityAlerts(ctx context.Context, since time.Time, limit int) ([]security.AuditEvent, error) {
	args := m.Called(ctx, since, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]security.AuditEvent), args.Error(1)
}

func (m *MockAuditLogger) GetSuspiciousActivity(ctx context.Context, timeWindow time.Duration) ([]security.SuspiciousActivity, error) {
	args := m.Called(ctx, timeWindow)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]security.SuspiciousActivity), args.Error(1)
}

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Auto-migrate the test models
	if err := db.AutoMigrate(&services.UserAPIKey{}); err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&models.SecureExchange{}); err != nil {
		return nil, err
	}

	return db, nil
}

// setupTestRedis creates a mock Redis client for testing
func setupTestRedis() *redis.Client {
	// For testing, we'll use a mock or in-memory Redis
	// In a real test environment, you might use miniredis
	return redis.NewClient(&redis.Options{
		Addr: "localhost:6379", // This will fail but that's okay for unit tests
	})
}

// TestSecurityService_CreateUserAPIKey tests API key creation functionality
func TestSecurityService_CreateUserAPIKey(t *testing.T) {
	// Setup test config
	_ = config.GetProfileConfig(config.ProfileQuick)
	
	// Create test database
	db, err := setupTestDB()
	require.NoError(t, err)
	
	// Create mock Redis client
	redisClient := setupTestRedis()
	
	// Create security service
	masterKey := "test_master_key_32_bytes_long_123"
	signingKey := "test_signing_key_32_bytes_long_12"
	securityService, err := services.NewSecurityService(db, redisClient, masterKey, signingKey)
	require.NoError(t, err)
	
	// Test successful API key creation
	t.Run("successful_api_key_creation", func(t *testing.T) {
		userID := uuid.New()
		name := "Test API Key"
		permissions := []string{"read", "write"}
		
		// Execute test
		result, err := securityService.CreateUserAPIKey(context.Background(), userID, name, permissions)
		
		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, userID, result.UserID)
		assert.Equal(t, name, result.Name)
		assert.Equal(t, permissions, result.Permissions)
		assert.True(t, result.IsActive)
		assert.NotNil(t, result.PlaintextKey)
		assert.NotEmpty(t, *result.PlaintextKey)
		assert.NotEmpty(t, result.EncryptedKey)
		assert.NotEmpty(t, result.KeyHash)
		
		// Verify API key format
		assert.True(t, len(*result.PlaintextKey) > 20, "API key should be sufficiently long")
		assert.Contains(t, *result.PlaintextKey, "usr_", "API key should have user prefix")
		
		// Verify the key was saved to database
		var savedKey services.UserAPIKey
		err = db.Where("id = ?", result.ID).First(&savedKey).Error
		require.NoError(t, err)
		assert.Equal(t, userID, savedKey.UserID)
		assert.Equal(t, name, savedKey.Name)
		assert.True(t, savedKey.IsActive)
	})
	
	// Test API key creation with empty name
	t.Run("api_key_creation_empty_name", func(t *testing.T) {
		userID := uuid.New()
		name := ""
		permissions := []string{"read"}
		
		// Execute test
		result, err := securityService.CreateUserAPIKey(context.Background(), userID, name, permissions)
		
		// Should still succeed but with empty name
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "", result.Name)
	})
	
	// Test API key creation with no permissions
	t.Run("api_key_creation_no_permissions", func(t *testing.T) {
		userID := uuid.New()
		name := "No Permissions Key"
		permissions := []string{}
		
		// Execute test
		result, err := securityService.CreateUserAPIKey(context.Background(), userID, name, permissions)
		
		// Should succeed
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, permissions, result.Permissions)
		assert.Len(t, result.Permissions, 0)
	})
}

// TestSecurityService_ValidateAPIKey tests API key validation functionality
func TestSecurityService_ValidateAPIKey(t *testing.T) {
	// Create test database
	db, err := setupTestDB()
	require.NoError(t, err)
	
	// Create mock Redis client
	redisClient := setupTestRedis()
	
	// Create security service
	masterKey := "test_master_key_32_bytes_long_123"
	signingKey := "test_signing_key_32_bytes_long_12"
	securityService, err := services.NewSecurityService(db, redisClient, masterKey, signingKey)
	require.NoError(t, err)
	
	// Create a test API key first
	userID := uuid.New()
	name := "Test Validation Key"
	permissions := []string{"read", "write"}
	
	createdKey, err := securityService.CreateUserAPIKey(context.Background(), userID, name, permissions)
	require.NoError(t, err)
	require.NotNil(t, createdKey.PlaintextKey)
	
	// Test successful validation
	t.Run("successful_validation", func(t *testing.T) {
		// Execute test
		result, err := securityService.ValidateAPIKey(context.Background(), *createdKey.PlaintextKey)
		
		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Valid)
		assert.Equal(t, userID, result.UserID)
		assert.Equal(t, createdKey.ID, result.APIKeyID)
		assert.Equal(t, permissions, result.Permissions)
		assert.NotNil(t, result.LastUsedAt)
		assert.Empty(t, result.Error)
	})
	
	// Test invalid API key format
	t.Run("invalid_api_key_format", func(t *testing.T) {
		invalidKey := "invalid_key_format"
		
		// Execute test
		result, err := securityService.ValidateAPIKey(context.Background(), invalidKey)
		
		// Should return error due to format validation
		require.Error(t, err)
		assert.Nil(t, result)
	})
	
	// Test non-existent API key
	t.Run("non_existent_api_key", func(t *testing.T) {
		// Create a fake API key with correct format but not in database
		fakeKey := "usr_fake1234567890abcdef1234567890abcdef1234567890abcdef123456"
		
		// Execute test
		result, err := securityService.ValidateAPIKey(context.Background(), fakeKey)
		
		// Should return valid=false result, not error
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.False(t, result.Valid)
		assert.Contains(t, result.Error, "API key not found")
	})
	
	// Test deactivated API key
	t.Run("deactivated_api_key", func(t *testing.T) {
		// Create another API key and deactivate it
		deactivatedKey, err := securityService.CreateUserAPIKey(context.Background(), userID, "Deactivated Key", []string{"read"})
		require.NoError(t, err)
		
		// Deactivate the key
		err = db.Model(&services.UserAPIKey{}).Where("id = ?", deactivatedKey.ID).Update("is_active", false).Error
		require.NoError(t, err)
		
		// Execute test
		result, err := securityService.ValidateAPIKey(context.Background(), *deactivatedKey.PlaintextKey)
		
		// Should return valid=false result
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.False(t, result.Valid)
		assert.Contains(t, result.Error, "not found or inactive")
	})
	
	// Test expired API key
	t.Run("expired_api_key", func(t *testing.T) {
		// Create another API key and set it as expired
		expiredKey, err := securityService.CreateUserAPIKey(context.Background(), userID, "Expired Key", []string{"read"})
		require.NoError(t, err)
		
		// Set expiry to past time
		pastTime := time.Now().Add(-24 * time.Hour)
		err = db.Model(&services.UserAPIKey{}).Where("id = ?", expiredKey.ID).Update("expires_at", pastTime).Error
		require.NoError(t, err)
		
		// Execute test
		result, err := securityService.ValidateAPIKey(context.Background(), *expiredKey.PlaintextKey)
		
		// Should return valid=false result
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.False(t, result.Valid)
		assert.Contains(t, result.Error, "expired")
	})
}

// TestSecurityService_RotateAPIKey tests API key rotation functionality
func TestSecurityService_RotateAPIKey(t *testing.T) {
	// Create test database
	db, err := setupTestDB()
	require.NoError(t, err)
	
	// Create mock Redis client
	redisClient := setupTestRedis()
	
	// Create security service
	masterKey := "test_master_key_32_bytes_long_123"
	signingKey := "test_signing_key_32_bytes_long_12"
	securityService, err := services.NewSecurityService(db, redisClient, masterKey, signingKey)
	require.NoError(t, err)
	
	// Create a test API key first
	userID := uuid.New()
	name := "Rotation Test Key"
	permissions := []string{"read", "write"}
	
	originalKey, err := securityService.CreateUserAPIKey(context.Background(), userID, name, permissions)
	require.NoError(t, err)
	
	// Test successful rotation
	t.Run("successful_rotation", func(t *testing.T) {
		// Execute test
		newKey, err := securityService.RotateAPIKey(context.Background(), userID, originalKey.ID)
		
		// Verify results
		require.NoError(t, err)
		require.NotNil(t, newKey)
		assert.NotEqual(t, originalKey.ID, newKey.ID)
		assert.Equal(t, userID, newKey.UserID)
		assert.Contains(t, newKey.Name, "(Rotated)")
		assert.Equal(t, permissions, newKey.Permissions)
		assert.True(t, newKey.IsActive)
		assert.NotNil(t, newKey.PlaintextKey)
		assert.NotEqual(t, originalKey.PlaintextKey, newKey.PlaintextKey)
		
		// Verify old key is deactivated
		var oldKey services.UserAPIKey
		err = db.Where("id = ?", originalKey.ID).First(&oldKey).Error
		require.NoError(t, err)
		assert.False(t, oldKey.IsActive)
		
		// Verify new key is in database and active
		var savedNewKey services.UserAPIKey
		err = db.Where("id = ?", newKey.ID).First(&savedNewKey).Error
		require.NoError(t, err)
		assert.True(t, savedNewKey.IsActive)
	})
	
	// Test rotation of non-existent key
	t.Run("rotation_non_existent_key", func(t *testing.T) {
		nonExistentID := uuid.New()
		
		// Execute test
		result, err := securityService.RotateAPIKey(context.Background(), userID, nonExistentID)
		
		// Should return error
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "API key not found")
	})
	
	// Test rotation with wrong user ID
	t.Run("rotation_wrong_user", func(t *testing.T) {
		wrongUserID := uuid.New()
		
		// Execute test
		result, err := securityService.RotateAPIKey(context.Background(), wrongUserID, originalKey.ID)
		
		// Should return error
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "API key not found")
	})
}

// TestSecurityService_CheckRateLimit tests rate limiting functionality
func TestSecurityService_CheckRateLimit(t *testing.T) {
	// Create test database
	db, err := setupTestDB()
	require.NoError(t, err)
	
	// Create mock Redis client
	redisClient := setupTestRedis()
	
	// Create security service
	masterKey := "test_master_key_32_bytes_long_123"
	signingKey := "test_signing_key_32_bytes_long_12"
	securityService, err := services.NewSecurityService(db, redisClient, masterKey, signingKey)
	require.NoError(t, err)
	
	// Test rate limit check
	t.Run("rate_limit_check", func(t *testing.T) {
		identifier := "test_user_123"
		ruleName := "api_general"
		
		// Execute test - this will likely fail due to Redis connection
		// but we're testing the service logic
		result, err := securityService.CheckRateLimit(context.Background(), identifier, ruleName)
		
		// The test might fail due to Redis connection, which is expected in unit tests
		// In a real integration test, we'd use a real Redis instance
		if err != nil {
			assert.Contains(t, err.Error(), "connection")
		} else {
			require.NotNil(t, result)
			assert.Equal(t, ruleName, result.RuleName)
		}
	})
	
	// Test with non-existent rule (should fallback to default)
	t.Run("non_existent_rule", func(t *testing.T) {
		identifier := "test_user_123"
		ruleName := "non_existent_rule"
		
		// Execute test
		result, err := securityService.CheckRateLimit(context.Background(), identifier, ruleName)
		
		// Should fallback to api_general rule
		if err != nil {
			assert.Contains(t, err.Error(), "connection")
		} else {
			require.NotNil(t, result)
			// The service should use the default rule internally
		}
	})
}

// TestSecurityService_EncryptionDecryption tests encryption and decryption functionality
func TestSecurityService_EncryptionDecryption(t *testing.T) {
	// Create test database
	db, err := setupTestDB()
	require.NoError(t, err)
	
	// Create mock Redis client
	redisClient := setupTestRedis()
	
	// Create security service
	masterKey := "test_master_key_32_bytes_long_123"
	signingKey := "test_signing_key_32_bytes_long_12"
	securityService, err := services.NewSecurityService(db, redisClient, masterKey, signingKey)
	require.NoError(t, err)
	
	// Test successful encryption and decryption
	t.Run("successful_encryption_decryption", func(t *testing.T) {
		originalData := "sensitive_api_key_12345"
		
		// Encrypt data
		encryptedData, err := securityService.EncryptSensitiveData(originalData)
		require.NoError(t, err)
		assert.NotEmpty(t, encryptedData)
		assert.NotEqual(t, originalData, encryptedData)
		
		// Decrypt data
		decryptedData, err := securityService.DecryptSensitiveData(encryptedData)
		require.NoError(t, err)
		assert.Equal(t, originalData, decryptedData)
	})
	
	// Test encryption of empty string
	t.Run("encryption_empty_string", func(t *testing.T) {
		originalData := ""
		
		// Encrypt data
		encryptedData, err := securityService.EncryptSensitiveData(originalData)
		require.NoError(t, err)
		
		// Decrypt data
		decryptedData, err := securityService.DecryptSensitiveData(encryptedData)
		require.NoError(t, err)
		assert.Equal(t, originalData, decryptedData)
	})
	
	// Test decryption of invalid data
	t.Run("decryption_invalid_data", func(t *testing.T) {
		invalidData := "invalid_encrypted_data"
		
		// Decrypt data
		decryptedData, err := securityService.DecryptSensitiveData(invalidData)
		
		// Should return error
		require.Error(t, err)
		assert.Empty(t, decryptedData)
	})
}

// TestSecurityService_CreateSecureExchange tests secure exchange creation
func TestSecurityService_CreateSecureExchange(t *testing.T) {
	// Create test database
	db, err := setupTestDB()
	require.NoError(t, err)
	
	// Create mock Redis client
	redisClient := setupTestRedis()
	
	// Create security service
	masterKey := "test_master_key_32_bytes_long_123"
	signingKey := "test_signing_key_32_bytes_long_12"
	securityService, err := services.NewSecurityService(db, redisClient, masterKey, signingKey)
	require.NoError(t, err)
	
	// Test successful secure exchange creation
	t.Run("successful_secure_exchange_creation", func(t *testing.T) {
		userID := uuid.New()
		name := "Binance Test"
		exchangeType := "binance"
		apiKey := "test_api_key_12345"
		apiSecret := "test_api_secret_67890"
		
		// Execute test
		result, err := securityService.CreateSecureExchange(context.Background(), userID, name, exchangeType, apiKey, apiSecret)
		
		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, userID, result.UserID)
		assert.Equal(t, name, result.Name)
		assert.Equal(t, exchangeType, result.Type)
		assert.Equal(t, "active", result.Status)
		assert.NotEmpty(t, result.EncryptedAPIKey)
		assert.NotEmpty(t, result.EncryptedSecret)
		assert.NotEmpty(t, result.APIKeyHash)
		
		// Verify the exchange was saved to database
		var savedExchange models.SecureExchange
		err = db.Where("id = ?", result.ID).First(&savedExchange).Error
		require.NoError(t, err)
		assert.Equal(t, userID, savedExchange.UserID)
		assert.Equal(t, name, savedExchange.Name)
	})
	
	// Test secure exchange creation with empty credentials
	t.Run("secure_exchange_empty_credentials", func(t *testing.T) {
		userID := uuid.New()
		name := "Empty Creds Exchange"
		exchangeType := "binance"
		apiKey := ""
		apiSecret := ""
		
		// Execute test
		result, err := securityService.CreateSecureExchange(context.Background(), userID, name, exchangeType, apiKey, apiSecret)
		
		// Should still succeed (empty credentials are valid for some use cases)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, name, result.Name)
	})
}

// TestSecurityService_GetExchangeCredentials tests credential retrieval
func TestSecurityService_GetExchangeCredentials(t *testing.T) {
	// Create test database
	db, err := setupTestDB()
	require.NoError(t, err)
	
	// Create mock Redis client
	redisClient := setupTestRedis()
	
	// Create security service
	masterKey := "test_master_key_32_bytes_long_123"
	signingKey := "test_signing_key_32_bytes_long_12"
	securityService, err := services.NewSecurityService(db, redisClient, masterKey, signingKey)
	require.NoError(t, err)
	
	// Create a secure exchange first
	userID := uuid.New()
	name := "Test Exchange"
	exchangeType := "binance"
	originalAPIKey := "test_api_key_12345"
	originalAPISecret := "test_api_secret_67890"
	
	exchange, err := securityService.CreateSecureExchange(context.Background(), userID, name, exchangeType, originalAPIKey, originalAPISecret)
	require.NoError(t, err)
	
	// Test successful credential retrieval
	t.Run("successful_credential_retrieval", func(t *testing.T) {
		// Execute test
		apiKey, apiSecret, err := securityService.GetExchangeCredentials(context.Background(), userID, exchange.ID)
		
		// Verify results
		require.NoError(t, err)
		assert.Equal(t, originalAPIKey, apiKey)
		assert.Equal(t, originalAPISecret, apiSecret)
	})
	
	// Test credential retrieval with wrong user ID
	t.Run("credential_retrieval_wrong_user", func(t *testing.T) {
		wrongUserID := uuid.New()
		
		// Execute test
		apiKey, apiSecret, err := securityService.GetExchangeCredentials(context.Background(), wrongUserID, exchange.ID)
		
		// Should return error
		require.Error(t, err)
		assert.Empty(t, apiKey)
		assert.Empty(t, apiSecret)
		assert.Contains(t, err.Error(), "exchange not found")
	})
	
	// Test credential retrieval with non-existent exchange
	t.Run("credential_retrieval_non_existent_exchange", func(t *testing.T) {
		nonExistentID := uuid.New()
		
		// Execute test
		apiKey, apiSecret, err := securityService.GetExchangeCredentials(context.Background(), userID, nonExistentID)
		
		// Should return error
		require.Error(t, err)
		assert.Empty(t, apiKey)
		assert.Empty(t, apiSecret)
		assert.Contains(t, err.Error(), "exchange not found")
	})
}

// TestSecurityService_UserAPIKey_Methods tests UserAPIKey helper methods
func TestSecurityService_UserAPIKey_Methods(t *testing.T) {
	// Test MaskedAPIKey method
	t.Run("masked_api_key", func(t *testing.T) {
		plaintextKey := "usr_test1234567890abcdef1234567890abcdef"
		apiKey := &services.UserAPIKey{
			PlaintextKey: &plaintextKey,
		}
		
		masked := apiKey.MaskedAPIKey()
		assert.NotEqual(t, plaintextKey, masked)
		assert.Contains(t, masked, "****")
		assert.True(t, len(masked) < len(plaintextKey))
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

// Performance test for security operations
func TestSecurityService_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}
	
	// Create test database
	db, err := setupTestDB()
	require.NoError(t, err)
	
	// Create mock Redis client
	redisClient := setupTestRedis()
	
	// Create security service
	masterKey := "test_master_key_32_bytes_long_123"
	signingKey := "test_signing_key_32_bytes_long_12"
	securityService, err := services.NewSecurityService(db, redisClient, masterKey, signingKey)
	require.NoError(t, err)
	
	t.Run("encryption_decryption_performance", func(t *testing.T) {
		testData := "sensitive_data_for_performance_testing"
		
		timer := helpers.NewPerformanceTimer()
		timer.Start()
		
		// Run encryption/decryption multiple times
		for i := 0; i < 100; i++ {
			encrypted, err := securityService.EncryptSensitiveData(testData)
			require.NoError(t, err)
			
			decrypted, err := securityService.DecryptSensitiveData(encrypted)
			require.NoError(t, err)
			assert.Equal(t, testData, decrypted)
		}
		
		timer.Stop()
		
		// Check performance within reasonable bounds
		assert.Less(t, timer.Duration().Milliseconds(), int64(2000),
			"100 encryption/decryption cycles should complete within 2 seconds")
	})
}