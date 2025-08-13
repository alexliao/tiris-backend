# Docker Development Environment

This directory contains the complete Docker setup for the Tiris Backend development environment.

## Quick Start

1. **Start the development environment:**
   ```bash
   docker-compose -f docker-compose.dev.yml up -d
   ```

2. **Run database migrations:**
   ```bash
   docker-compose -f docker-compose.dev.yml run --rm migrate
   ```

3. **Set up NATS streams:**
   ```bash
   docker-compose -f docker-compose.dev.yml --profile setup run --rm nats-setup
   ```

4. **View application logs:**
   ```bash
   docker-compose -f docker-compose.dev.yml logs -f app
   ```

## Services

### Core Services
- **PostgreSQL + TimescaleDB** (`postgres:5432`) - Main database with time-series capabilities
- **NATS JetStream** (`nats:4222`) - Message streaming with event deduplication
- **Redis** (`redis:6379`) - Caching and rate limiting storage

### Application Services
- **App** (`app:8080`) - Main application with hot reload via Air
- **Migrate** - Database migration service (run once)
- **NATS Setup** - Stream and consumer configuration (run once)
- **Tools** - Development utilities container

## Environment Variables

The application runs with the following development configuration:

```bash
ENV=development
DB_HOST=postgres
DB_PORT=5432
DB_NAME=tiris_dev
DB_USER=tiris_user
DB_PASSWORD=tiris_password
DB_SSL_MODE=disable
NATS_URL=nats://nats:4222
REDIS_URL=redis://:redis_password@redis:6379/0
JWT_SECRET=dev_jwt_secret_key_change_in_production
REFRESH_SECRET=dev_refresh_secret_key_change_in_production
JWT_EXPIRATION=3600
REFRESH_EXPIRATION=604800
LOG_LEVEL=debug
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:3001
```

## Development Workflow

### Hot Reload
The application automatically reloads when Go source files change using Air:
- Configuration: `.air.toml`
- Excluded: `*_test.go` files, `tmp/`, `coverage/`
- Build command: `go build -o ./tmp/main cmd/server/main.go`

### Debugging
Delve debugger is available on port `2345`:
```bash
# Connect with your IDE or use command line
dlv connect localhost:2345
```

### Database Access
Connect to PostgreSQL directly:
```bash
# Using docker exec
docker exec -it tiris-postgres-dev psql -U tiris_user -d tiris_dev

# Using psql client
psql -h localhost -p 5432 -U tiris_user -d tiris_dev
```

### NATS Monitoring
Access NATS monitoring dashboard at: http://localhost:8222

View NATS statistics:
```bash
# Stream information
docker exec tiris-nats-setup nats --server=nats:4222 stream ls

# Consumer information  
docker exec tiris-nats-setup nats --server=nats:4222 consumer ls TRADING_EVENTS
```

## Useful Commands

```bash
# Start all services
docker-compose -f docker-compose.dev.yml up -d

# Stop all services
docker-compose -f docker-compose.dev.yml down

# Rebuild application container
docker-compose -f docker-compose.dev.yml build app

# View service logs
docker-compose -f docker-compose.dev.yml logs -f [service_name]

# Run tests in tools container
docker-compose -f docker-compose.dev.yml --profile tools run --rm tools go test ./...

# Generate test coverage
docker-compose -f docker-compose.dev.yml --profile tools run --rm tools ./scripts/coverage.sh

# Clean up volumes (WARNING: destroys data)
docker-compose -f docker-compose.dev.yml down -v
```

## Persistent Data

The following volumes persist data between container restarts:
- `postgres_data` - Database data
- `nats_data` - NATS JetStream data
- `redis_data` - Redis data
- `go_modules` - Go module cache

## Networking

All services run on the `tiris-dev-network` bridge network with the following ports exposed:
- Application: `8080`
- PostgreSQL: `5432`
- NATS: `4222` (client), `8222` (monitoring)
- Redis: `6379`
- Delve Debugger: `2345`

## Troubleshooting

### Common Issues

1. **Port conflicts**: Ensure ports 5432, 4222, 6379, 8080, 2345 are not in use
2. **Permission issues**: The app runs as non-root user `appuser`
3. **Database connection**: Wait for health checks to pass before connecting
4. **NATS streams**: Run the nats-setup service if streams are missing

### Health Checks

All services include health checks:
```bash
# Check service health
docker-compose -f docker-compose.dev.yml ps

# View health check logs
docker inspect tiris-postgres-dev --format='{{.State.Health}}'
```

### Reset Environment

To completely reset the development environment:
```bash
# Stop and remove containers, networks, and volumes
docker-compose -f docker-compose.dev.yml down -v

# Remove all related images
docker images | grep tiris | awk '{print $3}' | xargs docker rmi

# Start fresh
docker-compose -f docker-compose.dev.yml up -d
docker-compose -f docker-compose.dev.yml run --rm migrate
docker-compose -f docker-compose.dev.yml --profile setup run --rm nats-setup
```