# Tiris Backend Deployment and DevOps Guide

## 1. Deployment Overview

### 1.1 Deployment Strategy
- **Current Phase**: Docker-based deployment on single host
- **Future Phase**: Kubernetes orchestration for scalability
- **Environment Progression**: Development → Staging → Production
- **Deployment Method**: Blue-Green deployment with rollback capability
- **Infrastructure**: Cloud-native with container orchestration

### 1.2 Environment Architecture

```
┌─────────────────────────────────────────────────────┐
│                Production Environment               │
├─────────────────────────────────────────────────────┤
│  Load Balancer (nginx/cloud LB)                    │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐   │
│  │tiris-backend│ │tiris-backend│ │tiris-backend│   │
│  │  (active)   │ │  (standby)  │ │  (standby)  │   │
│  └─────────────┘ └─────────────┘ └─────────────┘   │
│                         │                          │
│  ┌─────────────────────────────────────────────┐   │
│  │     PostgreSQL + TimescaleDB Cluster       │   │
│  │     (Primary + Replicas)                   │   │
│  └─────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────┘
```

## 2. Docker Configuration

### 2.1 Application Dockerfile

```dockerfile
# Multi-stage build for optimized production image
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o tiris-backend ./cmd/server

# Production image
FROM alpine:3.18

# Install ca-certificates for HTTPS calls
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy binary from builder stage
COPY --from=builder /app/tiris-backend .

# Copy configuration files
COPY --from=builder /app/configs ./configs

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health/live || exit 1

# Run the application
CMD ["./tiris-backend"]
```

### 2.2 Docker Compose for Local Development

```yaml
# docker-compose.yml
version: '3.8'

services:
  tiris-backend:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DB_HOST=postgres
      - DB_PORT=5432
      - DB_NAME=tiris_dev
      - DB_USER=tiris_user
      - DB_PASSWORD=dev_password
      - REDIS_HOST=redis
      - REDIS_PORT=6379
      - NATS_URL=nats://nats:4222
      - JWT_SECRET=dev_jwt_secret
      - OAUTH_GOOGLE_CLIENT_ID=${OAUTH_GOOGLE_CLIENT_ID}
      - OAUTH_GOOGLE_CLIENT_SECRET=${OAUTH_GOOGLE_CLIENT_SECRET}
    depends_on:
      - postgres
      - redis
      - nats
    volumes:
      - ./logs:/app/logs
    restart: unless-stopped

  postgres:
    image: timescale/timescaledb:latest-pg15
    environment:
      POSTGRES_DB: tiris_dev
      POSTGRES_USER: tiris_user
      POSTGRES_PASSWORD: dev_password
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    restart: unless-stopped

  nats:
    image: nats:2.10-alpine
    command: ["--jetstream", "--store_dir", "/data", "--max_memory_store", "1GB", "--max_file_store", "10GB"]
    ports:
      - "4222:4222"
      - "8222:8222"
    volumes:
      - nats_data:/data
    restart: unless-stopped

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
      - ./ssl:/etc/ssl/certs
    depends_on:
      - tiris-backend
    restart: unless-stopped

volumes:
  postgres_data:
  redis_data:
  nats_data:
```

### 2.3 Production Docker Compose

```yaml
# docker-compose.prod.yml
version: '3.8'

services:
  tiris-backend:
    image: tiris/backend:${VERSION}
    deploy:
      replicas: 3
      restart_policy:
        condition: on-failure
        delay: 5s
        max_attempts: 3
      resources:
        limits:
          cpus: '0.5'
          memory: 512M
        reservations:
          cpus: '0.25'
          memory: 256M
    environment:
      - DB_HOST=${DB_HOST}
      - DB_PORT=${DB_PORT}
      - DB_NAME=${DB_NAME}
      - DB_USER=${DB_USER}
      - DB_PASSWORD=${DB_PASSWORD}
      - REDIS_HOST=${REDIS_HOST}
      - NATS_URL=${NATS_URL}
      - JWT_SECRET=${JWT_SECRET}
      - OAUTH_GOOGLE_CLIENT_ID=${OAUTH_GOOGLE_CLIENT_ID}
      - OAUTH_GOOGLE_CLIENT_SECRET=${OAUTH_GOOGLE_CLIENT_SECRET}
      - LOG_LEVEL=info
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8080/health/live"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"

networks:
  tiris_network:
    driver: bridge
```

## 3. CI/CD Pipeline

### 3.1 GitHub Actions Workflow

```yaml
# .github/workflows/ci-cd.yml
name: CI/CD Pipeline

on:
  push:
    branches: [ master, develop ]
  pull_request:
    branches: [ master ]

env:
  REGISTRY: ghcr.io
  IMAGE_NAME: ${{ github.repository }}

jobs:
  test:
    runs-on: ubuntu-latest
    
    services:
      postgres:
        image: timescale/timescaledb:latest-pg15
        env:
          POSTGRES_DB: tiris_test
          POSTGRES_USER: test_user
          POSTGRES_PASSWORD: test_pass
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 5432:5432
      
      redis:
        image: redis:7-alpine
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 6379:6379
      
      nats:
        image: nats:2.10-alpine
        options: >-
          --health-cmd "nats pub test.subject test-message --timeout=2s"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 4222:4222
    
    steps:
    - name: Checkout code
      uses: actions/checkout@v4
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Cache Go modules
      uses: actions/cache@v3
      with:
        path: ~/go/pkg/mod
        key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
        restore-keys: |
          ${{ runner.os }}-go-
    
    - name: Install dependencies
      run: go mod download
    
    - name: Run linter
      uses: golangci/golangci-lint-action@v3
      with:
        version: latest
    
    - name: Run tests
      run: |
        go test -v -race -coverprofile=coverage.out ./...
        go tool cover -html=coverage.out -o coverage.html
      env:
        TEST_DB_HOST: localhost
        TEST_DB_PORT: 5432
        TEST_DB_NAME: tiris_test
        TEST_DB_USER: test_user
        TEST_DB_PASS: test_pass
        TEST_REDIS_HOST: localhost
        TEST_REDIS_PORT: 6379
        TEST_NATS_URL: nats://localhost:4222
    
    - name: Upload coverage reports
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.out
        flags: unittests
        name: codecov-umbrella
    
    - name: Run security scan
      uses: securecodewarrior/github-action-add-sarif@v1
      with:
        sarif-file: 'gosec-report.sarif'

  build:
    needs: test
    runs-on: ubuntu-latest
    if: github.event_name == 'push'
    
    permissions:
      contents: read
      packages: write
    
    steps:
    - name: Checkout code
      uses: actions/checkout@v4
    
    - name: Set up Docker Buildx
      uses: docker/setup-buildx-action@v3
    
    - name: Log in to Container Registry
      uses: docker/login-action@v3
      with:
        registry: ${{ env.REGISTRY }}
        username: ${{ github.actor }}
        password: ${{ secrets.GITHUB_TOKEN }}
    
    - name: Extract metadata
      id: meta
      uses: docker/metadata-action@v5
      with:
        images: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
        tags: |
          type=ref,event=branch
          type=ref,event=pr
          type=sha,prefix={{branch}}-
          type=raw,value=latest,enable={{is_default_branch}}
    
    - name: Build and push Docker image
      uses: docker/build-push-action@v5
      with:
        context: .
        push: true
        tags: ${{ steps.meta.outputs.tags }}
        labels: ${{ steps.meta.outputs.labels }}
        cache-from: type=gha
        cache-to: type=gha,mode=max

  deploy-staging:
    needs: build
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/develop'
    
    environment:
      name: staging
      url: https://staging-api.tiris.com
    
    steps:
    - name: Deploy to staging
      uses: appleboy/ssh-action@v1.0.0
      with:
        host: ${{ secrets.STAGING_HOST }}
        username: ${{ secrets.STAGING_USER }}
        key: ${{ secrets.STAGING_SSH_KEY }}
        script: |
          cd /opt/tiris-backend
          docker-compose -f docker-compose.staging.yml pull
          docker-compose -f docker-compose.staging.yml up -d
          docker system prune -f

  deploy-production:
    needs: build
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/master'
    
    environment:
      name: production
      url: https://api.tiris.com
    
    steps:
    - name: Deploy to production
      uses: appleboy/ssh-action@v1.0.0
      with:
        host: ${{ secrets.PROD_HOST }}
        username: ${{ secrets.PROD_USER }}
        key: ${{ secrets.PROD_SSH_KEY }}
        script: |
          cd /opt/tiris-backend
          ./deploy.sh ${{ github.sha }}
```

### 3.2 Deployment Script

```bash
#!/bin/bash
# deploy.sh - Blue-Green deployment script

set -e

VERSION=${1:-latest}
CURRENT_COLOR=$(docker-compose -f docker-compose.prod.yml ps --services --filter "status=running" | head -1 | grep -o -E "(blue|green)")
NEW_COLOR=$([ "$CURRENT_COLOR" = "blue" ] && echo "green" || echo "blue")

echo "Current deployment: $CURRENT_COLOR"
echo "Deploying to: $NEW_COLOR"
echo "Version: $VERSION"

# Update environment variables
export VERSION=$VERSION
export DEPLOY_COLOR=$NEW_COLOR

# Pull new image
docker pull ghcr.io/tiris/backend:$VERSION

# Start new deployment
docker-compose -f docker-compose.$NEW_COLOR.yml up -d

# Health check
echo "Waiting for $NEW_COLOR deployment to be healthy..."
for i in {1..30}; do
  if curl -f http://localhost:8080/health/ready; then
    echo "✅ $NEW_COLOR deployment is healthy"
    break
  fi
  sleep 10
  if [ $i -eq 30 ]; then
    echo "❌ $NEW_COLOR deployment failed health check"
    docker-compose -f docker-compose.$NEW_COLOR.yml logs
    docker-compose -f docker-compose.$NEW_COLOR.yml down
    exit 1
  fi
done

# Switch traffic to new deployment
echo "Switching traffic to $NEW_COLOR..."
./switch-traffic.sh $NEW_COLOR

# Stop old deployment
echo "Stopping $CURRENT_COLOR deployment..."
docker-compose -f docker-compose.$CURRENT_COLOR.yml down

# Cleanup old images
docker image prune -f

echo "✅ Deployment complete: $VERSION deployed to $NEW_COLOR"
```

## 4. Infrastructure as Code

### 4.1 Terraform Configuration

```hcl
# main.tf
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
  
  backend "s3" {
    bucket = "tiris-terraform-state"
    key    = "backend/terraform.tfstate"
    region = "us-west-2"
  }
}

provider "aws" {
  region = var.aws_region
}

# VPC and Networking
module "vpc" {
  source = "terraform-aws-modules/vpc/aws"
  
  name = "tiris-vpc"
  cidr = "10.0.0.0/16"
  
  azs             = ["${var.aws_region}a", "${var.aws_region}b", "${var.aws_region}c"]
  private_subnets = ["10.0.1.0/24", "10.0.2.0/24", "10.0.3.0/24"]
  public_subnets  = ["10.0.101.0/24", "10.0.102.0/24", "10.0.103.0/24"]
  
  enable_nat_gateway = true
  enable_vpn_gateway = true
  
  tags = {
    Environment = var.environment
    Project     = "tiris"
  }
}

# ECS Cluster
resource "aws_ecs_cluster" "tiris_cluster" {
  name = "tiris-backend-${var.environment}"
  
  configuration {
    execute_command_configuration {
      logging = "OVERRIDE"
      
      log_configuration {
        cloud_watch_encryption_enabled = true
        cloud_watch_log_group_name     = aws_cloudwatch_log_group.ecs_logs.name
      }
    }
  }
  
  tags = {
    Environment = var.environment
    Project     = "tiris"
  }
}

# RDS (PostgreSQL with TimescaleDB)
module "db" {
  source = "terraform-aws-modules/rds/aws"
  
  identifier = "tiris-db-${var.environment}"
  
  engine         = "postgres"
  engine_version = "15.3"
  instance_class = var.db_instance_class
  
  allocated_storage     = 100
  max_allocated_storage = 1000
  storage_type          = "gp3"
  storage_encrypted     = true
  
  db_name  = "tiris"
  username = var.db_username
  password = var.db_password
  port     = "5432"
  
  vpc_security_group_ids = [aws_security_group.rds.id]
  subnet_group_name      = module.vpc.database_subnet_group_name
  
  backup_retention_period = 7
  backup_window          = "03:00-04:00"
  maintenance_window     = "sun:04:00-sun:05:00"
  
  deletion_protection = var.environment == "production"
  
  tags = {
    Environment = var.environment
    Project     = "tiris"
  }
}

# ElastiCache (Redis)
resource "aws_elasticache_subnet_group" "redis" {
  name       = "tiris-redis-${var.environment}"
  subnet_ids = module.vpc.private_subnets
}

resource "aws_elasticache_cluster" "redis" {
  cluster_id           = "tiris-redis-${var.environment}"
  engine               = "redis"
  node_type            = "cache.t3.micro"
  num_cache_nodes      = 1
  parameter_group_name = "default.redis7"
  port                 = 6379
  subnet_group_name    = aws_elasticache_subnet_group.redis.name
  security_group_ids   = [aws_security_group.redis.id]
  
  tags = {
    Environment = var.environment
    Project     = "tiris"
  }
}
```

### 4.2 Kubernetes Manifests (Future)

```yaml
# k8s/namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: tiris
  labels:
    name: tiris

---
# k8s/configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: tiris-backend-config
  namespace: tiris
data:
  LOG_LEVEL: "info"
  DB_HOST: "postgres-service"
  DB_PORT: "5432"
  REDIS_HOST: "redis-service"
  REDIS_PORT: "6379"

---
# k8s/secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: tiris-backend-secrets
  namespace: tiris
type: Opaque
data:
  DB_PASSWORD: <base64-encoded-password>
  JWT_SECRET: <base64-encoded-jwt-secret>
  OAUTH_GOOGLE_CLIENT_SECRET: <base64-encoded-oauth-secret>

---
# k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tiris-backend
  namespace: tiris
  labels:
    app: tiris-backend
spec:
  replicas: 3
  selector:
    matchLabels:
      app: tiris-backend
  template:
    metadata:
      labels:
        app: tiris-backend
    spec:
      containers:
      - name: tiris-backend
        image: ghcr.io/tiris/backend:latest
        ports:
        - containerPort: 8080
        env:
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: tiris-backend-secrets
              key: DB_PASSWORD
        envFrom:
        - configMapRef:
            name: tiris-backend-config
        - secretRef:
            name: tiris-backend-secrets
        livenessProbe:
          httpGet:
            path: /health/live
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          limits:
            cpu: 500m
            memory: 512Mi
          requests:
            cpu: 250m
            memory: 256Mi

---
# k8s/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: tiris-backend-service
  namespace: tiris
spec:
  selector:
    app: tiris-backend
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
  type: ClusterIP

---
# k8s/ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: tiris-backend-ingress
  namespace: tiris
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/rate-limit: "100"
spec:
  tls:
  - hosts:
    - api.tiris.com
    secretName: tiris-tls
  rules:
  - host: api.tiris.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: tiris-backend-service
            port:
              number: 80
```

## 5. Environment Management

### 5.1 Environment Configuration

**Development Environment:**
```bash
# .env.development
DB_HOST=localhost
DB_PORT=5432
DB_NAME=tiris_dev
DB_USER=tiris_user
DB_PASSWORD=dev_password
NATS_URL=nats://localhost:4222
REDIS_HOST=localhost
REDIS_PORT=6379
LOG_LEVEL=debug
JWT_SECRET=dev_jwt_secret_change_in_production
OAUTH_GOOGLE_CLIENT_ID=dev_client_id
OAUTH_GOOGLE_CLIENT_SECRET=dev_client_secret
CORS_ORIGINS=http://localhost:3000,https://dev.tiris.ai
API_RATE_LIMIT=1000
```

**Staging Environment:**
```bash
# .env.staging
DB_HOST=staging-db.tiris.com
DB_PORT=5432
DB_NAME=tiris_staging
DB_USER=tiris_staging_user
DB_PASSWORD=${STAGING_DB_PASSWORD}
NATS_URL=${STAGING_NATS_URL}
REDIS_HOST=staging-redis.tiris.com
REDIS_PORT=6379
LOG_LEVEL=info
JWT_SECRET=${STAGING_JWT_SECRET}
OAUTH_GOOGLE_CLIENT_ID=${STAGING_OAUTH_CLIENT_ID}
OAUTH_GOOGLE_CLIENT_SECRET=${STAGING_OAUTH_CLIENT_SECRET}
CORS_ORIGINS=https://dev.tiris.ai
API_RATE_LIMIT=500
```

**Production Environment:**
```bash
# .env.production
DB_HOST=${PROD_DB_HOST}
DB_PORT=5432
DB_NAME=tiris_production
DB_USER=${PROD_DB_USER}
DB_PASSWORD=${PROD_DB_PASSWORD}
NATS_URL=${PROD_NATS_URL}
REDIS_HOST=${PROD_REDIS_HOST}
REDIS_PORT=6379
LOG_LEVEL=warn
JWT_SECRET=${PROD_JWT_SECRET}
OAUTH_GOOGLE_CLIENT_ID=${PROD_OAUTH_CLIENT_ID}
OAUTH_GOOGLE_CLIENT_SECRET=${PROD_OAUTH_CLIENT_SECRET}
CORS_ORIGINS=https://tiris.ai
API_RATE_LIMIT=1000
ENABLE_METRICS=true
METRICS_PORT=9090
```

### 5.2 Configuration Management

```go
// config/config.go
type Config struct {
    // Server Configuration
    Port         int    `env:"PORT" envDefault:"8080"`
    Host         string `env:"HOST" envDefault:"0.0.0.0"`
    LogLevel     string `env:"LOG_LEVEL" envDefault:"info"`
    Environment  string `env:"ENVIRONMENT" envDefault:"development"`
    
    // Database Configuration
    DBHost     string `env:"DB_HOST" envDefault:"localhost"`
    DBPort     int    `env:"DB_PORT" envDefault:"5432"`
    DBName     string `env:"DB_NAME" envDefault:"tiris"`
    DBUser     string `env:"DB_USER" envDefault:"tiris_user"`
    DBPassword string `env:"DB_PASSWORD" envDefault:"password"`
    DBSSLMode  string `env:"DB_SSL_MODE" envDefault:"disable"`
    
    // Redis Configuration
    RedisHost     string `env:"REDIS_HOST" envDefault:"localhost"`
    RedisPort     int    `env:"REDIS_PORT" envDefault:"6379"`
    RedisPassword string `env:"REDIS_PASSWORD" envDefault:""`
    RedisDB       int    `env:"REDIS_DB" envDefault:"0"`
    
    // NATS Configuration
    NATSUrl         string `env:"NATS_URL" envDefault:"nats://localhost:4222"`
    NATSMaxReconnect int    `env:"NATS_MAX_RECONNECT" envDefault:"10"`
    NATSReconnectWait time.Duration `env:"NATS_RECONNECT_WAIT" envDefault:"2s"`
    
    // JWT Configuration
    JWTSecret     string        `env:"JWT_SECRET" envDefault:"change-me-in-production"`
    JWTExpiration time.Duration `env:"JWT_EXPIRATION" envDefault:"24h"`
    
    // OAuth Configuration
    GoogleClientID     string `env:"OAUTH_GOOGLE_CLIENT_ID"`
    GoogleClientSecret string `env:"OAUTH_GOOGLE_CLIENT_SECRET"`
    WeChatClientID     string `env:"OAUTH_WECHAT_CLIENT_ID"`
    WeChatClientSecret string `env:"OAUTH_WECHAT_CLIENT_SECRET"`
    
    // API Configuration
    CORSOrigins  []string `env:"CORS_ORIGINS" envSeparator:","`
    RateLimit    int      `env:"API_RATE_LIMIT" envDefault:"1000"`
    
    // Monitoring
    EnableMetrics bool `env:"ENABLE_METRICS" envDefault:"false"`
    MetricsPort   int  `env:"METRICS_PORT" envDefault:"9090"`
}

func LoadConfig() (*Config, error) {
    cfg := &Config{}
    if err := env.Parse(cfg); err != nil {
        return nil, fmt.Errorf("failed to parse config: %w", err)
    }
    
    // Validate required fields in production
    if cfg.Environment == "production" {
        if err := validateProductionConfig(cfg); err != nil {
            return nil, fmt.Errorf("invalid production config: %w", err)
        }
    }
    
    return cfg, nil
}
```

## 6. Monitoring and Observability

### 6.1 Application Metrics

```go
// metrics/metrics.go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    // HTTP Metrics
    HTTPRequestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "endpoint", "status_code"},
    )
    
    HTTPRequestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Help:    "HTTP request duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "endpoint"},
    )
    
    // Database Metrics
    DBConnectionsActive = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "database_connections_active",
            Help: "Number of active database connections",
        },
    )
    
    DBQueriesTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "database_queries_total",
            Help: "Total number of database queries",
        },
        []string{"query_type", "status"},
    )
    
    // Business Metrics
    UsersTotal = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "users_total",
            Help: "Total number of users",
        },
    )
    
    TransactionsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "transactions_total",
            Help: "Total number of transactions",
        },
        []string{"direction", "reason"},
    )
)
```

### 6.2 Logging Configuration

```yaml
# logging.yaml
version: 1
formatters:
  json:
    format: '{"timestamp":"%(asctime)s","level":"%(levelname)s","logger":"%(name)s","message":"%(message)s","trace_id":"%(trace_id)s"}'
  
handlers:
  console:
    class: logging.StreamHandler
    level: INFO
    formatter: json
    stream: ext://sys.stdout
  
  file:
    class: logging.handlers.RotatingFileHandler
    level: INFO
    formatter: json
    filename: /app/logs/tiris-backend.log
    maxBytes: 10485760  # 10MB
    backupCount: 5
    encoding: utf8

loggers:
  tiris:
    level: INFO
    handlers: [console, file]
    propagate: no

root:
  level: INFO
  handlers: [console]
```

### 6.3 Health Checks

```go
// health/health.go
type HealthChecker struct {
    db    *sql.DB
    redis *redis.Client
}

type HealthStatus struct {
    Status    string            `json:"status"`
    Timestamp time.Time         `json:"timestamp"`
    Checks    map[string]string `json:"checks,omitempty"`
    Version   string            `json:"version,omitempty"`
}

func (h *HealthChecker) LivenessCheck(ctx context.Context) *HealthStatus {
    return &HealthStatus{
        Status:    "alive",
        Timestamp: time.Now(),
        Version:   version.BuildVersion,
    }
}

func (h *HealthChecker) ReadinessCheck(ctx context.Context) *HealthStatus {
    checks := make(map[string]string)
    status := "ready"
    
    // Database check
    if err := h.db.PingContext(ctx); err != nil {
        checks["database"] = "error: " + err.Error()
        status = "not_ready"
    } else {
        checks["database"] = "ok"
    }
    
    // Redis check
    if err := h.redis.Ping(ctx).Err(); err != nil {
        checks["redis"] = "error: " + err.Error()
        status = "not_ready"
    } else {
        checks["redis"] = "ok"
    }
    
    return &HealthStatus{
        Status:    status,
        Timestamp: time.Now(),
        Checks:    checks,
        Version:   version.BuildVersion,
    }
}
```

## 7. Security and Secrets Management

### 7.1 Secrets Management with HashiCorp Vault

```go
// secrets/vault.go
type VaultClient struct {
    client *vault.Client
    config *VaultConfig
}

type VaultConfig struct {
    Address   string
    Token     string
    MountPath string
}

func NewVaultClient(config *VaultConfig) (*VaultClient, error) {
    client, err := vault.NewClient(&vault.Config{
        Address: config.Address,
    })
    if err != nil {
        return nil, err
    }
    
    client.SetToken(config.Token)
    
    return &VaultClient{
        client: client,
        config: config,
    }, nil
}

func (v *VaultClient) GetSecret(path string) (map[string]interface{}, error) {
    secret, err := v.client.Logical().Read(v.config.MountPath + "/" + path)
    if err != nil {
        return nil, err
    }
    
    if secret == nil {
        return nil, fmt.Errorf("secret not found: %s", path)
    }
    
    return secret.Data, nil
}
```

### 7.2 SSL/TLS Configuration

```nginx
# nginx.conf
server {
    listen 443 ssl http2;
    server_name api.tiris.ai;
    
    ssl_certificate /etc/ssl/certs/tiris.crt;
    ssl_certificate_key /etc/ssl/private/tiris.key;
    
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512:ECDHE-RSA-AES256-GCM-SHA384:DHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;
    
    add_header Strict-Transport-Security "max-age=63072000" always;
    add_header X-Content-Type-Options nosniff;
    add_header X-Frame-Options DENY;
    add_header X-XSS-Protection "1; mode=block";
    
    location / {
        proxy_pass http://tiris-backend:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## 8. Backup and Disaster Recovery

### 8.1 Database Backup Strategy

```bash
#!/bin/bash
# backup.sh - Automated database backup

set -e

BACKUP_DIR="/backups/postgres"
DATE=$(date +%Y%m%d_%H%M%S)
DB_NAME="tiris_production"
RETENTION_DAYS=30

# Create backup directory
mkdir -p $BACKUP_DIR

# Full database backup
pg_dump -h $DB_HOST -U $DB_USER -d $DB_NAME -f $BACKUP_DIR/tiris_full_$DATE.sql

# Compress backup
gzip $BACKUP_DIR/tiris_full_$DATE.sql

# Upload to S3
aws s3 cp $BACKUP_DIR/tiris_full_$DATE.sql.gz s3://tiris-backups/database/

# Cleanup old backups
find $BACKUP_DIR -name "tiris_full_*.sql.gz" -mtime +$RETENTION_DAYS -delete

# Verify backup integrity
echo "Backup completed: tiris_full_$DATE.sql.gz"
```

### 8.2 Disaster Recovery Procedures

```bash
#!/bin/bash
# restore.sh - Database restoration script

set -e

BACKUP_FILE=$1
DB_NAME="tiris_production"
TEMP_DB="tiris_recovery_$(date +%s)"

if [ -z "$BACKUP_FILE" ]; then
    echo "Usage: $0 <backup_file>"
    exit 1
fi

echo "Starting database recovery..."
echo "Backup file: $BACKUP_FILE"

# Download backup from S3 if needed
if [[ $BACKUP_FILE == s3://* ]]; then
    LOCAL_FILE="/tmp/$(basename $BACKUP_FILE)"
    aws s3 cp $BACKUP_FILE $LOCAL_FILE
    BACKUP_FILE=$LOCAL_FILE
fi

# Create temporary database for validation
createdb -h $DB_HOST -U $DB_USER $TEMP_DB

# Restore to temporary database
gunzip -c $BACKUP_FILE | psql -h $DB_HOST -U $DB_USER -d $TEMP_DB

# Validate restoration
echo "Validating restored database..."
TABLES=$(psql -h $DB_HOST -U $DB_USER -d $TEMP_DB -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';")
echo "Tables restored: $TABLES"

# Switch databases (requires downtime)
echo "Switching to restored database..."
# 1. Stop application
# 2. Rename current database
# 3. Rename temp database to production
# 4. Start application

echo "Database recovery completed"
```

## 9. Performance Optimization

### 9.1 Application Performance Tuning

```go
// config/performance.go
type PerformanceConfig struct {
    // Connection Pool Settings
    MaxOpenConns    int `env:"DB_MAX_OPEN_CONNS" envDefault:"25"`
    MaxIdleConns    int `env:"DB_MAX_IDLE_CONNS" envDefault:"5"`
    ConnMaxLifetime time.Duration `env:"DB_CONN_MAX_LIFETIME" envDefault:"1h"`
    
    // Cache Settings
    CacheSize       int           `env:"CACHE_SIZE" envDefault:"1000"`
    CacheTTL        time.Duration `env:"CACHE_TTL" envDefault:"15m"`
    
    // Worker Pool Settings
    WorkerPoolSize  int `env:"WORKER_POOL_SIZE" envDefault:"10"`
    QueueSize       int `env:"QUEUE_SIZE" envDefault:"1000"`
}

func ApplyPerformanceSettings(db *sql.DB, config *PerformanceConfig) {
    db.SetMaxOpenConns(config.MaxOpenConns)
    db.SetMaxIdleConns(config.MaxIdleConns)
    db.SetConnMaxLifetime(config.ConnMaxLifetime)
}
```

### 9.2 Load Testing

```bash
#!/bin/bash
# load-test.sh - Load testing script

# Test scenarios
echo "Running load tests..."

# Authentication endpoint
wrk -t12 -c400 -d30s -s auth-test.lua https://api.tiris.com/auth/login

# API endpoints
wrk -t12 -c400 -d30s -H "Authorization: Bearer $JWT_TOKEN" https://api.tiris.com/users/me

# Transaction creation
wrk -t12 -c200 -d60s -s transaction-test.lua https://api.tiris.com/trading-logs

echo "Load testing completed"
```

## 10. Troubleshooting and Maintenance

### 10.1 Common Issues and Solutions

**Database Connection Issues:**
```bash
# Check connection pool status
SELECT * FROM pg_stat_activity WHERE datname = 'tiris_production';

# Kill long-running queries
SELECT pg_terminate_backend(pid) FROM pg_stat_activity 
WHERE datname = 'tiris_production' AND state = 'active' AND query_start < NOW() - INTERVAL '5 minutes';
```

**Performance Issues:**
```bash
# Analyze slow queries
SELECT query, mean_time, calls, total_time 
FROM pg_stat_statements 
ORDER BY total_time DESC 
LIMIT 10;

# Check table sizes
SELECT schemaname, tablename, pg_size_pretty(pg_total_relation_size(tablename::text)) as size
FROM pg_tables 
WHERE schemaname = 'public' 
ORDER BY pg_total_relation_size(tablename::text) DESC;
```

### 10.2 Maintenance Tasks

```bash
#!/bin/bash
# maintenance.sh - Regular maintenance tasks

# Database maintenance
echo "Running database maintenance..."
psql -h $DB_HOST -U $DB_USER -d tiris_production -c "VACUUM ANALYZE;"
psql -h $DB_HOST -U $DB_USER -d tiris_production -c "REINDEX DATABASE tiris_production;"

# Clear old logs
find /app/logs -name "*.log" -mtime +7 -delete

# Docker cleanup
docker system prune -f --volumes

# Update SSL certificates
certbot renew --quiet

echo "Maintenance completed"
```

### 10.3 Monitoring and Alerting

**Prometheus Alerts:**
```yaml
# alerts.yml
groups:
- name: tiris-backend
  rules:
  - alert: HighErrorRate
    expr: sum(rate(http_requests_total{status_code=~"5.."}[5m])) / sum(rate(http_requests_total[5m])) > 0.05
    for: 5m
    labels:
      severity: critical
    annotations:
      summary: High error rate detected
      
  - alert: DatabaseConnectionHigh
    expr: database_connections_active > 20
    for: 2m
    labels:
      severity: warning
    annotations:
      summary: High database connection usage
      
  - alert: MemoryUsageHigh
    expr: container_memory_usage_bytes{name="tiris-backend"} / container_spec_memory_limit_bytes > 0.8
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: High memory usage detected
```

## 11. Resources and Domain Configuration

### 11.1 Domain Setup

**Primary Domains:**
- **Production**: `tiris.ai`
- **Development**: `dev.tiris.ai`

**Service Subdomains:**
```
Production:
├── api.tiris.ai (Backend API)
├── admin.tiris.ai (Admin Panel)
├── docs.tiris.ai (Documentation)
└── status.tiris.ai (Status Page)

Development:
├── api.dev.tiris.ai (Backend API)
├── admin.dev.tiris.ai (Admin Panel)
├── docs.dev.tiris.ai (Documentation)
└── status.dev.tiris.ai (Status Page)
```

### 11.2 DNS Configuration

**A Records:**
```
tiris.ai                → Production Load Balancer IP
*.tiris.ai              → Production Load Balancer IP
dev.tiris.ai            → Development Server IP
*.dev.tiris.ai          → Development Server IP
```

**CNAME Records:**
```
api.tiris.ai            → tiris.ai
admin.tiris.ai          → tiris.ai
docs.tiris.ai           → tiris.ai
api.dev.tiris.ai        → dev.tiris.ai
admin.dev.tiris.ai      → dev.tiris.ai
docs.dev.tiris.ai       → dev.tiris.ai
```

### 11.3 SSL Certificate Management

**Production Certificates:**
```bash
# Let's Encrypt wildcard certificate
certbot certonly --dns-route53 \
  -d tiris.ai \
  -d *.tiris.ai \
  --email admin@tiris.ai \
  --agree-tos \
  --non-interactive

# Auto-renewal setup
echo "0 12 * * * /usr/bin/certbot renew --quiet" >> /etc/crontab
```

**Development Certificates:**
```bash
# Let's Encrypt for development
certbot certonly --dns-route53 \
  -d dev.tiris.ai \
  -d *.dev.tiris.ai \
  --email admin@tiris.ai \
  --agree-tos \
  --non-interactive
```

### 11.4 Environment-Specific Configuration

**Production Deployment Commands:**
```bash
# Deploy to production
export DOMAIN=tiris.ai
export API_URL=https://api.tiris.ai/v1
export FRONTEND_URL=https://tiris.ai
docker-compose -f docker-compose.prod.yml up -d
```

**Development Deployment Commands:**
```bash
# Deploy to development
export DOMAIN=dev.tiris.ai
export API_URL=https://api.dev.tiris.ai/v1
export FRONTEND_URL=https://dev.tiris.ai
docker-compose -f docker-compose.dev.yml up -d
```

### 11.5 Load Balancer Configuration

**Production (api.tiris.ai):**
```nginx
upstream backend_prod {
    server tiris-backend-1:8080;
    server tiris-backend-2:8080;
    server tiris-backend-3:8080;
}

server {
    listen 443 ssl http2;
    server_name api.tiris.ai;
    
    location / {
        proxy_pass http://backend_prod;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

**Development (api.dev.tiris.ai):**
```nginx
server {
    listen 443 ssl http2;
    server_name api.dev.tiris.ai;
    
    location / {
        proxy_pass http://tiris-backend-dev:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

This comprehensive deployment and DevOps guide provides the foundation for reliable, scalable, and maintainable deployment of the Tiris Backend system across both production (tiris.ai) and development (dev.tiris.ai) environments.