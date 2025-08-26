# Production Update Automation Plan

## Overview

This document outlines the automated production update strategy for Tiris Backend, leveraging existing infrastructure to provide reliable, safe, and efficient deployment updates.

## Current Infrastructure Assessment

### Existing Components
- ✅ **GitHub Actions Release Workflow** - Automated builds, Docker images, Helm charts
- ✅ **Docker Compose Production Setup** - Full production stack with monitoring
- ✅ **Quick Deploy Scripts** - Multi-app and simple deployment options
- ✅ **Health Check Systems** - Validation and monitoring capabilities
- ✅ **Multi-Platform Support** - Linux, macOS, Windows binaries

### Architecture
- **Multi-App Architecture**: Professional reverse proxy with subdomain routing
- **Production Stack**: TimescaleDB, NATS, Redis, Nginx load balancer
- **Container Orchestration**: Docker Compose with health checks and resource limits
- **Monitoring**: Grafana dashboards and automated backups

## Update Strategies

### 1. Automated Release Pipeline (Primary Method)

**Trigger Methods:**
```bash
# Method 1: Git tag push (automatic)
git tag v1.2.3
git push origin v1.2.3

# Method 2: Manual workflow dispatch
# Use GitHub UI to trigger release with custom tag
```

**Automated Process:**
1. **Build Phase**
   - Creates release binaries for multiple platforms (Linux, macOS, Windows)
   - Builds multi-platform Docker images (amd64, arm64)
   - Packages Helm charts for Kubernetes deployment
   - Generates checksums and security artifacts

2. **Deployment Artifacts**
   - GitHub Container Registry: `ghcr.io/alexliao/tiris-backend:v1.2.3`
   - Docker Hub: `tiris/backend:v1.2.3` (if configured)
   - Release binaries with checksums
   - Helm chart packages

3. **Notification**
   - GitHub release creation with changelog
   - Artifact summary and download links

### 2. Hot Fix Updates (Secondary Method)

**Use Case**: Critical fixes that need immediate deployment

```bash
# On production server
cd /opt/tiris/tiris-backend

# Pull latest changes
git pull origin main

# Quick rebuild and restart
docker compose -f docker-compose.prod.yml build app
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d --force-recreate app

# Validate deployment
./scripts/validate-deployment.sh
```

### 3. Full Production Deployment (Major Releases)

**Use Case**: Major version updates, infrastructure changes, or new production instances

```bash
# Complete production setup
curl -fsSL https://raw.githubusercontent.com/alexliao/tiris-backend/master/scripts/deploy-production.sh | bash

# Or manual deployment
./scripts/quick-deploy-multiapp.sh
```

## Safety Measures & Validation

### Pre-Deployment Checks
1. **Automated Testing**
   - Unit tests pass in CI/CD
   - Integration tests complete successfully
   - Security scans pass

2. **Environment Validation**
   - Database connectivity verified
   - Required secrets present
   - SSL certificates valid

### Post-Deployment Validation
1. **Health Check Sequence**
   ```bash
   # Automated validation script
   ./scripts/validate-deployment.sh
   
   # Manual health checks
   curl https://backend.tiris.ai/health/live
   curl https://backend.tiris.ai/health/ready
   ```

2. **Service Verification**
   ```bash
   # Container status
   docker compose -f docker-compose.prod.yml ps
   
   # Resource usage
   docker stats
   
   # Log verification
   docker compose -f docker-compose.prod.yml logs app --tail 50
   ```

### Rollback Procedures
1. **Quick Rollback**
   ```bash
   # Stop current deployment
   docker compose -f docker-compose.prod.yml down
   
   # Revert to previous version
   git checkout [previous-commit-or-tag]
   docker compose -f docker-compose.prod.yml build
   docker compose -f docker-compose.prod.yml --env-file .env.prod up -d
   ```

2. **Database Rollback**
   ```bash
   # Stop application
   docker compose -f docker-compose.prod.yml stop app
   
   # Restore from backup
   docker compose -f docker-compose.prod.yml exec postgres psql -U tiris_user -d tiris_prod < backup_file.sql
   
   # Restart application
   docker compose -f docker-compose.prod.yml start app
   ```

## Implementation Phases

### Phase 1: CI/CD Pipeline Setup
**Duration**: 1-2 hours

1. **Configure GitHub Repository Settings**
   - Enable GitHub Actions
   - Set up container registry permissions
   - Configure branch protection rules

2. **Environment Secrets Setup**
   ```
   Required GitHub Secrets:
   - DOCKER_USERNAME (optional, for Docker Hub)
   - DOCKER_PASSWORD (optional, for Docker Hub)
   - PRODUCTION_SERVER_HOST
   - PRODUCTION_SSH_KEY
   ```

3. **Verify Release Workflow**
   - Create test tag: `git tag v0.1.0-test`
   - Verify artifacts generation
   - Test Docker image builds

### Phase 2: Production Server Preparation
**Duration**: 2-3 hours

1. **Server Prerequisites**
   ```bash
   # Install required components
   sudo apt update && sudo apt upgrade -y
   sudo apt install -y docker.io docker-compose-plugin curl git
   sudo systemctl enable docker && sudo systemctl start docker
   
   # Create deployment user
   sudo useradd -m -s /bin/bash -G docker tiris
   sudo usermod -aG sudo tiris
   ```

2. **SSL Certificate Setup**
   ```bash
   # Install certbot
   sudo apt install -y certbot python3-certbot-nginx
   
   # Get certificates for your domain
   sudo certbot certonly --standalone -d backend.tiris.ai
   ```

3. **Environment Configuration**
   ```bash
   # Create production environment file
   cp .env.prod.template .env.prod
   
   # Generate secure secrets
   JWT_SECRET=$(openssl rand -base64 32)
   REFRESH_SECRET=$(openssl rand -base64 32)
   DB_PASSWORD=$(openssl rand -base64 24)
   ```

### Phase 3: Automated Update Integration
**Duration**: 1-2 hours

1. **Deployment Webhook Setup**
   - Configure GitHub webhook to trigger production updates
   - Set up secure endpoint for deployment triggers
   - Implement deployment status reporting

2. **Monitoring Integration**
   ```bash
   # Deploy monitoring stack
   docker compose -f docker-compose.monitoring.yml --env-file .env.prod up -d
   
   # Access Grafana at https://backend.tiris.ai:3000
   # Configure alerts for deployment events
   ```

3. **Backup Automation**
   ```bash
   # Enable automated backups
   docker compose -f docker-compose.prod.yml --env-file .env.prod --profile backup up -d
   
   # Verify backup schedule
   docker compose -f docker-compose.prod.yml exec backup crontab -l
   ```

## Update Workflow Examples

### Routine Feature Update
```bash
# Developer workflow
git add .
git commit -m "Add new trading feature"
git push origin main

# Create release
git tag v1.2.3
git push origin v1.2.3

# GitHub Actions automatically:
# 1. Runs tests
# 2. Builds Docker images
# 3. Creates GitHub release
# 4. Notifies via configured channels

# Production server update
ssh tiris@your-server.com
cd /opt/tiris/tiris-backend
docker compose -f docker-compose.prod.yml pull
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d app
./scripts/validate-deployment.sh
```

### Emergency Hot Fix
```bash
# Critical bug fix
git add .
git commit -m "Fix critical authentication bug"
git push origin main

# Immediate production update (skip release process)
ssh tiris@your-server.com
cd /opt/tiris/tiris-backend
git pull origin main
docker compose -f docker-compose.prod.yml build app
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d --force-recreate app

# Validate fix
curl -f https://backend.tiris.ai/health/live
./scripts/validate-deployment.sh

# Create release tag afterwards
git tag v1.2.4-hotfix
git push origin v1.2.4-hotfix
```

### Major Version Release
```bash
# Complete production deployment
git tag v2.0.0
git push origin v2.0.0

# On production server
cd /opt/tiris
rm -rf tiris-backend-backup
mv tiris-backend tiris-backend-backup

# Fresh deployment
git clone https://github.com/alexliao/tiris-backend.git
cd tiris-backend
cp ../tiris-backend-backup/.env.prod .
./scripts/quick-deploy-multiapp.sh

# Verify and cleanup
./scripts/validate-deployment.sh
rm -rf ../tiris-backend-backup
```

## Monitoring & Alerting

### Health Check Endpoints
- **Liveness**: `GET /health/live` - Basic service health
- **Readiness**: `GET /health/ready` - Service ready for traffic
- **Metrics**: `GET /metrics` - Prometheus metrics endpoint

### Key Metrics to Monitor
- **Application Health**: Response times, error rates, request volume
- **Infrastructure Health**: CPU, memory, disk usage, network connectivity
- **Database Performance**: Connection pool status, query performance
- **Message Queue**: NATS stream status, message processing rates

### Alert Conditions
- Service downtime > 30 seconds
- Error rate > 5% for 5 minutes
- Database connection failures
- Disk space < 20%
- Memory usage > 80%

## Security Considerations

### Deployment Security
- Use SSH keys for server access (no passwords)
- Rotate secrets regularly (JWT, database passwords)
- Keep SSL certificates up to date
- Run services with minimal privileges (no-new-privileges flag)

### Container Security
- Use official, minimal base images
- Regular security scans of Docker images
- Resource limits on all containers
- Network isolation with custom Docker networks

### Access Control
- Limit production server access to essential personnel
- Use bastion hosts for database access
- Audit all production changes
- Implement proper firewall rules (UFW configuration)

## Disaster Recovery

### Backup Strategy
- **Automated Daily Backups**: Database dumps with 7-day retention
- **Configuration Backup**: Environment files and SSL certificates
- **Code Repository**: Git-based version control with multiple remotes

### Recovery Procedures
1. **Service Restoration**
   ```bash
   # Restore from backup
   docker compose -f docker-compose.prod.yml --profile backup run --rm backup-restore
   
   # Restart services
   docker compose -f docker-compose.prod.yml --env-file .env.prod up -d
   ```

2. **Data Recovery**
   ```bash
   # Database restore
   docker compose -f docker-compose.prod.yml exec postgres psql -U tiris_user -d tiris_prod < /backups/latest.sql
   
   # Verify data integrity
   docker compose -f docker-compose.prod.yml exec app /app/scripts/verify-data.sh
   ```

## Performance Optimization

### Scaling Options
- **Horizontal Scaling**: Increase `APP_REPLICAS` in .env.prod
- **Database Optimization**: Connection pooling, query optimization
- **Caching Strategy**: Redis integration for frequent queries
- **CDN Integration**: Static asset delivery optimization

### Resource Management
- Monitor container resource usage
- Adjust memory/CPU limits based on actual usage
- Implement connection pooling for databases
- Use appropriate logging levels in production

## Maintenance Schedule

### Regular Maintenance Tasks
- **Weekly**: Review logs and performance metrics
- **Monthly**: Update system packages and Docker images
- **Quarterly**: Review and rotate secrets
- **Annually**: SSL certificate renewal (automated via Let's Encrypt)

### Planned Maintenance Windows
- **Standard Updates**: During low-traffic hours (2-4 AM UTC)
- **Major Releases**: Scheduled maintenance windows with user notification
- **Emergency Updates**: Immediate deployment with post-update communication

## Success Metrics

### Deployment Metrics
- **Deployment Frequency**: Target 2-3 releases per month
- **Lead Time**: Code commit to production < 2 hours
- **Mean Time to Recovery**: Service restoration < 15 minutes
- **Deployment Success Rate**: > 98% successful deployments

### System Reliability
- **Uptime**: > 99.9% availability
- **Response Time**: < 200ms average API response time
- **Error Rate**: < 0.1% application error rate
- **Recovery Time**: < 5 minutes for automated recovery

## Conclusion

This production update automation plan leverages the existing robust infrastructure of Tiris Backend while adding automated, safe, and reliable update mechanisms. The multi-tiered approach provides flexibility for different types of updates while maintaining high availability and system reliability.

The plan emphasizes safety through comprehensive validation, easy rollback procedures, and thorough monitoring. Implementation can be done incrementally, starting with the automated CI/CD pipeline and progressing to full production automation.

Regular review and refinement of this plan will ensure it continues to meet the evolving needs of the Tiris Backend production environment.