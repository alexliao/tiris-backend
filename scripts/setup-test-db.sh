#!/bin/bash

# PostgreSQL Test Database Setup Script for Tiris Backend
# This script creates the PostgreSQL test user and database needed for integration tests

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default PostgreSQL connection settings
POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_ADMIN_USER="${POSTGRES_ADMIN_USER:-postgres}"
POSTGRES_ADMIN_PASSWORD="${POSTGRES_ADMIN_PASSWORD:-}"

# Test database settings
TEST_DB_USER="tiris_test"
TEST_DB_PASSWORD="tiris_test"
TEST_DB_NAME="tiris_test"

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

Set up PostgreSQL test database and user for integration tests.

OPTIONS:
    --host HOST             PostgreSQL host (default: localhost)
    --port PORT             PostgreSQL port (default: 5432)
    --admin-user USER       PostgreSQL admin user (default: postgres)
    --admin-password PASS   PostgreSQL admin password
    --test-user USER        Test user name (default: tiris_test)
    --test-password PASS    Test user password (default: tiris_test)
    --test-db DATABASE      Test database name (default: tiris_test)
    --drop-existing         Drop existing test database and user
    --help                  Show this help message

EXAMPLES:
    # Basic setup with default values
    $0

    # Setup with custom admin password
    $0 --admin-password mypassword

    # Setup with custom connection details
    $0 --host db.example.com --port 5433 --admin-user admin

    # Drop and recreate existing test database
    $0 --drop-existing

ENVIRONMENT VARIABLES:
    POSTGRES_HOST           PostgreSQL host
    POSTGRES_PORT           PostgreSQL port
    POSTGRES_ADMIN_USER     PostgreSQL admin user
    POSTGRES_ADMIN_PASSWORD PostgreSQL admin password

EOF
}

# Function to check if PostgreSQL is accessible
check_postgres_connection() {
    print_status "Checking PostgreSQL connection to $POSTGRES_HOST:$POSTGRES_PORT..."
    
    local conn_string="host=$POSTGRES_HOST port=$POSTGRES_PORT user=$POSTGRES_ADMIN_USER"
    if [ -n "$POSTGRES_ADMIN_PASSWORD" ]; then
        export PGPASSWORD="$POSTGRES_ADMIN_PASSWORD"
        conn_string="$conn_string password=$POSTGRES_ADMIN_PASSWORD"
    fi
    
    if ! psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_ADMIN_USER" -d postgres -c "SELECT 1;" > /dev/null 2>&1; then
        print_error "Cannot connect to PostgreSQL with the provided credentials."
        echo "Please ensure:"
        echo "  1. PostgreSQL is running on $POSTGRES_HOST:$POSTGRES_PORT"
        echo "  2. User '$POSTGRES_ADMIN_USER' has sufficient privileges"
        echo "  3. Connection credentials are correct"
        echo "  4. For local Docker: docker compose up -d postgres"
        exit 1
    fi
    
    print_success "PostgreSQL connection verified"
}

# Function to check if user exists
user_exists() {
    local user="$1"
    local result=$(psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_ADMIN_USER" -d postgres -t -A -c "SELECT 1 FROM pg_roles WHERE rolname='$user';" 2>/dev/null)
    [ "$result" = "1" ]
}

# Function to check if database exists
database_exists() {
    local db="$1"
    local result=$(psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_ADMIN_USER" -d postgres -t -A -c "SELECT 1 FROM pg_database WHERE datname='$db';" 2>/dev/null)
    [ "$result" = "1" ]
}

# Function to drop test database and user
drop_test_database() {
    print_status "Dropping existing test database and user..."
    
    # Terminate active connections to the test database
    if database_exists "$TEST_DB_NAME"; then
        print_status "Terminating active connections to database '$TEST_DB_NAME'..."
        psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_ADMIN_USER" -d postgres -c "
            SELECT pg_terminate_backend(pid)
            FROM pg_stat_activity
            WHERE datname = '$TEST_DB_NAME'
              AND pid <> pg_backend_pid();
        " > /dev/null 2>&1 || true
        
        print_status "Dropping database '$TEST_DB_NAME'..."
        psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_ADMIN_USER" -d postgres -c "DROP DATABASE IF EXISTS $TEST_DB_NAME;" > /dev/null
        print_success "Database '$TEST_DB_NAME' dropped"
    fi
    
    # Drop test user
    if user_exists "$TEST_DB_USER"; then
        print_status "Dropping user '$TEST_DB_USER'..."
        psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_ADMIN_USER" -d postgres -c "DROP USER IF EXISTS $TEST_DB_USER;" > /dev/null
        print_success "User '$TEST_DB_USER' dropped"
    fi
}

# Function to create test user
create_test_user() {
    if user_exists "$TEST_DB_USER"; then
        print_warning "User '$TEST_DB_USER' already exists"
    else
        print_status "Creating test user '$TEST_DB_USER'..."
        psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_ADMIN_USER" -d postgres -c "
            CREATE USER $TEST_DB_USER WITH 
                PASSWORD '$TEST_DB_PASSWORD'
                CREATEDB
                LOGIN;
        " > /dev/null
        print_success "User '$TEST_DB_USER' created successfully"
    fi
}

# Function to create test database
create_test_database() {
    if database_exists "$TEST_DB_NAME"; then
        print_warning "Database '$TEST_DB_NAME' already exists"
    else
        print_status "Creating test database '$TEST_DB_NAME'..."
        psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_ADMIN_USER" -d postgres -c "
            CREATE DATABASE $TEST_DB_NAME 
                OWNER $TEST_DB_USER
                ENCODING 'UTF8'
                LC_COLLATE 'en_US.utf8'
                LC_CTYPE 'en_US.utf8'
                TEMPLATE template0;
        " > /dev/null
        print_success "Database '$TEST_DB_NAME' created successfully"
    fi
}

# Function to grant privileges
grant_privileges() {
    print_status "Granting privileges to user '$TEST_DB_USER'..."
    
    # Grant privileges on the database
    psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_ADMIN_USER" -d postgres -c "
        GRANT ALL PRIVILEGES ON DATABASE $TEST_DB_NAME TO $TEST_DB_USER;
    " > /dev/null
    
    # Connect to test database and grant schema privileges
    export PGPASSWORD="$TEST_DB_PASSWORD"
    psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$TEST_DB_USER" -d "$TEST_DB_NAME" -c "
        GRANT ALL ON SCHEMA public TO $TEST_DB_USER;
        GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO $TEST_DB_USER;
        GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO $TEST_DB_USER;
        ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO $TEST_DB_USER;
        ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO $TEST_DB_USER;
    " > /dev/null 2>&1 || true
    
    # Reset password env
    if [ -n "$POSTGRES_ADMIN_PASSWORD" ]; then
        export PGPASSWORD="$POSTGRES_ADMIN_PASSWORD"
    else
        unset PGPASSWORD
    fi
    
    print_success "Privileges granted successfully"
}

# Function to verify test setup
verify_test_setup() {
    print_status "Verifying test database setup..."
    
    # Test connection as test user
    export PGPASSWORD="$TEST_DB_PASSWORD"
    
    if psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$TEST_DB_USER" -d "$TEST_DB_NAME" -c "SELECT current_user, current_database();" > /dev/null 2>&1; then
        print_success "Test database connection verified"
    else
        print_error "Failed to connect to test database as test user"
        exit 1
    fi
    
    # Test table creation privileges
    if psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$TEST_DB_USER" -d "$TEST_DB_NAME" -c "
        CREATE TABLE IF NOT EXISTS test_privileges (id SERIAL PRIMARY KEY, name TEXT);
        INSERT INTO test_privileges (name) VALUES ('test');
        SELECT * FROM test_privileges;
        DROP TABLE test_privileges;
    " > /dev/null 2>&1; then
        print_success "Database privileges verified"
    else
        print_error "Failed to verify database privileges"
        exit 1
    fi
    
    # Reset password env
    if [ -n "$POSTGRES_ADMIN_PASSWORD" ]; then
        export PGPASSWORD="$POSTGRES_ADMIN_PASSWORD"
    else
        unset PGPASSWORD
    fi
}

# Main setup function
setup_test_database() {
    echo "==============================================="
    echo "🐘 PostgreSQL Test Database Setup"
    echo "==============================================="
    echo "Host:          $POSTGRES_HOST:$POSTGRES_PORT"
    echo "Admin User:    $POSTGRES_ADMIN_USER"
    echo "Test User:     $TEST_DB_USER"
    echo "Test Database: $TEST_DB_NAME"
    echo "==============================================="
    echo
    
    # Check PostgreSQL connection
    check_postgres_connection
    
    # Drop existing if requested
    if [ "$DROP_EXISTING" = "true" ]; then
        drop_test_database
    fi
    
    # Create test user and database
    create_test_user
    create_test_database
    grant_privileges
    verify_test_setup
    
    echo
    print_success "Test database setup completed successfully!"
    echo
    echo "==============================================="
    echo "📋 DATABASE CONNECTION DETAILS"
    echo "==============================================="
    echo "Host:     $POSTGRES_HOST"
    echo "Port:     $POSTGRES_PORT"
    echo "Database: $TEST_DB_NAME"
    echo "User:     $TEST_DB_USER"
    echo "Password: $TEST_DB_PASSWORD"
    echo
    echo "Connection string:"
    echo "postgresql://$TEST_DB_USER:$TEST_DB_PASSWORD@$POSTGRES_HOST:$POSTGRES_PORT/$TEST_DB_NAME?sslmode=disable"
    echo
    echo "==============================================="
    echo "🧪 RUNNING INTEGRATION TESTS"
    echo "==============================================="
    echo "You can now run integration tests with:"
    echo "  make test-integration"
    echo "  make test  # (includes integration tests)"
    echo
    echo "Or set environment variables:"
    echo "  export TEST_DB_HOST=$POSTGRES_HOST"
    echo "  export TEST_DB_PORT=$POSTGRES_PORT"
    echo "  export TEST_DB_USER=$TEST_DB_USER"
    echo "  export TEST_DB_PASSWORD=$TEST_DB_PASSWORD"
    echo "  export TEST_DB_NAME=$TEST_DB_NAME"
    echo "==============================================="
}

# Parse command line arguments
DROP_EXISTING="false"

while [[ $# -gt 0 ]]; do
    case $1 in
        --host)
            POSTGRES_HOST="$2"
            shift 2
            ;;
        --port)
            POSTGRES_PORT="$2"
            shift 2
            ;;
        --admin-user)
            POSTGRES_ADMIN_USER="$2"
            shift 2
            ;;
        --admin-password)
            POSTGRES_ADMIN_PASSWORD="$2"
            shift 2
            ;;
        --test-user)
            TEST_DB_USER="$2"
            shift 2
            ;;
        --test-password)
            TEST_DB_PASSWORD="$2"
            shift 2
            ;;
        --test-db)
            TEST_DB_NAME="$2"
            shift 2
            ;;
        --drop-existing)
            DROP_EXISTING="true"
            shift
            ;;
        --help)
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

# Check if psql is available
if ! command -v psql > /dev/null 2>&1; then
    print_error "psql command not found. Please install PostgreSQL client tools."
    echo "For Ubuntu/Debian: sudo apt-get install postgresql-client"
    echo "For macOS:         brew install postgresql"
    echo "For RHEL/CentOS:   sudo yum install postgresql"
    exit 1
fi

# Run the setup
setup_test_database