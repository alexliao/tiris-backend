#!/bin/bash
set -e

# Monitoring Setup Script for Tiris Backend
# Sets up Prometheus, Grafana, and Alertmanager with proper dashboards and alerts

echo "📊 Setting up comprehensive monitoring for Tiris Backend..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
MONITORING_DIR="/opt/tiris/monitoring"
CONFIG_DIR="/opt/tiris/tiris-backend/configs/monitoring"
GRAFANA_VERSION=${GRAFANA_VERSION:-"10.2.0"}
PROMETHEUS_VERSION=${PROMETHEUS_VERSION:-"2.45.0"}

# Logging functions
log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

warn() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}"
    exit 1
}

# Check prerequisites
check_prerequisites() {
    log "Checking prerequisites..."
    
    if ! command -v docker &> /dev/null; then
        error "Docker is not installed"
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        error "Docker Compose is not installed"
    fi
    
    if [[ ! -f "/opt/tiris/tiris-backend/.env.prod" ]]; then
        error "Production environment file not found. Run deployment first."
    fi
    
    log "Prerequisites check passed"
}

# Create monitoring directories
setup_monitoring_dirs() {
    log "Setting up monitoring directories..."
    
    mkdir -p "$MONITORING_DIR"/{prometheus,grafana,alertmanager,loki,promtail}
    mkdir -p "$MONITORING_DIR/grafana"/{dashboards,provisioning/{dashboards,datasources}}
    mkdir -p "$MONITORING_DIR/prometheus"/{data,rules}
    mkdir -p "$MONITORING_DIR/alertmanager"/{data,templates}
    
    # Set proper permissions
    sudo chown -R 472:472 "$MONITORING_DIR/grafana" # Grafana user
    sudo chown -R 65534:65534 "$MONITORING_DIR/prometheus" # Nobody user for Prometheus
    sudo chown -R 65534:65534 "$MONITORING_DIR/alertmanager"
    
    log "Monitoring directories created"
}

# Configure Prometheus
setup_prometheus() {
    log "Configuring Prometheus..."
    
    cat > "$MONITORING_DIR/prometheus/prometheus.yml" << 'EOF'
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - "/etc/prometheus/rules/*.yml"

alerting:
  alertmanagers:
    - static_configs:
        - targets:
          - alertmanager:9093

scrape_configs:
  # Prometheus itself
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']

  # Tiris Backend Application
  - job_name: 'tiris-backend'
    static_configs:
      - targets: ['app:8080']
    metrics_path: '/metrics'
    scrape_interval: 10s

  # PostgreSQL Exporter
  - job_name: 'postgres'
    static_configs:
      - targets: ['postgres-exporter:9187']

  # Redis Exporter
  - job_name: 'redis'
    static_configs:
      - targets: ['redis-exporter:9121']

  # Node Exporter (System Metrics)
  - job_name: 'node'
    static_configs:
      - targets: ['node-exporter:9100']

  # NATS Exporter
  - job_name: 'nats'
    static_configs:
      - targets: ['nats:8222']
    metrics_path: '/metrics'

  # Nginx Exporter
  - job_name: 'nginx'
    static_configs:
      - targets: ['nginx-exporter:9113']

  # Docker containers
  - job_name: 'docker'
    static_configs:
      - targets: ['cadvisor:8080']
EOF

    log "Prometheus configuration created"
}

# Configure alert rules
setup_alert_rules() {
    log "Setting up alert rules..."
    
    cat > "$MONITORING_DIR/prometheus/rules/tiris-backend.yml" << 'EOF'
groups:
- name: tiris-backend
  rules:
  # Application Health
  - alert: TirisBackendDown
    expr: up{job="tiris-backend"} == 0
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "Tiris Backend is down"
      description: "Tiris Backend has been down for more than 1 minute"

  # High Error Rate
  - alert: HighErrorRate
    expr: sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m])) > 0.05
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High error rate detected"
      description: "Error rate is {{ $value | humanizePercentage }} over the last 5 minutes"

  # High Response Time
  - alert: HighResponseTime
    expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 2
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High response time"
      description: "95th percentile response time is {{ $value }}s"

  # Database Health
  - alert: DatabaseDown
    expr: up{job="postgres"} == 0
    for: 2m
    labels:
      severity: critical
    annotations:
      summary: "PostgreSQL database is down"
      description: "PostgreSQL database has been down for more than 2 minutes"

  # High Database Connections
  - alert: HighDatabaseConnections
    expr: pg_stat_activity_count > 80
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High database connection count"
      description: "Database has {{ $value }} active connections"

  # Redis Health
  - alert: RedisDown
    expr: up{job="redis"} == 0
    for: 2m
    labels:
      severity: critical
    annotations:
      summary: "Redis is down"
      description: "Redis has been down for more than 2 minutes"

  # System Resources
  - alert: HighCPUUsage
    expr: 100 - (avg by (instance) (rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100) > 80
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High CPU usage"
      description: "CPU usage is {{ $value }}% on {{ $labels.instance }}"

  - alert: HighMemoryUsage
    expr: (1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100 > 85
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High memory usage"
      description: "Memory usage is {{ $value }}% on {{ $labels.instance }}"

  - alert: LowDiskSpace
    expr: (1 - (node_filesystem_avail_bytes{fstype!="tmpfs"} / node_filesystem_size_bytes)) * 100 > 85
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Low disk space"
      description: "Disk usage is {{ $value }}% on {{ $labels.instance }}"

  # NATS Health
  - alert: NATSDown
    expr: up{job="nats"} == 0
    for: 2m
    labels:
      severity: critical
    annotations:
      summary: "NATS server is down"
      description: "NATS server has been down for more than 2 minutes"
EOF

    log "Alert rules configured"
}

# Configure Alertmanager
setup_alertmanager() {
    log "Configuring Alertmanager..."
    
    cat > "$MONITORING_DIR/alertmanager/alertmanager.yml" << 'EOF'
global:
  smtp_smarthost: 'localhost:587'
  smtp_from: 'alerts@tiris.ai'

route:
  group_by: ['alertname']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 1h
  receiver: 'web.hook'

receivers:
- name: 'web.hook'
  webhook_configs:
  - url: 'http://localhost:5001/'
    send_resolved: true

# Add your notification channels here
# - name: 'slack'
#   slack_configs:
#   - api_url: 'YOUR_SLACK_WEBHOOK_URL'
#     channel: '#alerts'
#     title: 'Tiris Backend Alert'
#     text: '{{ range .Alerts }}{{ .Annotations.summary }}{{ end }}'

# - name: 'email'
#   email_configs:
#   - to: 'admin@tiris.ai'
#     subject: 'Tiris Backend Alert'
#     body: |
#       {{ range .Alerts }}
#       Alert: {{ .Annotations.summary }}
#       Description: {{ .Annotations.description }}
#       {{ end }}
EOF

    log "Alertmanager configuration created"
}

# Configure Grafana
setup_grafana() {
    log "Configuring Grafana..."
    
    # Grafana configuration
    cat > "$MONITORING_DIR/grafana/grafana.ini" << 'EOF'
[server]
http_port = 3000
domain = localhost

[security]
admin_user = admin
admin_password = admin

[users]
allow_sign_up = false

[auth.anonymous]
enabled = false

[dashboards]
default_home_dashboard_path = /var/lib/grafana/dashboards/tiris-overview.json

[log]
mode = console
level = info
EOF

    # Datasource configuration
    cat > "$MONITORING_DIR/grafana/provisioning/datasources/prometheus.yml" << 'EOF'
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
    editable: true

  - name: Loki
    type: loki
    access: proxy
    url: http://loki:3100
    editable: true
EOF

    # Dashboard provisioning
    cat > "$MONITORING_DIR/grafana/provisioning/dashboards/dashboards.yml" << 'EOF'
apiVersion: 1

providers:
  - name: 'tiris-dashboards'
    type: file
    folder: 'Tiris Backend'
    options:
      path: /var/lib/grafana/dashboards
EOF

    log "Grafana configuration created"
}

# Create Grafana dashboards
create_dashboards() {
    log "Creating Grafana dashboards..."
    
    # Create a comprehensive dashboard for Tiris Backend
    cat > "$MONITORING_DIR/grafana/dashboards/tiris-overview.json" << 'EOF'
{
  "dashboard": {
    "id": null,
    "title": "Tiris Backend Overview",
    "tags": ["tiris", "backend"],
    "timezone": "browser",
    "panels": [
      {
        "id": 1,
        "title": "Application Status",
        "type": "stat",
        "targets": [
          {
            "expr": "up{job=\"tiris-backend\"}",
            "legendFormat": "Backend Status"
          }
        ],
        "fieldConfig": {
          "defaults": {
            "color": {
              "mode": "thresholds"
            },
            "thresholds": {
              "steps": [
                {"color": "red", "value": 0},
                {"color": "green", "value": 1}
              ]
            }
          }
        },
        "gridPos": {"h": 8, "w": 6, "x": 0, "y": 0}
      },
      {
        "id": 2,
        "title": "Request Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(http_requests_total[5m])",
            "legendFormat": "{{method}} {{endpoint}}"
          }
        ],
        "gridPos": {"h": 8, "w": 18, "x": 6, "y": 0}
      },
      {
        "id": 3,
        "title": "Response Time",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))",
            "legendFormat": "95th Percentile"
          },
          {
            "expr": "histogram_quantile(0.50, rate(http_request_duration_seconds_bucket[5m]))",
            "legendFormat": "50th Percentile"
          }
        ],
        "gridPos": {"h": 8, "w": 12, "x": 0, "y": 8}
      },
      {
        "id": 4,
        "title": "Error Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(http_requests_total{status=~\"5..\"}[5m])",
            "legendFormat": "5xx Errors"
          },
          {
            "expr": "rate(http_requests_total{status=~\"4..\"}[5m])",
            "legendFormat": "4xx Errors"
          }
        ],
        "gridPos": {"h": 8, "w": 12, "x": 12, "y": 8}
      }
    ],
    "time": {
      "from": "now-1h",
      "to": "now"
    },
    "refresh": "30s"
  }
}
EOF

    log "Grafana dashboards created"
}

# Create monitoring Docker Compose file
create_monitoring_compose() {
    log "Creating monitoring Docker Compose configuration..."
    
    cat > "$MONITORING_DIR/docker-compose.monitoring.yml" << 'EOF'
version: '3.8'

services:
  prometheus:
    image: prom/prometheus:latest
    container_name: tiris-prometheus
    restart: unless-stopped
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus/prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - ./prometheus/rules:/etc/prometheus/rules:ro
      - ./prometheus/data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'
      - '--storage.tsdb.retention.time=30d'
      - '--web.enable-lifecycle'
      - '--web.enable-admin-api'
    networks:
      - tiris-monitoring

  grafana:
    image: grafana/grafana:latest
    container_name: tiris-grafana
    restart: unless-stopped
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    volumes:
      - ./grafana/grafana.ini:/etc/grafana/grafana.ini:ro
      - ./grafana/provisioning:/etc/grafana/provisioning:ro
      - ./grafana/dashboards:/var/lib/grafana/dashboards:ro
      - grafana-storage:/var/lib/grafana
    networks:
      - tiris-monitoring

  alertmanager:
    image: prom/alertmanager:latest
    container_name: tiris-alertmanager
    restart: unless-stopped
    ports:
      - "9093:9093"
    volumes:
      - ./alertmanager/alertmanager.yml:/etc/alertmanager/alertmanager.yml:ro
      - ./alertmanager/data:/alertmanager
    command:
      - '--config.file=/etc/alertmanager/alertmanager.yml'
      - '--storage.path=/alertmanager'
      - '--web.external-url=http://localhost:9093'
    networks:
      - tiris-monitoring

  node-exporter:
    image: prom/node-exporter:latest
    container_name: tiris-node-exporter
    restart: unless-stopped
    ports:
      - "9100:9100"
    volumes:
      - /proc:/host/proc:ro
      - /sys:/host/sys:ro
      - /:/rootfs:ro
    command:
      - '--path.procfs=/host/proc'
      - '--path.rootfs=/rootfs'
      - '--path.sysfs=/host/sys'
      - '--collector.filesystem.mount-points-exclude=^/(sys|proc|dev|host|etc)($$|/)'
    networks:
      - tiris-monitoring

  postgres-exporter:
    image: prometheuscommunity/postgres-exporter:latest
    container_name: tiris-postgres-exporter
    restart: unless-stopped
    ports:
      - "9187:9187"
    environment:
      - DATA_SOURCE_NAME=postgresql://tiris_user:${DB_PASSWORD}@postgres:5432/tiris_prod?sslmode=require
    networks:
      - tiris-monitoring

  redis-exporter:
    image: oliver006/redis_exporter:latest
    container_name: tiris-redis-exporter
    restart: unless-stopped
    ports:
      - "9121:9121"
    environment:
      - REDIS_ADDR=redis://redis:6379
      - REDIS_PASSWORD=${REDIS_PASSWORD}
    networks:
      - tiris-monitoring

  cadvisor:
    image: gcr.io/cadvisor/cadvisor:latest
    container_name: tiris-cadvisor
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - /:/rootfs:ro
      - /var/run:/var/run:ro
      - /sys:/sys:ro
      - /var/lib/docker/:/var/lib/docker:ro
      - /dev/disk/:/dev/disk:ro
    privileged: true
    devices:
      - /dev/kmsg
    networks:
      - tiris-monitoring

volumes:
  grafana-storage:

networks:
  tiris-monitoring:
    driver: bridge
    external: true
EOF

    log "Monitoring Docker Compose configuration created"
}

# Start monitoring services
start_monitoring() {
    log "Starting monitoring services..."
    
    cd "$MONITORING_DIR"
    
    # Create external network if it doesn't exist
    docker network create tiris-monitoring 2>/dev/null || true
    
    # Start monitoring stack
    docker-compose -f docker-compose.monitoring.yml --env-file /opt/tiris/tiris-backend/.env.prod up -d
    
    # Wait for services to start
    sleep 30
    
    # Check service health
    local services=("prometheus" "grafana" "alertmanager" "node-exporter")
    for service in "${services[@]}"; do
        if docker ps | grep -q "tiris-$service"; then
            log "✅ $service is running"
        else
            warn "❌ $service failed to start"
        fi
    done
    
    log "Monitoring services started"
}

# Main execution
main() {
    log "🚀 Starting monitoring setup for Tiris Backend..."
    
    check_prerequisites
    setup_monitoring_dirs
    setup_prometheus
    setup_alert_rules
    setup_alertmanager
    setup_grafana
    create_dashboards
    create_monitoring_compose
    start_monitoring
    
    log "✅ Monitoring setup completed successfully!"
    
    echo ""
    echo "=== Monitoring Services ==="
    echo "📊 Grafana: http://localhost:3000 (admin/admin)"
    echo "🔍 Prometheus: http://localhost:9090"
    echo "🚨 Alertmanager: http://localhost:9093"
    echo "💻 Node Exporter: http://localhost:9100"
    echo ""
    echo "=== Quick Commands ==="
    echo "📈 View containers: docker ps | grep tiris-"
    echo "📋 View logs: docker-compose -f $MONITORING_DIR/docker-compose.monitoring.yml logs -f [service]"
    echo "🔄 Restart monitoring: docker-compose -f $MONITORING_DIR/docker-compose.monitoring.yml restart"
    echo "🛑 Stop monitoring: docker-compose -f $MONITORING_DIR/docker-compose.monitoring.yml down"
    echo ""
    warn "Remember to:"
    warn "1. Change the default Grafana password"
    warn "2. Configure alerting channels (Slack, email, etc.)"
    warn "3. Set up reverse proxy for external access"
    warn "4. Configure firewall rules for monitoring ports"
}

# Error handling
trap 'error "Monitoring setup interrupted"' INT TERM

# Execute main function
main "$@"
EOF