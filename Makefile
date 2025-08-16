.PHONY: build build-migrate run test clean dev deps migrate-up migrate-down migrate-version docker-build docker-run check-ports create-test-user

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

# Run tests
test:
	go test -v ./...

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

# Create test user
create-test-user:
	@./scripts/create-test-user.sh $(ARGS)

# Help
help:
	@echo "Available commands:"
	@echo "  build         - Build the application"
	@echo "  run          - Run the application"
	@echo "  dev          - Run with hot reload"
	@echo "  test         - Run tests"
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
	@echo "  create-test-user - Create a test user (use ARGS for options)"
	@echo "  help         - Show this help message"