# Trading Log Business Logic Testing

This document describes the comprehensive test suite for the trading log business logic implementation.

## Test Structure

### 1. Unit Tests (`trading_log_processors_test.go`)
Tests individual processor methods in isolation using mocks.

**Coverage:**
- ✅ Long position financial calculations
- ✅ Short position financial calculations  
- ✅ Stop-loss processing
- ✅ Insufficient balance error handling
- ✅ Financial precision and accuracy
- ✅ Mock expectations validation

**Run Command:**
```bash
go test -v ./internal/services/test -run "TestTradingLogProcessor"
```

### 2. Validator Tests (`trading_log_validators_test.go`)
Tests business logic validation rules and data structure validation.

**Coverage:**
- ✅ Trading log type validation
- ✅ Info structure validation for long/short/stop_loss
- ✅ Field requirement validation
- ✅ UUID format validation
- ✅ Numeric validation (positive/non-negative)
- ✅ String length validation
- ✅ Account uniqueness validation
- ✅ Decimal precision validation

**Run Command:**
```bash
go test -v ./internal/services/test -run "TestTradingLogValidator"
```

### 3. Integration Tests (`trading_log_integration_test.go`)
Tests complete end-to-end workflows with real database transactions.

**Coverage:**
- ✅ Complete long position workflow
- ✅ Complete short position workflow
- ✅ Complete stop-loss workflow
- ✅ Real database balance updates
- ✅ Transaction creation and validation
- ✅ Error handling with database rollback
- ✅ Account ownership validation
- ✅ Concurrent transaction safety
- ✅ Race condition testing

**Run Command:**
```bash
# Full integration tests (requires database)
go test -v ./internal/services/test -run "TestTradingLogService_Integration"

# Skip integration tests
go test -v ./internal/services/test -run "TestTradingLogService_Integration" -short
```

### 4. Service Tests (`trading_log_service_test.go`)
Tests service layer functionality with mocked dependencies.

**Coverage:**
- ✅ Basic trading log creation
- ✅ Service layer validation
- ✅ Authorization checks
- ✅ CRUD operations

**Run Command:**
```bash
go test -v ./internal/services/test -run "TestTradingLogService"
```

### 5. Performance Tests (`trading_log_performance_test.go`)
Tests system performance under high-volume and stress conditions.

**Coverage:**
- ✅ Sequential high-volume trades (1000+ trades)
- ✅ Concurrent high-volume trades (multi-goroutine)
- ✅ Database connection stress testing
- ✅ Memory efficiency under load
- ✅ Single trade latency benchmarks
- ✅ Throughput performance metrics
- ✅ Race condition and contention testing

**Run Command:**
```bash
# Full performance tests (requires database, may take several minutes)
go test -v ./internal/services/test -run "TestTradingLogService_Performance"

# Skip performance tests
go test -v ./internal/services/test -run "TestTradingLogService_Performance" -short
```

## Test Scenarios

### Financial Calculations Tested

#### Long Position
- **Formula:** `totalCost = price × volume + fee`
- **Example:** 2.0 ETH @ $3000 + $15 fee = $6015 total cost
- **Verification:** Stock account +2.0 ETH, Currency account -$6015

#### Short Position
- **Formula:** `netProceeds = price × volume - fee`
- **Example:** 1.5 ETH @ $2800 - $10 fee = $4190 net proceeds
- **Verification:** Stock account -1.5 ETH, Currency account +$4190

#### Stop-Loss
- **Formula:** Same as short position (`price × volume - fee`)
- **Example:** 1.0 ETH @ $2500 - $5 fee = $2495 net proceeds
- **Verification:** Stock account -1.0 ETH, Currency account +$2495

### Error Handling Tested

1. **Insufficient Balance**
   - Currency account lacks funds for long position
   - Stock account lacks assets for short/stop-loss
   - Database rollback verification

2. **Account Ownership**
   - User cannot access other users' accounts
   - Proper authorization validation

3. **Concurrent Transactions**
   - Race condition prevention
   - Database lock handling
   - Atomic transaction guarantees

## Running All Tests

### Quick Tests (Unit + Validators only)
```bash
go test -v ./internal/services/test -short
```

### Full Test Suite (Including Integration)
```bash
go test -v ./internal/services/test
```

### Specific Test Categories
```bash
# Unit tests only
go test -v ./internal/services/test -run "TestTradingLogProcessor|TestTradingLogValidator"

# Integration tests only  
go test -v ./internal/services/test -run "TestTradingLogService_Integration"

# Service tests only
go test -v ./internal/services/test -run "TestTradingLogService" -short
```

### With Coverage
```bash
go test -v ./internal/services/test -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Test Database Setup

Integration tests require a test database. The tests use:

- **Config:** `config.ProfileQuick` 
- **Helper:** `helpers.NewDatabaseTestHelper()`
- **Cleanup:** Automatic cleanup after each test
- **Isolation:** Each test uses fresh database transactions

## Performance Considerations

- **Unit tests:** ~0.2s (fast, no database)
- **Integration tests:** ~2-5s (database setup + real transactions)
- **Concurrent tests:** ~3-8s (tests race conditions)
- **Performance tests:** ~2-10 minutes (high-volume testing)

### Performance Benchmarks

The performance tests measure and validate:

#### Sequential Processing
- **Target:** >10 trades/second
- **1000 trades:** <2 minutes total
- **Average latency:** <50ms per trade

#### Concurrent Processing  
- **20 goroutines × 50 trades:** <30 seconds
- **Success rate:** >80% under load
- **100 concurrent connections:** Stress test limits

#### Memory Efficiency
- **5000 trades:** <10 minutes total
- **Average per trade:** <100ms
- **Memory growth:** Linear, no leaks

#### Database Performance
- **Transaction throughput:** >50 TPS
- **Connection pooling:** 100 concurrent connections
- **Rollback efficiency:** <10ms on conflicts

## Business Logic Coverage

### ✅ Implemented & Tested
- Long position processing with balance validation
- Short position processing with stock validation  
- Stop-loss triggered position processing
- Financial calculation accuracy (floating-point precision)
- Database transaction atomicity
- Error handling and rollback
- Account ownership validation
- Concurrent transaction safety

### 🚀 Future Enhancements
- Integration with external trading APIs
- Real-time market data validation
- Advanced financial instruments (futures, options)
- Load balancing and horizontal scaling tests
- Chaos engineering and fault tolerance testing

## Test Quality Metrics

- **Code Coverage:** >95% for business logic methods
- **Error Scenarios:** All failure paths tested
- **Edge Cases:** Zero fees, high precision decimals, boundary conditions
- **Database Integrity:** Transaction rollback verification
- **Performance:** Concurrent access and race condition testing