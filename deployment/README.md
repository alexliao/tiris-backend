# Tiris Backend Deployment Options

Choose the deployment approach that best fits your needs:

## 🚀 Quick Deploy (Recommended for Development)

**Perfect for:** Development, Testing, Demos, Learning  
**Time:** ~5 minutes  
**Complexity:** Minimal  

Get your application online with minimal effort:

```bash
./scripts/quick-deploy.sh
```

[📖 Quick Deploy Guide](../QUICK_DEPLOY.md)

**What you get:**
- ✅ PostgreSQL + Application + Nginx (optional)
- ✅ Basic security configuration
- ✅ Essential environment setup
- ✅ Health checks and basic monitoring

**What you don't get:**
- ❌ Advanced monitoring (Prometheus/Grafana)
- ❌ Automated backups
- ❌ Redis/NATS services
- ❌ SSL/HTTPS setup
- ❌ Production-grade security hardening

## 🏭 Production Deploy (Recommended for Production)

**Perfect for:** Production, High-traffic applications, Mission-critical systems  
**Time:** ~30 minutes  
**Complexity:** Comprehensive  

Full production deployment with all operational features:

```bash
./deployment/scripts/deploy-production.sh
```

[📖 Production Deploy Guide](./docs/PRODUCTION_DEPLOYMENT.md)

**What you get:**
- ✅ All services (PostgreSQL, Redis, NATS, Application, Nginx)
- ✅ Comprehensive monitoring (Prometheus, Grafana, Alertmanager)
- ✅ Automated backups and retention policies
- ✅ SSL/HTTPS with Let's Encrypt
- ✅ Security hardening and rate limiting
- ✅ Log aggregation and rotation
- ✅ Performance optimization
- ✅ Health checks and validation
- ✅ Operational runbooks and procedures

## 📊 Comparison

| Feature | Quick Deploy | Production Deploy |
|---------|--------------|-------------------|
| **Setup Time** | 5 minutes | 30 minutes |
| **Services** | App + DB + Nginx | All services + monitoring |
| **SSL/HTTPS** | Manual | Automated |
| **Monitoring** | Basic | Comprehensive |
| **Backups** | Manual | Automated |
| **Security** | Basic | Production-grade |
| **Scaling** | Limited | Full support |
| **Maintenance** | Manual | Automated |

## 🎯 Decision Guide

### Choose Quick Deploy if:
- You're developing or testing
- You need something running quickly
- You're learning the system
- You want minimal complexity
- You don't need advanced features

### Choose Production Deploy if:
- You're deploying to production
- You need monitoring and alerts
- You need automated backups
- You need SSL/HTTPS
- You need high availability
- You need operational procedures

## 🔄 Migration Path

Start with Quick Deploy and upgrade later:

1. **Start Quick**: Use Quick Deploy for development
2. **Test Features**: Validate your application works
3. **Upgrade**: Switch to Production Deploy when ready
4. **Migrate Data**: Use backup/restore procedures

## 📁 Directory Structure

```
deployment/
├── README.md                    # This file
├── scripts/                     # Production deployment scripts
│   ├── deploy-production.sh     # Main production deployment
│   ├── vps-setup.sh            # VPS preparation
│   ├── generate-production-env.sh # Environment configuration
│   ├── backup-production.sh     # Backup and maintenance
│   ├── setup-monitoring.sh     # Monitoring stack
│   └── validate-deployment.sh  # Deployment validation
├── docs/                       # Documentation
│   ├── PRODUCTION_DEPLOYMENT.md # Complete production guide
│   └── OPERATIONS_RUNBOOK.md   # Operational procedures
└── configs/                    # Configuration templates
    └── .env.prod.template      # Production environment template
```

## 🆘 Support

- **Quick Deploy Issues**: Check [QUICK_DEPLOY.md](../QUICK_DEPLOY.md)
- **Production Issues**: Check [OPERATIONS_RUNBOOK.md](./docs/OPERATIONS_RUNBOOK.md)
- **General Questions**: Review the documentation in `docs/`

---

**Tip:** Start with Quick Deploy to get familiar with the system, then upgrade to Production Deploy when you're ready for production use!