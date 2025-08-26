# Integration Testing Guide

This document describes the integration testing setup for Simple Easy Tasks.

## Overview

Integration tests validate the interaction between different components of the application, particularly database operations and API endpoints. Unlike unit tests that use mocks, integration tests use real database instances to ensure data persistence and retrieval work correctly.

## Test Organization

### Test Directories

```
internal/testutil/integration/  # Core integration testing infrastructure
├── database.go                # Database setup and teardown
├── factory.go                 # Test data factory
├── suite.go                   # Test suite helpers
├── *.test.go                  # Infrastructure tests
└── test_collections.json      # PocketBase collection schema for tests

test/integration/              # Application-level integration tests
├── user_repository_*.go       # User repository tests
├── project_repository_*.go    # Project repository tests
├── concurrency_test.go        # Concurrency and transaction tests
├── performance_test.go        # Performance benchmarks
└── error_handling_test.go     # Error condition tests
```

### Test Categories

1. **Repository Tests**: CRUD operations, constraints, relationships
2. **Service Integration Tests**: Business logic with real database
3. **Concurrency Tests**: Race conditions and transaction isolation
4. **Performance Tests**: Benchmarks and load testing
5. **Foreign Key Tests**: Referential integrity validation
6. **Error Handling Tests**: Database error scenarios

## Running Integration Tests

### Local Development

```bash
# Run all integration tests
make test-integration

# Run with coverage reporting
make test-integration-coverage

# Run with race detection
make test-integration-race

# Run performance benchmarks
make benchmark-integration

# Run with verbose output
make test-integration-verbose
```

### Manual Commands

```bash
# Basic integration test run
go test -tags=integration -v ./internal/testutil/integration ./test/integration

# With coverage
go test -tags=integration -coverprofile=coverage.out -coverpkg=./internal/... ./...

# Performance benchmarks only
go test -tags=integration -bench=. -run=^$ ./test/integration
```

### Environment Variables

- `TEST_DB_PATH`: Directory for test database files (default: `/tmp/test-dbs`)
- `TEST_VERBOSE`: Enable verbose test output
- `TEST_PARALLEL`: Number of parallel test processes

## CI/CD Integration

### GitHub Actions Workflow

The integration tests run in a separate GitHub Actions workflow (`integration-tests.yml`) that:

1. **Parallel Execution**: Runs test packages in parallel for speed
2. **Test Database Isolation**: Each test gets its own database instance
3. **Coverage Reporting**: Generates and uploads coverage reports
4. **Performance Monitoring**: Tracks benchmark results over time
5. **Quality Gates**: Enforces 90% coverage threshold

### Workflow Triggers

- Push to `develop`, `release`, or `main` branches
- Pull requests targeting those branches
- Changes to integration test files or Go modules
- Manual workflow dispatch

### Coverage Requirements

- **Threshold**: 90% minimum coverage for integration tests
- **Enforcement**: CI fails if coverage is below threshold
- **Reporting**: Coverage reports are uploaded as artifacts
- **PR Comments**: Coverage results posted on pull requests

## Test Infrastructure

### Database Setup

Each test gets an isolated SQLite database:

```go
// Automatic setup
db := integration.SetupDatabaseTest(t)

// Manual setup with custom path
testDB := integration.NewTestDatabase(t, "/custom/path/test.db")
defer testDB.Cleanup()
```

### Test Data Factory

The factory creates consistent test data:

```go
factory := integration.NewTestDataFactory(db)

// Create test user
user := factory.CreateUser(integration.UserOverride{
    Email: "test@example.com",
    Username: "testuser",
})

// Create test project
project := factory.CreateProject(integration.ProjectOverride{
    Title: "Test Project",
    Owner: user.ID,
})

// Create test task
task := factory.CreateTask(integration.TaskOverride{
    Title: "Test Task",
    Project: project.ID,
    Assignee: &user.ID,
})
```

### Assertion Helpers

Database-specific assertions:

```go
suite := integration.SetupDatabaseTest(t)

// Check entity existence
suite.Assert.UserExists(userID)
suite.Assert.ProjectHasOwner(projectID, ownerID)
suite.Assert.TaskHasParent(childTaskID, parentTaskID)

// Check constraints
suite.Assert.ConstraintViolated(err, "users_email_unique")
```

## Test Development Guidelines

### 1. Test Independence

Each test must be completely independent:

```go
func TestUserCreation(t *testing.T) {
    suite := integration.SetupDatabaseTest(t) // Fresh database
    // Test logic...
    // Automatic cleanup in t.Cleanup()
}
```

### 2. Real Data, No Mocks

Always use real database operations:

```go
// ✅ Good - uses real repository
user, err := suite.Repositories.UserRepository.Create(ctx, newUser)

// ❌ Bad - uses mock
mockRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(user, nil)
```

### 3. Build Tags

All integration tests must have build tags:

```go
//go:build integration
// +build integration

package integration

import "testing"

func TestExample(t *testing.T) {
    // Test code...
}
```

### 4. Error Testing

Test both success and failure scenarios:

```go
func TestUserCreation_DuplicateEmail(t *testing.T) {
    suite := integration.SetupDatabaseTest(t)
    
    // Create first user
    user1 := suite.Factory.CreateUser()
    
    // Attempt duplicate email
    user2 := domain.User{Email: user1.Email, Username: "different"}
    _, err := suite.Repositories.UserRepository.Create(context.Background(), user2)
    
    // Verify constraint violation
    suite.Assert.ConstraintViolated(err, "users_email_unique")
}
```

### 5. Concurrency Testing

Test concurrent operations:

```go
func TestConcurrentTaskUpdates(t *testing.T) {
    suite := integration.SetupDatabaseTest(t)
    task := suite.Factory.CreateTask()
    
    // Run concurrent updates
    var wg sync.WaitGroup
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(status string) {
            defer wg.Done()
            task.Status = domain.TaskStatus(status)
            suite.Repositories.TaskRepository.Update(context.Background(), task)
        }(fmt.Sprintf("status-%d", i))
    }
    wg.Wait()
    
    // Verify final state is consistent
    finalTask, _ := suite.Repositories.TaskRepository.GetByID(context.Background(), task.ID)
    assert.NotNil(t, finalTask)
}
```

## Performance Benchmarks

### Benchmark Tests

```go
func BenchmarkTaskRepository_Create(b *testing.B) {
    db := integration.NewTestDatabase(b, "")
    defer db.Cleanup()
    
    repo := repository.NewPocketBaseTaskRepository(db.App)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        task := domain.Task{Title: fmt.Sprintf("Task %d", i)}
        _, err := repo.Create(context.Background(), task)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

### Performance Expectations

- **Single operations**: < 10ms
- **Batch operations**: < 100ms for 100 records
- **Memory usage**: < 50MB for test suite
- **Database size**: < 100MB for full test run

## Troubleshooting

### Common Issues

1. **Tests hanging**: Check for database locks or missing cleanup
2. **Flaky tests**: Ensure proper test isolation and cleanup
3. **Performance degradation**: Check for missing indexes or inefficient queries
4. **Coverage drops**: Add tests for new repository methods

### Debug Commands

```bash
# Run specific test with verbose output
go test -tags=integration -v -run TestSpecific ./test/integration

# Check test compilation
go test -tags=integration -c ./internal/testutil/integration

# Run with race detection and verbose output
go test -tags=integration -race -v ./...
```

### Log Analysis

Integration tests use structured logging:

```bash
# Filter test logs
go test -tags=integration -v ./... 2>&1 | grep "TEST"

# Check database operations
export TEST_VERBOSE=1
go test -tags=integration -v ./...
```

## Coverage Analysis

### Viewing Coverage

```bash
# Generate coverage report
make test-integration-coverage

# View in browser
open coverage/integration.html
```

### Coverage Targets

- **Repository layer**: 95%+ coverage
- **Service layer**: 90%+ coverage  
- **Domain models**: 85%+ coverage
- **Overall integration**: 90%+ coverage

### Improving Coverage

1. Add tests for error conditions
2. Test all repository methods
3. Test edge cases and boundary conditions
4. Add concurrency and performance tests
5. Test failure scenarios and recovery

## Best Practices

1. **Isolation**: Each test gets fresh database
2. **Cleanup**: Always use `t.Cleanup()` or `defer`
3. **Deterministic**: Use factory for consistent test data
4. **Fast**: Keep tests under 5 seconds each
5. **Readable**: Clear test names and assertions
6. **Comprehensive**: Test all code paths and edge cases
7. **Maintainable**: Keep tests simple and focused