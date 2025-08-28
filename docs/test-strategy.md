# Tiris Backend Test Strategy and Test Plan

## 1. Test Strategy Overview

### 1.1 Testing Objectives
- Ensure functional correctness of all API endpoints
- Validate data integrity and consistency across all operations
- Verify security measures and access controls
- Confirm performance requirements are met
- Ensure system reliability and error handling
- Validate integration with external services (OAuth, tradings)

### 1.2 Testing Principles
- **Test-Driven Development (TDD)**: Write tests before implementation
- **Shift-Left Testing**: Integrate testing early in development cycle
- **Risk-Based Testing**: Prioritize testing based on business risk
- **Automated Testing**: Maximize automation for regression and CI/CD
- **Comprehensive Coverage**: Target 80%+ code coverage

### 1.3 Test Pyramid Strategy

```
    ┌─────────────────┐
    │   E2E Tests     │  ← Few, slow, expensive
    │   (Manual/Auto) │
    ├─────────────────┤
    │ Integration     │  ← Some, medium speed
    │ Tests           │
    ├─────────────────┤
    │ Unit Tests      │  ← Many, fast, cheap
    │                 │
    └─────────────────┘
```

**Distribution:**
- Unit Tests: 70%
- Integration Tests: 20%
- End-to-End Tests: 10%

## 2. Test Levels and Types

### 2.1 Unit Tests

**Scope**: Individual functions, methods, and components

**Test Areas:**
- Business logic validation
- Data transformation functions
- Utility functions
- Model validations
- Error handling
- JSON info column serialization/deserialization
- Info field validation and constraints

**Test Framework**: Go testing package + testify

**Coverage Target**: 85%

**Example Test Structure:**
```go
func TestUserService_CreateUser(t *testing.T) {
    tests := []struct {
        name    string
        input   CreateUserRequest
        want    *User
        wantErr bool
    }{
        {
            name: "valid user creation",
            input: CreateUserRequest{
                Username: "testuser",
                Email: "test@example.com",
            },
            want: &User{
                Username: "testuser",
                Email: "test@example.com",
                Status: "active",
                Info: map[string]interface{}{},
            },
            wantErr: false,
        },
        {
            name: "user creation with info data",
            input: CreateUserRequest{
                Username: "testuser2",
                Email: "test2@example.com",
                Info: map[string]interface{}{
                    "oauth_provider": "google",
                    "preferences": map[string]interface{}{
                        "theme": "dark",
                        "language": "en",
                    },
                },
            },
            want: &User{
                Username: "testuser2",
                Email: "test2@example.com",
                Status: "active",
                Info: map[string]interface{}{
                    "oauth_provider": "google",
                    "preferences": map[string]interface{}{
                        "theme": "dark",
                        "language": "en",
                    },
                },
            },
            wantErr: false,
        },
        // More test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### 2.2 Integration Tests

**Scope**: Component interactions, database operations, external service integration

**Test Areas:**
- API endpoint functionality
- Database operations
- OAuth provider integration
- Trading Platform API communication
- Service layer interactions
- NATS event processing integration
- Event consumer functionality

**Test Framework**: Go testing + Docker containers for dependencies

**Coverage Target**: 70%

**Test Environment:**
- PostgreSQL test database
- Mock OAuth providers
- Mock trading platform APIs
- In-memory caching
- NATS test server
- Mock event publishers

### 2.3 End-to-End Tests

**Scope**: Complete user workflows

**Test Areas:**
- User registration and authentication flow
- Trading Platform binding and management
- Sub-account creation and operations
- Trading log creation and transaction generation
- Complete trading workflow simulation
- End-to-end event processing from bot to database
- Event replay and recovery scenarios

**Test Framework**: Go testing + real test environment

**Coverage Target**: Key user journeys (20+ scenarios)

## 3. Testing Framework and Tools

### 3.1 Primary Tools

**Unit Testing:**
- Go built-in `testing` package
- `testify/assert` for assertions
- `testify/mock` for mocking
- `testify/suite` for test suites

**Integration Testing:**
- `dockertest` for containerized dependencies
- `gomock` for interface mocking
- `httptest` for HTTP testing

**Database Testing:**
- `go-txdb` for transaction-based test isolation
- `migrate` for test database setup
- Custom test fixtures and factories

**Code Coverage:**
- Go built-in coverage tools
- `gocov` for coverage reporting
- CI/CD coverage reporting

### 3.2 Test Data Management

**Test Database Strategy:**
- Separate test database per test suite
- Transaction rollback for test isolation
- Fixtures for consistent test data
- Factory pattern for dynamic test data

**Test Data Factories:**
```go
type UserFactory struct{}

func (f *UserFactory) Build() *User {
    return &User{
        ID:       uuid.New(),
        Username: fmt.Sprintf("user_%d", time.Now().UnixNano()),
        Email:    fmt.Sprintf("test_%d@example.com", time.Now().UnixNano()),
        Status:   "active",
    }
}

func (f *UserFactory) WithEmail(email string) *User {
    user := f.Build()
    user.Email = email
    return user
}
```

## 4. Test Environment Setup

### 4.1 Development Environment

**Local Testing Setup:**
```yaml
# docker-compose.test.yml
version: '3.8'
services:
  postgres-test:
    image: timescale/timescaledb:latest-pg15
    environment:
      POSTGRES_DB: tiris_test
      POSTGRES_USER: test_user
      POSTGRES_PASSWORD: test_pass
    ports:
      - "5433:5432"
  
  redis-test:
    image: redis:7-alpine
    ports:
      - "6380:6379"
  
  nats-test:
    image: nats:2.10-alpine
    command: ["--jetstream", "--store_dir", "/data"]
    ports:
      - "4222:4222"
      - "8222:8222"
    volumes:
      - nats_data:/data

volumes:
  nats_data:
```

**Environment Variables:**
```bash
export TEST_DB_HOST=localhost
export TEST_DB_PORT=5433
export TEST_DB_NAME=tiris_test
export TEST_DB_USER=test_user
export TEST_DB_PASS=test_pass
export TEST_NATS_URL=nats://localhost:4222
export TEST_OAUTH_MOCK=true
export TEST_TRADING_PLATFORM_MOCK=true
```

### 4.2 CI/CD Environment

**GitHub Actions Configuration:**
```yaml
name: Test Suite
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    
    services:
      postgres:
        image: timescale/timescaledb:latest-pg15
        env:
          POSTGRES_DB: tiris_test
          POSTGRES_USER: test_user
          POSTGRES_PASSWORD: test_pass
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    
    steps:
    - uses: actions/checkout@v3
    - uses: actions/setup-go@v3
      with:
        go-version: '1.21'
    
    - name: Run tests
      run: |
        go mod download
        go test -v -race -coverprofile=coverage.out ./...
        go tool cover -html=coverage.out -o coverage.html
    
    - name: Upload coverage
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.out
```

## 5. Test Cases and Scenarios

### 5.1 Authentication and Authorization Tests

**Test Cases:**

```go
func TestAuthenticationFlow(t *testing.T) {
    testCases := []struct {
        name           string
        provider       string
        mockResponse   OAuthResponse
        expectedStatus int
        expectedUser   *User
    }{
        {
            name:     "Google OAuth success",
            provider: "google",
            mockResponse: OAuthResponse{
                Email:    "user@gmail.com",
                Name:     "Test User",
                Picture:  "https://avatar.url",
            },
            expectedStatus: 200,
            expectedUser: &User{
                Email:    "user@gmail.com",
                Username: "Test User",
                Avatar:   "https://avatar.url",
            },
        },
        {
            name:     "Invalid OAuth provider",
            provider: "invalid",
            expectedStatus: 400,
        },
        // More test cases...
    }
}

func TestJWTTokenValidation(t *testing.T) {
    testCases := []struct {
        name        string
        token       string
        expectValid bool
    }{
        {
            name:        "Valid JWT token",
            token:       generateValidJWT(),
            expectValid: true,
        },
        {
            name:        "Expired JWT token",
            token:       generateExpiredJWT(),
            expectValid: false,
        },
        // More test cases...
    }
}
```

### 5.2 User Management Tests

**Test Scenarios:**
- User profile creation from OAuth
- User profile updates
- User settings management
- Account status changes (admin)
- Data validation and constraints
- Info column JSON storage and retrieval
- Info field updates and merging

### 5.3 Trading Platform Management Tests

**Test Scenarios:**
- Trading Platform binding with valid credentials
- Trading Platform binding with invalid credentials
- Trading Platform configuration updates
- Trading Platform removal
- API key encryption/decryption
- Multiple tradings per user

### 5.4 Sub-account Management Tests

**Test Scenarios:**
- Sub-account creation with valid balance
- Balance validation (non-negative, within limits)
- Deposit and withdrawal operations
- Balance calculation accuracy
- Locked balance management
- Sub-account deletion constraints

### 5.5 Transaction Management Tests

**Test Scenarios:**
- Transaction creation from trading logs
- Balance update accuracy
- Transaction history queries
- Time-series data integrity
- Concurrent transaction handling
- Transaction rollback scenarios

### 5.6 Trading Log Management Tests

**Test Scenarios:**
- Log creation from different sources (bot/manual)
- Log type validation
- Related transaction creation
- Log querying with filters
- Real-time log streaming
- Log retention policies

### 5.7 JSON Info Column Tests

**Test Scenarios:**
- Info field storage and retrieval for all entities
- JSON validation and schema enforcement
- Info field updates and partial updates
- Complex nested JSON structure handling
- JSON field querying and filtering
- Info field size limits and constraints
- JSON serialization/deserialization accuracy

### 5.8 NATS Event Processing Tests

**Test Scenarios:**
- Event consumer connection and subscription
- Event message parsing and validation
- Event deduplication logic
- Event processing order per user/sub-account
- Failed event handling and dead letter queue
- Event replay functionality
- Consumer group load balancing
- Event acknowledgment and retry mechanisms
- Connection recovery and reconnection
- Event processing metrics and monitoring

## 6. Performance Testing

### 6.1 Load Testing

**Test Scenarios:**
- Concurrent user authentication (100 users/second)
- High-frequency trading log ingestion (1000 logs/minute)
- High-frequency NATS event processing (10,000 events/minute)
- Concurrent sub-account operations (50 operations/second)
- Large transaction history queries (10,000+ records)
- API rate limiting validation
- NATS consumer throughput and latency testing

**Tools:**
- `wrk` for HTTP load testing
- Custom Go benchmarks for specific operations
- Database connection pool testing
- NATS benchmarking tools for event throughput
- Custom event publishers for load testing

**Performance Benchmarks:**
```go
func BenchmarkCreateTransaction(b *testing.B) {
    // Setup
    db := setupTestDB()
    service := NewTransactionService(db)
    
    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            _, err := service.CreateTransaction(context.Background(), testTransactionData)
            if err != nil {
                b.Error(err)
            }
        }
    })
}

func BenchmarkEventProcessing(b *testing.B) {
    // Setup
    processor := setupEventProcessor()
    testEvent := generateTestTradingEvent()
    
    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            err := processor.ProcessEvent(context.Background(), testEvent)
            if err != nil {
                b.Error(err)
            }
        }
    })
}
```

### 6.2 Stress Testing

**Test Scenarios:**
- Database connection exhaustion
- Memory leak detection
- CPU usage under load
- Disk I/O performance
- Network timeout handling

## 7. Security Testing

### 7.1 Authentication Security Tests

**Test Areas:**
- JWT token security (signature, expiration, payload)
- OAuth flow security (state parameter, CSRF protection)
- Password policy enforcement (if applicable)
- Session management security
- Brute force protection

### 7.2 Authorization Security Tests

**Test Areas:**
- Role-based access control (RBAC)
- Resource ownership validation
- Cross-user data access prevention
- Admin privilege escalation
- API endpoint authorization

### 7.3 Data Security Tests

**Test Areas:**
- API key encryption/decryption
- Sensitive data exposure in logs
- SQL injection prevention
- Input validation and sanitization
- Cross-site scripting (XSS) prevention

## 8. Test Data and Fixtures

### 8.1 Test Data Strategy

**Static Fixtures:**
```go
var TestUsers = []User{
    {
        ID:       uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
        Username: "testuser1",
        Email:    "test1@example.com",
        Status:   "active",
    },
    // More users...
}

var TestTradingPlatforms = []TradingPlatform{
    {
        ID:     uuid.MustParse("456e7890-e89b-12d3-a456-426614174001"),
        UserID: uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
        Name:   "Test Binance",
        Type:   "binance",
        Status: "active",
    },
    // More tradings...
}
```

**Dynamic Factories:**
```go
type TestDataFactory struct {
    db *sql.DB
}

func (f *TestDataFactory) CreateUserWithTradingPlatform() (*User, *TradingPlatform) {
    user := f.CreateUser()
    tradingPlatform := f.CreateTradingPlatform(user.ID)
    return user, tradingPlatform
}

func (f *TestDataFactory) CreateCompleteUserSetup() (*User, *TradingPlatform, *SubAccount) {
    user := f.CreateUser()
    tradingPlatform := f.CreateTradingPlatform(user.ID)
    subAccount := f.CreateSubAccount(user.ID, tradingPlatform.ID)
    return user, tradingPlatform, subAccount
}
```

### 8.2 Test Database Management

**Database Setup:**
```go
func SetupTestDB(t *testing.T) *sql.DB {
    config := &Config{
        Host:     "localhost",
        Port:     5433,
        Database: "tiris_test",
        User:     "test_user",
        Password: "test_pass",
    }
    
    db, err := sql.Open("postgres", config.ConnectionString())
    require.NoError(t, err)
    
    // Run migrations
    err = RunMigrations(db)
    require.NoError(t, err)
    
    t.Cleanup(func() {
        db.Close()
    })
    
    return db
}

func CleanupTestData(db *sql.DB) {
    tables := []string{"trading_logs", "transactions", "sub_accounts", "tradings", "oauth_tokens", "users"}
    for _, table := range tables {
        db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
    }
}
```

## 9. Test Execution and Reporting

### 9.1 Test Execution Strategy

**Local Development:**
```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific test suite
go test -v ./internal/services/user

# Run integration tests only
go test -v -tags=integration ./...

# Run benchmarks
go test -bench=. ./...
```

**CI/CD Pipeline:**
```bash
# Fast feedback tests (unit tests)
go test -short ./...

# Full test suite (including integration)
go test -v -race -coverprofile=coverage.out ./...

# Performance regression tests
go test -bench=. -benchmem ./...
```

### 9.2 Test Reporting

**Coverage Reports:**
- Minimum 80% overall coverage
- Critical paths require 95% coverage
- HTML coverage reports for analysis
- Coverage trends tracking

**Test Results:**
- JUnit XML format for CI/CD integration
- Detailed failure reports with stack traces
- Performance benchmark comparisons
- Flaky test identification and tracking

### 9.3 Test Metrics

**Key Metrics:**
- Test execution time
- Code coverage percentage
- Test success rate
- Flaky test frequency
- Performance benchmark trends

## 10. Test Maintenance and Best Practices

### 10.1 Test Code Quality

**Best Practices:**
- Clear and descriptive test names
- Single assertion per test (when possible)
- Proper test data cleanup
- Avoid test interdependencies
- Use table-driven tests for multiple scenarios

**Code Review Guidelines:**
- Test coverage for new features
- Test maintenance for refactored code
- Performance impact assessment
- Security test validation

### 10.2 Continuous Improvement

**Test Suite Evolution:**
- Regular test suite performance review
- Flaky test analysis and fixes
- Test data management optimization
- New testing tool evaluation
- Test strategy adaptation based on defects

**Defect Analysis:**
- Root cause analysis for production bugs
- Test gap identification
- Test scenario enhancement
- Automated test creation for bug fixes

## 11. Risk-Based Testing

### 11.1 High-Risk Areas

**Critical Business Logic:**
- Financial calculations and balance updates
- Transaction integrity and consistency
- User authentication and authorization
- Trading Platform API integrations
- Data encryption and security

**High-Impact Components:**
- Database operations (especially time-series)
- API endpoints handling financial data
- OAuth integration flows
- Error handling and recovery
- Performance-critical paths

### 11.2 Risk Mitigation Strategies

**Financial Data Accuracy:**
- Comprehensive unit tests for calculations
- Integration tests for end-to-end workflows
- Data integrity validation tests
- Concurrent operation testing

**Security and Privacy:**
- Penetration testing scenarios
- Encryption validation tests
- Access control verification
- Data exposure prevention tests

**System Reliability:**
- Failover scenario testing
- Error recovery validation
- Resource exhaustion testing
- Dependency failure simulation

## 12. Test Schedule and Milestones

### 12.1 Development Phase Testing

**Week 1-2: Foundation Testing**
- Database schema validation
- Basic CRUD operations testing
- Authentication flow testing

**Week 3-4: Core Functionality Testing**
- User management testing
- Trading Platform integration testing
- Sub-account operations testing

**Week 5-6: Advanced Features Testing**
- Transaction processing testing
- Trading log management testing
- Performance optimization testing

**Week 7-8: Integration and System Testing**
- End-to-end workflow testing
- Security penetration testing
- Load and stress testing

### 12.2 Release Testing Checklist

**Pre-Release Validation:**
- [ ] All unit tests passing (>80% coverage)
- [ ] All integration tests passing
- [ ] Security tests validated
- [ ] Performance benchmarks met
- [ ] End-to-end workflows tested
- [ ] Database migration tested
- [ ] Deployment process validated
- [ ] Rollback procedures tested
- [ ] Documentation updated
- [ ] Test reports generated

This comprehensive test strategy ensures thorough validation of the Tiris Backend system while maintaining development velocity and code quality.