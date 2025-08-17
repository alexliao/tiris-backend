# Tiris Backend

A Go-based microservice for quantitative trading data management, providing RESTful APIs for user management, exchange integration, and trading operations.

## Features

- User authentication with OAuth (Google, WeChat)
- Exchange API integration and management
- Sub-account and balance management
- Trading log and transaction tracking
- Event-driven architecture with NATS JetStream
- TimescaleDB for time-series data optimization
- Comprehensive API with rate limiting and security

## Project Structure

```
├── cmd/
│   └── server/          # Application entry point
├── internal/
│   ├── api/            # HTTP handlers and routes
│   ├── config/         # Configuration management
│   ├── database/       # Database connection and utilities
│   ├── middleware/     # HTTP middleware
│   ├── models/         # Database models
│   ├── repositories/   # Data access layer
│   ├── services/       # Business logic layer
│   └── utils/          # Utility functions
├── pkg/
│   ├── auth/           # Authentication utilities
│   ├── crypto/         # Encryption utilities
│   └── validator/      # Validation utilities
├── migrations/         # Database migrations
└── docs/              # Project documentation
```

## Quick Start

### Prerequisites

- Go 1.21+
- Docker & Docker Compose
- Make (for running commands)

### 🚀 Starting Development

#### 1. Clone and Setup
```bash
# Clone the repository
git clone <repository-url>
cd tiris-backend

# Check if required ports are available
make check-ports

# If ports are in use, free them up
make kill-ports
```

#### 2. Start Infrastructure Services
```bash
# Start PostgreSQL, NATS, and Redis
docker compose -f docker-compose.dev.yml up -d postgres nats redis

# Wait for services to be ready (30-60 seconds)
docker compose -f docker-compose.dev.yml ps
```

#### 3. Setup Database
```bash
# Run database migrations (database is created automatically on first run)
make migrate-up

# Verify tables were created
docker exec tiris-postgres-dev psql -U tiris_user -d tiris_dev -c "\dt"
```

#### 4. Start the Application
```bash
# Start the API server
make run

# The application will be available at:
# - API: http://localhost:8080/v1
# - Health: http://localhost:8080/health
# - Metrics: http://localhost:8080/metrics
```

#### 5. Create a Test User (for API Testing)
```bash
# Create a test user with OAuth authentication
make create-test-user ARGS="--name 'Your Name'"

# Or use the script directly
./scripts/create-test-user.sh --name "Developer User"

# This creates a user with 1-year token validity for testing
```

#### 6. Verify Everything is Working
```bash
# Check health endpoints
curl http://localhost:8080/health/live
curl http://localhost:8080/health/ready
curl http://localhost:8080/health

# All should return "healthy" status

# Test API with your new user (use token from step 5)
curl -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
     http://localhost:8080/v1/users/me
```

### 🛑 Stopping Development

#### Option 1: Stop Everything
```bash
# Stop the application (Ctrl+C if running in foreground)
# Then stop all Docker services
docker compose -f docker-compose.dev.yml down

# This stops and removes containers but keeps data
```

#### Option 2: Stop with Data Cleanup
```bash
# Stop and remove all data (database, logs, etc.)
docker compose -f docker-compose.dev.yml down -v

# Use this for a completely fresh start
```

#### Option 3: Emergency Port Cleanup
```bash
# If services are stuck, force kill all processes on project ports
make kill-ports

# Check what's still running
make check-ports-detailed
```

### 📋 Development Commands

#### Essential Commands
```bash
make check-ports       # Check if development ports are available
make kill-ports        # Kill processes blocking development ports
make run              # Run the application
make test             # Run all tests
make migrate-up       # Apply database migrations
make migrate-down     # Rollback database migrations
```

#### Additional Commands
```bash
make build            # Build the application binary
make dev              # Run with hot reload (requires air)
make test-coverage    # Run tests with coverage report
make lint             # Run code linter
make fmt              # Format code
make setup            # Initial development setup
make help             # Show all available commands
```

#### Port Checking Commands
```bash
make check-ports           # Check development ports (8080, 5432, 6379, 4222, 8222)
make check-ports-all       # Check all ports including monitoring
make check-ports-detailed  # Show detailed port usage information
make kill-ports           # Kill processes on development ports
make create-test-user     # Create OAuth test user (use ARGS for options)
```

### 🔧 Troubleshooting

#### Port Conflicts
```bash
# Check what's using your ports
make check-ports-detailed

# Kill specific processes
lsof -ti:8080 | xargs kill -9  # Kill process on port 8080

# Or use the automated cleanup
make kill-ports
```

#### Database Issues
```bash
# Reset database completely
docker compose -f docker-compose.dev.yml down -v
docker compose -f docker-compose.dev.yml up -d postgres
make migrate-up
```

#### NATS Issues
```bash
# Restart NATS service
docker compose -f docker-compose.dev.yml restart nats

# Check NATS logs
docker compose -f docker-compose.dev.yml logs nats
```

#### Application Won't Start
```bash
# Check if all dependencies are healthy
curl http://localhost:8080/health/ready

# Check application logs for specific errors
# Look for database connection, NATS connection, or port binding issues
```

### 🐳 Using Docker for Development

#### Full Docker Development (Optional)
```bash
# Run everything in Docker including the app
docker compose -f docker-compose.dev.yml up -d

# This includes hot reload and debugging support
# App will be available at http://localhost:8080
```

#### Monitoring Stack (Optional)
```bash
# Start monitoring services (Grafana, Prometheus, etc.)
docker compose -f docker-compose.monitoring.yml up -d

# Access:
# - Grafana: http://localhost:3000
# - Prometheus: http://localhost:9090
```

## Configuration

All configuration is managed through environment variables. See `.env.example` for all available options.

### Required Environment Variables

- `JWT_SECRET` - JWT signing secret
- `REFRESH_SECRET` - Refresh token secret
- `GOOGLE_CLIENT_ID` - Google OAuth client ID
- `GOOGLE_CLIENT_SECRET` - Google OAuth client secret
- `WECHAT_APP_ID` - WeChat OAuth app ID
- `WECHAT_APP_SECRET` - WeChat OAuth app secret

## API Documentation

- Base URL: `https://api.tiris.ai/v1` (production)
- Development URL: `https://api.dev.tiris.ai/v1`
- Full API specification: [docs/api-specification.md](docs/api-specification.md)

### Health Checks

- `GET /health/live` - Liveness probe
- `GET /health/ready` - Readiness probe with dependency checks

## Development

### Environment Configuration

The application requires several environment variables. A complete `.env` file is automatically created during setup with development defaults:

```bash
# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USERNAME=tiris_user
DB_PASSWORD=tiris_password
DB_NAME=tiris_dev
DB_SSL_MODE=disable

# NATS Configuration
NATS_URL=nats://localhost:4222

# Authentication Secrets
JWT_SECRET=dev_jwt_secret_key_change_in_production
REFRESH_SECRET=dev_refresh_secret_key_change_in_production

# OAuth Configuration (required for authentication)
GOOGLE_CLIENT_ID=dummy
GOOGLE_CLIENT_SECRET=dummy
GOOGLE_REDIRECT_URL=http://localhost:8080/auth/google/callback

WECHAT_APP_ID=dummy
WECHAT_APP_SECRET=dummy
WECHAT_REDIRECT_URL=http://localhost:8080/auth/wechat/callback
```

### Port Reference

**Development Ports:**
- `8080` - API Server (REST endpoints, health checks, metrics)
- `5432` - PostgreSQL/TimescaleDB database
- `6379` - Redis cache and rate limiting  
- `4222` - NATS client protocol
- `8222` - NATS HTTP monitoring

**Monitoring Ports (optional):**
- `3000` - Grafana dashboard
- `9090` - Prometheus server
- `9093` - AlertManager

See [Port Checker Documentation](docs/PORT_CHECKER.md) for detailed port management.

## Architecture

The application follows a layered architecture pattern:

- **API Layer**: HTTP request handling and routing
- **Service Layer**: Business logic implementation
- **Repository Layer**: Data access abstraction
- **Database Layer**: PostgreSQL with TimescaleDB

### Event-Driven Architecture

Uses NATS JetStream for processing trading events from tiris-bot:
- Order events (created, filled, cancelled)
- Balance updates
- System events and errors

## Security

- JWT-based authentication
- OAuth 2.0 integration (Google, WeChat)
- API rate limiting
- Encrypted storage of sensitive data
- HTTPS-only communication

## Documentation

### Development & Operations
- [Port Checker Guide](docs/PORT_CHECKER.md) - Port management and troubleshooting
- [Test User Creation Guide](docs/TEST_USER_CREATION.md) - Creating OAuth test users for API development

### System Documentation
- [System Definition](docs/system-definition.md)
- [Requirements](docs/requirements.md)
- [Architecture Design](docs/architecture.md)
- [API Specification](docs/api-specification.md)
- [Database Schema](docs/database-schema.md)
- [Implementation Roadmap](docs/implementation-roadmap.md)

## License

Copyright © 2025 Tiris AI. All rights reserved.