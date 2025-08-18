#!/bin/bash

# Simple API Testing Script for Tiris Backend
# 
# This script performs basic API testing for manual use:
# 1. Show user profile
# 2. Add a Kraken exchange for the user (or get existing one if it already exists)
# 3. Add a sub-account to the exchange (or get existing one if it already exists)
# 4. Update sub-account balance via dedicated balance API (starts with 0.0 by design)
# 5. Retrieve transaction records to show automatic audit trail
# 6. Add a trading log entry
# 
# IMPORTANT: This uses a JWT token for API authentication.
# To get a fresh JWT token:
# 1. Create a test user: ./scripts/create-test-user.sh --name "Your Name"  
# 2. Copy the JWT token from the output
# 3. Update the ACCESS_TOKEN variable below

# JWT Access Token for API Authentication
ACCESS_TOKEN=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiN2U2YTg5ZTYtM2U0Ni00MWQzLWFlYzYtMjg1ZmQ5Mjg5ODNiIiwidXNlcm5hbWUiOiJhbGV4IiwiZW1haWwiOiJhbGV4QHRpcmlzLmxvY2FsIiwicm9sZSI6InVzZXIiLCJpc3MiOiJ0aXJpcy1iYWNrZW5kIiwic3ViIjoiN2U2YTg5ZTYtM2U0Ni00MWQzLWFlYzYtMjg1ZmQ5Mjg5ODNiIiwiZXhwIjoxNzg2OTk5NjQyLCJuYmYiOjE3NTU0NjM2NDIsImlhdCI6MTc1NTQ2MzY0Mn0.T-c21MuwByZ8aeaahM45eECZt9FNwB5JC1OcS1glee8

# Base URL for API
BASE_URL="http://localhost:8080/v1"

# Common headers
AUTH_HEADER="Authorization: Bearer $ACCESS_TOKEN"
CONTENT_HEADER="Content-Type: application/json"

# Color codes for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Test status tracking
USER_PROFILE_SUCCESS=false
EXCHANGE_SUCCESS=false
SUBACCOUNT_SUCCESS=false
BALANCE_UPDATE_SUCCESS=false
TRANSACTION_HISTORY_SUCCESS=false
TRADING_LOG_SUCCESS=false

print_header() {
    echo ""
    echo "========================================" 
    echo -e "${BLUE}$1${NC}"
    echo "========================================"
}

# Test 1: Show User Profile
print_header "👤 Getting User Profile"
echo "Endpoint: GET /v1/users/me"
echo ""

USER_RESPONSE=$(curl -s -H "$AUTH_HEADER" -H "$CONTENT_HEADER" "$BASE_URL/users/me")
echo "$USER_RESPONSE" | jq . 2>/dev/null || echo "$USER_RESPONSE"

if echo "$USER_RESPONSE" | jq -e '.success == true and .data.id' > /dev/null 2>&1; then
    echo -e "${GREEN}✅ User profile retrieved successfully${NC}"
    USER_PROFILE_SUCCESS=true
else
    echo -e "${RED}❌ User profile retrieval failed${NC}"
    USER_PROFILE_SUCCESS=false
fi

# Test 2: Add Kraken Exchange or Get Kraken Exchange
print_header "🏦 Adding Kraken Exchange"
echo "Endpoint: POST /v1/exchanges"
echo ""

KRAKEN_PAYLOAD='{
  "name": "My Kraken Exchange",
  "type": "kraken",
  "api_key": "kraken_api_key_12345",
  "api_secret": "kraken_secret_67890"
}'

echo "Request payload:"
echo "$KRAKEN_PAYLOAD" | jq .
echo ""

echo "Response:"
KRAKEN_RESPONSE=$(curl -s -X POST \
  -H "$AUTH_HEADER" \
  -H "$CONTENT_HEADER" \
  -d "$KRAKEN_PAYLOAD" \
  "$BASE_URL/exchanges")

echo "$KRAKEN_RESPONSE" | jq . 2>/dev/null || echo "$KRAKEN_RESPONSE"

if echo "$KRAKEN_RESPONSE" | jq -e '.success == true and .data.id' > /dev/null 2>&1; then
    EXCHANGE_ID=$(echo "$KRAKEN_RESPONSE" | jq -r '.data.id')
    echo -e "${GREEN}✅ Kraken exchange created successfully${NC}"
    echo "Exchange ID: $EXCHANGE_ID"
    EXCHANGE_SUCCESS=true
else
    echo "❌ Kraken exchange creation failed, trying to get existing exchange"
    
    print_header "🔍 Getting Existing Kraken Exchange"
    echo "Endpoint: GET /v1/exchanges"
    echo ""
    
    EXCHANGES_RESPONSE=$(curl -s -H "$AUTH_HEADER" -H "$CONTENT_HEADER" "$BASE_URL/exchanges")
    echo "All exchanges response:"
    echo "$EXCHANGES_RESPONSE" | jq . 2>/dev/null || echo "$EXCHANGES_RESPONSE"
    echo ""
    
    # Extract the first Kraken exchange ID
    EXCHANGE_ID=$(echo "$EXCHANGES_RESPONSE" | jq -r '.data.exchanges[] | select(.type == "kraken") | .id' | head -1)
    
    if [ -n "$EXCHANGE_ID" ] && [ "$EXCHANGE_ID" != "null" ]; then
        echo -e "${GREEN}✅ Found existing Kraken exchange${NC}"
        echo "Exchange ID: $EXCHANGE_ID"
        EXCHANGE_SUCCESS=true
        
        # Get specific exchange details
        echo ""
        echo "Getting exchange details:"
        KRAKEN_DETAILS=$(curl -s -H "$AUTH_HEADER" -H "$CONTENT_HEADER" "$BASE_URL/exchanges/$EXCHANGE_ID")
        echo "$KRAKEN_DETAILS" | jq . 2>/dev/null || echo "$KRAKEN_DETAILS"
    else
        echo "❌ No existing Kraken exchange found"
        EXCHANGE_SUCCESS=false
        exit 1
    fi
fi

# Test 3: Add a sub-account to the Kraken exchange or get the first existing sub-account
print_header "👤 Adding Sub-Account to Kraken Exchange"
echo "Endpoint: POST /v1/sub-accounts"
echo ""

SUB_ACCOUNT_PAYLOAD='{
  "exchange_id": "'$EXCHANGE_ID'",
  "name": "Trade 1",
  "symbol": "USDT"
}'

echo "Request payload:"
echo "$SUB_ACCOUNT_PAYLOAD" | jq .
echo ""

echo "Response:"
SUB_ACCOUNT_RESPONSE=$(curl -s -X POST \
  -H "$AUTH_HEADER" \
  -H "$CONTENT_HEADER" \
  -d "$SUB_ACCOUNT_PAYLOAD" \
  "$BASE_URL/sub-accounts")

echo "$SUB_ACCOUNT_RESPONSE" | jq . 2>/dev/null || echo "$SUB_ACCOUNT_RESPONSE"

if echo "$SUB_ACCOUNT_RESPONSE" | jq -e '.success == true and .data.id' > /dev/null 2>&1; then
    SUB_ACCOUNT_ID=$(echo "$SUB_ACCOUNT_RESPONSE" | jq -r '.data.id')
    echo -e "${GREEN}✅ Sub-account created successfully${NC}"
    echo "Sub-Account ID: $SUB_ACCOUNT_ID"
    SUBACCOUNT_SUCCESS=true
else
    echo "❌ Sub-account creation failed, trying to get existing sub-accounts"
    
    print_header "🔍 Getting Existing Sub-Accounts"
    echo "Endpoint: GET /v1/sub-accounts"
    echo ""
    
    SUB_ACCOUNTS_RESPONSE=$(curl -s -H "$AUTH_HEADER" -H "$CONTENT_HEADER" "$BASE_URL/sub-accounts")
    echo "All sub-accounts response:"
    echo "$SUB_ACCOUNTS_RESPONSE" | jq . 2>/dev/null || echo "$SUB_ACCOUNTS_RESPONSE"
    echo ""
    
    # Extract the first sub-account ID for the current exchange
    SUB_ACCOUNT_ID=$(echo "$SUB_ACCOUNTS_RESPONSE" | jq -r --arg exchange_id "$EXCHANGE_ID" '.data.sub_accounts[] | select(.exchange_id == $exchange_id) | .id' | head -1)
    
    if [ -n "$SUB_ACCOUNT_ID" ] && [ "$SUB_ACCOUNT_ID" != "null" ]; then
        echo -e "${GREEN}✅ Found existing sub-account for this exchange${NC}"
        echo "Sub-Account ID: $SUB_ACCOUNT_ID"
        SUBACCOUNT_SUCCESS=true
        
        # Get specific sub-account details
        echo ""
        echo "Getting sub-account details:"
        SUB_ACCOUNT_DETAILS=$(curl -s -H "$AUTH_HEADER" -H "$CONTENT_HEADER" "$BASE_URL/sub-accounts/$SUB_ACCOUNT_ID")
        echo "$SUB_ACCOUNT_DETAILS" | jq . 2>/dev/null || echo "$SUB_ACCOUNT_DETAILS"
    else
        echo "❌ No existing sub-accounts found for this exchange"
        SUBACCOUNT_SUCCESS=false
        exit 1
    fi
fi

# Test 4: Update Sub-Account Balance
print_header "💰 Updating Sub-Account Balance"
echo "Endpoint: PUT /v1/sub-accounts/{id}/balance"
echo ""

BALANCE_UPDATE_PAYLOAD='{
  "amount": 1000,
  "direction": "credit",
  "reason": "initialization",
  "info": {
    "source": "test_script",
    "currency": "USDT",
    "test_purpose": "API demonstration"
  }
}'

echo "Request payload:"
echo "$BALANCE_UPDATE_PAYLOAD" | jq .
echo ""

echo "Response:"
BALANCE_UPDATE_RESPONSE=$(curl -s -X PUT \
  -H "$AUTH_HEADER" \
  -H "$CONTENT_HEADER" \
  -d "$BALANCE_UPDATE_PAYLOAD" \
  "$BASE_URL/sub-accounts/$SUB_ACCOUNT_ID/balance")

echo "$BALANCE_UPDATE_RESPONSE" | jq . 2>/dev/null || echo "$BALANCE_UPDATE_RESPONSE"

if echo "$BALANCE_UPDATE_RESPONSE" | jq -e '.success == true and .data.balance' > /dev/null 2>&1; then
    UPDATED_BALANCE=$(echo "$BALANCE_UPDATE_RESPONSE" | jq -r '.data.balance')
    echo -e "${GREEN}✅ Balance updated successfully${NC}"
    echo "Updated Balance: $UPDATED_BALANCE"
    BALANCE_UPDATE_SUCCESS=true
else
    echo "❌ Balance update failed"
    echo "Skipping transaction history retrieval due to balance update failure"
    BALANCE_UPDATE_SUCCESS=false
fi

# Test 5: Retrieve Transaction Records
if echo "$BALANCE_UPDATE_RESPONSE" | jq -e '.success == true' > /dev/null 2>&1; then
    print_header "📊 Retrieving Transaction Records"
    echo "Endpoint: GET /v1/transactions/sub-account/{sub_account_id}"
    echo ""
    
    TRANSACTIONS_RESPONSE=$(curl -s -H "$AUTH_HEADER" -H "$CONTENT_HEADER" "$BASE_URL/transactions/sub-account/$SUB_ACCOUNT_ID?limit=5&offset=0")
    echo "Recent transactions for sub-account:"
    echo "$TRANSACTIONS_RESPONSE" | jq . 2>/dev/null || echo "$TRANSACTIONS_RESPONSE"
    
    if echo "$TRANSACTIONS_RESPONSE" | jq -e '.success == true and .data.transactions' > /dev/null 2>&1; then
        TRANSACTION_COUNT=$(echo "$TRANSACTIONS_RESPONSE" | jq -r '.data.transactions | length')
        echo -e "${GREEN}✅ Retrieved $TRANSACTION_COUNT transaction record(s)${NC}"
        TRANSACTION_HISTORY_SUCCESS=true
        
        # Show details of the most recent transaction
        if [ "$TRANSACTION_COUNT" -gt 0 ]; then
            echo ""
            echo "Most recent transaction details:"
            LATEST_TRANSACTION=$(echo "$TRANSACTIONS_RESPONSE" | jq -r '.data.transactions[0]')
            echo "$LATEST_TRANSACTION" | jq .
        fi
    else
        echo "❌ Failed to retrieve transaction records"
        TRANSACTION_HISTORY_SUCCESS=false
    fi
else
    echo "❌ Balance update failed - skipping transaction history"
    TRANSACTION_HISTORY_SUCCESS=false
fi

# Test 6: Add a trading log
print_header "📊 Adding Trading Log for ETH Buy Order"
echo "Endpoint: POST /v1/trading-logs"
echo ""

TRADING_LOG_PAYLOAD='{ 
  "exchange_id": "'$EXCHANGE_ID'",
  "sub_account_id": "'$SUB_ACCOUNT_ID'",
  "type": "buy",
  "source": "bot", 
  "message": "Predicted as long - ETH buy order: 2.0 @ $3000 (fee: $12)",
  "info": {
    "symbol": "ETH",
    "amount": 2.0,
    "price": 3000,
    "fee": 12,
    "status": "completed", 
    "created_at": "2025-01-01T00:00:00Z",
    "order_type": "buy",
    "total_cost": 6012,
    "currency": "USD"
  }
}'

echo "Request payload:"
echo "$TRADING_LOG_PAYLOAD" | jq .
echo ""

echo "Response:"
TRADING_LOG_RESPONSE=$(curl -s -X POST \
  -H "$AUTH_HEADER" \
  -H "$CONTENT_HEADER" \
  -d "$TRADING_LOG_PAYLOAD" \
  "$BASE_URL/trading-logs")

echo "$TRADING_LOG_RESPONSE" | jq . 2>/dev/null || echo "$TRADING_LOG_RESPONSE"

if echo "$TRADING_LOG_RESPONSE" | jq -e '.success == true and .data.id' > /dev/null 2>&1; then
    TRADING_LOG_ID=$(echo "$TRADING_LOG_RESPONSE" | jq -r '.data.id')
    echo -e "${GREEN}✅ Trading log created successfully${NC}"
    echo "Trading Log ID: $TRADING_LOG_ID"
    TRADING_LOG_SUCCESS=true
else
    echo "❌ Trading log creation failed, trying to get existing trading logs"
    
    print_header "🔍 Getting Existing Trading Logs"
    echo "Endpoint: GET /v1/trading-logs"
    echo ""
    
    TRADING_LOGS_RESPONSE=$(curl -s -H "$AUTH_HEADER" -H "$CONTENT_HEADER" "$BASE_URL/trading-logs")
    echo "All trading logs response:"
    echo "$TRADING_LOGS_RESPONSE" | jq . 2>/dev/null || echo "$TRADING_LOGS_RESPONSE"
    echo ""
    
    # Extract the first trading log ID
    TRADING_LOG_ID=$(echo "$TRADING_LOGS_RESPONSE" | jq -r '.data.trading_logs[0].id // empty' | head -1)
    
    if [ -n "$TRADING_LOG_ID" ] && [ "$TRADING_LOG_ID" != "null" ]; then
        echo -e "${GREEN}✅ Found existing trading log${NC}"
        echo "Trading Log ID: $TRADING_LOG_ID"
        TRADING_LOG_SUCCESS=true
        
        # Get specific trading log details
        echo ""
        echo "Getting trading log details:"
        TRADING_LOG_DETAILS=$(curl -s -H "$AUTH_HEADER" -H "$CONTENT_HEADER" "$BASE_URL/trading-logs/$TRADING_LOG_ID")
        echo "$TRADING_LOG_DETAILS" | jq . 2>/dev/null || echo "$TRADING_LOG_DETAILS"
    else
        echo "❌ No existing trading logs found"
        TRADING_LOG_SUCCESS=false
        exit 1
    fi
fi

print_header "📋 Test Summary"

# Show status for each test based on success tracking
if [ "$USER_PROFILE_SUCCESS" = true ]; then
    echo "✅ User profile test completed successfully"
else
    echo "❌ User profile test failed"
fi

if [ "$EXCHANGE_SUCCESS" = true ]; then
    echo "✅ Kraken exchange test completed successfully"
else
    echo "❌ Kraken exchange test failed"
fi

if [ "$SUBACCOUNT_SUCCESS" = true ]; then
    echo "✅ Sub-account test completed successfully"
else
    echo "❌ Sub-account test failed"
fi

if [ "$BALANCE_UPDATE_SUCCESS" = true ]; then
    echo "✅ Balance update test completed successfully"
else
    echo "❌ Balance update test failed"
fi

if [ "$TRANSACTION_HISTORY_SUCCESS" = true ]; then
    echo "✅ Transaction history test completed successfully"
else
    echo "❌ Transaction history test failed"
fi

if [ "$TRADING_LOG_SUCCESS" = true ]; then
    echo "✅ Trading log test completed successfully"
else
    echo "❌ Trading log test failed"
fi

echo ""

# Overall test result summary
if [ "$USER_PROFILE_SUCCESS" = true ] && [ "$EXCHANGE_SUCCESS" = true ] && [ "$SUBACCOUNT_SUCCESS" = true ] && [ "$BALANCE_UPDATE_SUCCESS" = true ] && [ "$TRANSACTION_HISTORY_SUCCESS" = true ] && [ "$TRADING_LOG_SUCCESS" = true ]; then
    echo -e "${GREEN}🎉 All tests completed successfully!${NC}"
else
    echo -e "${RED}⚠️ Some tests failed. Check the output above for details.${NC}"
fi

echo ""
echo "💡 Notes:"
echo "- If you get 401 Unauthorized, the JWT token may be expired"
echo "- Create a new test user to get a fresh JWT token: ./scripts/create-test-user.sh"
echo "- Check the API documentation at http://localhost:8080/docs for more details"

# # Exit with appropriate code
# if [ "$USER_PROFILE_SUCCESS" = true ] && [ "$EXCHANGE_SUCCESS" = true ] && [ "$SUBACCOUNT_SUCCESS" = true ] && [ "$BALANCE_UPDATE_SUCCESS" = true ] && [ "$TRANSACTION_HISTORY_SUCCESS" = true ] && [ "$TRADING_LOG_SUCCESS" = true ]; then
#     exit 0
# else
#     exit 1
# fi