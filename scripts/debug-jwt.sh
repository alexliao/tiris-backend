#!/bin/bash

# Debug script to test JWT generation on VPS
# This helps isolate JWT generation issues

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

print_status "Starting JWT generation debug..."

# Check Go installation
print_status "Checking Go installation..."
if ! command -v go > /dev/null; then
    print_error "Go not found. Please install Go first."
    exit 1
fi

go_version=$(go version)
print_success "Go found: $go_version"

# Check current directory
print_status "Current directory: $(pwd)"

# Check if we're in the right directory
if [ ! -f "go.mod" ]; then
    print_error "go.mod not found. Please run this from the tiris-backend root directory."
    exit 1
fi
print_success "go.mod found"

# Check .env file
print_status "Checking .env file..."
if [ -f ".env" ]; then
    print_success ".env file found"
    if grep -q "JWT_SECRET" .env; then
        print_success "JWT_SECRET found in .env"
    else
        print_error "JWT_SECRET not found in .env"
    fi
    if grep -q "REFRESH_SECRET" .env; then
        print_success "REFRESH_SECRET found in .env"
    else
        print_error "REFRESH_SECRET not found in .env"
    fi
elif [ -f ".env.simple" ]; then
    print_warning ".env not found, but .env.simple exists"
    print_status "Copying .env.simple to .env for JWT generation..."
    cp .env.simple .env
    print_success ".env created from .env.simple"
else
    print_error "No .env or .env.simple file found"
    print_error "JWT generation requires JWT_SECRET and REFRESH_SECRET"
    exit 1
fi

# Check Go modules
print_status "Checking Go module dependencies..."
if go mod download; then
    print_success "Go module dependencies downloaded successfully"
else
    print_error "Failed to download Go module dependencies"
    exit 1
fi

# Test JWT generation
print_status "Testing JWT token generation..."
print_status "Using test user ID: 00000000-0000-0000-0000-000000000000"

jwt_output=$(go run scripts/generate-jwt-token.go \
    --user-id "00000000-0000-0000-0000-000000000000" \
    --username "debug_user" \
    --email "debug@tiris.local" \
    --role "user" \
    --duration "1h" \
    --output "token" 2>&1)

exit_code=$?

if [ $exit_code -eq 0 ] && [ -n "$jwt_output" ]; then
    print_success "JWT token generated successfully!"
    print_status "Token: $jwt_output"
    
    # Validate JWT format
    jwt_parts=$(echo "$jwt_output" | tr '.' '\n' | wc -l)
    if [ "$jwt_parts" -eq 3 ]; then
        print_success "JWT token format is valid (3 parts)"
    else
        print_warning "JWT token format may be invalid (expected 3 parts, got $jwt_parts)"
    fi
else
    print_error "JWT generation failed with exit code: $exit_code"
    print_error "Output: $jwt_output"
fi

print_status "Debug complete."