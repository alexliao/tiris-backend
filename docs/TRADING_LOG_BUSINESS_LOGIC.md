# Trading Log Business Logic Implementation

This document demonstrates the implemented trading log business logic functionality.

## Overview

The trading log business logic automatically processes `long`, `short`, and `stop_loss` trading log types to:

1. **Validate** trading log data structure and financial constraints
2. **Calculate** financial transactions based on trading parameters
3. **Update** sub-account balances atomically
4. **Create** audit trail of all financial transactions

## Implementation Files

### Core Components

- **`trading_log_validators.go`**: Data validation and structure verification
- **`trading_log_processors.go`**: Business logic processing and transaction management
- **`trading_log_service.go`**: Enhanced service with atomic processing

### Test Coverage

- **`trading_log_validators_test.go`**: Comprehensive validation testing
- **`trading_log_business_logic_test.go`**: Integration test placeholder
- **`trading_log_service_test.go`**: Existing service functionality tests

## API Usage Examples

### Long Position (Buy Order)

```json
POST /v1/trading-logs
{
  "exchange_id": "uuid-of-exchange",
  "type": "long",
  "source": "manual",
  "message": "ETH long position opened",
  "info": {
    "stock_account_id": "eth-account-uuid",
    "currency_account_id": "usdt-account-uuid", 
    "price": 3000.00,
    "volume": 2.0,
    "stock": "ETH",
    "currency": "USDT",
    "fee": 12.00
  }
}
```

**Business Logic Processing:**
1. **Stock Account**: +2.0 ETH (credit)
2. **Currency Account**: -6,012.00 USDT (debit: 3000 × 2 + 12)

### Short Position (Sell Order)

```json
POST /v1/trading-logs
{
  "exchange_id": "uuid-of-exchange",
  "type": "short",
  "source": "manual", 
  "message": "ETH short position opened",
  "info": {
    "stock_account_id": "eth-account-uuid",
    "currency_account_id": "usdt-account-uuid",
    "price": 3000.00,
    "volume": 1.5,
    "stock": "ETH", 
    "currency": "USDT",
    "fee": 9.00
  }
}
```

**Business Logic Processing:**
1. **Stock Account**: -1.5 ETH (debit)
2. **Currency Account**: +4,491.00 USDT (credit: 3000 × 1.5 - 9)

### Stop-Loss Order

```json
POST /v1/trading-logs
{
  "exchange_id": "uuid-of-exchange",
  "type": "stop_loss",
  "source": "bot",
  "message": "ETH stop-loss triggered",
  "info": {
    "stock_account_id": "eth-account-uuid",
    "currency_account_id": "usdt-account-uuid",
    "price": 2500.00,
    "volume": 1.0,
    "stock": "ETH",
    "currency": "USDT", 
    "fee": 5.00
  }
}
```

**Business Logic Processing:**
1. **Stock Account**: -1.0 ETH (debit)
2. **Currency Account**: +2,495.00 USDT (credit: 2500 × 1.0 - 5)

## Enhanced Response Format

When business logic is applied, the response includes additional metadata:

```json
{
  "success": true,
  "data": {
    "id": "trading-log-uuid",
    "user_id": "user-uuid",
    "exchange_id": "exchange-uuid",
    "type": "long",
    "source": "manual",
    "message": "ETH long position opened",
    "timestamp": "2025-01-18T10:00:00Z",
    "info": {
      "stock_account_id": "eth-account-uuid",
      "currency_account_id": "usdt-account-uuid",
      "price": 3000.00,
      "volume": 2.0,
      "stock": "ETH",
      "currency": "USDT",
      "fee": 12.00,
      "processed_transactions": 2,
      "updated_accounts": 2,
      "transaction_ids": ["tx-uuid-1", "tx-uuid-2"]
    }
  }
}
```

## Validation Rules

### Required Fields (for business logic types)
- `stock_account_id`: Valid UUID of existing sub-account
- `currency_account_id`: Valid UUID of existing sub-account (must be different from stock account)
- `price`: Positive decimal (up to 8 decimal places)
- `volume`: Positive decimal (up to 8 decimal places)
- `stock`: Asset symbol (1-20 characters)
- `currency`: Currency symbol (1-20 characters)
- `fee`: Non-negative decimal (up to 8 decimal places)

### Business Rules
- **Account Ownership**: Both sub-accounts must belong to the requesting user
- **Sufficient Balance**: 
  - Long positions: Currency account must have enough balance for (price × volume + fee)
  - Short/Stop-loss: Stock account must have enough balance for volume
- **Account Validation**: Referenced exchange and sub-accounts must exist and be accessible

### Error Responses

**Validation Error Example:**
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "failed to process trading log: info validation failed: validation error for price in long: must be a positive number",
    "trace_id": "trace-uuid"
  }
}
```

**Insufficient Balance Example:**
```json
{
  "success": false,
  "error": {
    "code": "BUSINESS_LOGIC_ERROR", 
    "message": "failed to process trading log: insufficient balance in currency account: required 6012.00000000, available 1000.00000000",
    "trace_id": "trace-uuid"
  }
}
```

## Backward Compatibility

- **Non-business logic types** (e.g., "trade", "strategy", "manual") continue to work as before
- **Existing API clients** are unaffected unless they use the new business logic types
- **Database schema** remains unchanged - all enhancements use existing models

## Transaction Safety

- **Atomic Operations**: All balance updates and transaction creation happen within a single database transaction
- **Rollback on Failure**: Any error during processing rolls back all changes
- **Audit Trail**: Complete transaction history for regulatory compliance
- **Consistency**: No partial updates can occur due to transaction boundaries

## Testing

### Unit Tests
- ✅ **Validation Logic**: Comprehensive field validation and error scenarios
- ✅ **Type Processing**: Business logic calculations for all trading types
- ✅ **Error Handling**: Insufficient balance, invalid accounts, malformed data

### Integration Tests
- 🔄 **Database Integration**: Full end-to-end testing with real database (planned)
- 🔄 **API Integration**: Complete request-response cycle testing (planned)
- 🔄 **Concurrency Testing**: Multi-user concurrent trading scenarios (planned)

## Performance Considerations

- **Database Transactions**: Minimal transaction scope for optimal performance
- **Validation Caching**: Account lookups are optimized within transaction boundaries  
- **Error Fast-Fail**: Invalid requests fail quickly before expensive operations
- **Memory Efficiency**: Streaming JSON processing for large info objects

## Monitoring and Observability

The implementation provides hooks for:
- **Metrics Collection**: Transaction counts, processing times, error rates
- **Audit Logging**: Complete financial transaction audit trail
- **Error Tracking**: Detailed error context for debugging
- **Performance Monitoring**: Database transaction timing and success rates