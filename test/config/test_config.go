package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// TestConfig holds all configuration for integration tests
type TestConfig struct {
	Database DatabaseTestConfig `json:"database"`
	Redis    RedisTestConfig    `json:"redis"`
	NATS     NATSTestConfig     `json:"nats"`
	Auth     AuthTestConfig     `json:"auth"`
	Security SecurityTestConfig `json:"security"`
	Test     TestSettings       `json:"test"`
}

// DatabaseTestConfig holds database configuration for tests
type DatabaseTestConfig struct {
	Host                  string        `json:"host"`
	Port                  int           `json:"port"`
	User                  string        `json:"user"`
	Password              string        `json:"password"`
	DBName                string        `json:"db_name"`
	SSLMode               string        `json:"ssl_mode"`
	MaxOpenConnections    int           `json:"max_open_connections"`
	MaxIdleConnections    int           `json:"max_idle_connections"`
	ConnectionMaxLifetime time.Duration `json:"connection_max_lifetime"`
}

// RedisTestConfig holds Redis configuration for tests
type RedisTestConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

// NATSTestConfig holds NATS configuration for tests
type NATSTestConfig struct {
	URL              string        `json:"url"`
	ClusterID        string        `json:"cluster_id"`
	ClientID         string        `json:"client_id"`
	DurableName      string        `json:"durable_name"`
	ConnectTimeout   time.Duration `json:"connect_timeout"`
	ReconnectWait    time.Duration `json:"reconnect_wait"`
	MaxReconnect     int           `json:"max_reconnect"`
	StreamName       string        `json:"stream_name"`
	SubjectPrefix    string        `json:"subject_prefix"`
}

// AuthTestConfig holds authentication configuration for tests
type AuthTestConfig struct {
	JWTSecret         string        `json:"jwt_secret"`
	RefreshSecret     string        `json:"refresh_secret"`
	JWTExpiration     time.Duration `json:"jwt_expiration"`
	RefreshExpiration time.Duration `json:"refresh_expiration"`
}

// SecurityTestConfig holds security configuration for tests
type SecurityTestConfig struct {
	MasterKey  string `json:"master_key"`
	SigningKey string `json:"signing_key"`
}

// TestSettings holds test-specific configuration
type TestSettings struct {
	Timeout             time.Duration `json:"timeout"`
	DatabaseCleanup     bool          `json:"database_cleanup"`
	RedisCleanup        bool          `json:"redis_cleanup"`
	VerboseLogging      bool          `json:"verbose_logging"`
	SkipPerformance     bool          `json:"skip_performance"`
	ConcurrencyLevel    int           `json:"concurrency_level"`
	PerformanceTimeout  time.Duration `json:"performance_timeout"`
	MaxMemoryUsage      int64         `json:"max_memory_usage"` // in MB
	MaxResponseTime     time.Duration `json:"max_response_time"`
	MinThroughput       float64       `json:"min_throughput"` // ops per second
}

// LoadTestConfig loads test configuration from environment variables with defaults
func LoadTestConfig() *TestConfig {
	return &TestConfig{
		Database: DatabaseTestConfig{
			Host:                  getEnvString("TEST_DB_HOST", "localhost"),
			Port:                  getEnvInt("TEST_DB_PORT", 5432),
			User:                  getEnvString("TEST_DB_USER", "tiris_test"),
			Password:              getEnvString("TEST_DB_PASSWORD", "tiris_test"),
			DBName:                getEnvString("TEST_DB_NAME", fmt.Sprintf("tiris_test_%d", time.Now().UnixNano())),
			SSLMode:               getEnvString("TEST_DB_SSL_MODE", "disable"),
			MaxOpenConnections:    getEnvInt("TEST_DB_MAX_OPEN_CONNECTIONS", 10),
			MaxIdleConnections:    getEnvInt("TEST_DB_MAX_IDLE_CONNECTIONS", 5),
			ConnectionMaxLifetime: getEnvDuration("TEST_DB_CONNECTION_MAX_LIFETIME", 5*time.Minute),
		},
		Redis: RedisTestConfig{
			Host:     getEnvString("TEST_REDIS_HOST", "localhost"),
			Port:     getEnvInt("TEST_REDIS_PORT", 6380),
			Password: getEnvString("TEST_REDIS_PASSWORD", ""),
			DB:       getEnvInt("TEST_REDIS_DB", 1),
		},
		NATS: NATSTestConfig{
			URL:              getEnvString("TEST_NATS_URL", "nats://localhost:4223"),
			ClusterID:        getEnvString("TEST_NATS_CLUSTER_ID", "tiris-test-cluster"),
			ClientID:         getEnvString("TEST_NATS_CLIENT_ID", "tiris-test-client"),
			DurableName:      getEnvString("TEST_NATS_DURABLE", "tiris-test-durable"),
			ConnectTimeout:   getEnvDuration("TEST_NATS_CONNECT_TIMEOUT", 10*time.Second),
			ReconnectWait:    getEnvDuration("TEST_NATS_RECONNECT_WAIT", 2*time.Second),
			MaxReconnect:     getEnvInt("TEST_NATS_MAX_RECONNECT", 5),
			StreamName:       getEnvString("TEST_NATS_STREAM", "TIRIS_TEST"),
			SubjectPrefix:    getEnvString("TEST_NATS_SUBJECT_PREFIX", "tiris.test"),
		},
		Auth: AuthTestConfig{
			JWTSecret:         getEnvString("TEST_JWT_SECRET", "integration-test-jwt-secret-key-32-chars"),
			RefreshSecret:     getEnvString("TEST_REFRESH_SECRET", "integration-test-refresh-secret-32-chars"),
			JWTExpiration:     getEnvDuration("TEST_JWT_EXPIRATION", 1*time.Hour),
			RefreshExpiration: getEnvDuration("TEST_REFRESH_EXPIRATION", 24*time.Hour),
		},
		Security: SecurityTestConfig{
			MasterKey:  getEnvString("TEST_MASTER_KEY", "integration-test-master-key-32-chars-minimum"),
			SigningKey: getEnvString("TEST_SIGNING_KEY", "integration-test-signing-key-32-chars-minimum"),
		},
		Test: TestSettings{
			Timeout:             getEnvDuration("TEST_TIMEOUT", 30*time.Minute),
			DatabaseCleanup:     getEnvBool("TEST_DATABASE_CLEANUP", true),
			RedisCleanup:        getEnvBool("TEST_REDIS_CLEANUP", true),
			VerboseLogging:      getEnvBool("TEST_VERBOSE", false),
			SkipPerformance:     getEnvBool("TEST_SKIP_PERFORMANCE", false),
			ConcurrencyLevel:    getEnvInt("TEST_CONCURRENCY_LEVEL", 10),
			PerformanceTimeout:  getEnvDuration("TEST_PERFORMANCE_TIMEOUT", 10*time.Minute),
			MaxMemoryUsage:      getEnvInt64("TEST_MAX_MEMORY_MB", 1024), // 1GB
			MaxResponseTime:     getEnvDuration("TEST_MAX_RESPONSE_TIME", 100*time.Millisecond),
			MinThroughput:       getEnvFloat64("TEST_MIN_THROUGHPUT", 100.0), // 100 ops/sec
		},
	}
}

// GetConnectionString returns the database connection string for tests
func (c *TestConfig) GetConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.DBName,
		c.Database.SSLMode,
	)
}

// GetRedisAddress returns the Redis address for tests
func (c *TestConfig) GetRedisAddress() string {
	return fmt.Sprintf("%s:%d", c.Redis.Host, c.Redis.Port)
}

// GetNATSURL returns the NATS URL for tests
func (c *TestConfig) GetNATSURL() string {
	return c.NATS.URL
}

// IsPerformanceTestEnabled returns true if performance tests should run
func (c *TestConfig) IsPerformanceTestEnabled() bool {
	return !c.Test.SkipPerformance
}

// ShouldCleanup returns true if test cleanup should be performed
func (c *TestConfig) ShouldCleanup() bool {
	return c.Test.DatabaseCleanup && c.Test.RedisCleanup
}

// Validate checks if the test configuration is valid
func (c *TestConfig) Validate() error {
	// Validate database config
	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if c.Database.Port <= 0 || c.Database.Port > 65535 {
		return fmt.Errorf("invalid database port: %d", c.Database.Port)
	}
	if c.Database.User == "" {
		return fmt.Errorf("database user is required")
	}
	if c.Database.DBName == "" {
		return fmt.Errorf("database name is required")
	}

	// Validate Redis config
	if c.Redis.Host == "" {
		return fmt.Errorf("redis host is required")
	}
	if c.Redis.Port <= 0 || c.Redis.Port > 65535 {
		return fmt.Errorf("invalid redis port: %d", c.Redis.Port)
	}

	// Validate auth config
	if len(c.Auth.JWTSecret) < 32 {
		return fmt.Errorf("JWT secret must be at least 32 characters")
	}
	if len(c.Auth.RefreshSecret) < 32 {
		return fmt.Errorf("refresh secret must be at least 32 characters")
	}

	// Validate security config
	if len(c.Security.MasterKey) < 32 {
		return fmt.Errorf("master key must be at least 32 characters")
	}
	if len(c.Security.SigningKey) < 32 {
		return fmt.Errorf("signing key must be at least 32 characters")
	}

	// Validate test settings
	if c.Test.Timeout <= 0 {
		return fmt.Errorf("test timeout must be positive")
	}
	if c.Test.ConcurrencyLevel <= 0 {
		return fmt.Errorf("concurrency level must be positive")
	}
	if c.Test.MaxMemoryUsage <= 0 {
		return fmt.Errorf("max memory usage must be positive")
	}

	return nil
}

// Helper functions to read environment variables with defaults

func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvFloat64(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// TestProfile represents different test execution profiles
type TestProfile string

const (
	ProfileQuick        TestProfile = "quick"        // Fast tests only, minimal setup
	ProfileStandard     TestProfile = "standard"     // Standard test suite
	ProfileComprehensive TestProfile = "comprehensive" // Full test suite including performance
	ProfilePerformance  TestProfile = "performance"  // Performance tests only
	ProfileSecurity     TestProfile = "security"     // Security-focused tests
)

// GetProfileConfig returns test configuration for a specific profile
func GetProfileConfig(profile TestProfile) *TestConfig {
	config := LoadTestConfig()

	switch profile {
	case ProfileQuick:
		config.Test.Timeout = 5 * time.Minute
		config.Test.SkipPerformance = true
		config.Test.ConcurrencyLevel = 5
		config.Test.VerboseLogging = false
		
	case ProfileStandard:
		// Use default configuration
		
	case ProfileComprehensive:
		config.Test.Timeout = 60 * time.Minute
		config.Test.SkipPerformance = false
		config.Test.ConcurrencyLevel = 20
		config.Test.PerformanceTimeout = 15 * time.Minute
		config.Test.VerboseLogging = true
		
	case ProfilePerformance:
		config.Test.Timeout = 30 * time.Minute
		config.Test.SkipPerformance = false
		config.Test.ConcurrencyLevel = 50
		config.Test.PerformanceTimeout = 20 * time.Minute
		config.Test.MaxResponseTime = 50 * time.Millisecond
		config.Test.MinThroughput = 500.0
		
	case ProfileSecurity:
		config.Test.Timeout = 20 * time.Minute
		config.Test.ConcurrencyLevel = 10
		config.Test.VerboseLogging = true
		// Focus on security-related limits
		config.Test.MaxResponseTime = 200 * time.Millisecond
	}

	return config
}

// TestMetrics holds metrics collected during test execution
type TestMetrics struct {
	TestsRun        int           `json:"tests_run"`
	TestsPassed     int           `json:"tests_passed"`
	TestsFailed     int           `json:"tests_failed"`
	TestsSkipped    int           `json:"tests_skipped"`
	TotalDuration   time.Duration `json:"total_duration"`
	AverageResponse time.Duration `json:"average_response"`
	MaxResponse     time.Duration `json:"max_response"`
	MinResponse     time.Duration `json:"min_response"`
	Throughput      float64       `json:"throughput"` // operations per second
	ErrorRate       float64       `json:"error_rate"` // percentage
	MemoryUsage     int64         `json:"memory_usage"` // in MB
	DatabaseOps     int           `json:"database_ops"`
	RedisOps        int           `json:"redis_ops"`
	APIRequests     int           `json:"api_requests"`
	SecurityEvents  int           `json:"security_events"`
}

// CalculateMetrics calculates test metrics from collected data
func (tm *TestMetrics) CalculateMetrics() {
	if tm.TestsRun > 0 {
		tm.ErrorRate = float64(tm.TestsFailed) / float64(tm.TestsRun) * 100
	}
	
	if tm.TotalDuration > 0 {
		tm.Throughput = float64(tm.TestsRun) / tm.TotalDuration.Seconds()
	}
}

// IsWithinSLA checks if metrics are within acceptable service level agreement
func (tm *TestMetrics) IsWithinSLA(config *TestConfig) bool {
	if tm.ErrorRate > 5.0 { // Max 5% error rate
		return false
	}
	
	if tm.AverageResponse > config.Test.MaxResponseTime {
		return false
	}
	
	if tm.Throughput < config.Test.MinThroughput {
		return false
	}
	
	if tm.MemoryUsage > config.Test.MaxMemoryUsage {
		return false
	}
	
	return true
}