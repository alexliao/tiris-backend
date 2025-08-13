#!/bin/bash

# Test Coverage Report Generator
# This script generates comprehensive test coverage reports for the Tiris Backend

set -e

echo "🧪 Generating comprehensive test coverage reports..."

# Create coverage directory
mkdir -p coverage

# Run tests with coverage for different packages
echo "📊 Running services tests with coverage..."
go test ./internal/services/... -coverprofile=coverage/services.out -covermode=count

echo "📊 Running API tests with coverage..."
go test ./internal/api/... -coverprofile=coverage/api.out -covermode=count

echo "📊 Running repository tests with coverage..."
go test ./internal/repositories/... -coverprofile=coverage/repositories.out -covermode=count 2>/dev/null || echo "⚠️  Repository tests not found or failed"

echo "📊 Running middleware tests with coverage..."
go test ./internal/middleware/... -coverprofile=coverage/middleware.out -covermode=count 2>/dev/null || echo "⚠️  Middleware tests not found"

echo "📊 Running database tests with coverage..."
go test ./internal/database/... -coverprofile=coverage/database.out -covermode=count 2>/dev/null || echo "⚠️  Database tests not found"

echo "📊 Running NATS tests with coverage..."
go test ./internal/nats/... -coverprofile=coverage/nats.out -covermode=count 2>/dev/null || echo "⚠️  NATS tests not found"

# Merge all coverage profiles
echo "🔗 Merging coverage profiles..."
echo "mode: count" > coverage/merged.out

# Merge all available coverage files
for file in coverage/*.out; do
    if [[ "$file" != "coverage/merged.out" && -f "$file" ]]; then
        tail -n +2 "$file" >> coverage/merged.out 2>/dev/null || continue
    fi
done

# Generate HTML reports
echo "📄 Generating HTML coverage reports..."
go tool cover -html=coverage/services.out -o coverage/services.html 2>/dev/null || echo "⚠️  Could not generate services HTML report"
go tool cover -html=coverage/api.out -o coverage/api.html 2>/dev/null || echo "⚠️  Could not generate API HTML report"
go tool cover -html=coverage/merged.out -o coverage/merged.html

# Generate text summary
echo "📋 Generating coverage summary..."
go tool cover -func=coverage/merged.out > coverage/summary.txt

# Display results
echo ""
echo "🎯 Coverage Results:"
echo "==================="

if [[ -f coverage/services.out ]]; then
    SERVICES_COVERAGE=$(go tool cover -func=coverage/services.out | tail -1 | awk '{print $3}')
    echo "Services:     $SERVICES_COVERAGE"
fi

if [[ -f coverage/api.out ]]; then
    API_COVERAGE=$(go tool cover -func=coverage/api.out | tail -1 | awk '{print $3}')
    echo "API:          $API_COVERAGE"
fi

if [[ -f coverage/merged.out ]]; then
    TOTAL_COVERAGE=$(go tool cover -func=coverage/merged.out | tail -1 | awk '{print $3}')
    echo "Total:        $TOTAL_COVERAGE"
fi

echo ""
echo "📁 Coverage files generated:"
echo "  - HTML Reports: coverage/*.html"
echo "  - Raw Data:     coverage/*.out" 
echo "  - Summary:      coverage/summary.txt"
echo ""

# Check if we meet the target
if [[ -f coverage/services.out ]]; then
    SERVICES_PCT=$(go tool cover -func=coverage/services.out | tail -1 | awk '{print $3}' | sed 's/%//')
    if (( $(echo "$SERVICES_PCT >= 90" | bc -l) )); then
        echo "✅ Services coverage meets 90%+ target!"
    else
        echo "⚠️  Services coverage ($SERVICES_PCT%) is below 90% target"
    fi
fi

echo "🎉 Coverage report generation complete!"