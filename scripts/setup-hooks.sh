#!/bin/bash

# Setup Git hooks for the Simple Easy Tasks project

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "Setting up Git hooks for Simple Easy Tasks..."

# Ensure we're in the project root
cd "$PROJECT_ROOT"

# Check if .git directory exists
if [ ! -d ".git" ]; then
    echo "Error: Not a Git repository. Please run 'git init' first."
    exit 1
fi

# Create hooks directory if it doesn't exist
mkdir -p .git/hooks

# Copy pre-commit hook if it exists
if [ -f ".git/hooks/pre-commit" ]; then
    echo "✅ Pre-commit hook already exists"
else
    echo "❌ Pre-commit hook not found. Please ensure it's been created."
    exit 1
fi

# Make hooks executable
chmod +x .git/hooks/pre-commit

# Install golangci-lint if not present
if ! command -v golangci-lint >/dev/null 2>&1 && [ ! -f "$HOME/go/bin/golangci-lint" ]; then
    echo "Installing golangci-lint..."
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
fi

echo ""
echo "🎉 Git hooks setup complete!"
echo ""
echo "The following hooks are now active:"
echo "  • pre-commit: Runs go fmt, go vet, go mod tidy, and golangci-lint"
echo ""
echo "To test the setup, try making a commit:"
echo "  git add ."
echo "  git commit -m 'Test commit'"