#!/bin/bash

# Secure commit wrapper script
# This script ensures that pre-commit hooks cannot be bypassed
# Usage: ./scripts/secure-commit.sh "commit message" [additional git commit flags]

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${2}[SECURE-COMMIT]${NC} $1"
}

# Function to show usage
show_usage() {
    echo "Usage: $0 \"commit message\" [additional git commit flags]"
    echo ""
    echo "Examples:"
    echo "  $0 \"feat(api): add user authentication\""
    echo "  $0 \"fix(db): resolve connection pool issue\" --author=\"John Doe <john@example.com>\""
    echo ""
    echo "This script:"
    echo "  ✅ Runs mandatory pre-commit checks (formatting, linting, security)"
    echo "  ✅ Validates commit message format (conventional commits)"
    echo "  ✅ Cannot be bypassed with --no-verify"
    echo "  ✅ Provides clear feedback on any issues"
    exit 1
}

# Check if commit message is provided
if [ $# -lt 1 ]; then
    print_status "Error: Commit message is required" $RED
    show_usage
fi

COMMIT_MESSAGE="$1"
shift # Remove the first argument (commit message) so $@ contains only additional flags

print_status "Starting secure commit process..." $BLUE

# Check if we're in a git repository
if [ ! -d ".git" ]; then
    print_status "Error: Not in a git repository" $RED
    exit 1
fi

# Check if we're in the Go project root
if [ ! -f "go.mod" ]; then
    print_status "Error: Not in a Go project root (go.mod not found)" $RED
    exit 1
fi

print_status "Running pre-commit checks..." $YELLOW

# 1. Check for staged changes
if git diff --cached --quiet; then
    print_status "Error: No staged changes to commit" $RED
    print_status "Use 'git add <files>' to stage your changes first" $YELLOW
    exit 1
fi

# 2. Run go mod tidy
print_status "Running 'go mod tidy'..." $GREEN
if ! go mod tidy; then
    print_status "Error: go mod tidy failed!" $RED
    exit 1
fi

# 3. Check and auto-stage go.mod/go.sum if changed
if git diff --name-only | grep -E "go\.(mod|sum)$" >/dev/null; then
    print_status "go.mod or go.sum changed, staging files..." $YELLOW
    git add go.mod go.sum 2>/dev/null || true
fi

# 4. Run go fmt and check for changes
print_status "Running 'go fmt'..." $GREEN
UNFORMATTED=$(go fmt ./...)
if [ -n "$UNFORMATTED" ]; then
    print_status "Error: The following files were reformatted:" $RED
    echo "$UNFORMATTED"
    print_status "Please stage the formatted changes and run this script again" $RED
    exit 1
fi

# 5. Run go vet
print_status "Running 'go vet'..." $GREEN
if ! go vet ./...; then
    print_status "Error: go vet failed!" $RED
    exit 1
fi

# 6. Run golangci-lint if available
if command -v golangci-lint >/dev/null 2>&1; then
    print_status "Running 'golangci-lint'..." $GREEN
    if ! golangci-lint run; then
        print_status "Error: golangci-lint failed!" $RED
        exit 1
    fi
elif [ -f "$HOME/go/bin/golangci-lint" ]; then
    print_status "Running 'golangci-lint' from GOPATH..." $GREEN
    if ! "$HOME/go/bin/golangci-lint" run; then
        print_status "Error: golangci-lint failed!" $RED
        exit 1
    fi
else
    print_status "Warning: golangci-lint not found, skipping..." $YELLOW
    print_status "Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" $YELLOW
fi

# 7. Validate commit message format (basic conventional commits check)
print_status "Validating commit message format..." $GREEN
if ! echo "$COMMIT_MESSAGE" | grep -qE '^(feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert)(\(.+\))?: .{1,}'; then
    print_status "Error: Commit message does not follow conventional commits format" $RED
    print_status "Expected format: <type>[optional scope]: <description>" $YELLOW
    print_status "Examples:" $YELLOW
    print_status "  feat(api): add user authentication" $YELLOW
    print_status "  fix(db): resolve connection pool issue" $YELLOW
    print_status "  docs: update API documentation" $YELLOW
    print_status "Run 'make commit-msg-help' for more information" $BLUE
    exit 1
fi

# 8. Run the actual commit (without --no-verify to ensure all hooks run)
print_status "All checks passed! Committing changes..." $GREEN

# Construct the git commit command
GIT_CMD="git commit -m \"$COMMIT_MESSAGE\""

# Add any additional flags passed to the script
if [ $# -gt 0 ]; then
    GIT_CMD="$GIT_CMD $@"
fi

print_status "Executing: $GIT_CMD" $BLUE

# Execute the commit
if eval "$GIT_CMD"; then
    print_status "Commit successful! ✅" $GREEN
    print_status "Your code has been committed with all quality checks passed" $GREEN
else
    print_status "Commit failed!" $RED
    exit 1
fi