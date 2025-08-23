# Tiris Backend Operations Runbook

## Overview
This runbook provides operational procedures for managing the Tiris Backend in production. It includes deployment procedures, troubleshooting guides, maintenance tasks, and emergency procedures.

## Quick Reference

### 🚀 Deployment Commands
```bash
# VPS Setup (run once)
curl -fsSL https://raw.githubusercontent.com/your-repo/tiris-backend/master/scripts/vps-setup.sh | sudo bash

# Production Deployment
/opt/tiris/tiris-backend/scripts/deploy-production.sh

# Environment Configuration
/opt/tiris/tiris-backend/scripts/generate-production-env.sh

# Monitoring Setup
/opt/tiris/tiris-backend/scripts/setup-monitoring.sh

# Backup
/opt/tiris/tiris-backend/scripts/backup-production.sh

# Validation
/opt/tiris/tiris-backend/scripts/validate-deployment.sh
```

### 📊 System Status
```bash
# Container Status
docker-compose -f /opt/tiris/tiris-backend/docker-compose.prod.yml ps

# Application Health
curl https://your-domain.com/health/live
curl https://your-domain.com/health/ready

# System Resources
/opt/tiris/monitor.sh

# Logs
docker-compose -f /opt/tiris/tiris-backend/docker-compose.prod.yml logs -f
```

## 🔧 Day-to-Day Operations

### Starting Services
```bash
cd /opt/tiris/tiris-backend

# Start all services
docker-compose -f docker-compose.prod.yml --env-file .env.prod up -d

# Start specific service
docker-compose -f docker-compose.prod.yml --env-file .env.prod up -d [service-name]

# Start with rebuild
docker-compose -f docker-compose.prod.yml --env-file .env.prod up -d --build
```

### Stopping Services
```bash
cd /opt/tiris/tiris-backend

# Stop all services
docker-compose -f docker-compose.prod.yml down

# Stop specific service
docker-compose -f docker-compose.prod.yml stop [service-name]

# Stop and remove volumes (⚠️ DESTRUCTIVE)
docker-compose -f docker-compose.prod.yml down -v
```

### Service Management
```bash
# Restart a service
docker-compose -f docker-compose.prod.yml restart [service-name]

# Scale application
docker-compose -f docker-compose.prod.yml up -d --scale app=3

# View service logs
docker-compose -f docker-compose.prod.yml logs -f [service-name]

# Execute command in container
docker-compose -f docker-compose.prod.yml exec [service-name] [command]
```

## 📋 Monitoring & Health Checks

### Application Health
```bash
# Liveness probe
curl -f https://your-domain.com/health/live

# Readiness probe
curl -f https://your-domain.com/health/ready

# Metrics endpoint
curl https://your-domain.com/metrics

# API test
curl -X GET https://your-domain.com/api/v1/health
```

### System Monitoring
```bash
# Container resource usage
docker stats

# System resources
htop
df -h
free -h

# Network connections
netstat -tulpn | grep :8080

# Process monitoring
ps aux | grep tiris
```

### Database Health
```bash
# PostgreSQL status
docker exec tiris-postgres-prod pg_isready -U tiris_user -d tiris_prod

# Database size
docker exec tiris-postgres-prod psql -U tiris_user -d tiris_prod -c "SELECT pg_size_pretty(pg_database_size('tiris_prod'));"

# Active connections
docker exec tiris-postgres-prod psql -U tiris_user -d tiris_prod -c "SELECT count(*) FROM pg_stat_activity;"

# Database queries
docker exec tiris-postgres-prod psql -U tiris_user -d tiris_prod -c "SELECT query, state, query_start FROM pg_stat_activity WHERE state != 'idle';"
```

### Redis Health
```bash
# Redis info
docker exec tiris-redis-prod redis-cli info

# Redis memory usage
docker exec tiris-redis-prod redis-cli info memory

# Redis connected clients
docker exec tiris-redis-prod redis-cli info clients
```

## 🛠️ Maintenance Tasks

### Daily Tasks
```bash
# Check system health
/opt/tiris/monitor.sh

# Review logs for errors
docker-compose -f /opt/tiris/tiris-backend/docker-compose.prod.yml logs --since="24h" | grep -i error

# Check disk space
df -h /opt/tiris

# Verify backups
ls -la /opt/tiris/backups/$(date +%Y-%m-%d)/
```

### Weekly Tasks
```bash
# Run backup manually
/opt/tiris/tiris-backend/scripts/backup-production.sh

# Update system packages
sudo apt update && sudo apt upgrade -y

# Clean up Docker
docker system prune -f

# Check SSL certificate expiry
openssl x509 -in /opt/tiris/ssl/fullchain.pem -noout -dates

# Review monitoring alerts
curl http://localhost:9093/api/v1/alerts
```

### Monthly Tasks
```bash
# Rotate logs
sudo logrotate -f /etc/logrotate.conf

# Update Docker images
docker-compose -f /opt/tiris/tiris-backend/docker-compose.prod.yml pull
docker-compose -f /opt/tiris/tiris-backend/docker-compose.prod.yml up -d

# Database maintenance
docker exec tiris-postgres-prod psql -U tiris_user -d tiris_prod -c "VACUUM ANALYZE;"

# Security scan
docker scan tiris/backend:latest
```

## 🚨 Troubleshooting Guide

### Application Won't Start

**Symptoms:**
- Container exits immediately
- Health checks fail
- 502/503 errors from proxy

**Diagnosis:**
```bash
# Check container logs
docker-compose -f docker-compose.prod.yml logs app

# Check container status
docker-compose -f docker-compose.prod.yml ps

# Check environment variables
docker-compose -f docker-compose.prod.yml exec app env | grep -E "(DB_|REDIS_|NATS_)"

# Test database connection
docker exec tiris-postgres-prod pg_isready -U tiris_user
```

**Solutions:**
1. Verify environment configuration
2. Check database/Redis/NATS connectivity
3. Review application logs for specific errors
4. Restart dependent services
5. Check resource constraints

### Database Connection Issues

**Symptoms:**
- "Connection refused" errors
- "Too many connections" errors
- Slow database queries

**Diagnosis:**
```bash
# Check PostgreSQL status
docker exec tiris-postgres-prod pg_isready

# Check connection count
docker exec tiris-postgres-prod psql -U postgres -c "SELECT count(*) FROM pg_stat_activity;"

# Check for locks
docker exec tiris-postgres-prod psql -U postgres -c "SELECT * FROM pg_locks WHERE NOT granted;"

# Check slow queries
docker exec tiris-postgres-prod psql -U postgres -c "SELECT query, query_start FROM pg_stat_activity WHERE state = 'active' AND query_start < NOW() - INTERVAL '1 minute';"
```

**Solutions:**
1. Restart PostgreSQL container
2. Check connection pool settings
3. Kill long-running queries
4. Increase connection limits if needed
5. Check disk space for database

### High Memory/CPU Usage

**Symptoms:**
- System slowness
- Out of memory errors
- High load average

**Diagnosis:**
```bash
# System resources
htop
free -h
iostat 1 5

# Container resources
docker stats

# Application metrics
curl http://localhost:8080/metrics | grep -E "(memory|cpu)"
```

**Solutions:**
1. Scale application horizontally
2. Optimize database queries
3. Increase system resources
4. Review memory leaks in application
5. Adjust container resource limits

### SSL Certificate Issues

**Symptoms:**
- SSL certificate expired warnings
- HTTPS not working
- Certificate validation errors

**Diagnosis:**
```bash
# Check certificate expiry (updated paths)
openssl x509 -in /etc/letsencrypt/live/dev.tiris.ai/fullchain.pem -noout -dates

# Test SSL configuration
openssl s_client -connect backend.dev.tiris.ai:443 -servername backend.dev.tiris.ai

# Check certificate renewal
sudo certbot certificates

# Check nginx container SSL status
docker logs tiris-nginx-simple --tail 20 | grep -i ssl

# Verify SSL profile deployment
docker-compose -f docker-compose.simple.yml --profile ssl ps
```

**Solutions:**
1. Renew certificate manually: `./scripts/setup-letsencrypt.sh --renew`
2. Check auto-renewal cron job
3. Verify DNS configuration
4. Restart nginx container: `docker-compose -f docker-compose.simple.yml --profile ssl restart nginx`
5. Redeploy with SSL: `docker-compose -f docker-compose.simple.yml --profile ssl up -d`

### Linux VPS Docker Networking Issues

**Symptoms:**
- nginx container restarting continuously
- "host not found in upstream host.docker.internal" errors
- 502 Bad Gateway errors

**Diagnosis:**
```bash
# Check nginx container logs
docker logs tiris-nginx-simple --tail 50

# Check Docker network gateway
docker network inspect tiris-backend-network | grep Gateway

# Check if host.docker.internal is in config
grep "host.docker.internal" nginx.simple.conf
```

**Solutions:**
```bash
# 1. Fix host.docker.internal compatibility (Linux VPS)
GATEWAY_IP=$(docker network inspect tiris-backend-network | grep Gateway | cut -d'"' -f4)
sed -i "s/host.docker.internal/${GATEWAY_IP}/g" nginx.simple.conf

# 2. Replace domain placeholders if needed
sed -i "s/DOMAIN_PLACEHOLDER/dev.tiris.ai/g" nginx.simple.conf

# 3. Restart nginx container
docker-compose -f docker-compose.simple.yml --profile ssl restart nginx

# 4. Test connectivity from container to host
docker exec tiris-nginx-simple nc -zv $GATEWAY_IP 8082
```

### Network Connectivity Issues

**Symptoms:**
- Services can't communicate
- External API calls fail
- DNS resolution errors

**Diagnosis:**
```bash
# Test container networking
docker exec tiris-app-prod nc -zv postgres 5432
docker exec tiris-app-prod nc -zv redis 6379
docker exec tiris-app-prod nc -zv nats 4222

# Check DNS resolution
docker exec tiris-app-prod nslookup google.com

# Check iptables/firewall
sudo iptables -L
sudo ufw status
```

**Solutions:**
1. Restart Docker daemon
2. Recreate Docker networks
3. Check firewall rules
4. Verify container network configuration

## 🔄 Deployment Procedures

### Standard Deployment
```bash
cd /opt/tiris/tiris-backend

# 1. Pull latest code
git pull origin master

# 2. Backup current deployment
cp .env.prod .env.prod.backup.$(date +%Y%m%d)

# 3. Check for Linux VPS specific configurations
GATEWAY_IP=$(docker network inspect tiris-backend-network | grep Gateway | cut -d'"' -f4)
if grep -q "host.docker.internal" nginx.simple.conf; then
    echo "Fixing Linux VPS Docker network compatibility..."
    sed -i "s/host.docker.internal/${GATEWAY_IP}/g" nginx.simple.conf
fi

# 4. Build and deploy
docker-compose -f docker-compose.simple.yml build
docker-compose -f docker-compose.simple.yml --env-file .env.simple up -d --force-recreate

# 5. Deploy SSL version if certificates exist
if [ -d "/etc/letsencrypt/live" ] && [ "$(ls -A /etc/letsencrypt/live 2>/dev/null)" ]; then
    echo "SSL certificates found, deploying with SSL profile..."
    docker-compose -f docker-compose.simple.yml --env-file .env.simple --profile ssl up -d
fi

# 6. Validate deployment
./scripts/validate-deployment.sh 2>/dev/null || echo "Validation script not found, continuing..."

# 7. Monitor for issues
docker-compose -f docker-compose.simple.yml logs -f
```

### Rolling Update (Zero Downtime)
```bash
# 1. Scale up with new version
docker-compose -f docker-compose.prod.yml build app
docker-compose -f docker-compose.prod.yml --env-file .env.prod up -d --scale app=3

# 2. Wait for health checks
sleep 30

# 3. Scale down old instances
docker-compose -f docker-compose.prod.yml --env-file .env.prod up -d --scale app=2

# 4. Verify and finish
/opt/tiris/tiris-backend/scripts/validate-deployment.sh
```

### Emergency Rollback
```bash
# 1. Stop current deployment
docker-compose -f docker-compose.prod.yml down

# 2. Revert to previous code
git checkout [previous-commit-hash]

# 3. Restore previous environment
cp .env.prod.backup.[date] .env.prod

# 4. Deploy previous version
docker-compose -f docker-compose.prod.yml build
docker-compose -f docker-compose.prod.yml --env-file .env.prod up -d

# 5. Validate rollback
/opt/tiris/tiris-backend/scripts/validate-deployment.sh
```

## 💾 Backup & Recovery

### Manual Backup
```bash
# Run backup script
/opt/tiris/tiris-backend/scripts/backup-production.sh

# Verify backup
ls -la /opt/tiris/backups/$(date +%Y-%m-%d)/

# Test backup integrity
gzip -t /opt/tiris/backups/$(date +%Y-%m-%d)/*/database/*.sql.gz
```

### Database Recovery
```bash
# 1. Stop application
docker-compose -f docker-compose.prod.yml stop app

# 2. Create backup of current database
docker exec tiris-postgres-prod pg_dump -U tiris_user tiris_prod > current_backup.sql

# 3. Restore from backup
gunzip -c backup_file.sql.gz | docker exec -i tiris-postgres-prod psql -U tiris_user -d tiris_prod

# 4. Restart application
docker-compose -f docker-compose.prod.yml start app

# 5. Validate recovery
/opt/tiris/tiris-backend/scripts/validate-deployment.sh
```

### Configuration Recovery
```bash
# Restore environment configuration
cp /opt/tiris/backups/[date]/config/tiris_config_[timestamp].tar.gz ./
tar -xzf tiris_config_[timestamp].tar.gz

# Restore SSL certificates
sudo cp -r etc/letsencrypt/* /etc/letsencrypt/
sudo systemctl reload nginx
```

## 🔐 Security Procedures

### Security Monitoring
```bash
# Check failed login attempts
sudo journalctl -u ssh | grep "Failed password"

# Review application security logs
docker-compose -f docker-compose.prod.yml logs | grep -i "security\|auth\|unauthorized"

# Check firewall status
sudo ufw status verbose

# Review SSL configuration
nmap --script ssl-enum-ciphers -p 443 your-domain.com
```

### Security Updates
```bash
# Update system packages
sudo apt update && sudo apt upgrade -y

# Update Docker images
docker-compose -f docker-compose.prod.yml pull
docker-compose -f docker-compose.prod.yml up -d

# Rotate secrets (quarterly)
/opt/tiris/tiris-backend/scripts/generate-production-env.sh

# Update SSL certificates
sudo certbot renew
```

### Incident Response
```bash
# 1. Isolate affected systems
docker-compose -f docker-compose.prod.yml stop [affected-service]

# 2. Collect logs and evidence
docker-compose -f docker-compose.prod.yml logs > incident_logs_$(date +%Y%m%d_%H%M%S).txt

# 3. Review access logs
sudo tail -f /var/log/nginx/access.log

# 4. Check for indicators of compromise
grep -r "suspicious_pattern" /opt/tiris/logs/

# 5. Apply security patches and restart
# Follow standard deployment procedure
```

## 📞 Emergency Contacts & Escalation

### Emergency Procedures
1. **Service Down**: Follow troubleshooting guide, attempt restart
2. **Data Breach**: Stop affected services, collect logs, notify security team
3. **Performance Issues**: Check resources, scale if needed, investigate root cause
4. **SSL/Security Issues**: Renew certificates, check security configurations

### Contact Information
- **Operations Team**: ops@tiris.ai
- **Development Team**: dev@tiris.ai  
- **Security Team**: security@tiris.ai
- **On-call Engineer**: +1-XXX-XXX-XXXX

### External Services
- **Cloud Provider Support**: [Support URL]
- **Domain Registrar**: [Support URL]
- **CDN Provider**: [Support URL]

## 📚 Additional Resources

### Documentation
- [API Documentation](https://api.tiris.ai/docs)
- [Architecture Overview](./docs/architecture.md)
- [Deployment Guide](./PRODUCTION_DEPLOYMENT.md)
- [Development Setup](./README.md)

### Monitoring Dashboards
- **Grafana**: http://your-domain.com:3000
- **Prometheus**: http://your-domain.com:9090
- **Alertmanager**: http://your-domain.com:9093

### Log Locations
- **Application Logs**: `/opt/tiris/logs/app/`
- **Database Logs**: `/opt/tiris/logs/postgres/`
- **Nginx Logs**: `/opt/tiris/logs/nginx/`
- **System Logs**: `/var/log/`

---

**Important Notes:**
- Always test procedures in staging before production
- Keep this runbook updated with any infrastructure changes
- Document any incidents and lessons learned
- Regular review and update of security procedures