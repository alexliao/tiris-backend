package utils

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"tiris-backend/internal/config"
	"tiris-backend/internal/database"
	"tiris-backend/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestDB provides a test database instance
type TestDB struct {
	DB       *gorm.DB
	Config   *config.DatabaseConfig
	instance *database.Database
}

// NewTestDB creates a new test database connection
func NewTestDB(t *testing.T) *TestDB {
	cfg := &config.DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "tiris_test",
		Password: "tiris_test",
		DBName:   fmt.Sprintf("tiris_test_%d", time.Now().UnixNano()),
		SSLMode:  "disable",
	}

	db, err := database.Initialize(cfg)
	require.NoError(t, err, "Failed to initialize test database")

	return &TestDB{
		DB:       db.DB,
		Config:   cfg,
		instance: db,
	}
}

// Close closes the test database connection
func (tdb *TestDB) Close() {
	if tdb.instance != nil {
		database.Close(tdb.instance)
	}
}

// CleanDB cleans all tables in the test database
func (tdb *TestDB) CleanDB(t *testing.T) {
	// List of tables to clean in dependency order
	tables := []string{
		"trading_logs",
		"transactions",
		"sub_accounts",
		"exchanges",
		"oauth_tokens",
		"users",
		"event_processing_logs",
	}

	for _, table := range tables {
		err := tdb.DB.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)).Error
		require.NoError(t, err, "Failed to clean table %s", table)
	}
}

// CreateTestUser creates a test user in the database
func (tdb *TestDB) CreateTestUser(t *testing.T, email string) *models.User {
	user := &models.User{
		ID:          uuid.New(),
		Email:       email,
		DisplayName: sql.NullString{String: "Test User", Valid: true},
		AvatarURL:   sql.NullString{String: "https://example.com/avatar.png", Valid: true},
		IsAdmin:     false,
		IsActive:    true,
		LastLoginAt: sql.NullTime{Time: time.Now(), Valid: true},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := tdb.DB.Create(user).Error
	require.NoError(t, err, "Failed to create test user")

	return user
}

// CreateTestExchange creates a test exchange in the database
func (tdb *TestDB) CreateTestExchange(t *testing.T, userID uuid.UUID, exchangeName string) *models.Exchange {
	exchange := &models.Exchange{
		ID:          uuid.New(),
		UserID:      userID,
		Name:        exchangeName,
		DisplayName: fmt.Sprintf("Test %s", exchangeName),
		APIKey:      "test_api_key",
		SecretKey:   "test_secret_key",
		Passphrase:  sql.NullString{String: "test_passphrase", Valid: true},
		Sandbox:     true,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := tdb.DB.Create(exchange).Error
	require.NoError(t, err, "Failed to create test exchange")

	return exchange
}

// CreateTestSubAccount creates a test sub-account in the database
func (tdb *TestDB) CreateTestSubAccount(t *testing.T, exchangeID uuid.UUID, name string) *models.SubAccount {
	subAccount := &models.SubAccount{
		ID:         uuid.New(),
		ExchangeID: exchangeID,
		Name:       name,
		Balance: map[string]interface{}{
			"BTC":  1.5,
			"ETH":  10.0,
			"USDT": 1000.0,
		},
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := tdb.DB.Create(subAccount).Error
	require.NoError(t, err, "Failed to create test sub-account")

	return subAccount
}

// CreateTestTransaction creates a test transaction in the database
func (tdb *TestDB) CreateTestTransaction(t *testing.T, subAccountID uuid.UUID, direction, reason string, amount float64) *models.Transaction {
	transaction := &models.Transaction{
		ID:           uuid.New(),
		SubAccountID: subAccountID,
		Direction:    direction,
		Amount:       amount,
		Symbol:       "USDT",
		Reason:       reason,
		Metadata: map[string]interface{}{
			"source": "test",
		},
		Timestamp: time.Now(),
		CreatedAt: time.Now(),
	}

	err := tdb.DB.Create(transaction).Error
	require.NoError(t, err, "Failed to create test transaction")

	return transaction
}

// CreateTestTradingLog creates a test trading log in the database
func (tdb *TestDB) CreateTestTradingLog(t *testing.T, exchangeID, subAccountID uuid.UUID, logType, source string) *models.TradingLog {
	tradingLog := &models.TradingLog{
		ID:           uuid.New(),
		ExchangeID:   exchangeID,
		SubAccountID: sql.NullString{String: subAccountID.String(), Valid: true},
		Type:         logType,
		Source:       source,
		Content:      "Test trading log content",
		Metadata: map[string]interface{}{
			"test": true,
		},
		Timestamp: time.Now(),
		CreatedAt: time.Now(),
	}

	err := tdb.DB.Create(tradingLog).Error
	require.NoError(t, err, "Failed to create test trading log")

	return tradingLog
}

// AssertErrorContains checks that an error contains a specific message
func AssertErrorContains(t *testing.T, err error, expectedMessage string) {
	require.Error(t, err)
	require.Contains(t, err.Error(), expectedMessage)
}

// GetTestContext returns a test context with timeout
func GetTestContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}

// ValidateUUID checks if a string is a valid UUID
func ValidateUUID(t *testing.T, uuidStr string) {
	_, err := uuid.Parse(uuidStr)
	require.NoError(t, err, "Invalid UUID format: %s", uuidStr)
}

// CompareTime compares two times with a tolerance of 1 second
func CompareTime(t *testing.T, expected, actual time.Time, message string) {
	diff := expected.Sub(actual)
	if diff < 0 {
		diff = -diff
	}
	require.True(t, diff < time.Second, "%s: time difference too large: %v", message, diff)
}
