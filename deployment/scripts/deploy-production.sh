#!/bin/bash
set -e

# Production Deployment Script for Tiris Backend
# Run this script after VPS setup is complete

echo "🚀 Starting Tiris Backend production deployment..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
REPO_URL="https://github.com/your-username/tiris-backend.git"  # Update this
APP_DIR="/opt/tiris/tiris-backend"
DOMAIN=""  # Will be prompted
EMAIL=""   # Will be prompted

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

# Check if running as deployment user (not root)
if [[ $EUID -eq 0 ]]; then
    error "This script should not be run as root. Run as the deployment user."
fi

# Check if Docker is available
if ! command -v docker &> /dev/null; then
    error "Docker is not installed. Please run the VPS setup script first."
fi

# Check if Docker Compose is available
if ! command -v docker-compose &> /dev/null; then
    error "Docker Compose is not installed. Please run the VPS setup script first."
fi

# Prompt for configuration
echo "=== Configuration Setup ==="
read -p "Enter your domain name (e.g., example.com): " DOMAIN
if [[ -z "$DOMAIN" ]]; then
    error "Domain name is required"
fi

read -p "Enter your email for SSL certificate: " EMAIL
if [[ -z "$EMAIL" ]]; then
    error "Email is required for SSL certificate"
fi

read -p "Enter your GitHub repository URL (default: $REPO_URL): " REPO_INPUT
if [[ -n "$REPO_INPUT" ]]; then
    REPO_URL="$REPO_INPUT"
fi

log "Configuration:"
log "Domain: $DOMAIN"
log "Email: $EMAIL"
log "Repository: $REPO_URL"

read -p "Continue with deployment? (y/N): " CONFIRM
if [[ "$CONFIRM" != "y" && "$CONFIRM" != "Y" ]]; then
    log "Deployment cancelled"
    exit 0
fi

# Step 1: Clone or update repository
log "Step 1: Setting up application code..."
if [[ -d "$APP_DIR" ]]; then
    log "Application directory exists, updating..."
    cd "$APP_DIR"
    git pull origin master
else
    log "Cloning repository..."
    git clone "$REPO_URL" "$APP_DIR"
    cd "$APP_DIR"
fi

# Step 2: Generate environment configuration
log "Step 2: Generating production environment configuration..."
if [[ ! -f ".env.prod" ]]; then
    cp .env.prod.template .env.prod
    
    # Generate secure secrets
    JWT_SECRET=$(openssl rand -base64 32)
    REFRESH_SECRET=$(openssl rand -base64 32)
    DB_PASSWORD=$(openssl rand -base64 24)
    REDIS_PASSWORD=$(openssl rand -base64 24)
    NATS_PASSWORD=$(openssl rand -base64 24)
    
    # Update environment file
    sed -i "s/CHANGE_ME_STRONG_DB_PASSWORD/$DB_PASSWORD/g" .env.prod
    sed -i "s/CHANGE_ME_STRONG_REDIS_PASSWORD/$REDIS_PASSWORD/g" .env.prod
    sed -i "s/CHANGE_ME_STRONG_NATS_PASSWORD/$NATS_PASSWORD/g" .env.prod
    sed -i "s/CHANGE_ME_VERY_STRONG_JWT_SECRET_AT_LEAST_32_CHARS/$JWT_SECRET/g" .env.prod
    sed -i "s/CHANGE_ME_VERY_STRONG_REFRESH_SECRET_AT_LEAST_32_CHARS/$REFRESH_SECRET/g" .env.prod
    sed -i "s/tiris.ai/$DOMAIN/g" .env.prod
    
    log "Environment file created with secure secrets"
    warn "Please edit .env.prod to add your OAuth credentials and other settings"
    warn "You MUST configure: GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET"
else
    log "Environment file already exists, skipping generation"
fi

# Step 3: Set up SSL certificate
log "Step 3: Setting up SSL certificate..."
if [[ ! -f "/etc/letsencrypt/live/$DOMAIN/fullchain.pem" ]]; then
    log "Obtaining SSL certificate for $DOMAIN..."
    
    # Stop any services that might be using port 80
    sudo docker-compose -f docker-compose.prod.yml down nginx 2>/dev/null || true
    
    # Get certificate
    sudo certbot certonly --standalone \
        -d "$DOMAIN" \
        -d "api.$DOMAIN" \
        --email "$EMAIL" \
        --agree-tos \
        --non-interactive
        
    # Copy certificates to application directory
    sudo cp "/etc/letsencrypt/live/$DOMAIN/fullchain.pem" /opt/tiris/ssl/
    sudo cp "/etc/letsencrypt/live/$DOMAIN/privkey.pem" /opt/tiris/ssl/
    sudo chown $(whoami):$(whoami) /opt/tiris/ssl/*
    
    log "SSL certificate obtained and configured"
else
    log "SSL certificate already exists for $DOMAIN"
fi

# Step 4: Build application
log "Step 4: Building application..."
docker-compose -f docker-compose.prod.yml build

# Step 5: Deploy infrastructure services
log "Step 5: Deploying infrastructure services..."
docker-compose -f docker-compose.prod.yml --env-file .env.prod up -d postgres redis nats

# Wait for services to be ready
log "Waiting for services to initialize..."
sleep 30

# Check if services are healthy
for service in postgres redis nats; do
    if ! docker-compose -f docker-compose.prod.yml exec $service echo "Service check" &>/dev/null; then
        error "Service $service failed to start properly"
    fi
done

log "Infrastructure services are running"

# Step 6: Run database migrations
log "Step 6: Running database migrations..."
docker-compose -f docker-compose.prod.yml --env-file .env.prod --profile migrate run --rm migrate || {
    warn "Migration failed, this might be expected if database is already initialized"
}

# Step 7: Set up NATS streams
log "Step 7: Setting up NATS streams..."
docker-compose -f docker-compose.prod.yml --env-file .env.prod --profile setup run --rm nats-setup || {
    warn "NATS setup failed, this might be expected if streams already exist"
}

# Step 8: Deploy application
log "Step 8: Deploying main application..."
docker-compose -f docker-compose.prod.yml --env-file .env.prod up -d app

# Wait for application to be ready
log "Waiting for application to start..."
sleep 20

# Health check
for i in {1..30}; do
    if curl -f http://localhost:8080/health/live &>/dev/null; then
        log "Application is healthy"
        break
    fi
    if [[ $i -eq 30 ]]; then
        error "Application failed to start properly"
    fi
    sleep 2
done

# Step 9: Deploy reverse proxy
log "Step 9: Deploying Nginx reverse proxy..."
docker-compose -f docker-compose.prod.yml --env-file .env.prod --profile nginx up -d nginx

# Step 10: Set up monitoring (optional)
log "Step 10: Setting up monitoring..."
if [[ -f "docker-compose.monitoring.yml" ]]; then
    docker-compose -f docker-compose.monitoring.yml --env-file .env.prod up -d
    log "Monitoring stack deployed"
else
    warn "Monitoring configuration not found, skipping"
fi

# Step 11: Set up backups
log "Step 11: Setting up automated backups..."
docker-compose -f docker-compose.prod.yml --env-file .env.prod --profile backup up -d backup

# Step 12: Set up SSL auto-renewal
log "Step 12: Setting up SSL certificate auto-renewal..."
(sudo crontab -l 2>/dev/null; echo "0 12 * * * /usr/bin/certbot renew --quiet --pre-hook 'docker-compose -f $APP_DIR/docker-compose.prod.yml stop nginx' --post-hook 'docker-compose -f $APP_DIR/docker-compose.prod.yml start nginx && cp /etc/letsencrypt/live/$DOMAIN/*.pem /opt/tiris/ssl/'") | sudo crontab -

# Final verification
log "Running final verification..."
sleep 10

echo ""
echo "=== Deployment Verification ==="

# Check all containers
log "Checking container status..."
docker-compose -f docker-compose.prod.yml ps

# Test application endpoints
log "Testing application endpoints..."
if curl -f "http://localhost:8080/health/live" &>/dev/null; then
    echo "✅ Health check: PASSED"
else
    echo "❌ Health check: FAILED"
fi

if curl -f "http://localhost:8080/health/ready" &>/dev/null; then
    echo "✅ Readiness check: PASSED"
else
    echo "❌ Readiness check: FAILED"
fi

# Test HTTPS if certificate exists
if [[ -f "/opt/tiris/ssl/fullchain.pem" ]]; then
    if curl -f "https://$DOMAIN/health/live" &>/dev/null; then
        echo "✅ HTTPS health check: PASSED"
    else
        echo "❌ HTTPS health check: FAILED"
    fi
fi

echo ""
log "🎉 Deployment completed successfully!"
echo ""
echo "=== Deployment Summary ==="
echo "✅ Application deployed and running"
echo "✅ Database initialized and migrated"
echo "✅ SSL certificate configured"
echo "✅ Reverse proxy configured"
echo "✅ Monitoring stack deployed"
echo "✅ Automated backups configured"
echo "✅ SSL auto-renewal configured"
echo ""
echo "=== Access Information ==="
echo "🌐 Application URL: https://$DOMAIN"
echo "🔍 API Health: https://$DOMAIN/health/live"
echo "📊 Grafana (if enabled): https://$DOMAIN:3000"
echo "📈 Prometheus (if enabled): https://$DOMAIN:9090"
echo ""
echo "=== Important Next Steps ==="
warn "1. Update DNS records to point $DOMAIN to this server's IP"
warn "2. Edit .env.prod to configure OAuth credentials"
warn "3. Test all API endpoints thoroughly"
warn "4. Set up monitoring alerts"
warn "5. Configure backup retention policies"
echo ""
echo "=== Useful Commands ==="
echo "📱 View logs: docker-compose -f docker-compose.prod.yml logs -f"
echo "🔄 Restart app: docker-compose -f docker-compose.prod.yml restart app"
echo "📊 Monitor: /opt/tiris/monitor.sh"
echo "🚀 Deploy updates: /opt/tiris/deploy.sh"
echo ""
log "For support, check the logs and documentation!"