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

# Integration test targets
test-integration: ## Run integration tests
	@echo "Running integration tests..."
	@mkdir -p /tmp/test-dbs
	@export TEST_DB_PATH="/tmp/test-dbs"; \
	go test -tags=integration -v -timeout=20m ./internal/testutil/integration ./test/integration
	@rm -rf /tmp/test-dbs/*.db*

test-integration-coverage: ## Run integration tests with coverage
	@echo "Running integration tests with coverage..."
	@mkdir -p coverage /tmp/test-dbs
	@export TEST_DB_PATH="/tmp/test-dbs"; \
	go test -tags=integration -v -timeout=20m \
		-coverprofile=coverage/integration.out \
		-covermode=atomic \
		-coverpkg=./internal/... \
		./internal/testutil/integration ./test/integration
	@go tool cover -html=coverage/integration.out -o coverage/integration.html
	@echo "Integration coverage report generated: coverage/integration.html"
	@rm -rf /tmp/test-dbs/*.db*

test-integration-verbose: ## Run integration tests with verbose output
	@echo "Running integration tests (verbose)..."
	@mkdir -p /tmp/test-dbs
	@export TEST_DB_PATH="/tmp/test-dbs" TEST_VERBOSE=1; \
	go test -tags=integration -v -timeout=20m ./internal/testutil/integration ./test/integration
	@rm -rf /tmp/test-dbs/*.db*

test-integration-race: ## Run integration tests with race detector
	@echo "Running integration tests with race detector..."
	@mkdir -p /tmp/test-dbs
	@export TEST_DB_PATH="/tmp/test-dbs"; \
	go test -tags=integration -race -timeout=20m ./internal/testutil/integration ./test/integration
	@rm -rf /tmp/test-dbs/*.db*

test-integration-parallel: ## Run integration tests in parallel
	@echo "Running integration tests in parallel..."
	@mkdir -p /tmp/test-dbs
	@export TEST_DB_PATH="/tmp/test-dbs"; \
	go test -tags=integration -v -parallel=4 -timeout=30m ./internal/testutil/integration ./test/integration
	@rm -rf /tmp/test-dbs/*.db*

benchmark-integration: ## Run integration benchmarks
	@echo "Running integration benchmarks..."
	@mkdir -p /tmp/perf-test-dbs
	@export TEST_DB_PATH="/tmp/perf-test-dbs"; \
	go test -tags=integration -bench=. -benchmem -timeout=30m -run=^$$ ./test/integration
	@rm -rf /tmp/perf-test-dbs/*.db*

# Commit linting targets
commit-lint: ## Lint the last commit message
	@echo "Linting last commit message..."
	@if command -v npm >/dev/null 2>&1; then \
		npm run commitlint; \
	else \
		echo "npm not found. Please install Node.js and npm"; \
		exit 1; \
	fi

commit-lint-range: ## Lint commit messages in range (usage: make commit-lint-range FROM=commit1 TO=commit2)
	@echo "Linting commit messages from $(FROM) to $(TO)..."
	@if command -v npm >/dev/null 2>&1; then \
		npx commitlint --from $(FROM) --to $(TO) --verbose; \
	else \
		echo "npm not found. Please install Node.js and npm"; \
		exit 1; \
	fi

commit-lint-branch: ## Lint all commits in current branch compared to develop
	@echo "Linting all commits in current branch compared to develop..."
	@if command -v npm >/dev/null 2>&1; then \
		npm run lint-commits; \
	else \
		echo "npm not found. Please install Node.js and npm"; \
		exit 1; \
	fi

commit-msg-help: ## Show commit message format help
	@echo "📋 Conventional Commits Format:"
	@echo ""
	@echo "   <type>[optional scope]: <description>"
	@echo ""
	@echo "🏷️  Types:"
	@echo "   feat     - ✨ A new feature"
	@echo "   fix      - 🐛 A bug fix"
	@echo "   docs     - 📚 Documentation only changes"
	@echo "   style    - 💄 Code style changes (formatting, etc)"
	@echo "   refactor - ♻️  Code change that neither fixes a bug nor adds a feature"
	@echo "   perf     - ⚡ Performance improvements"
	@echo "   test     - 🧪 Adding missing tests"
	@echo "   build    - 📦 Changes affecting build system or dependencies"
	@echo "   ci       - 👷 Changes to CI configuration"
	@echo "   chore    - 🔧 Other changes that don't modify src or test files"
	@echo "   revert   - ⏪ Reverts a previous commit"
	@echo ""
	@echo "🔍 Common scopes:"
	@echo "   api, handlers, middleware, auth, validation"
	@echo "   db, repository, migrations, collections"
	@echo "   domain, services, models"
	@echo "   tests, integration, unit, e2e"
	@echo "   ci, docker, deployment, scripts"
	@echo "   tasks, projects, users, comments"
	@echo ""
	@echo "✅ Examples:"
	@echo "   feat(api): add user authentication endpoint"
	@echo "   fix(db): resolve connection pool exhaustion"
	@echo "   docs: update API documentation"
	@echo "   test(integration): add user repository tests"
	@echo ""
	@echo "📖 Learn more: https://conventionalcommits.org/"

husky-install: ## Install git hooks with husky
	@echo "Installing git hooks with husky..."
	@if command -v npm >/dev/null 2>&1; then \
		npm run prepare; \
	else \
		echo "npm not found. Please install Node.js and npm"; \
		exit 1; \
	fi

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
	rm -f server server-linux coverage.out coverage.html integration.test integration*.test
	rm -rf coverage/
	rm -rf /tmp/test-dbs/*.db* 2>/dev/null || true
	rm -rf /tmp/perf-test-dbs/*.db* 2>/dev/null || true

clean-all: clean ## Clean all generated files
	@echo "Cleaning all generated files..."
	go clean -cache -testcache -modcache

# CI targets
ci: fmt-check vet staticcheck test test-race ci-integration-check ## Run CI pipeline
	@echo "CI pipeline completed successfully"

ci-coverage: fmt-check vet staticcheck test-coverage ## Run CI with coverage
	@echo "CI pipeline with coverage completed"

ci-integration-check: ## Check that integration tests compile
	@echo "Verifying integration tests compile..."
	@go test -tags=integration -c ./internal/testutil/integration
	@go test -tags=integration -c ./test/integration
	@rm -f integration.test integration*.test
	@echo "Integration tests compilation check passed"

ci-commit-lint: ## Lint commit messages (CI)
	@echo "Linting commit messages for CI..."
	@if command -v npm >/dev/null 2>&1; then \
		npm run commitlint; \
	else \
		echo "Skipping commit lint - npm not available in CI environment"; \
	fi

ci-full: fmt-check vet staticcheck test test-race test-integration ci-commit-lint ## Run full CI with integration tests
	@echo "Full CI pipeline completed successfully"

ci-pre-commit: fmt-check vet ci-integration-check ci-commit-lint ## Run pre-commit checks
	@echo "Pre-commit checks completed successfully"

# Setup targets
setup: ## Setup development environment
	@echo "Setting up development environment..."
	go mod download
	@if command -v npm >/dev/null 2>&1; then \
		echo "Installing npm dependencies..."; \
		npm ci; \
		echo "Setting up git hooks..."; \
		npm run prepare; \
	else \
		echo "⚠️  npm not found. Please install Node.js and npm for commit linting."; \
	fi
	@if [ -f ./scripts/setup-hooks.sh ]; then \
		./scripts/setup-hooks.sh; \
	fi
	@echo "Development environment setup complete"
	@echo ""
	@echo "📋 Next steps:"
	@echo "  • Run 'make commit-msg-help' to see commit message format"
	@echo "  • Use conventional commits: <type>[scope]: <description>"
	@echo "  • Run 'make test-integration' to run integration tests"

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