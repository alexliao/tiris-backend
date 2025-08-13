package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"tiris-backend/internal/config"
	"tiris-backend/internal/database"
	"tiris-backend/internal/models"
	"tiris-backend/internal/services"
	"tiris-backend/pkg/auth"
	"tiris-backend/pkg/security"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// IntegrationTestSuite provides setup for integration tests
type IntegrationTestSuite struct {
	suite.Suite
	DB              *gorm.DB
	Redis           *redis.Client
	Config          *config.Config
	UserService     *services.UserService
	ExchangeService *services.ExchangeService
	SecurityService *services.SecurityService
	JWTManager      *auth.JWTManager
	TestUser        *models.User
	TestExchange    *models.SecureExchange
	ctx             context.Context
	cancel          context.CancelFunc
}

// SetupSuite runs once before all tests in the suite
func (s *IntegrationTestSuite) SetupSuite() {
	// Set test environment
	os.Setenv("ENV", "test")
	
	// Create test context
	s.ctx, s.cancel = context.WithTimeout(context.Background(), 30*time.Second)

	// Load configuration
	cfg, err := config.Load()
	s.Require().NoError(err, "Failed to load configuration")
	
	// Override with test database settings
	cfg.Database = &config.DatabaseConfig{
		Host:                  "localhost",
		Port:                  5432,
		User:                  "tiris_test",
		Password:              "tiris_test",
		DBName:                fmt.Sprintf("tiris_integration_test_%d", time.Now().UnixNano()),
		SSLMode:               "disable",
		MaxOpenConnections:    10,
		MaxIdleConnections:    5,
		ConnectionMaxLifetime: 300,
	}

	cfg.Redis = &config.RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		DB:       1, // Use different DB for tests
	}

	cfg.Auth = &config.AuthConfig{
		JWTSecret:        "integration-test-jwt-secret-key-32-chars",
		RefreshSecret:    "integration-test-refresh-secret-32-chars",
		JWTExpiration:    3600,
		RefreshExpiration: 86400,
	}

	cfg.Security = &config.SecurityConfig{
		MasterKey:  "integration-test-master-key-32-chars-minimum",
		SigningKey: "integration-test-signing-key-32-chars-minimum",
	}

	s.Config = cfg

	// Initialize database
	db, err := database.Initialize(cfg.Database)
	s.Require().NoError(err, "Failed to initialize test database")
	s.DB = db.DB

	// Initialize Redis
	s.Redis = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Test Redis connection
	_, err = s.Redis.Ping(s.ctx).Result()
	s.Require().NoError(err, "Failed to connect to Redis")

	// Run migrations
	err = s.DB.AutoMigrate(
		&models.User{},
		&models.OAuthToken{},
		&models.SecureExchange{},
		&models.SubAccount{},
		&models.Transaction{},
		&models.TradingLog{},
		&security.AuditEvent{},
		&services.UserAPIKey{},
	)
	s.Require().NoError(err, "Failed to run database migrations")

	// Initialize services
	s.JWTManager = auth.NewJWTManager(cfg.Auth.JWTSecret, cfg.Auth.RefreshSecret)

	s.UserService, err = services.NewUserService(s.DB)
	s.Require().NoError(err, "Failed to create user service")

	s.ExchangeService, err = services.NewExchangeService(s.DB)
	s.Require().NoError(err, "Failed to create exchange service")

	s.SecurityService, err = services.NewSecurityService(
		s.DB,
		s.Redis,
		cfg.Security.MasterKey,
		cfg.Security.SigningKey,
	)
	s.Require().NoError(err, "Failed to create security service")
}

// TearDownSuite runs once after all tests in the suite
func (s *IntegrationTestSuite) TearDownSuite() {
	// Cancel context
	if s.cancel != nil {
		s.cancel()
	}

	// Clean up Redis
	if s.Redis != nil {
		s.Redis.FlushDB(s.ctx)
		s.Redis.Close()
	}

	// Clean up database
	if s.DB != nil {
		// Drop the test database
		sqlDB, err := s.DB.DB()
		if err == nil {
			sqlDB.Close()
		}
	}
}

// SetupTest runs before each test
func (s *IntegrationTestSuite) SetupTest() {
	// Clean all tables
	s.cleanTables()

	// Create test user
	s.TestUser = s.createTestUser()

	// Create test secure exchange
	s.TestExchange = s.createTestSecureExchange()
}

// TearDownTest runs after each test
func (s *IntegrationTestSuite) TearDownTest() {
	// Clean up any test-specific data
	s.cleanTables()
	s.Redis.FlushDB(s.ctx)
}

// Helper methods

func (s *IntegrationTestSuite) cleanTables() {
	// Order matters due to foreign key constraints
	tables := []string{
		"audit_events",
		"user_api_keys",
		"trading_logs",
		"transactions",
		"sub_accounts",
		"exchanges",
		"oauth_tokens",
		"users",
	}

	for _, table := range tables {
		err := s.DB.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)).Error
		s.Require().NoError(err, "Failed to clean table %s", table)
	}
}

func (s *IntegrationTestSuite) createTestUser() *models.User {
	user := &models.User{
		ID:       uuid.New(),
		Username: "integration_test_user",
		Email:    fmt.Sprintf("test_%d@example.com", time.Now().UnixNano()),
		Avatar:   nil,
		Settings: nil,
		Info:     nil,
	}

	err := s.DB.Create(user).Error
	s.Require().NoError(err, "Failed to create test user")

	return user
}

func (s *IntegrationTestSuite) createTestSecureExchange() *models.SecureExchange {
	exchange, err := s.SecurityService.CreateSecureExchange(
		s.ctx,
		s.TestUser.ID,
		"Test Exchange",
		"binance",
		"test_api_key_12345",
		"test_api_secret_67890",
	)
	s.Require().NoError(err, "Failed to create test secure exchange")

	return exchange
}

func (s *IntegrationTestSuite) createTestSubAccount(exchangeID uuid.UUID) *models.SubAccount {
	subAccount := &models.SubAccount{
		ID:         uuid.New(),
		UserID:     s.TestUser.ID,
		ExchangeID: exchangeID,
		Name:       "Test Sub Account",
		Symbol:     "USDT",
		Balance:    1000.0,
		Info:       nil,
	}

	err := s.DB.Create(subAccount).Error
	s.Require().NoError(err, "Failed to create test sub account")

	return subAccount
}

func (s *IntegrationTestSuite) createTestTransaction(subAccountID uuid.UUID) *models.Transaction {
	transaction := &models.Transaction{
		ID:             uuid.New(),
		UserID:         s.TestUser.ID,
		ExchangeID:     s.TestExchange.ID,
		SubAccountID:   subAccountID,
		Timestamp:      time.Now(),
		Direction:      "credit",
		Reason:         "test_deposit",
		Amount:         100.0,
		Symbol:         "USDT",
		TransactionFee: 0.1,
		Info:           nil,
	}

	err := s.DB.Create(transaction).Error
	s.Require().NoError(err, "Failed to create test transaction")

	return transaction
}

func (s *IntegrationTestSuite) createUserAPIKey(userID uuid.UUID, name string) *services.UserAPIKey {
	apiKey, err := s.SecurityService.CreateUserAPIKey(
		s.ctx,
		userID,
		name,
		[]string{"read", "write"},
	)
	s.Require().NoError(err, "Failed to create user API key")

	return apiKey
}

func (s *IntegrationTestSuite) generateJWTToken(userID uuid.UUID, username, email string) (string, error) {
	claims := &auth.Claims{
		UserID:   userID,
		Username: username,
		Email:    email,
		Role:     "user",
	}

	return s.JWTManager.GenerateToken(claims)
}

// Helper for testing rate limits
func (s *IntegrationTestSuite) hitRateLimit(identifier, ruleName string, count int) {
	rules := security.DefaultRules()
	rule, exists := rules[ruleName]
	s.Require().True(exists, "Rule %s should exist", ruleName)

	// Hit the rate limit multiple times
	for i := 0; i < count; i++ {
		_, err := s.SecurityService.CheckRateLimit(s.ctx, identifier, ruleName)
		s.Require().NoError(err, "Failed to check rate limit")
	}
}

// Helper for testing audit events
func (s *IntegrationTestSuite) waitForAuditEvent(action security.AuditAction, timeout time.Duration) *security.AuditEvent {
	ctx, cancel := context.WithTimeout(s.ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.Fail("Timeout waiting for audit event", "Action: %s", action)
			return nil
		case <-ticker.C:
			var event security.AuditEvent
			err := s.DB.Where("action = ?", action).Order("timestamp DESC").First(&event).Error
			if err == nil {
				return &event
			}
		}
	}
}

// TestMain sets up the test environment
func TestMain(m *testing.M) {
	// Ensure test database exists
	// This would typically be handled by your CI/CD pipeline
	
	code := m.Run()
	os.Exit(code)
}