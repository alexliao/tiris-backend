# Test Database Setup Guide

This guide explains how to set up PostgreSQL test databases for running integration tests in the Tiris Backend project.

## Overview

The integration tests require a PostgreSQL database to test real database interactions. We provide two approaches:

1. **Local PostgreSQL Setup** - Use an existing PostgreSQL installation
2. **Docker-based Setup** - Use Docker containers (recommended for development)

## Quick Start

### Option 1: Docker Setup (Recommended)

```bash
# Start test database with Docker
make setup-test-db-docker

# Run integration tests
make test-integration-docker

# Stop test database when done
make stop-test-db-docker
```

### Option 2: Local PostgreSQL Setup

```bash
# Set up test database (requires local PostgreSQL)
make setup-test-db

# Run integration tests
make test-integration
```

## Detailed Setup Instructions

### Docker-based Setup

1. **Prerequisites**
   - Docker and Docker Compose installed
   - No local PostgreSQL conflicts on port 5433

2. **Start Test Database**
   ```bash
   make setup-test-db-docker
   ```
   This will:
   - Start PostgreSQL container on port 5433
   - Create `tiris_test` database and user
   - Wait for database to be ready
   - Run initialization scripts

3. **Run Tests**
   ```bash
   make test-integration-docker
   ```

4. **Clean Up**
   ```bash
   # Stop database
   make stop-test-db-docker
   
   # Remove all data volumes
   make clean-test-db-docker
   ```

### Local PostgreSQL Setup

1. **Prerequisites**
   - PostgreSQL server running locally
   - `psql` command available in PATH
   - PostgreSQL admin access (usually `postgres` user)

2. **Set Up Test Database**
   ```bash
   make setup-test-db
   ```
   
   Or with custom connection details:
   ```bash
   ./scripts/setup-test-db.sh --host localhost --port 5432 --admin-user postgres
   ```

3. **Run Tests**
   ```bash
   make test-integration
   ```

4. **Clean Up**
   ```bash
   make clean-test-db
   ```

## Configuration

### Environment Variables

You can customize the database connection using environment variables:

```bash
export TEST_DB_HOST=localhost
export TEST_DB_PORT=5432
export TEST_DB_USER=tiris_test
export TEST_DB_PASSWORD=tiris_test
export TEST_DB_NAME=tiris_test
```

### Script Options

The setup script supports various options:

```bash
./scripts/setup-test-db.sh --help
```

Common options:
- `--host HOST` - PostgreSQL host (default: localhost)
- `--port PORT` - PostgreSQL port (default: 5432)
- `--admin-user USER` - Admin user (default: postgres)
- `--admin-password PASS` - Admin password
- `--drop-existing` - Drop and recreate existing database

## Test Database Details

### Created Resources

The setup creates:
- **User**: `tiris_test` with password `tiris_test`
- **Database**: `tiris_test` owned by `tiris_test`
- **Privileges**: Full access to database and schema

### Connection String

```
postgresql://tiris_test:tiris_test@localhost:5432/tiris_test?sslmode=disable
```

For Docker setup:
```
postgresql://tiris_test:tiris_test@localhost:5433/tiris_test?sslmode=disable
```

## Available Test Commands

| Command | Description |
|---------|-------------|
| `make test` | Run all tests (unit + integration) |
| `make test-unit` | Run unit tests only |
| `make test-integration` | Run integration tests (local DB) |
| `make test-integration-docker` | Run integration tests (Docker DB) |
| `make setup-test-db` | Set up local test database |
| `make setup-test-db-docker` | Start Docker test database |
| `make clean-test-db` | Clean local test database |
| `make stop-test-db-docker` | Stop Docker test database |
| `make clean-test-db-docker` | Remove Docker data volumes |

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15-alpine
        env:
          POSTGRES_PASSWORD: postgres
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 5432:5432

    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Set up test database
        run: make setup-test-db
        
      - name: Run tests
        run: make test
```

### GitLab CI Example

```yaml
test:
  image: golang:1.21
  services:
    - postgres:15-alpine
  variables:
    POSTGRES_DB: postgres
    POSTGRES_USER: postgres
    POSTGRES_PASSWORD: postgres
    TEST_DB_HOST: postgres
  script:
    - make setup-test-db
    - make test
```

## Troubleshooting

### Common Issues

1. **Connection Refused**
   ```
   Error: connection refused
   ```
   - Ensure PostgreSQL is running
   - Check host and port settings
   - Verify firewall settings

2. **Authentication Failed**
   ```
   Error: password authentication failed
   ```
   - Check admin credentials
   - Use `--admin-password` flag
   - Verify user permissions

3. **Database Already Exists**
   ```
   Error: database "tiris_test" already exists
   ```
   - Use `--drop-existing` flag to recreate
   - Or manually clean up: `make clean-test-db`

4. **Docker Port Conflicts**
   ```
   Error: port 5433 already in use
   ```
   - Stop conflicting services
   - Or modify port in `docker-compose.test.yml`

### Debug Commands

```bash
# Check PostgreSQL connection
psql -h localhost -p 5432 -U postgres -c "SELECT version();"

# Test database connection
psql -h localhost -p 5432 -U tiris_test -d tiris_test -c "SELECT current_user, current_database();"

# Check Docker container logs
docker logs tiris-postgres-test

# Check container status
docker compose -f docker-compose.test.yml ps
```

## Security Considerations

### Test Environment Only

The test database:
- Uses simple credentials (`tiris_test`/`tiris_test`)
- Has relaxed security settings
- Should NEVER be used in production

### Network Isolation

- Docker setup uses isolated network
- Local setup uses localhost only
- No external access by default

## Performance Optimization

### PostgreSQL Settings

The Docker setup uses optimized settings for testing:
- Reduced fsync overhead
- Increased shared buffers
- Optimized for fast startup

### Test Data Management

- Tests use transaction rollback for cleanup
- Separate test database prevents conflicts
- In-memory options available for speed

## Development Workflow

### Typical Development Flow

```bash
# 1. Start development environment
make setup-test-db-docker

# 2. Run tests during development
make test-unit          # Fast feedback
make test-integration-docker  # Full validation

# 3. Clean up when done
make stop-test-db-docker
```

### Continuous Testing

```bash
# Watch mode (if you have a file watcher)
make test-unit

# Or use Go's built-in test caching
go test -count=1 ./...
```