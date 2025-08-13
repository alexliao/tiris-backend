#!/bin/bash

# Integration Test Runner Script
# This script sets up the test environment and runs comprehensive integration tests

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
TEST_DB_NAME="tiris_integration_test_$(date +%s)"
TEST_REDIS_DB=1
TEST_TIMEOUT="30m"
VERBOSE=${VERBOSE:-false}
CLEANUP=${CLEANUP:-true}

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

# Function to check if a service is running
check_service() {
    local service=$1
    local host=$2
    local port=$3
    
    if timeout 5 bash -c "</dev/tcp/$host/$port"; then
        print_success "$service is running on $host:$port"
        return 0
    else
        print_error "$service is not running on $host:$port"
        return 1
    fi
}

# Function to wait for service to be ready
wait_for_service() {
    local service=$1
    local host=$2
    local port=$3
    local max_attempts=30
    local attempt=1
    
    print_status "Waiting for $service to be ready..."
    
    while [ $attempt -le $max_attempts ]; do
        if check_service "$service" "$host" "$port"; then
            return 0
        fi
        
        echo "Attempt $attempt/$max_attempts failed, waiting 2 seconds..."
        sleep 2
        attempt=$((attempt + 1))
    done
    
    print_error "$service failed to start within $((max_attempts * 2)) seconds"
    return 1
}

# Function to setup test database
setup_test_database() {
    print_status "Setting up test database: $TEST_DB_NAME"
    
    # Check if PostgreSQL is running
    if ! check_service "PostgreSQL" "localhost" "5432"; then
        print_error "PostgreSQL is not running. Please start PostgreSQL first."
        print_status "You can start PostgreSQL with: brew services start postgresql"
        exit 1
    fi
    
    # Create test database and user
    psql -h localhost -U postgres -c "CREATE DATABASE \"$TEST_DB_NAME\";" 2>/dev/null || {
        print_warning "Database might already exist or using different superuser"
        # Try with common alternative users
        psql -h localhost -U $(whoami) -c "CREATE DATABASE \"$TEST_DB_NAME\";" 2>/dev/null || {
            print_error "Failed to create test database. Please ensure you have PostgreSQL access."
            exit 1
        }
    }
    
    # Create test user if not exists
    psql -h localhost -U postgres -d "$TEST_DB_NAME" -c "
        DO \$\$
        BEGIN
            IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'tiris_test') THEN
                CREATE ROLE tiris_test WITH LOGIN PASSWORD 'tiris_test';
            END IF;
        END
        \$\$;
        GRANT ALL PRIVILEGES ON DATABASE \"$TEST_DB_NAME\" TO tiris_test;
        CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";
        CREATE EXTENSION IF NOT EXISTS pgcrypto;
    " 2>/dev/null || {
        # Try with alternative user
        psql -h localhost -U $(whoami) -d "$TEST_DB_NAME" -c "
            DO \$\$
            BEGIN
                IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'tiris_test') THEN
                    CREATE ROLE tiris_test WITH LOGIN PASSWORD 'tiris_test';
                END IF;
            END
            \$\$;
            GRANT ALL PRIVILEGES ON DATABASE \"$TEST_DB_NAME\" TO tiris_test;
            CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";
            CREATE EXTENSION IF NOT EXISTS pgcrypto;
        " 2>/dev/null || {
            print_error "Failed to setup test database user and extensions"
            exit 1
        }
    }
    
    print_success "Test database setup completed"
}

# Function to setup test Redis
setup_test_redis() {
    print_status "Setting up test Redis"
    
    # Check if Redis is running
    if ! check_service "Redis" "localhost" "6379"; then
        print_error "Redis is not running. Please start Redis first."
        print_status "You can start Redis with: brew services start redis"
        exit 1
    fi
    
    # Clear test Redis database
    redis-cli -n $TEST_REDIS_DB FLUSHDB >/dev/null 2>&1 || {
        print_error "Failed to setup test Redis database"
        exit 1
    }
    
    print_success "Test Redis setup completed"
}

# Function to setup test environment variables
setup_test_env() {
    print_status "Setting up test environment variables"
    
    export ENV=test
    export DB_HOST=localhost
    export DB_PORT=5432
    export DB_USER=tiris_test
    export DB_PASSWORD=tiris_test
    export DB_NAME="$TEST_DB_NAME"
    export DB_SSL_MODE=disable
    export REDIS_HOST=localhost
    export REDIS_PORT=6379
    export REDIS_DB=$TEST_REDIS_DB
    export REDIS_PASSWORD=""
    export JWT_SECRET="integration-test-jwt-secret-key-32-chars"
    export REFRESH_SECRET="integration-test-refresh-secret-32-chars"
    export MASTER_KEY="integration-test-master-key-32-chars-minimum"
    export SIGNING_KEY="integration-test-signing-key-32-chars-minimum"
    export LOG_LEVEL=error  # Reduce log noise during tests
    
    print_success "Test environment variables set"
}

# Function to run tests
run_tests() {
    local test_suite=$1
    local test_flags=""
    
    if [ "$VERBOSE" = "true" ]; then
        test_flags="$test_flags -v"
    fi
    
    print_status "Running integration tests: $test_suite"
    print_status "Test database: $TEST_DB_NAME"
    print_status "Test Redis DB: $TEST_REDIS_DB"
    print_status "Timeout: $TEST_TIMEOUT"
    
    case $test_suite in
        "all")
            go test $test_flags -timeout $TEST_TIMEOUT ./test/integration/...
            ;;
        "api")
            go test $test_flags -timeout $TEST_TIMEOUT ./test/integration -run "TestAPIIntegration"
            ;;
        "security")
            go test $test_flags -timeout $TEST_TIMEOUT ./test/integration -run "TestSecurityIntegration"
            ;;
        "database")
            go test $test_flags -timeout $TEST_TIMEOUT ./test/integration -run "TestDatabaseIntegration"
            ;;
        "performance")
            if [ "$SKIP_PERFORMANCE" != "true" ]; then
                go test $test_flags -timeout $TEST_TIMEOUT ./test/integration -run "TestPerformance"
            else
                print_warning "Skipping performance tests (SKIP_PERFORMANCE=true)"
            fi
            ;;
        *)
            print_error "Unknown test suite: $test_suite"
            print_status "Available suites: all, api, security, database, performance"
            exit 1
            ;;
    esac
}

# Function to cleanup test environment
cleanup_test_env() {
    if [ "$CLEANUP" != "true" ]; then
        print_warning "Skipping cleanup (CLEANUP=false)"
        print_status "Test database: $TEST_DB_NAME (not cleaned up)"
        return
    fi
    
    print_status "Cleaning up test environment"
    
    # Clean up Redis
    redis-cli -n $TEST_REDIS_DB FLUSHDB >/dev/null 2>&1 || {
        print_warning "Failed to clean up Redis test database"
    }
    
    # Clean up database
    psql -h localhost -U postgres -c "DROP DATABASE IF EXISTS \"$TEST_DB_NAME\";" 2>/dev/null || {
        psql -h localhost -U $(whoami) -c "DROP DATABASE IF EXISTS \"$TEST_DB_NAME\";" 2>/dev/null || {
            print_warning "Failed to clean up test database: $TEST_DB_NAME"
        }
    }
    
    print_success "Test environment cleanup completed"
}

# Function to check prerequisites
check_prerequisites() {
    print_status "Checking prerequisites..."
    
    # Check if Go is installed
    if ! command -v go >/dev/null 2>&1; then
        print_error "Go is not installed or not in PATH"
        exit 1
    fi
    
    # Check if PostgreSQL client is installed
    if ! command -v psql >/dev/null 2>&1; then
        print_error "PostgreSQL client (psql) is not installed"
        print_status "Install with: brew install postgresql"
        exit 1
    fi
    
    # Check if Redis client is installed
    if ! command -v redis-cli >/dev/null 2>&1; then
        print_error "Redis client (redis-cli) is not installed"
        print_status "Install with: brew install redis"
        exit 1
    fi
    
    print_success "Prerequisites check passed"
}

# Function to print usage
print_usage() {
    echo "Usage: $0 [OPTIONS] [TEST_SUITE]"
    echo ""
    echo "Test Suites:"
    echo "  all         Run all integration tests (default)"
    echo "  api         Run API integration tests"
    echo "  security    Run security integration tests"  
    echo "  database    Run database integration tests"
    echo "  performance Run performance tests"
    echo ""
    echo "Options:"
    echo "  -v, --verbose           Enable verbose output"
    echo "  -h, --help              Show this help message"
    echo "  --no-cleanup            Skip cleanup after tests"
    echo "  --skip-performance      Skip performance tests"
    echo "  --timeout DURATION      Set test timeout (default: 30m)"
    echo ""
    echo "Environment Variables:"
    echo "  VERBOSE=true            Enable verbose output"
    echo "  CLEANUP=false           Skip cleanup after tests"
    echo "  SKIP_PERFORMANCE=true   Skip performance tests"
    echo ""
    echo "Examples:"
    echo "  $0                      # Run all tests"
    echo "  $0 api                  # Run only API tests"
    echo "  $0 -v security          # Run security tests with verbose output"
    echo "  $0 --no-cleanup all     # Run all tests but keep test data"
}

# Parse command line arguments
TEST_SUITE="all"

while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -h|--help)
            print_usage
            exit 0
            ;;
        --no-cleanup)
            CLEANUP=false
            shift
            ;;
        --skip-performance)
            SKIP_PERFORMANCE=true
            shift
            ;;
        --timeout)
            TEST_TIMEOUT="$2"
            shift 2
            ;;
        all|api|security|database|performance)
            TEST_SUITE="$1"
            shift
            ;;
        *)
            print_error "Unknown option: $1"
            print_usage
            exit 1
            ;;
    esac
done

# Main execution
main() {
    print_status "Starting Tiris Backend Integration Tests"
    print_status "Test suite: $TEST_SUITE"
    
    # Set trap for cleanup on exit
    trap cleanup_test_env EXIT
    
    # Run setup steps
    check_prerequisites
    setup_test_env
    setup_test_database
    setup_test_redis
    
    # Run the tests
    print_status "Starting test execution..."
    
    if run_tests "$TEST_SUITE"; then
        print_success "All tests passed!"
        exit 0
    else
        print_error "Some tests failed!"
        exit 1
    fi
}

# Run main function
main "$@"