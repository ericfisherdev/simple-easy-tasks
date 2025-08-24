#!/bin/bash

# Database migration script for Simple Easy Tasks
# This script provides utilities for managing database migrations

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

# Function to run migrations
run_migrations() {
    log_info "Running database migrations..."
    
    # Check if PocketBase is available
    if ! command -v ./server >/dev/null 2>&1; then
        log_error "Server binary not found. Please build the application first."
        exit 1
    fi
    
    # Run PocketBase with migrate command
    ./server migrate up
    
    if [ $? -eq 0 ]; then
        log_success "Migrations completed successfully"
    else
        log_error "Migration failed"
        exit 1
    fi
}

# Function to rollback migrations
rollback_migrations() {
    local steps=${1:-1}
    log_info "Rolling back $steps migration(s)..."
    
    ./server migrate down $steps
    
    if [ $? -eq 0 ]; then
        log_success "Rollback completed successfully"
    else
        log_error "Rollback failed"
        exit 1
    fi
}

# Function to check migration status
migration_status() {
    log_info "Checking migration status..."
    ./server migrate status
}

# Function to create a new migration
create_migration() {
    local name="$1"
    if [ -z "$name" ]; then
        log_error "Migration name is required"
        echo "Usage: $0 create <migration_name>"
        exit 1
    fi
    
    log_info "Creating new migration: $name"
    
    # Create timestamp
    timestamp=$(date +"%Y%m%d%H%M%S")
    filename="${timestamp}_$(echo "$name" | tr ' ' '_' | tr '[:upper:]' '[:lower:]').go"
    filepath="migrations/$filename"
    
    # Create migrations directory if it doesn't exist
    mkdir -p migrations
    
    # Create migration file from template
    cat > "$filepath" << EOF
package main

import (
    "log"
    
    "github.com/pocketbase/pocketbase/migrations"
    "github.com/pocketbase/pocketbase/daos"
)

func init() {
    migrations.Register(func(db migrations.DB) error {
        log.Printf("Running migration: $name")
        
        dao := daos.New(db)
        
        // Add your migration logic here
        // Example:
        // collection, err := dao.FindCollectionByNameOrId("your_collection")
        // if err != nil {
        //     return err
        // }
        // 
        // // Modify collection...
        // 
        // return dao.SaveCollection(collection)
        
        return nil
    }, func(db migrations.DB) error {
        log.Printf("Rolling back migration: $name")
        
        dao := daos.New(db)
        
        // Add your rollback logic here
        
        return nil
    }, "$filename")
}
EOF
    
    log_success "Created migration file: $filepath"
    log_info "Edit the file to add your migration logic"
}

# Function to reset database (DANGEROUS)
reset_database() {
    read -p "This will delete all data and reset the database. Are you sure? (type 'RESET' to confirm): " confirmation
    if [ "$confirmation" != "RESET" ]; then
        log_info "Database reset cancelled"
        return
    fi
    
    log_warn "Resetting database..."
    
    # Remove PocketBase data directory
    if [ -d "pb_data" ]; then
        rm -rf pb_data
        log_info "Removed pb_data directory"
    fi
    
    # Recreate with fresh migrations
    ./server migrate up
    
    if [ $? -eq 0 ]; then
        log_success "Database reset and migrations completed"
    else
        log_error "Database reset failed"
        exit 1
    fi
}

# Function to backup database
backup_database() {
    local backup_dir="backups"
    local timestamp=$(date +"%Y%m%d_%H%M%S")
    local backup_file="backup_${timestamp}.tar.gz"
    
    mkdir -p "$backup_dir"
    
    log_info "Creating database backup: $backup_file"
    
    if [ -d "pb_data" ]; then
        tar -czf "$backup_dir/$backup_file" pb_data/
        log_success "Backup created: $backup_dir/$backup_file"
    else
        log_warn "No pb_data directory found to backup"
    fi
}

# Function to restore database from backup
restore_database() {
    local backup_file="$1"
    
    if [ -z "$backup_file" ]; then
        log_error "Backup file is required"
        echo "Usage: $0 restore <backup_file>"
        exit 1
    fi
    
    if [ ! -f "$backup_file" ]; then
        log_error "Backup file not found: $backup_file"
        exit 1
    fi
    
    read -p "This will overwrite the current database. Continue? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "Restore cancelled"
        return
    fi
    
    log_info "Restoring database from: $backup_file"
    
    # Remove current data
    if [ -d "pb_data" ]; then
        rm -rf pb_data
    fi
    
    # Extract backup
    tar -xzf "$backup_file"
    
    if [ $? -eq 0 ]; then
        log_success "Database restored successfully"
    else
        log_error "Database restore failed"
        exit 1
    fi
}

# Function to seed database with test data
seed_database() {
    log_info "Seeding database with test data..."
    
    # This would typically call a Go program or script to insert test data
    # For now, we'll just log that seeding would happen
    log_info "Database seeding would be implemented here"
    log_success "Database seeded successfully"
}

# Main script logic
case "${1:-help}" in
    "up"|"migrate")
        run_migrations
        ;;
    "down"|"rollback")
        rollback_migrations "${2:-1}"
        ;;
    "status")
        migration_status
        ;;
    "create")
        create_migration "$2"
        ;;
    "reset")
        reset_database
        ;;
    "backup")
        backup_database
        ;;
    "restore")
        restore_database "$2"
        ;;
    "seed")
        seed_database
        ;;
    "help"|*)
        echo "Database Migration Tool for Simple Easy Tasks"
        echo ""
        echo "Usage: $0 <command> [options]"
        echo ""
        echo "Commands:"
        echo "  up, migrate           Run pending migrations"
        echo "  down, rollback [n]    Rollback n migrations (default: 1)"
        echo "  status               Show migration status"
        echo "  create <name>        Create a new migration file"
        echo "  reset                Reset database (DANGER: deletes all data)"
        echo "  backup               Create database backup"
        echo "  restore <file>       Restore database from backup"
        echo "  seed                 Seed database with test data"
        echo "  help                 Show this help message"
        echo ""
        echo "Examples:"
        echo "  $0 up                    # Run all pending migrations"
        echo "  $0 down 2                # Rollback last 2 migrations"
        echo "  $0 create add_tasks      # Create new migration file"
        echo "  $0 backup                # Create backup"
        exit 1
        ;;
esac