#!/bin/bash
set -e

# Quick Deploy Script for Tiris Backend
# Gets the application online with minimal effort and configuration

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Show deployment options
show_deployment_options() {
    echo -e "${BLUE}🚀 Tiris Backend Quick Deploy${NC}"
    echo "Choose your deployment architecture:"
    echo
    echo -e "${GREEN}1. Multi-App Architecture (Recommended)${NC}"
    echo "   • Professional reverse proxy setup"
    echo "   • Subdomain-based routing (backend.dev.tiris.ai)"
    echo "   • Ready for multiple applications"
    echo "   • SSL/HTTPS ready"
    echo
    echo -e "${YELLOW}2. Simple Single-App Deployment${NC}"
    echo "   • Basic deployment on single port"
    echo "   • Quick setup for development/testing"
    echo "   • Direct port access only"
    echo
    echo -e "${BLUE}3. Help & Documentation${NC}"
    echo "   • View deployment guides"
    echo "   • Architecture information"
    echo
    read -p "Select option (1-3): " -n 1 -r
    echo
    
    case $REPLY in
        1)
            echo -e "${GREEN}Starting Multi-App Architecture deployment...${NC}"
            exec ./scripts/quick-deploy-multiapp.sh
            ;;
        2)
            echo -e "${YELLOW}Starting Simple Single-App deployment...${NC}"
            deploy_simple_app
            ;;
        3)
            show_help
            exit 0
            ;;
        *)
            echo -e "${RED}Invalid option. Please choose 1, 2, or 3.${NC}"
            exit 1
            ;;
    esac
}

show_help() {
    echo -e "${BLUE}📚 Tiris Backend Deployment Help${NC}"
    echo
    echo -e "${GREEN}Multi-App Architecture:${NC}"
    echo "• Best for production environments"
    echo "• Supports multiple applications on one VPS"
    echo "• Professional subdomain routing"
    echo "• Easy SSL/HTTPS setup"
    echo "• Command: ./scripts/quick-deploy-multiapp.sh"
    echo
    echo -e "${YELLOW}Simple Deployment:${NC}"
    echo "• Best for development/testing"
    echo "• Single application setup"
    echo "• Direct port access"
    echo "• Minimal configuration"
    echo "• Command: ./scripts/quick-deploy.sh (option 2)"
    echo
    echo -e "${BLUE}Documentation:${NC}"
    echo "• Multi-app guide: MULTI_APP_DEPLOYMENT.md"
    echo "• Architecture overview in README.md"
    echo "• Configuration examples in .env templates"
    echo
}

# Simple deployment function (original logic)
deploy_simple_app() {
    echo "🚀 Quick Deploy - Tiris Backend (Simple Mode)"
    echo "Getting your application online in under 5 minutes!"

# Configuration
DOMAIN=${DOMAIN:-localhost}
USE_PROXY=${USE_PROXY:-false}

# Helper functions
log() {
    echo -e "${GREEN}✓ $1${NC}"
}

warn() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

error() {
    echo -e "${RED}✗ $1${NC}"
    exit 1
}

info() {
    echo -e "${BLUE}ℹ $1${NC}"
}

# Check prerequisites
check_prerequisites() {
    info "Checking prerequisites..."
    
    if ! command -v docker &> /dev/null; then
        error "Docker is not installed. Please install Docker first."
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        error "Docker Compose is not installed. Please install Docker Compose first."
    fi
    
    log "Prerequisites check passed"
}

# Detect OS
detect_os() {
    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        OS_ID=$ID
    elif [[ -f /etc/redhat-release ]]; then
        OS_ID="rhel"
    elif [[ -f /etc/debian_version ]]; then
        OS_ID="debian"
    else
        OS_ID="unknown"
    fi
    
    case $OS_ID in
        ubuntu|debian)
            PKG_MANAGER="apt"
            PKG_UPDATE="apt update -qq"
            PKG_INSTALL="apt install -y"
            ;;
        centos|rhel|rocky|almalinux)
            PKG_MANAGER="dnf"
            PKG_UPDATE="dnf update -y -q"
            PKG_INSTALL="dnf install -y"
            ;;
        *)
            warn "Unknown OS, using default commands"
            PKG_MANAGER="apt"
            PKG_UPDATE="apt update -qq"
            PKG_INSTALL="apt install -y"
            ;;
    esac
}

# Quick system setup for first-time deployment
quick_system_setup() {
    info "Quick system setup..."
    
    # Detect OS first
    detect_os
    
    # Update package lists (if running as root/sudo)
    if [[ $EUID -eq 0 ]] || sudo -n true 2>/dev/null; then
        log "Updating system packages..."
        sudo $PKG_UPDATE || warn "Could not update packages (continuing anyway)"
        
        # Install curl if not present
        if ! command -v curl &> /dev/null; then
            sudo $PKG_INSTALL curl
        fi
    fi
    
    log "System setup completed"
}

# Setup environment file
setup_environment() {
    info "Setting up environment configuration..."
    
    if [[ -f ".env.simple" ]]; then
        warn "Environment file already exists. Using existing .env.simple"
        return
    fi
    
    # Copy template
    cp .env.simple.template .env.simple
    
    # Generate secure secrets
    JWT_SECRET=$(openssl rand -base64 32 2>/dev/null || echo "CHANGE_ME_GENERATE_SECURE_JWT_SECRET")
    REFRESH_SECRET=$(openssl rand -base64 32 2>/dev/null || echo "CHANGE_ME_GENERATE_SECURE_REFRESH_SECRET")
    DB_PASSWORD=$(openssl rand -base64 16 2>/dev/null | tr -d "=+/" | cut -c1-12 || echo "changeme123")
    
    # Update environment file with generated values (using | as delimiter to avoid issues with / in base64)
    sed -i.bak "s|change_me_in_production|$DB_PASSWORD|g" .env.simple
    sed -i.bak "s|change_me_very_strong_jwt_secret_32_chars_minimum|$JWT_SECRET|g" .env.simple
    sed -i.bak "s|change_me_very_strong_refresh_secret_32_chars_minimum|$REFRESH_SECRET|g" .env.simple
    
    # Clean up backup file
    rm -f .env.simple.bak
    
    log "Environment file created with secure secrets"
}

# Deploy application
deploy_application() {
    info "Deploying application..."
    
    # Build and start services
    log "Building Docker images..."
    docker compose -f docker-compose.simple.yml --env-file .env.simple build --quiet
    
    log "Starting services..."
    if [[ "$USE_PROXY" == "true" ]]; then
        docker compose -f docker-compose.simple.yml --env-file .env.simple --profile proxy up -d
    else
        docker compose -f docker-compose.simple.yml --env-file .env.simple up -d
    fi
    
    log "Services started successfully"
}

# Wait for application to be ready
wait_for_application() {
    info "Waiting for application to be ready..."
    
    local max_attempts=30
    local attempt=1
    
    while [[ $attempt -le $max_attempts ]]; do
        if curl -s http://localhost:8080/health/live > /dev/null 2>&1; then
            log "Application is ready!"
            return 0
        fi
        
        echo -n "."
        sleep 2
        attempt=$((attempt + 1))
    done
    
    error "Application failed to start within 60 seconds"
}

# Run basic tests
run_basic_tests() {
    info "Running basic health checks..."
    
    # Test health endpoint
    if curl -f http://localhost:8080/health/live > /dev/null 2>&1; then
        log "Health check: PASS"
    else
        warn "Health check: FAIL"
    fi
    
    # Test database connection
    if curl -f http://localhost:8080/health/ready > /dev/null 2>&1; then
        log "Database connection: PASS"
    else
        warn "Database connection: FAIL"
    fi
    
    # Test through proxy if enabled
    if [[ "$USE_PROXY" == "true" ]]; then
        if curl -f http://localhost:80/health/live > /dev/null 2>&1; then
            log "Proxy health check: PASS"
        else
            warn "Proxy health check: FAIL"
        fi
    fi
}

# Display success information
show_success_info() {
    echo ""
    echo "🎉 Quick deployment completed successfully!"
    echo ""
    echo "=== Your Application is Online! ==="
    
    if [[ "$USE_PROXY" == "true" ]]; then
        echo "🌐 Application URL: http://$DOMAIN"
        echo "🔍 Health Check: http://$DOMAIN/health/live"
    else
        echo "🌐 Application URL: http://$DOMAIN:8080"
        echo "🔍 Health Check: http://$DOMAIN:8080/health/live"
    fi
    
    echo ""
    echo "=== Quick Commands ==="
    echo "📊 View logs: docker compose -f docker-compose.simple.yml --env-file .env.simple logs -f"
    echo "🔄 Restart: docker compose -f docker-compose.simple.yml --env-file .env.simple restart"
    echo "🛑 Stop: docker compose -f docker-compose.simple.yml --env-file .env.simple down"
    echo "📋 Status: docker compose -f docker-compose.simple.yml --env-file .env.simple ps"
    echo ""
    echo "=== Next Steps ==="
    echo "1. Test your API endpoints"
    echo "2. Configure OAuth credentials if needed (edit .env.simple)"
    echo "3. Set up SSL certificate for HTTPS (see deployment/docs/)"
    echo "4. Consider upgrading to full production deployment for advanced features"
    echo ""
    warn "Remember: This is a minimal setup for quick deployment."
    warn "For production use, consider the full deployment in deployment/docs/"
}

# Prompt for proxy setup
ask_proxy_setup() {
    echo ""
    read -p "Do you want to use Nginx reverse proxy? (y/N): " use_proxy_input
    if [[ "$use_proxy_input" == "y" || "$use_proxy_input" == "Y" ]]; then
        USE_PROXY=true
        log "Nginx proxy will be configured"
    else
        USE_PROXY=false
        log "Direct application access (port 8080)"
    fi
}

# Ask for domain
ask_domain() {
    echo ""
    read -p "Enter your domain name (or press Enter for localhost): " domain_input
    if [[ -n "$domain_input" ]]; then
        DOMAIN="$domain_input"
        # Update CORS origins in environment file
        if [[ -f ".env.simple" ]]; then
            sed -i.bak "s|http://localhost:3000,https://localhost:3000|http://$DOMAIN,https://$DOMAIN,http://localhost:3000|g" .env.simple
            rm -f .env.simple.bak
        fi
        log "Domain set to: $DOMAIN"
    else
        DOMAIN="localhost"
        log "Using localhost"
    fi
}

# Main execution
main() {
    echo ""
    info "This script will deploy Tiris Backend with minimal configuration"
    info "Perfect for development, testing, or getting started quickly"
    echo ""
    
    # Interactive setup
    ask_domain
    ask_proxy_setup
    
    echo ""
    info "Starting deployment with domain: $DOMAIN"
    
    # Execute deployment steps
    check_prerequisites
    quick_system_setup
    setup_environment
    deploy_application
    wait_for_application
    run_basic_tests
    show_success_info
    
    echo ""
    log "🎯 Quick deployment completed! Your application is online."
}

# Handle interrupts
trap 'error "Deployment interrupted"' INT TERM

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --domain)
            DOMAIN="$2"
            shift 2
            ;;
        --proxy)
            USE_PROXY=true
            shift
            ;;
        --no-proxy)
            USE_PROXY=false
            shift
            ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo "Options:"
            echo "  --domain DOMAIN    Set domain name (default: localhost)"
            echo "  --proxy           Enable Nginx reverse proxy"
            echo "  --no-proxy        Disable Nginx reverse proxy"
            echo "  --help            Show this help message"
            exit 0
            ;;
        *)
            warn "Unknown option: $1"
            shift
            ;;
    esac
done

# Check for command line arguments to skip menu
if [ "$1" = "--simple" ]; then
    deploy_simple_app
elif [ "$1" = "--multi-app" ]; then
    exec ./scripts/quick-deploy-multiapp.sh
elif [ "$1" = "--help" ]; then
    show_help
else
    show_deployment_options
fi