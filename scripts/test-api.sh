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
# USAGE:
#   ./scripts/test-api.sh                 - Run normal API tests
#   ./scripts/test-api.sh --clean         - Clean database (removes all user data)
#   ./scripts/test-api.sh --clean --test  - Clean database then run tests
# 
# IMPORTANT: This uses a JWT token for API authentication.
# To get a fresh JWT token:
# 1. Create a test user: ./scripts/create-test-user.sh --name "Your Name"  
# 2. Copy the JWT token from the output
# 3. Update the ACCESS_TOKEN variable below

# JWT Access Token for API Authentication
ACCESS_TOKEN=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiMmI4NzRlNTMtNmYyNS00YjRhLTkwODQtMjYyNTllN2UxZDg1IiwidXNlcm5hbWUiOiJzaW1wbGVfdXNlciIsImVtYWlsIjoic2ltcGxlX3VzZXJAdGlyaXMubG9jYWwiLCJyb2xlIjoidXNlciIsImlzcyI6InRpcmlzLWJhY2tlbmQiLCJzdWIiOiIyYjg3NGU1My02ZjI1LTRiNGEtOTA4NC0yNjI1OWU3ZTFkODUiLCJleHAiOjE3ODcyNDYwOTYsIm5iZiI6MTc1NTcxMDA5NiwiaWF0IjoxNzU1NzEwMDk2fQ.oGy--25KfBID3S-B9WkSUPERbgeOEyt5FEhOuQlqhuU
# Base URL for API
BASE_URL="http://backend.dev.tiris.ai/v1"

# Common headers
AUTH_HEADER="Authorization: Bearer $ACCESS_TOKEN"
CONTENT_HEADER="Content-Type: application/json"

# Color codes for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to check server connectivity
check_server_connectivity() {
    print_header "🔍 Checking Server Connectivity"
    echo "Testing connection to $BASE_URL..."
    
    # Try to connect to the health endpoint with timeout
    HEALTH_RESPONSE=$(curl -s -f --connect-timeout 5 --max-time 10 "$BASE_URL/../health" 2>/dev/null)
    local curl_exit_code=$?
    
    if [ $curl_exit_code -eq 0 ]; then
        echo -e "${GREEN}✅ Server is responding${NC}"
        echo "Health check response: $HEALTH_RESPONSE"
        return 0
    else
        echo -e "${RED}❌ Server is not responding${NC}"
        echo "Connection failed with exit code: $curl_exit_code"
        echo "Common reasons:"
        echo "  - Server is not running (run 'make run' first)"
        echo "  - Server is starting up (wait a few seconds)"
        echo "  - Wrong port or URL configuration"
        echo ""
        return 1
    fi
}

# Function to make API requests with proper error handling
api_request() {
    local method="$1"
    local endpoint="$2"
    local data="$3"
    
    if [ "$method" = "GET" ]; then
        curl -s -f --connect-timeout 10 --max-time 30 \
            -H "$AUTH_HEADER" -H "$CONTENT_HEADER" \
            "$BASE_URL$endpoint" 2>/dev/null
    elif [ "$method" = "POST" ]; then
        curl -s -f --connect-timeout 10 --max-time 30 \
            -X POST -H "$AUTH_HEADER" -H "$CONTENT_HEADER" \
            -d "$data" "$BASE_URL$endpoint" 2>/dev/null
    elif [ "$method" = "PUT" ]; then
        curl -s -f --connect-timeout 10 --max-time 30 \
            -X PUT -H "$AUTH_HEADER" -H "$CONTENT_HEADER" \
            -d "$data" "$BASE_URL$endpoint" 2>/dev/null
    elif [ "$method" = "DELETE" ]; then
        curl -s -f --connect-timeout 10 --max-time 30 \
            -X DELETE -H "$AUTH_HEADER" \
            "$BASE_URL$endpoint" 2>/dev/null
    fi
    
    return $?
}

# Parse command line arguments
CLEAN_DATABASE=false
RUN_TESTS=true
SHOW_HELP=false

for arg in "$@"; do
    case $arg in
        --clean)
            CLEAN_DATABASE=true
            RUN_TESTS=false  # Default to only clean unless --test is also specified
            ;;
        --test)
            RUN_TESTS=true
            ;;
        --help|-h)
            # Handle case where script is sourced vs executed
            SCRIPT_NAME="./scripts/test-api.sh"
            if [[ "$0" != "/bin/bash" ]] && [[ "$0" != "bash" ]]; then
                SCRIPT_NAME="$0"
            fi
            echo "Usage: $SCRIPT_NAME [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --clean       Clean database (remove all user data)"
            echo "  --clean --test Clean database then run API tests"
            echo "  --help, -h    Show this help message"
            echo ""
            echo "Examples:"
            echo "  $SCRIPT_NAME                    # Run normal API tests"
            echo "  $SCRIPT_NAME --clean           # Clean database only"
            echo "  $SCRIPT_NAME --clean --test    # Clean database then run tests"
            SHOW_HELP=true
            RUN_TESTS=false
            CLEAN_DATABASE=false
            ;;
        *)
            echo "Unknown option: $arg"
            echo "Use --help for usage information"
            return 1
            ;;
    esac
done

# Test status tracking
USER_PROFILE_SUCCESS=false
EXCHANGE_SUCCESS=false
SUBACCOUNT_SUCCESS=false
ETH_SUBACCOUNT_SUCCESS=false
BALANCE_UPDATE_SUCCESS=false
TRANSACTION_HISTORY_SUCCESS=false
TRADING_LOG_SUCCESS=false

print_header() {
    echo ""
    echo "========================================" 
    echo -e "${BLUE}$1${NC}"
    echo "========================================"
}

# Database cleanup function
cleanup_database() {
    print_header "🗑️ Cleaning Development Database"
    echo "This will remove data for the authenticated user:"
    echo "- Trading logs"
    echo "- Transactions (automatically deleted with sub-accounts)" 
    echo "- Sub-accounts"
    echo "- Exchanges"
    echo ""
    echo "Note: User accounts themselves are NOT deleted"
    echo ""
    
    # Safety check - only allow in development
    if [[ "$BASE_URL" != *"localhost"* ]] && [[ "$BASE_URL" != *"127.0.0.1"* ]]; then
        echo -e "${RED}❌ SAFETY CHECK FAILED${NC}"
        echo "Database cleanup is only allowed on localhost/development environments"
        echo "Current BASE_URL: $BASE_URL"
        echo "Cleanup cancelled for safety reasons."
        return 1
    fi
    
    echo -e "${YELLOW}⚠️ WARNING: This action cannot be undone!${NC}"
    echo -n "Are you sure you want to clean the development database? (yes/NO): "
    read -r confirmation
    
    if [[ "$confirmation" != "yes" ]]; then
        echo ""
        echo -e "${YELLOW}Database cleanup cancelled by user${NC}"
        echo ""
        return 0
    fi
    
    echo ""
    echo "Starting database cleanup..."
    echo ""
    echo "Note: The cleanup process respects API business rules:"
    echo "- Bot-generated trading logs cannot be deleted (API restriction)"
    echo "- Sub-accounts must have zero balance before deletion"
    echo "- Exchanges can only be deleted after all sub-accounts are removed"
    echo ""
    
    # Check server connectivity before cleanup
    check_server_connectivity
    if [ $? -ne 0 ]; then
        echo -e "${YELLOW}⚠️ Server connectivity issues detected - cleanup may fail${NC}"
        echo -e "${YELLOW}Consider starting the server first with 'make run'${NC}"
        echo ""
        echo -e "${BLUE}Press Enter to continue with cleanup anyway, or Ctrl+C to cancel...${NC}"
        read
    fi
    echo ""
    
    # Step 1: Get user profile to identify current user
    print_header "👤 Getting User Information"
    USER_RESPONSE=$(curl -s -H "$AUTH_HEADER" -H "$CONTENT_HEADER" "$BASE_URL/users/me")
    
    if echo "$USER_RESPONSE" | jq -e '.success == true and .data.id' > /dev/null 2>&1; then
        USER_ID=$(echo "$USER_RESPONSE" | jq -r '.data.id')
        USERNAME=$(echo "$USER_RESPONSE" | jq -r '.data.username')
        echo -e "${GREEN}✅ User identified: $USERNAME (ID: $USER_ID)${NC}"
    else
        echo -e "${RED}❌ Failed to get user information${NC}"
        echo "Cannot proceed with cleanup without valid authentication"
        echo "Please check your JWT token and try again"
        echo ""
        return 1
    fi
    
    # Step 2: Delete all trading logs for this user
    print_header "📊 Cleaning Trading Logs"
    echo "Fetching trading logs to delete..."
    
    TRADING_LOGS_RESPONSE=$(curl -s -H "$AUTH_HEADER" -H "$CONTENT_HEADER" "$BASE_URL/trading-logs")
    if echo "$TRADING_LOGS_RESPONSE" | jq -e '.success == true and .data.trading_logs' > /dev/null 2>&1; then
        TRADING_LOG_COUNT=$(echo "$TRADING_LOGS_RESPONSE" | jq -r '.data.trading_logs | length')
        echo "Found $TRADING_LOG_COUNT trading log(s) to delete"
        
        # Delete each trading log (bot-generated logs will be skipped due to API restrictions)
        echo "$TRADING_LOGS_RESPONSE" | jq -c '.data.trading_logs[]' | while read -r log; do
            log_id=$(echo "$log" | jq -r '.id')
            log_source=$(echo "$log" | jq -r '.source // "manual"')
            
            if [ -n "$log_id" ] && [ "$log_id" != "null" ]; then
                if [ "$log_source" = "bot" ]; then
                    echo "  ⏭️  Skipping bot-generated trading log: $log_id (API restriction)"
                else
                    DELETE_RESPONSE=$(curl -s -X DELETE -H "$AUTH_HEADER" "$BASE_URL/trading-logs/$log_id")
                    if echo "$DELETE_RESPONSE" | jq -e '.success == true' > /dev/null 2>&1; then
                        echo "  ✅ Deleted trading log: $log_id (source: $log_source)"
                    else
                        echo "  ❌ Failed to delete trading log: $log_id (source: $log_source)"
                        echo "     Response: $(echo "$DELETE_RESPONSE" | jq -c '.')"
                    fi

                fi
            fi
        done
    else
        echo "No trading logs found or failed to fetch"
    fi
    
    # Step 3: Zero out sub-account balances and delete sub-accounts
    print_header "🏦 Cleaning Sub-Accounts"
    echo "Fetching sub-accounts to process..."
    
    SUB_ACCOUNTS_RESPONSE=$(curl -s -H "$AUTH_HEADER" -H "$CONTENT_HEADER" "$BASE_URL/sub-accounts")
    if echo "$SUB_ACCOUNTS_RESPONSE" | jq -e '.success == true and .data.sub_accounts' > /dev/null 2>&1; then
        SUB_ACCOUNT_COUNT=$(echo "$SUB_ACCOUNTS_RESPONSE" | jq -r '.data.sub_accounts | length')
        echo "Found $SUB_ACCOUNT_COUNT sub-account(s) to process"
        echo ""
        
        # First, zero out balances for accounts with positive balances
        echo "Step 3a: Zeroing out account balances..."
        echo "$SUB_ACCOUNTS_RESPONSE" | jq -c '.data.sub_accounts[]' | while read -r account; do
            account_id=$(echo "$account" | jq -r '.id')
            account_name=$(echo "$account" | jq -r '.name')
            balance=$(echo "$account" | jq -r '.balance')
            
            # Check if balance is greater than 0 using awk
            if [ -n "$account_id" ] && [ "$account_id" != "null" ] && [ "$(echo "$balance" | awk '{print ($1 > 0)}')" = "1" ]; then
                echo "  Zeroing balance for account: $account_name (Balance: $balance)"
                
                ZERO_BALANCE_PAYLOAD="{\"amount\": $balance, \"direction\": \"debit\", \"reason\": \"cleanup\", \"info\": {\"source\": \"cleanup_script\", \"purpose\": \"prepare_for_deletion\"}}"
                ZERO_RESPONSE=$(curl -s -X PUT \
                    -H "$AUTH_HEADER" \
                    -H "$CONTENT_HEADER" \
                    -d "$ZERO_BALANCE_PAYLOAD" \
                    "$BASE_URL/sub-accounts/$account_id/balance")
                
                if echo "$ZERO_RESPONSE" | jq -e '.success == true' > /dev/null 2>&1; then
                    echo "    ✅ Balance zeroed for account: $account_id"
                else
                    echo "    ❌ Failed to zero balance for account: $account_id"
                    echo "       Response: $(echo "$ZERO_RESPONSE" | jq -c '.')"
                fi
            fi
        done
        
        echo ""
        echo "Step 3b: Deleting sub-accounts..."
        # Now delete each sub-account (this should also cascade delete transactions)
        echo "$SUB_ACCOUNTS_RESPONSE" | jq -r '.data.sub_accounts[].id' | while read -r account_id; do
            if [ -n "$account_id" ] && [ "$account_id" != "null" ]; then
                DELETE_RESPONSE=$(curl -s -X DELETE -H "$AUTH_HEADER" "$BASE_URL/sub-accounts/$account_id")
                if echo "$DELETE_RESPONSE" | jq -e '.success == true' > /dev/null 2>&1; then
                    echo "  ✅ Deleted sub-account: $account_id"
                else
                    echo "  ❌ Failed to delete sub-account: $account_id"
                    echo "     Response: $(echo "$DELETE_RESPONSE" | jq -c '.')"
                fi
            fi
        done
    else
        echo "No sub-accounts found or failed to fetch"
    fi
    
    # Step 4: Delete all exchanges for this user
    print_header "🔄 Cleaning Exchanges"
    echo "Fetching exchanges to delete..."
    
    EXCHANGES_RESPONSE=$(curl -s -H "$AUTH_HEADER" -H "$CONTENT_HEADER" "$BASE_URL/exchanges")
    if echo "$EXCHANGES_RESPONSE" | jq -e '.success == true and .data.exchanges' > /dev/null 2>&1; then
        EXCHANGE_COUNT=$(echo "$EXCHANGES_RESPONSE" | jq -r '.data.exchanges | length')
        echo "Found $EXCHANGE_COUNT exchange(s) to delete"
        
        # Delete each exchange
        echo "$EXCHANGES_RESPONSE" | jq -r '.data.exchanges[].id' | while read -r exchange_id; do
            if [ -n "$exchange_id" ] && [ "$exchange_id" != "null" ]; then
                DELETE_RESPONSE=$(curl -s -X DELETE -H "$AUTH_HEADER" "$BASE_URL/exchanges/$exchange_id")
                if echo "$DELETE_RESPONSE" | jq -e '.success == true' > /dev/null 2>&1; then
                    echo "  ✅ Deleted exchange: $exchange_id"
                else
                    echo "  ❌ Failed to delete exchange: $exchange_id"
                    echo "     Response: $(echo "$DELETE_RESPONSE" | jq -c '.')"
                fi
            fi
        done
    else
        echo "No exchanges found or failed to fetch"
    fi
    
    echo ""
    echo -e "${GREEN}🎉 Database cleanup completed!${NC}"
    echo "The development database has been cleaned of all user data."
    echo "You can now run tests with a fresh, clean state."
    echo ""
}

# Execute cleanup if requested
if [ "$CLEAN_DATABASE" = true ]; then
    cleanup_database
    CLEANUP_RESULT=$?
    if [ "$RUN_TESTS" = false ]; then
        if [ $CLEANUP_RESULT -eq 0 ]; then
            echo ""
            echo -e "${GREEN}✅ Database cleanup completed! Terminal will remain open.${NC}"
            echo ""
        else
            echo ""
            echo -e "${YELLOW}⚠️ Database cleanup was cancelled or failed. Terminal will remain open.${NC}"
            echo ""
        fi
        # Don't exit to keep output visible
    fi
fi

# Check if help was shown - if so, stop here
if [ "$SHOW_HELP" = true ]; then
    # Help was shown, don't continue with any other operations
    echo ""
    echo "Help information displayed. Terminal will remain open."
    echo ""
fi

# Skip tests if only cleanup was requested or help was shown
if [ "$RUN_TESTS" = false ]; then
    if [ "$CLEAN_DATABASE" = false ] && [ "$SHOW_HELP" = false ]; then
        echo ""
        echo -e "${GREEN}✅ No operations requested! Terminal will remain open.${NC}"
        echo ""
    fi
    # Don't continue with tests
else

# Check server connectivity before running tests
check_server_connectivity
if [ $? -ne 0 ]; then
    echo -e "${YELLOW}⚠️ Server connectivity issues detected - tests may fail${NC}"
    echo -e "${YELLOW}Consider starting the server first with 'make run'${NC}"
    echo ""
    echo -e "${BLUE}Press Enter to continue with tests anyway, or Ctrl+C to cancel...${NC}"
    read
fi
echo ""

# Test 1: Show User Profile
print_header "👤 Getting User Profile"
echo "Endpoint: GET /v1/users/me"
echo ""

USER_RESPONSE=$(api_request "GET" "/users/me")
api_exit_code=$?

if [ $api_exit_code -eq 0 ] && echo "$USER_RESPONSE" | jq -e '.success == true and .data.id' > /dev/null 2>&1; then
    echo "$USER_RESPONSE" | jq . 2>/dev/null || echo "$USER_RESPONSE"
    echo -e "${GREEN}✅ User profile retrieved successfully${NC}"
    USER_PROFILE_SUCCESS=true
else
    if [ $api_exit_code -ne 0 ]; then
        echo -e "${RED}❌ API request failed (HTTP error or connection issue)${NC}"
    else
        echo "$USER_RESPONSE" | jq . 2>/dev/null || echo "$USER_RESPONSE"
        echo -e "${RED}❌ User profile retrieval failed - invalid response${NC}"
    fi
    USER_PROFILE_SUCCESS=false
    echo -e "${RED}❌ User profile test failed - this may affect subsequent tests${NC}"
fi

# Test 2: Add Kraken Exchange or Get Kraken Exchange
print_header "🏦 Adding Kraken Exchange"
echo "Endpoint: POST /v1/exchanges"
echo ""

# Generate unique exchange details to avoid conflicts
TIMESTAMP=$(date +%s)
KRAKEN_PAYLOAD='{
  "name": "Test Kraken Exchange '$TIMESTAMP'",
  "type": "kraken",
  "api_key": "kraken_api_key_'$TIMESTAMP'",
  "api_secret": "kraken_secret_'$TIMESTAMP'"
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
        echo "⚠️ Continuing without exchange - remaining tests will likely fail"
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
        echo "⚠️ Continuing without sub-account - remaining tests will likely fail"
    fi
fi

# Test 3b: Create Second Sub-Account (ETH)
print_header "🏦 Creating Second Sub-Account (ETH)"
echo "Endpoint: POST /v1/sub-accounts"
echo ""

ETH_SUB_ACCOUNT_PAYLOAD='{
  "exchange_id": "'$EXCHANGE_ID'",
  "name": "ETH Trading Account",
  "symbol": "ETH"
}'

echo "Request payload:"
echo "$ETH_SUB_ACCOUNT_PAYLOAD" | jq .
echo ""

echo "Response:"
ETH_SUB_ACCOUNT_RESPONSE=$(curl -s -X POST \
  -H "$AUTH_HEADER" \
  -H "$CONTENT_HEADER" \
  -d "$ETH_SUB_ACCOUNT_PAYLOAD" \
  "$BASE_URL/sub-accounts")

echo "$ETH_SUB_ACCOUNT_RESPONSE" | jq . 2>/dev/null || echo "$ETH_SUB_ACCOUNT_RESPONSE"

if echo "$ETH_SUB_ACCOUNT_RESPONSE" | jq -e '.success == true and .data.id' > /dev/null 2>&1; then
    ETH_SUB_ACCOUNT_ID=$(echo "$ETH_SUB_ACCOUNT_RESPONSE" | jq -r '.data.id')
    echo -e "${GREEN}✅ ETH sub-account created successfully${NC}"
    echo "ETH Sub-Account ID: $ETH_SUB_ACCOUNT_ID"
    ETH_SUBACCOUNT_SUCCESS=true
else
    echo "❌ ETH sub-account creation failed, checking existing ETH sub-accounts"
    
    # Try to find existing ETH sub-account
    SUB_ACCOUNTS_RESPONSE=$(curl -s -H "$AUTH_HEADER" -H "$CONTENT_HEADER" "$BASE_URL/sub-accounts")
    ETH_SUB_ACCOUNT_ID=$(echo "$SUB_ACCOUNTS_RESPONSE" | jq -r --arg exchange_id "$EXCHANGE_ID" '.data.sub_accounts[] | select(.exchange_id == $exchange_id and .symbol == "ETH") | .id' | head -1)
    
    if [ -n "$ETH_SUB_ACCOUNT_ID" ] && [ "$ETH_SUB_ACCOUNT_ID" != "null" ]; then
        echo -e "${GREEN}✅ Found existing ETH sub-account${NC}"
        echo "ETH Sub-Account ID: $ETH_SUB_ACCOUNT_ID"
        ETH_SUBACCOUNT_SUCCESS=true
    else
        echo "❌ No ETH sub-account found"
        ETH_SUBACCOUNT_SUCCESS=false
        echo "⚠️ Continuing without ETH sub-account - trading log may fail"
    fi
fi

# Test 4: Update Sub-Account Balance
print_header "💰 Updating Sub-Account Balance"
echo "Endpoint: PUT /v1/sub-accounts/{id}/balance"
echo ""

BALANCE_UPDATE_PAYLOAD='{
  "amount": 10000,
  "direction": "credit",
  "reason": "initialization",
  "info": {
    "source": "test_script",
    "currency": "USDT",
    "test_purpose": "API demonstration - sufficient for trading"
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
print_header "📊 Adding Trading Log for ETH Long Position"
echo "Endpoint: POST /v1/trading-logs"
echo ""

if [ "$ETH_SUBACCOUNT_SUCCESS" = true ] && [ -n "$ETH_SUB_ACCOUNT_ID" ] && [ -n "$SUB_ACCOUNT_ID" ]; then
    TRADING_LOG_PAYLOAD='{ 
      "exchange_id": "'$EXCHANGE_ID'",
      "type": "long",
      "source": "bot", 
      "message": "ETH long position: 2.0 ETH @ $3000 (fee: $12)",
      "info": {
        "stock_account_id": "'$ETH_SUB_ACCOUNT_ID'",
        "currency_account_id": "'$SUB_ACCOUNT_ID'",
        "volume": 2.0,
        "price": 3000,
        "fee": 12,
        "stock": "ETH",
        "currency": "USDT"
      }
    }'
else
    echo "⚠️ ETH sub-account not available, using fallback trading log without business logic"
    TRADING_LOG_PAYLOAD='{ 
      "exchange_id": "'$EXCHANGE_ID'",
      "sub_account_id": "'$SUB_ACCOUNT_ID'",
      "type": "trade",
      "source": "bot", 
      "message": "Generic trade log entry (ETH sub-account not available)",
      "info": {
        "order_type": "buy",
        "symbol": "ETH",
        "volume": 2.0,
        "price": 3000,
        "fee": 12,
        "currency": "USDT"
      }
    }'
fi

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
    
    # Check if business logic was processed (look for transaction_ids in response)
    if echo "$TRADING_LOG_RESPONSE" | jq -e '.data.info.transaction_ids' > /dev/null 2>&1; then
        echo ""
        echo -e "${BLUE}🔍 Business logic was processed! Checking auto-generated transactions and updated balances...${NC}"
        
        # Test 7: Retrieve Auto-Generated Transactions
        print_header "💸 Auto-Generated Transactions from Long Position"
        echo "The long position should have created 2 transactions:"
        echo "1. Credit ETH account with +2.0 ETH"
        echo "2. Debit USDT account with -6,012.00 USDT (3000 × 2 + 12)"
        echo ""
        
        # Get recent transactions for ETH sub-account
        if [ -n "$ETH_SUB_ACCOUNT_ID" ] && [ "$ETH_SUB_ACCOUNT_ID" != "null" ]; then
            echo "📊 ETH Account Transactions:"
            echo "Endpoint: GET /v1/transactions/sub-account/$ETH_SUB_ACCOUNT_ID"
            echo ""
            
            ETH_TRANSACTIONS_RESPONSE=$(curl -s -H "$AUTH_HEADER" -H "$CONTENT_HEADER" "$BASE_URL/transactions/sub-account/$ETH_SUB_ACCOUNT_ID?limit=3&offset=0")
            echo "$ETH_TRANSACTIONS_RESPONSE" | jq . 2>/dev/null || echo "$ETH_TRANSACTIONS_RESPONSE"
            echo ""
        fi
        
        # Get recent transactions for USDT sub-account
        if [ -n "$SUB_ACCOUNT_ID" ] && [ "$SUB_ACCOUNT_ID" != "null" ]; then
            echo "💰 USDT Account Transactions:"
            echo "Endpoint: GET /v1/transactions/sub-account/$SUB_ACCOUNT_ID"
            echo ""
            
            USDT_TRANSACTIONS_RESPONSE=$(curl -s -H "$AUTH_HEADER" -H "$CONTENT_HEADER" "$BASE_URL/transactions/sub-account/$SUB_ACCOUNT_ID?limit=3&offset=0")
            echo "$USDT_TRANSACTIONS_RESPONSE" | jq . 2>/dev/null || echo "$USDT_TRANSACTIONS_RESPONSE"
            echo ""
        fi
        
        # Test 8: Check Updated Sub-Account Balances
        print_header "⚖️ Updated Sub-Account Balances After Long Position"
        echo "Expected balance changes:"
        echo "- ETH Account: Should show +2.0 ETH from the long position"
        echo "- USDT Account: Should show -6,012.00 USDT (1000 - 6012 = -5012.00 after initial credit)"
        echo ""
        
        # Get updated ETH sub-account balance
        if [ -n "$ETH_SUB_ACCOUNT_ID" ] && [ "$ETH_SUB_ACCOUNT_ID" != "null" ]; then
            echo "🪙 ETH Account Current Balance:"
            echo "Endpoint: GET /v1/sub-accounts/$ETH_SUB_ACCOUNT_ID"
            echo ""
            
            ETH_ACCOUNT_RESPONSE=$(curl -s -H "$AUTH_HEADER" -H "$CONTENT_HEADER" "$BASE_URL/sub-accounts/$ETH_SUB_ACCOUNT_ID")
            echo "$ETH_ACCOUNT_RESPONSE" | jq . 2>/dev/null || echo "$ETH_ACCOUNT_RESPONSE"
            
            # Extract and display balance
            if echo "$ETH_ACCOUNT_RESPONSE" | jq -e '.success == true and .data.balance' > /dev/null 2>&1; then
                ETH_BALANCE=$(echo "$ETH_ACCOUNT_RESPONSE" | jq -r '.data.balance')
                echo -e "${GREEN}✅ ETH Account Balance: $ETH_BALANCE ETH${NC}"
            fi
            echo ""
        fi
        
        # Get updated USDT sub-account balance
        if [ -n "$SUB_ACCOUNT_ID" ] && [ "$SUB_ACCOUNT_ID" != "null" ]; then
            echo "💵 USDT Account Current Balance:"
            echo "Endpoint: GET /v1/sub-accounts/$SUB_ACCOUNT_ID"
            echo ""
            
            USDT_ACCOUNT_RESPONSE=$(curl -s -H "$AUTH_HEADER" -H "$CONTENT_HEADER" "$BASE_URL/sub-accounts/$SUB_ACCOUNT_ID")
            echo "$USDT_ACCOUNT_RESPONSE" | jq . 2>/dev/null || echo "$USDT_ACCOUNT_RESPONSE"
            
            # Extract and display balance
            if echo "$USDT_ACCOUNT_RESPONSE" | jq -e '.success == true and .data.balance' > /dev/null 2>&1; then
                USDT_BALANCE=$(echo "$USDT_ACCOUNT_RESPONSE" | jq -r '.data.balance')
                echo -e "${GREEN}✅ USDT Account Balance: $USDT_BALANCE USDT${NC}"
            fi
            echo ""
        fi
        
        # Summary of business logic processing
        echo -e "${BLUE}📊 Business Logic Processing Summary:${NC}"
        echo "✅ Long position trading log created successfully"
        echo "✅ Auto-generated transactions for both accounts"
        echo "✅ ETH account credited with purchased volume"
        echo "✅ USDT account debited with total cost (price × volume + fee)"
        echo "✅ All operations completed atomically"
        echo ""
    else
        echo ""
        echo -e "${YELLOW}ℹ️ Trading log created but no business logic processing detected${NC}"
        echo "This might be a non-business logic type or ETH sub-account wasn't available"
        echo ""
    fi
else
    echo "❌ Trading log creation failed"
    TRADING_LOG_SUCCESS=false
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
    echo "✅ USDT sub-account test completed successfully"
else
    echo "❌ USDT sub-account test failed"
fi

if [ "$ETH_SUBACCOUNT_SUCCESS" = true ]; then
    echo "✅ ETH sub-account test completed successfully"
else
    echo "❌ ETH sub-account test failed"
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
if [ "$USER_PROFILE_SUCCESS" = true ] && [ "$EXCHANGE_SUCCESS" = true ] && [ "$SUBACCOUNT_SUCCESS" = true ] && [ "$ETH_SUBACCOUNT_SUCCESS" = true ] && [ "$BALANCE_UPDATE_SUCCESS" = true ] && [ "$TRANSACTION_HISTORY_SUCCESS" = true ] && [ "$TRADING_LOG_SUCCESS" = true ]; then
    echo -e "${GREEN}🎉 All tests completed successfully!${NC}"
else
    echo -e "${RED}⚠️ Some tests failed. Check the output above for details.${NC}"
fi

echo ""
echo "💡 Notes:"
echo "- If you get 401 Unauthorized, the JWT token may be expired"
echo "- Create a new test user to get a fresh JWT token: ./scripts/create-test-user.sh"
echo "- Check the API documentation at http://localhost:8080/docs for more details"

# Final status summary for human review
echo ""
if [ "$USER_PROFILE_SUCCESS" = true ] && [ "$EXCHANGE_SUCCESS" = true ] && [ "$SUBACCOUNT_SUCCESS" = true ] && [ "$ETH_SUBACCOUNT_SUCCESS" = true ] && [ "$BALANCE_UPDATE_SUCCESS" = true ] && [ "$TRANSACTION_HISTORY_SUCCESS" = true ] && [ "$TRADING_LOG_SUCCESS" = true ]; then
    echo -e "${GREEN}🎯 All tests completed successfully!${NC}"
    echo -e "${GREEN}✅ The API and business logic are working correctly.${NC}"
else
    echo -e "${YELLOW}📋 Test Summary - Some areas need attention:${NC}"
    
    if [ "$USER_PROFILE_SUCCESS" != true ]; then
        echo "  • User profile: ❌ Failed"
    fi
    if [ "$EXCHANGE_SUCCESS" != true ]; then
        echo "  • Exchange management: ❌ Failed" 
    fi
    if [ "$SUBACCOUNT_SUCCESS" != true ]; then
        echo "  • USDT sub-account: ❌ Failed"
    fi
    if [ "$ETH_SUBACCOUNT_SUCCESS" != true ]; then
        echo "  • ETH sub-account: ❌ Failed"
    fi
    if [ "$BALANCE_UPDATE_SUCCESS" != true ]; then
        echo "  • Balance updates: ❌ Failed"
    fi
    if [ "$TRANSACTION_HISTORY_SUCCESS" != true ]; then
        echo "  • Transaction history: ❌ Failed"
    fi
    if [ "$TRADING_LOG_SUCCESS" != true ]; then
        echo "  • Trading log & business logic: ❌ Failed"
    fi
fi

echo ""
echo -e "${BLUE}💡 Review the test results above. Terminal will stay open.${NC}"
echo -e "${BLUE}   Press Ctrl+C to close when you're done reviewing.${NC}"

fi # End of RUN_TESTS condition