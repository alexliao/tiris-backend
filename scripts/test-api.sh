#!/bin/bash

# Simple API Testing Script for Tiris Backend
# 
# This script performs basic API testing for manual use:
# 1. Show user profile
# 2. Add a Kraken exchange for the user (or get existing one if it already exists)
# 3. Add a sub-account to the exchange (or get existing one if it already exists)
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
NC='\033[0m' # No Color

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
echo -e "${GREEN}✅ User profile retrieved${NC}"

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
        
        # Get specific exchange details
        echo ""
        echo "Getting exchange details:"
        KRAKEN_DETAILS=$(curl -s -H "$AUTH_HEADER" -H "$CONTENT_HEADER" "$BASE_URL/exchanges/$EXCHANGE_ID")
        echo "$KRAKEN_DETAILS" | jq . 2>/dev/null || echo "$KRAKEN_DETAILS"
    else
        echo "❌ No existing Kraken exchange found"
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
  "symbol": "USDT",
  "balance": 1000
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
        
        # Get specific sub-account details
        echo ""
        echo "Getting sub-account details:"
        SUB_ACCOUNT_DETAILS=$(curl -s -H "$AUTH_HEADER" -H "$CONTENT_HEADER" "$BASE_URL/sub-accounts/$SUB_ACCOUNT_ID")
        echo "$SUB_ACCOUNT_DETAILS" | jq . 2>/dev/null || echo "$SUB_ACCOUNT_DETAILS"
    else
        echo "❌ No existing sub-accounts found for this exchange"
        exit 1
    fi
fi


print_header "📋 Test Summary"
echo "✅ User profile test completed"
echo "✅ Kraken exchange test completed"
echo "✅ Sub-account test completed"
echo ""
echo "💡 Notes:"
echo "- If you get 401 Unauthorized, the JWT token may be expired"
echo "- Create a new test user to get a fresh JWT token: ./scripts/create-test-user.sh"
echo "- Check the API documentation at http://localhost:8080/docs for more details"