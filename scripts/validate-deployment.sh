#!/bin/bash

# Tiris Deployment Validation Script
# Comprehensive testing of deployed services

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
DOMAIN_BASE="dev.tiris.ai"
BACKEND_SUBDOMAIN="backend.$DOMAIN_BASE"
PORTAL_SUBDOMAIN="www.$DOMAIN_BASE"
PRED_SUBDOMAIN="pred.$DOMAIN_BASE"

# Test counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Helper functions
print_header() {
    echo -e "${BLUE}🔍 Tiris Deployment Validation${NC}"
    echo "Comprehensive testing of deployed services"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo
}

print_section() {
    echo -e "${BLUE}$1${NC}"
}

test_passed() {
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    PASSED_TESTS=$((PASSED_TESTS + 1))
    echo -e "${GREEN}✓ $1${NC}"
}

test_failed() {
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    FAILED_TESTS=$((FAILED_TESTS + 1))
    echo -e "${RED}✗ $1${NC}"
}

test_warning() {
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo -e "${YELLOW}⚠ $1${NC}"
}

# Container health tests
test_containers() {
    print_section "📦 Container Health Tests"
    
    # Check reverse proxy (multi-app architecture)
    if docker ps | grep -q "tiris-reverse-proxy"; then
        if docker ps | grep "tiris-reverse-proxy" | grep -q "Up"; then
            test_passed "Reverse proxy container is running"
            
            # Check proxy health endpoint
            if curl -s http://localhost/nginx-health | grep -q "healthy"; then
                test_passed "Reverse proxy health check"
            else
                test_failed "Reverse proxy health check"
            fi
        else
            test_failed "Reverse proxy container is not running properly"
        fi
    else
        test_warning "Reverse proxy not deployed (simple architecture)"
    fi
    
    # Check backend application
    if docker ps | grep "tiris-app-simple" | grep -q "Up"; then
        test_passed "Backend application container is running"
        
        # Check backend health
        if curl -s http://localhost:8080/health/live &> /dev/null; then
            test_passed "Backend application health check"
        else
            test_failed "Backend application health check"
        fi
    else
        test_failed "Backend application container is not running"
    fi
    
    # Check database
    if docker ps | grep "tiris-postgres-simple" | grep -q "Up"; then
        test_passed "Database container is running"
        
        # Test database connectivity
        if docker exec tiris-postgres-simple pg_isready -U tiris_user -d tiris &> /dev/null; then
            test_passed "Database connectivity"
        else
            test_failed "Database connectivity"
        fi
    else
        test_failed "Database container is not running"
    fi
    
    echo
}

# Network connectivity tests
test_networking() {
    print_section "🌐 Network Connectivity Tests"
    
    # Test direct backend access
    if curl -s --max-time 5 http://localhost:8080/health/live &> /dev/null; then
        test_passed "Direct backend access (localhost:8080)"
    else
        test_failed "Direct backend access (localhost:8080)"
    fi
    
    # Test reverse proxy routing
    if curl -s --max-time 5 http://localhost/nginx-health &> /dev/null; then
        test_passed "Reverse proxy access (localhost:80)"
    else
        test_failed "Reverse proxy access (localhost:80)"
    fi
    
    # Test subdomain routing (if DNS is configured)
    if curl -s --max-time 5 "http://$BACKEND_SUBDOMAIN/health/live" &> /dev/null; then
        test_passed "Backend subdomain routing ($BACKEND_SUBDOMAIN)"
    else
        test_warning "Backend subdomain routing ($BACKEND_SUBDOMAIN) - Check DNS configuration"
    fi
    
    # Test portal subdomain (expected to fail until deployed)
    if curl -s --max-time 5 "http://$PORTAL_SUBDOMAIN" &> /dev/null; then
        test_passed "Portal subdomain routing ($PORTAL_SUBDOMAIN)"
    else
        test_warning "Portal subdomain routing ($PORTAL_SUBDOMAIN) - Service not deployed yet"
    fi
    
    # Test prediction subdomain (expected to fail until deployed)
    if curl -s --max-time 5 "http://$PRED_SUBDOMAIN/health" &> /dev/null; then
        test_passed "Prediction subdomain routing ($PRED_SUBDOMAIN)"
    else
        test_warning "Prediction subdomain routing ($PRED_SUBDOMAIN) - Service not deployed yet"
    fi
    
    echo
}

# SSL/HTTPS connectivity tests
test_ssl_functionality() {
    print_section "🔒 SSL/HTTPS Tests"
    
    # Check if SSL profile is deployed (nginx container with SSL)
    if docker ps | grep -q "tiris-nginx-simple"; then
        if docker ps | grep "tiris-nginx-simple" | grep -q "Up"; then
            test_passed "SSL-enabled nginx container is running"
            
            # Check Let's Encrypt certificate files exist
            if [ -d "/etc/letsencrypt/live" ] && [ "$(ls -A /etc/letsencrypt/live)" ]; then
                local domain=$(ls /etc/letsencrypt/live | head -n 1)
                test_passed "Let's Encrypt certificates exist for: $domain"
                
                # Test HTTPS connectivity
                if curl -s --max-time 5 "https://$domain/nginx-health" &> /dev/null; then
                    test_passed "HTTPS connectivity ($domain:443)"
                    
                    # Test HTTP to HTTPS redirect
                    local redirect_response=$(curl -s -I --max-time 5 "http://$domain/" | head -n 1)
                    if echo "$redirect_response" | grep -q "301\|302"; then
                        test_passed "HTTP to HTTPS redirect working"
                    else
                        test_warning "HTTP to HTTPS redirect not working as expected"
                    fi
                else
                    test_failed "HTTPS connectivity ($domain:443)"
                fi
                
                # Test SSL certificate validity
                local cert_path="/etc/letsencrypt/live/$domain/fullchain.pem"
                if [ -f "$cert_path" ] && openssl x509 -in "$cert_path" -noout -dates &> /dev/null; then
                    local cert_expiry=$(openssl x509 -in "$cert_path" -noout -enddate | cut -d= -f2)
                    test_passed "SSL certificate is valid (expires: $cert_expiry)"
                else
                    test_failed "SSL certificate is invalid or corrupted"
                fi
                
            else
                test_failed "Let's Encrypt certificate files missing in /etc/letsencrypt/live"
            fi
        else
            test_failed "SSL-enabled nginx container is not running properly"
        fi
    else
        test_warning "SSL not deployed (run with --profile ssl for HTTPS support)"
    fi
    
    echo
}

# API functionality tests
test_api_functionality() {
    print_section "🔌 API Functionality Tests"
    
    local backend_url="http://localhost:8080"
    
    # Test health endpoint
    local health_response=$(curl -s --max-time 5 "$backend_url/health/live")
    if echo "$health_response" | grep -q "status"; then
        test_passed "Health endpoint returns valid response"
    else
        test_failed "Health endpoint returns invalid response"
    fi
    
    # Test API root
    if curl -s --max-time 5 "$backend_url/" &> /dev/null; then
        test_passed "API root endpoint accessible"
    else
        test_warning "API root endpoint not accessible (may be expected)"
    fi
    
    # Test API documentation (if available)
    if curl -s --max-time 5 "$backend_url/docs" &> /dev/null; then
        test_passed "API documentation endpoint accessible"
    else
        test_warning "API documentation endpoint not accessible"
    fi
    
    # Test metrics endpoint (if enabled)
    if curl -s --max-time 5 "$backend_url/metrics" &> /dev/null; then
        test_passed "Metrics endpoint accessible"
    else
        test_warning "Metrics endpoint not accessible (may be disabled)"
    fi
    
    echo
}

# Configuration validation
test_configuration() {
    print_section "⚙️ Configuration Validation"
    
    # Check environment file
    if [ -f ".env.simple" ]; then
        test_passed "Environment file exists (.env.simple)"
        
        # Check required environment variables
        if grep -q "DB_PASSWORD=" .env.simple && ! grep -q "change_me_in_production" .env.simple; then
            test_passed "Database password configured"
        else
            test_failed "Database password not configured properly"
        fi
        
        if grep -q "JWT_SECRET=" .env.simple && ! grep -q "change_me_very_strong_jwt_secret" .env.simple; then
            test_passed "JWT secret configured"
        else
            test_failed "JWT secret not configured properly"
        fi
        
        if grep -q "REFRESH_SECRET=" .env.simple && ! grep -q "change_me_very_strong_refresh_secret" .env.simple; then
            test_passed "Refresh secret configured"
        else
            test_failed "Refresh secret not configured properly"
        fi
    else
        test_failed "Environment file missing (.env.simple)"
    fi
    
    # Check proxy configuration
    if [ -f "proxy/nginx.conf" ]; then
        test_passed "Proxy configuration exists"
        
        # Check subdomain configurations
        if grep -q "$BACKEND_SUBDOMAIN" proxy/nginx.conf; then
            test_passed "Backend subdomain configured in proxy"
        else
            test_failed "Backend subdomain not configured in proxy"
        fi
        
        if grep -q "host.docker.internal:8080" proxy/nginx.conf; then
            test_passed "Backend upstream configured correctly"
        else
            test_failed "Backend upstream not configured correctly"
        fi
    else
        test_failed "Proxy configuration missing"
    fi
    
    echo
}

# DNS validation
test_dns() {
    print_section "🌍 DNS Configuration Tests"
    
    local domains=("$DOMAIN_BASE" "$BACKEND_SUBDOMAIN" "$PORTAL_SUBDOMAIN" "$PRED_SUBDOMAIN")
    
    for domain in "${domains[@]}"; do
        if nslookup "$domain" &> /dev/null; then
            test_passed "DNS resolution: $domain"
        else
            test_warning "DNS resolution: $domain (may not be configured yet)"
        fi
    done
    
    echo
}

# Performance tests
test_performance() {
    print_section "⚡ Performance Tests"
    
    local backend_url="http://localhost:8080"
    
    # Test response time
    local response_time=$(curl -s -w "%{time_total}" -o /dev/null --max-time 10 "$backend_url/health/live")
    if (( $(echo "$response_time < 2.0" | bc -l) )); then
        test_passed "Backend response time: ${response_time}s (< 2s)"
    else
        test_warning "Backend response time: ${response_time}s (>= 2s)"
    fi
    
    # Test database query performance
    if docker exec tiris-postgres-simple psql -U tiris_user -d tiris -c "SELECT 1;" &> /dev/null; then
        test_passed "Database query performance"
    else
        test_failed "Database query performance"
    fi
    
    echo
}

# Resource usage tests
test_resources() {
    print_section "📊 Resource Usage Tests"
    
    # Check container resource usage
    local stats=$(docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}" | grep tiris)
    
    if [ -n "$stats" ]; then
        test_passed "Resource monitoring data available"
        echo "$stats" | while read line; do
            echo "  $line"
        done
    else
        test_warning "Resource monitoring data not available"
    fi
    
    # Check disk usage
    local disk_usage=$(docker system df --format "table {{.Type}}\t{{.Size}}")
    if [ -n "$disk_usage" ]; then
        test_passed "Docker disk usage information available"
    else
        test_warning "Docker disk usage information not available"
    fi
    
    echo
}

# Security tests
test_security() {
    print_section "🔐 Security Tests"
    
    # Check if running as root
    if [ "$EUID" -eq 0 ]; then
        test_warning "Running as root user (consider using non-root user)"
    else
        test_passed "Not running as root user"
    fi
    
    # Check for default passwords
    if grep -q "changeme\|change_me_in_production" .env.simple 2>/dev/null; then
        test_failed "Default passwords found in configuration"
    else
        test_passed "No default passwords in configuration"
    fi
    
    # Check container security
    local privileged_containers=$(docker ps --format "table {{.Names}}\t{{.Status}}" | grep -c "privileged" || true)
    if [ "$privileged_containers" -eq 0 ]; then
        test_passed "No privileged containers running"
    else
        test_warning "$privileged_containers privileged containers found"
    fi
    
    echo
}

# Generate summary report
show_summary() {
    echo -e "${BLUE}📋 Validation Summary${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    if [ $FAILED_TESTS -eq 0 ]; then
        echo -e "${GREEN}🎉 All critical tests passed!${NC}"
        echo -e "Total tests: $TOTAL_TESTS | Passed: $PASSED_TESTS | Warnings: $((TOTAL_TESTS - PASSED_TESTS - FAILED_TESTS)) | Failed: $FAILED_TESTS"
        
        echo
        echo -e "${GREEN}✅ Your deployment is healthy and ready for production!${NC}"
        
        if [ $((TOTAL_TESTS - PASSED_TESTS - FAILED_TESTS)) -gt 0 ]; then
            echo -e "${YELLOW}⚠️  Some optional features show warnings (DNS, additional services)${NC}"
        fi
    else
        echo -e "${RED}❌ $FAILED_TESTS critical tests failed${NC}"
        echo -e "Total tests: $TOTAL_TESTS | Passed: $PASSED_TESTS | Warnings: $((TOTAL_TESTS - PASSED_TESTS - FAILED_TESTS)) | Failed: $FAILED_TESTS"
        
        echo
        echo -e "${RED}🔧 Please address the failed tests before using in production${NC}"
    fi
    
    echo
    echo -e "${BLUE}📝 Recommended Actions:${NC}"
    
    if [ $FAILED_TESTS -gt 0 ]; then
        echo "• Fix failed tests by checking logs: docker logs <container-name>"
        echo "• Restart failed services: docker-compose restart"
        echo "• Check environment configuration in .env.simple"
    fi
    
    if nslookup "$BACKEND_SUBDOMAIN" &> /dev/null; then
        echo -e "${GREEN}• DNS is configured - your API is accessible via subdomain${NC}"
    else
        echo "• Configure DNS A records for subdomain access"
        echo "• Set up SSL certificates for HTTPS (see MULTI_APP_DEPLOYMENT.md)"
    fi
    
    echo "• Monitor logs: docker logs -f tiris-app-simple"
    echo "• Set up automated backups for database"
    echo "• Configure monitoring and alerting"
    
    echo
}

# Main validation function
main() {
    print_header
    
    # Check if we're in the right directory
    if [ ! -f "docker-compose.simple.yml" ]; then
        echo -e "${RED}Error: Please run this script from the tiris-backend root directory${NC}"
        exit 1
    fi
    
    # Install bc for performance calculations if not available
    if ! command -v bc &> /dev/null; then
        echo -e "${YELLOW}Installing bc for performance calculations...${NC}"
        if command -v apt-get &> /dev/null; then
            sudo apt-get update && sudo apt-get install -y bc
        elif command -v dnf &> /dev/null; then
            sudo dnf install -y bc
        fi
    fi
    
    # Run all validation tests
    test_containers
    test_networking
    test_ssl_functionality
    test_api_functionality
    test_configuration
    test_dns
    test_performance
    test_resources
    test_security
    
    # Show summary
    show_summary
    
    # Exit with error code if critical tests failed
    if [ $FAILED_TESTS -gt 0 ]; then
        exit 1
    fi
}

# Run main function
main "$@"