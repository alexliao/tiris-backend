#!/bin/bash
set -e

# VPS Setup Script for Tiris Backend Production Deployment
# Supports Ubuntu/Debian and CentOS/RHEL/Rocky Linux distributions

echo "🚀 Starting VPS setup for Tiris Backend..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Detect OS
detect_os() {
    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        OS=$NAME
        VER=$VERSION_ID
        OS_ID=$ID
    elif [[ -f /etc/redhat-release ]]; then
        OS=$(cat /etc/redhat-release)
        OS_ID="rhel"
    elif [[ -f /etc/debian_version ]]; then
        OS="Debian"
        OS_ID="debian"
    else
        OS="Unknown"
        OS_ID="unknown"
    fi
    
    case $OS_ID in
        ubuntu|debian)
            PKG_MANAGER="apt"
            PKG_UPDATE="apt update"
            PKG_INSTALL="apt install -y"
            FIREWALL_CMD="ufw"
            ;;
        centos|rhel|rocky|almalinux)
            PKG_MANAGER="dnf"
            PKG_UPDATE="dnf update -y"
            PKG_INSTALL="dnf install -y"
            FIREWALL_CMD="firewalld"
            ;;
        *)
            error "Unsupported operating system: $OS"
            ;;
    esac
    
    log "Detected OS: $OS ($OS_ID)"
    log "Package manager: $PKG_MANAGER"
}

# Logging function
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

# Check if running as root
if [[ $EUID -ne 0 ]]; then
   error "This script must be run as root. Use: sudo $0"
fi

# Get current user (the one who used sudo)
SUDO_USER_NAME=${SUDO_USER:-$(whoami)}
if [[ "$SUDO_USER_NAME" == "root" ]]; then
    read -p "Enter the username for the deployment user (default: tiris): " DEPLOY_USER
    DEPLOY_USER=${DEPLOY_USER:-tiris}
else
    DEPLOY_USER=$SUDO_USER_NAME
fi

log "Setting up VPS for user: $DEPLOY_USER"

# Detect OS first
detect_os

# Update system
log "Updating system packages..."
$PKG_UPDATE

# Install essential packages
log "Installing essential packages..."
if [[ "$PKG_MANAGER" == "apt" ]]; then
    # Ubuntu/Debian packages
    $PKG_INSTALL \
        curl \
        git \
        ufw \
        htop \
        nano \
        vim \
        wget \
        unzip \
        software-properties-common \
        apt-transport-https \
        ca-certificates \
        gnupg \
        lsb-release \
        certbot \
        python3-certbot-nginx \
        fail2ban \
        logrotate
elif [[ "$PKG_MANAGER" == "dnf" ]]; then
    # CentOS/RHEL packages
    $PKG_INSTALL epel-release  # Enable EPEL repository
    $PKG_INSTALL \
        curl \
        git \
        firewalld \
        htop \
        nano \
        vim \
        wget \
        unzip \
        ca-certificates \
        gnupg \
        certbot \
        python3-certbot-nginx \
        fail2ban \
        logrotate \
        policycoreutils-python-utils \
        tar \
        which
fi

# Install Docker
log "Installing Docker..."
if ! command -v docker &> /dev/null; then
    curl -fsSL https://get.docker.com | sh
    systemctl enable docker
    systemctl start docker
    log "Docker installed successfully"
else
    log "Docker is already installed"
fi

# Install Docker Compose
log "Installing Docker Compose..."
if ! command -v docker-compose &> /dev/null; then
    DOCKER_COMPOSE_VERSION=$(curl -s https://api.github.com/repos/docker/compose/releases/latest | grep 'tag_name' | cut -d\" -f4)
    curl -L "https://github.com/docker/compose/releases/download/$DOCKER_COMPOSE_VERSION/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
    chmod +x /usr/local/bin/docker-compose
    log "Docker Compose installed successfully"
else
    log "Docker Compose is already installed"
fi

# Create deployment user if it doesn't exist
if ! id "$DEPLOY_USER" &>/dev/null; then
    log "Creating deployment user: $DEPLOY_USER"
    useradd -m -s /bin/bash "$DEPLOY_USER"
    usermod -aG docker "$DEPLOY_USER"
    
    # Add to admin group (sudo for Ubuntu/Debian, wheel for CentOS/RHEL)
    if [[ "$PKG_MANAGER" == "apt" ]]; then
        usermod -aG sudo "$DEPLOY_USER"
        log "Added $DEPLOY_USER to sudo group"
    elif [[ "$PKG_MANAGER" == "dnf" ]]; then
        usermod -aG wheel "$DEPLOY_USER"
        log "Added $DEPLOY_USER to wheel group"
    fi
else
    log "User $DEPLOY_USER already exists"
    usermod -aG docker "$DEPLOY_USER"
    
    # Add to admin group (sudo for Ubuntu/Debian, wheel for CentOS/RHEL)
    if [[ "$PKG_MANAGER" == "apt" ]]; then
        usermod -aG sudo "$DEPLOY_USER" 2>/dev/null || log "User already in sudo group"
    elif [[ "$PKG_MANAGER" == "dnf" ]]; then
        usermod -aG wheel "$DEPLOY_USER" 2>/dev/null || log "User already in wheel group"
    fi
fi

# Set up SSH directory for deployment user
log "Setting up SSH access for $DEPLOY_USER"
USER_HOME="/home/$DEPLOY_USER"
mkdir -p "$USER_HOME/.ssh"
chmod 700 "$USER_HOME/.ssh"

# Copy root's authorized_keys if it exists and user doesn't have one
if [[ -f /root/.ssh/authorized_keys ]] && [[ ! -f "$USER_HOME/.ssh/authorized_keys" ]]; then
    cp /root/.ssh/authorized_keys "$USER_HOME/.ssh/authorized_keys"
    log "Copied SSH keys from root to $DEPLOY_USER"
fi

# Set proper ownership
chown -R "$DEPLOY_USER:$DEPLOY_USER" "$USER_HOME/.ssh"
chmod 600 "$USER_HOME/.ssh/authorized_keys" 2>/dev/null || true

# Configure firewall
log "Configuring firewall..."
if [[ "$FIREWALL_CMD" == "ufw" ]]; then
    # Ubuntu/Debian - UFW
    ufw --force reset
    ufw default deny incoming
    ufw default allow outgoing
    ufw allow ssh
    ufw allow 80/tcp   # HTTP
    ufw allow 443/tcp  # HTTPS
    ufw allow 8080/tcp # Application (if needed for direct access)
    ufw --force enable
elif [[ "$FIREWALL_CMD" == "firewalld" ]]; then
    # CentOS/RHEL - firewalld
    systemctl enable firewalld
    systemctl start firewalld
    
    # Configure zones and rules
    firewall-cmd --set-default-zone=public
    firewall-cmd --permanent --zone=public --add-service=ssh
    firewall-cmd --permanent --zone=public --add-service=http
    firewall-cmd --permanent --zone=public --add-service=https
    firewall-cmd --permanent --zone=public --add-port=8080/tcp  # Application
    firewall-cmd --reload
    
    log "Firewall configured with firewalld"
fi

# Configure fail2ban
log "Configuring fail2ban..."
systemctl enable fail2ban
systemctl start fail2ban

# Create application directories
log "Creating application directories..."
mkdir -p /opt/tiris
chown "$DEPLOY_USER:$DEPLOY_USER" /opt/tiris

# Create directories for data persistence
mkdir -p /opt/tiris/{data,logs,backups}/{postgres,nats,redis,app,nginx}
chown -R 1001:1001 /opt/tiris/data
chown -R 1001:1001 /opt/tiris/logs
chown -R "$DEPLOY_USER:$DEPLOY_USER" /opt/tiris/backups

# Create SSL directory
mkdir -p /opt/tiris/ssl
chown "$DEPLOY_USER:$DEPLOY_USER" /opt/tiris/ssl

# Set up log rotation for Docker
log "Setting up log rotation..."
cat > /etc/logrotate.d/docker-containers << 'EOF'
/var/lib/docker/containers/*/*.log {
    rotate 7
    daily
    compress
    size=1M
    missingok
    delaycompress
    copytruncate
}
EOF

# Configure sysctl for better performance
log "Optimizing system parameters..."
cat >> /etc/sysctl.conf << 'EOF'
# Tiris Backend optimizations
vm.max_map_count=262144
net.core.somaxconn=65535
net.ipv4.tcp_max_syn_backlog=65535
net.core.netdev_max_backlog=5000
EOF

sysctl -p

# Install Node.js (for potential frontend builds)
log "Installing Node.js..."
curl -fsSL https://deb.nodesource.com/setup_18.x | bash -
apt-get install -y nodejs

# Create basic deployment script
log "Creating deployment helper script..."
cat > /opt/tiris/deploy.sh << 'EOF'
#!/bin/bash
set -e

# Quick deployment script for Tiris Backend
cd /opt/tiris/tiris-backend

# Pull latest changes
git pull origin master

# Build and restart services
docker-compose -f docker-compose.prod.yml build
docker-compose -f docker-compose.prod.yml --env-file .env.prod up -d --force-recreate

echo "✅ Deployment completed successfully!"
EOF

chmod +x /opt/tiris/deploy.sh
chown "$DEPLOY_USER:$DEPLOY_USER" /opt/tiris/deploy.sh

# Create monitoring script
cat > /opt/tiris/monitor.sh << 'EOF'
#!/bin/bash

# System monitoring script
echo "=== System Status ==="
echo "Date: $(date)"
echo ""

echo "=== Disk Usage ==="
df -h

echo ""
echo "=== Memory Usage ==="
free -h

echo ""
echo "=== Docker Containers ==="
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

echo ""
echo "=== Service Health ==="
curl -s http://localhost:8080/health/live || echo "❌ Application health check failed"
EOF

chmod +x /opt/tiris/monitor.sh
chown "$DEPLOY_USER:$DEPLOY_USER" /opt/tiris/monitor.sh

# Set up automatic security updates
log "Configuring automatic security updates..."
if [[ "$PKG_MANAGER" == "apt" ]]; then
    # Ubuntu/Debian - unattended-upgrades
    $PKG_INSTALL unattended-upgrades
    cat > /etc/apt/apt.conf.d/20auto-upgrades << 'EOF'
APT::Periodic::Update-Package-Lists "1";
APT::Periodic::Unattended-Upgrade "1";
EOF
elif [[ "$PKG_MANAGER" == "dnf" ]]; then
    # CentOS/RHEL - dnf-automatic
    $PKG_INSTALL dnf-automatic
    sed -i 's/apply_updates = no/apply_updates = yes/' /etc/dnf/automatic.conf
    systemctl enable --now dnf-automatic.timer
    log "Automatic security updates configured with dnf-automatic"
fi

# Create backup cleanup script
cat > /opt/tiris/cleanup-backups.sh << 'EOF'
#!/bin/bash
# Remove backups older than 30 days
find /opt/tiris/backups -name "*.sql.gz" -mtime +30 -delete
find /opt/tiris/backups -name "*.sql" -mtime +7 -delete
echo "Old backups cleaned up"
EOF

chmod +x /opt/tiris/cleanup-backups.sh
chown "$DEPLOY_USER:$DEPLOY_USER" /opt/tiris/cleanup-backups.sh

# Set up cron job for backup cleanup
(crontab -u "$DEPLOY_USER" -l 2>/dev/null; echo "0 2 * * 0 /opt/tiris/cleanup-backups.sh") | crontab -u "$DEPLOY_USER" -

# Display summary
log "VPS setup completed successfully! 🎉"
echo ""
echo "=== Setup Summary ==="
echo "✅ System updated and essential packages installed"
echo "✅ Docker and Docker Compose installed"
echo "✅ User '$DEPLOY_USER' created and configured"
echo "✅ Firewall configured (ports 22, 80, 443, 8080)"
echo "✅ Application directories created"
echo "✅ Log rotation configured"
echo "✅ Security hardening applied"
echo ""
echo "=== Next Steps ==="
echo "1. Switch to deployment user: sudo su - $DEPLOY_USER"
echo "2. Clone repository: git clone <your-repo-url> /opt/tiris/tiris-backend"
echo "3. Configure environment: cp .env.prod.template .env.prod && nano .env.prod"
echo "4. Set up SSL certificate: sudo certbot certonly --standalone -d your-domain.com"
echo "5. Deploy application: cd /opt/tiris/tiris-backend && docker-compose -f docker-compose.prod.yml up -d"
echo ""
echo "=== Important Files ==="
echo "📁 Application directory: /opt/tiris/tiris-backend"
echo "📁 Data directory: /opt/tiris/data"
echo "📁 Logs directory: /opt/tiris/logs"
echo "📁 Backups directory: /opt/tiris/backups"
echo "🔧 Deploy script: /opt/tiris/deploy.sh"
echo "📊 Monitor script: /opt/tiris/monitor.sh"
echo ""
warn "Remember to:"
warn "1. Change all default passwords"
warn "2. Set up your domain DNS records"
warn "3. Configure OAuth credentials"
warn "4. Test the deployment thoroughly"
echo ""
log "VPS is ready for Tiris Backend deployment! 🚀"