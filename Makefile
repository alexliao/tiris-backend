.PHONY: build build-migrate run test test-unit test-integration test-integration-docker test-coverage clean dev deps migrate-up migrate-down migrate-version docker-build docker-run check-ports create-test-user setup-test-db clean-test-db setup-test-db-docker stop-test-db-docker clean-test-db-docker docs-generate docs-serve docs-validate docs-clean

# Build the application
build:
	go build -o bin/server cmd/server/main.go

# Build migration tool
build-migrate:
	go build -o bin/migrate cmd/migrate/main.go

# Run the application
run:
	go run cmd/server/main.go

# Run in development mode with hot reload (requires air)
dev:
	@which air > /dev/null || (echo "air not found, installing..." && go install github.com/cosmtrek/air@latest)
	air

# Run all tests (unit + integration)
test: setup-test-db-docker
	@echo "Running comprehensive test suite..."
	@TEST_DB_HOST=localhost TEST_DB_PORT=5433 TEST_DB_USER=tiris_test TEST_DB_PASSWORD=tiris_test TEST_DB_NAME=tiris_test go test -v ./... && echo "✅ All tests completed successfully!" || (echo "❌ Some tests failed - check output above" && exit 1)

# Run unit tests only
test-unit:
	go test -v -short ./...

# Run integration tests only (requires test database)
test-integration: setup-test-db
	go test -v ./internal/integration/... ./internal/performance/...

# Run integration tests with Docker database
test-integration-docker: setup-test-db-docker
	@echo "Running integration tests with Docker database..."
	@TEST_DB_HOST=localhost TEST_DB_PORT=5433 TEST_DB_USER=tiris_test TEST_DB_PASSWORD=tiris_test TEST_DB_NAME=tiris_test go test -v ./internal/integration/... ./internal/performance/...

# Run tests with coverage
test-coverage:
	mkdir -p coverage
	go test -v -short -coverprofile=coverage/coverage.out ./...
	go tool cover -html=coverage/coverage.out -o coverage/coverage.html

# Clean build artifacts
clean:
	rm -rf bin/
	rm -rf coverage/

# Install dependencies
deps:
	go mod download
	go mod tidy

# Database migrations
migrate-up: build-migrate
	./bin/migrate up

migrate-down: build-migrate
	./bin/migrate down

migrate-version: build-migrate
	./bin/migrate version

# Docker commands
docker-build:
	docker build -t tiris-backend .

docker-run:
	docker run -p 8080:8080 --env-file .env tiris-backend

# Linting
lint:
	@which golangci-lint > /dev/null || (echo "golangci-lint not found, installing..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run

# Format code
fmt:
	go fmt ./...

# Development setup
setup: deps
	cp .env.example .env
	@echo "Please edit .env file with your configuration"

# Port checking
check-ports:
	@./scripts/check-ports.sh --dev

check-ports-all:
	@./scripts/check-ports.sh --all

check-ports-detailed:
	@./scripts/check-ports.sh --detailed

kill-ports:
	@./scripts/check-ports.sh --kill-dev

# Test database setup
setup-test-db:
	@echo "Setting up test database..."
	@./scripts/setup-test-db.sh

# Clean test database
clean-test-db:
	@echo "Cleaning test database..."
	@./scripts/setup-test-db.sh --drop-existing

# Docker-based test database (alternative to local PostgreSQL)
setup-test-db-docker: build-migrate
	@echo "Starting test database with Docker..."
	@docker compose -f docker-compose.test.yml up -d postgres-test --remove-orphans
	@echo "Waiting for database to be ready..."
	@echo "Waiting for PostgreSQL to be ready (max 30 seconds)..."
	@for i in `seq 1 30`; do \
		if docker compose -f docker-compose.test.yml exec postgres-test pg_isready -U postgres >/dev/null 2>&1; then \
			break; \
		fi; \
		echo "Waiting... ($$i/30)"; \
		sleep 1; \
	done
	@echo "Setting up database schema..."
	@echo "Applying initial schema migration..."
	@PGPASSWORD=tiris_test psql -h localhost -p 5433 -U tiris_test -d tiris_test -f migrations/000001_initial_schema.up.sql || { echo "Warning: Initial schema migration failed (possibly due to missing TimescaleDB)"; }
	@echo "Applying soft delete columns migration..."
	@PGPASSWORD=tiris_test psql -h localhost -p 5433 -U tiris_test -d tiris_test -f migrations/000002_add_soft_delete_columns.up.sql || { echo "Error: Soft delete columns migration failed"; exit 1; }
	@echo "Applying uniqueness constraints migration..."
	@PGPASSWORD=tiris_test psql -h localhost -p 5433 -U tiris_test -d tiris_test -f migrations/000003_add_uniqueness_constraints.up.sql || { echo "Error: Uniqueness constraints migration failed"; exit 1; }
	@echo "Applying soft deletion compatibility migration..."
	@PGPASSWORD=tiris_test psql -h localhost -p 5433 -U tiris_test -d tiris_test -f migrations/000004_fix_soft_delete_unique_constraints.up.sql || { echo "Error: Soft deletion compatibility migration failed"; exit 1; }
	@echo "Database schema setup completed successfully (migration tracking skipped for test environment)"
	@echo "Test database is ready!"

# Stop Docker test database
stop-test-db-docker:
	@echo "Stopping test database..."
	@docker compose -f docker-compose.test.yml down

# Clean Docker test database volumes
clean-test-db-docker:
	@echo "Cleaning test database volumes..."
	@docker compose -f docker-compose.test.yml down -v

# Create test user
create-test-user:
	@./scripts/create-test-user.sh $(ARGS)

# Documentation generation
docs-generate:
	@echo "Generating API documentation..."
	@which swag > /dev/null || (echo "Installing swag..." && go install github.com/swaggo/swag/cmd/swag@latest)
	@$(shell go env GOPATH)/bin/swag init -g cmd/server/main.go --output docs/

docs-serve: docs-generate
	@echo "Starting server with documentation at http://localhost:8080/docs"
	@make run

docs-validate:
	@echo "Validating Swagger annotations..."
	@which swag > /dev/null || (echo "Installing swag..." && go install github.com/swaggo/swag/cmd/swag@latest)
	@$(shell go env GOPATH)/bin/swag init -g cmd/server/main.go --output docs/ --parseVendor

docs-clean:
	@echo "Cleaning generated documentation..."
	@rm -f docs/docs.go docs/swagger.json docs/swagger.yaml

# Help
help:
	@echo "Available commands:"
	@echo "  build         - Build the application"
	@echo "  run          - Run the application"
	@echo "  dev          - Run with hot reload"
	@echo "  test         - Run all tests (unit + integration)"
	@echo "  test-unit    - Run unit tests only"
	@echo "  test-integration - Run integration tests (sets up test DB)"
	@echo "  test-integration-docker - Run integration tests with Docker DB"
	@echo "  test-coverage - Run tests with coverage"
	@echo "  clean        - Clean build artifacts"
	@echo "  deps         - Install dependencies"
	@echo "  migrate-up   - Run database migrations up"
	@echo "  migrate-down - Run database migrations down"
	@echo "  migrate-version - Show current migration version"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run Docker container"
	@echo "  lint         - Run linter"
	@echo "  fmt          - Format code"
	@echo "  setup        - Initial development setup"
	@echo "  check-ports  - Check development port availability"
	@echo "  check-ports-all - Check all ports"
	@echo "  check-ports-detailed - Show detailed port usage"
	@echo "  kill-ports   - Kill processes on development ports"
	@echo "  setup-test-db - Set up PostgreSQL test database"
	@echo "  clean-test-db - Clean/drop test database"
	@echo "  setup-test-db-docker - Start test database with Docker"
	@echo "  stop-test-db-docker - Stop Docker test database"
	@echo "  clean-test-db-docker - Clean Docker test database volumes"
	@echo "  create-test-user - Create a test user (use ARGS for options, supports different deployments)"
	@echo "  docs-generate - Generate API documentation from code annotations"
	@echo "  docs-serve   - Generate docs and start server (docs at /docs)"
	@echo "  docs-validate - Validate Swagger annotations"
	@echo "  docs-clean   - Clean generated documentation files"
	@echo "  help         - Show this help message"