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

🚧 **Currently in Phase 1**: Foundation & Infrastructure Setup

See [planning/phase1.md](../planning/phase1.md) for detailed development roadmap.

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

*Note: Setup instructions will be added as development progresses*

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