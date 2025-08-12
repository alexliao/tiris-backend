.PHONY: build build-migrate run test clean dev deps migrate-up migrate-down migrate-version docker-build docker-run

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
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

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
	@echo "  help         - Show this help message"