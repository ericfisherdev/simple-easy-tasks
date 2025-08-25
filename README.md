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
- Full authentication system with JWT tokens and refresh mechanism
- User management with profile and avatar support
- Project management with CRUD operations
- Role-based access control (RBAC)
- Comprehensive test coverage with unit and integration tests
- Docker containerization with health checks
- PocketBase v0.29.3 integration for data persistence

### API Endpoints Available
- **Authentication**: Login, logout, register, password reset, token refresh
- **Users**: Profile management, avatar upload
- **Projects**: Full CRUD operations with member management

See [planning/phase1.md](../planning/phase1.md) for detailed development accomplishments.

## Project Structure

```
simple-easy-tasks/
├── cmd/server/          # Application entrypoint
├── internal/            # Private application code
│   ├── api/            # HTTP handlers and routes
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

4. Run with Docker:
```bash
docker-compose up -d
```

5. Or run locally:
```bash
# Run the Gin API server
go run cmd/server/main.go

# In a separate terminal, run PocketBase (for database and migrations)
./scripts/pocketbase.sh
# Or directly: go run cmd/pocketbase/main.go serve
```

The application will be available at:
- API Server: `http://localhost:8080`
- PocketBase Admin: `http://localhost:8090/_/`

### Running Tests
```bash
go test ./... -v -cover
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