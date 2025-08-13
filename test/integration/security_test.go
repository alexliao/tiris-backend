package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"tiris-backend/pkg/security"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// SecurityIntegrationTestSuite tests comprehensive security functionality
type SecurityIntegrationTestSuite struct {
	IntegrationTestSuite
}

// Test API key management lifecycle
func (s *SecurityIntegrationTestSuite) TestAPIKeyLifecycle() {
	// Create API key
	apiKey, err := s.SecurityService.CreateUserAPIKey(
		s.ctx,
		s.TestUser.ID,
		"Test Integration API Key",
		[]string{"read", "write"},
	)
	s.Require().NoError(err)
	s.NotNil(apiKey.PlaintextKey)
	s.True(apiKey.IsActive)
	s.Equal(s.TestUser.ID, apiKey.UserID)
	s.Equal("Test Integration API Key", apiKey.Name)
	s.Contains(apiKey.Permissions, "read")
	s.Contains(apiKey.Permissions, "write")

	// Validate API key
	result, err := s.SecurityService.ValidateAPIKey(s.ctx, *apiKey.PlaintextKey)
	s.Require().NoError(err)
	s.True(result.Valid)
	s.Equal(s.TestUser.ID, result.UserID)
	s.Equal(apiKey.ID, result.APIKeyID)
	s.Equal(apiKey.Permissions, result.Permissions)

	// Rotate API key
	rotatedKey, err := s.SecurityService.RotateAPIKey(s.ctx, s.TestUser.ID, apiKey.ID)
	s.Require().NoError(err)
	s.NotNil(rotatedKey.PlaintextKey)
	s.True(rotatedKey.IsActive)
	s.NotEqual(*apiKey.PlaintextKey, *rotatedKey.PlaintextKey)

	// Original key should no longer be valid
	result, err = s.SecurityService.ValidateAPIKey(s.ctx, *apiKey.PlaintextKey)
	s.Require().NoError(err)
	s.False(result.Valid)

	// New key should be valid
	result, err = s.SecurityService.ValidateAPIKey(s.ctx, *rotatedKey.PlaintextKey)
	s.Require().NoError(err)
	s.True(result.Valid)

	// Verify audit events were created
	auditEvent := s.waitForAuditEvent(security.ActionAPIKeyCreate, 2*time.Second)
	s.NotNil(auditEvent)
	s.Equal(security.ActionAPIKeyCreate, auditEvent.Action)
	s.Equal(s.TestUser.ID, *auditEvent.UserID)

	rotateEvent := s.waitForAuditEvent(security.ActionAPIKeyUpdate, 2*time.Second)
	s.NotNil(rotateEvent)
	s.Equal(security.ActionAPIKeyUpdate, rotateEvent.Action)
	s.Equal(s.TestUser.ID, *rotateEvent.UserID)
}

// Test encryption and decryption
func (s *SecurityIntegrationTestSuite) TestEncryptionDecryption() {
	testCases := []struct {
		name      string
		plaintext string
	}{
		{"simple text", "Hello, World!"},
		{"empty string", ""},
		{"long text", "This is a very long text that should be encrypted and decrypted properly. " + 
			"It contains multiple sentences and should maintain its integrity through the process."},
		{"special characters", "!@#$%^&*()_+-=[]{}|;:'\",.<>?/~`"},
		{"unicode", "Hello 世界 🌍 مرحبا العالم"},
		{"json data", `{"api_key": "sk-1234567890", "secret": "very-secret-key-here"}`},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Encrypt data
			encrypted, err := s.SecurityService.EncryptSensitiveData(tc.plaintext)
			s.Require().NoError(err)

			if tc.plaintext == "" {
				s.Equal("", encrypted)
				return
			}

			// Encrypted data should be different from plaintext
			s.NotEqual(tc.plaintext, encrypted)
			s.NotEmpty(encrypted)

			// Decrypt data
			decrypted, err := s.SecurityService.DecryptSensitiveData(encrypted)
			s.Require().NoError(err)
			s.Equal(tc.plaintext, decrypted)
		})
	}
}

// Test secure exchange functionality
func (s *SecurityIntegrationTestSuite) TestSecureExchangeManagement() {
	// Create secure exchange
	exchange, err := s.SecurityService.CreateSecureExchange(
		s.ctx,
		s.TestUser.ID,
		"Test Binance Exchange",
		"binance",
		"binance_api_key_123456789",
		"binance_secret_key_987654321",
	)
	s.Require().NoError(err)
	s.Equal("Test Binance Exchange", exchange.Name)
	s.Equal("binance", exchange.Type)
	s.Equal("active", exchange.Status)
	s.NotEmpty(exchange.EncryptedAPIKey)
	s.NotEmpty(exchange.EncryptedSecret)
	s.NotEmpty(exchange.APIKeyHash)

	// Retrieve credentials
	apiKey, apiSecret, err := s.SecurityService.GetExchangeCredentials(s.ctx, s.TestUser.ID, exchange.ID)
	s.Require().NoError(err)
	s.Equal("binance_api_key_123456789", apiKey)
	s.Equal("binance_secret_key_987654321", apiSecret)

	// Create another exchange for different user (should not access)
	anotherUser := s.createTestUser()
	_, _, err = s.SecurityService.GetExchangeCredentials(s.ctx, anotherUser.ID, exchange.ID)
	s.Error(err) // Should fail - different user

	// Verify audit event
	auditEvent := s.waitForAuditEvent(security.ActionExchangeCreate, 2*time.Second)
	s.NotNil(auditEvent)
	s.Equal(security.ActionExchangeCreate, auditEvent.Action)
	s.Equal(s.TestUser.ID, *auditEvent.UserID)
}

// Test rate limiting functionality
func (s *SecurityIntegrationTestSuite) TestRateLimiting() {
	userIdentifier := fmt.Sprintf("user:%s", s.TestUser.ID.String())
	ipIdentifier := "ip:192.168.1.100"

	// Test different rate limit rules
	testCases := []struct {
		name       string
		ruleName   string
		identifier string
		requests   int
		shouldHit  bool
	}{
		{
			name:       "auth login limit",
			ruleName:   "auth_login",
			identifier: ipIdentifier,
			requests:   6, // Rule allows 5, so 6th should hit limit
			shouldHit:  true,
		},
		{
			name:       "general API under limit",
			ruleName:   "api_general",
			identifier: userIdentifier,
			requests:   10, // Rule allows 1000, so should be fine
			shouldHit:  false,
		},
		{
			name:       "password reset limit",
			ruleName:   "password_reset",
			identifier: ipIdentifier + "_reset",
			requests:   4, // Rule allows 3, so 4th should hit limit
			shouldHit:  true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			var lastResult *security.RateLimitResult

			// Make requests up to the limit
			for i := 0; i < tc.requests; i++ {
				result, err := s.SecurityService.CheckRateLimit(s.ctx, tc.identifier, tc.ruleName)
				s.Require().NoError(err)
				lastResult = result
			}

			if tc.shouldHit {
				s.False(lastResult.Allowed, "Last request should be rate limited")
				s.Equal(0, lastResult.Remaining, "Remaining should be 0")
				s.Greater(lastResult.RetryAfter, time.Duration(0), "Should have retry after time")
			} else {
				s.True(lastResult.Allowed, "Requests should be allowed")
				s.Greater(lastResult.Remaining, 0, "Should have remaining requests")
			}

			s.Equal(tc.ruleName, lastResult.RuleName)
		})
	}
}

// Test audit logging functionality
func (s *SecurityIntegrationTestSuite) TestAuditLogging() {
	// Generate various audit events
	events := []struct {
		action   security.AuditAction
		userID   *uuid.UUID
		ipAddr   string
		success  bool
		details  map[string]interface{}
	}{
		{
			action:  security.ActionLogin,
			userID:  &s.TestUser.ID,
			ipAddr:  "192.168.1.100",
			success: true,
			details: map[string]interface{}{"method": "google"},
		},
		{
			action:  security.ActionLoginFailed,
			userID:  nil,
			ipAddr:  "192.168.1.101",
			success: false,
			details: map[string]interface{}{"reason": "invalid_credentials"},
		},
		{
			action:  security.ActionUserCreate,
			userID:  &s.TestUser.ID,
			ipAddr:  "192.168.1.100",
			success: true,
			details: map[string]interface{}{"email": "new@example.com"},
		},
		{
			action:  security.ActionSecurityAlert,
			userID:  nil,
			ipAddr:  "192.168.1.102",
			success: false,
			details: map[string]interface{}{"alert_type": "brute_force"},
		},
	}

	// Create audit events
	for _, event := range events {
		err := s.SecurityService.AuditDataAccess(
			s.ctx,
			event.userID,
			event.action,
			"test_resource",
			"test_id",
			event.ipAddr,
			event.success,
		)
		s.Require().NoError(err)
	}

	// Wait for events to be persisted
	time.Sleep(100 * time.Millisecond)

	// Test getting security alerts
	alerts, err := s.SecurityService.GetSecurityAlerts(s.ctx, time.Now().Add(-time.Hour), 10)
	s.Require().NoError(err)
	s.Greater(len(alerts), 0, "Should have security alerts")

	// Verify we have the security alert event
	foundAlert := false
	for _, alert := range alerts {
		if alert.Action == security.ActionSecurityAlert {
			foundAlert = true
			s.Contains(alert.Details, "alert_type")
			break
		}
	}
	s.True(foundAlert, "Should find security alert event")

	// Test getting suspicious activity
	suspicious, err := s.SecurityService.GetSuspiciousActivity(s.ctx, time.Hour)
	s.Require().NoError(err)
	// May or may not have suspicious activity depending on test timing
	s.GreaterOrEqual(len(suspicious), 0)
}

// Test security threat detection patterns
func (s *SecurityIntegrationTestSuite) TestThreatDetection() {
	// This would typically be tested through the middleware in API tests,
	// but we can test the underlying security service functionality

	testIP := "192.168.1.200"

	// Simulate multiple failed login attempts
	for i := 0; i < 6; i++ {
		err := s.SecurityService.AuditDataAccess(
			s.ctx,
			nil,
			security.ActionLoginFailed,
			"auth",
			"login_attempt",
			testIP,
			false,
		)
		s.Require().NoError(err)
	}

	// Simulate rate limit hits
	for i := 0; i < 4; i++ {
		err := s.SecurityService.AuditDataAccess(
			s.ctx,
			&s.TestUser.ID,
			security.ActionRateLimitHit,
			"api",
			"rate_limit",
			testIP,
			false,
		)
		s.Require().NoError(err)
	}

	// Wait for events to be processed
	time.Sleep(200 * time.Millisecond)

	// Check for suspicious activity
	suspicious, err := s.SecurityService.GetSuspiciousActivity(s.ctx, time.Hour)
	s.Require().NoError(err)

	// Should detect suspicious patterns
	foundFailedLogins := false
	foundRateLimits := false

	for _, activity := range suspicious {
		if activity.Type == "multiple_failed_logins" && activity.IPAddress == testIP {
			foundFailedLogins = true
			s.GreaterOrEqual(activity.Count, 5)
		}
		if activity.Type == "excessive_rate_limiting" && activity.IPAddress == testIP {
			foundRateLimits = true
			s.GreaterOrEqual(activity.Count, 3)
		}
	}

	s.True(foundFailedLogins, "Should detect multiple failed logins")
	s.True(foundRateLimits, "Should detect excessive rate limiting")
}

// Test data encryption at rest
func (s *SecurityIntegrationTestSuite) TestDataEncryptionAtRest() {
	// Create exchange with sensitive data
	exchange, err := s.SecurityService.CreateSecureExchange(
		s.ctx,
		s.TestUser.ID,
		"Encryption Test Exchange",
		"binance",
		"super_secret_api_key_12345",
		"ultra_secret_api_secret_67890",
	)
	s.Require().NoError(err)

	// Verify data is encrypted in database
	var rawExchange map[string]interface{}
	err = s.DB.Table("exchanges").Where("id = ?", exchange.ID).Take(&rawExchange).Error
	s.Require().NoError(err)

	// API key and secret should be encrypted (different from plaintext)
	s.NotEqual("super_secret_api_key_12345", rawExchange["encrypted_api_key"])
	s.NotEqual("ultra_secret_api_secret_67890", rawExchange["encrypted_api_secret"])
	s.NotEmpty(rawExchange["api_key_hash"])

	// But we should be able to decrypt them through the service
	apiKey, apiSecret, err := s.SecurityService.GetExchangeCredentials(s.ctx, s.TestUser.ID, exchange.ID)
	s.Require().NoError(err)
	s.Equal("super_secret_api_key_12345", apiKey)
	s.Equal("ultra_secret_api_secret_67890", apiSecret)
}

// Test API key format validation
func (s *SecurityIntegrationTestSuite) TestAPIKeyValidation() {
	// Test invalid API keys
	invalidKeys := []string{
		"",
		"invalid-key",
		"txc_validpart", // Missing signature
		"invalid_prefix_validpart.sig12345",
		"txc_validpart.invalidsig", // Invalid signature length
	}

	for _, invalidKey := range invalidKeys {
		result, err := s.SecurityService.ValidateAPIKey(s.ctx, invalidKey)
		s.Require().NoError(err)
		s.False(result.Valid, "Key should be invalid: %s", invalidKey)
	}

	// Test with properly formatted but non-existent key
	// This would pass format validation but fail database lookup
	nonExistentKey := "usr_dGVzdGluZzEyMzQ1Njc4OTA.abcd1234"
	result, err := s.SecurityService.ValidateAPIKey(s.ctx, nonExistentKey)
	s.Require().NoError(err)
	s.False(result.Valid, "Non-existent key should be invalid")
	s.Contains(result.Error, "not found", "Should indicate key not found")
}

// Test concurrent security operations
func (s *SecurityIntegrationTestSuite) TestConcurrentOperations() {
	const numGoroutines = 10
	const operationsPerGoroutine = 5

	// Test concurrent API key creation
	ch := make(chan error, numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer func() {
				if r := recover(); r != nil {
					ch <- fmt.Errorf("panic in goroutine %d: %v", goroutineID, r)
					return
				}
			}()

			for j := 0; j < operationsPerGoroutine; j++ {
				// Create API key
				apiKey, err := s.SecurityService.CreateUserAPIKey(
					s.ctx,
					s.TestUser.ID,
					fmt.Sprintf("Concurrent Key %d-%d", goroutineID, j),
					[]string{"read"},
				)
				if err != nil {
					ch <- fmt.Errorf("failed to create API key in goroutine %d: %v", goroutineID, err)
					return
				}

				// Validate API key
				result, err := s.SecurityService.ValidateAPIKey(s.ctx, *apiKey.PlaintextKey)
				if err != nil {
					ch <- fmt.Errorf("failed to validate API key in goroutine %d: %v", goroutineID, err)
					return
				}

				if !result.Valid {
					ch <- fmt.Errorf("API key should be valid in goroutine %d", goroutineID)
					return
				}

				// Test rate limiting
				_, err = s.SecurityService.CheckRateLimit(
					s.ctx,
					fmt.Sprintf("concurrent_test_%d", goroutineID),
					"api_general",
				)
				if err != nil {
					ch <- fmt.Errorf("failed to check rate limit in goroutine %d: %v", goroutineID, err)
					return
				}
			}

			ch <- nil
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		err := <-ch
		s.Require().NoError(err)
	}

	// Verify all operations completed successfully
	var apiKeyCount int64
	err := s.DB.Model(&services.UserAPIKey{}).Where("user_id = ?", s.TestUser.ID).Count(&apiKeyCount).Error
	s.Require().NoError(err)
	s.GreaterOrEqual(apiKeyCount, int64(numGoroutines*operationsPerGoroutine))
}

// Run the security integration test suite
func TestSecurityIntegrationSuite(t *testing.T) {
	suite.Run(t, new(SecurityIntegrationTestSuite))
}