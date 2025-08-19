# Trading Log Business Logic Specification

**Version**: 1.0  
**Last Updated**: 2025-01-18  
**Status**: Active  

## Table of Contents

1. [Overview](#overview)
2. [Trading Log Processing](#trading-log-processing)
3. [Trading Log Types](#trading-log-types)
4. [Data Structures](#data-structures)
5. [Transaction Processing](#transaction-processing)
6. [Error Handling](#error-handling)
7. [Validation Rules](#validation-rules)
8. [Examples](#examples)

## Overview

This document defines the business logic for processing trading logs within the Tiris Backend system. When trading logs are submitted via the `POST /v1/trading-logs` API endpoint, the system performs automated financial calculations and account balance updates based on the trading log type.

### Purpose

- Define standardized processing rules for different trading log types
- Ensure consistent financial calculations across all trading operations
- Maintain data integrity through atomic transaction processing
- Provide clear specifications for system implementation and maintenance

### Scope

This specification covers the business logic for processing the following trading log types:
- Long positions (`long`)
- Short positions (`short`)
- Stop-loss orders (`stop_loss`)

## Trading Log Processing

### Process Flow

1. **API Request**: Trading log received via `POST /v1/trading-logs`
2. **Validation**: Validate log structure and required fields
3. **Type Identification**: Determine processing rules based on log type
4. **Account Resolution**: Locate referenced sub-accounts
5. **Transaction Creation**: Generate appropriate financial transactions
6. **Balance Updates**: Update sub-account balances atomically
7. **Persistence**: Save all changes within a single database transaction

### Atomicity Requirements

All operations for a single trading log must be processed within an atomic database transaction to ensure:
- Data consistency across all affected accounts
- Complete rollback in case of any processing errors
- No partial updates that could corrupt financial data

## Trading Log Types

### 1. Long Position (`long`)

**Description**: Opening a long position by purchasing an asset with the expectation that its value will increase.

**Financial Impact**:
- **Stock Account**: Credit with purchased asset volume
- **Currency Account**: Debit with purchase cost plus fees

**Processing Rules**:
1. Credit stock sub-account with asset volume
2. Debit currency sub-account with total cost (price × volume + fee)

### 2. Short Position (`short`)

**Description**: Opening a short position by selling an asset with the expectation that its value will decrease.

**Financial Impact**:
- **Stock Account**: Debit with sold asset volume
- **Currency Account**: Credit with sale proceeds minus fees

**Processing Rules**:
1. Debit stock sub-account with asset volume
2. Credit currency sub-account with net proceeds (price × volume - fee)

### 3. Stop-Loss Order (`stop_loss`)

**Description**: Executing a stop-loss order to limit losses by automatically selling an asset when it reaches a predetermined price.

**Financial Impact**: Identical to short position processing

**Processing Rules**: Same as short position, but with `reason="stop_loss"` for audit trail differentiation

## Data Structures

### Trading Log Info Field Schema

```json
{
  "stock_account_id": "uuid",      // Required: Sub-account ID for the asset
  "currency_account_id": "uuid",   // Required: Sub-account ID for the currency
  "price": "number",               // Required: Price per unit (positive decimal)
  "volume": "number",              // Required: Quantity traded (positive decimal)
  "stock": "string",               // Required: Asset symbol (e.g., "ETH")
  "currency": "string",            // Required: Currency symbol (e.g., "USDT")
  "fee": "number"                  // Required: Trading fee (non-negative decimal)
}
```

### Transaction Record Structure

Each trading log generates transactions with the following structure:

```json
{
  "sub_account_id": "uuid",
  "amount": "number",
  "direction": "credit|debit",
  "closing_balance": "number",
  "price": "number",
  "quote_symbol": "string",
  "reason": "long|short|stop_loss",
  "info": "object"  // Complete trading log JSON record
}
```

## Transaction Processing

### Long Position Processing

| Account Type | Transaction | Amount | Direction | Closing Balance |
|-------------|-------------|--------|-----------|----------------|
| Stock | Asset Purchase | `volume` | Credit | `current_balance + volume` |
| Currency | Payment | `price × volume + fee` | Debit | `current_balance - (price × volume + fee)` |

### Short Position Processing

| Account Type | Transaction | Amount | Direction | Closing Balance |
|-------------|-------------|--------|-----------|----------------|
| Stock | Asset Sale | `volume` | Debit | `current_balance - volume` |
| Currency | Proceeds | `price × volume - fee` | Credit | `current_balance + (price × volume - fee)` |

### Stop-Loss Processing

Identical to short position processing with `reason="stop_loss"` for transaction records.

## Error Handling

### Validation Errors

- **Invalid Trading Log Type**: Return 400 Bad Request
- **Missing Required Fields**: Return 400 Bad Request with field details
- **Invalid Data Types**: Return 400 Bad Request with type requirements
- **Non-existent Sub-accounts**: Return 404 Not Found

### Business Logic Errors

- **Insufficient Balance**: Return 422 Unprocessable Entity
- **Negative Values**: Return 400 Bad Request
- **Account Ownership Mismatch**: Return 403 Forbidden

### System Errors

- **Database Transaction Failure**: Return 500 Internal Server Error
- **Concurrent Modification**: Retry with exponential backoff

## Validation Rules

### Data Validation

- `price`: Must be positive decimal with up to 8 decimal places
- `volume`: Must be positive decimal with up to 8 decimal places
- `fee`: Must be non-negative decimal with up to 8 decimal places
- `stock_account_id`: Must be valid UUID of existing sub-account
- `currency_account_id`: Must be valid UUID of existing sub-account
- Account ownership: Both sub-accounts must belong to the requesting user

### Business Logic Validation

- **Sufficient Balance**: For debit operations, verify adequate sub-account balance
- **Account Types**: Validate account symbols match expected asset types
- **Fee Reasonableness**: Optional validation of fee amounts within expected ranges

## Examples

### Example 1: Long Position

**Request**:
```json
{
  "exchange_id": "123e4567-e89b-12d3-a456-426614174000",
  "type": "long",
  "source": "api",
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

**Generated Transactions**:
1. ETH Account: +2.0 ETH (credit)
2. USDT Account: -6,012.00 USDT (debit: 3000 × 2 + 12)

### Example 2: Short Position

**Request**:
```json
{
  "exchange_id": "123e4567-e89b-12d3-a456-426614174000",
  "type": "short",
  "source": "api",
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

**Generated Transactions**:
1. ETH Account: -1.5 ETH (debit)
2. USDT Account: +4,491.00 USDT (credit: 3000 × 1.5 - 9)

---

**Document Revision History**:
- v1.0 (2025-01-18): Initial professional specification based on draft requirements
