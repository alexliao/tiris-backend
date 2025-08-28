package helpers

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"tiris-backend/test/config"

	"github.com/golang-migrate/migrate/v4"
	migrationpostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DatabaseTestHelper provides utilities for database testing
type DatabaseTestHelper struct {
	Config *config.TestConfig
	DB     *gorm.DB
	SqlDB  *sql.DB
	tx     *gorm.DB
}

// NewDatabaseTestHelper creates a new database test helper
func NewDatabaseTestHelper(t *testing.T, testConfig *config.TestConfig) *DatabaseTestHelper {
	helper := &DatabaseTestHelper{
		Config: testConfig,
	}

	err := helper.setupDatabase(t)
	require.NoError(t, err, "Failed to setup test database")

	// Setup cleanup
	t.Cleanup(func() {
		helper.Cleanup()
	})

	return helper
}

// setupDatabase initializes the test database connection
func (h *DatabaseTestHelper) setupDatabase(t *testing.T) error {
	// Create database connection
	var err error
	h.SqlDB, err = sql.Open("postgres", h.Config.GetConnectionString())
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	h.SqlDB.SetMaxOpenConns(h.Config.Database.MaxOpenConnections)
	h.SqlDB.SetMaxIdleConns(h.Config.Database.MaxIdleConnections)
	h.SqlDB.SetConnMaxLifetime(h.Config.Database.ConnectionMaxLifetime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	if err := h.SqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Setup GORM
	gormConfig := &gorm.Config{}
	if h.Config.Test.VerboseLogging {
		gormConfig.Logger = logger.Default.LogMode(logger.Info)
	} else {
		gormConfig.Logger = logger.Default.LogMode(logger.Silent)
	}

	h.DB, err = gorm.Open(postgres.New(postgres.Config{
		DSN: h.Config.GetConnectionString(),
	}), gormConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize GORM: %w", err)
	}

	return nil
}

// RunMigrations runs database migrations for tests
func (h *DatabaseTestHelper) RunMigrations(t *testing.T) error {
	driver, err := migrationpostgres.WithInstance(h.SqlDB, &migrationpostgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://../../migrations",
		"postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// BeginTransaction starts a database transaction for test isolation
func (h *DatabaseTestHelper) BeginTransaction() *gorm.DB {
	h.tx = h.DB.Begin()
	return h.tx
}

// RollbackTransaction rolls back the current transaction
func (h *DatabaseTestHelper) RollbackTransaction() {
	if h.tx != nil {
		h.tx.Rollback()
		h.tx = nil
	}
}

// CommitTransaction commits the current transaction
func (h *DatabaseTestHelper) CommitTransaction() error {
	if h.tx != nil {
		err := h.tx.Commit().Error
		h.tx = nil
		return err
	}
	return nil
}

// GetTransactionDB returns the transaction database connection
func (h *DatabaseTestHelper) GetTransactionDB() *gorm.DB {
	if h.tx != nil {
		return h.tx
	}
	return h.DB
}

// TruncateAllTables truncates all tables for cleanup
func (h *DatabaseTestHelper) TruncateAllTables() error {
	tables := []string{
		"trading_logs",
		"transactions", 
		"sub_accounts",
		"tradings",
		"oauth_tokens",
		"users",
	}

	for _, table := range tables {
		if err := h.DB.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)).Error; err != nil {
			return fmt.Errorf("failed to truncate table %s: %w", table, err)
		}
	}

	return nil
}

// Cleanup closes database connections and performs cleanup
func (h *DatabaseTestHelper) Cleanup() {
	if h.Config.Test.DatabaseCleanup {
		h.TruncateAllTables()
	}

	if h.tx != nil {
		h.RollbackTransaction()
	}

	if h.SqlDB != nil {
		h.SqlDB.Close()
	}
}

// AssertTableExists verifies that a table exists in the database
func (h *DatabaseTestHelper) AssertTableExists(t *testing.T, tableName string) {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = $1
		)
	`
	err := h.SqlDB.QueryRow(query, tableName).Scan(&exists)
	require.NoError(t, err, "Failed to check if table exists")
	require.True(t, exists, "Table %s should exist", tableName)
}

// AssertRecordCount verifies the number of records in a table
func (h *DatabaseTestHelper) AssertRecordCount(t *testing.T, tableName string, expectedCount int64) {
	var count int64
	err := h.GetTransactionDB().Table(tableName).Count(&count).Error
	require.NoError(t, err, "Failed to count records in table %s", tableName)
	require.Equal(t, expectedCount, count, "Expected %d records in table %s, got %d", expectedCount, tableName, count)
}

// WaitForDatabase waits for database to be ready
func (h *DatabaseTestHelper) WaitForDatabase(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for database to be ready")
		case <-ticker.C:
			if err := h.SqlDB.PingContext(ctx); err == nil {
				return nil
			}
		}
	}
}

// ExecuteInTransaction executes a function within a database transaction
func (h *DatabaseTestHelper) ExecuteInTransaction(fn func(tx *gorm.DB) error) error {
	return h.DB.Transaction(fn)
}

// SeedTestData inserts test data using provided fixtures
func (h *DatabaseTestHelper) SeedTestData(fixtures interface{}) error {
	return h.GetTransactionDB().Create(fixtures).Error
}

// GetDatabaseStats returns database connection statistics
func (h *DatabaseTestHelper) GetDatabaseStats() sql.DBStats {
	return h.SqlDB.Stats()
}

// CreateDatabase creates a new test database (useful for parallel tests)
func CreateTestDatabase(config *config.TestConfig, dbName string) error {
	// Connect to postgres database to create new test database
	adminConfig := *config
	adminConfig.Database.DBName = "postgres"
	
	adminDB, err := sql.Open("postgres", adminConfig.GetConnectionString())
	if err != nil {
		return fmt.Errorf("failed to connect to admin database: %w", err)
	}
	defer adminDB.Close()

	// Create database
	_, err = adminDB.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
	if err != nil {
		// Database might already exist, check if it's a "already exists" error
		if !isAlreadyExistsError(err) {
			return fmt.Errorf("failed to create test database %s: %w", dbName, err)
		}
	}

	return nil
}

// DropTestDatabase drops a test database
func DropTestDatabase(config *config.TestConfig, dbName string) error {
	// Connect to postgres database to drop test database
	adminConfig := *config
	adminConfig.Database.DBName = "postgres"
	
	adminDB, err := sql.Open("postgres", adminConfig.GetConnectionString())
	if err != nil {
		return fmt.Errorf("failed to connect to admin database: %w", err)
	}
	defer adminDB.Close()

	// Terminate existing connections to the database
	_, err = adminDB.Exec(fmt.Sprintf(`
		SELECT pg_terminate_backend(pid) 
		FROM pg_stat_activity 
		WHERE datname = '%s' AND pid <> pg_backend_pid()
	`, dbName))
	if err != nil {
		// Not critical if this fails
	}

	// Drop database
	_, err = adminDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
	if err != nil {
		return fmt.Errorf("failed to drop test database %s: %w", dbName, err)
	}

	return nil
}

// isAlreadyExistsError checks if the error is a "database already exists" error
func isAlreadyExistsError(err error) bool {
	// PostgreSQL error code for "database already exists" is 42P04
	return err != nil && (
		err.Error() == "pq: database \""+err.Error()+"\" already exists" ||
		// Add other database-specific error checks as needed
		false)
}

// DatabaseTestSuite provides a test suite with database setup/teardown
type DatabaseTestSuite struct {
	Config   *config.TestConfig
	Helper   *DatabaseTestHelper
	T        *testing.T
}

// NewDatabaseTestSuite creates a new database test suite
func NewDatabaseTestSuite(t *testing.T, profile config.TestProfile) *DatabaseTestSuite {
	testConfig := config.GetProfileConfig(profile)
	helper := NewDatabaseTestHelper(t, testConfig)

	return &DatabaseTestSuite{
		Config: testConfig,
		Helper: helper,
		T:      t,
	}
}

// SetupSuite runs before all tests in the suite
func (suite *DatabaseTestSuite) SetupSuite() {
	err := suite.Helper.RunMigrations(suite.T)
	require.NoError(suite.T, err, "Failed to run migrations")
}

// SetupTest runs before each test
func (suite *DatabaseTestSuite) SetupTest() {
	suite.Helper.BeginTransaction()
}

// TearDownTest runs after each test
func (suite *DatabaseTestSuite) TearDownTest() {
	suite.Helper.RollbackTransaction()
}

// TearDownSuite runs after all tests in the suite
func (suite *DatabaseTestSuite) TearDownSuite() {
	suite.Helper.Cleanup()
}