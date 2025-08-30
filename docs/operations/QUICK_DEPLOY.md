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

## 🔒 SSL/HTTPS Setup with Let's Encrypt

The simple deployment supports production-ready SSL/HTTPS with Let's Encrypt certificates!

### Option 1: Quick SSL Deployment (Recommended)

Deploy with HTTPS using the interactive menu:

```bash
./scripts/quick-deploy.sh
# Choose option 4: "Simple Single-App Deployment (HTTPS)"
# Enter your domain name and email when prompted
# Choose between single domain or wildcard certificate
```

Or use the command line:

```bash
# Single domain certificate (HTTP challenge)
./scripts/quick-deploy.sh --ssl --ssl-domain yourdomain.com --ssl-email your@email.com

# Wildcard certificate (DNS challenge - covers all subdomains)
./scripts/quick-deploy.sh --ssl --ssl-domain yourdomain.com --ssl-email your@email.com --ssl-wildcard
```

This will:
- ✅ Generate Let's Encrypt SSL certificates automatically
- ✅ Deploy with nginx reverse proxy and SSL termination
- ✅ Redirect all HTTP traffic to HTTPS
- ✅ Configure CORS for your domain
- ✅ Set up automatic certificate renewal

Your app will be available at:
- `https://yourdomain.com` (HTTPS - secure)
- `http://yourdomain.com` (redirects to HTTPS)

### Option 2: Manual SSL Setup

For more control over the SSL setup:

```bash
# 1. Setup Let's Encrypt certificates
./scripts/setup-letsencrypt.sh --domain yourdomain.com --email your@email.com

# 2. Update environment for HTTPS
cp .env.simple.template .env.simple
# CORS will be updated automatically

# 3. Deploy with SSL profile
docker-compose -f docker-compose.simple.yml --env-file .env.simple --profile ssl up -d
```

**Certificate Options:**
```bash
# Single domain certificate (HTTP challenge)
./scripts/setup-letsencrypt.sh --domain yourdomain.com --email your@email.com

# Wildcard certificate - covers ALL subdomains (DNS challenge)
./scripts/setup-letsencrypt.sh --domain yourdomain.com --wildcard --email your@email.com

# Multiple specific subdomains (HTTP challenge) 
./scripts/setup-letsencrypt.sh --domain yourdomain.com --additional-domains api.yourdomain.com,www.yourdomain.com --email your@email.com
```

### Prerequisites for SSL

Before setting up SSL, ensure:
1. **Domain is configured**: DNS A record points to your server's IP
2. **Ports are open**: 80 (HTTP) and 443 (HTTPS)
3. **Server access**: SSH access to your production server
4. **No conflicting services**: Nothing else using port 80/443

### SSL Features

✅ **Production-ready certificates** from Let's Encrypt
✅ **Automatic renewal** (runs daily via cron)
✅ **Modern SSL configuration** (TLS 1.2/1.3)
✅ **HTTP to HTTPS redirects**
✅ **HSTS security headers**
✅ **OCSP stapling** for better performance

### Wildcard SSL with GoDaddy DNS

If you use GoDaddy as your DNS provider, you can get a wildcard certificate that covers all subdomains:

#### Step-by-Step Process:

1. **Start the wildcard certificate generation:**
```bash
./scripts/setup-letsencrypt.sh --domain yourdomain.com --wildcard --email your@email.com
```

2. **Add DNS TXT record in GoDaddy:**
   - Certbot will display a TXT record to add (e.g., `_acme-challenge.yourdomain.com`)
   - Log into your GoDaddy account → DNS Management
   - Click "Add Record" → Choose "TXT" record type
   - **Name:** `_acme-challenge` (without the domain part)
   - **Value:** The long string provided by certbot
   - **TTL:** 1 hour (3600 seconds)
   - Click "Save"

3. **Wait for DNS propagation:**
   - Wait 1-2 minutes for the record to propagate
   - You can verify with: `nslookup -q=TXT _acme-challenge.yourdomain.com`

4. **Continue in certbot:**
   - Press Enter in the certbot terminal to continue verification
   - Certbot will verify the TXT record and issue your wildcard certificate

5. **Certificate covers unlimited subdomains:**
   - `yourdomain.com` ✅
   - `api.yourdomain.com` ✅  
   - `www.yourdomain.com` ✅
   - `backend.yourdomain.com` ✅
   - `anything.yourdomain.com` ✅

#### Benefits of Wildcard Certificate:
- ✅ **One certificate** covers unlimited subdomains
- ✅ **No regeneration needed** when adding new subdomains
- ✅ **Cost effective** (Let's Encrypt is free)
- ✅ **Future-proof** for multi-app expansion

### SSL Troubleshooting

**Domain not resolving?**
```bash
# Test DNS resolution
nslookup yourdomain.com

# Test from server
curl -I http://yourdomain.com/nginx-health
```

**Certificate generation failed?**
```bash
# Check if port 80 is accessible
sudo netstat -tulpn | grep :80

# Test with staging certificates first
./scripts/setup-letsencrypt.sh --domain yourdomain.com --email your@email.com --staging

# Check certbot logs
sudo tail -f /var/log/letsencrypt/letsencrypt.log
```

**HTTPS not working?**
```bash
# Check SSL container status
docker ps | grep nginx

# Check certificates exist
sudo ls -la /etc/letsencrypt/live/yourdomain.com/

# Check logs
docker logs tiris-nginx-simple

# Test SSL connection
openssl s_client -connect yourdomain.com:443
```

## 🛠️ Management Commands

### Standard Deployment

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

### SSL Deployment

```bash
# View logs (including nginx)
docker-compose -f docker-compose.simple.yml --profile ssl logs -f

# Restart SSL services
docker-compose -f docker-compose.simple.yml --profile ssl restart

# Stop SSL deployment
docker-compose -f docker-compose.simple.yml --profile ssl down

# Update and restart SSL deployment
git pull origin master
docker-compose -f docker-compose.simple.yml build
docker-compose -f docker-compose.simple.yml --env-file .env.simple --profile ssl up -d

# Renew SSL certificates (automatic via cron, manual if needed)
sudo docker run --rm -v /etc/letsencrypt:/etc/letsencrypt -v /var/www/certbot:/var/www/certbot certbot/certbot:latest renew
docker-compose -f docker-compose.simple.yml --profile ssl restart nginx
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

### Standard HTTP Deployment

Once deployed, test your API:

```bash
# Health checks
curl http://localhost:8080/health/live     # Should return "OK"
curl http://localhost:8080/health/ready    # Should return readiness status

# API endpoints (adjust based on your routes)
curl http://localhost:8080/api/v1/health   # Your API health
curl http://localhost:8080/metrics         # Prometheus metrics
```

### SSL/HTTPS Deployment

For SSL deployments with Let's Encrypt, test using HTTPS:

```bash
# Replace yourdomain.com with your actual domain

# Health checks (HTTPS)
curl https://yourdomain.com/health/live      # Should return "OK"
curl https://yourdomain.com/health/ready     # Should return readiness status

# Test HTTP redirect to HTTPS
curl -I http://yourdomain.com/               # Should return 301 redirect

# Nginx health check
curl https://yourdomain.com/nginx-health     # Should return "healthy"

# API endpoints via HTTPS proxy
curl https://yourdomain.com/api/v1/health    # Your API health
curl https://yourdomain.com/metrics          # Prometheus metrics

# Verify SSL certificate (shows Let's Encrypt cert details)
openssl s_client -connect yourdomain.com:443 -servername yourdomain.com < /dev/null

# Check certificate expiry
echo | openssl s_client -connect yourdomain.com:443 2>/dev/null | openssl x509 -noout -dates
```

**Note:** With Let's Encrypt certificates, you don't need the `-k` flag as they are trusted by browsers.

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
- ❌ Advanced security features

**✅ SSL/HTTPS support is now available!** See SSL setup section below.

## ⬆️ Upgrade to Production

When you're ready for production features:

```bash
# Use the full deployment
./deployment/scripts/deploy-production.sh

# Or see full documentation
cat deployment/docs/PRODUCTION_DEPLOYMENT.md
```

### Future Multi-App Extensibility

The current SSL architecture is designed to easily extend to multiple applications (tiris-portal, tiris-pred) later:

**Extensible Components:**
- ✅ **Let's Encrypt Script** - Already supports `--additional-domains` for multiple subdomains
- ✅ **Docker Network** - Shared network allows additional services to connect
- ✅ **SSL Certificate** - Single certificate can cover multiple subdomains
- ✅ **Nginx Architecture** - Can be extended with additional server blocks

**When you're ready to add tiris-portal or tiris-pred:**
1. Generate certificates for additional subdomains
2. Add new server blocks to nginx.simple.conf  
3. Deploy additional services on the same Docker network
4. Update CORS origins to include new domains

This foundation makes multi-app expansion straightforward while keeping the current setup simple.

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