# Tiris Backend Monitoring System

This document describes the comprehensive monitoring, logging, and alerting system implemented for the Tiris Backend application.

## Overview

The monitoring system provides:
- **Metrics Collection**: Prometheus metrics for application, database, Redis, and system monitoring
- **Structured Logging**: JSON and text format logging with contextual information
- **Alerting**: Multi-channel alerting (Slack, Webhook, Email) with configurable rules
- **Health Checking**: Automated health checks for all system components
- **Observability Dashboard**: Grafana dashboards for visualization

## Components

### 1. Metrics Collection (`pkg/monitoring/metrics.go`)

Collects comprehensive metrics including:

#### HTTP Metrics
- Request count by method, path, status code
- Request duration histograms
- Response size histograms

#### Database Metrics
- Query duration by operation type
- Connection pool metrics
- Query success/failure rates

#### Redis Metrics
- Command duration by command type
- Connection metrics
- Command success/failure rates

#### Security Metrics
- Security events by type and severity
- Authentication/authorization failures
- Rate limiting violations

#### Business Metrics
- User registrations, logins
- Exchange operations
- Transaction volumes
- Trading activities

### 2. Structured Logging (`pkg/monitoring/logger.go`)

Provides contextual logging with:

#### Log Levels
- `debug`: Detailed debugging information
- `info`: General information
- `warn`: Warning conditions
- `error`: Error conditions

#### Log Formats
- **JSON**: Machine-readable structured logs
- **Text**: Human-readable formatted logs

#### Contextual Fields
- Request ID, User ID, IP Address
- HTTP method, path, status code
- Duration, error details
- Custom business context

### 3. Alert Management (`pkg/monitoring/alerting.go`)

Multi-channel alerting system:

#### Alert Types
- **Performance**: High latency, error rates
- **Security**: Authentication failures, suspicious activity
- **Business**: Registration rates, transaction volumes
- **Health**: Service availability, resource usage

#### Alert Receivers
- **Slack**: Channel-based notifications
- **Webhook**: Custom HTTP endpoints
- **Email**: SMTP-based notifications (future)

#### Alert Cooldown
- Prevents alert spam with configurable cooldown periods
- Rule-based alert suppression

### 4. Health Checking (`pkg/monitoring/health.go`)

Automated health checks for:

#### System Components
- Database connectivity and query performance
- Redis connectivity and response time
- HTTP endpoint availability
- Memory usage and limits

#### Kubernetes Integration
- Liveness probes: `/health/live`
- Readiness probes: `/health/ready`
- Health status aggregation

## Configuration

### Environment Variables

```bash
# Metrics Configuration
MONITORING_METRICS_ENABLED=true
MONITORING_METRICS_PORT=9090
MONITORING_METRICS_PATH=/metrics

# Logging Configuration
MONITORING_LOGGING_ENABLED=true
MONITORING_LOG_LEVEL=info
MONITORING_LOG_FORMAT=json
MONITORING_LOG_OUTPUT=stdout

# Alert Configuration
MONITORING_ALERTS_ENABLED=true
MONITORING_SLACK_WEBHOOK_URL=https://hooks.slack.com/...
MONITORING_SLACK_CHANNEL=#alerts

# Health Check Configuration
MONITORING_HEALTH_CHECK_ENABLED=true
MONITORING_HEALTH_CHECK_INTERVAL=30s
MONITORING_HEALTH_CHECK_TIMEOUT=5s
```

### Configuration File

Copy `config/monitoring.env.example` to `config/monitoring.env` and customize for your environment.

## Deployment

### Docker Compose Monitoring Stack

Deploy the complete monitoring stack:

```bash
# Start monitoring services
docker-compose -f docker-compose.monitoring.yml up -d

# Check service status
docker-compose -f docker-compose.monitoring.yml ps
```

This includes:
- **Prometheus** (metrics collection): http://localhost:9090
- **Grafana** (visualization): http://localhost:3000
- **Alertmanager** (alert routing): http://localhost:9093
- **Loki** (log aggregation): http://localhost:3100
- **Node Exporter** (system metrics): http://localhost:9100
- **cAdvisor** (container metrics): http://localhost:8080

### Kubernetes Deployment

Use the Helm chart with monitoring enabled:

```bash
helm install tiris-backend ./helm/tiris-backend \
  --set monitoring.enabled=true \
  --set monitoring.serviceMonitor.enabled=true
```

## Usage

### Application Integration

```go
import (
    "tiris-backend/config"
    "tiris-backend/pkg/monitoring"
)

// Initialize monitoring
monitoringConfig, err := config.LoadMonitoringConfig()
if err != nil {
    log.Fatal(err)
}

monitoringManager, err := monitoring.NewMonitoringManager(monitoringConfig)
if err != nil {
    log.Fatal(err)
}

// Start monitoring
ctx := context.Background()
if err := monitoringManager.Start(ctx); err != nil {
    log.Fatal(err)
}

// Setup database monitoring
if err := monitoringManager.SetupDatabaseMonitoring(db); err != nil {
    log.Fatal(err)
}

// Setup Redis monitoring
if err := monitoringManager.SetupRedisMonitoring(redisClient); err != nil {
    log.Fatal(err)
}
```

### Middleware Integration

```go
import "tiris-backend/internal/middleware"

// Add monitoring middleware to Gin router
router.Use(middleware.MonitoringMiddleware(
    monitoringManager.GetMetricsCollector(),
    monitoringManager.GetLogger(),
))

router.Use(middleware.SecurityMonitoringMiddleware(
    monitoringManager.GetMetricsCollector(),
    monitoringManager.GetLogger(),
))
```

### Manual Logging

```go
// Get logger from monitoring manager
logger := monitoringManager.GetLogger()

// Log with context
logger.LogInfo("User operation completed", map[string]interface{}{
    "user_id": userID,
    "operation": "create_exchange",
    "duration_ms": duration.Milliseconds(),
})

logger.LogError("Database operation failed", map[string]interface{}{
    "error": err.Error(),
    "query": sqlQuery,
    "user_id": userID,
})
```

### Custom Metrics

```go
// Get metrics collector
metrics := monitoringManager.GetMetricsCollector()

// Record business events
metrics.RecordBusinessEvent("payment_processed")

// Record custom metrics
metrics.RecordSecurityEvent("suspicious_login", "medium")
```

## Dashboards

### Grafana Dashboards

Access Grafana at http://localhost:3000 (admin/admin):

1. **Application Overview**
   - HTTP request rates and latencies
   - Error rates and status codes
   - Business metrics trends

2. **Database Performance**
   - Query performance and slow queries
   - Connection pool utilization
   - Database resource usage

3. **Redis Performance**
   - Command latencies and throughput
   - Memory usage and key statistics
   - Connection metrics

4. **Security Monitoring**
   - Authentication/authorization events
   - Rate limiting violations
   - Suspicious activity patterns

5. **System Resources**
   - CPU, memory, disk usage
   - Container resource utilization
   - Network I/O patterns

### Custom Dashboards

Create custom dashboards using the provided metrics:

- `tiris_http_requests_total`: HTTP request counts
- `tiris_http_request_duration_seconds`: HTTP request latencies  
- `tiris_db_query_duration_seconds`: Database query performance
- `tiris_redis_command_duration_seconds`: Redis command performance
- `tiris_security_events_total`: Security event counts
- `tiris_business_events_total`: Business metric counts

## Alerting Rules

### Prometheus Alert Rules

Located in `config/prometheus/alerts.yml`:

#### Critical Alerts
- Application down (>1 minute)
- Database unavailable
- High error rate (>10% for 2 minutes)
- Critical security events

#### Warning Alerts  
- High latency (>500ms average)
- High resource usage (CPU >80%, Memory >85%)
- Low business activity rates
- Slow database queries

### Alert Configuration

Configure alerting in `config/alertmanager/config.yml`:

```yaml
receivers:
  - name: 'critical-alerts'
    slack_configs:
      - api_url: 'YOUR_SLACK_WEBHOOK_URL'
        channel: '#critical-alerts'
        title: 'Critical Alert: {{ .GroupLabels.alertname }}'
```

## Troubleshooting

### Common Issues

1. **Metrics not appearing in Prometheus**
   - Check application metrics endpoint: http://localhost:8080/metrics
   - Verify Prometheus scrape configuration
   - Check network connectivity between services

2. **Alerts not firing**
   - Verify alert rules syntax in Prometheus
   - Check Alertmanager configuration
   - Test webhook endpoints manually

3. **Logs not appearing in Grafana**
   - Check Loki data source configuration
   - Verify Promtail is collecting logs
   - Check log file paths and permissions

4. **High resource usage**
   - Adjust metric retention periods
   - Optimize log collection rules
   - Scale monitoring services if needed

### Debug Commands

```bash
# Check Prometheus targets
curl http://localhost:9090/api/v1/targets

# Test Alertmanager webhook
curl -X POST http://localhost:9093/api/v1/alerts

# Check Loki logs
curl http://localhost:3100/ready

# View application metrics
curl http://localhost:8080/metrics | grep tiris_

# Check monitoring manager health
curl http://localhost:8080/health
```

## Security Considerations

1. **Metric Exposure**
   - Metrics endpoint should be internal-only in production
   - Use authentication/authorization for Grafana access
   - Sanitize sensitive data from logs and metrics

2. **Log Security**
   - Never log passwords, tokens, or secrets
   - Use data masking for sensitive information
   - Implement log rotation and retention policies

3. **Alert Security**
   - Secure webhook endpoints with authentication
   - Use encrypted channels for alert delivery
   - Implement rate limiting on alert endpoints

## Performance Impact

The monitoring system is designed for minimal performance impact:

- **Metrics**: ~1-2ms overhead per HTTP request
- **Logging**: Asynchronous with configurable buffering  
- **Health Checks**: Background goroutines with configurable intervals
- **Memory Usage**: ~10-20MB additional memory footprint

Monitor the monitoring system itself using the provided metrics and adjust configuration as needed for your performance requirements.