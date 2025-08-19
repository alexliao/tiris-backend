#!/bin/bash

# Tiris Multi-App Quick Deploy Script
# Deploys the complete multi-application architecture with reverse proxy

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

# Helper functions
print_header() {
    echo -e "${BLUE}🚀 Tiris Multi-App Quick Deploy${NC}"
    echo "Deploying complete multi-application architecture"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo
}

print_step() {
    echo -e "${BLUE}$1${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

check_prerequisites() {
    print_step "Checking prerequisites..."
    
    # Check if running as root or with sudo access
    if [[ $EUID -eq 0 ]]; then
        print_warning "Running as root - consider using a non-root user with docker permissions"
    fi
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed. Please install Docker first."
        exit 1
    fi
    
    # Check Docker Compose
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed. Please install Docker first."
        exit 1
    fi
    
    if ! docker compose version &> /dev/null; then
        print_error "Docker Compose plugin is not available. Please install Docker Compose plugin."
        exit 1
    fi
    
    # Check if Docker is running
    if ! docker info &> /dev/null; then
        print_error "Docker is not running. Please start Docker service."
        exit 1
    fi
    
    print_success "Prerequisites check passed"
}

check_dns() {
    print_step "Checking DNS configuration..."
    
    local dns_ok=true
    local domains=("$DOMAIN_BASE" "$BACKEND_SUBDOMAIN" "$PORTAL_SUBDOMAIN" "$PRED_SUBDOMAIN")
    
    for domain in "${domains[@]}"; do
        if nslookup "$domain" &> /dev/null; then
            print_success "DNS OK: $domain"
        else
            print_warning "DNS not resolved: $domain"
            dns_ok=false
        fi
    done
    
    if [ "$dns_ok" = false ]; then
        echo
        print_warning "Some DNS records are not configured yet."
        echo "Configure these A records with your DNS provider:"
        echo "  A Record: $DOMAIN_BASE          → YOUR_VPS_IP"
        echo "  A Record: $BACKEND_SUBDOMAIN    → YOUR_VPS_IP"
        echo "  A Record: $PORTAL_SUBDOMAIN     → YOUR_VPS_IP"
        echo "  A Record: $PRED_SUBDOMAIN       → YOUR_VPS_IP"
        echo
        read -p "Continue anyway? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_error "Deployment cancelled. Please configure DNS first."
            exit 1
        fi
    fi
}

setup_environment() {
    print_step "Setting up environment configuration..."
    
    if [ ! -f ".env.simple.template" ]; then
        print_error "Environment template not found: .env.simple.template"
        exit 1
    fi
    
    # Check if .env.simple already exists
    if [ -f ".env.simple" ]; then
        print_warning "Environment file already exists. Using existing .env.simple"
        return
    fi
    
    print_step "Generating secure environment configuration..."
    
    # Generate secure passwords
    DB_PASSWORD=$(openssl rand -base64 16 | tr -d '/+=' | cut -c1-12)
    JWT_SECRET=$(openssl rand -base64 32 | tr -d '/+=' | cut -c1-32)
    REFRESH_SECRET=$(openssl rand -base64 32 | tr -d '/+=' | cut -c1-32)
    
    # Create environment file
    cp .env.simple.template .env.simple
    
    # Update environment file with generated values (using | as delimiter)
    sed -i.bak "s|change_me_in_production|$DB_PASSWORD|g" .env.simple
    sed -i.bak "s|change_me_very_strong_jwt_secret_32_chars_minimum|$JWT_SECRET|g" .env.simple
    sed -i.bak "s|change_me_very_strong_refresh_secret_32_chars_minimum|$REFRESH_SECRET|g" .env.simple
    
    # Clean up backup file
    rm -f .env.simple.bak
    
    print_success "Environment configuration generated"
}

deploy_reverse_proxy() {
    print_step "Deploying reverse proxy..."
    
    if [ ! -d "proxy" ]; then
        print_error "Proxy directory not found. Make sure you have the latest multi-app code."
        exit 1
    fi
    
    cd proxy
    
    # Stop existing proxy if running
    docker compose down 2>/dev/null || true
    
    # Start reverse proxy
    docker compose up -d
    
    # Wait for proxy to be ready
    sleep 5
    
    # Check if proxy is running
    if docker ps | grep -q "tiris-reverse-proxy"; then
        print_success "Reverse proxy deployed successfully"
        
        # Test health endpoint
        if curl -s http://localhost/nginx-health | grep -q "healthy"; then
            print_success "Reverse proxy health check passed"
        else
            print_warning "Reverse proxy health check failed, but container is running"
        fi
    else
        print_error "Failed to deploy reverse proxy"
        docker compose logs
        exit 1
    fi
    
    cd ..
}

deploy_backend() {
    print_step "Deploying backend API..."
    
    # Stop existing backend deployment
    docker compose -f docker-compose.simple.yml --env-file .env.simple down 2>/dev/null || true
    
    # Start backend with new architecture
    docker compose -f docker-compose.simple.yml --env-file .env.simple up -d --build
    
    # Wait for backend to be ready
    print_step "Waiting for backend to be ready..."
    local retries=30
    local count=0
    
    while [ $count -lt $retries ]; do
        if curl -s http://localhost:8080/health/live &> /dev/null; then
            print_success "Backend API deployed successfully"
            return
        fi
        
        count=$((count + 1))
        echo -n "."
        sleep 2
    done
    
    echo
    print_error "Backend deployment failed or timed out"
    
    # Show logs for debugging
    echo "Backend logs:"
    docker logs tiris-app-simple --tail 20
    
    exit 1
}

test_deployment() {
    print_step "Testing deployment..."
    
    # Test direct backend access
    if curl -s http://localhost:8080/health/live &> /dev/null; then
        print_success "Backend direct access: OK"
    else
        print_error "Backend direct access: FAILED"
        return 1
    fi
    
    # Test subdomain access (if DNS is configured)
    if curl -s "http://$BACKEND_SUBDOMAIN/health/live" &> /dev/null; then
        print_success "Backend subdomain access: OK"
    else
        print_warning "Backend subdomain access: Not available (check DNS configuration)"
    fi
    
    # Check container status
    local containers_ok=true
    
    if docker ps | grep -q "tiris-reverse-proxy.*Up"; then
        print_success "Reverse proxy container: Running"
    else
        print_error "Reverse proxy container: Not running"
        containers_ok=false
    fi
    
    if docker ps | grep -q "tiris-app-simple.*Up"; then
        print_success "Backend API container: Running"
    else
        print_error "Backend API container: Not running"
        containers_ok=false
    fi
    
    if docker ps | grep -q "tiris-postgres-simple.*Up"; then
        print_success "Database container: Running"
    else
        print_error "Database container: Not running"
        containers_ok=false
    fi
    
    if [ "$containers_ok" = true ]; then
        print_success "All containers are running correctly"
        return 0
    else
        print_error "Some containers are not running correctly"
        return 1
    fi
}

show_deployment_summary() {
    echo
    echo -e "${GREEN}🎉 Multi-App Deployment Complete!${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo
    echo -e "${BLUE}📋 Deployment Summary:${NC}"
    echo "• Reverse Proxy: Running on port 80"
    echo "• Backend API: Running on port 8080"
    echo "• Database: PostgreSQL with TimescaleDB"
    echo
    echo -e "${BLUE}🌐 Access URLs:${NC}"
    echo "• Backend API: http://$BACKEND_SUBDOMAIN"
    echo "• Health Check: http://$BACKEND_SUBDOMAIN/health/live"
    echo "• Direct Access: http://localhost:8080"
    echo
    echo -e "${BLUE}📁 Application Structure:${NC}"
    echo "• proxy/ - Reverse proxy (port 80)"
    echo "• tiris-backend/ - API Backend (port 8080)"
    echo "• tiris-portal/ - Frontend Portal (port 8081) - Ready for deployment"
    echo "• tiris-pred/ - Prediction Service (port 8082) - Ready for deployment"
    echo
    echo -e "${BLUE}🔧 Management Commands:${NC}"
    echo "• View logs: docker logs tiris-app-simple -f"
    echo "• Stop all: docker compose -f docker-compose.simple.yml --env-file .env.simple down && cd proxy && docker compose down"
    echo "• Restart backend: docker compose -f docker-compose.simple.yml --env-file .env.simple restart app"
    echo "• Health check: curl http://$BACKEND_SUBDOMAIN/health/live"
    echo
    echo -e "${YELLOW}📝 Next Steps:${NC}"
    echo "1. Configure SSL certificates for HTTPS (see MULTI_APP_DEPLOYMENT.md)"
    echo "2. Deploy portal application to tiris-portal/ when ready"
    echo "3. Deploy prediction service to tiris-pred/ when ready"
    echo "4. Set up monitoring and automated backups"
    echo
    echo -e "${GREEN}✅ Your Tiris Backend is now running with multi-app architecture!${NC}"
}

show_rollback_info() {
    echo
    echo -e "${YELLOW}🔄 Rollback Information:${NC}"
    echo "If you need to rollback to single-app deployment:"
    echo "1. Stop multi-app deployment:"
    echo "   cd proxy && docker compose down"
    echo "   docker compose -f docker-compose.simple.yml --env-file .env.simple down"
    echo
    echo "2. Use original deployment method:"
    echo "   ./scripts/quick-deploy.sh"
    echo
}

# Main deployment flow
main() {
    print_header
    
    # Check if we're in the right directory
    if [ ! -f "docker-compose.simple.yml" ]; then
        print_error "Please run this script from the tiris-backend root directory"
        exit 1
    fi
    
    # Run deployment steps
    check_prerequisites
    check_dns
    setup_environment
    deploy_reverse_proxy
    deploy_backend
    
    if test_deployment; then
        show_deployment_summary
        show_rollback_info
    else
        print_error "Deployment completed with errors. Check the logs above."
        exit 1
    fi
}

# Handle script interruption
trap 'echo -e "\n${RED}Deployment interrupted!${NC}"; exit 1' INT TERM

# Run main function
main "$@"