#!/bin/sh
set -e

# Docker entrypoint script for Tiris Backend
# Handles different deployment modes and initialization

# Default values
COMMAND=${1:-server}
WAIT_FOR_DB=${WAIT_FOR_DB:-true}
RUN_MIGRATIONS=${RUN_MIGRATIONS:-false}
DB_MAX_RETRIES=${DB_MAX_RETRIES:-30}
DB_RETRY_INTERVAL=${DB_RETRY_INTERVAL:-2}

echo "🚀 Starting Tiris Backend..."
echo "Command: $COMMAND"
echo "Wait for DB: $WAIT_FOR_DB"
echo "Run Migrations: $RUN_MIGRATIONS"

# Function to wait for database
wait_for_database() {
    if [ "$WAIT_FOR_DB" = "true" ]; then
        echo "⏳ Waiting for database connection..."
        
        for i in $(seq 1 $DB_MAX_RETRIES); do
            if pg_isready -h "${DB_HOST:-localhost}" -p "${DB_PORT:-5432}" -U "${DB_USER:-postgres}" > /dev/null 2>&1; then
                echo "✅ Database is ready!"
                return 0
            fi
            
            if [ $i -eq $DB_MAX_RETRIES ]; then
                echo "❌ Failed to connect to database after $DB_MAX_RETRIES attempts"
                exit 1
            fi
            
            echo "   Attempt $i/$DB_MAX_RETRIES failed, retrying in $DB_RETRY_INTERVAL seconds..."
            sleep $DB_RETRY_INTERVAL
        done
    fi
}

# Function to run database migrations
run_migrations() {
    if [ "$RUN_MIGRATIONS" = "true" ]; then
        echo "🔄 Running database migrations..."
        
        if [ -f "./migrate" ]; then
            ./migrate up
            echo "✅ Migrations completed successfully"
        else
            echo "⚠️  Migration binary not found, skipping migrations"
        fi
    fi
}

# Function to validate required environment variables
validate_environment() {
    required_vars="DB_HOST DB_PORT DB_NAME DB_USER DB_PASSWORD JWT_SECRET REFRESH_SECRET"
    missing_vars=""
    
    # Add NATS_URL to required vars only if NATS is enabled
    if [ "${NATS_ENABLED:-true}" = "true" ]; then
        required_vars="$required_vars NATS_URL"
    fi
    
    for var in $required_vars; do
        eval value=\$$var
        if [ -z "$value" ]; then
            missing_vars="$missing_vars $var"
        fi
    done
    
    if [ -n "$missing_vars" ]; then
        echo "❌ Missing required environment variables:$missing_vars"
        echo "💡 Please set all required environment variables before starting the application"
        exit 1
    fi
    
    echo "✅ Environment validation passed"
}

# Function to display version information
show_version() {
    if [ -f "./server" ]; then
        echo "📋 Version Information:"
        ./server --version 2>/dev/null || echo "   Version info not available"
    fi
}

# Main execution logic
case "$COMMAND" in
    "server")
        echo "🌟 Starting Tiris Backend Server..."
        validate_environment
        show_version
        wait_for_database
        run_migrations
        
        echo "🎯 Starting HTTP server on port 8080..."
        exec ./server
        ;;
        
    "migrate")
        echo "🔄 Running database migrations..."
        wait_for_database
        exec ./migrate "${@:2}"
        ;;
        
    "migrate-up")
        echo "⬆️  Running migration up..."
        wait_for_database
        exec ./migrate up
        ;;
        
    "migrate-down")
        echo "⬇️  Running migration down..."
        wait_for_database
        exec ./migrate down
        ;;
        
    "migrate-status")
        echo "📊 Checking migration status..."
        wait_for_database
        exec ./migrate version
        ;;
        
    "health-check")
        echo "🏥 Running health check..."
        if [ -f "./server" ]; then
            # Check if server binary exists and can start
            timeout 5 ./server --version > /dev/null 2>&1
            echo "✅ Health check passed"
        else
            echo "❌ Health check failed - server binary not found"
            exit 1
        fi
        ;;
        
    "shell")
        echo "🐚 Starting interactive shell..."
        exec /bin/sh
        ;;
        
    "--help"|"help")
        echo "🆘 Tiris Backend Docker Container"
        echo ""
        echo "Available commands:"
        echo "  server          - Start the HTTP server (default)"
        echo "  migrate         - Run migration commands"
        echo "  migrate-up      - Run all pending migrations"
        echo "  migrate-down    - Rollback last migration"
        echo "  migrate-status  - Show current migration version"
        echo "  health-check    - Verify container health"
        echo "  shell           - Start interactive shell"
        echo "  help            - Show this help message"
        echo ""
        echo "Environment variables:"
        echo "  WAIT_FOR_DB          - Wait for database before starting (default: true)"
        echo "  RUN_MIGRATIONS       - Auto-run migrations on startup (default: false)"
        echo "  DB_MAX_RETRIES       - Max database connection retries (default: 30)"
        echo "  DB_RETRY_INTERVAL    - Retry interval in seconds (default: 2)"
        echo ""
        echo "Required environment variables:"
        echo "  DB_HOST, DB_PORT, DB_NAME, DB_USER, DB_PASSWORD"
        echo "  NATS_URL, JWT_SECRET, REFRESH_SECRET"
        ;;
        
    *)
        echo "❓ Unknown command: $COMMAND"
        echo "💡 Use 'help' to see available commands"
        exit 1
        ;;
esac