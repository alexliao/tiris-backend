# Port Availability Checker

The Tiris Backend project includes a comprehensive port checker script to ensure all required ports are available before starting services.

## Quick Usage

```bash
# Check development ports (most common)
make check-ports

# Check all ports (dev + monitoring + production)
make check-ports-all

# See detailed port usage
make check-ports-detailed

# Kill processes on development ports
make kill-ports
```

## Script Usage

```bash
# Direct script usage
./scripts/check-ports.sh [OPTION]

# Available options:
./scripts/check-ports.sh --dev          # Development ports only
./scripts/check-ports.sh --monitoring   # Monitoring ports only
./scripts/check-ports.sh --prod         # Production ports only
./scripts/check-ports.sh --all          # All ports
./scripts/check-ports.sh --detailed     # Show what's using each port
./scripts/check-ports.sh --kill-dev     # Free up development ports
./scripts/check-ports.sh --help         # Show help
```

## Port Reference

### Development Ports (Required)
- **8080** - API Server (REST endpoints, health checks, metrics)
- **5432** - PostgreSQL/TimescaleDB database
- **6379** - Redis cache and rate limiting
- **4222** - NATS client protocol
- **8222** - NATS HTTP monitoring

### Monitoring Ports (Optional)
- **3000** - Grafana dashboard
- **9090** - Prometheus server
- **9093** - AlertManager
- **3100** - Loki log aggregation
- **9080** - Promtail log collection

### Production Ports
- **80** - HTTP (redirects to HTTPS)
- **443** - HTTPS/TLS
- **5432** - PostgreSQL (internal)
- **6379** - Redis (internal)
- **4222** - NATS (internal)

### Development/Debug Ports
- **2345** - Delve debugger (container only)

## Common Port Conflicts

### Port 8080 (API Server)
```bash
# Find what's using the port
lsof -i :8080

# Kill the process
./scripts/check-ports.sh --kill-dev
# OR
lsof -ti:8080 | xargs kill -9
```

### Port 5432 (PostgreSQL)
Usually caused by:
- Local PostgreSQL installation
- Other Docker PostgreSQL containers

```bash
# Check for other PostgreSQL processes
ps aux | grep postgres

# Stop Docker containers
docker compose -f docker-compose.dev.yml down
```

### Port 3000 (Grafana/React)
Usually caused by:
- React development servers
- Other Node.js applications

```bash
# Kill Node.js processes on port 3000
lsof -ti:3000 | xargs kill -9
```

## Troubleshooting

### Script Not Working
1. Make sure the script is executable:
   ```bash
   chmod +x scripts/check-ports.sh
   ```

2. Check if required tools are available:
   ```bash
   which lsof  # Should be available on macOS/Linux
   ```

### Persistent Port Conflicts
1. Check Docker containers:
   ```bash
   docker ps
   docker compose -f docker-compose.dev.yml down
   ```

2. Check system services:
   ```bash
   sudo netstat -tlnp | grep :5432  # Linux
   lsof -i :5432                    # macOS
   ```

3. Restart network services (last resort):
   ```bash
   # macOS
   sudo dscacheutil -flushcache
   
   # Linux
   sudo systemctl restart networking
   ```

### Permission Issues
If you get permission denied errors when killing processes:

```bash
# Use sudo for system processes
sudo ./scripts/check-ports.sh --kill-dev

# Or kill specific processes manually
sudo kill -9 <PID>
```

## Integration with Development Workflow

### Before Starting Development
```bash
# 1. Check ports are available
make check-ports

# 2. Start services if ports are free
docker compose -f docker-compose.dev.yml up -d

# 3. Run migrations
make migrate-up

# 4. Start the application
make run
```

### Before Deployment
```bash
# Check production ports
./scripts/check-ports.sh --prod

# Check all monitoring ports
./scripts/check-ports.sh --monitoring
```

### Automated Checks
You can integrate the port checker into your CI/CD pipeline:

```bash
# In your CI script
./scripts/check-ports.sh --dev || exit 1
```

## Output Examples

### All Ports Available
```
=== Development Ports ===
✓ Port 8080 (API Server) - AVAILABLE
✓ Port 5432 (PostgreSQL/TimescaleDB) - AVAILABLE
✓ Port 6379 (Redis Cache) - AVAILABLE
✓ Port 4222 (NATS Client Protocol) - AVAILABLE
✓ Port 8222 (NATS HTTP Monitoring) - AVAILABLE

Summary: 5/5 ports available
✓ All Development Ports ports are available
```

### Port Conflicts Detected
```
=== Development Ports ===
✗ Port 8080 (API Server) - IN USE by PID 12345 (node)
✓ Port 5432 (PostgreSQL/TimescaleDB) - AVAILABLE
✓ Port 6379 (Redis Cache) - AVAILABLE
✓ Port 4222 (NATS Client Protocol) - AVAILABLE
✓ Port 8222 (NATS HTTP Monitoring) - AVAILABLE

Summary: 4/5 ports available
✗ Some Development Ports ports are in use
```

### Detailed Usage
```
=== Detailed Port Usage ===

Active listeners on project ports:

Port 8080 (API Server):
  PID 12345    CMD: node

Docker containers with exposed ports:
NAMES               PORTS
tiris-postgres-dev  0.0.0.0:5432->5432/tcp
```