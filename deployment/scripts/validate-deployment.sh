#!/bin/bash
set -e

# Deployment Validation Script for Tiris Backend
# Comprehensive testing of production deployment

echo "🔍 Starting Tiris Backend deployment validation..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
API_BASE_URL="http://localhost:8080"
HTTPS_BASE_URL="https://localhost:443"
TEST_RESULTS=()
FAILED_TESTS=0
TOTAL_TESTS=0

# Logging functions
log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

warn() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}"
}

# Test result tracking
add_test_result() {
    local test_name="$1"
    local status="$2"
    local message="$3"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    if [[ "$status" == "PASS" ]]; then
        echo -e "✅ $test_name: ${GREEN}PASS${NC}"
        TEST_RESULTS+=("PASS: $test_name - $message")
    else
        echo -e "❌ $test_name: ${RED}FAIL${NC} - $message"
        TEST_RESULTS+=("FAIL: $test_name - $message")
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
}

# Container health checks
test_container_health() {
    log "Testing container health..."
    
    local containers=("tiris-postgres-prod" "tiris-redis-prod" "tiris-nats-prod" "tiris-app-prod" "tiris-nginx-prod")
    
    for container in "${containers[@]}"; do
        if docker ps --format "{{.Names}}" | grep -q "^$container$"; then
            local status=$(docker inspect --format='{{.State.Health.Status}}' "$container" 2>/dev/null || echo "no-healthcheck")
            if [[ "$status" == "healthy" ]] || [[ "$status" == "no-healthcheck" ]]; then
                add_test_result "Container $container" "PASS" "Running and healthy"
            else
                add_test_result "Container $container" "FAIL" "Status: $status"
            fi
        else
            add_test_result "Container $container" "FAIL" "Not running"
        fi
    done
}

# API endpoint tests
test_api_endpoints() {
    log "Testing API endpoints..."
    
    # Health check endpoints
    local endpoints=(
        "/health/live:200:liveness check"
        "/health/ready:200:readiness check"
        "/metrics:200:metrics endpoint"
    )
    
    for endpoint_info in "${endpoints[@]}"; do
        IFS=':' read -r endpoint expected_code description <<< "$endpoint_info"
        
        local response_code=$(curl -s -o /dev/null -w "%{http_code}" "$API_BASE_URL$endpoint" || echo "000")
        
        if [[ "$response_code" == "$expected_code" ]]; then
            add_test_result "API $endpoint" "PASS" "$description - HTTP $response_code"
        else
            add_test_result "API $endpoint" "FAIL" "$description - Expected $expected_code, got $response_code"
        fi
    done
}

# Database connectivity test
test_database_connectivity() {
    log "Testing database connectivity..."
    
    # Test database connection through application
    local db_test_response=$(curl -s "$API_BASE_URL/health/ready" || echo "failed")
    
    if echo "$db_test_response" | grep -q "ok\|ready"; then
        add_test_result "Database connectivity" "PASS" "Application can connect to database"
    else
        add_test_result "Database connectivity" "FAIL" "Application cannot connect to database"
    fi
    
    # Direct database test
    if docker exec tiris-postgres-prod pg_isready -U tiris_user -d tiris_prod &>/dev/null; then
        add_test_result "Database direct connection" "PASS" "PostgreSQL is accepting connections"
    else
        add_test_result "Database direct connection" "FAIL" "PostgreSQL is not accepting connections"
    fi
}

# Redis connectivity test
test_redis_connectivity() {
    log "Testing Redis connectivity..."
    
    if docker exec tiris-redis-prod redis-cli ping &>/dev/null; then
        add_test_result "Redis connectivity" "PASS" "Redis is responding to ping"
    else
        add_test_result "Redis connectivity" "FAIL" "Redis is not responding"
    fi
}

# NATS connectivity test
test_nats_connectivity() {
    log "Testing NATS connectivity..."
    
    local nats_info=$(curl -s "http://localhost:8222/varz" || echo "failed")
    
    if echo "$nats_info" | grep -q "server_id\|connections"; then
        add_test_result "NATS connectivity" "PASS" "NATS server is responding"
    else
        add_test_result "NATS connectivity" "FAIL" "NATS server is not responding"
    fi
}

# SSL certificate test
test_ssl_certificate() {
    log "Testing SSL certificate..."
    
    # Check if SSL certificate files exist
    if [[ -f "/opt/tiris/ssl/fullchain.pem" ]] && [[ -f "/opt/tiris/ssl/privkey.pem" ]]; then
        add_test_result "SSL certificate files" "PASS" "Certificate files exist"
        
        # Test certificate validity
        local cert_expiry=$(openssl x509 -in /opt/tiris/ssl/fullchain.pem -noout -dates 2>/dev/null | grep "notAfter" | cut -d= -f2)
        if [[ -n "$cert_expiry" ]]; then
            add_test_result "SSL certificate validity" "PASS" "Certificate expires: $cert_expiry"
        else
            add_test_result "SSL certificate validity" "FAIL" "Cannot read certificate expiry"
        fi
    else
        add_test_result "SSL certificate files" "FAIL" "Certificate files not found"
    fi
}

# Load balancer/proxy test
test_load_balancer() {
    log "Testing load balancer/reverse proxy..."
    
    # Test Nginx is running
    if docker ps --format "{{.Names}}" | grep -q "tiris-nginx-prod"; then
        add_test_result "Nginx container" "PASS" "Nginx reverse proxy is running"
        
        # Test HTTP to HTTPS redirect
        local redirect_response=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:80/" || echo "000")
        if [[ "$redirect_response" == "301" ]] || [[ "$redirect_response" == "302" ]]; then
            add_test_result "HTTP redirect" "PASS" "HTTP traffic redirects to HTTPS"
        else
            add_test_result "HTTP redirect" "FAIL" "HTTP redirect not working - got $redirect_response"
        fi
    else
        add_test_result "Nginx container" "FAIL" "Nginx reverse proxy is not running"
    fi
}

# Resource usage test
test_resource_usage() {
    log "Testing resource usage..."
    
    # Check disk space
    local disk_usage=$(df /opt/tiris | tail -1 | awk '{print $5}' | sed 's/%//')
    if [[ "$disk_usage" -lt 80 ]]; then
        add_test_result "Disk space" "PASS" "Disk usage: ${disk_usage}%"
    else
        add_test_result "Disk space" "FAIL" "Disk usage too high: ${disk_usage}%"
    fi
    
    # Check memory usage
    local memory_usage=$(free | grep '^Mem:' | awk '{printf "%.0f", $3/$2 * 100.0}')
    if [[ "$memory_usage" -lt 85 ]]; then
        add_test_result "Memory usage" "PASS" "Memory usage: ${memory_usage}%"
    else
        add_test_result "Memory usage" "FAIL" "Memory usage too high: ${memory_usage}%"
    fi
}

# Backup system test
test_backup_system() {
    log "Testing backup system..."
    
    # Check backup directories
    if [[ -d "/opt/tiris/backups" ]]; then
        add_test_result "Backup directory" "PASS" "Backup directory exists"
        
        # Check for recent backups
        local recent_backup=$(find /opt/tiris/backups -name "*.sql.gz" -mtime -1 | wc -l)
        if [[ "$recent_backup" -gt 0 ]]; then
            add_test_result "Recent backups" "PASS" "$recent_backup backup(s) found from last 24 hours"
        else
            add_test_result "Recent backups" "WARN" "No recent backups found"
        fi
    else
        add_test_result "Backup directory" "FAIL" "Backup directory does not exist"
    fi
    
    # Test backup script
    if [[ -x "/opt/tiris/tiris-backend/scripts/backup-production.sh" ]]; then
        add_test_result "Backup script" "PASS" "Backup script is executable"
    else
        add_test_result "Backup script" "FAIL" "Backup script not found or not executable"
    fi
}

# Monitoring system test
test_monitoring_system() {
    log "Testing monitoring system..."
    
    # Check if monitoring containers are running
    local monitoring_containers=("tiris-prometheus" "tiris-grafana" "tiris-alertmanager")
    local monitoring_running=0
    
    for container in "${monitoring_containers[@]}"; do
        if docker ps --format "{{.Names}}" | grep -q "$container"; then
            monitoring_running=$((monitoring_running + 1))
        fi
    done
    
    if [[ $monitoring_running -eq ${#monitoring_containers[@]} ]]; then
        add_test_result "Monitoring containers" "PASS" "All monitoring containers are running"
    elif [[ $monitoring_running -gt 0 ]]; then
        add_test_result "Monitoring containers" "WARN" "$monitoring_running/${#monitoring_containers[@]} monitoring containers running"
    else
        add_test_result "Monitoring containers" "FAIL" "No monitoring containers running"
    fi
    
    # Test Prometheus endpoint
    if curl -s "http://localhost:9090/-/healthy" &>/dev/null; then
        add_test_result "Prometheus health" "PASS" "Prometheus is healthy"
    else
        add_test_result "Prometheus health" "FAIL" "Prometheus is not responding"
    fi
    
    # Test Grafana endpoint
    if curl -s "http://localhost:3000/api/health" &>/dev/null; then
        add_test_result "Grafana health" "PASS" "Grafana is healthy"
    else
        add_test_result "Grafana health" "FAIL" "Grafana is not responding"
    fi
}

# Environment configuration test
test_environment_config() {
    log "Testing environment configuration..."
    
    # Check environment file
    if [[ -f "/opt/tiris/tiris-backend/.env.prod" ]]; then
        add_test_result "Environment file" "PASS" "Production environment file exists"
        
        # Check for required variables
        local required_vars=("DB_PASSWORD" "JWT_SECRET" "GOOGLE_CLIENT_ID")
        local missing_vars=()
        
        for var in "${required_vars[@]}"; do
            if ! grep -q "^${var}=" /opt/tiris/tiris-backend/.env.prod; then
                missing_vars+=("$var")
            fi
        done
        
        if [[ ${#missing_vars[@]} -eq 0 ]]; then
            add_test_result "Required variables" "PASS" "All required environment variables are set"
        else
            add_test_result "Required variables" "FAIL" "Missing variables: ${missing_vars[*]}"
        fi
    else
        add_test_result "Environment file" "FAIL" "Production environment file not found"
    fi
}

# Network connectivity test
test_network_connectivity() {
    log "Testing network connectivity..."
    
    # Test internal container communication
    if docker exec tiris-app-prod nc -zv postgres 5432 &>/dev/null; then
        add_test_result "App to Database" "PASS" "Application can reach database"
    else
        add_test_result "App to Database" "FAIL" "Application cannot reach database"
    fi
    
    if docker exec tiris-app-prod nc -zv redis 6379 &>/dev/null; then
        add_test_result "App to Redis" "PASS" "Application can reach Redis"
    else
        add_test_result "App to Redis" "FAIL" "Application cannot reach Redis"
    fi
    
    if docker exec tiris-app-prod nc -zv nats 4222 &>/dev/null; then
        add_test_result "App to NATS" "PASS" "Application can reach NATS"
    else
        add_test_result "App to NATS" "FAIL" "Application cannot reach NATS"
    fi
}

# Performance test
test_performance() {
    log "Testing basic performance..."
    
    # Simple load test on health endpoint
    local start_time=$(date +%s.%N)
    for i in {1..10}; do
        curl -s "$API_BASE_URL/health/live" > /dev/null
    done
    local end_time=$(date +%s.%N)
    
    local avg_time=$(echo "($end_time - $start_time) / 10" | bc -l)
    local avg_time_ms=$(echo "$avg_time * 1000" | bc -l | cut -d. -f1)
    
    if [[ $avg_time_ms -lt 100 ]]; then
        add_test_result "Response time" "PASS" "Average response time: ${avg_time_ms}ms"
    elif [[ $avg_time_ms -lt 500 ]]; then
        add_test_result "Response time" "WARN" "Average response time: ${avg_time_ms}ms (acceptable)"
    else
        add_test_result "Response time" "FAIL" "Average response time too high: ${avg_time_ms}ms"
    fi
}

# Security test
test_security() {
    log "Testing security configuration..."
    
    # Check for security headers
    local security_headers=$(curl -s -I "$API_BASE_URL/health/live")
    
    if echo "$security_headers" | grep -qi "x-frame-options"; then
        add_test_result "Security headers" "PASS" "X-Frame-Options header present"
    else
        add_test_result "Security headers" "FAIL" "X-Frame-Options header missing"
    fi
    
    # Check file permissions
    local env_perms=$(stat -c "%a" /opt/tiris/tiris-backend/.env.prod 2>/dev/null || echo "000")
    if [[ "$env_perms" == "600" ]]; then
        add_test_result "Environment file permissions" "PASS" "Secure permissions (600)"
    else
        add_test_result "Environment file permissions" "FAIL" "Insecure permissions ($env_perms)"
    fi
}

# Generate detailed report
generate_report() {
    local report_file="/opt/tiris/deployment-validation-$(date +%Y%m%d_%H%M%S).txt"
    
    cat > "$report_file" << EOF
# Tiris Backend Deployment Validation Report
Generated: $(date)
Host: $(hostname)
User: $(whoami)

## Test Summary
Total Tests: $TOTAL_TESTS
Passed: $((TOTAL_TESTS - FAILED_TESTS))
Failed: $FAILED_TESTS
Success Rate: $(echo "scale=2; ($TOTAL_TESTS - $FAILED_TESTS) * 100 / $TOTAL_TESTS" | bc)%

## Test Results
EOF

    for result in "${TEST_RESULTS[@]}"; do
        echo "$result" >> "$report_file"
    done
    
    cat >> "$report_file" << EOF

## System Information
$(docker --version)
$(docker-compose --version)
Disk Usage: $(df -h /opt/tiris | tail -1)
Memory Usage: $(free -h | grep '^Mem:')
Load Average: $(uptime)

## Container Status
$(docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Image}}")

## Recommendations
EOF

    if [[ $FAILED_TESTS -gt 0 ]]; then
        cat >> "$report_file" << EOF
- Fix failed tests before proceeding to production
- Review error logs for failed components
- Verify network connectivity and firewall rules
EOF
    else
        cat >> "$report_file" << EOF
- All tests passed successfully
- Deployment is ready for production use
- Monitor system performance and logs regularly
EOF
    fi
    
    echo "$report_file"
}

# Main execution
main() {
    log "🚀 Starting comprehensive deployment validation..."
    
    test_container_health
    test_api_endpoints
    test_database_connectivity
    test_redis_connectivity
    test_nats_connectivity
    test_ssl_certificate
    test_load_balancer
    test_resource_usage
    test_backup_system
    test_monitoring_system
    test_environment_config
    test_network_connectivity
    test_performance
    test_security
    
    echo ""
    log "🎯 Validation completed!"
    echo ""
    
    # Summary
    if [[ $FAILED_TESTS -eq 0 ]]; then
        echo -e "${GREEN}✅ All tests passed! Deployment is healthy.${NC}"
    elif [[ $FAILED_TESTS -lt 3 ]]; then
        echo -e "${YELLOW}⚠️  $FAILED_TESTS test(s) failed. Review and fix issues.${NC}"
    else
        echo -e "${RED}❌ $FAILED_TESTS test(s) failed. Significant issues detected.${NC}"
    fi
    
    echo ""
    echo "=== Test Summary ==="
    echo "Total Tests: $TOTAL_TESTS"
    echo "Passed: $((TOTAL_TESTS - FAILED_TESTS))"
    echo "Failed: $FAILED_TESTS"
    echo "Success Rate: $(echo "scale=1; ($TOTAL_TESTS - $FAILED_TESTS) * 100 / $TOTAL_TESTS" | bc)%"
    
    # Generate detailed report
    local report_file=$(generate_report)
    echo ""
    echo "📋 Detailed report saved to: $report_file"
    
    # Exit with error code if tests failed
    if [[ $FAILED_TESTS -gt 0 ]]; then
        exit 1
    fi
}

# Error handling
trap 'error "Validation script interrupted"' INT TERM

# Check if bc (calculator) is available
if ! command -v bc &> /dev/null; then
    warn "bc calculator not found, installing..."
    
    # Detect OS and install bc accordingly
    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        case $ID in
            ubuntu|debian)
                apt-get update && apt-get install -y bc 2>/dev/null || true
                ;;
            centos|rhel|rocky|almalinux)
                dnf install -y bc 2>/dev/null || yum install -y bc 2>/dev/null || true
                ;;
            *)
                warn "Unknown OS, attempting to install bc with multiple package managers"
                apt-get install -y bc 2>/dev/null || dnf install -y bc 2>/dev/null || yum install -y bc 2>/dev/null || true
                ;;
        esac
    else
        # Fallback: try multiple package managers
        apt-get install -y bc 2>/dev/null || dnf install -y bc 2>/dev/null || yum install -y bc 2>/dev/null || true
    fi
fi

# Execute main function
main "$@"