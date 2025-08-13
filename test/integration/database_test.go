package integration

import (
	"context"
	"testing"
	"time"

	"tiris-backend/internal/models"
	"tiris-backend/internal/services"
	"tiris-backend/pkg/security"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// DatabaseIntegrationTestSuite tests database operations and data integrity
type DatabaseIntegrationTestSuite struct {
	IntegrationTestSuite
}

// Test complete user data lifecycle
func (s *DatabaseIntegrationTestSuite) TestUserDataLifecycle() {
	// Create user
	user := &models.User{
		ID:       uuid.New(),
		Username: "test_user_lifecycle",
		Email:    "lifecycle@example.com",
		Avatar:   nil,
		Settings: nil,
		Info:     nil,
	}

	err := s.DB.Create(user).Error
	s.Require().NoError(err)

	// Verify user creation
	var retrievedUser models.User
	err = s.DB.Where("id = ?", user.ID).First(&retrievedUser).Error
	s.Require().NoError(err)
	s.Equal(user.Username, retrievedUser.Username)
	s.Equal(user.Email, retrievedUser.Email)

	// Update user
	user.Username = "updated_username"
	err = s.DB.Save(user).Error
	s.Require().NoError(err)

	// Verify update
	err = s.DB.Where("id = ?", user.ID).First(&retrievedUser).Error
	s.Require().NoError(err)
	s.Equal("updated_username", retrievedUser.Username)

	// Create related data (exchanges, sub-accounts, transactions)
	exchange := &models.SecureExchange{
		ID:               uuid.New(),
		UserID:           user.ID,
		Name:             "Test Exchange",
		Type:             "binance",
		EncryptedAPIKey:  "encrypted_key_data",
		EncryptedSecret:  "encrypted_secret_data",
		APIKeyHash:       "hash_value",
		Status:           "active",
		SecuritySettings: nil,
	}

	err = s.DB.Create(exchange).Error
	s.Require().NoError(err)

	subAccount := &models.SubAccount{
		ID:         uuid.New(),
		UserID:     user.ID,
		ExchangeID: exchange.ID,
		Name:       "Main Account",
		Symbol:     "USDT",
		Balance:    1000.0,
		Info:       nil,
	}

	err = s.DB.Create(subAccount).Error
	s.Require().NoError(err)

	transaction := &models.Transaction{
		ID:             uuid.New(),
		UserID:         user.ID,
		ExchangeID:     exchange.ID,
		SubAccountID:   subAccount.ID,
		Timestamp:      time.Now(),
		Direction:      "credit",
		Reason:         "deposit",
		Amount:         500.0,
		Symbol:         "USDT",
		TransactionFee: 2.5,
		Info:           nil,
	}

	err = s.DB.Create(transaction).Error
	s.Require().NoError(err)

	// Test cascade relationships
	err = s.DB.Select("Exchanges", "SubAccounts", "Transactions").Where("id = ?", user.ID).Delete(&models.User{}).Error
	s.Require().NoError(err)

	// Verify cascade deletion worked
	var count int64
	s.DB.Model(&models.SecureExchange{}).Where("user_id = ?", user.ID).Count(&count)
	s.Equal(int64(0), count, "Exchanges should be deleted")

	s.DB.Model(&models.SubAccount{}).Where("user_id = ?", user.ID).Count(&count)
	s.Equal(int64(0), count, "SubAccounts should be deleted")

	s.DB.Model(&models.Transaction{}).Where("user_id = ?", user.ID).Count(&count)
	s.Equal(int64(0), count, "Transactions should be deleted")
}

// Test database constraints and validation
func (s *DatabaseIntegrationTestSuite) TestDatabaseConstraints() {
	user := s.createTestUser()

	// Test unique constraints
	duplicateUser := &models.User{
		ID:       uuid.New(),
		Username: user.Username, // Duplicate username
		Email:    "different@example.com",
	}

	err := s.DB.Create(duplicateUser).Error
	s.Error(err, "Should fail due to unique username constraint")

	duplicateUser.Username = "different_username"
	duplicateUser.Email = user.Email // Duplicate email
	err = s.DB.Create(duplicateUser).Error
	s.Error(err, "Should fail due to unique email constraint")

	// Test foreign key constraints
	invalidExchange := &models.SecureExchange{
		ID:               uuid.New(),
		UserID:           uuid.New(), // Non-existent user ID
		Name:             "Invalid Exchange",
		Type:             "binance",
		EncryptedAPIKey:  "encrypted_key",
		EncryptedSecret:  "encrypted_secret",
		APIKeyHash:       "hash_value",
		Status:           "active",
	}

	err = s.DB.Create(invalidExchange).Error
	s.Error(err, "Should fail due to foreign key constraint")

	// Test check constraints
	invalidTransaction := &models.Transaction{
		ID:             uuid.New(),
		UserID:         user.ID,
		ExchangeID:     s.TestExchange.ID,
		SubAccountID:   uuid.New(), // Will be created below
		Timestamp:      time.Now(),
		Direction:      "invalid_direction", // Should fail check constraint
		Reason:         "test",
		Amount:         100.0,
		Symbol:         "USDT",
		TransactionFee: 0.0,
	}

	subAccount := s.createTestSubAccount(s.TestExchange.ID)
	invalidTransaction.SubAccountID = subAccount.ID

	err = s.DB.Create(invalidTransaction).Error
	s.Error(err, "Should fail due to check constraint on direction")
}

// Test complex queries and joins
func (s *DatabaseIntegrationTestSuite) TestComplexQueries() {
	user := s.createTestUser()

	// Create test data
	exchanges := make([]*models.SecureExchange, 3)
	for i := 0; i < 3; i++ {
		exchange, err := s.SecurityService.CreateSecureExchange(
			s.ctx,
			user.ID,
			fmt.Sprintf("Exchange %d", i+1),
			"binance",
			fmt.Sprintf("api_key_%d", i+1),
			fmt.Sprintf("secret_%d", i+1),
		)
		s.Require().NoError(err)
		exchanges[i] = exchange
	}

	subAccounts := make([]*models.SubAccount, 6) // 2 per exchange
	transactions := make([]*models.Transaction, 12) // 2 per sub-account

	for i, exchange := range exchanges {
		for j := 0; j < 2; j++ {
			subIdx := i*2 + j
			subAccounts[subIdx] = s.createTestSubAccount(exchange.ID)

			for k := 0; k < 2; k++ {
				transIdx := subIdx*2 + k
				transactions[transIdx] = s.createTestTransaction(subAccounts[subIdx].ID)
			}
		}
	}

	// Test query: Get all transactions for a user with exchange and sub-account info
	var results []struct {
		Transaction   models.Transaction
		Exchange      models.SecureExchange
		SubAccount    models.SubAccount
		ExchangeName  string
		SubAccountName string
		TransactionCount int64
	}

	query := `
		SELECT 
			t.*,
			e.name as exchange_name,
			sa.name as sub_account_name,
			COUNT(*) OVER (PARTITION BY e.id) as transaction_count
		FROM transactions t
		JOIN exchanges e ON t.exchange_id = e.id
		JOIN sub_accounts sa ON t.sub_account_id = sa.id
		WHERE t.user_id = ?
		ORDER BY t.timestamp DESC
	`

	err := s.DB.Raw(query, user.ID).Scan(&results).Error
	s.Require().NoError(err)
	s.Len(results, 12, "Should have 12 transactions")

	// Test aggregation query: Get balance summary by exchange
	var balanceSummary []struct {
		ExchangeID   uuid.UUID
		ExchangeName string
		TotalBalance float64
		AccountCount int64
	}

	aggregateQuery := `
		SELECT 
			e.id as exchange_id,
			e.name as exchange_name,
			SUM(sa.balance) as total_balance,
			COUNT(sa.id) as account_count
		FROM exchanges e
		LEFT JOIN sub_accounts sa ON e.id = sa.exchange_id
		WHERE e.user_id = ?
		GROUP BY e.id, e.name
		ORDER BY total_balance DESC
	`

	err = s.DB.Raw(aggregateQuery, user.ID).Scan(&balanceSummary).Error
	s.Require().NoError(err)
	s.Len(balanceSummary, 3, "Should have 3 exchanges")

	for _, summary := range balanceSummary {
		s.Equal(int64(2), summary.AccountCount, "Each exchange should have 2 sub-accounts")
		s.Equal(2000.0, summary.TotalBalance, "Each exchange should have total balance of 2000")
	}
}

// Test database transactions and rollbacks
func (s *DatabaseIntegrationTestSuite) TestDatabaseTransactions() {
	user := s.createTestUser()

	// Test successful transaction
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		exchange := &models.SecureExchange{
			ID:               uuid.New(),
			UserID:           user.ID,
			Name:             "Transaction Test Exchange",
			Type:             "binance",
			EncryptedAPIKey:  "encrypted_key",
			EncryptedSecret:  "encrypted_secret",
			APIKeyHash:       "hash_value",
			Status:           "active",
		}

		if err := tx.Create(exchange).Error; err != nil {
			return err
		}

		subAccount := &models.SubAccount{
			ID:         uuid.New(),
			UserID:     user.ID,
			ExchangeID: exchange.ID,
			Name:       "Transaction Test Account",
			Symbol:     "USDT",
			Balance:    1000.0,
		}

		return tx.Create(subAccount).Error
	})

	s.Require().NoError(err)

	// Verify data was committed
	var exchangeCount, subAccountCount int64
	s.DB.Model(&models.SecureExchange{}).Where("user_id = ? AND name = ?", user.ID, "Transaction Test Exchange").Count(&exchangeCount)
	s.DB.Model(&models.SubAccount{}).Where("user_id = ? AND name = ?", user.ID, "Transaction Test Account").Count(&subAccountCount)

	s.Equal(int64(1), exchangeCount, "Exchange should be created")
	s.Equal(int64(1), subAccountCount, "SubAccount should be created")

	// Test failed transaction with rollback
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		exchange := &models.SecureExchange{
			ID:               uuid.New(),
			UserID:           user.ID,
			Name:             "Rollback Test Exchange",
			Type:             "binance",
			EncryptedAPIKey:  "encrypted_key",
			EncryptedSecret:  "encrypted_secret",
			APIKeyHash:       "hash_value",
			Status:           "active",
		}

		if err := tx.Create(exchange).Error; err != nil {
			return err
		}

		// This should fail due to invalid direction
		transaction := &models.Transaction{
			ID:             uuid.New(),
			UserID:         user.ID,
			ExchangeID:     exchange.ID,
			SubAccountID:   uuid.New(), // Invalid sub-account ID
			Timestamp:      time.Now(),
			Direction:      "invalid_direction",
			Reason:         "test",
			Amount:         100.0,
			Symbol:         "USDT",
		}

		return tx.Create(transaction).Error
	})

	s.Error(err, "Transaction should fail")

	// Verify rollback worked
	s.DB.Model(&models.SecureExchange{}).Where("user_id = ? AND name = ?", user.ID, "Rollback Test Exchange").Count(&exchangeCount)
	s.Equal(int64(0), exchangeCount, "Exchange should not exist due to rollback")
}

// Test audit events database operations
func (s *DatabaseIntegrationTestSuite) TestAuditEventsDatabase() {
	// Create various audit events
	events := []security.AuditEvent{
		{
			ID:        uuid.New(),
			Timestamp: time.Now().Add(-2 * time.Hour),
			Level:     security.AuditLevelInfo,
			Action:    security.ActionLogin,
			UserID:    &s.TestUser.ID,
			IPAddress: "192.168.1.100",
			UserAgent: "Mozilla/5.0",
			Resource:  "auth/login",
			Success:   true,
			Details:   map[string]interface{}{"method": "google"},
		},
		{
			ID:        uuid.New(),
			Timestamp: time.Now().Add(-1 * time.Hour),
			Level:     security.AuditLevelWarn,
			Action:    security.ActionLoginFailed,
			IPAddress: "192.168.1.101",
			UserAgent: "curl/7.68.0",
			Resource:  "auth/login",
			Success:   false,
			Details:   map[string]interface{}{"reason": "invalid_credentials"},
		},
		{
			ID:        uuid.New(),
			Timestamp: time.Now().Add(-30 * time.Minute),
			Level:     security.AuditLevelCritical,
			Action:    security.ActionSecurityAlert,
			IPAddress: "192.168.1.102",
			UserAgent: "sqlmap/1.0",
			Resource:  "api/users",
			Success:   false,
			Details:   map[string]interface{}{"alert_type": "sql_injection_attempt"},
		},
	}

	// Insert events
	for _, event := range events {
		err := s.DB.Create(&event).Error
		s.Require().NoError(err)
	}

	// Test querying by different criteria
	var results []security.AuditEvent

	// Query by user
	err := s.DB.Where("user_id = ?", s.TestUser.ID).Find(&results).Error
	s.Require().NoError(err)
	s.Len(results, 1, "Should find 1 event for the user")

	// Query by level
	err = s.DB.Where("level = ?", security.AuditLevelWarn).Find(&results).Error
	s.Require().NoError(err)
	s.Len(results, 1, "Should find 1 warning event")

	// Query by time range
	since := time.Now().Add(-90 * time.Minute)
	err = s.DB.Where("timestamp >= ?", since).Find(&results).Error
	s.Require().NoError(err)
	s.Len(results, 2, "Should find 2 events in the last 90 minutes")

	// Query by success status
	err = s.DB.Where("success = ?", false).Find(&results).Error
	s.Require().NoError(err)
	s.Len(results, 2, "Should find 2 failed events")

	// Test complex query with JSON operations
	err = s.DB.Where("details->>'method' = ?", "google").Find(&results).Error
	s.Require().NoError(err)
	s.Len(results, 1, "Should find 1 event with google method")

	// Test aggregation on audit events
	var summary []struct {
		Level string
		Count int64
	}

	err = s.DB.Model(&security.AuditEvent{}).
		Select("level, COUNT(*) as count").
		Group("level").
		Order("count DESC").
		Scan(&summary).Error
	s.Require().NoError(err)

	s.Len(summary, 3, "Should have 3 different levels")
}

// Test user API keys database operations
func (s *DatabaseIntegrationTestSuite) TestUserAPIKeysDatabase() {
	user := s.createTestUser()

	// Create API keys
	apiKeys := []services.UserAPIKey{
		{
			ID:           uuid.New(),
			UserID:       user.ID,
			Name:         "Production Key",
			EncryptedKey: "encrypted_prod_key",
			KeyHash:      "hash1",
			Permissions:  []string{"read", "write"},
			IsActive:     true,
		},
		{
			ID:           uuid.New(),
			UserID:       user.ID,
			Name:         "Development Key",
			EncryptedKey: "encrypted_dev_key",
			KeyHash:      "hash2",
			Permissions:  []string{"read"},
			IsActive:     true,
		},
		{
			ID:           uuid.New(),
			UserID:       user.ID,
			Name:         "Revoked Key",
			EncryptedKey: "encrypted_revoked_key",
			KeyHash:      "hash3",
			Permissions:  []string{"read", "write", "delete"},
			IsActive:     false,
		},
	}

	for _, apiKey := range apiKeys {
		err := s.DB.Create(&apiKey).Error
		s.Require().NoError(err)
	}

	// Test querying active keys
	var activeKeys []services.UserAPIKey
	err := s.DB.Where("user_id = ? AND is_active = true", user.ID).Find(&activeKeys).Error
	s.Require().NoError(err)
	s.Len(activeKeys, 2, "Should find 2 active keys")

	// Test querying by hash
	var keyByHash services.UserAPIKey
	err = s.DB.Where("key_hash = ?", "hash1").First(&keyByHash).Error
	s.Require().NoError(err)
	s.Equal("Production Key", keyByHash.Name)

	// Test permissions array query
	var keysWithWrite []services.UserAPIKey
	err = s.DB.Where("user_id = ? AND 'write' = ANY(permissions)", user.ID).Find(&keysWithWrite).Error
	s.Require().NoError(err)
	s.Len(keysWithWrite, 2, "Should find 2 keys with write permission")

	// Test updating last used timestamp
	now := time.Now()
	err = s.DB.Model(&keyByHash).Update("last_used_at", now).Error
	s.Require().NoError(err)

	var updatedKey services.UserAPIKey
	err = s.DB.Where("id = ?", keyByHash.ID).First(&updatedKey).Error
	s.Require().NoError(err)
	s.NotNil(updatedKey.LastUsedAt)
	s.WithinDuration(now, *updatedKey.LastUsedAt, time.Second)
}

// Test database indexes performance
func (s *DatabaseIntegrationTestSuite) TestDatabaseIndexes() {
	user := s.createTestUser()

	// Create a large number of audit events to test index performance
	const numEvents = 1000
	events := make([]security.AuditEvent, numEvents)

	for i := 0; i < numEvents; i++ {
		events[i] = security.AuditEvent{
			ID:        uuid.New(),
			Timestamp: time.Now().Add(-time.Duration(i) * time.Minute),
			Level:     security.AuditLevelInfo,
			Action:    security.ActionUserView,
			UserID:    &user.ID,
			IPAddress: fmt.Sprintf("192.168.1.%d", i%255),
			Resource:  fmt.Sprintf("resource_%d", i),
			Success:   i%2 == 0,
			Details:   map[string]interface{}{"index": i},
		}
	}

	// Batch insert for performance
	err := s.DB.CreateInBatches(events, 100).Error
	s.Require().NoError(err)

	// Test queries that should use indexes
	start := time.Now()

	// Query by timestamp (should use idx_audit_events_timestamp)
	var timeRangeResults []security.AuditEvent
	since := time.Now().Add(-30 * time.Minute)
	err = s.DB.Where("timestamp >= ?", since).Limit(10).Find(&timeRangeResults).Error
	s.Require().NoError(err)

	timeRangeQuery := time.Since(start)
	s.Less(timeRangeQuery, 100*time.Millisecond, "Timestamp query should be fast")

	start = time.Now()

	// Query by user and timestamp (should use idx_audit_events_user_timestamp)
	var userTimeResults []security.AuditEvent
	err = s.DB.Where("user_id = ? AND timestamp >= ?", user.ID, since).Find(&userTimeResults).Error
	s.Require().NoError(err)

	userTimeQuery := time.Since(start)
	s.Less(userTimeQuery, 100*time.Millisecond, "User+timestamp query should be fast")

	start = time.Now()

	// Query by action (should use idx_audit_events_action)
	var actionResults []security.AuditEvent
	err = s.DB.Where("action = ?", security.ActionUserView).Limit(10).Find(&actionResults).Error
	s.Require().NoError(err)

	actionQuery := time.Since(start)
	s.Less(actionQuery, 100*time.Millisecond, "Action query should be fast")

	// Verify we got expected results
	s.Len(timeRangeResults, 10, "Should limit to 10 results")
	s.Greater(len(userTimeResults), 0, "Should find user events")
	s.Len(actionResults, 10, "Should find action events")
}

// Run the database integration test suite
func TestDatabaseIntegrationSuite(t *testing.T) {
	suite.Run(t, new(DatabaseIntegrationTestSuite))
}