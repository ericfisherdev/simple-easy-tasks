# Simple Easy Tasks - Makefile

.PHONY: help build test test-verbose test-coverage clean lint run dev docker-build docker-run

# Default target
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build targets
build: ## Build the server binary
	@echo "Building server..."
	go build -o server ./cmd/server

build-linux: ## Build for Linux
	@echo "Building for Linux..."
	GOOS=linux GOARCH=amd64 go build -o server-linux ./cmd/server

# Test targets
test: ## Run all tests
	@echo "Running tests..."
	go test ./...

test-verbose: ## Run tests with verbose output
	@echo "Running tests (verbose)..."
	go test -v ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-race: ## Run tests with race detector
	@echo "Running tests with race detector..."
	go test -race ./...

test-short: ## Run only short tests
	@echo "Running short tests..."
	go test -short ./...

benchmark: ## Run benchmarks
	@echo "Running benchmarks..."
	go test -bench=. ./...

# Code quality targets
lint: ## Run linter
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run; \
	fi

fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...

fmt-check: ## Check if code is formatted
	@echo "Checking code formatting..."
	@if [ -n "$$(go fmt ./...)" ]; then \
		echo "Code is not formatted properly"; \
		exit 1; \
	fi

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

staticcheck: ## Run staticcheck
	@echo "Running staticcheck..."
	@if command -v staticcheck >/dev/null 2>&1; then \
		staticcheck ./...; \
	else \
		echo "staticcheck not found. Installing..."; \
		go install honnef.co/go/tools/cmd/staticcheck@latest; \
		staticcheck ./...; \
	fi

# Development targets
run: build ## Build and run the server
	@echo "Starting server..."
	./server

dev: ## Run the server in development mode
	@echo "Starting server in development mode..."
	@export GIN_MODE=debug; \
	go run ./cmd/server

pocketbase: ## Run PocketBase with migrations
	@echo "Starting PocketBase with migrations..."
	go run ./cmd/pocketbase serve --http="0.0.0.0:8090" --dir="./pb_data"

pocketbase-migrate: ## Run PocketBase migrations only
	@echo "Running PocketBase migrations..."
	go run ./cmd/pocketbase migrate up

watch: ## Watch for changes and restart server
	@echo "Watching for changes..."
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "air not found. Install with: go install github.com/cosmtrek/air@latest"; \
		echo "Falling back to basic run..."; \
		$(MAKE) dev; \
	fi

# Docker targets
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t simple-easy-tasks .

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	docker run -p 8080:8080 simple-easy-tasks

# Database targets
migrate-up: ## Run database migrations up
	@echo "Running migrations up..."
	@if [ -f ./scripts/migrate.sh ]; then \
		./scripts/migrate.sh up; \
	else \
		echo "Migration script not found"; \
	fi

migrate-down: ## Run database migrations down
	@echo "Running migrations down..."
	@if [ -f ./scripts/migrate.sh ]; then \
		./scripts/migrate.sh down; \
	else \
		echo "Migration script not found"; \
	fi

# Dependency targets
mod-tidy: ## Tidy Go modules
	@echo "Tidying Go modules..."
	go mod tidy

mod-vendor: ## Vendor Go modules
	@echo "Vendoring Go modules..."
	go mod vendor

mod-update: ## Update Go modules
	@echo "Updating Go modules..."
	go get -u ./...
	go mod tidy

# Security targets
security-check: ## Run security checks
	@echo "Running security checks..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not found. Installing..."; \
		go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; \
		gosec ./...; \
	fi

# Clean targets
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -f server server-linux coverage.out coverage.html

clean-all: clean ## Clean all generated files
	@echo "Cleaning all generated files..."
	go clean -cache -testcache -modcache

# CI targets
ci: fmt-check vet staticcheck test test-race ## Run CI pipeline
	@echo "CI pipeline completed successfully"

ci-coverage: fmt-check vet staticcheck test-coverage ## Run CI with coverage
	@echo "CI pipeline with coverage completed"

# Setup targets
setup: ## Setup development environment
	@echo "Setting up development environment..."
	go mod download
	@if [ -f ./scripts/setup-hooks.sh ]; then \
		./scripts/setup-hooks.sh; \
	fi
	@echo "Development environment setup complete"

# Production targets
build-prod: ## Build production binary
	@echo "Building production binary..."
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o server ./cmd/server

# Version info
version: ## Show version information
	@echo "Go version: $$(go version)"
	@echo "Git commit: $$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"
	@echo "Build time: $$(date -u +%Y-%m-%dT%H:%M:%SZ)"

# Health check
health: ## Check if server is running
	@echo "Checking server health..."
	@curl -s http://localhost:8080/health || echo "Server is not running"