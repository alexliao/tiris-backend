# Tiris Backend - Quick Deploy (5 Minutes)

Get your Tiris Backend online quickly with minimal configuration. Perfect for development, testing, or getting started fast!

## 🚀 One-Command Deployment

```bash
# Clone and deploy in one go
git clone https://github.com/your-username/tiris-backend.git
cd tiris-backend
./scripts/quick-deploy.sh
```

That's it! Your application will be running at `http://localhost:8080`

## 📋 Prerequisites

- Docker and Docker Compose installed
- 2GB+ RAM available
- Port 8080 (and optionally 80) available

### Supported Operating Systems
- **Ubuntu 20.04+** or **Debian 11+**
- **CentOS 9 Stream**, **Rocky Linux 9**, or **AlmaLinux 9**

### Install Docker (Any Supported OS)
```bash
# Universal Docker installation (works on Ubuntu, CentOS, etc.)
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
# Log out and back in
```

**CentOS-specific notes**: The deployment scripts automatically handle CentOS differences (dnf vs apt, firewalld vs ufw, etc.). See [CentOS Deployment Guide](./deployment/docs/CENTOS_DEPLOYMENT.md) for details.

## 🎯 Manual Step-by-Step (if you prefer)

### 1. Setup Environment (30 seconds)
```bash
# Copy and edit environment file
cp .env.simple.template .env.simple
nano .env.simple  # Change passwords and secrets
```

### 2. Deploy (2 minutes)
```bash
# Build and start
docker-compose -f docker-compose.simple.yml --env-file .env.simple up -d

# Wait for startup
docker-compose -f docker-compose.simple.yml logs -f app
```

### 3. Verify (30 seconds)
```bash
# Test the application
curl http://localhost:8080/health/live
curl http://localhost:8080/health/ready
```

## 🔧 Configuration Options

### Basic Configuration
Edit `.env.simple` to change:
- `DB_PASSWORD` - Database password
- `JWT_SECRET` - JWT signing secret  
- `GOOGLE_CLIENT_ID` - For Google OAuth (optional)
- `GOOGLE_CLIENT_SECRET` - For Google OAuth (optional)

### Generate Secure Secrets
```bash
# Generate JWT secret
openssl rand -base64 32

# Generate database password
openssl rand -base64 16
```

## 🌐 With Nginx Reverse Proxy

If you want to use port 80 (standard HTTP):

```bash
# Deploy with proxy
docker-compose -f docker-compose.simple.yml --env-file .env.simple --profile proxy up -d
```

Your app will be available at `http://localhost` (port 80)

## 🛠️ Management Commands

```bash
# View logs
docker-compose -f docker-compose.simple.yml logs -f

# Restart application
docker-compose -f docker-compose.simple.yml restart app

# Stop everything
docker-compose -f docker-compose.simple.yml down

# Update and restart
git pull origin master
docker-compose -f docker-compose.simple.yml build
docker-compose -f docker-compose.simple.yml --env-file .env.simple up -d
```

## 🔍 Troubleshooting

### App won't start?
```bash
# Check logs
docker-compose -f docker-compose.simple.yml logs app

# Check container status
docker-compose -f docker-compose.simple.yml ps
```

### Database connection issues?
```bash
# Check database
docker-compose -f docker-compose.simple.yml logs postgres

# Test database connection
docker-compose -f docker-compose.simple.yml exec postgres pg_isready -U tiris_user
```

### Port conflicts?
```bash
# Check what's using ports
sudo netstat -tulpn | grep -E ":8080|:5432|:80"

# Change ports in .env.simple
APP_PORT=8081
DB_PORT=5433
HTTP_PORT=81
```

## 🚀 API Testing

Once deployed, test your API:

```bash
# Health checks
curl http://localhost:8080/health/live     # Should return "OK"
curl http://localhost:8080/health/ready    # Should return readiness status

# API endpoints (adjust based on your routes)
curl http://localhost:8080/api/v1/health   # Your API health
curl http://localhost:8080/metrics         # Prometheus metrics
```

## 📊 What's Included

**Services:**
- ✅ PostgreSQL with TimescaleDB
- ✅ Tiris Backend Application
- ✅ Nginx Reverse Proxy (optional)

**What's NOT included (but available in full deployment):**
- ❌ Redis (caching)
- ❌ NATS (messaging)
- ❌ Monitoring (Prometheus/Grafana)
- ❌ Automated backups
- ❌ SSL/HTTPS setup
- ❌ Advanced security features

## ⬆️ Upgrade to Production

When you're ready for production features:

```bash
# Use the full deployment
./deployment/scripts/deploy-production.sh

# Or see full documentation
cat deployment/docs/PRODUCTION_DEPLOYMENT.md
```

## 🆘 Need Help?

1. **Check logs first**: `docker-compose -f docker-compose.simple.yml logs`
2. **Verify environment**: Make sure `.env.simple` has correct values
3. **Test connectivity**: Ensure ports 8080, 5432 are available
4. **Resource check**: Ensure you have 2GB+ RAM available

## 📝 Common Use Cases

**Development:**
```bash
# Quick setup for development
./scripts/quick-deploy.sh --domain localhost --no-proxy
```

**Local testing with domain:**
```bash
# Setup with custom domain
./scripts/quick-deploy.sh --domain myapp.local --proxy
```

**Production preview:**
```bash
# Quick production-like setup
./scripts/quick-deploy.sh --domain your-domain.com --proxy
```

---

**Perfect for:** Development, Testing, Demos, Learning
**Not recommended for:** High-traffic production, Mission-critical applications

For production deployments, use the full deployment guide in `deployment/docs/`!