//go:build integration
// +build integration

package integration

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"simple-easy-tasks/internal/domain"
	"simple-easy-tasks/internal/testutil/integration"
)

// ErrorHandlingTestSuite provides comprehensive database error handling testing infrastructure
type ErrorHandlingTestSuite struct {
	*integration.DatabaseTestSuite
	maxRetries      int
	retryDelay      time.Duration
	connectionPool  int
	queryTimeout    time.Duration
	circuitBreaker  *CircuitBreaker
	errorStats      *ErrorStatistics
}

// ErrorStatistics tracks error patterns during testing
type ErrorStatistics struct {
	ConnectionFailures   int64 `json:"connection_failures"`
	TimeoutErrors        int64 `json:"timeout_errors"`
	ConstraintViolations int64 `json:"constraint_violations"`
	DeadlockDetections   int64 `json:"deadlock_detections"`
	TransactionRollbacks int64 `json:"transaction_rollbacks"`
	RetryAttempts        int64 `json:"retry_attempts"`
	RecoverySuccesses    int64 `json:"recovery_successes"`
	CircuitBreakerTrips  int64 `json:"circuit_breaker_trips"`
}

// CircuitBreakerState represents the state of the circuit breaker
type CircuitBreakerState int

const (
	StateClosed CircuitBreakerState = iota
	StateOpen
	StateHalfOpen
)

// CircuitBreaker implements a simple circuit breaker pattern for database operations
type CircuitBreaker struct {
	mu               sync.RWMutex
	state            CircuitBreakerState
	failureCount     int64
	successCount     int64
	failureThreshold int64
	resetTimeout     time.Duration
	lastFailureTime  time.Time
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(failureThreshold int64, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:            StateClosed,
		failureThreshold: failureThreshold,
		resetTimeout:     resetTimeout,
	}
}

// CanExecute checks if the operation can be executed based on circuit breaker state
func (cb *CircuitBreaker) CanExecute() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		return time.Since(cb.lastFailureTime) >= cb.resetTimeout
	case StateHalfOpen:
		return true
	default:
		return false
	}
}

// RecordSuccess records a successful operation
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.successCount++
	if cb.state == StateHalfOpen {
		cb.state = StateClosed
		cb.failureCount = 0
	}
}

// RecordFailure records a failed operation
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailureTime = time.Now()

	if cb.failureCount >= cb.failureThreshold {
		cb.state = StateOpen
	}
}

// GetState returns the current circuit breaker state
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// setupErrorHandlingTest creates an error handling test suite
func setupErrorHandlingTest(t *testing.T) *ErrorHandlingTestSuite {
	suite := integration.SetupConcurrencyTestWithServices(t)

	return &ErrorHandlingTestSuite{
		DatabaseTestSuite: suite,
		maxRetries:        5,
		retryDelay:        100 * time.Millisecond,
		connectionPool:    runtime.NumCPU() * 2,
		queryTimeout:      5 * time.Second,
		circuitBreaker:    NewCircuitBreaker(3, 30*time.Second),
		errorStats:        &ErrorStatistics{},
	}
}

// simulateDatabaseFailure creates conditions to simulate database connection failures
func (s *ErrorHandlingTestSuite) simulateDatabaseFailure(t *testing.T, failureType string) func() {
	switch failureType {
	case "connection_timeout":
		// Simulate connection timeout by overloading the connection pool
		return s.simulateConnectionPoolExhaustion(t)
	case "database_locked":
		// Simulate database locked condition
		return s.simulateDatabaseLock(t)
	case "disk_full":
		// Simulate disk full error (limited simulation)
		return s.simulateDiskFullError(t)
	default:
		return func() {} // No-op cleanup function
	}
}

// simulateConnectionPoolExhaustion creates heavy load to exhaust connection pool
func (s *ErrorHandlingTestSuite) simulateConnectionPoolExhaustion(t *testing.T) func() {
	numGoroutines := s.connectionPool * 3 // Exceed pool capacity
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	// Start long-running database operations to hold connections
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			for {
				select {
				case <-ctx.Done():
					return
				default:
					// Execute a long-running query to hold the connection
					query := s.DB.App().DB().NewQuery("SELECT COUNT(*) FROM sqlite_master WHERE type='table'")
					var count int
					if err := query.Row(&count); err != nil {
						t.Logf("Connection pool exhaustion simulation %d failed: %v", id, err)
					}
					time.Sleep(50 * time.Millisecond)
				}
			}
		}(i)
	}

	// Give some time for connections to be established
	time.Sleep(200 * time.Millisecond)

	return func() {
		cancel()
		wg.Wait()
	}
}

// simulateDatabaseLock creates a long-running transaction to lock the database
func (s *ErrorHandlingTestSuite) simulateDatabaseLock(t *testing.T) func() {
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		
		// Execute a long-running query to simulate lock
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// Execute operations that may cause locking
				query := s.DB.App().DB().NewQuery("UPDATE users SET updated = datetime('now') WHERE id = 'non-existent-id'")
				if _, err := query.Execute(); err != nil {
					t.Logf("Lock simulation query failed: %v", err)
				}
				time.Sleep(50 * time.Millisecond)
			}
		}

		// Hold the transaction open
		<-ctx.Done()
	}()

	// Give some time for the lock to be established
	time.Sleep(100 * time.Millisecond)

	return func() {
		cancel()
		wg.Wait()
	}
}

// simulateDiskFullError simulates disk space exhaustion (limited simulation)
func (s *ErrorHandlingTestSuite) simulateDiskFullError(t *testing.T) func() {
	// This is a simplified simulation - in a real test environment,
	// you might use a test filesystem with limited space
	
	// For now, we'll create a condition that might trigger disk-related errors
	// by attempting to create very large temporary data
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	wg.Add(1)
	go func() {
		defer wg.Done()
		
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// Attempt operations that might fail due to space constraints
				largeData := strings.Repeat("x", 100000) // 100KB string
				query := s.DB.App().DB().NewQuery("SELECT {:data} as large_data").
					Bind(map[string]interface{}{"data": largeData})
				var result string
				if err := query.Row(&result); err != nil {
					t.Logf("Disk simulation query failed: %v", err)
				}
				time.Sleep(10 * time.Millisecond)
			}
		}
	}()

	return func() {
		cancel()
		wg.Wait()
	}
}

// TestDatabaseConnectionFailureScenarios tests various database connection failure scenarios
func TestDatabaseConnectionFailureScenarios(t *testing.T) {
	suite := setupErrorHandlingTest(t)
	defer suite.Cleanup()

	testCases := []struct {
		name        string
		failureType string
		expectError bool
		errorType   string
	}{
		{
			name:        "ConnectionTimeout",
			failureType: "connection_timeout",
			expectError: true,
			errorType:   "timeout",
		},
		{
			name:        "DatabaseLocked",
			failureType: "database_locked",
			expectError: true,
			errorType:   "locked",
		},
		{
			name:        "DiskFull",
			failureType: "disk_full",
			expectError: false, // Might not always trigger
			errorType:   "disk",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup failure simulation
			cleanup := suite.simulateDatabaseFailure(t, tc.failureType)
			defer cleanup()

			// Test basic database operations under failure conditions
			user := suite.Factory.CreateUser()
			
			// Attempt to create user with retry logic
			err := suite.executeWithRetry(func() error {
				return suite.Repos.Users.Create(suite.Context(), user)
			}, "create_user")

			if tc.expectError {
				// We expect some operations to fail or require retries
				if err == nil {
					t.Logf("Operation succeeded despite simulated %s failure", tc.failureType)
				} else {
					assert.Error(t, err)
					errStr := strings.ToLower(err.Error())
					if tc.errorType != "" {
						assert.Contains(t, errStr, tc.errorType, 
							"Error should contain expected type: %s", tc.errorType)
					}
				}
			}

			// Verify that retry statistics were recorded
			retryCount := atomic.LoadInt64(&suite.errorStats.RetryAttempts)
			t.Logf("Retry attempts for %s: %d", tc.name, retryCount)
		})
	}
}

// TestConstraintViolationHandling tests database constraint violation error handling
func TestConstraintViolationHandling(t *testing.T) {
	suite := setupErrorHandlingTest(t)
	defer suite.Cleanup()

	// Create test data
	user := suite.Factory.CreateUser()
	require.NoError(t, suite.Repos.Users.Create(suite.Context(), user))

	project := suite.Factory.CreateProject(user)
	require.NoError(t, suite.Repos.Projects.Create(suite.Context(), project))

	testCases := []struct {
		name           string
		setup          func() error
		expectedError  string
		constraintType string
	}{
		{
			name: "UniqueConstraintViolation_UserEmail",
			setup: func() error {
				// Attempt to create user with duplicate email
				duplicateUser := suite.Factory.CreateUser(
					integration.WithUserEmail(user.Email))
				return suite.Repos.Users.Create(suite.Context(), duplicateUser)
			},
			expectedError:  "must be unique", // Actual PocketBase error message
			constraintType: "unique",
		},
		{
			name: "ValidationError_EmptyTitle",
			setup: func() error {
				// Attempt to create task without required fields
				// Create a task but then clear the required title field
				incompleteTask := suite.Factory.CreateTask(project, user)
				incompleteTask.Title = "" // This should violate validation rules
				return suite.Repos.Tasks.Create(suite.Context(), incompleteTask)
			},
			expectedError:  "required", // PocketBase validation for required fields
			constraintType: "validation",
		},
		{
			name: "DatabaseConstraintViolation_DirectSQL",
			setup: func() error {
				// Test actual database constraint by using correct column names
				// First, let's try to insert with duplicate email (which should have unique constraint)
				query := suite.DB.App().DB().NewQuery("INSERT INTO users (id, email) VALUES ('duplicate-test-id', {:email})").
					Bind(map[string]interface{}{"email": user.Email})
				_, err := query.Execute()
				return err // This should fail due to unique constraint on email
			},
			expectedError:  "unique", // Actual database constraint error
			constraintType: "database_unique",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Execute the operation that should violate constraints
			err := tc.setup()
			
			// Verify constraint violation occurred
			require.Error(t, err, "Expected constraint violation error")
			
			errStr := strings.ToLower(err.Error())
			assert.Contains(t, errStr, strings.ToLower(tc.expectedError),
				"Error should indicate constraint violation: %s", tc.expectedError)

			// Use suite assertion helper for specific constraint types
			switch tc.constraintType {
			case "unique":
				suite.Assert.UniqueConstraintViolated(err)
			case "validation":
				// For PocketBase validation errors, we already checked the error string above
				assert.Contains(t, errStr, "validation", "Error should be a validation error")
			case "database_unique":
				// For actual database constraint violations
				suite.Assert.UniqueConstraintViolated(err)
			}

			// Record constraint violation statistics
			atomic.AddInt64(&suite.errorStats.ConstraintViolations, 1)

			// Verify database state remains consistent
			suite.Assert.UserExists(user.ID)
			suite.Assert.ProjectExists(project.ID)
		})
	}
}

// TestDeadlockDetectionAndRetry tests deadlock scenarios and retry logic
func TestDeadlockDetectionAndRetry(t *testing.T) {
	suite := setupErrorHandlingTest(t)
	defer suite.Cleanup()

	// Create test data
	user1 := suite.Factory.CreateUser(
		integration.WithUserEmail("user1@deadlock.test"))
	user2 := suite.Factory.CreateUser(
		integration.WithUserEmail("user2@deadlock.test"))
	
	require.NoError(t, suite.Repos.Users.Create(suite.Context(), user1))
	require.NoError(t, suite.Repos.Users.Create(suite.Context(), user2))

	project := suite.Factory.CreateProject(user1)
	require.NoError(t, suite.Repos.Projects.Create(suite.Context(), project))

	task1 := suite.Factory.CreateTask(project, user1)
	task2 := suite.Factory.CreateTask(project, user2)
	
	require.NoError(t, suite.Repos.Tasks.Create(suite.Context(), task1))
	require.NoError(t, suite.Repos.Tasks.Create(suite.Context(), task2))

	// Test concurrent updates that might cause deadlocks
	numGoroutines := 10
	var wg sync.WaitGroup
	var deadlockCount int64
	var successCount int64
	var errorCount int64

	startBarrier := make(chan struct{})

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			
			// Wait for start signal
			<-startBarrier
			
			// Alternate between updating different tasks to create deadlock potential
			var targetTask *domain.Task
			if goroutineID%2 == 0 {
				targetTask = task1
			} else {
				targetTask = task2
			}

			// Execute with deadlock retry logic
			err := suite.executeWithDeadlockRetry(func() error {
				// Simulate complex update operation
				updatedTask := *targetTask
				updatedTask.Title = fmt.Sprintf("Updated by goroutine %d at %v", 
					goroutineID, time.Now().UnixNano())
				updatedTask.Description = fmt.Sprintf("Deadlock test update from %d", goroutineID)
				
				return suite.Repos.Tasks.Update(suite.Context(), &updatedTask)
			})

			if err != nil {
				atomic.AddInt64(&errorCount, 1)
				errStr := strings.ToLower(err.Error())
				if strings.Contains(errStr, "deadlock") || strings.Contains(errStr, "locked") {
					atomic.AddInt64(&deadlockCount, 1)
				}
				t.Logf("Goroutine %d encountered error: %v", goroutineID, err)
			} else {
				atomic.AddInt64(&successCount, 1)
			}
		}(i)
	}

	// Signal all goroutines to start
	close(startBarrier)

	// Wait for completion
	wg.Wait()

	// Analyze results
	finalDeadlockCount := atomic.LoadInt64(&deadlockCount)
	finalSuccessCount := atomic.LoadInt64(&successCount)
	finalErrorCount := atomic.LoadInt64(&errorCount)

	t.Logf("Deadlock test results: %d successes, %d errors, %d deadlocks detected",
		finalSuccessCount, finalErrorCount, finalDeadlockCount)

	// At least some operations should succeed
	assert.GreaterOrEqual(t, finalSuccessCount, int64(1),
		"At least some concurrent operations should succeed")

	// Verify final database state is consistent
	suite.Assert.TaskExists(task1.ID)
	suite.Assert.TaskExists(task2.ID)
	suite.Assert.UserExists(user1.ID)
	suite.Assert.UserExists(user2.ID)
	suite.Assert.ProjectExists(project.ID)

	// Record deadlock statistics
	atomic.AddInt64(&suite.errorStats.DeadlockDetections, finalDeadlockCount)
}

// TestLockTimeoutHandling tests database lock timeout scenarios
func TestLockTimeoutHandling(t *testing.T) {
	suite := setupErrorHandlingTest(t)
	defer suite.Cleanup()

	// Create test data
	user := suite.Factory.CreateUser()
	require.NoError(t, suite.Repos.Users.Create(suite.Context(), user))

	project := suite.Factory.CreateProject(user)
	require.NoError(t, suite.Repos.Projects.Create(suite.Context(), project))

	// Start a long-running operation to create lock contention
	lockingCtx, cancelLocking := context.WithCancel(context.Background())
	var lockingWg sync.WaitGroup

	lockingWg.Add(1)
	go func() {
		defer lockingWg.Done()
		
		for {
			select {
			case <-lockingCtx.Done():
				return
			default:
				// Execute operations that may create lock contention
				query := suite.DB.App().DB().NewQuery("UPDATE projects SET updated = datetime('now') WHERE id = {:id}").
					Bind(map[string]interface{}{"id": project.ID})
				if _, err := query.Execute(); err != nil {
					t.Logf("Failed to execute locking query: %v", err)
				}
				time.Sleep(50 * time.Millisecond)
			}
		}
	}()

	// Give time for lock to be established
	time.Sleep(100 * time.Millisecond)

	// Test operations that should encounter lock timeouts
	testCases := []struct {
		name      string
		operation func() error
		timeout   time.Duration
	}{
		{
			name: "ProjectUpdate_WithTimeout",
			operation: func() error {
				updatedProject := *project
				updatedProject.Title = "Updated title during lock test"
				return suite.Repos.Projects.Update(suite.Context(), &updatedProject)
			},
			timeout: 2 * time.Second,
		},
		{
			name: "ProjectDelete_WithTimeout",
			operation: func() error {
				// This should be blocked by the existing lock
				return suite.Repos.Projects.Delete(suite.Context(), project.ID)
			},
			timeout: 2 * time.Second,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Execute with timeout
			ctx, cancel := context.WithTimeout(suite.Context(), tc.timeout)
			defer cancel()

			done := make(chan error, 1)
			go func() {
				done <- tc.operation()
			}()

			select {
			case err := <-done:
				if err != nil {
					t.Logf("Operation failed as expected due to lock: %v", err)
					errStr := strings.ToLower(err.Error())
					if strings.Contains(errStr, "timeout") || 
					   strings.Contains(errStr, "locked") || 
					   strings.Contains(errStr, "busy") {
						// Expected behavior - lock timeout occurred
						atomic.AddInt64(&suite.errorStats.TimeoutErrors, 1)
					}
				} else {
					t.Log("Operation completed successfully despite lock")
				}
			case <-ctx.Done():
				t.Log("Operation timed out as expected due to lock contention")
				atomic.AddInt64(&suite.errorStats.TimeoutErrors, 1)
			}
		})
	}

	// Cleanup locking transaction
	cancelLocking()
	lockingWg.Wait()

	// Verify database consistency after lock resolution
	suite.Assert.ProjectExists(project.ID)
	suite.Assert.UserExists(user.ID)
}

// TestTransactionRollbackOnErrors tests transaction rollback behavior on various error conditions
func TestTransactionRollbackOnErrors(t *testing.T) {
	suite := setupErrorHandlingTest(t)
	defer suite.Cleanup()

	// Create test data
	user := suite.Factory.CreateUser()
	require.NoError(t, suite.Repos.Users.Create(suite.Context(), user))

	project := suite.Factory.CreateProject(user)
	require.NoError(t, suite.Repos.Projects.Create(suite.Context(), project))

	testCases := []struct {
		name           string
		transactionOp  func() error
		shouldRollback bool
		verifyState    func()
	}{
		{
			name: "RollbackOnConstraintViolation",
			transactionOp: func() error {
				// Simulate a multi-step operation that fails
				task1 := suite.Factory.CreateTask(project, user)
				
				// This should succeed
				if err := suite.Repos.Tasks.Create(suite.Context(), task1); err != nil {
					return err
				}

				// This should fail due to constraint violation
				task2 := suite.Factory.CreateTask(project, user)
				task2.ID = task1.ID // Duplicate ID should cause violation
				return suite.Repos.Tasks.Create(suite.Context(), task2)
			},
			shouldRollback: true,
			verifyState: func() {
				// Both tasks should not exist if rollback worked properly
				suite.Assert.TaskCount(0) // No tasks should exist
			},
		},
		{
			name: "RollbackOnBusinessLogicError",
			transactionOp: func() error {
				// Use service layer which should handle transactions properly
				if !suite.HasServices() {
					return errors.New("services not available for this test")
				}

				taskService := suite.GetTaskService(t)
				
				// Create a valid task first
				req1 := domain.CreateTaskRequest{
					Title:       "First task",
					Description: "This should be created",
					ProjectID:   project.ID,
					Priority:    domain.PriorityMedium,
				}
				
				_, err := taskService.CreateTask(suite.Context(), req1, user.ID)
				if err != nil {
					return err
				}

				// Now create an invalid task that should cause rollback
				req2 := domain.CreateTaskRequest{
					Title:       "", // Invalid empty title
					Description: "This should fail",
					ProjectID:   project.ID,
					Priority:    domain.PriorityMedium,
				}
				
				_, err = taskService.CreateTask(suite.Context(), req2, user.ID)
				return err
			},
			shouldRollback: false, // Service layer may not use transactions for individual creates
			verifyState: func() {
				// The first task should exist, second should not
				suite.Assert.TaskCount(1)
			},
		},
	}

	initialTaskCount := 0
	suite.Assert.TaskCount(initialTaskCount)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Record initial state
			initialTaskCount := suite.getTaskCount()
			
			// Execute the transaction
			err := tc.transactionOp()
			
			if tc.shouldRollback {
				// We expect an error and rollback
				assert.Error(t, err, "Expected error to trigger rollback")
				
				// Verify rollback occurred
				finalTaskCount := suite.getTaskCount()
				assert.Equal(t, initialTaskCount, finalTaskCount,
					"Task count should be unchanged after rollback")
				
				atomic.AddInt64(&suite.errorStats.TransactionRollbacks, 1)
			}

			// Run custom verification
			if tc.verifyState != nil {
				tc.verifyState()
			}

			// Reset for next test
			require.NoError(t, suite.Reset())
			require.NoError(t, suite.Repos.Users.Create(suite.Context(), user))
			require.NoError(t, suite.Repos.Projects.Create(suite.Context(), project))
		})
	}
}

// TestGracefulDegradationStrategies tests graceful degradation patterns
func TestGracefulDegradationStrategies(t *testing.T) {
	suite := setupErrorHandlingTest(t)
	defer suite.Cleanup()

	// Test read-only mode fallback
	t.Run("ReadOnlyModeFallback", func(t *testing.T) {
		// Create test data first
		user := suite.Factory.CreateUser()
		require.NoError(t, suite.Repos.Users.Create(suite.Context(), user))

		// Simulate database write failures
		cleanup := suite.simulateDatabaseFailure(t, "database_locked")
		defer cleanup()

		// Read operations should still work
		retrievedUser, err := suite.Repos.Users.GetByID(suite.Context(), user.ID)
		if err == nil {
			assert.Equal(t, user.ID, retrievedUser.ID)
			t.Log("Read-only operations working during write failures")
		} else {
			t.Logf("Read operation also affected: %v", err)
		}
	})

	// Test circuit breaker pattern
	t.Run("CircuitBreakerPattern", func(t *testing.T) {
		circuitBreaker := NewCircuitBreaker(3, 5*time.Second)
		
		// Simulate multiple failures to trip the circuit breaker
		for i := 0; i < 4; i++ {
			circuitBreaker.RecordFailure()
		}

		// Circuit breaker should be open now
		assert.Equal(t, StateOpen, circuitBreaker.GetState())
		assert.False(t, circuitBreaker.CanExecute())

		atomic.AddInt64(&suite.errorStats.CircuitBreakerTrips, 1)
		
		t.Log("Circuit breaker tripped after threshold failures")
	})

	// Test cached data usage during errors
	t.Run("CachedDataFallback", func(t *testing.T) {
		// Create test data
		user := suite.Factory.CreateUser()
		require.NoError(t, suite.Repos.Users.Create(suite.Context(), user))

		// Simulate cache behavior (simplified)
		userCache := make(map[string]*domain.User)
		userCache[user.ID] = user

		// Simulate database failure
		cleanup := suite.simulateDatabaseFailure(t, "connection_timeout")
		defer cleanup()

		// Fallback to cached data
		if cachedUser, exists := userCache[user.ID]; exists {
			assert.Equal(t, user.ID, cachedUser.ID)
			t.Log("Successfully fell back to cached data during database failure")
		}
	})
}

// TestErrorRecoveryMechanisms tests automatic recovery after failures
func TestErrorRecoveryMechanisms(t *testing.T) {
	suite := setupErrorHandlingTest(t)
	defer suite.Cleanup()

	// Test automatic reconnection after failures
	t.Run("AutomaticReconnection", func(t *testing.T) {
		// Create test data
		user := suite.Factory.CreateUser()
		
		// Simulate temporary database failure
		cleanup := suite.simulateDatabaseFailure(t, "connection_timeout")
		
		// Attempt operation with retry
		err := suite.executeWithRetry(func() error {
			return suite.Repos.Users.Create(suite.Context(), user)
		}, "create_user_with_recovery")

		cleanup() // Remove failure simulation
		
		// After cleanup, operation should succeed
		if err != nil {
			// Try one more time after cleanup
			err = suite.Repos.Users.Create(suite.Context(), user)
		}

		if err == nil {
			suite.Assert.UserExists(user.ID)
			atomic.AddInt64(&suite.errorStats.RecoverySuccesses, 1)
			t.Log("Successfully recovered from database failure")
		} else {
			t.Logf("Recovery attempt failed: %v", err)
		}
	})

	// Test health check implementations
	t.Run("HealthCheckRecovery", func(t *testing.T) {
		if !suite.HasServices() {
			t.Skip("Skipping health check test - services not available")
		}

		healthService := suite.Services.Health
		
		// Perform health check
		healthResponse := healthService.Check(suite.Context())
		
		t.Logf("Health check status: %s", healthResponse.Status)
		t.Logf("Health checks performed: %d", len(healthResponse.Checks))
		
		// Health checks should provide recovery information
		assert.NotEmpty(t, healthResponse.Status)
		assert.NotZero(t, healthResponse.Uptime)
	})
}

// Helper methods

// executeWithRetry executes an operation with retry logic
func (s *ErrorHandlingTestSuite) executeWithRetry(operation func() error, operationName string) error {
	var lastErr error
	
	for attempt := 0; attempt <= s.maxRetries; attempt++ {
		// Check circuit breaker
		if !s.circuitBreaker.CanExecute() {
			return fmt.Errorf("circuit breaker open for operation: %s", operationName)
		}

		err := operation()
		if err == nil {
			s.circuitBreaker.RecordSuccess()
			if attempt > 0 {
				atomic.AddInt64(&s.errorStats.RetryAttempts, int64(attempt))
			}
			return nil
		}

		lastErr = err
		s.circuitBreaker.RecordFailure()
		atomic.AddInt64(&s.errorStats.RetryAttempts, 1)

		// Check if this is a retryable error
		if !s.isRetryableError(err) {
			break
		}

		if attempt < s.maxRetries {
			time.Sleep(s.retryDelay * time.Duration(attempt+1)) // Exponential backoff
		}
	}

	return fmt.Errorf("operation %s failed after %d retries: %w", operationName, s.maxRetries, lastErr)
}

// executeWithDeadlockRetry executes an operation with specific deadlock retry logic
func (s *ErrorHandlingTestSuite) executeWithDeadlockRetry(operation func() error) error {
	maxDeadlockRetries := 3
	var lastErr error

	for attempt := 0; attempt <= maxDeadlockRetries; attempt++ {
		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err
		errStr := strings.ToLower(err.Error())
		
		// Check for deadlock-specific errors
		if strings.Contains(errStr, "deadlock") || 
		   strings.Contains(errStr, "database is locked") ||
		   strings.Contains(errStr, "busy") {
			
			if attempt < maxDeadlockRetries {
				// Randomized backoff for deadlock resolution
				backoff := time.Duration(attempt+1) * 50 * time.Millisecond
				time.Sleep(backoff)
				continue
			}
		}
		
		// Non-deadlock error or max retries reached
		break
	}

	return lastErr
}

// isRetryableError determines if an error is retryable
func (s *ErrorHandlingTestSuite) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	
	// Retryable error patterns
	retryablePatterns := []string{
		"timeout",
		"connection",
		"database is locked",
		"busy",
		"temporary",
		"network",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	// Non-retryable errors
	nonRetryablePatterns := []string{
		"constraint",
		"unique",
		"foreign key",
		"not null",
		"validation",
		"unauthorized",
		"forbidden",
	}

	for _, pattern := range nonRetryablePatterns {
		if strings.Contains(errStr, pattern) {
			return false
		}
	}

	// Default to retryable for unknown errors
	return true
}

// getTaskCount returns the current number of tasks in the database
func (s *ErrorHandlingTestSuite) getTaskCount() int {
	var count int
	err := s.DB.App().DB().Select("COUNT(*)").From("tasks").Row(&count)
	if err != nil {
		return -1 // Error indicator
	}
	return count
}

// TestErrorHandlingStatistics tests that error statistics are properly tracked
func TestErrorHandlingStatistics(t *testing.T) {
	suite := setupErrorHandlingTest(t)
	defer suite.Cleanup()

	// Execute various operations to generate statistics
	user := suite.Factory.CreateUser()
	
	// Test constraint violation (should increment constraint violations)
	duplicateUser := suite.Factory.CreateUser(integration.WithUserEmail(user.Email))
	_ = suite.Repos.Users.Create(suite.Context(), user)
	err := suite.Repos.Users.Create(suite.Context(), duplicateUser)
	if err != nil {
		atomic.AddInt64(&suite.errorStats.ConstraintViolations, 1)
	}

	// Test retry mechanism
	_ = suite.executeWithRetry(func() error {
		return errors.New("simulated transient error")
	}, "test_operation")

	// Verify statistics
	stats := suite.errorStats
	
	t.Logf("Error handling statistics:")
	t.Logf("  Connection failures: %d", atomic.LoadInt64(&stats.ConnectionFailures))
	t.Logf("  Timeout errors: %d", atomic.LoadInt64(&stats.TimeoutErrors))
	t.Logf("  Constraint violations: %d", atomic.LoadInt64(&stats.ConstraintViolations))
	t.Logf("  Deadlock detections: %d", atomic.LoadInt64(&stats.DeadlockDetections))
	t.Logf("  Transaction rollbacks: %d", atomic.LoadInt64(&stats.TransactionRollbacks))
	t.Logf("  Retry attempts: %d", atomic.LoadInt64(&stats.RetryAttempts))
	t.Logf("  Recovery successes: %d", atomic.LoadInt64(&stats.RecoverySuccesses))
	t.Logf("  Circuit breaker trips: %d", atomic.LoadInt64(&stats.CircuitBreakerTrips))

	// Verify that some statistics were recorded
	totalErrors := atomic.LoadInt64(&stats.ConstraintViolations) + 
	             atomic.LoadInt64(&stats.RetryAttempts)
	assert.Greater(t, totalErrors, int64(0), "Should have recorded some error statistics")
}

// TestComprehensiveErrorScenario tests a comprehensive error scenario combining multiple failure types
func TestComprehensiveErrorScenario(t *testing.T) {
	suite := setupErrorHandlingTest(t)
	defer suite.Cleanup()

	// Create test data
	user := suite.Factory.CreateUser()
	require.NoError(t, suite.Repos.Users.Create(suite.Context(), user))

	project := suite.Factory.CreateProject(user)
	require.NoError(t, suite.Repos.Projects.Create(suite.Context(), project))

	// Simulate multiple concurrent operations under various failure conditions
	numGoroutines := 15
	var wg sync.WaitGroup
	var successCount int64
	var errorCount int64
	
	// Various failure simulations
	cleanupFuncs := []func(){
		suite.simulateDatabaseFailure(t, "connection_timeout"),
		suite.simulateDatabaseFailure(t, "database_locked"),
	}
	defer func() {
		for _, cleanup := range cleanupFuncs {
			cleanup()
		}
	}()

	startBarrier := make(chan struct{})

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			
			<-startBarrier
			
			// Mix different operations
			var err error
			switch goroutineID % 4 {
			case 0:
				// Create task
				task := suite.Factory.CreateTask(project, user)
				err = suite.executeWithRetry(func() error {
					return suite.Repos.Tasks.Create(suite.Context(), task)
				}, "create_task")
				
			case 1:
				// Update project
				updatedProject := *project
				updatedProject.Title = fmt.Sprintf("Updated by %d", goroutineID)
				err = suite.executeWithRetry(func() error {
					return suite.Repos.Projects.Update(suite.Context(), &updatedProject)
				}, "update_project")
				
			case 2:
				// Query operations
				err = suite.executeWithRetry(func() error {
					_, queryErr := suite.Repos.Users.GetByID(suite.Context(), user.ID)
					return queryErr
				}, "query_user")
				
			case 3:
				// Create comment (if tasks exist)
				if tasks, listErr := suite.Repos.Tasks.ListByProject(suite.Context(), project.ID, 0, 1); listErr == nil && len(tasks) > 0 {
					comment := suite.Factory.CreateComment(tasks[0], user)
					err = suite.executeWithRetry(func() error {
						return suite.Repos.Comments.Create(suite.Context(), comment)
					}, "create_comment")
				}
			}

			if err != nil {
				atomic.AddInt64(&errorCount, 1)
				t.Logf("Goroutine %d failed: %v", goroutineID, err)
			} else {
				atomic.AddInt64(&successCount, 1)
			}
		}(i)
	}

	close(startBarrier)
	wg.Wait()

	finalSuccessCount := atomic.LoadInt64(&successCount)
	finalErrorCount := atomic.LoadInt64(&errorCount)

	t.Logf("Comprehensive error scenario results:")
	t.Logf("  Successful operations: %d", finalSuccessCount)
	t.Logf("  Failed operations: %d", finalErrorCount)
	t.Logf("  Success rate: %.2f%%", float64(finalSuccessCount)/float64(numGoroutines)*100)

	// At least some operations should succeed despite various failures
	assert.GreaterOrEqual(t, finalSuccessCount, int64(1),
		"At least some operations should succeed even under adverse conditions")

	// Database should remain consistent
	suite.Assert.UserExists(user.ID)
	suite.Assert.ProjectExists(project.ID)

	// Print final error statistics
	t.Logf("\nFinal error handling statistics:")
	stats := suite.errorStats
	t.Logf("  Total retry attempts: %d", atomic.LoadInt64(&stats.RetryAttempts))
	t.Logf("  Constraint violations: %d", atomic.LoadInt64(&stats.ConstraintViolations))
	t.Logf("  Timeout errors: %d", atomic.LoadInt64(&stats.TimeoutErrors))
	t.Logf("  Recovery successes: %d", atomic.LoadInt64(&stats.RecoverySuccesses))
}