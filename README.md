# Simple Easy Tasks

A modern, lightweight task management application built with Go, PocketBase, and HTMX.

## Overview

Simple Easy Tasks is designed to provide an intuitive, fast, and reliable task management experience. Built following Go best practices and SOLID principles, it offers a clean architecture that's easy to maintain and extend.

## Architecture

- **Backend**: Go with clean architecture and SOLID principles
- **Database**: PocketBase for data persistence and real-time features
- **Frontend**: HTMX with Alpine.js for interactive UI
- **Deployment**: Docker containerization

## Development Status

✅ **Phase 1 Complete**: Foundation & Infrastructure Setup (100%)

### Completed Milestones
- **Week 1**: ✅ Project structure, Go best practices, error handling, dependency injection
- **Week 2**: ✅ Docker infrastructure, PocketBase integration, health monitoring, database migrations
- **Week 3**: ✅ Authentication system, domain models, repository patterns, JWT implementation
- **Week 4**: ✅ RESTful API endpoints, comprehensive testing framework

### Current Capabilities
- **Full authentication system** with JWT tokens and refresh mechanism
- **User management** with profile and avatar support
- **Project management** with CRUD operations
- **Task management** with advanced filtering and status tracking
- **Command Line Interface (CLI)** for automation and scripting
- **Role-based access control (RBAC)**
- **Multiple output formats** (JSON, YAML, CSV, rich tables)
- **Comprehensive test coverage** with unit and integration tests
- **Docker containerization** with health checks
- **PocketBase v0.29.3 integration** for data persistence

### API Endpoints Available
- **Authentication**: Login, logout, register, password reset, token refresh
- **Users**: Profile management, avatar upload
- **Projects**: Full CRUD operations with member management
- **Tasks**: Complete lifecycle management with filtering

### CLI Tool Features
- **Interactive authentication** with secure profile management
- **Rich output formatting** with color-coded status indicators
- **Multiple environment support** with profile switching
- **Comprehensive task operations** with advanced filtering
- **Scriptable automation** with JSON/YAML/CSV export formats

See [planning/phase1.md](../planning/phase1.md) for detailed development accomplishments.

## Command Line Interface (CLI)

Simple Easy Tasks includes a powerful CLI tool (`set-cli`) for managing tasks, projects, and authentication from the command line.

### CLI Installation

#### Option 1: Build from Source
```bash
# Build the CLI tool
make build

# The binary will be available as ./set-cli
./set-cli --help
```

#### Option 2: Install to System Path
```bash
# Build and install to /usr/local/bin
sudo make install

# Now available system-wide
set-cli --help
```

#### Option 3: Development Build
```bash
# Quick development build
go build -o set-cli cmd/set-cli/main.go
```

### CLI Usage

#### Authentication
```bash
# Login (interactive prompts for email/password)
set-cli auth login

# Login with specific server
set-cli auth login --server https://api.yourdomain.com

# Check authentication status
set-cli auth status

# Logout
set-cli auth logout
```

#### Profile Management
```bash
# List all profiles
set-cli auth profile list

# Create a new profile with API token
set-cli auth profile create staging \
  --server https://staging.api.com \
  --token your-api-token

# Switch between profiles
set-cli auth profile select staging

# Show current profile details
set-cli auth profile show
```

#### Project Management
```bash
# List all projects
set-cli project list

# List projects with JSON output
set-cli project list --format json

# Show project details with tasks
set-cli project show PROJECT_ID --include-tasks

# Create a new project
set-cli project create "My New Project" \
  --description "Project description"
```

#### Task Management
```bash
# List all tasks for a project
set-cli task list --project-id PROJECT_ID

# List tasks with filtering
set-cli task list \
  --project-id PROJECT_ID \
  --status todo,developing \
  --priority high \
  --limit 10

# Create a new task
set-cli task create \
  --project-id PROJECT_ID \
  --title "New Task" \
  --description "Task description" \
  --priority high

# Update task status
set-cli task update PROJECT_ID TASK_ID \
  --status developing

# Delete a task
set-cli task delete PROJECT_ID TASK_ID
```

#### Output Formats
The CLI supports multiple output formats:
```bash
# Table format (default)
set-cli project list

# JSON output
set-cli project list --format json

# YAML output
set-cli project list --format yaml

# CSV output (projects and tasks)
set-cli project list --format csv
```

### CLI Configuration

The CLI stores configuration in `~/.set-cli.yaml`:
```yaml
default_profile: default
profiles:
  default:
    name: default
    server_url: http://localhost:8090
    token: your-jwt-token
  staging:
    name: staging 
    server_url: https://staging.api.com
    token: staging-token
```

### Environment Variables

Override configuration with environment variables:
```bash
export SET_CLI_PROFILE=staging
export SET_CLI_SERVER=https://api.example.com
export SET_CLI_TOKEN=your-token

set-cli task list --project-id PROJECT_ID
```

## Project Structure

```
simple-easy-tasks/
├── cmd/
│   ├── server/          # Web API server entrypoint
│   ├── pocketbase/      # PocketBase server entrypoint
│   └── set-cli/         # CLI tool entrypoint
├── internal/            # Private application code
│   ├── api/            # HTTP handlers and routes
│   ├── cli/            # CLI commands and client
│   ├── config/         # Configuration management
│   ├── domain/         # Domain models and business logic
│   ├── services/       # Business services
│   └── repository/     # Data access layer
├── pkg/                # Public library code
├── web/                # Frontend assets and templates
├── api/                # API specifications
├── configs/            # Configuration files
├── scripts/            # Build and deployment scripts
└── docs/               # Documentation
```

## Quick Start

### Prerequisites
- Go 1.21 or higher
- Docker and Docker Compose
- Git

### Installation

1. Clone the repository:
```bash
git clone https://github.com/ericfisherdev/simple-easy-tasks.git
cd simple-easy-tasks
```

2. Install dependencies:
```bash
go mod download
```

3. Set up environment variables:
```bash
cp .env.example .env
# Edit .env with your configuration
```

4. Build all components:
```bash
# Build web server, PocketBase, and CLI tool
make build
```

### Running the Application

#### Option 1: Docker (Recommended)
```bash
docker-compose up -d
```

#### Option 2: Local Development
```bash
# Run the Gin API server
go run cmd/server/main.go

# In a separate terminal, run PocketBase (for database and migrations)
./scripts/pocketbase.sh
# Or directly: go run cmd/pocketbase/main.go serve
```

The application will be available at:
- **API Server**: `http://localhost:8080`
- **PocketBase Admin**: `http://localhost:8090/_/`

### Using the CLI Tool

Once the server is running, you can use the CLI tool:

```bash
# Authenticate with the API
./set-cli auth login --server http://localhost:8090

# List projects
./set-cli project list

# Create a new project
./set-cli project create "My First Project"

# List tasks (replace PROJECT_ID with actual ID)
./set-cli task list --project-id PROJECT_ID
```

### Running Tests
```bash
# Run all tests
go test ./... -v -cover

# Run integration tests
make test-integration

# Run linting
make lint
```

## Development Requirements

- Go 1.21+
- Docker & Docker Compose
- Git

## Contributing

This project follows Go best practices:
- All code must be formatted with `gofmt`
- Static analysis with `staticcheck`
- Comprehensive test coverage (80%+ target)
- Clean architecture with dependency injection

## License

See [LICENSE](LICENSE) for details.