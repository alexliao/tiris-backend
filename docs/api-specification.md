# Tiris Backend API Specification

## 1. API Overview

### 1.1 Base Information
- **Production URL**: `https://backend.tiris.ai/v1`
- **Development URL**: `https://backend.dev.tiris.ai/v1`
- **Protocol**: HTTPS only
- **Authentication**: JWT Bearer tokens
- **Content Type**: `application/json`
- **API Version**: v1
- **Message Queue**: NATS JetStream for trading events
- **Event Processing**: Asynchronous via message queue

### 1.2 Response Format
All API responses follow a consistent format:

**Success Response:**
```json
{
  "success": true,
  "data": {
    // Response data object
    // Note: All data objects include an "info" field for extended/variable information
  },
  "metadata": {
    "timestamp": "2024-01-15T10:30:00Z",
    "trace_id": "abc123"
  }
}
```

**Error Response:**
```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message",
    "details": "Technical details for debugging"
  },
  "metadata": {
    "timestamp": "2024-01-15T10:30:00Z",
    "trace_id": "abc123"
  }
}
```

### 1.3 HTTP Status Codes
- `200 OK`: Successful request
- `201 Created`: Resource created successfully
- `400 Bad Request`: Invalid request data
- `401 Unauthorized`: Authentication required
- `403 Forbidden`: Access denied
- `404 Not Found`: Resource not found
- `409 Conflict`: Resource conflict
- `422 Unprocessable Entity`: Validation errors
- `429 Too Many Requests`: Rate limit exceeded
- `500 Internal Server Error`: Server error
- `503 Service Unavailable`: Service temporarily unavailable

## 2. Authentication API

### 2.1 OAuth Login
**Endpoint:** `POST /auth/login`

**Description:** Initiate OAuth login with supported providers.

**Request Body:**
```json
{
  "provider": "google|wechat",
  "redirect_uri": "https://backend.tiris.ai/callback"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "auth_url": "https://accounts.google.com/oauth/authorize?...",
    "state": "random_state_string"
  }
}
```

### 2.2 OAuth Callback
**Endpoint:** `POST /auth/callback`

**Description:** Handle OAuth callback and issue JWT token.

**Request Body:**
```json
{
  "provider": "google|wechat",
  "code": "authorization_code",
  "state": "state_from_login"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "access_token": "jwt_token_here",
    "refresh_token": "refresh_token_here",
    "expires_in": 3600,
    "user": {
      "id": "user123",
      "username": "john_doe",
      "email": "john@example.com",
      "avatar": "https://avatar.url",
      "info": {}
    }
  }
}
```

### 2.3 Token Refresh
**Endpoint:** `POST /auth/refresh`

**Description:** Refresh an expired JWT token.

**Request Body:**
```json
{
  "refresh_token": "refresh_token_here"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "access_token": "new_jwt_token",
    "expires_in": 3600
  }
}
```

### 2.4 Logout
**Endpoint:** `POST /auth/logout`

**Description:** Invalidate current session.

**Headers:**
```
Authorization: Bearer {jwt_token}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Logged out successfully"
  }
}
```

## 3. User Management API

### 3.1 Get Current User
**Endpoint:** `GET /users/me`

**Description:** Get current user's profile information.

**Headers:**
```
Authorization: Bearer {jwt_token}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "user123",
    "username": "john_doe",
    "email": "john@example.com",
    "avatar": "https://avatar.url",
    "settings": {
      "timezone": "UTC",
      "currency": "USD",
      "notifications": true
    },
    "info": {
      "oauth_provider": "google",
      "last_login": "2024-01-15T10:00:00Z"
    },
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-15T10:30:00Z"
  }
}
```

### 3.2 Update User Profile
**Endpoint:** `PUT /users/me`

**Description:** Update current user's profile information.

**Headers:**
```
Authorization: Bearer {jwt_token}
```

**Request Body:**
```json
{
  "username": "new_username",
  "avatar": "https://new-avatar.url",
  "settings": {
    "timezone": "EST",
    "currency": "EUR",
    "notifications": false
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "user123",
    "username": "new_username",
    "email": "john@example.com",
    "avatar": "https://new-avatar.url",
    "settings": {
      "timezone": "EST",
      "currency": "EUR",
      "notifications": false
    },
    "updated_at": "2024-01-15T10:35:00Z"
  }
}
```

### 3.3 Disable User Account (Admin)
**Endpoint:** `PUT /users/{user_id}/disable`

**Description:** Disable a user account (admin only).

**Headers:**
```
Authorization: Bearer {admin_jwt_token}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "User account disabled successfully"
  }
}
```

## 4. Exchange Binding Management API

### 4.1 List User Exchange Bindings
**Endpoint:** `GET /exchange-bindings`

**Description:** Get all exchange bindings for the current user.

**Headers:**
```
Authorization: Bearer {jwt_token}
```

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "eb123",
      "name": "My Binance Connection",
      "exchange": "binance",
      "type": "private",
      "status": "active",
      "created_at": "2024-01-01T00:00:00Z",
      "info": {
        "testnet": false,
        "permissions": ["spot", "futures"]
      }
    }
  ]
}
```

### 4.2 Create Exchange Binding
**Endpoint:** `POST /exchange-bindings`

**Description:** Create a new exchange binding with API credentials.

**Headers:**
```
Authorization: Bearer {jwt_token}
```

**Request Body:**
```json
{
  "name": "My Binance Connection",
  "exchange": "binance",
  "type": "private",
  "api_key": "your_api_key",
  "api_secret": "your_api_secret",
  "info": {
    "testnet": false,
    "description": "Main trading account"
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "eb123",
    "name": "My Binance Connection", 
    "exchange": "binance",
    "type": "private",
    "status": "active",
    "created_at": "2024-01-15T10:30:00Z"
  }
}
```

### 4.3 Get Exchange Binding
**Endpoint:** `GET /exchange-bindings/{id}`

**Description:** Get details of a specific exchange binding.

**Headers:**
```
Authorization: Bearer {jwt_token}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "eb123",
    "name": "My Binance Connection",
    "exchange": "binance", 
    "type": "private",
    "status": "active",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-15T10:30:00Z",
    "info": {
      "testnet": false,
      "permissions": ["spot", "futures"]
    }
  }
}
```

### 4.4 Update Exchange Binding
**Endpoint:** `PUT /exchange-bindings/{id}`

**Description:** Update an existing exchange binding.

**Headers:**
```
Authorization: Bearer {jwt_token}
```

**Request Body:**
```json
{
  "name": "Updated Binance Connection",
  "api_key": "new_api_key",
  "api_secret": "new_api_secret",
  "status": "active"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "eb123",
    "name": "Updated Binance Connection",
    "exchange": "binance",
    "type": "private", 
    "status": "active",
    "updated_at": "2024-01-15T10:30:00Z"
  }
}
```

### 4.5 Delete Exchange Binding
**Endpoint:** `DELETE /exchange-bindings/{id}`

**Description:** Delete an exchange binding (only if not referenced by active tradings).

**Headers:**
```
Authorization: Bearer {jwt_token}
```

**Response:**
```json
{
  "success": true
}
```

### 4.6 Get Public Exchange Bindings
**Endpoint:** `GET /exchange-bindings/public`

**Description:** Get all public exchange bindings available for simulation and backtesting.

**Headers:**
```
Authorization: Bearer {jwt_token}
```

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "public_binance",
      "name": "Binance",
      "exchange": "binance", 
      "type": "public",
      "status": "active",
      "info": {
        "description": "A virtual Binance exchange for simulation and backtesting"
      }
    }
  ]
}
```

## 5. Trading Management API

### 5.1 List User Tradings
**Endpoint:** `GET /tradings`

**Description:** Get all tradings created by the current user.

**Headers:**
```
Authorization: Bearer {jwt_token}
```

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "trading123",
      "name": "My Real Trading",
      "exchange_binding_id": "eb123",
      "type": "real",
      "status": "active",
      "created_at": "2024-01-01T00:00:00Z",
      "info": {
        "strategy": "momentum",
        "risk_level": "medium"
      }
    }
  ]
}
```

### 5.2 Create Trading
**Endpoint:** `POST /tradings`

**Description:** Create a new trading using an existing exchange binding.

**Headers:**
```
Authorization: Bearer {jwt_token}
```

**Request Body:**
```json
{
  "name": "My Real Trading",
  "exchange_binding_id": "eb123",
  "type": "real",
  "info": {
    "strategy": "momentum",
    "risk_level": "medium",
    "description": "Live trading with momentum strategy"
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "trading123",
    "name": "My Real Trading",
    "exchange_binding_id": "eb123",
    "type": "real",
    "status": "active",
    "created_at": "2024-01-15T10:30:00Z"
  }
}
```

### 5.3 Get Trading Details
**Endpoint:** `GET /tradings/{trading_id}`

**Description:** Get details of a specific trading.

**Headers:**
```
Authorization: Bearer {jwt_token}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "trading123",
    "name": "My Binance Account",
    "type": "binance",
    "status": "active",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-15T10:00:00Z",
    "info": {
      "last_sync": "2024-01-15T10:00:00Z",
      "balance_sync": true
    }
  }
}
```

### 5.4 Update Trading
**Endpoint:** `PUT /tradings/{trading_id}`

**Description:** Update trading configuration.

**Headers:**
```
Authorization: Bearer {jwt_token}
```

**Request Body:**
```json
{
  "name": "Updated Binance Account",
  "api_key": "new_encrypted_api_key",
  "api_secret": "new_encrypted_api_secret"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "trading123",
    "name": "Updated Binance Account",
    "type": "binance",
    "status": "active",
    "updated_at": "2024-01-15T10:35:00Z"
  }
}
```

### 5.5 Delete Trading
**Endpoint:** `DELETE /tradings/{trading_id}`

**Description:** Remove trading binding from user account.

**Headers:**
```
Authorization: Bearer {jwt_token}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Trading removed successfully"
  }
}
```

## 6. Sub-account Management API

### 6.1 List Sub-accounts
**Endpoint:** `GET /sub-accounts`

**Description:** Get all sub-accounts for the current user.

**Query Parameters:**
- `trading_id` (optional): Filter by trading ID

**Headers:**
```
Authorization: Bearer {jwt_token}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "sub_accounts": [
      {
        "id": "sub123",
        "name": "BTC Trading Account",
        "symbol": "BTC",
        "balance": "0.5",
        "trading_id": "trading123",
        "created_at": "2024-01-01T00:00:00Z",
        "info": {
          "initial_balance": "0.5",
          "description": "Account for BTC trading strategies",
          "bot_config": {
            "strategy": "momentum",
            "risk_level": "medium"
          }
        }
      }
    ]
  }
}
```

### 6.2 Create Sub-account
**Endpoint:** `POST /sub-accounts`

**Description:** Create a new sub-account.

**Headers:**
```
Authorization: Bearer {jwt_token}
```

**Request Body:**
```json
{
  "name": "BTC Trading Account",
  "symbol": "BTC",
  "balance": "0.5",
  "trading_id": "trading123",
  "info": {
    "description": "Account for BTC trading strategies"
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "sub123",
    "name": "BTC Trading Account",
    "symbol": "BTC",
    "balance": "0.5",
    "trading_id": "trading123",
    "created_at": "2024-01-15T10:30:00Z"
  }
}
```

### 6.3 Get Sub-account Details
**Endpoint:** `GET /sub-accounts/{sub_account_id}`

**Description:** Get details of a specific sub-account.

**Headers:**
```
Authorization: Bearer {jwt_token}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "sub123",
    "name": "BTC Trading Account",
    "symbol": "BTC",
    "balance": "0.5",
    "trading_id": "trading123",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-15T10:00:00Z",
    "info": {
      "initial_balance": "0.5",
      "description": "Account for BTC trading strategies",
      "bot_config": {
        "strategy": "momentum",
        "risk_level": "medium"
      }
    }
  }
}
```

### 6.4 Update Sub-account
**Endpoint:** `PUT /sub-accounts/{sub_account_id}`

**Description:** Update sub-account information.

**Headers:**
```
Authorization: Bearer {jwt_token}
```

**Request Body:**
```json
{
  "name": "Updated BTC Account",
  "info": {
    "description": "Updated description"
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "sub123",
    "name": "Updated BTC Account",
    "symbol": "BTC",
    "balance": "0.5",
    "trading_id": "trading123",
    "updated_at": "2024-01-15T10:35:00Z"
  }
}
```

### 6.5 Delete Sub-account
**Endpoint:** `DELETE /sub-accounts/{sub_account_id}`

**Description:** Delete a sub-account (only if balance is zero).

**Headers:**
```
Authorization: Bearer {jwt_token}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Sub-account deleted successfully"
  }
}
```

## 7. Transaction Management API

### 7.1 List Transactions
**Endpoint:** `GET /transactions`

**Description:** Get transaction history.

**Query Parameters:**
- `sub_account_id` (optional): Filter by sub-account
- `trading_id` (optional): Filter by trading
- `start_date` (optional): Start date (ISO 8601)
- `end_date` (optional): End date (ISO 8601)
- `direction` (optional): Filter by direction (debit/credit)
- `limit` (optional): Number of records (default: 100, max: 1000)
- `offset` (optional): Pagination offset

**Headers:**
```
Authorization: Bearer {jwt_token}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "transactions": [
      {
        "id": "tx123",
        "timestamp": "2024-01-15T10:30:00Z",
        "direction": "credit",
        "reason": "long_profit",
        "amount": "0.001",
        "closing_balance": "0.501",
        "price": "45000.00",
        "quote_symbol": "USD",
        "sub_account_id": "sub123",
        "trading_id": "trading123",
        "info": {
          "trade_id": "trade456",
          "symbol": "BTC/USD",
          "order_type": "market",
          "side": "sell"
        }
      }
    ],
    "pagination": {
      "total": 150,
      "limit": 100,
      "offset": 0,
      "has_more": true
    }
  }
}
```

### 7.2 Get Transaction Details
**Endpoint:** `GET /transactions/{transaction_id}`

**Description:** Get details of a specific transaction.

**Headers:**
```
Authorization: Bearer {jwt_token}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "tx123",
    "timestamp": "2024-01-15T10:30:00Z",
    "direction": "credit",
    "reason": "long_profit",
    "amount": "0.001",
    "closing_balance": "0.501",
    "price": "45000.00",
    "quote_symbol": "USD",
    "sub_account_id": "sub123",
    "trading_id": "trading123",
    "user_id": "user123",
    "info": {
      "trade_id": "trade456",
      "symbol": "BTC/USD",
      "order_type": "market",
      "side": "buy"
    }
  }
}
```

## 8. Trading Log Management API

### 8.1 List Trading Logs
**Endpoint:** `GET /trading-logs`

**Description:** Get trading log history.

**Query Parameters:**
- `sub_account_id` (optional): Filter by sub-account
- `trading_id` (optional): Filter by trading
- `type` (optional): Filter by log type
- `source` (optional): Filter by source (manual/bot)
- `start_date` (optional): Start date (ISO 8601)
- `end_date` (optional): End date (ISO 8601)
- `limit` (optional): Number of records (default: 100, max: 1000)
- `offset` (optional): Pagination offset

**Headers:**
```
Authorization: Bearer {jwt_token}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "trading_logs": [
      {
        "id": "log123",
        "timestamp": "2024-01-15T10:30:00Z",
        "type": "buy_order",
        "source": "bot",
        "message": "Executed buy order for 0.001 BTC",
        "transaction_id": "tx123",
        "sub_account_id": "sub123",
        "trading_id": "trading123",
        "info": {
          "order_id": "order456",
          "symbol": "BTC/USD",
          "price": "45000.00",
          "quantity": "0.001",
          "strategy": "momentum_bot",
          "confidence": 0.85
        }
      }
    ],
    "pagination": {
      "total": 500,
      "limit": 100,
      "offset": 0,
      "has_more": true
    }
  }
}
```

### 7.2 Create Trading Log
**Endpoint:** `POST /trading-logs`

**Description:** Add a new trading log entry manually. Note: Most trading logs are created automatically from NATS events sent by tiris-bot.

**⚠️ Important:** The `info` field structure must match the `type` field value. Certain trading log types trigger automatic financial calculations and require specific structured data.

**Headers:**
```
Authorization: Bearer {jwt_token}
```

#### Complete Request Structure

**Required Fields:**
- `trading_id` (string): UUID of the trading where trading activity occurred  
- `type` (string): Trading log type (1-50 characters, enum values: `long`, `short`, `stop_loss`, `deposit`, `withdraw`, `trade_execution`, `api_call`, `system_event`, `error`, `custom`)
- `source` (string): Source of the entry (`manual` or `bot`)
- `message` (string): Human-readable description (minimum 1 character)

**Optional Fields:**
- `sub_account_id` (string): Sub-account UUID (used for some trading log types)
- `transaction_id` (string): Transaction UUID for linking to specific transactions
- `info` (object): Type-specific structured data (structure depends on `type` field)

#### Business Logic Types
For these types, the backend automatically performs financial calculations and account balance updates:

**Long Position (`type: "long"`)** - Required `info` fields:
```json
{
  "stock_account_id": "eth-account-uuid",        // Sub-account UUID for the asset (required, valid UUID)
  "currency_account_id": "usdt-account-uuid",   // Sub-account UUID for the currency (required, valid UUID)
  "price": 3000.00,                             // Price per unit (required, must be > 0)
  "volume": 2.0,                                // Quantity traded (required, must be > 0)
  "stock": "ETH",                               // Asset symbol (required, 1-20 characters)
  "currency": "USDT",                           // Currency symbol (required, 1-20 characters)
  "fee": 12.00                                  // Trading fee (required, must be >= 0)
}
```

**Short Position (`type: "short"`)** - Required `info` fields:
- Same as long position structure above

**Stop-Loss (`type: "stop_loss"`)** - Required `info` fields:
- Same as long position structure above

**Deposit (`type: "deposit"`)** - Required `info` fields:
```json
{
  "account_id": "usdt-account-uuid",    // Target sub-account UUID (required, valid UUID)
  "amount": 1000.00,                    // Amount to deposit (required, must be > 0)
  "currency": "USDT"                    // Currency symbol (required, 1-20 characters)
}
```

**Withdraw (`type: "withdraw"`)** - Required `info` fields:
```json
{
  "account_id": "usdt-account-uuid",    // Source sub-account UUID (required, valid UUID)
  "amount": 500.00,                     // Amount to withdraw (required, must be > 0)
  "currency": "USDT"                    // Currency symbol (required, 1-20 characters)
}
```

#### Other Types
For non-business logic types (`trade_execution`, `api_call`, `system_event`, `error`, `custom`), the `info` field can contain any object structure.

#### Request Examples

**Long Position Request:**
```json
{
  "trading_id": "453f0347-3959-49de-8e3f-1cf7c8e0827c",
  "type": "long",
  "source": "bot", 
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

**Deposit Request:**
```json
{
  "trading_id": "453f0347-3959-49de-8e3f-1cf7c8e0827c",
  "type": "deposit",
  "source": "api",
  "message": "USDT deposit to account", 
  "info": {
    "account_id": "usdt-account-uuid",
    "amount": 1000.00,
    "currency": "USDT"
  }
}
```

**Custom Type Request:**
```json
{
  "trading_id": "453f0347-3959-49de-8e3f-1cf7c8e0827c",
  "type": "custom",
  "source": "bot",
  "message": "Custom trading event",
  "info": {
    "custom_field": "any_value",
    "metadata": {
      "strategy": "momentum",
      "confidence": 0.85
    }
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "log123",
    "timestamp": "2024-01-15T10:30:00Z",
    "type": "long",
    "source": "bot",
    "message": "ETH long position opened",
    "transaction_id": "tx123",
    "sub_account_id": null,
    "trading_id": "453f0347-3959-49de-8e3f-1cf7c8e0827c"
  }
}
```

#### Error Responses

**400 Bad Request** - Invalid `info` structure:
```json
{
  "success": false,
  "error": {
    "code": "INVALID_INFO_STRUCTURE",
    "message": "Required field 'stock_account_id' missing for trading log type 'long'",
    "details": "Business logic types require specific info field structures"
  }
}
```

**422 Unprocessable Entity** - Business logic validation failed:
```json
{
  "success": false, 
  "error": {
    "code": "INSUFFICIENT_BALANCE",
    "message": "Insufficient balance for withdraw operation",
    "details": "Account balance: 100.00, requested withdrawal: 500.00"
  }
}
```

**Note:** Trading logs from tiris-bot are created automatically via NATS message queue and do not require API calls.

### 7.3 Get Trading Log Details
**Endpoint:** `GET /trading-logs/{log_id}`

**Description:** Get details of a specific trading log.

**Headers:**
```
Authorization: Bearer {jwt_token}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "log123",
    "timestamp": "2024-01-15T10:30:00Z",
    "type": "buy_order",
    "source": "bot",
    "message": "Executed buy order for 0.001 BTC",
    "transaction_id": "tx123",
    "sub_account_id": "sub123",
    "trading_id": "trading123",
    "user_id": "user123",
    "info": {
      "order_id": "order456",
      "symbol": "BTC/USD",
      "price": "45000.00",
      "quantity": "0.001"
    }
  }
}
```

### 7.4 Delete Trading Log (Admin)
**Endpoint:** `DELETE /trading-logs/{log_id}`

**Description:** Delete a trading log (admin only).

**Headers:**
```
Authorization: Bearer {admin_jwt_token}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Trading log deleted successfully"
  }
}
```

## 8. Event Processing API

**⚠️ TODO: This API section is planned but not yet implemented in the codebase**

### 8.1 Event Processing Status
**Endpoint:** `GET /events/status`

**Description:** Get NATS event processing status and metrics.

**Headers:**
```
Authorization: Bearer {jwt_token}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "status": "healthy",
    "consumers": {
      "trading_orders": {
        "status": "active",
        "pending_messages": 5,
        "last_processed": "2024-01-15T10:30:00Z"
      },
      "trading_balance": {
        "status": "active", 
        "pending_messages": 2,
        "last_processed": "2024-01-15T10:29:55Z"
      }
    },
    "metrics": {
      "events_processed_today": 15420,
      "events_failed_today": 2,
      "average_processing_time_ms": 45
    }
  }
}
```

### 8.2 Event Replay
**Endpoint:** `POST /events/replay`

**Description:** Replay events for a specific time range (admin only).

**Headers:**
```
Authorization: Bearer {admin_jwt_token}
```

**Request Body:**
```json
{
  "start_time": "2024-01-15T10:00:00Z",
  "end_time": "2024-01-15T11:00:00Z",
  "user_id": "user123",
  "event_types": ["trading.orders.filled", "trading.balance.updated"]
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "replay_id": "replay_456",
    "status": "started",
    "estimated_events": 150,
    "message": "Event replay initiated"
  }
}
```

## 9. Health Check API

### 9.1 Liveness Check
**Endpoint:** `GET /health/live`

**Description:** Check if the service is running.

**Response:**
```json
{
  "success": true,
  "data": {
    "status": "alive",
    "timestamp": "2024-01-15T10:30:00Z"
  }
}
```

### 8.2 Readiness Check
**Endpoint:** `GET /health/ready`

**Description:** Check if the service is ready to accept requests.

**Response:**
```json
{
  "success": true,
  "data": {
    "status": "ready",
    "checks": {
      "database": "ok",
      "oauth_providers": "ok",
      "nats_connection": "ok",
      "event_consumers": "ok"
    },
    "timestamp": "2024-01-15T10:30:00Z"
  }
}
```

## 9. Error Codes Reference

### 9.1 Authentication Errors
- `AUTH_REQUIRED`: Authentication required (401)
- `INVALID_TOKEN`: Invalid or expired token (401)
- `ACCESS_DENIED`: Access denied (403)
- `OAUTH_ERROR`: OAuth provider error (400)

### 9.2 Validation Errors
- `INVALID_REQUEST`: Invalid request format (400)
- `MISSING_FIELD`: Required field missing (400)
- `INVALID_VALUE`: Field value invalid (400)

### 9.3 Uniqueness Constraint Errors (409 Conflict)
- `TRADING_NAME_EXISTS`: Trading name already exists for this user
- `API_KEY_EXISTS`: API key already exists for this user  
- `API_SECRET_EXISTS`: API secret already exists for this user
- `SUBACCOUNT_NAME_EXISTS`: Sub-account name already exists for this trading
- `EMAIL_EXISTS`: Email address already exists (global uniqueness)

### 9.4 Resource Errors
- `NOT_FOUND`: Resource not found (404)
- `INSUFFICIENT_BALANCE`: Not enough balance (400)
- `TRADING_ERROR`: Trading API error (502)
- `INVALID_INFO_STRUCTURE`: Trading log info field structure invalid (400)

### 9.5 System Errors
- `INTERNAL_ERROR`: Internal server error (500)
- `DATABASE_ERROR`: Database operation failed (500)
- `SERVICE_UNAVAILABLE`: Service temporarily unavailable (503)

## 10. Rate Limiting

### 10.1 Rate Limits
- **General API**: 1000 requests per hour per user
- **Authentication**: 60 requests per hour per IP
- **Trading Operations**: 600 requests per hour per user

**Note**: Rate limits are configurable via environment variables:
- `RATE_LIMIT_ENABLED` - Enable/disable rate limiting globally (default: true, set to 'false' to disable)
- `API_RATE_LIMIT_PER_HOUR` - General API endpoints (default: 1000)
- `AUTH_RATE_LIMIT_PER_HOUR` - Authentication endpoints (default: 60)  
- `TRADING_RATE_LIMIT_PER_HOUR` - Trading operations (default: 600)

### 10.2 Rate Limit Headers
```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1642176000
X-RateLimit-Window: 3600
```

## 11. Pagination

### 11.1 Query Parameters
- `limit`: Number of records to return (default: 100, max: 1000)
- `offset`: Number of records to skip

### 11.2 Response Format
```json
{
  "success": true,
  "data": {
    "items": [...],
    "pagination": {
      "total": 1500,
      "limit": 100,
      "offset": 0,
      "has_more": true,
      "next_offset": 100
    }
  }
}
```

## 12. Resources

### 12.1 Domain Configuration
- **Production API**: `https://backend.tiris.ai/v1`
- **Development API**: `https://backend.dev.tiris.ai/v1`
- **Frontend Portal**: `https://backend.tiris.ai`
- **Development Portal**: `https://backend.dev.tiris.ai`

### 12.2 Environment-Specific Endpoints

**Production OAuth Callbacks:**
- Google: `https://backend.tiris.ai/auth/google/callback`
- WeChat: `https://backend.tiris.ai/auth/wechat/callback`

**Development OAuth Callbacks:**
- Google: `https://backend.dev.tiris.ai/auth/google/callback`
- WeChat: `https://backend.dev.tiris.ai/auth/wechat/callback`

### 12.3 CORS Configuration
- Production allowed origins: `https://backend.tiris.ai`
- Development allowed origins: `https://backend.dev.tiris.ai, http://localhost:3000`