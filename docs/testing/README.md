# Testing Documentation

This directory contains comprehensive testing documentation for the Tiris Backend project.

## 📋 Documents Overview

### Core Testing Strategy
- **[Test Strategy](./test-strategy.md)** - Overall testing approach and methodology
- **[API Testing Updates](./TEST_API_UPDATES.md)** - Recent changes to API test scripts  
- **[Trading Log Testing](./TESTING.md)** - Business logic testing for trading logs

## 🧪 Test Categories

### Unit Testing
- Service layer business logic validation
- Repository data access testing
- Utility function testing
- Mock-based isolation testing

### Integration Testing
- Full API endpoint testing with real database
- Multi-service interaction testing
- External service integration validation

### Performance Testing
- Load testing and benchmarking
- Resource usage monitoring
- Connection pooling validation

### Security Testing
- Authentication and authorization flows
- Input validation and sanitization
- JWT token security testing

## 🚀 Quick Start

### Run All Tests
```bash
# Run comprehensive test suite
make test

# Run unit tests only
make test-unit

# Run integration tests only (requires test database)
make test-integration

# Run tests with coverage reporting
make test-coverage
```

### Test Database Setup
```bash
# Set up test database (Docker)
make setup-test-db-docker

# Set up test database (local PostgreSQL)
make setup-test-db
```

### API Testing
```bash
# Run API integration tests
./scripts/test-api.sh

# Validate deployment
./scripts/validate-deployment.sh
```

## 📊 Coverage Reports

Coverage reports are automatically generated and stored in the `coverage/` directory:
- **HTML Report**: `coverage/coverage.html` 
- **Profile Data**: `coverage/coverage.out`

## 🔧 Testing Tools

- **Framework**: Go testing package with testify/suite
- **Mocking**: testify/mock for service layer isolation
- **Database**: In-memory SQLite for unit tests, PostgreSQL for integration
- **HTTP Testing**: Gin test context with httptest recorder
- **Coverage**: Go built-in coverage tools with HTML reporting

## 📚 Additional Resources

- [Go Testing Best Practices](https://golang.org/doc/code.html#Testing)
- [Testify Documentation](https://github.com/stretchr/testify)
- [Testing HTTP Handlers in Go](https://golang.org/pkg/net/http/httptest/)