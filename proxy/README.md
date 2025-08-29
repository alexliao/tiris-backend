# ⚠️ REFERENCE ONLY - NOT USED IN CURRENT DEPLOYMENT

**This proxy configuration is a reference implementation and is NOT active in the current deployment.**

## Current Active Setup
- **Deployment**: `docker-compose.simple.yml` with `--profile ssl`
- **Nginx Config**: `nginx.simple.conf` (multi-app + HTTPS/SSL)
- **Container**: `tiris-nginx-simple`
- **Status**: ✅ Working with full SSL support

## This Folder (Reference Only)
- **Purpose**: Alternative centralized proxy architecture
- **Config**: `nginx.conf` (HTTP-only, no SSL)
- **Status**: ❌ Not used, kept for reference

---

# Tiris Reverse Proxy (Reference Implementation)

Central reverse proxy for routing traffic to multiple Tiris applications based on subdomain.

## Overview

This Nginx reverse proxy handles all incoming traffic on port 80/443 and routes requests to the appropriate backend service based on the subdomain:

- `backend.dev.tiris.ai` → `tiris-backend:8080` (API Backend)
- `www.dev.tiris.ai` → `tiris-portal:8081` (Frontend Portal)
- `pred.dev.tiris.ai` → `tiris-pred:8082` (Prediction Service)

## Quick Start

```bash
# Start the reverse proxy
docker-compose up -d

# Check status
docker logs tiris-reverse-proxy -f

# Test health
curl http://localhost/nginx-health
```

## Configuration

### Adding New Applications

1. **Add upstream** in `nginx.conf`:
```nginx
upstream new-service {
    server 172.20.0.1:8083;  # Use Docker network gateway IP for Linux VPS
    keepalive 32;
}
```

2. **Add server block**:
```nginx
server {
    listen 80;
    server_name new.dev.tiris.ai;
    
    location / {
        proxy_pass http://new-service;
        # ... standard proxy headers
    }
}
```

3. **Restart proxy**:
```bash
docker-compose restart
```

### SSL Configuration

1. **Create SSL directory**:
```bash
mkdir -p ssl
```

2. **Add certificates**:
```bash
# Copy Let's Encrypt certificates
sudo cp /etc/letsencrypt/live/dev.tiris.ai/* ssl/
```

3. **Update nginx.conf**:
```nginx
server {
    listen 443 ssl;
    ssl_certificate /etc/nginx/ssl/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/privkey.pem;
    # ... rest of configuration
}
```

## Monitoring

```bash
# View access logs
docker exec tiris-reverse-proxy tail -f /var/log/nginx/access.log

# View error logs  
docker exec tiris-reverse-proxy tail -f /var/log/nginx/error.log

# Check upstream status
curl -s http://localhost/nginx-health
```

## Troubleshooting

### Common Issues

1. **502 Bad Gateway**: Backend service not running
   - Check if target service is running: `docker ps`
   - Test direct access: `curl http://localhost:8080`

2. **404 Not Found**: Subdomain not configured
   - Verify DNS records point to VPS IP
   - Check nginx.conf has server block for subdomain

3. **Connection Refused**: Network connectivity
   - Verify Docker network gateway IP resolves (typically `172.20.0.1` on Linux VPS)
   - Check Docker network configuration

### Debug Commands

```bash
# Test nginx configuration
docker exec tiris-reverse-proxy nginx -t

# Reload nginx (without restart)
docker exec tiris-reverse-proxy nginx -s reload

# Check Docker network gateway (Linux VPS)
docker network inspect tiris-backend-network | grep Gateway

# Test upstream connectivity
docker exec tiris-reverse-proxy curl -s http://172.20.0.1:8080/health/live

# Test network connectivity to host
docker exec tiris-reverse-proxy nc -zv 172.20.0.1 8080
```