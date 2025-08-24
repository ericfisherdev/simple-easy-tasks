#!/bin/bash

# Environment setup script for Simple Easy Tasks
# This script helps set up environment files and validates configuration

set -e

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to generate a random JWT secret
generate_jwt_secret() {
    if command -v openssl >/dev/null 2>&1; then
        openssl rand -base64 32
    elif command -v python3 >/dev/null 2>&1; then
        python3 -c "import secrets, base64; print(base64.b64encode(secrets.token_bytes(32)).decode())"
    else
        # Fallback: generate from /dev/urandom
        head -c 32 /dev/urandom | base64
    fi
}

# Function to setup environment files
setup_env_files() {
    log_info "Setting up environment files..."

    # Copy .env.example to .env if it doesn't exist
    if [ ! -f ".env" ]; then
        cp ".env.example" ".env"
        log_success "Created .env from .env.example"
        
        # Generate a new JWT secret
        JWT_SECRET=$(generate_jwt_secret)
        if [ -n "$JWT_SECRET" ]; then
            sed -i "s|your-super-secret-jwt-key-with-at-least-32-characters|$JWT_SECRET|g" ".env"
            log_success "Generated new JWT secret"
        else
            log_warn "Could not generate JWT secret automatically. Please update manually."
        fi
    else
        log_info ".env file already exists, skipping creation"
    fi

    # Create .env.local if it doesn't exist
    if [ ! -f ".env.local" ]; then
        cat > ".env.local" << EOF
# Local environment overrides
# This file is ignored by git and safe for local secrets

# Uncomment and modify as needed:
# JWT_SECRET=your-local-jwt-secret
# PB_ADMIN_PASSWORD=your-local-password
# REDIS_PASSWORD=your-local-redis-password
EOF
        log_success "Created .env.local template"
    fi
}

# Function to validate environment configuration
validate_env() {
    log_info "Validating environment configuration..."
    
    # Source the environment file
    set -a
    source .env
    set +a
    
    # Check required variables
    REQUIRED_VARS=(
        "SERVER_PORT"
        "JWT_SECRET"
        "ENVIRONMENT"
        "DATABASE_URL"
    )
    
    for var in "${REQUIRED_VARS[@]}"; do
        if [ -z "${!var}" ]; then
            log_error "Required environment variable $var is not set"
            exit 1
        fi
    done
    
    # Validate JWT secret length
    if [ ${#JWT_SECRET} -lt 32 ]; then
        log_error "JWT_SECRET must be at least 32 characters long"
        exit 1
    fi
    
    # Validate environment value
    if [[ ! "$ENVIRONMENT" =~ ^(development|staging|production)$ ]]; then
        log_error "ENVIRONMENT must be one of: development, staging, production"
        exit 1
    fi
    
    # Validate port number
    if ! [[ "$SERVER_PORT" =~ ^[0-9]+$ ]] || [ "$SERVER_PORT" -lt 1 ] || [ "$SERVER_PORT" -gt 65535 ]; then
        log_error "SERVER_PORT must be a valid port number (1-65535)"
        exit 1
    fi
    
    log_success "Environment configuration is valid"
}

# Function to show environment info
show_env_info() {
    log_info "Current environment configuration:"
    echo "  Environment: ${ENVIRONMENT:-not set}"
    echo "  Server Port: ${SERVER_PORT:-not set}"
    echo "  Database URL: ${DATABASE_URL:-not set}"
    echo "  Log Level: ${LOG_LEVEL:-not set}"
    echo "  JWT Secret Length: ${#JWT_SECRET} characters"
}

# Function to clean environment files
clean_env() {
    log_info "Cleaning environment files..."
    
    read -p "This will remove .env and .env.local files. Continue? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -f .env .env.local
        log_success "Environment files removed"
    else
        log_info "Clean cancelled"
    fi
}

# Main script logic
case "${1:-setup}" in
    "setup")
        setup_env_files
        validate_env
        show_env_info
        ;;
    "validate")
        validate_env
        show_env_info
        ;;
    "info")
        show_env_info
        ;;
    "clean")
        clean_env
        ;;
    "generate-jwt")
        JWT_SECRET=$(generate_jwt_secret)
        echo "Generated JWT Secret: $JWT_SECRET"
        ;;
    *)
        echo "Usage: $0 {setup|validate|info|clean|generate-jwt}"
        echo ""
        echo "Commands:"
        echo "  setup        Setup environment files and validate configuration (default)"
        echo "  validate     Validate current environment configuration"
        echo "  info         Show current environment information"
        echo "  clean        Remove environment files"
        echo "  generate-jwt Generate a new JWT secret"
        exit 1
        ;;
esac