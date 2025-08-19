# Multi-Application VPS Deployment Guide

This guide covers deploying multiple Tiris applications on a single VPS with subdomain-based routing.

## Architecture Overview

```
Internet → dev.tiris.ai:80 (Nginx Reverse Proxy)
├── backend.dev.tiris.ai → tiris-backend:8080 (API Backend)
├── www.dev.tiris.ai     → tiris-portal:8081  (Frontend Portal)
└── pred.dev.tiris.ai    → tiris-pred:8082    (Prediction Service)
```

## Directory Structure

```
/vps-apps/
├── proxy/                          # Reverse proxy (port 80/443)
│   ├── docker-compose.yml
│   ├── nginx.conf
│   └── ssl/                        # SSL certificates (created later)
├── tiris-backend/                  # API Backend (port 8080)
│   ├── docker-compose.simple.yml
│   ├── .env.simple
│   └── [existing backend files]
├── tiris-portal/                   # Frontend Portal (port 8081)
│   ├── docker-compose.yml
│   ├── .env
│   └── [portal application files]
└── tiris-pred/                     # Prediction Service (port 8082)
    ├── docker-compose.yml
    ├── .env
    └── [prediction service files]
```

## Prerequisites

### 1. DNS Configuration
Configure these A records with your DNS provider:

```
A Record: dev.tiris.ai          → YOUR_VPS_IP
A Record: backend.dev.tiris.ai  → YOUR_VPS_IP  
A Record: www.dev.tiris.ai      → YOUR_VPS_IP
A Record: pred.dev.tiris.ai     → YOUR_VPS_IP
```

### 2. VPS Requirements
- **RAM**: 4GB+ recommended for all services
- **Storage**: 50GB+ for databases and applications
- **OS**: CentOS 9 / Ubuntu 20.04+
- **Docker**: Latest version installed
- **Docker Compose**: v2.0+ installed

## Deployment Steps

### Phase 1: Deploy Reverse Proxy

1. **Start the reverse proxy**:
```bash
cd /path/to/tiris-backend/proxy
docker-compose up -d
```

2. **Verify proxy is running**:
```bash
docker ps | grep tiris-reverse-proxy
docker logs tiris-reverse-proxy
```

3. **Test proxy health**:
```bash
curl http://localhost/nginx-health
# Should return: healthy
```

### Phase 2: Deploy Backend (Port 8080)

1. **Prepare backend environment**:
```bash
cd /path/to/tiris-backend
# Ensure .env.simple exists with proper configuration
```

2. **Deploy backend without proxy**:
```bash
docker-compose -f docker-compose.simple.yml down
docker-compose -f docker-compose.simple.yml up -d
```

3. **Verify backend is accessible**:
```bash
curl http://localhost:8080/health/live
# Should return backend health status
```

4. **Test via subdomain** (requires DNS setup):
```bash
curl http://backend.dev.tiris.ai/health/live
```

### Phase 3: Deploy Portal (Port 8081) - When Ready

1. **Prepare portal application**:
```bash
cd tiris-portal
cp .env.template .env
# Edit .env with your configuration
```

2. **Deploy portal**:
```bash
docker-compose up -d
```

3. **Test portal**:
```bash
curl http://localhost:8081
curl http://www.dev.tiris.ai  # Via subdomain
```

### Phase 4: Deploy Prediction Service (Port 8082) - When Ready

1. **Prepare prediction service**:
```bash
cd tiris-pred
cp .env.template .env
# Edit .env with your configuration
```

2. **Deploy prediction service**:
```bash
docker-compose up -d
```

3. **Test prediction service**:
```bash
curl http://localhost:8082/health
curl http://pred.dev.tiris.ai/health  # Via subdomain
```

## Management Commands

### Start All Services
```bash
# Start in correct order
cd proxy && docker-compose up -d
cd ../tiris-backend && docker-compose -f docker-compose.simple.yml up -d
cd ../tiris-portal && docker-compose up -d      # When ready
cd ../tiris-pred && docker-compose up -d        # When ready
```

### Stop All Services
```bash
# Stop in reverse order
cd tiris-pred && docker-compose down
cd ../tiris-portal && docker-compose down
cd ../tiris-backend && docker-compose -f docker-compose.simple.yml down
cd ../proxy && docker-compose down
```

### Check Service Status
```bash
# View all containers
docker ps -a

# Check specific service logs
docker logs tiris-reverse-proxy -f
docker logs tiris-app-simple -f
docker logs tiris-portal -f
docker logs tiris-pred -f

# Check resource usage
docker stats
```

### Update Single Service
```bash
# Example: Update backend
cd tiris-backend
git pull origin main
docker-compose -f docker-compose.simple.yml down
docker-compose -f docker-compose.simple.yml up -d --build
```

## SSL/HTTPS Setup (Optional)

### Install Certbot
```bash
sudo apt install certbot python3-certbot-nginx  # Ubuntu
sudo dnf install certbot python3-certbot-nginx  # CentOS
```

### Generate Certificates
```bash
sudo certbot certonly --standalone -d dev.tiris.ai -d backend.dev.tiris.ai -d www.dev.tiris.ai -d pred.dev.tiris.ai
```

### Update Nginx for HTTPS
```bash
# Copy certificates to proxy/ssl/
sudo cp /etc/letsencrypt/live/dev.tiris.ai/* proxy/ssl/

# Update nginx.conf to include SSL configuration
# Restart proxy
cd proxy && docker-compose restart
```

## Troubleshooting

### Common Issues

1. **Port Conflicts**:
```bash
# Check what's using ports
netstat -tulpn | grep -E ':80|:8080|:8081|:8082'
sudo lsof -i :80
```

2. **DNS Not Resolving**:
```bash
# Test DNS resolution
nslookup backend.dev.tiris.ai
dig backend.dev.tiris.ai
```

3. **Container Communication Issues**:
```bash
# Check networks
docker network ls
docker network inspect tiris-proxy-network

# Test container connectivity
docker exec tiris-reverse-proxy ping host.docker.internal
```

4. **Application Not Starting**:
```bash
# Check logs for errors
docker logs tiris-app-simple --tail 50
docker logs tiris-reverse-proxy --tail 50

# Check environment variables
docker exec tiris-app-simple env | grep -E "(DB_|JWT_)"
```

### Performance Monitoring

```bash
# Resource usage
docker stats --no-stream

# Disk usage
docker system df
df -h

# Log sizes
du -sh proxy/logs/ tiris-backend/logs/
```

## Security Considerations

1. **Firewall Configuration**:
```bash
# Only allow necessary ports
sudo ufw allow 22    # SSH
sudo ufw allow 80    # HTTP
sudo ufw allow 443   # HTTPS
sudo ufw enable
```

2. **Database Security**:
- Use strong passwords in .env files
- Limit database access to application containers only
- Regular backups

3. **Application Security**:
- Keep Docker images updated
- Use non-root users in containers
- Implement proper authentication/authorization

## Backup Strategy

### Database Backup
```bash
# Backup main database
docker exec tiris-postgres-simple pg_dump -U tiris_user tiris > backup_$(date +%Y%m%d).sql

# Restore backup
docker exec -i tiris-postgres-simple psql -U tiris_user tiris < backup_file.sql
```

### Configuration Backup
```bash
# Backup environment files and configs
tar -czf config_backup_$(date +%Y%m%d).tar.gz \
  proxy/nginx.conf \
  tiris-backend/.env.simple \
  tiris-portal/.env \
  tiris-pred/.env
```

## Next Steps

1. **Deploy Portal Application**: Replace placeholder with actual frontend app
2. **Deploy Prediction Service**: Implement ML/AI prediction endpoints  
3. **Set up SSL/HTTPS**: Configure Let's Encrypt certificates
4. **Implement Monitoring**: Add Prometheus, Grafana, or similar
5. **Set up CI/CD**: Automate deployments with GitHub Actions
6. **Load Balancing**: Scale individual services as needed

## Support

For issues with this deployment setup:
1. Check logs using commands above
2. Verify DNS configuration
3. Confirm all environment variables are set
4. Test individual services on their direct ports first
5. Check Docker network connectivity