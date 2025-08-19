# Production VPS Deployment Guide for Tiris Backend

## Overview
This guide will walk you through deploying the Tiris backend to a production VPS using Docker Compose with complete monitoring, security, and backup systems.

## Prerequisites
- VPS with supported Linux distribution:
  - **Ubuntu 20.04+** or **Debian 11+**
  - **CentOS 9 Stream**, **Rocky Linux 9**, or **AlmaLinux 9**
- Root or sudo access to the VPS
- Domain name pointing to your VPS IP
- At least 4GB RAM and 50GB storage recommended

## OS-Specific Guides
- **CentOS/RHEL Users**: See [CentOS 9 Deployment Guide](./CENTOS_DEPLOYMENT.md) for specific instructions
- **Ubuntu/Debian Users**: Follow this guide (default instructions are for Ubuntu)

## Quick Start Commands

### 1. Initial VPS Setup
```bash
# Run this on your VPS as root or with sudo
curl -fsSL https://raw.githubusercontent.com/your-repo/tiris-backend/master/scripts/vps-setup.sh | bash
```

### 2. Deploy Application
```bash
# Run this after VPS setup is complete
curl -fsSL https://raw.githubusercontent.com/your-repo/tiris-backend/master/scripts/deploy-production.sh | bash
```

## Manual Deployment Steps

### Step 1: VPS Initial Setup

**Connect to your VPS:**
```bash
ssh root@your-vps-ip
```

**Update system and install prerequisites:**
```bash
# Update package index
apt update && apt upgrade -y

# Install required packages
apt install -y curl git ufw certbot python3-certbot-nginx

# Install Docker
curl -fsSL https://get.docker.com | sh
systemctl enable docker
systemctl start docker

# Install Docker Compose
curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
chmod +x /usr/local/bin/docker-compose

# Create deployment user
useradd -m -s /bin/bash -G docker tiris
usermod -aG sudo tiris

# Set up SSH key for tiris user (copy your public key)
mkdir -p /home/tiris/.ssh
# Copy your SSH public key to /home/tiris/.ssh/authorized_keys
chown -R tiris:tiris /home/tiris/.ssh
chmod 700 /home/tiris/.ssh
chmod 600 /home/tiris/.ssh/authorized_keys
```

**Configure firewall:**
```bash
# Configure UFW firewall
ufw default deny incoming
ufw default allow outgoing
ufw allow ssh
ufw allow 80/tcp
ufw allow 443/tcp
ufw --force enable
```

### Step 2: Prepare Application Directory

**Switch to tiris user and set up application:**
```bash
# Switch to tiris user
su - tiris

# Create application directory
sudo mkdir -p /opt/tiris
sudo chown -R tiris:tiris /opt/tiris
cd /opt/tiris

# Clone the repository
git clone https://github.com/your-username/tiris-backend.git
cd tiris-backend

# Create data directories
sudo mkdir -p /opt/tiris/{data,logs,backups}/{postgres,nats,redis,app,nginx}
sudo chown -R 1001:1001 /opt/tiris/data
sudo chown -R 1001:1001 /opt/tiris/logs
sudo chown -R tiris:tiris /opt/tiris/backups
```

### Step 3: Configure Environment

**Create production environment file:**
```bash
# Copy template and edit
cp .env.prod.template .env.prod

# Generate secure secrets
echo "Generating secure secrets..."
echo "JWT_SECRET=$(openssl rand -base64 32)"
echo "REFRESH_SECRET=$(openssl rand -base64 32)"
echo "DB_PASSWORD=$(openssl rand -base64 24)"
echo "REDIS_PASSWORD=$(openssl rand -base64 24)"
echo "NATS_PASSWORD=$(openssl rand -base64 24)"

# Edit .env.prod with your specific values
nano .env.prod
```

**Required environment variables to set:**
- `DOMAIN=your-domain.com`
- `CORS_ALLOWED_ORIGINS=https://your-domain.com`
- All password fields with the generated secrets above
- OAuth credentials (Google, WeChat)

### Step 4: SSL Certificate Setup

**Get SSL certificate with Let's Encrypt:**
```bash
# Install certificate for your domain
sudo certbot certonly --standalone -d your-domain.com -d api.your-domain.com

# Create SSL directory and copy certificates
sudo mkdir -p /opt/tiris/ssl
sudo cp /etc/letsencrypt/live/your-domain.com/fullchain.pem /opt/tiris/ssl/
sudo cp /etc/letsencrypt/live/your-domain.com/privkey.pem /opt/tiris/ssl/
sudo chown -R tiris:tiris /opt/tiris/ssl

# Set up auto-renewal
sudo crontab -e
# Add this line: 0 12 * * * /usr/bin/certbot renew --quiet --pre-hook "docker-compose -f /opt/tiris/tiris-backend/docker-compose.prod.yml stop nginx" --post-hook "docker-compose -f /opt/tiris/tiris-backend/docker-compose.prod.yml start nginx"
```

### Step 5: Deploy Infrastructure Services

**Start database and supporting services:**
```bash
cd /opt/tiris/tiris-backend

# Build the application
docker-compose -f docker-compose.prod.yml build

# Start infrastructure services
docker-compose -f docker-compose.prod.yml --env-file .env.prod up -d postgres redis nats

# Wait for services to be ready
sleep 30

# Run database migrations
docker-compose -f docker-compose.prod.yml --env-file .env.prod --profile migrate run --rm migrate

# Set up NATS streams
docker-compose -f docker-compose.prod.yml --env-file .env.prod --profile setup run --rm nats-setup
```

### Step 6: Deploy Application and Proxy

**Start the application:**
```bash
# Start main application
docker-compose -f docker-compose.prod.yml --env-file .env.prod up -d app

# Start Nginx reverse proxy
docker-compose -f docker-compose.prod.yml --env-file .env.prod --profile nginx up -d nginx

# Verify all services are running
docker-compose -f docker-compose.prod.yml ps
```

### Step 7: Set Up Monitoring

**Deploy monitoring stack:**
```bash
# Start monitoring services
docker-compose -f docker-compose.monitoring.yml --env-file .env.prod up -d

# Access Grafana at https://your-domain.com:3000
# Default login: admin/admin (change immediately)
```

### Step 8: Configure Backups

**Set up automated backups:**
```bash
# Start backup service
docker-compose -f docker-compose.prod.yml --env-file .env.prod --profile backup up -d backup

# Test backup manually
docker-compose -f docker-compose.prod.yml --env-file .env.prod exec backup /scripts/backup-db.sh
```

## Verification

### Health Checks
```bash
# Check application health
curl https://your-domain.com/health/live
curl https://your-domain.com/health/ready

# Check all containers
docker-compose -f docker-compose.prod.yml ps

# Check logs
docker-compose -f docker-compose.prod.yml logs app
```

### API Testing
```bash
# Test API endpoints
curl https://your-domain.com/api/v1/health
curl -X POST https://your-domain.com/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"testpass123"}'
```

## Maintenance Commands

### Application Updates
```bash
cd /opt/tiris/tiris-backend

# Pull latest code
git pull origin master

# Rebuild and restart
docker-compose -f docker-compose.prod.yml build
docker-compose -f docker-compose.prod.yml --env-file .env.prod up -d --force-recreate app
```

### Database Maintenance
```bash
# Manual backup
docker-compose -f docker-compose.prod.yml exec postgres pg_dump -U tiris_user tiris_prod > backup_$(date +%Y%m%d).sql

# View database logs
docker-compose -f docker-compose.prod.yml logs postgres

# Connect to database
docker-compose -f docker-compose.prod.yml exec postgres psql -U tiris_user -d tiris_prod
```

### Log Management
```bash
# View application logs
docker-compose -f docker-compose.prod.yml logs -f app

# Clean up old logs
docker system prune -f

# Rotate logs
sudo logrotate -f /etc/logrotate.d/docker-containers
```

## Troubleshooting

### Common Issues

**Container won't start:**
```bash
# Check container logs
docker-compose -f docker-compose.prod.yml logs [service-name]

# Check system resources
df -h
free -m
```

**Database connection issues:**
```bash
# Check database is running
docker-compose -f docker-compose.prod.yml exec postgres pg_isready

# Check connection from app
docker-compose -f docker-compose.prod.yml exec app nc -zv postgres 5432
```

**SSL certificate issues:**
```bash
# Check certificate status
sudo certbot certificates

# Renew certificates manually
sudo certbot renew --dry-run
```

### Emergency Procedures

**Rollback deployment:**
```bash
# Stop current deployment
docker-compose -f docker-compose.prod.yml down

# Switch to previous version
git checkout [previous-commit]
docker-compose -f docker-compose.prod.yml build
docker-compose -f docker-compose.prod.yml --env-file .env.prod up -d
```

**Database restore:**
```bash
# Stop application
docker-compose -f docker-compose.prod.yml stop app

# Restore from backup
docker-compose -f docker-compose.prod.yml exec postgres psql -U tiris_user -d tiris_prod < backup_file.sql

# Restart application
docker-compose -f docker-compose.prod.yml start app
```

## Security Considerations

- Change all default passwords immediately
- Use strong, unique secrets for JWT and database passwords
- Keep SSL certificates up to date
- Regularly update Docker images and system packages
- Monitor logs for suspicious activity
- Use SSH keys instead of passwords
- Configure fail2ban for additional security

## Performance Optimization

- Monitor resource usage with provided Grafana dashboards
- Scale application replicas if needed: `APP_REPLICAS=3` in .env.prod
- Optimize database with regular VACUUM and ANALYZE
- Use Redis for caching frequently accessed data
- Configure rate limiting appropriately

## Support

For issues and questions:
1. Check the logs: `docker-compose logs [service]`
2. Review this documentation
3. Check GitHub issues
4. Contact the development team

---

**Important**: Replace `your-domain.com` and `your-username` with your actual domain and GitHub username throughout this guide.