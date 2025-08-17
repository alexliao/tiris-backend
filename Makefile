.PHONY: build build-migrate run test test-unit test-integration test-integration-docker test-coverage clean dev deps migrate-up migrate-down migrate-version docker-build docker-run check-ports create-test-user setup-test-db clean-test-db setup-test-db-docker stop-test-db-docker clean-test-db-docker

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
test: test-unit test-integration-docker

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
	go test -v -coverprofile=coverage/coverage.out ./...
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
setup-test-db-docker:
	@echo "Starting test database with Docker..."
	@docker compose -f docker-compose.test.yml up -d postgres-test
	@echo "Waiting for database to be ready..."
	@echo "Waiting for PostgreSQL to be ready (max 30 seconds)..."
	@for i in `seq 1 30`; do \
		if docker compose -f docker-compose.test.yml exec postgres-test pg_isready -U postgres >/dev/null 2>&1; then \
			break; \
		fi; \
		echo "Waiting... ($$i/30)"; \
		sleep 1; \
	done
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
	@echo "  create-test-user - Create a test user (use ARGS for options)"
	@echo "  help         - Show this help message"