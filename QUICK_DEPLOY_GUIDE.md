# Quick Deployment Guide

This guide covers the fastest ways to deploy Tiris Backend with different architecture options.

## 🚀 **One-Command Deployment**

### **Option 1: Multi-App Architecture (Recommended)**
```bash
./scripts/quick-deploy-multiapp.sh
```

**Features:**
- Professional reverse proxy with subdomain routing
- Ready for multiple applications (portal, prediction service)
- SSL/HTTPS ready
- Production-grade setup
- Access via: `backend.dev.tiris.ai`

### **Option 2: Interactive Menu**
```bash
./scripts/quick-deploy.sh
```
Choose from deployment options with guided menu.

### **Option 3: Simple Single-App**
```bash
./scripts/quick-deploy.sh --simple
```
Basic deployment for development/testing.

## 📋 **Quick Deployment Steps**

### **Multi-App Deployment (5 minutes)**

#### **Prerequisites:**
```bash
# 1. Configure DNS A records (with your DNS provider):
A Record: dev.tiris.ai          → YOUR_VPS_IP
A Record: backend.dev.tiris.ai  → YOUR_VPS_IP  
A Record: www.dev.tiris.ai      → YOUR_VPS_IP
A Record: pred.dev.tiris.ai     → YOUR_VPS_IP

# 2. Ensure Docker is running
sudo systemctl start docker
```

#### **Deploy:**
```bash
# Clone and deploy
git clone https://github.com/your-repo/tiris-backend.git
cd tiris-backend

# One-command deployment
./scripts/quick-deploy-multiapp.sh
```

#### **Validation:**
```bash
# Comprehensive testing
./scripts/validate-deployment.sh

# Quick health check
curl http://backend.dev.tiris.ai/health/live
```

## 🔒 **SSL/HTTPS Setup (Post-Deployment)**

### **Quick SSL Setup:**
```bash
# After basic deployment is running, add SSL/HTTPS support
./scripts/setup-letsencrypt.sh --domain dev.tiris.ai --email your@email.com

# Verify SSL deployment
curl https://backend.dev.tiris.ai/health
curl https://pred.dev.tiris.ai/version
```

### **SSL Setup Options:**
```bash
# Option 1: Production certificates
./scripts/setup-letsencrypt.sh --domain dev.tiris.ai --email admin@dev.tiris.ai

# Option 2: Testing certificates (no rate limits)
./scripts/setup-letsencrypt.sh --domain dev.tiris.ai --email admin@dev.tiris.ai --staging

# Option 3: Wildcard certificate (covers all subdomains)
./scripts/setup-letsencrypt.sh --domain dev.tiris.ai --wildcard --email admin@dev.tiris.ai
```

### **Linux VPS Compatibility Fix:**
If you encounter nginx startup issues on Linux VPS:
```bash
# Check Docker network gateway
GATEWAY_IP=$(docker network inspect tiris-backend-network | grep Gateway | cut -d'"' -f4)

# Fix host.docker.internal compatibility
sed -i "s/host.docker.internal/${GATEWAY_IP}/g" nginx.simple.conf

# Restart with SSL
docker-compose -f docker-compose.simple.yml --profile ssl restart nginx
```

## 🏗️ **Architecture Comparison**

| Feature | Multi-App | Simple |
|---------|-----------|---------|
| **Deployment Time** | 5 minutes | 3 minutes |
| **Subdomain Routing** | ✅ Yes | ❌ No |
| **Reverse Proxy** | ✅ Nginx | ❌ Direct |
| **SSL Ready** | ✅ Automated | ✅ Script-based |
| **Multiple Apps** | ✅ Yes | ❌ No |
| **Production Ready** | ✅ Yes | ⚠️ Dev/Test |
| **Port Access** | 80, 443 | 8080 |

## 🔍 **Post-Deployment Validation**

### **Automatic Validation:**
```bash
./scripts/validate-deployment.sh
```

**Tests performed:**
- Container health
- Network connectivity
- API functionality
- Configuration validation
- DNS resolution
- Performance metrics
- Security checks
- Resource usage

### **Manual Validation:**
```bash
# Check containers
docker ps

# Test API access
curl http://backend.dev.tiris.ai/health/live

# View logs
docker logs tiris-app-simple -f

# Check resource usage
docker stats
```

## 🔧 **Management Commands**

### **View Service Status:**
```bash
# All containers
docker ps -a

# Specific service logs
docker logs tiris-reverse-proxy -f
docker logs tiris-app-simple -f
docker logs tiris-postgres-simple -f
```

### **Restart Services:**
```bash
# Restart backend only
docker compose -f docker-compose.simple.yml --env-file .env.simple restart app

# Restart reverse proxy
cd proxy && docker compose restart

# Full restart
cd proxy && docker compose restart && cd .. && docker-compose -f docker-compose.simple.yml restart
```

### **Stop/Start Everything:**
```bash
# Stop all services
cd proxy && docker compose down
docker compose -f docker-compose.simple.yml --env-file .env.simple down

# Start all services
cd proxy && docker compose up -d
docker compose -f docker-compose.simple.yml --env-file .env.simple up -d
```

## 📊 **Monitoring & Health Checks**

### **Health Endpoints:**
```bash
# Nginx proxy health
curl http://localhost/nginx-health

# Backend API health
curl http://localhost:8080/health/live
curl http://backend.dev.tiris.ai/health/live

# Database health
docker exec tiris-postgres-simple pg_isready -U tiris_user -d tiris
```

### **Resource Monitoring:**
```bash
# Container resources
docker stats

# Disk usage
docker system df

# Network usage
docker network ls
```

### **Log Monitoring:**
```bash
# Follow all logs
docker logs -f tiris-app-simple

# View recent logs
docker logs --tail 50 tiris-app-simple

# Search logs
docker logs tiris-app-simple | grep ERROR
```

## 🔄 **Updates & Maintenance**

### **Update Application:**
```bash
# Pull latest code
git pull origin main

# Rebuild and restart
docker compose -f docker-compose.simple.yml --env-file .env.simple up -d --build app

# Validate deployment
./scripts/validate-deployment.sh
```

### **Update Configuration:**
```bash
# Update environment variables
nano .env.simple

# Restart to apply changes
docker compose -f docker-compose.simple.yml --env-file .env.simple restart app
```

### **Database Backup:**
```bash
# Create backup
docker exec tiris-postgres-simple pg_dump -U tiris_user tiris > backup_$(date +%Y%m%d).sql

# Restore backup
docker exec -i tiris-postgres-simple psql -U tiris_user tiris < backup_file.sql
```

## 🚨 **Troubleshooting**

### **Common Issues:**

#### **1. DNS Not Resolving:**
```bash
# Check DNS configuration
nslookup backend.dev.tiris.ai

# Test with direct IP
curl http://YOUR_VPS_IP/nginx-health
```

#### **2. Container Not Starting:**
```bash
# Check logs for errors
docker logs tiris-app-simple --tail 50

# Check environment variables
docker exec tiris-app-simple env | grep -E "(DB_|JWT_)"

# Restart specific container
docker compose -f docker-compose.simple.yml --env-file .env.simple restart app
```

#### **3. Database Connection Issues:**
```bash
# Test database connectivity
docker exec tiris-postgres-simple pg_isready -U tiris_user -d tiris

# Check database logs
docker logs tiris-postgres-simple --tail 20

# Reset database (⚠️ Data loss)
docker compose -f docker-compose.simple.yml --env-file .env.simple down -v
docker compose -f docker-compose.simple.yml --env-file .env.simple up -d postgres
```

#### **4. Port Conflicts:**
```bash
# Check what's using ports
sudo netstat -tulpn | grep -E ':80|:8080|:5432'

# Stop conflicting services
sudo systemctl stop apache2  # If Apache is running
sudo systemctl stop nginx    # If system Nginx is running
```

#### **5. Permission Issues:**
```bash
# Fix Docker permissions (non-root user)
sudo usermod -aG docker $USER
newgrp docker

# Or run with sudo (not recommended for production)
sudo ./scripts/quick-deploy-multiapp.sh
```

### **Emergency Rollback:**
```bash
# Quick rollback to previous version
docker compose -f docker-compose.simple.yml --env-file .env.simple down
docker tag tiris/backend:backup-$(date +%Y%m%d) tiris/backend:simple
docker compose -f docker-compose.simple.yml --env-file .env.simple up -d

# Or rollback to simple deployment
docker compose -f docker-compose.simple.yml --env-file .env.simple down
cd proxy && docker compose down
./scripts/quick-deploy.sh --simple
```

## 🔐 **Security Considerations**

### **Before Production:**
1. **Change default passwords** - Environment file should have secure generated passwords
2. **Set up SSL certificates** - Use Let's Encrypt for HTTPS
3. **Configure firewall** - Only allow necessary ports (22, 80, 443)
4. **Use non-root user** - Don't run Docker as root in production
5. **Enable monitoring** - Set up log aggregation and alerting
6. **Regular backups** - Automate database and configuration backups

### **SSL Setup (Production):**
```bash
# Install certbot
sudo apt install certbot python3-certbot-nginx

# Get certificates
sudo certbot certonly --standalone \
  -d dev.tiris.ai \
  -d backend.dev.tiris.ai \
  -d www.dev.tiris.ai \
  -d pred.dev.tiris.ai

# Update nginx configuration for SSL
# (See MULTI_APP_DEPLOYMENT.md for full SSL setup)
```

## 📈 **Adding More Applications**

### **Portal Frontend (Port 8081):**
```bash
cd tiris-portal
cp .env.template .env
# Edit .env with your configuration
docker-compose up -d

# Test
curl http://www.dev.tiris.ai
```

### **Prediction Service (Port 8082):**
```bash
cd tiris-pred
cp .env.template .env
# Edit .env with your configuration
docker-compose up -d

# Test
curl http://pred.dev.tiris.ai/health
```

## 📚 **Additional Documentation**

- **Multi-App Architecture**: `MULTI_APP_DEPLOYMENT.md`
- **Development Setup**: `README.md`
- **API Documentation**: Available at `/docs` endpoint
- **Environment Configuration**: `.env.simple.template`

## ⚡ **Quick Reference**

```bash
# Deploy multi-app architecture
./scripts/quick-deploy-multiapp.sh

# Validate deployment
./scripts/validate-deployment.sh

# View all services
docker ps

# Check health
curl http://backend.dev.tiris.ai/health/live

# View logs
docker logs tiris-app-simple -f

# Update application
git pull && docker compose -f docker-compose.simple.yml --env-file .env.simple up -d --build app

# Emergency stop
cd proxy && docker compose down && cd .. && docker compose -f docker-compose.simple.yml --env-file .env.simple down
```

Your Tiris Backend is now deployed with professional multi-application architecture! 🎉