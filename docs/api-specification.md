# Tiris Backend API Specification

## 1. API Overview

### 1.1 Base Information
- **Production URL**: `https://api.tiris.ai/v1`
- **Development URL**: `https://api.dev.tiris.ai/v1`
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
  "redirect_uri": "https://tiris.ai/callback"
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

## 4. Exchange Management API

### 4.1 List User Exchanges
**Endpoint:** `GET /exchanges`

**Description:** Get all exchanges bound to the current user.

**Headers:**
```
Authorization: Bearer {jwt_token}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "exchanges": [
      {
        "id": "exchange123",
        "name": "My Binance Account",
        "type": "binance",
        "status": "active",
        "created_at": "2024-01-01T00:00:00Z",
        "info": {
          "last_sync": "2024-01-15T10:00:00Z",
          "testnet": false,
          "permissions": ["spot", "futures"]
        }
      }
    ]
  }
}
```

### 4.2 Create Exchange Binding
**Endpoint:** `POST /exchanges`

**Description:** Bind a new exchange to the user account.

**Headers:**
```
Authorization: Bearer {jwt_token}
```

**Request Body:**
```json
{
  "name": "My Binance Account",
  "type": "binance",
  "api_key": "encrypted_api_key",
  "api_secret": "encrypted_api_secret",
  "info": {
    "testnet": false,
    "permissions": ["spot", "futures"]
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "exchange123",
    "name": "My Binance Account",
    "type": "binance",
    "status": "active",
    "created_at": "2024-01-15T10:30:00Z"
  }
}
```

### 4.3 Get Exchange Details
**Endpoint:** `GET /exchanges/{exchange_id}`

**Description:** Get details of a specific exchange.

**Headers:**
```
Authorization: Bearer {jwt_token}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "exchange123",
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

### 4.4 Update Exchange
**Endpoint:** `PUT /exchanges/{exchange_id}`

**Description:** Update exchange configuration.

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
    "id": "exchange123",
    "name": "Updated Binance Account",
    "type": "binance",
    "status": "active",
    "updated_at": "2024-01-15T10:35:00Z"
  }
}
```

### 4.5 Delete Exchange
**Endpoint:** `DELETE /exchanges/{exchange_id}`

**Description:** Remove exchange binding from user account.

**Headers:**
```
Authorization: Bearer {jwt_token}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Exchange removed successfully"
  }
}
```

## 5. Sub-account Management API

### 5.1 List Sub-accounts
**Endpoint:** `GET /sub-accounts`

**Description:** Get all sub-accounts for the current user.

**Query Parameters:**
- `exchange_id` (optional): Filter by exchange ID

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
        "exchange_id": "exchange123",
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

### 5.2 Create Sub-account
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
  "exchange_id": "exchange123",
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
    "exchange_id": "exchange123",
    "created_at": "2024-01-15T10:30:00Z"
  }
}
```

### 5.3 Get Sub-account Details
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
    "exchange_id": "exchange123",
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

### 5.4 Update Sub-account
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
    "exchange_id": "exchange123",
    "updated_at": "2024-01-15T10:35:00Z"
  }
}
```

### 5.5 Deposit to Sub-account
**Endpoint:** `POST /sub-accounts/{sub_account_id}/deposit`

**Description:** Deposit funds to a sub-account.

**Headers:**
```
Authorization: Bearer {jwt_token}
```

**Request Body:**
```json
{
  "amount": "0.1",
  "reason": "manual_deposit",
  "info": {
    "note": "Adding funds for new strategy"
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "transaction_id": "tx123",
    "amount": "0.1",
    "new_balance": "0.6",
    "timestamp": "2024-01-15T10:30:00Z"
  }
}
```

### 5.6 Withdraw from Sub-account
**Endpoint:** `POST /sub-accounts/{sub_account_id}/withdraw`

**Description:** Withdraw funds from a sub-account.

**Headers:**
```
Authorization: Bearer {jwt_token}
```

**Request Body:**
```json
{
  "amount": "0.05",
  "reason": "manual_withdrawal",
  "info": {
    "note": "Reducing exposure"
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "transaction_id": "tx124",
    "amount": "0.05",
    "new_balance": "0.55",
    "timestamp": "2024-01-15T10:30:00Z"
  }
}
```

### 5.7 Delete Sub-account
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

## 6. Transaction Management API

### 6.1 List Transactions
**Endpoint:** `GET /transactions`

**Description:** Get transaction history.

**Query Parameters:**
- `sub_account_id` (optional): Filter by sub-account
- `exchange_id` (optional): Filter by exchange
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
        "exchange_id": "exchange123",
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

### 6.2 Get Transaction Details
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
    "exchange_id": "exchange123",
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

## 7. Trading Log Management API

### 7.1 List Trading Logs
**Endpoint:** `GET /trading-logs`

**Description:** Get trading log history.

**Query Parameters:**
- `sub_account_id` (optional): Filter by sub-account
- `exchange_id` (optional): Filter by exchange
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
        "exchange_id": "exchange123",
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

**Headers:**
```
Authorization: Bearer {jwt_token}
```

**Request Body:**
```json
{
  "type": "buy_order",
  "source": "bot",
  "message": "Executed buy order for 0.001 BTC",
  "sub_account_id": "sub123",
  "exchange_id": "exchange123",
  "info": {
    "order_id": "order456",
    "symbol": "BTC/USD",
    "price": "45000.00",
    "quantity": "0.001",
    "transaction_data": {
      "direction": "debit",
      "reason": "long_entry",
      "amount": "45.00",
      "new_balance": "955.00"
    }
  }
}
```

**Note on Balance Updates:**
When `transaction_data` is included, the backend will:
1. Create a trading log entry
2. Automatically create a transaction record
3. Update the sub-account balance to `new_balance` value
4. The bot is responsible for calculating the correct new balance

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "log123",
    "timestamp": "2024-01-15T10:30:00Z",
    "type": "buy_order",
    "source": "manual",
    "message": "Manual buy order for 0.001 BTC",
    "transaction_id": "tx123",
    "sub_account_id": "sub123",
    "exchange_id": "exchange123"
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
    "exchange_id": "exchange123",
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
- `EXCHANGE_NAME_EXISTS`: Exchange name already exists for this user
- `API_KEY_EXISTS`: API key already exists for this user  
- `API_SECRET_EXISTS`: API secret already exists for this user
- `SUBACCOUNT_NAME_EXISTS`: Sub-account name already exists for this exchange
- `EMAIL_EXISTS`: Email address already exists (global uniqueness)

### 9.4 Resource Errors
- `NOT_FOUND`: Resource not found (404)
- `INSUFFICIENT_BALANCE`: Not enough balance (400)
- `EXCHANGE_ERROR`: Exchange API error (502)

### 9.5 System Errors
- `INTERNAL_ERROR`: Internal server error (500)
- `DATABASE_ERROR`: Database operation failed (500)
- `SERVICE_UNAVAILABLE`: Service temporarily unavailable (503)

## 10. Rate Limiting

### 10.1 Rate Limits
- **General API**: 1000 requests per hour per user
- **Authentication**: 10 requests per minute per IP
- **Trading Operations**: 100 requests per minute per user

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
- **Production API**: `https://api.tiris.ai/v1`
- **Development API**: `https://api.dev.tiris.ai/v1`
- **Frontend Portal**: `https://tiris.ai`
- **Development Portal**: `https://dev.tiris.ai`

### 12.2 Environment-Specific Endpoints

**Production OAuth Callbacks:**
- Google: `https://tiris.ai/auth/google/callback`
- WeChat: `https://tiris.ai/auth/wechat/callback`

**Development OAuth Callbacks:**
- Google: `https://dev.tiris.ai/auth/google/callback`
- WeChat: `https://dev.tiris.ai/auth/wechat/callback`

### 12.3 CORS Configuration
- Production allowed origins: `https://tiris.ai`
- Development allowed origins: `https://dev.tiris.ai, http://localhost:3000`