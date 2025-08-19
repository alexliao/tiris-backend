#!/bin/bash
set -e

# Production Environment Configuration Generator for Tiris Backend
# This script generates a secure .env.prod file with strong secrets

echo "🔐 Generating production environment configuration..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

warn() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}"
    exit 1
}

# Check if we're in the right directory
if [[ ! -f ".env.prod.template" ]]; then
    error "This script must be run from the project root directory where .env.prod.template exists"
fi

# Check if .env.prod already exists
if [[ -f ".env.prod" ]]; then
    warn ".env.prod already exists"
    read -p "Do you want to overwrite it? (y/N): " OVERWRITE
    if [[ "$OVERWRITE" != "y" && "$OVERWRITE" != "Y" ]]; then
        log "Cancelled. Existing .env.prod preserved."
        exit 0
    fi
    cp .env.prod .env.prod.backup.$(date +%Y%m%d_%H%M%S)
    log "Existing .env.prod backed up"
fi

log "Starting configuration generation..."

# Prompt for basic configuration
echo ""
echo "=== Basic Configuration ==="
read -p "Enter your domain name (e.g., tiris.ai): " DOMAIN
if [[ -z "$DOMAIN" ]]; then
    error "Domain name is required"
fi

read -p "Enter additional CORS origins (comma-separated, optional): " ADDITIONAL_CORS
if [[ -n "$ADDITIONAL_CORS" ]]; then
    CORS_ORIGINS="https://$DOMAIN,https://app.$DOMAIN,https://admin.$DOMAIN,$ADDITIONAL_CORS"
else
    CORS_ORIGINS="https://$DOMAIN,https://app.$DOMAIN,https://admin.$DOMAIN"
fi

read -p "Enter application version/tag (default: latest): " APP_VERSION
APP_VERSION=${APP_VERSION:-latest}

read -p "Enter number of app replicas (default: 2): " APP_REPLICAS
APP_REPLICAS=${APP_REPLICAS:-2}

# OAuth Configuration
echo ""
echo "=== OAuth Configuration ==="
read -p "Enter Google OAuth Client ID: " GOOGLE_CLIENT_ID
read -s -p "Enter Google OAuth Client Secret: " GOOGLE_CLIENT_SECRET
echo ""

read -p "Enter WeChat App ID (optional): " WECHAT_APP_ID
if [[ -n "$WECHAT_APP_ID" ]]; then
    read -s -p "Enter WeChat App Secret: " WECHAT_APP_SECRET
    echo ""
fi

# Generate secure secrets
log "Generating secure secrets..."
JWT_SECRET=$(openssl rand -base64 32)
REFRESH_SECRET=$(openssl rand -base64 32)
DB_PASSWORD=$(openssl rand -base64 24 | tr -d "=+/" | cut -c1-16)
REDIS_PASSWORD=$(openssl rand -base64 24 | tr -d "=+/" | cut -c1-16)
NATS_PASSWORD=$(openssl rand -base64 24 | tr -d "=+/" | cut -c1-16)

# Get current timestamp and commit
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT=$(git rev-parse HEAD 2>/dev/null || echo "unknown")

# Create the production environment file
log "Creating production environment file..."
cat > .env.prod << EOF
# Production Environment Configuration for Tiris Backend
# Generated on $(date)
# SECURITY WARNING: Keep this file secure and never commit to version control

# ============================================================================
# Application Configuration
# ============================================================================
APP_VERSION=$APP_VERSION
BUILD_TIME=$BUILD_TIME
GIT_COMMIT=$GIT_COMMIT
APP_PORT=8080
APP_REPLICAS=$APP_REPLICAS
LOG_LEVEL=info

# ============================================================================
# Domain Configuration
# ============================================================================
DOMAIN=$DOMAIN
CORS_ALLOWED_ORIGINS=$CORS_ORIGINS

# ============================================================================
# Database Configuration (PostgreSQL + TimescaleDB)
# ============================================================================
DB_HOST=postgres
DB_PORT=5432
DB_NAME=tiris_prod
DB_USER=tiris_user
DB_PASSWORD=$DB_PASSWORD
DB_SSL_MODE=require

# ============================================================================
# NATS Configuration
# ============================================================================
NATS_PORT=4222
NATS_HTTP_PORT=8222
NATS_USER=nats_user
NATS_PASSWORD=$NATS_PASSWORD

# ============================================================================
# Redis Configuration
# ============================================================================
REDIS_PORT=6379
REDIS_PASSWORD=$REDIS_PASSWORD

# ============================================================================
# JWT Configuration
# ============================================================================
JWT_SECRET=$JWT_SECRET
REFRESH_SECRET=$REFRESH_SECRET
JWT_EXPIRATION=3600
REFRESH_EXPIRATION=604800

# ============================================================================
# OAuth Configuration
# ============================================================================
GOOGLE_CLIENT_ID=$GOOGLE_CLIENT_ID
GOOGLE_CLIENT_SECRET=$GOOGLE_CLIENT_SECRET
EOF

# Add WeChat configuration if provided
if [[ -n "$WECHAT_APP_ID" ]]; then
    cat >> .env.prod << EOF
WECHAT_APP_ID=$WECHAT_APP_ID
WECHAT_APP_SECRET=$WECHAT_APP_SECRET
EOF
else
    cat >> .env.prod << EOF
WECHAT_APP_ID=
WECHAT_APP_SECRET=
EOF
fi

# Add remaining configuration
cat >> .env.prod << EOF

# ============================================================================
# SSL Configuration
# ============================================================================
HTTP_PORT=80
HTTPS_PORT=443
SSL_CERT_PATH=/etc/ssl/certs/$DOMAIN.crt
SSL_KEY_PATH=/etc/ssl/private/$DOMAIN.key

# ============================================================================
# Storage Paths
# ============================================================================
DATA_PATH=/opt/tiris/data
LOG_PATH=/opt/tiris/logs
BACKUP_PATH=/opt/tiris/backups

# ============================================================================
# Backup Configuration
# ============================================================================
BACKUP_SCHEDULE=0 2 * * *
BACKUP_RETENTION_DAYS=30

# ============================================================================
# Monitoring Configuration
# ============================================================================
METRICS_ENABLED=true
HEALTH_CHECK_INTERVAL=30s

# ============================================================================
# Rate Limiting
# ============================================================================
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS_PER_MINUTE=1000
RATE_LIMIT_BURST=100

# ============================================================================
# Security Configuration
# ============================================================================
# Additional security settings can be added here as needed
ENABLE_SECURITY_HEADERS=true
ENABLE_RATE_LIMITING=true
ENABLE_CORS=true

# ============================================================================
# End of Configuration
# ============================================================================
EOF

# Set secure permissions
chmod 600 .env.prod

log "Production environment file created successfully!"

# Display summary
echo ""
echo "=== Configuration Summary ==="
echo "📂 File: .env.prod"
echo "🌐 Domain: $DOMAIN"
echo "🔗 CORS Origins: $CORS_ORIGINS"
echo "📦 App Version: $APP_VERSION"
echo "🔢 App Replicas: $APP_REPLICAS"
echo "🔐 JWT Secret: *** (32 bytes, base64)"
echo "🔐 DB Password: *** (16 chars, alphanumeric)"
echo "🔐 Redis Password: *** (16 chars, alphanumeric)"
echo "🔐 NATS Password: *** (16 chars, alphanumeric)"
echo ""

# Create a secrets summary file for secure storage
cat > .env.secrets.txt << EOF
# Tiris Backend Production Secrets
# Generated on $(date)
# Store this file securely and separately from the codebase

Domain: $DOMAIN
JWT Secret: $JWT_SECRET
Refresh Secret: $REFRESH_SECRET
Database Password: $DB_PASSWORD
Redis Password: $REDIS_PASSWORD
NATS Password: $NATS_PASSWORD
Google Client Secret: $GOOGLE_CLIENT_SECRET
EOF

if [[ -n "$WECHAT_APP_SECRET" ]]; then
    echo "WeChat App Secret: $WECHAT_APP_SECRET" >> .env.secrets.txt
fi

chmod 600 .env.secrets.txt

log "Secrets summary created: .env.secrets.txt"

echo ""
echo "=== Security Reminders ==="
warn "1. Store .env.secrets.txt in a secure location (password manager, encrypted storage)"
warn "2. Never commit .env.prod or .env.secrets.txt to version control"
warn "3. Restrict file permissions: chmod 600 .env.prod"
warn "4. Rotate secrets regularly (quarterly recommended)"
warn "5. Use different secrets for different environments"

echo ""
echo "=== Next Steps ==="
echo "1. Review the generated .env.prod file"
echo "2. Test OAuth credentials with your applications"
echo "3. Set up SSL certificates for $DOMAIN"
echo "4. Deploy using: docker-compose -f docker-compose.prod.yml --env-file .env.prod up -d"

echo ""
log "🎉 Production environment configuration completed!"

# Optional: validate the configuration
read -p "Do you want to validate the configuration? (Y/n): " VALIDATE
if [[ "$VALIDATE" != "n" && "$VALIDATE" != "N" ]]; then
    log "Validating configuration..."
    
    # Check required variables
    required_vars=("DOMAIN" "JWT_SECRET" "DB_PASSWORD" "GOOGLE_CLIENT_ID")
    missing_vars=()
    
    for var in "${required_vars[@]}"; do
        if ! grep -q "^${var}=" .env.prod; then
            missing_vars+=("$var")
        fi
    done
    
    if [[ ${#missing_vars[@]} -eq 0 ]]; then
        echo "✅ Configuration validation passed"
    else
        error "Missing required variables: ${missing_vars[*]}"
    fi
fi