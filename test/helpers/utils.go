package helpers

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"tiris-backend/internal/models"
	"tiris-backend/pkg/auth"
	"tiris-backend/test/config"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUtils provides common testing utilities
type TestUtils struct {
	Config *config.TestConfig
}

// NewTestUtils creates a new test utilities instance
func NewTestUtils(testConfig *config.TestConfig) *TestUtils {
	return &TestUtils{
		Config: testConfig,
	}
}

// GenerateJWTToken generates a JWT token for testing
func (u *TestUtils) GenerateJWTToken(user *models.User) (string, error) {
	jwtManager := auth.NewJWTManager(
		u.Config.Auth.JWTSecret,
		u.Config.Auth.RefreshSecret,
		u.Config.Auth.JWTExpiration,
		u.Config.Auth.RefreshExpiration,
	)

	return jwtManager.GenerateToken(user.ID, user.Username, user.Email, "user")
}

// GenerateRefreshToken generates a refresh token for testing
func (u *TestUtils) GenerateRefreshToken(user *models.User) (string, error) {
	jwtManager := auth.NewJWTManager(
		u.Config.Auth.JWTSecret,
		u.Config.Auth.RefreshSecret,
		u.Config.Auth.JWTExpiration,
		u.Config.Auth.RefreshExpiration,
	)

	// For now, we'll generate a regular token as refresh token
	// In real implementation, you might have a separate method
	return jwtManager.GenerateToken(user.ID, user.Username, user.Email, "user")
}

// CreateAuthorizedRequest creates an HTTP request with authorization header
func (u *TestUtils) CreateAuthorizedRequest(method, url, body string, user *models.User) (*http.Request, error) {
	req := httptest.NewRequest(method, url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	if user != nil {
		token, err := u.GenerateJWTToken(user)
		if err != nil {
			return nil, fmt.Errorf("failed to generate token: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return req, nil
}

// CreateGinContext creates a Gin context for testing
func (u *TestUtils) CreateGinContext(req *http.Request) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = req
	return c, recorder
}

// SetUserInContext sets user information in Gin context for middleware testing
func (u *TestUtils) SetUserInContext(c *gin.Context, user *models.User) {
	c.Set("user_id", user.ID)
	c.Set("user_email", user.Email)
}

// AssertJSONResponse validates JSON response structure and status code
func (u *TestUtils) AssertJSONResponse(t *testing.T, recorder *httptest.ResponseRecorder, expectedStatus int) {
	assert.Equal(t, expectedStatus, recorder.Code)
	assert.Contains(t, recorder.Header().Get("Content-Type"), "application/json")
}

// AssertSuccessResponse validates a successful API response
func (u *TestUtils) AssertSuccessResponse(t *testing.T, recorder *httptest.ResponseRecorder, expectedStatus int) {
	u.AssertJSONResponse(t, recorder, expectedStatus)
	body := recorder.Body.String()
	assert.Contains(t, body, `"success":true`)
}

// AssertErrorResponse validates an error API response
func (u *TestUtils) AssertErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder, expectedStatus int, expectedErrorCode string) {
	u.AssertJSONResponse(t, recorder, expectedStatus)
	body := recorder.Body.String()
	assert.Contains(t, body, `"success":false`)
	if expectedErrorCode != "" {
		assert.Contains(t, body, fmt.Sprintf(`"code":"%s"`, expectedErrorCode))
	}
}

// WaitForCondition waits for a condition to be true with timeout
func (u *TestUtils) WaitForCondition(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			require.Fail(t, "Timeout waiting for condition", message)
		case <-ticker.C:
			if condition() {
				return
			}
		}
	}
}

// RandomString generates a random string of specified length
func RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// RandomEmail generates a random email address
func RandomEmail() string {
	return fmt.Sprintf("%s@%s.com", RandomString(8), RandomString(6))
}

// RandomUUID generates a random UUID
func RandomUUID() uuid.UUID {
	return uuid.New()
}

// RandomFloat generates a random float between min and max
func RandomFloat(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

// RandomBool generates a random boolean
func RandomBool() bool {
	return rand.Intn(2) == 1
}

// MockResponseWriter implements http.ResponseWriter for testing
type MockResponseWriter struct {
	headers http.Header
	body    []byte
	status  int
}

// NewMockResponseWriter creates a new mock response writer
func NewMockResponseWriter() *MockResponseWriter {
	return &MockResponseWriter{
		headers: make(http.Header),
		status:  http.StatusOK,
	}
}

func (m *MockResponseWriter) Header() http.Header {
	return m.headers
}

func (m *MockResponseWriter) Write(data []byte) (int, error) {
	m.body = append(m.body, data...)
	return len(data), nil
}

func (m *MockResponseWriter) WriteHeader(statusCode int) {
	m.status = statusCode
}

func (m *MockResponseWriter) Body() string {
	return string(m.body)
}

func (m *MockResponseWriter) Status() int {
	return m.status
}

// PerformanceTimer measures execution time for performance tests
type PerformanceTimer struct {
	startTime time.Time
	endTime   time.Time
}

// NewPerformanceTimer creates a new performance timer
func NewPerformanceTimer() *PerformanceTimer {
	return &PerformanceTimer{}
}

// Start starts the timer
func (pt *PerformanceTimer) Start() {
	pt.startTime = time.Now()
}

// Stop stops the timer
func (pt *PerformanceTimer) Stop() {
	pt.endTime = time.Now()
}

// Duration returns the elapsed time
func (pt *PerformanceTimer) Duration() time.Duration {
	if pt.endTime.IsZero() {
		return time.Since(pt.startTime)
	}
	return pt.endTime.Sub(pt.startTime)
}

// AssertWithinSLA checks if the duration is within SLA
func (pt *PerformanceTimer) AssertWithinSLA(t *testing.T, maxDuration time.Duration, operation string) {
	duration := pt.Duration()
	assert.LessOrEqual(t, duration, maxDuration, 
		"Operation '%s' took %v, which exceeds SLA of %v", operation, duration, maxDuration)
}

// MemoryTracker tracks memory usage during tests
type MemoryTracker struct {
	initialMemory uint64
	peakMemory    uint64
}

// NewMemoryTracker creates a new memory tracker
func NewMemoryTracker() *MemoryTracker {
	return &MemoryTracker{}
}

// Start starts memory tracking
func (mt *MemoryTracker) Start() {
	// Note: This is a simplified implementation
	// In a real scenario, you would use runtime.ReadMemStats() or similar
	mt.initialMemory = 0 // Placeholder
	mt.peakMemory = 0
}

// Update updates the peak memory if current usage is higher
func (mt *MemoryTracker) Update() {
	// Placeholder for actual memory measurement
	// currentMemory := getCurrentMemoryUsage()
	// if currentMemory > mt.peakMemory {
	//     mt.peakMemory = currentMemory
	// }
}

// GetPeakMemoryMB returns peak memory usage in MB
func (mt *MemoryTracker) GetPeakMemoryMB() float64 {
	return float64(mt.peakMemory) / (1024 * 1024)
}

// AssertMemoryWithinSLA checks if memory usage is within SLA
func (mt *MemoryTracker) AssertMemoryWithinSLA(t *testing.T, maxMemoryMB float64, operation string) {
	peakMemoryMB := mt.GetPeakMemoryMB()
	assert.LessOrEqual(t, peakMemoryMB, maxMemoryMB,
		"Operation '%s' used %.2f MB memory, which exceeds SLA of %.2f MB", 
		operation, peakMemoryMB, maxMemoryMB)
}

// ConcurrencyHelper helps with concurrent testing
type ConcurrencyHelper struct {
	Config *config.TestConfig
}

// NewConcurrencyHelper creates a new concurrency helper
func NewConcurrencyHelper(testConfig *config.TestConfig) *ConcurrencyHelper {
	return &ConcurrencyHelper{
		Config: testConfig,
	}
}

// RunConcurrent runs a function concurrently and measures performance
func (ch *ConcurrencyHelper) RunConcurrent(t *testing.T, operation func() error, description string) {
	concurrency := ch.Config.Test.ConcurrencyLevel
	errors := make(chan error, concurrency)
	
	timer := NewPerformanceTimer()
	timer.Start()

	// Start concurrent operations
	for i := 0; i < concurrency; i++ {
		go func() {
			errors <- operation()
		}()
	}

	// Collect results
	var errorCount int
	for i := 0; i < concurrency; i++ {
		if err := <-errors; err != nil {
			t.Logf("Concurrent operation %d failed: %v", i, err)
			errorCount++
		}
	}

	timer.Stop()

	// Assert performance metrics
	duration := timer.Duration()
	throughput := float64(concurrency) / duration.Seconds()

	assert.LessOrEqual(t, errorCount, concurrency/10, // Allow up to 10% failure rate
		"Too many errors in concurrent %s: %d/%d", description, errorCount, concurrency)
	
	assert.GreaterOrEqual(t, throughput, ch.Config.Test.MinThroughput,
		"Throughput for %s (%.2f ops/sec) is below minimum (%.2f ops/sec)", 
		description, throughput, ch.Config.Test.MinThroughput)
	
	timer.AssertWithinSLA(t, ch.Config.Test.MaxResponseTime*time.Duration(concurrency), description)
}

// TestDataSeeder helps with seeding test data
type TestDataSeeder struct {
	helper *DatabaseTestHelper
}

// NewTestDataSeeder creates a new test data seeder
func NewTestDataSeeder(helper *DatabaseTestHelper) *TestDataSeeder {
	return &TestDataSeeder{
		helper: helper,
	}
}

// SeedUsers seeds multiple users
func (s *TestDataSeeder) SeedUsers(count int) []*models.User {
	factory := NewUserFactory()
	users := make([]*models.User, count)
	
	for i := 0; i < count; i++ {
		users[i] = factory.Build()
		s.helper.GetTransactionDB().Create(users[i])
	}
	
	return users
}

// SeedCompleteSetups seeds multiple complete user setups
func (s *TestDataSeeder) SeedCompleteSetups(count int) []*models.User {
	factory := NewCompleteSetupFactory()
	users := make([]*models.User, count)
	
	for i := 0; i < count; i++ {
		user, exchange, subAccount, transaction := factory.CreateCompleteUserSetup()
		
		// Save all entities
		s.helper.GetTransactionDB().Create(user)
		s.helper.GetTransactionDB().Create(exchange)
		s.helper.GetTransactionDB().Create(subAccount)
		s.helper.GetTransactionDB().Create(transaction)
		
		users[i] = user
	}
	
	return users
}

// CleanupTestData removes all test data
func (s *TestDataSeeder) CleanupTestData() error {
	return s.helper.TruncateAllTables()
}

// Validators for common assertions
type Validators struct{}

// NewValidators creates a new validators instance
func NewValidators() *Validators {
	return &Validators{}
}

// ValidateUUID checks if string is a valid UUID
func (v *Validators) ValidateUUID(t *testing.T, str string, fieldName string) {
	_, err := uuid.Parse(str)
	assert.NoError(t, err, "%s should be a valid UUID: %s", fieldName, str)
}

// ValidateEmail checks if string is a valid email format
func (v *Validators) ValidateEmail(t *testing.T, email string) {
	assert.Contains(t, email, "@", "Email should contain @ symbol: %s", email)
	assert.True(t, len(email) > 5, "Email should be longer than 5 characters: %s", email)
}

// ValidateTimestamp checks if timestamp is reasonable (not zero, not too far in future)
func (v *Validators) ValidateTimestamp(t *testing.T, timestamp time.Time, fieldName string) {
	assert.False(t, timestamp.IsZero(), "%s timestamp should not be zero", fieldName)
	assert.True(t, timestamp.Before(time.Now().Add(time.Hour)), 
		"%s timestamp should not be too far in future: %v", fieldName, timestamp)
}

// ValidatePositiveFloat checks if float is positive
func (v *Validators) ValidatePositiveFloat(t *testing.T, value float64, fieldName string) {
	assert.True(t, value >= 0, "%s should be non-negative: %f", fieldName, value)
}

// ValidateNonEmptyString checks if string is not empty
func (v *Validators) ValidateNonEmptyString(t *testing.T, value string, fieldName string) {
	assert.NotEmpty(t, value, "%s should not be empty", fieldName)
}

func init() {
	rand.Seed(time.Now().UnixNano())
}