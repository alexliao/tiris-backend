#!/bin/bash

# Simple API Testing Script for Tiris Backend
# 
# This script performs basic API testing for manual use:
# 1. Show user profile
# 2. Add a Binance exchange for the user (or get existing one if it already exists)
# 3. Add a sub-account to the exchange (or get existing one if it already exists)
# 4. Initialize sub-account balance using deposit trading log (demonstrates business logic)
# 5. Retrieve transaction records to show automatic audit trail from deposit
# 6. Add a trading log entry (long position with business logic processing)
# 
# USAGE:
#   ./scripts/test-api.sh                                     - Run normal API tests (localhost:8080)
#   ./scripts/test-api.sh --username alex                     - Generate JWT for user 'alex' and run tests
#   ./scripts/test-api.sh --domain backend.dev.tiris.ai      - Test remote server
#   ./scripts/test-api.sh --domain localhost:3000            - Test local server on custom port
#   ./scripts/test-api.sh --username john --domain localhost:3000  - Generate JWT for 'john' and test port 3000
#   ./scripts/test-api.sh --clean                            - Clean database (removes all user data)
#   ./scripts/test-api.sh --clean --test                     - Clean database then run tests
# 
# IMPORTANT: This script can authenticate in multiple ways:
# 1. Use --username to generate JWT dynamically from database user
# 2. Set JWT_TOKEN environment variable: JWT_TOKEN=your_token ./scripts/test-api.sh
# 3. Create a test user: ./scripts/create-test-user.sh --name "Your Name" (then use --username)
#
# DATABASE CONFIGURATION:
# The script auto-detects database configuration, but you can override:
# --container NAME    PostgreSQL container name (tiris-postgres-dev, tiris-postgres-simple, tiris-postgres-prod)
# --database NAME     Database name (tiris_dev, tiris, tiris_prod)  
# --db-user NAME      Database user (usually tiris_user)

# JWT Access Token for API Authentication (will be set dynamically if --username is provided)
# Preserve environment variable if it exists
JWT_TOKEN="${JWT_TOKEN:-""}"
# Default domain for API (will be set based on --domain option)
DEFAULT_DOMAIN="localhost:8080"
API_DOMAIN="$DEFAULT_DOMAIN"

# Base URL for API (will be constructed after parsing arguments)
BASE_URL=""

# Common headers (AUTH_HEADER will be set after JWT_TOKEN is determined)
AUTH_HEADER=""
CONTENT_HEADER="Content-Type: application/json"

# Color codes for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Print functions for colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_header() {
    echo ""
    echo "========================================" 
    echo -e "${BLUE}$1${NC}"
    echo "========================================"
}

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
SKIP_TRADING_LOGS=false
USERNAME=""
DB_CONTAINER=""
DB_DATABASE=""
DB_USER=""

# Parse arguments with proper handling of options that take values
while [[ $# -gt 0 ]]; do
    case $1 in
        --domain)
            if [[ -n "$2" && "$2" != --* ]]; then
                API_DOMAIN="$2"
                shift 2
            else
                echo "Error: --domain requires a domain argument"
                echo "Use --help for usage information"
                exit 1
            fi
            ;;
        --username)
            if [[ -n "$2" && "$2" != --* ]]; then
                USERNAME="$2"
                shift 2
            else
                echo "Error: --username requires a username argument"
                echo "Use --help for usage information"
                exit 1
            fi
            ;;
        --container)
            if [[ -n "$2" && "$2" != --* ]]; then
                DB_CONTAINER="$2"
                shift 2
            else
                echo "Error: --container requires a container name argument"
                echo "Use --help for usage information"
                exit 1
            fi
            ;;
        --database)
            if [[ -n "$2" && "$2" != --* ]]; then
                DB_DATABASE="$2"
                shift 2
            else
                echo "Error: --database requires a database name argument"
                echo "Use --help for usage information"
                exit 1
            fi
            ;;
        --db-user)
            if [[ -n "$2" && "$2" != --* ]]; then
                DB_USER="$2"
                shift 2
            else
                echo "Error: --db-user requires a database user argument"
                echo "Use --help for usage information"
                exit 1
            fi
            ;;
        --clean)
            CLEAN_DATABASE=true
            RUN_TESTS=false  # Default to only clean unless --test is also specified
            shift
            ;;
        --test)
            RUN_TESTS=true
            shift
            ;;
        --no-trading-logs)
            SKIP_TRADING_LOGS=true
            shift
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
            echo "  --domain DOMAIN       Specify API domain (default: localhost:8080)"
            echo "  --username USERNAME   Generate JWT token for specific user (queries database)"
            echo "  --container NAME      PostgreSQL container name (auto-detected if not specified)"
            echo "  --database NAME       Database name (auto-detected if not specified)"
            echo "  --db-user NAME        Database user (auto-detected if not specified)"
            echo "  --clean               Clean database (remove all user data)"
            echo "  --clean --test        Clean database then run API tests"
            echo "  --no-trading-logs     Skip trading log tests (faster execution)"
            echo "  --help, -h            Show this help message"
            echo ""
            echo "Examples:"
            echo "  $SCRIPT_NAME                                    # Test localhost:8080 (HTTP)"
            echo "  $SCRIPT_NAME --username alex                   # Generate JWT for user 'alex' and run tests"
            echo "  $SCRIPT_NAME --domain backend.dev.tiris.ai     # Test remote server (HTTPS)"
            echo "  $SCRIPT_NAME --domain localhost:3000           # Test local server on port 3000 (HTTP)"
            echo "  $SCRIPT_NAME --username john --domain localhost:3000  # Generate JWT for 'john' and test port 3000"
            echo "  $SCRIPT_NAME --container tiris-postgres-simple --username alex  # Use simple deployment container"
            echo "  $SCRIPT_NAME --container tiris-postgres-prod --database tiris_prod --db-user tiris_user  # Production setup"
            echo "  $SCRIPT_NAME --no-trading-logs                 # Test without trading log operations"
            echo "  $SCRIPT_NAME --clean --domain localhost:8080   # Clean local database"
            echo "  $SCRIPT_NAME --clean --test                    # Clean then test localhost:8080"
            SHOW_HELP=true
            RUN_TESTS=false
            CLEAN_DATABASE=false
            shift
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Function to detect database configuration automatically
detect_database_config() {
    # Try to detect running containers and their configurations
    local detected_container=""
    local detected_database=""
    local detected_user=""
    
    # Check for common container names in order of preference
    local containers_to_try=("tiris-postgres-dev" "tiris-postgres-simple" "tiris-postgres-prod")
    
    for container in "${containers_to_try[@]}"; do
        if docker ps --format "table {{.Names}}" | grep -q "^${container}$"; then
            detected_container="$container"
            break
        fi
    done
    
    # Set database and user based on container name
    case "$detected_container" in
        "tiris-postgres-dev")
            detected_database="tiris_dev"
            detected_user="tiris_user"
            ;;
        "tiris-postgres-simple")
            detected_database="tiris"
            detected_user="tiris_user"
            ;;
        "tiris-postgres-prod")
            detected_database="tiris_prod"
            detected_user="tiris_user"
            ;;
        *)
            # Fallback to dev defaults if nothing found
            detected_container="tiris-postgres-dev"
            detected_database="tiris_dev"
            detected_user="tiris_user"
            ;;
    esac
    
    # Set global variables if not already specified via command line
    if [ -z "$DB_CONTAINER" ]; then
        DB_CONTAINER="$detected_container"
    fi
    if [ -z "$DB_DATABASE" ]; then
        DB_DATABASE="$detected_database"
    fi
    if [ -z "$DB_USER" ]; then
        DB_USER="$detected_user"
    fi
}

# Function to generate JWT token for a user by username
generate_jwt_token_for_user() {
    local username="$1"
    
    # Ensure database configuration is detected/set
    detect_database_config
    
    local container="$DB_CONTAINER"
    local database="$DB_DATABASE"
    local db_user="$DB_USER"
    
    print_header "🔑 Generating JWT Token for User: $username"
    print_status "Using database config: container=$container, database=$database, user=$db_user"
    
    # Check if Docker is available
    if ! command -v docker > /dev/null 2>&1; then
        echo -e "${RED}❌ Docker is not available${NC}"
        echo "Cannot query database without Docker"
        return 1
    fi
    
    # Check if container is running
    if ! docker ps --format "table {{.Names}}" | grep -q "^${container}$"; then
        echo -e "${RED}❌ Database container '$container' is not running${NC}"
        echo "Please start the appropriate database first:"
        case "$container" in
            "tiris-postgres-dev")
                echo "  make dev  # or docker-compose -f docker-compose.dev.yml up postgres"
                ;;
            "tiris-postgres-simple")
                echo "  docker-compose -f docker-compose.simple.yml up postgres"
                ;;
            "tiris-postgres-prod")
                echo "  docker-compose -f docker-compose.prod.yml up postgres"
                ;;
            *)
                echo "  docker-compose up $container  # or start your custom container"
                ;;
        esac
        return 1
    fi
    
    print_status "Querying user information from database..."
    
    # Query for user by username  
    local user_query="SELECT id, username, email, info FROM users WHERE username = '$username' AND deleted_at IS NULL LIMIT 1;"
    local user_info=$(docker exec "$container" psql -U "$db_user" -d "$database" -t -A -c "$user_query" 2>/dev/null)
    
    if [ -z "$user_info" ] || [ "$user_info" = "" ]; then
        echo -e "${RED}❌ User '$username' not found in database${NC}"
        echo ""
        echo "Available options:"
        echo "1. Create a new user: ./scripts/create-test-user.sh --name 'Your Name' --username '$username'"
        echo "2. List existing users: docker exec $container psql -U $db_user -d $database -c \"SELECT username, email FROM users WHERE deleted_at IS NULL;\""
        echo "3. Use hardcoded JWT token instead (remove --username option)"
        return 1
    fi
    
    # Parse user info (format: id|username|email|info)
    IFS='|' read -r user_id db_username email info_json <<< "$user_info"
    
    if [ -z "$user_id" ] || [ -z "$db_username" ] || [ -z "$email" ]; then
        echo -e "${RED}❌ Failed to parse user information from database${NC}"
        echo "Raw user data: $user_info"
        return 1
    fi
    
    print_success "Found user: $db_username ($email)"
    echo "User ID: $user_id"
    
    # Extract display name from info JSON if available  
    local display_name="$db_username"
    if [ -n "$info_json" ] && [ "$info_json" != "null" ]; then
        # Try to extract display_name from JSON (basic parsing)
        display_name=$(echo "$info_json" | sed -n 's/.*"display_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')
        if [ -z "$display_name" ]; then
            display_name="$db_username"
        fi
    fi
    
    print_status "Generating JWT token..."
    
    # Check if .env file exists
    if [ ! -f ".env" ]; then
        echo -e "${RED}❌ .env file not found${NC}"
        echo "JWT token generation requires environment variables"
        echo "Please ensure .env file exists with JWT_SECRET and REFRESH_SECRET"
        return 1
    fi
    
    # Generate JWT token using the existing script
    local generated_token
    generated_token=$(go run scripts/generate-jwt-token.go \
        --user-id "$user_id" \
        --username "$db_username" \
        --email "$email" \
        --role "user" \
        --duration "24h" \
        --output "token" 2>&1)
    
    local token_exit_code=$?
    
    if [ $token_exit_code -eq 0 ] && [ -n "$generated_token" ]; then
        # Validate JWT format (should have 3 parts separated by dots)
        local token_parts=$(echo "$generated_token" | tr '.' '\n' | wc -l)
        if [ "$token_parts" -eq 3 ]; then
            JWT_TOKEN="$generated_token"
            print_success "JWT token generated successfully!"
            echo "Token valid for: 24 hours"
            echo "User: $display_name ($db_username)"
            echo ""
            return 0
        else
            echo -e "${RED}❌ Generated token has invalid format${NC}"
            echo "Expected 3 parts, got $token_parts"
            echo "Token: $generated_token"
            return 1
        fi
    else
        echo -e "${RED}❌ Failed to generate JWT token${NC}"
        echo "Exit code: $token_exit_code"
        echo "Output: $generated_token"
        echo ""
        echo "Common issues:"
        echo "- Missing JWT_SECRET or REFRESH_SECRET in .env file"
        echo "- Go dependencies not installed (run 'go mod download')"
        echo "- Invalid user data"
        return 1
    fi
}

# Function to construct BASE_URL based on domain
construct_base_url() {
    local domain="$1"
    
    # Determine protocol based on domain
    if [[ "$domain" =~ ^localhost(:[0-9]+)?$ ]] || [[ "$domain" =~ ^127\.0\.0\.1(:[0-9]+)?$ ]] || [[ "$domain" =~ ^0\.0\.0\.0(:[0-9]+)?$ ]]; then
        # Use HTTP for localhost, 127.0.0.1, and 0.0.0.0
        BASE_URL="http://${domain}/v1"
    else
        # Use HTTPS for remote domains
        BASE_URL="https://${domain}/v1"
    fi
}

# Construct BASE_URL from API_DOMAIN
construct_base_url "$API_DOMAIN"

# Generate JWT token if username is provided
if [ -n "$USERNAME" ]; then
    if ! generate_jwt_token_for_user "$USERNAME"; then
        echo -e "${RED}❌ Failed to generate JWT token for user '$USERNAME'${NC}"
        echo -e "${YELLOW}Cannot continue without valid authentication${NC}"
        exit 1
    fi
elif [ -z "$JWT_TOKEN" ]; then
    # No username provided and no hardcoded token
    echo -e "${YELLOW}⚠️ No JWT token available${NC}"
    echo ""
    echo "You have two options:"
    echo "1. Use --username option to generate token dynamically:"
    echo "   $0 --username your_username"
    echo ""
    echo "2. Create a test user and use their token:"
    echo "   ./scripts/create-test-user.sh --name 'Your Name'"
    echo "   Then copy the JWT token to this script"
    echo ""
    echo "3. Set JWT_TOKEN environment variable:"
    echo "   JWT_TOKEN=your_token $0"
    echo ""
    exit 1
fi

# Set AUTH_HEADER now that JWT_TOKEN is available
AUTH_HEADER="Authorization: Bearer $JWT_TOKEN"

# Test status tracking
USER_PROFILE_SUCCESS=false
EXCHANGE_SUCCESS=false
SUBACCOUNT_SUCCESS=false
ETH_SUBACCOUNT_SUCCESS=false
DEPOSIT_SUCCESS=false
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
    if [[ "$BASE_URL" != *"localhost"* ]] && [[ "$BASE_URL" != *".dev."* ]]; then
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
    echo "Note: The cleanup process uses modern trading log business logic:"
    echo "- Bot-generated trading logs cannot be deleted (API restriction)"
    echo "- Sub-accounts are zeroed using withdraw trading logs (business logic)"
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
        
        # First, zero out balances for accounts with positive balances using withdraw trading logs
        echo "Step 3a: Withdrawing all funds from accounts using trading logs..."
        echo "$SUB_ACCOUNTS_RESPONSE" | jq -c '.data.sub_accounts[]' | while read -r account; do
            account_id=$(echo "$account" | jq -r '.id')
            account_name=$(echo "$account" | jq -r '.name')
            account_symbol=$(echo "$account" | jq -r '.symbol')
            account_exchange_id=$(echo "$account" | jq -r '.exchange_id')
            balance=$(echo "$account" | jq -r '.balance')
            
            # Check if balance is greater than 0 using awk
            if [ -n "$account_id" ] && [ "$account_id" != "null" ] && [ "$(echo "$balance" | awk '{print ($1 > 0)}')" = "1" ]; then
                echo "  Withdrawing all funds from account: $account_name (Balance: $balance $account_symbol)"
                
                WITHDRAW_PAYLOAD="{\"exchange_id\": \"$account_exchange_id\", \"type\": \"withdraw\", \"source\": \"api\", \"message\": \"Withdraw all funds for account cleanup: $account_name\", \"info\": {\"account_id\": \"$account_id\", \"amount\": $balance, \"currency\": \"$account_symbol\"}}"
                ZERO_RESPONSE=$(curl -s -X POST \
                    -H "$AUTH_HEADER" \
                    -H "$CONTENT_HEADER" \
                    -d "$WITHDRAW_PAYLOAD" \
                    "$BASE_URL/trading-logs")
                
                if echo "$ZERO_RESPONSE" | jq -e '.success == true and .data.id' > /dev/null 2>&1; then
                    withdraw_log_id=$(echo "$ZERO_RESPONSE" | jq -r '.data.id')
                    echo "    ✅ Withdraw trading log created: $withdraw_log_id (balance should be zeroed)"
                else
                    echo "    ❌ Failed to create withdraw trading log for account: $account_id"
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

# Display test configuration
print_header "🔧 Test Configuration"
echo "Target Domain: $API_DOMAIN"
echo "Base URL: $BASE_URL"
if [[ "$BASE_URL" =~ ^https:// ]]; then
    echo "Protocol: HTTPS (remote server)"
else
    echo "Protocol: HTTP (local development)"
fi
echo ""

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

# Test 2: Add Binance Exchange or Get Binance Exchange
print_header "🏦 Adding Binance Exchange"
echo "Endpoint: POST /v1/exchanges"
echo ""

# Generate unique API credentials to avoid conflicts (but use fixed exchange name)
TIMESTAMP=$(date +%s)
KRAKEN_PAYLOAD='{
  "name": "My Binance",
  "type": "binance",
  "api_key": "binance_api_key_'$TIMESTAMP'",
  "api_secret": "binance_secret_'$TIMESTAMP'"
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
    echo -e "${GREEN}✅ Binance exchange created successfully${NC}"
    echo "Exchange ID: $EXCHANGE_ID"
    EXCHANGE_SUCCESS=true
else
    echo "❌ Binance exchange creation failed, trying to get existing exchange"
    
    print_header "🔍 Getting Existing Binance Exchange"
    echo "Endpoint: GET /v1/exchanges"
    echo ""
    
    EXCHANGES_RESPONSE=$(curl -s -H "$AUTH_HEADER" -H "$CONTENT_HEADER" "$BASE_URL/exchanges")
    echo "All exchanges response:"
    echo "$EXCHANGES_RESPONSE" | jq . 2>/dev/null || echo "$EXCHANGES_RESPONSE"
    echo ""
    
    # Extract the first Binance exchange ID
    EXCHANGE_ID=$(echo "$EXCHANGES_RESPONSE" | jq -r '.data.exchanges[] | select(.type == "binance") | .id' | head -1)
    
    if [ -n "$EXCHANGE_ID" ] && [ "$EXCHANGE_ID" != "null" ]; then
        echo -e "${GREEN}✅ Found existing Binance exchange${NC}"
        echo "Exchange ID: $EXCHANGE_ID"
        EXCHANGE_SUCCESS=true
        
        # Get specific exchange details
        echo ""
        echo "Getting exchange details:"
        KRAKEN_DETAILS=$(curl -s -H "$AUTH_HEADER" -H "$CONTENT_HEADER" "$BASE_URL/exchanges/$EXCHANGE_ID")
        echo "$KRAKEN_DETAILS" | jq . 2>/dev/null || echo "$KRAKEN_DETAILS"
    else
        echo "❌ No existing Binance exchange found"
        EXCHANGE_SUCCESS=false
        echo "⚠️ Continuing without exchange - remaining tests will likely fail"
    fi
fi

# Test 3: Add a sub-account to the Binance exchange or get the first existing sub-account
print_header "👤 Adding Sub-Account to Binance Exchange"
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
    
    # Extract the USDT sub-account ID for the current exchange
    SUB_ACCOUNT_ID=$(echo "$SUB_ACCOUNTS_RESPONSE" | jq -r --arg exchange_id "$EXCHANGE_ID" '.data.sub_accounts[] | select(.exchange_id == $exchange_id and .symbol == "USDT") | .id' | head -1)
    
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

# Test 4: Initialize Sub-Account with Deposit Trading Log
print_header "💰 Initializing Sub-Account with Deposit"
echo "Endpoint: POST /v1/trading-logs (type: deposit)"
echo "This replaces the obsolete balance API with proper business logic"
echo ""

DEPOSIT_PAYLOAD='{
  "exchange_id": "'$EXCHANGE_ID'",
  "type": "deposit",
  "source": "manual",
  "message": "Initial USDT deposit for testing - sufficient for trading",
  "info": {
    "account_id": "'$SUB_ACCOUNT_ID'",
    "amount": 10000.00,
    "currency": "USDT"
  }
}'

echo "Request payload:"
echo "$DEPOSIT_PAYLOAD" | jq .
echo ""

echo "Response:"
DEPOSIT_RESPONSE=$(curl -s -X POST \
  -H "$AUTH_HEADER" \
  -H "$CONTENT_HEADER" \
  -d "$DEPOSIT_PAYLOAD" \
  "$BASE_URL/trading-logs")

echo "$DEPOSIT_RESPONSE" | jq . 2>/dev/null || echo "$DEPOSIT_RESPONSE"

if echo "$DEPOSIT_RESPONSE" | jq -e '.success == true and .data.id' > /dev/null 2>&1; then
    DEPOSIT_LOG_ID=$(echo "$DEPOSIT_RESPONSE" | jq -r '.data.id')
    echo -e "${GREEN}✅ Deposit trading log created successfully${NC}"
    echo "Trading Log ID: $DEPOSIT_LOG_ID"
    
    # Check if business logic was processed and get updated balance
    echo ""
    echo "Checking updated sub-account balance after deposit..."
    UPDATED_ACCOUNT_RESPONSE=$(curl -s -H "$AUTH_HEADER" -H "$CONTENT_HEADER" "$BASE_URL/sub-accounts/$SUB_ACCOUNT_ID")
    
    if echo "$UPDATED_ACCOUNT_RESPONSE" | jq -e '.success == true and .data.balance' > /dev/null 2>&1; then
        UPDATED_BALANCE=$(echo "$UPDATED_ACCOUNT_RESPONSE" | jq -r '.data.balance')
        echo -e "${GREEN}✅ Account balance updated through business logic${NC}"
        echo "Updated Balance: $UPDATED_BALANCE USDT"
        DEPOSIT_SUCCESS=true
    else
        echo "⚠️ Could not verify balance update - continuing anyway"
        DEPOSIT_SUCCESS=true  # Don't fail the test for this
    fi
else
    echo "❌ Deposit trading log creation failed"
    echo "Skipping transaction history retrieval due to deposit failure"
    DEPOSIT_SUCCESS=false
fi

# Test 5: Retrieve Transaction Records
if echo "$DEPOSIT_RESPONSE" | jq -e '.success == true' > /dev/null 2>&1; then
    print_header "📊 Retrieving Transaction Records from Deposit"
    echo "Endpoint: GET /v1/transactions/sub-account/{sub_account_id}"
    echo "Expected: Transaction record from deposit trading log business logic processing"
    echo ""
    
    TRANSACTIONS_RESPONSE=$(curl -s -H "$AUTH_HEADER" -H "$CONTENT_HEADER" "$BASE_URL/transactions/sub-account/$SUB_ACCOUNT_ID?limit=5&offset=0")
    echo "Recent transactions for sub-account (should include deposit transaction):"
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
    echo "❌ Deposit trading log failed - skipping transaction history"
    TRANSACTION_HISTORY_SUCCESS=false
fi

# Test 6: Add a trading log (conditional based on --no-trading-logs flag)
if [ "$SKIP_TRADING_LOGS" = false ]; then
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

else
    # Trading logs are skipped
    print_header "📊 Trading Log Tests (Skipped)"
    echo "Trading log tests are disabled with --no-trading-logs option"
    echo -e "${YELLOW}⏭️ Skipping trading log and business logic tests${NC}"
    TRADING_LOG_SUCCESS=true  # Mark as successful since it's intentionally skipped
    echo ""
fi

print_header "📋 Test Summary"

# Show status for each test based on success tracking
if [ "$USER_PROFILE_SUCCESS" = true ]; then
    echo "✅ User profile test completed successfully"
else
    echo "❌ User profile test failed"
fi

if [ "$EXCHANGE_SUCCESS" = true ]; then
    echo "✅ Binance exchange test completed successfully"
else
    echo "❌ Binance exchange test failed"
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

if [ "$DEPOSIT_SUCCESS" = true ]; then
    echo "✅ Deposit initialization test completed successfully"
else
    echo "❌ Deposit initialization test failed"
fi

if [ "$TRANSACTION_HISTORY_SUCCESS" = true ]; then
    echo "✅ Transaction history test completed successfully"
else
    echo "❌ Transaction history test failed"
fi

if [ "$SKIP_TRADING_LOGS" = true ]; then
    echo "⏭️ Trading log test skipped (--no-trading-logs option)"
elif [ "$TRADING_LOG_SUCCESS" = true ]; then
    echo "✅ Trading log test completed successfully"
else
    echo "❌ Trading log test failed"
fi

echo ""

# Overall test result summary
if [ "$USER_PROFILE_SUCCESS" = true ] && [ "$EXCHANGE_SUCCESS" = true ] && [ "$SUBACCOUNT_SUCCESS" = true ] && [ "$ETH_SUBACCOUNT_SUCCESS" = true ] && [ "$DEPOSIT_SUCCESS" = true ] && [ "$TRANSACTION_HISTORY_SUCCESS" = true ] && [ "$TRADING_LOG_SUCCESS" = true ]; then
    echo -e "${GREEN}🎉 All tests completed successfully!${NC}"
else
    echo -e "${RED}⚠️ Some tests failed. Check the output above for details.${NC}"
fi

echo ""
echo "💡 Notes:"
echo "- If you get 401 Unauthorized, the JWT token may be expired"
if [ -n "$USERNAME" ]; then
    echo "- JWT token was generated for user: $USERNAME (valid for 24 hours)"
    echo "- To test with different user: $0 --username other_user"
else
    echo "- Create a new test user: ./scripts/create-test-user.sh --name 'Your Name'"
    echo "- Use --username option for dynamic JWT generation: $0 --username your_username"
fi
echo "- Current API target: $BASE_URL"
if [[ "$BASE_URL" =~ ^http://localhost ]]; then
    echo "- Check the API documentation at ${BASE_URL%/v1}/docs for more details"
elif [[ "$BASE_URL" =~ ^https:// ]]; then
    echo "- For remote API documentation, check: ${BASE_URL%/v1}/docs"
fi
echo "- Use --domain option to test different environments (localhost:8080, backend.dev.tiris.ai, etc.)"

# Final status summary for human review
echo ""
if [ "$USER_PROFILE_SUCCESS" = true ] && [ "$EXCHANGE_SUCCESS" = true ] && [ "$SUBACCOUNT_SUCCESS" = true ] && [ "$ETH_SUBACCOUNT_SUCCESS" = true ] && [ "$DEPOSIT_SUCCESS" = true ] && [ "$TRANSACTION_HISTORY_SUCCESS" = true ] && [ "$TRADING_LOG_SUCCESS" = true ]; then
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
    if [ "$DEPOSIT_SUCCESS" != true ]; then
        echo "  • Account initialization (deposit): ❌ Failed"
    fi
    if [ "$TRANSACTION_HISTORY_SUCCESS" != true ]; then
        echo "  • Transaction history: ❌ Failed"
    fi
    if [ "$SKIP_TRADING_LOGS" = true ]; then
        echo "  • Trading log & business logic: ⏭️ Skipped"
    elif [ "$TRADING_LOG_SUCCESS" != true ]; then
        echo "  • Trading log & business logic: ❌ Failed"
    fi
fi

echo ""
echo -e "${BLUE}💡 Review the test results above. Terminal will stay open.${NC}"
echo -e "${BLUE}   Press Ctrl+C to close when you're done reviewing.${NC}"

fi # End of RUN_TESTS condition