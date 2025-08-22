#!/bin/bash
set -e

# Let's Encrypt SSL Certificate Setup for Tiris Backend
# Production-ready SSL certificate generation and management

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
DOMAIN=""
ADDITIONAL_DOMAINS=""
EMAIL=""
STAGING=false
RENEW_ONLY=false
WILDCARD=false

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

show_usage() {
    echo "Usage: $0 --domain DOMAIN --email EMAIL [OPTIONS]"
    echo ""
    echo "Required options:"
    echo "  --domain DOMAIN           Primary domain name for SSL certificate"
    echo "  --email EMAIL             Email for Let's Encrypt registration"
    echo ""
    echo "Optional:"
    echo "  --additional-domains LIST Comma-separated list of additional domains"
    echo "  --wildcard               Request wildcard certificate (*.domain.com)"
    echo "  --staging                Use Let's Encrypt staging server (for testing)"
    echo "  --renew                  Only renew existing certificates"
    echo "  --help                   Show this help message"
    echo ""
    echo "Examples:"
    echo "  # Single domain (typical usage)"
    echo "  $0 --domain example.com --email admin@example.com"
    echo ""
    echo "  # Staging server for testing"
    echo "  $0 --domain example.com --email admin@example.com --staging"
    echo ""
    echo "  # Wildcard certificate (covers all subdomains - requires DNS challenge)"
    echo "  $0 --domain example.com --wildcard --email admin@example.com"
    echo ""
    echo "  # Multiple specific subdomains (HTTP challenge)"
    echo "  $0 --domain example.com --additional-domains api.example.com,www.example.com --email admin@example.com"
    echo ""
    echo "  # Renew existing certificates"
    echo "  $0 --renew"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --domain)
            DOMAIN="$2"
            shift 2
            ;;
        --additional-domains)
            ADDITIONAL_DOMAINS="$2"
            shift 2
            ;;
        --email)
            EMAIL="$2"
            shift 2
            ;;
        --wildcard)
            WILDCARD=true
            shift
            ;;
        --staging)
            STAGING=true
            shift
            ;;
        --renew)
            RENEW_ONLY=true
            shift
            ;;
        --help)
            show_usage
            exit 0
            ;;
        *)
            error "Unknown option: $1. Use --help for usage information."
            ;;
    esac
done

# Validate required parameters
if [[ "$RENEW_ONLY" != "true" ]]; then
    if [[ -z "$DOMAIN" ]]; then
        error "Domain is required. Use --domain DOMAIN"
    fi
    
    if [[ -z "$EMAIL" ]]; then
        error "Email is required. Use --email EMAIL"
    fi
    
    # Validate domain format (supports subdomains)
    if [[ ! "$DOMAIN" =~ ^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*\.[a-zA-Z]{2,}$ ]]; then
        error "Invalid domain format: $DOMAIN"
    fi
    
    # Validate email format
    if [[ ! "$EMAIL" =~ ^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$ ]]; then
        error "Invalid email format: $EMAIL"
    fi
fi

# Check prerequisites
check_prerequisites() {
    info "Checking prerequisites..."
    
    if ! command -v docker &> /dev/null; then
        error "Docker is not installed. Please install Docker first."
    fi
    
    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        error "Docker Compose is not installed. Please install Docker Compose first."
    fi
    
    # Check if running as root or with sudo access for creating directories
    if [[ ! -d "/etc/letsencrypt" ]] && [[ $EUID -ne 0 ]]; then
        error "Need sudo access to create /etc/letsencrypt directory. Run with sudo or as root."
    fi
    
    log "Prerequisites check passed"
}

# Prepare directories
prepare_directories() {
    info "Preparing directories..."
    
    # Create Let's Encrypt directories
    sudo mkdir -p /etc/letsencrypt
    sudo mkdir -p /var/www/certbot
    
    # Set proper permissions
    sudo chown -R $USER:$USER /var/www/certbot
    
    log "Directories prepared"
}

# Configure nginx for domain
configure_nginx() {
    if [[ "$RENEW_ONLY" == "true" ]]; then
        info "Renew mode - skipping nginx configuration"
        return
    fi
    
    info "Configuring nginx for domain: $DOMAIN"
    
    # Replace domain placeholder in nginx config
    sed "s/DOMAIN_PLACEHOLDER/$DOMAIN/g" nginx.simple.conf > nginx.simple.conf.tmp
    mv nginx.simple.conf.tmp nginx.simple.conf
    
    log "Nginx configured for domain: $DOMAIN"
}

# Start nginx for HTTP-01 challenge
start_nginx_for_challenge() {
    info "Starting nginx for Let's Encrypt challenge..."
    
    # Create a temporary nginx config for challenge only
    cat > nginx.challenge.conf << EOF
events {
    worker_connections 1024;
}

http {
    server {
        listen 80;
        server_name $DOMAIN;
        
        location ^~ /.well-known/acme-challenge/ {
            root /var/www/certbot;
            try_files \$uri =404;
        }
        
        location /nginx-health {
            access_log off;
            return 200 "healthy\\n";
            add_header Content-Type text/plain;
        }
        
        location / {
            return 200 "Let's Encrypt challenge server\\n";
            add_header Content-Type text/plain;
        }
    }
}
EOF

    # Start nginx with challenge config
    docker run --rm -d --name nginx-challenge \
        -p 80:80 \
        -v $(pwd)/nginx.challenge.conf:/etc/nginx/nginx.conf:ro \
        -v /var/www/certbot:/var/www/certbot:ro \
        nginx:alpine
    
    # Wait for nginx to start
    sleep 5
    
    log "Nginx started for challenge"
}

# Stop challenge nginx
stop_nginx_challenge() {
    info "Stopping challenge nginx..."
    docker stop nginx-challenge 2>/dev/null || true
    rm -f nginx.challenge.conf
    log "Challenge nginx stopped"
}

# Generate certificates
generate_certificates() {
    if [[ "$RENEW_ONLY" == "true" ]]; then
        info "Renewing existing certificates..."
        docker run --rm \
            -v /etc/letsencrypt:/etc/letsencrypt \
            -v /var/www/certbot:/var/www/certbot \
            certbot/certbot:latest renew --quiet
        log "Certificates renewed"
        return
    fi
    
    # Build domain list
    local domain_list="$DOMAIN"
    if [[ -n "$ADDITIONAL_DOMAINS" ]]; then
        # Replace commas with space and add -d prefix to each domain
        local additional_list=$(echo "$ADDITIONAL_DOMAINS" | sed 's/,/ /g')
        domain_list="$DOMAIN $additional_list"
        info "Generating Let's Encrypt certificates for: $DOMAIN and additional domains: $ADDITIONAL_DOMAINS"
    else
        info "Generating Let's Encrypt certificates for: $DOMAIN"
    fi
    
    # Handle wildcard certificates
    if [[ "$WILDCARD" == "true" ]]; then
        info "Wildcard certificate requested - using DNS challenge method"
        generate_wildcard_certificate
        return
    fi
    
    # Prepare certbot command
    local certbot_cmd="certonly --webroot -w /var/www/certbot"
    
    if [[ "$STAGING" == "true" ]]; then
        certbot_cmd="$certbot_cmd --staging"
        warn "Using Let's Encrypt staging server (test certificates)"
    fi
    
    # Build domain arguments
    local domain_args=""
    for domain in $domain_list; do
        domain_args="$domain_args -d $domain"
    done
    
    # Generate certificate
    docker run --rm \
        -v /etc/letsencrypt:/etc/letsencrypt \
        -v /var/www/certbot:/var/www/certbot \
        certbot/certbot:latest \
        $certbot_cmd \
        --email "$EMAIL" \
        --agree-tos \
        --no-eff-email \
        $domain_args
    
    log "Certificates generated successfully for all domains"
}

# Generate wildcard certificate using DNS challenge
generate_wildcard_certificate() {
    info "Generating wildcard certificate for: *.$DOMAIN"
    warn "This requires DNS challenge - you'll need to add a TXT record to your DNS"
    echo ""
    
    # Prepare certbot command for DNS challenge
    local certbot_cmd="certonly --manual --preferred-challenges=dns"
    
    if [[ "$STAGING" == "true" ]]; then
        certbot_cmd="$certbot_cmd --staging"
        warn "Using Let's Encrypt staging server (test certificates)"
    fi
    
    echo -e "${BLUE}📋 DNS Challenge Process (2 Steps):${NC}"
    echo ""
    echo -e "${GREEN}Step 1: GET the TXT record details${NC}"
    info "Press Enter below to start Certbot and get the TXT record details"
    echo ""
    echo -e "${YELLOW}Step 2: VERIFY after adding DNS record${NC}"
    echo "1. Certbot will show you a TXT record to add"
    echo "2. Log into your GoDaddy DNS management console"
    echo "3. Add the TXT record as instructed"
    echo "4. Wait for DNS propagation (1-2 minutes)"
    echo "5. Then press Enter in Certbot to verify"
    echo ""
    warn "⚠️  In Step 2, do NOT press Enter in Certbot until the TXT record has propagated!"
    echo ""
    
    read -p "➤ Press Enter now to start Step 1 (get TXT record details)..." -r
    echo ""
    
    # Generate wildcard certificate
    docker run --rm -it \
        -v /etc/letsencrypt:/etc/letsencrypt \
        -v /var/www/certbot:/var/www/certbot \
        certbot/certbot:latest \
        $certbot_cmd \
        --email "$EMAIL" \
        --agree-tos \
        --no-eff-email \
        -d "$DOMAIN" \
        -d "*.$DOMAIN"
    
    log "Wildcard certificate generation completed"
    echo ""
    info "Your wildcard certificate covers:"
    echo "  • $DOMAIN (apex domain)"
    echo "  • *.$DOMAIN (all subdomains)"
    echo "  • Examples: api.$DOMAIN, www.$DOMAIN, backend.$DOMAIN, pred.$DOMAIN"
}

# Verify certificates
verify_certificates() {
    if [[ "$RENEW_ONLY" == "true" ]]; then
        return
    fi
    
    info "Verifying certificates..."
    
    local cert_path="/etc/letsencrypt/live/$DOMAIN"
    
    if [[ -f "$cert_path/fullchain.pem" && -f "$cert_path/privkey.pem" ]]; then
        # Check certificate validity
        local expiry=$(sudo openssl x509 -in "$cert_path/fullchain.pem" -noout -enddate | cut -d= -f2)
        log "Certificate is valid and expires on: $expiry"
        
        # Check if certificate matches domain
        local cert_domain=$(sudo openssl x509 -in "$cert_path/fullchain.pem" -noout -subject | grep -o "CN=[^,]*" | cut -d= -f2)
        if [[ "$cert_domain" == "$DOMAIN" ]]; then
            log "Certificate domain matches: $DOMAIN"
        else
            warn "Certificate domain mismatch: expected $DOMAIN, got $cert_domain"
        fi
    else
        error "Certificate files not found in $cert_path"
    fi
}

# Setup auto-renewal
setup_auto_renewal() {
    info "Setting up automatic certificate renewal..."
    
    # Create renewal script
    sudo tee /etc/cron.d/certbot-renew > /dev/null << EOF
# Let's Encrypt certificate renewal for Tiris Backend
0 2 * * * root docker run --rm -v /etc/letsencrypt:/etc/letsencrypt -v /var/www/certbot:/var/www/certbot certbot/certbot:latest renew --quiet && docker-compose -f $(pwd)/docker-compose.simple.yml --profile ssl restart nginx
EOF
    
    log "Auto-renewal configured (runs daily at 2 AM)"
}

# Deploy with SSL
deploy_with_ssl() {
    info "Deploying application with SSL..."
    
    # Update CORS for HTTPS
    if [[ -f ".env.simple" ]]; then
        if grep -q "CORS_ALLOWED_ORIGINS=http://" .env.simple; then
            sed -i.bak "s|CORS_ALLOWED_ORIGINS=http://.*|CORS_ALLOWED_ORIGINS=https://$DOMAIN|g" .env.simple
            rm -f .env.simple.bak
            log "Updated CORS origins for HTTPS"
        fi
    fi
    
    # Deploy with SSL profile
    docker-compose -f docker-compose.simple.yml --env-file .env.simple --profile ssl up -d
    
    log "Application deployed with SSL"
}

# Main execution
main() {
    echo -e "${BLUE}🔒 Let's Encrypt SSL Setup for Tiris Backend${NC}"
    echo "=================================================="
    
    if [[ "$RENEW_ONLY" == "true" ]]; then
        info "Certificate renewal mode"
    else
        info "Setting up SSL for domain: $DOMAIN"
        info "Contact email: $EMAIL"
        if [[ "$STAGING" == "true" ]]; then
            warn "Using staging server (test certificates)"
        fi
    fi
    
    echo ""
    
    # Execute setup steps
    check_prerequisites
    prepare_directories
    
    if [[ "$RENEW_ONLY" == "false" ]]; then
        configure_nginx
        start_nginx_for_challenge
    fi
    
    # Always stop any existing challenge nginx
    trap 'stop_nginx_challenge' EXIT
    
    generate_certificates
    verify_certificates
    
    if [[ "$RENEW_ONLY" == "false" ]]; then
        stop_nginx_challenge
        setup_auto_renewal
        deploy_with_ssl
        
        echo ""
        log "🎉 SSL setup completed successfully!"
        log "Your application is now available at: https://$DOMAIN"
        log "HTTP traffic will automatically redirect to HTTPS"
        
        echo ""
        info "Next steps:"
        echo "1. Test your SSL setup: https://$DOMAIN"
        echo "2. Verify certificate: openssl s_client -connect $DOMAIN:443"
        echo "3. Check auto-renewal: sudo systemctl status cron"
    else
        log "🎉 Certificate renewal completed!"
    fi
}

# Run main function
main "$@"