# Tiris Backend Helm Chart

This Helm chart deploys the Tiris Backend application - a Trading Infrastructure and Risk Management System.

## Prerequisites

- Kubernetes 1.20+
- Helm 3.2+
- PV provisioner support in the underlying infrastructure
- PostgreSQL client (for migrations)

## Installing the Chart

To install the chart with the release name `tiris-backend`:

```bash
# Add the Bitnami repository (for PostgreSQL and Redis dependencies)
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update

# Install the chart
helm install tiris-backend ./helm/tiris-backend
```

The command deploys Tiris Backend on the Kubernetes cluster in the default namespace with the default configuration.

## Uninstalling the Chart

To uninstall/delete the `tiris-backend` deployment:

```bash
helm delete tiris-backend
```

## Configuration

The following table lists the configurable parameters of the Tiris Backend chart and their default values.

### Application Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `app.name` | Application name | `tiris-backend` |
| `app.version` | Application version | `1.0.0` |
| `image.registry` | Image registry | `docker.io` |
| `image.repository` | Image repository | `tiris/backend` |
| `image.tag` | Image tag | `latest` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |

### Deployment Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `deployment.replicaCount` | Number of replicas | `2` |
| `deployment.strategy.type` | Deployment strategy | `RollingUpdate` |
| `deployment.resources.requests.memory` | Memory request | `512Mi` |
| `deployment.resources.requests.cpu` | CPU request | `500m` |
| `deployment.resources.limits.memory` | Memory limit | `1Gi` |
| `deployment.resources.limits.cpu` | CPU limit | `1000m` |

### Service Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `service.type` | Kubernetes service type | `ClusterIP` |
| `service.port` | Service port | `8080` |
| `service.targetPort` | Container target port | `8080` |

### Ingress Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `ingress.enabled` | Enable ingress | `true` |
| `ingress.className` | Ingress class name | `nginx` |
| `ingress.hosts[0].host` | Hostname | `tiris.ai` |
| `ingress.hosts[1].host` | API hostname | `api.tiris.ai` |
| `ingress.tls[0].secretName` | TLS secret name | `tiris-tls-secret` |

### Auto-scaling Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `autoscaling.enabled` | Enable HPA | `true` |
| `autoscaling.minReplicas` | Minimum replicas | `2` |
| `autoscaling.maxReplicas` | Maximum replicas | `10` |
| `autoscaling.targetCPUUtilizationPercentage` | Target CPU percentage | `70` |
| `autoscaling.targetMemoryUtilizationPercentage` | Target memory percentage | `80` |

### Database Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `config.database.host` | Database host | `tiris-postgresql` |
| `config.database.port` | Database port | `5432` |
| `config.database.name` | Database name | `tiris_prod` |
| `config.database.user` | Database user | `tiris_user` |
| `config.database.sslMode` | SSL mode | `require` |

### PostgreSQL Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `postgresql.enabled` | Deploy PostgreSQL | `true` |
| `postgresql.image.repository` | PostgreSQL image | `timescale/timescaledb` |
| `postgresql.image.tag` | PostgreSQL tag | `latest-pg15` |
| `postgresql.auth.database` | Database name | `tiris_prod` |
| `postgresql.primary.persistence.size` | Storage size | `20Gi` |

### NATS Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `nats.enabled` | Deploy NATS | `true` |
| `nats.jetstream.enabled` | Enable JetStream | `true` |
| `nats.jetstream.maxStorage` | Max storage | `10Gi` |
| `nats.persistence.size` | Storage size | `5Gi` |

### Redis Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `redis.enabled` | Deploy Redis | `true` |
| `redis.architecture` | Redis architecture | `standalone` |
| `redis.master.persistence.size` | Storage size | `2Gi` |

### Security Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `networkPolicy.enabled` | Enable network policies | `true` |
| `deployment.securityContext.runAsNonRoot` | Run as non-root | `true` |
| `deployment.securityContext.runAsUser` | User ID | `1001` |
| `deployment.securityContext.readOnlyRootFilesystem` | Read-only filesystem | `true` |

### Monitoring Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `monitoring.enabled` | Enable monitoring | `true` |
| `monitoring.serviceMonitor.enabled` | Enable ServiceMonitor | `true` |
| `monitoring.prometheusRule.enabled` | Enable PrometheusRule | `true` |

## Production Deployment

For production deployments, create a custom values file:

```yaml
# values-production.yaml
config:
  environment: production
  logLevel: warn

secrets:
  databasePassword: "your-secure-database-password"
  jwtSecret: "your-32-char-jwt-secret-key-here"
  refreshSecret: "your-32-char-refresh-secret-here"
  googleClientSecret: "your-google-oauth-secret"
  wechatAppSecret: "your-wechat-app-secret"
  natsPassword: "your-nats-password"
  redisPassword: "your-redis-password"

ingress:
  hosts:
    - host: api.yourcompany.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: api-tls-secret
      hosts:
        - api.yourcompany.com

deployment:
  replicaCount: 3
  resources:
    requests:
      memory: 1Gi
      cpu: 1000m
    limits:
      memory: 2Gi
      cpu: 2000m

postgresql:
  primary:
    persistence:
      size: 100Gi
    resources:
      requests:
        memory: 2Gi
        cpu: 1000m
      limits:
        memory: 4Gi
        cpu: 2000m
```

Then deploy with:

```bash
helm install tiris-backend ./helm/tiris-backend -f values-production.yaml
```

## Upgrading

To upgrade the deployment:

```bash
helm upgrade tiris-backend ./helm/tiris-backend
```

## Backup and Restore

### Database Backup

The chart includes backup scripts in the main application. To perform manual backups:

```bash
kubectl exec -it deployment/tiris-backend -- /scripts/backup-db.sh
```

### Restore from Backup

```bash
kubectl exec -it deployment/tiris-backend -- /scripts/restore-db.sh /backups/backup-file.sql.gz
```

## Troubleshooting

### Check Pod Status

```bash
kubectl get pods -l app.kubernetes.io/name=tiris-backend
```

### View Logs

```bash
kubectl logs -l app.kubernetes.io/name=tiris-backend -f
```

### Check Migration Status

```bash
kubectl get jobs -l app.kubernetes.io/component=migration
kubectl logs job/tiris-backend-migration
```

### Database Connection Issues

```bash
kubectl exec -it deployment/tiris-backend -- pg_isready -h tiris-postgresql -p 5432 -U tiris_user
```

### Check Health Endpoints

```bash
kubectl port-forward svc/tiris-backend 8080:8080
curl http://localhost:8080/health/live
curl http://localhost:8080/health/ready
```

## Security Considerations

1. **Update Default Secrets**: Never use default values for secrets in production
2. **TLS Certificates**: Configure proper SSL/TLS certificates for ingress
3. **RBAC**: Set up appropriate Role-Based Access Control
4. **Network Policies**: Enable network policies to restrict traffic
5. **Pod Security**: Use security contexts and read-only filesystems
6. **Secret Management**: Consider using external secret management solutions

## Monitoring

The chart includes built-in monitoring support:

- **Metrics**: Prometheus metrics exposed on `/metrics`
- **ServiceMonitor**: Automatic discovery by Prometheus Operator
- **Alerting Rules**: Pre-configured alerts for common issues
- **Health Checks**: Kubernetes liveness and readiness probes

## Support

For issues and support:

- Check the application logs
- Review Kubernetes events
- Verify configuration values
- Ensure all dependencies are running

## License

Proprietary - Tiris Development Team