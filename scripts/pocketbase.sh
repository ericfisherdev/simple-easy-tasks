#!/bin/bash

# Script to run PocketBase with migrations

set -e

# Change to project root
cd "$(dirname "$0")/.."

echo "Starting PocketBase with migrations..."

# Run PocketBase
go run cmd/pocketbase/main.go serve \
    --http="0.0.0.0:8090" \
    --dir="./pb_data" \
    "$@"