#!/bin/bash

# Test script that mirrors the GitHub Actions integration test workflow exactly

set -e

echo "🔧 Setting up test environment like CI..."

# Environment variables from CI
export TEST_DB_PATH="/tmp/test-dbs"
export TEST_PARALLEL=1
export TEST_VERBOSE=1

# Create test databases directory like CI
mkdir -p /tmp/test-dbs

echo "📋 Running integration tests for both packages..."

# Test package 1: internal/testutil/integration
echo "🧪 Testing ./internal/testutil/integration"
go test -tags=integration \
  -v \
  -race \
  -timeout=20m \
  -parallel=2 \
  -coverprofile=coverage/integration-0.out \
  -covermode=atomic \
  -coverpkg=./internal/... \
  ./internal/testutil/integration

# Test package 2: test/integration  
echo "🧪 Testing ./test/integration"
go test -tags=integration \
  -v \
  -race \
  -timeout=20m \
  -parallel=2 \
  -coverprofile=coverage/integration-1.out \
  -covermode=atomic \
  -coverpkg=./internal/... \
  ./test/integration

echo "✅ All tests completed successfully!"

# Clean up like CI
echo "🧹 Cleaning up test databases..."
rm -rf /tmp/test-dbs/*.db
rm -rf /tmp/test-dbs/*.db-*