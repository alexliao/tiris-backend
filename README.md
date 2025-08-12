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
- PostgreSQL 15+ with TimescaleDB extension
- NATS 2.10+ with JetStream

### Development Setup

1. Clone the repository
2. Copy environment configuration:
   ```bash
   cp .env.example .env
   ```
3. Edit `.env` with your configuration
4. Install dependencies:
   ```bash
   make deps
   ```
5. Run the application:
   ```bash
   make run
   ```

### Available Commands

```bash
make build          # Build the application
make run            # Run the application
make dev            # Run with hot reload
make test           # Run tests
make test-coverage  # Run tests with coverage
make lint           # Run linter
make fmt            # Format code
make setup          # Initial development setup
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

### Database Migrations

```bash
make migrate-up     # Apply migrations
make migrate-down   # Rollback migrations
```

### Testing

```bash
make test           # Run all tests
make test-coverage  # Run tests with coverage report
```

### Docker

```bash
make docker-build   # Build Docker image
make docker-run     # Run in container
```

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

- [System Definition](docs/system-definition.md)
- [Requirements](docs/requirements.md)
- [Architecture Design](docs/architecture.md)
- [API Specification](docs/api-specification.md)
- [Database Schema](docs/database-schema.md)
- [Implementation Roadmap](docs/implementation-roadmap.md)

## License

Copyright © 2024 Tiris. All rights reserved.