#!/bin/bash

# Create Test User Script for Tiris Backend
# This script creates a test user with OAuth authentication for development/testing

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
DEFAULT_PROVIDER="google"
DEFAULT_EXPIRY="1 year"
DEFAULT_CONTAINER="tiris-postgres-dev"
DEFAULT_DATABASE="tiris_dev"
DEFAULT_USER="tiris_user"

# Function to print colored output
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

# Function to show usage
show_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Create a test user with OAuth authentication for development/testing.

OPTIONS:
    -n, --name NAME         User's display name (required)
    -u, --username USERNAME Username (auto-generated if not provided)
    -e, --email EMAIL       Email address (auto-generated if not provided)
    -p, --provider PROVIDER OAuth provider (default: google)
    -t, --expiry EXPIRY     Token expiry (default: 1 year)
    -c, --container NAME    PostgreSQL container name (default: tiris-postgres-dev)
    -d, --database NAME     Database name (default: tiris_dev)
    --db-user USER          Database user (default: tiris_user)
    -h, --help             Show this help message

EXAMPLES:
    # Create user with just a name (other fields auto-generated)
    $0 --name "John Doe"
    
    # Create user with custom details
    $0 --name "Jane Smith" --username "jane_dev" --email "jane@example.com"
    
    # Create user with WeChat provider
    $0 --name "李明" --provider "wechat"
    
    # Create user with 6 month expiry
    $0 --name "Test User" --expiry "6 months"
    
    # Use with simple deployment (different container/database names)
    $0 --name "Simple User" --container "tiris-postgres-simple" --database "tiris"
    
    # Use with custom database configuration
    $0 --name "Custom User" --container "my-postgres" --database "my_db" --db-user "my_user"

EOF
}

# Function to check if PostgreSQL is running
check_postgres() {
    local container="$1"
    local db_user="$2"
    local database="$3"
    
    print_status "Checking PostgreSQL connection (container: $container, database: $database)..."
    
    if ! docker exec "$container" psql -U "$db_user" -d "$database" -c "SELECT 1;" > /dev/null 2>&1; then
        print_error "Cannot connect to PostgreSQL container '$container'."
        print_error "Make sure the database is running. Common commands:"
        echo "  # For development:"
        echo "  docker compose -f docker-compose.dev.yml up -d postgres"
        echo ""
        echo "  # For simple deployment:"
        echo "  docker compose -f docker-compose.simple.yml --env-file .env.simple up -d postgres"
        echo ""
        echo "  # Check available containers:"
        echo "  docker ps | grep postgres"
        exit 1
    fi
    
    print_success "PostgreSQL connection verified"
}

# Function to generate username from name
generate_username() {
    local name="$1"
    # Convert to lowercase, replace spaces with underscores, remove special chars
    echo "$name" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9]/_/g' | sed 's/__*/_/g' | sed 's/^_\|_$//g'
}

# Function to generate email from username
generate_email() {
    local username="$1"
    echo "${username}@tiris.local"
}

# Function to generate provider user ID
generate_provider_user_id() {
    local username="$1"
    local provider="$2"
    echo "${provider}_${username}_$(date +%s)"
}

# Function to generate access token
generate_access_token() {
    local username="$1"
    local provider="$2"
    echo "${provider}_token_${username}_$(date +%s)"
}

# Function to create the test user
create_test_user() {
    local name="$1"
    local username="$2"
    local email="$3"
    local provider="$4"
    local expiry="$5"
    local container="$6"
    local db_user="$7"
    local database="$8"
    
    print_status "Creating test user: $name"
    
    # Generate provider-specific details
    local provider_user_id=$(generate_provider_user_id "$username" "$provider")
    local access_token=$(generate_access_token "$username" "$provider")
    local refresh_token="${provider}_refresh_${username}_$(date +%s)"
    
    # Create user record
    print_status "Creating user record..."
    local user_id=$(docker exec "$container" psql -U "$db_user" -d "$database" -t -A -c "INSERT INTO users (id, username, email, avatar, settings, info, created_at, updated_at) VALUES (gen_random_uuid(), '$username', '$email', 'https://lh3.googleusercontent.com/a/test-user-avatar', '{\"theme\": \"light\", \"notifications\": true}', '{\"display_name\": \"$name\", \"locale\": \"en\", \"test_user\": true}', NOW(), NOW()) RETURNING id;" | head -n 1)
    
    if [ -z "$user_id" ]; then
        print_error "Failed to create user record"
        exit 1
    fi
    print_success "User record created with ID: $user_id"
    
    # Create OAuth token record
    print_status "Creating OAuth token record..."
    local token_id=$(docker exec "$container" psql -U "$db_user" -d "$database" -t -A -c "INSERT INTO oauth_tokens (user_id, provider, provider_user_id, access_token, refresh_token, expires_at, info) VALUES ('$user_id', '$provider', '$provider_user_id', '$access_token', '$refresh_token', NOW() + INTERVAL '$expiry', '{\"email\": \"$email\", \"verified_email\": true, \"name\": \"$name\", \"test_user\": true}') RETURNING id;" | head -n 1)
    
    if [ -z "$token_id" ]; then
        print_error "Failed to create OAuth token record"
        exit 1
    fi
    print_success "OAuth token record created with ID: $token_id"
    
    # Generate JWT token for API authentication
    print_status "Generating JWT token for API authentication..."
    
    # Build and run the JWT token generator
    local jwt_token=""
    local jwt_error=""
    
    if ! command -v go > /dev/null; then
        print_error "Go not found. Go is required to generate JWT tokens."
        print_error "Please install Go and try again."
        exit 1
    fi
    
    # Change to script directory to access the Go module
    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local project_dir="$(dirname "$script_dir")"
    
    cd "$project_dir" || {
        print_error "Failed to change to project directory: $project_dir"
        exit 1
    }
    
    # Generate JWT token with detailed error handling
    print_status "Running JWT token generator..."
    jwt_error=$(go run scripts/generate-jwt-token.go \
        --user-id "$user_id" \
        --username "$username" \
        --email "$email" \
        --role "user" \
        --duration "8760h" \
        --output "token" 2>&1)
    
    local exit_code=$?
    
    if [ $exit_code -eq 0 ] && [ -n "$jwt_error" ]; then
        jwt_token="$jwt_error"
        print_success "JWT token generated successfully"
        
        # Validate JWT token format (should have 3 parts separated by dots)
        local jwt_parts=$(echo "$jwt_token" | tr '.' '\n' | wc -l)
        if [ "$jwt_parts" -ne 3 ]; then
            print_error "Generated token is not in valid JWT format (expected 3 parts, got $jwt_parts)"
            print_error "Token: $jwt_token"
            exit 1
        fi
    else
        print_error "Failed to generate JWT token"
        print_error "Exit code: $exit_code"
        print_error "Error output: $jwt_error"
        print_error ""
        print_error "Common causes:"
        print_error "  - Missing .env file with JWT_SECRET and REFRESH_SECRET"
        print_error "  - Invalid environment variables"
        print_error "  - Go module dependencies not installed (run 'go mod download')"
        exit 1
    fi

    # Display summary
    echo
    print_success "Test user created successfully!"
    echo
    echo "==============================================="
    echo "📋 USER DETAILS"
    echo "==============================================="
    echo "Name:          $name"
    echo "Username:      $username"
    echo "Email:         $email"
    echo "User ID:       $user_id"
    echo "Provider:      $provider"
    echo "Expires:       1 year from now"
    echo
    echo "==============================================="
    echo "🔑 TOKEN INFORMATION"
    echo "==============================================="
    echo "OAuth Token:   $access_token"
    echo "  ↳ Purpose:   Stored in database for OAuth provider integration"
    echo "  ↳ Usage:     Internal record keeping only"
    echo
    echo "JWT Token:     $jwt_token"
    echo "  ↳ Purpose:   API authentication and authorization"
    echo "  ↳ Usage:     Use this token for all API requests"
    echo
    echo "==============================================="
    echo "🔧 API TESTING"
    echo "==============================================="
    echo "Use this Authorization header in your API requests:"
    echo
    echo "Authorization: Bearer $jwt_token"
    echo
    echo "Example curl command:"
    echo "curl -H \"Authorization: Bearer $jwt_token\" \\"
    echo "     http://localhost:8080/v1/users/me"
    echo
    echo "==============================================="
}

# Parse command line arguments
NAME=""
USERNAME=""
EMAIL=""
PROVIDER="$DEFAULT_PROVIDER"
EXPIRY="$DEFAULT_EXPIRY"
CONTAINER="$DEFAULT_CONTAINER"
DATABASE="$DEFAULT_DATABASE"
DB_USER="$DEFAULT_USER"

while [[ $# -gt 0 ]]; do
    case $1 in
        -n|--name)
            NAME="$2"
            shift 2
            ;;
        -u|--username)
            USERNAME="$2"
            shift 2
            ;;
        -e|--email)
            EMAIL="$2"
            shift 2
            ;;
        -p|--provider)
            PROVIDER="$2"
            shift 2
            ;;
        -t|--expiry)
            EXPIRY="$2"
            shift 2
            ;;
        -c|--container)
            CONTAINER="$2"
            shift 2
            ;;
        -d|--database)
            DATABASE="$2"
            shift 2
            ;;
        --db-user)
            DB_USER="$2"
            shift 2
            ;;
        -h|--help)
            show_usage
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Validate required arguments
if [ -z "$NAME" ]; then
    print_error "Name is required. Use --name or -n to specify."
    show_usage
    exit 1
fi

# Validate provider
if [[ "$PROVIDER" != "google" && "$PROVIDER" != "wechat" ]]; then
    print_error "Provider must be 'google' or 'wechat'"
    exit 1
fi

# Generate missing fields
if [ -z "$USERNAME" ]; then
    USERNAME=$(generate_username "$NAME")
    print_status "Generated username: $USERNAME"
fi

if [ -z "$EMAIL" ]; then
    EMAIL=$(generate_email "$USERNAME")
    print_status "Generated email: $EMAIL"
fi

# Check PostgreSQL connection
check_postgres "$CONTAINER" "$DB_USER" "$DATABASE"

# Create the test user
create_test_user "$NAME" "$USERNAME" "$EMAIL" "$PROVIDER" "$EXPIRY" "$CONTAINER" "$DB_USER" "$DATABASE"

print_success "Script completed successfully!"