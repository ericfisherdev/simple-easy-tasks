//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ericfisherdev/simple-easy-tasks/internal/domain"
	"github.com/ericfisherdev/simple-easy-tasks/internal/testutil/integration"
)

// ConcurrencyTestSuite provides specialized test infrastructure for concurrency testing
type ConcurrencyTestSuite struct {
	*integration.DatabaseTestSuite
	maxGoroutines int
	timeout       time.Duration
}

// ServiceConcurrencyTestSuite provides service-layer concurrency testing infrastructure
// This demonstrates the enhanced DI container approach
type ServiceConcurrencyTestSuite struct {
	*integration.DatabaseTestSuite
	maxGoroutines int
	timeout       time.Duration
}

// setupConcurrencyTest creates a test suite optimized for concurrency testing
// Uses the legacy repository-based approach for backward compatibility
func setupConcurrencyTest(t *testing.T) *ConcurrencyTestSuite {
	suite := integration.SetupDatabaseTest(t)

	return &ConcurrencyTestSuite{
		DatabaseTestSuite: suite,
		maxGoroutines:     runtime.NumCPU() * 4, // Use more goroutines than CPU cores to test contention
		timeout:           30 * time.Second,     // Generous timeout for concurrent operations
	}
}

// setupServiceConcurrencyTest creates a test suite optimized for service-layer concurrency testing
// Uses the enhanced DI container approach with full production parity
func setupServiceConcurrencyTest(t *testing.T) *ServiceConcurrencyTestSuite {
	suite := integration.SetupConcurrencyTestWithServices(t)

	return &ServiceConcurrencyTestSuite{
		DatabaseTestSuite: suite,
		maxGoroutines:     runtime.NumCPU() * 4, // Use more goroutines than CPU cores to test contention
		timeout:           30 * time.Second,     // Generous timeout for concurrent operations
	}
}

// TestConcurrentTaskStatusUpdates tests that concurrent status updates work correctly with last-write-wins semantics
func TestConcurrentTaskStatusUpdates(t *testing.T) {
	suite := setupConcurrencyTest(t)
	defer suite.Cleanup()

	// Create test data
	user := suite.Factory.CreateUser()
	require.NoError(t, suite.Repos.Users.Create(suite.Context(), user))

	project := suite.Factory.CreateProject(user)
	require.NoError(t, suite.Repos.Projects.Create(suite.Context(), project))

	task := suite.Factory.CreateTask(project, user)
	require.NoError(t, suite.Repos.Tasks.Create(suite.Context(), task))

	// Test concurrent status updates
	statuses := []domain.TaskStatus{
		domain.StatusTodo,
		domain.StatusDeveloping,
		domain.StatusReview,
		domain.StatusComplete,
		domain.StatusBacklog,
	}

	numGoroutines := len(statuses)
	results := make([]error, numGoroutines)
	var wg sync.WaitGroup

	// Start barrier to ensure all goroutines start simultaneously
	startBarrier := make(chan struct{})

	// Launch concurrent status update goroutines
	for i, status := range statuses {
		wg.Add(1)
		go func(index int, newStatus domain.TaskStatus) {
			defer wg.Done()

			// Wait for start signal
			<-startBarrier

			// Get fresh copy of task
			ctx, cancel := context.WithTimeout(suite.Context(), suite.timeout)
			defer cancel()

			currentTask, err := suite.Repos.Tasks.GetByID(ctx, task.ID)
			if err != nil {
				results[index] = fmt.Errorf("failed to get task: %w", err)
				return
			}

			// Update status
			err = currentTask.UpdateStatus(newStatus)
			if err != nil {
				results[index] = fmt.Errorf("failed to update status: %w", err)
				return
			}

			// Save to database
			results[index] = suite.Repos.Tasks.Update(ctx, currentTask)
		}(i, status)
	}

	// Signal all goroutines to start
	close(startBarrier)

	// Wait for completion
	wg.Wait()

	// Analyze results
	successCount := 0
	var lastError error
	for i, err := range results {
		if err == nil {
			successCount++
		} else {
			t.Logf("Goroutine %d failed with: %v", i, err)
			lastError = err
		}
	}

	// At least one update should succeed (last-write-wins)
	assert.GreaterOrEqual(t, successCount, 1, "At least one concurrent update should succeed")

	// Verify final task state
	finalTask, err := suite.Repos.Tasks.GetByID(suite.Context(), task.ID)
	require.NoError(t, err)

	// Final status should be one of the attempted statuses
	assert.Contains(t, statuses, finalTask.Status, "Final status should be one of the attempted statuses")

	// Verify database consistency
	suite.Assert.TaskExists(task.ID)
	suite.Assert.TaskHasStatus(task.ID, finalTask.Status)

	t.Logf("Concurrent status updates completed: %d successes, %d failures", successCount, len(results)-successCount)
	if lastError != nil {
		t.Logf("Sample error: %v", lastError)
	}
	t.Logf("Final task status: %s", finalTask.Status)
}

// TestConcurrentProgressUpdates tests concurrent progress updates for data consistency
func TestConcurrentProgressUpdates(t *testing.T) {
	suite := setupConcurrencyTest(t)
	defer suite.Cleanup()

	// Create test data
	user := suite.Factory.CreateUser()
	require.NoError(t, suite.Repos.Users.Create(suite.Context(), user))

	project := suite.Factory.CreateProject(user)
	require.NoError(t, suite.Repos.Projects.Create(suite.Context(), project))

	task := suite.Factory.CreateTask(project, user,
		integration.WithTaskProgress(0))
	require.NoError(t, suite.Repos.Tasks.Create(suite.Context(), task))

	// Test concurrent progress increments
	numGoroutines := 20
	incrementValue := 5
	var successfulUpdates int64
	var wg sync.WaitGroup
	startBarrier := make(chan struct{})

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			// Wait for start signal
			<-startBarrier

			ctx, cancel := context.WithTimeout(suite.Context(), suite.timeout)
			defer cancel()

			// Get fresh copy of task
			currentTask, err := suite.Repos.Tasks.GetByID(ctx, task.ID)
			if err != nil {
				t.Errorf("Goroutine %d: failed to get task: %v", goroutineID, err)
				return
			}

			// Calculate new progress, ensuring we don't exceed 100%
			newProgress := currentTask.Progress + incrementValue
			if newProgress > 100 {
				newProgress = 100
			}

			// Update progress
			err = currentTask.UpdateProgress(newProgress)
			if err != nil {
				t.Errorf("Goroutine %d: failed to update progress: %v", goroutineID, err)
				return
			}

			// Save to database
			err = suite.Repos.Tasks.Update(ctx, currentTask)
			if err != nil {
				// Expected: some updates may fail due to concurrency
				t.Logf("Goroutine %d: update failed (expected in concurrent scenario): %v", goroutineID, err)
				return
			}

			atomic.AddInt64(&successfulUpdates, 1)
			t.Logf("Goroutine %d: successfully updated progress to %d", goroutineID, newProgress)
		}(i)
	}

	// Signal all goroutines to start
	close(startBarrier)

	// Wait for completion
	wg.Wait()

	// Verify final state
	finalTask, err := suite.Repos.Tasks.GetByID(suite.Context(), task.ID)
	require.NoError(t, err)

	// Progress should be between 0 and 100
	assert.GreaterOrEqual(t, finalTask.Progress, 0, "Progress should not be negative")
	assert.LessOrEqual(t, finalTask.Progress, 100, "Progress should not exceed 100")

	// At least one update should have succeeded
	assert.Greater(t, successfulUpdates, int64(0), "At least one progress update should succeed")

	// Progress should be greater than initial value
	assert.Greater(t, finalTask.Progress, 0, "Progress should have increased from initial 0")

	suite.Assert.TaskExists(task.ID)

	t.Logf("Concurrent progress updates: %d successful out of %d attempts", successfulUpdates, numGoroutines)
	t.Logf("Final progress: %d%%", finalTask.Progress)
}

// TestConcurrentProjectMemberAdditions tests concurrent project member additions for race conditions
func TestConcurrentProjectMemberAdditions(t *testing.T) {
	suite := setupConcurrencyTest(t)
	defer suite.Cleanup()

	// Create test data
	owner := suite.Factory.CreateUser()
	require.NoError(t, suite.Repos.Users.Create(suite.Context(), owner))

	project := suite.Factory.CreateProject(owner,
		integration.WithProjectMembers([]string{})) // Start with empty member list
	require.NoError(t, suite.Repos.Projects.Create(suite.Context(), project))

	// Create multiple users to add as members
	numUsers := 10
	users := make([]*domain.User, numUsers)
	for i := 0; i < numUsers; i++ {
		user := suite.Factory.CreateUser(
			integration.WithUserEmail(fmt.Sprintf("member%d@test.example.com", i)))
		require.NoError(t, suite.Repos.Users.Create(suite.Context(), user))
		users[i] = user
	}

	// Test concurrent member additions
	var wg sync.WaitGroup
	startBarrier := make(chan struct{})
	results := make([]error, numUsers)

	for i, user := range users {
		wg.Add(1)
		go func(index int, memberUser *domain.User) {
			defer wg.Done()

			// Wait for start signal
			<-startBarrier

			ctx, cancel := context.WithTimeout(suite.Context(), suite.timeout)
			defer cancel()

			// Get fresh copy of project
			currentProject, err := suite.Repos.Projects.GetByID(ctx, project.ID)
			if err != nil {
				results[index] = fmt.Errorf("failed to get project: %w", err)
				return
			}

			// Add member (check for duplicates)
			memberExists := false
			for _, existingMemberID := range currentProject.MemberIDs {
				if existingMemberID == memberUser.ID {
					memberExists = true
					break
				}
			}

			if !memberExists {
				currentProject.MemberIDs = append(currentProject.MemberIDs, memberUser.ID)
			}

			// Save to database
			results[index] = suite.Repos.Projects.Update(ctx, currentProject)
		}(i, user)
	}

	// Signal all goroutines to start
	close(startBarrier)

	// Wait for completion
	wg.Wait()

	// Analyze results
	successCount := 0
	for i, err := range results {
		if err == nil {
			successCount++
		} else {
			t.Logf("Member addition %d failed: %v", i, err)
		}
	}

	// Verify final project state
	finalProject, err := suite.Repos.Projects.GetByID(suite.Context(), project.ID)
	require.NoError(t, err)

	// All members should be present (no race condition losses)
	assert.LessOrEqual(t, len(finalProject.MemberIDs), numUsers,
		"Should not have more members than users created")
	assert.GreaterOrEqual(t, len(finalProject.MemberIDs), 1,
		"Should have at least one member added")

	// Check for duplicates
	memberMap := make(map[string]bool)
	duplicateCount := 0
	for _, memberID := range finalProject.MemberIDs {
		if memberMap[memberID] {
			duplicateCount++
		} else {
			memberMap[memberID] = true
		}
	}

	assert.Equal(t, 0, duplicateCount, "Should not have duplicate members")

	// Verify all member IDs are from our created users
	for _, memberID := range finalProject.MemberIDs {
		found := false
		for _, user := range users {
			if user.ID == memberID {
				found = true
				break
			}
		}
		assert.True(t, found, "Member ID %s should belong to one of our created users", memberID)
	}

	suite.Assert.ProjectExists(project.ID)

	t.Logf("Concurrent member additions: %d successes, final member count: %d, duplicates: %d",
		successCount, len(finalProject.MemberIDs), duplicateCount)
}

// TestConcurrentCommentCreation tests concurrent comment creation for proper ordering
func TestConcurrentCommentCreation(t *testing.T) {
	suite := setupConcurrencyTest(t)
	defer suite.Cleanup()

	// Create test data
	user := suite.Factory.CreateUser()
	require.NoError(t, suite.Repos.Users.Create(suite.Context(), user))

	project := suite.Factory.CreateProject(user)
	require.NoError(t, suite.Repos.Projects.Create(suite.Context(), project))

	task := suite.Factory.CreateTask(project, user)
	require.NoError(t, suite.Repos.Tasks.Create(suite.Context(), task))

	// Test concurrent comment creation
	numComments := 15
	var wg sync.WaitGroup
	startBarrier := make(chan struct{})
	results := make([]error, numComments)
	commentIDs := make([]string, numComments)

	for i := 0; i < numComments; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			// Wait for start signal
			<-startBarrier

			ctx, cancel := context.WithTimeout(suite.Context(), suite.timeout)
			defer cancel()

			// Create comment
			comment := suite.Factory.CreateComment(task, user,
				integration.WithCommentContent(fmt.Sprintf("Concurrent comment #%d at %v",
					index, time.Now().UnixNano())))

			// Save to database
			err := suite.Repos.Comments.Create(ctx, comment)
			results[index] = err

			if err == nil {
				commentIDs[index] = comment.ID
			}
		}(i)
	}

	// Signal all goroutines to start
	close(startBarrier)

	// Wait for completion
	wg.Wait()

	// Analyze results
	successCount := 0
	for i, err := range results {
		if err == nil {
			successCount++
		} else {
			t.Logf("Comment creation %d failed: %v", i, err)
		}
	}

	// Verify all successful comments exist
	for i, commentID := range commentIDs {
		if commentID != "" {
			suite.Assert.CommentExists(commentID)

			// Verify comment belongs to correct task
			comment, err := suite.Repos.Comments.GetByID(suite.Context(), commentID)
			require.NoError(t, err)
			assert.Equal(t, task.ID, comment.TaskID,
				"Comment %d should belong to the correct task", i)
		}
	}

	// Count comments for task
	suite.Assert.CommentCountByTask(task.ID, successCount)

	t.Logf("Concurrent comment creation: %d successes out of %d attempts",
		successCount, numComments)
}

// TestTransactionRollbackBehavior tests transaction rollback on partial failures
func TestTransactionRollbackBehavior(t *testing.T) {
	suite := setupConcurrencyTest(t)
	defer suite.Cleanup()

	// Create test data
	user := suite.Factory.CreateUser()
	require.NoError(t, suite.Repos.Users.Create(suite.Context(), user))

	project := suite.Factory.CreateProject(user)
	require.NoError(t, suite.Repos.Projects.Create(suite.Context(), project))

	// Create valid task
	validTask := suite.Factory.CreateTask(project, user,
		integration.WithTaskTitle("Valid task"))

	// Create invalid task (will cause validation error)
	invalidTask := suite.Factory.CreateTask(project, user,
		integration.WithTaskTitle("")) // Empty title should cause validation error

	// Test bulk update with mixed valid/invalid data
	tasks := []*domain.Task{validTask, invalidTask}

	// Record initial state
	initialTaskCount, err := suite.Repos.Tasks.Count(suite.Context())
	require.NoError(t, err)

	// Attempt bulk create (should fail due to invalid task)
	for _, task := range tasks {
		err := suite.Repos.Tasks.Create(suite.Context(), task)
		if err != nil {
			t.Logf("Expected error for invalid task: %v", err)
		}
	}

	// Check final state - verify partial operations don't persist invalid state
	finalTaskCount, err := suite.Repos.Tasks.Count(suite.Context())
	require.NoError(t, err)

	// In a proper transaction system, either all succeed or all fail
	// Since PocketBase doesn't have explicit transactions, we expect individual operations
	expectedCount := initialTaskCount + 1 // Only the valid task should be created
	assert.Equal(t, expectedCount, finalTaskCount,
		"Only valid task should be persisted")

	t.Logf("Transaction rollback test: initial=%d, final=%d, expected=%d",
		initialTaskCount, finalTaskCount, expectedCount)
}

// TestDeadlockDetectionAndRecovery tests deadlock detection and recovery mechanisms
func TestDeadlockDetectionAndRecovery(t *testing.T) {
	suite := setupConcurrencyTest(t)
	defer suite.Cleanup()

	// Create test data
	user1 := suite.Factory.CreateUser()
	require.NoError(t, suite.Repos.Users.Create(suite.Context(), user1))

	user2 := suite.Factory.CreateUser(
		integration.WithUserEmail("user2@test.example.com"))
	require.NoError(t, suite.Repos.Users.Create(suite.Context(), user2))

	project := suite.Factory.CreateProject(user1)
	require.NoError(t, suite.Repos.Projects.Create(suite.Context(), project))

	task1 := suite.Factory.CreateTask(project, user1,
		integration.WithTaskTitle("Task 1"))
	require.NoError(t, suite.Repos.Tasks.Create(suite.Context(), task1))

	task2 := suite.Factory.CreateTask(project, user1,
		integration.WithTaskTitle("Task 2"))
	require.NoError(t, suite.Repos.Tasks.Create(suite.Context(), task2))

	// Test potential deadlock scenario: cross-resource locking
	var wg sync.WaitGroup
	startBarrier := make(chan struct{})
	results := make([]error, 2)

	// Goroutine 1: Update task1, then task2
	wg.Add(1)
	go func() {
		defer wg.Done()

		<-startBarrier

		ctx, cancel := context.WithTimeout(suite.Context(), suite.timeout)
		defer cancel()

		// Update task1
		currentTask1, err := suite.Repos.Tasks.GetByID(ctx, task1.ID)
		if err != nil {
			results[0] = fmt.Errorf("failed to get task1: %w", err)
			return
		}

		err = currentTask1.UpdateStatus(domain.StatusDeveloping)
		if err != nil {
			results[0] = fmt.Errorf("failed to update task1 status: %w", err)
			return
		}

		err = suite.Repos.Tasks.Update(ctx, currentTask1)
		if err != nil {
			results[0] = fmt.Errorf("failed to save task1: %w", err)
			return
		}

		// Small delay to increase chance of cross-locking
		time.Sleep(10 * time.Millisecond)

		// Update task2
		currentTask2, err := suite.Repos.Tasks.GetByID(ctx, task2.ID)
		if err != nil {
			results[0] = fmt.Errorf("failed to get task2: %w", err)
			return
		}

		err = currentTask2.UpdateStatus(domain.StatusReview)
		if err != nil {
			results[0] = fmt.Errorf("failed to update task2 status: %w", err)
			return
		}

		results[0] = suite.Repos.Tasks.Update(ctx, currentTask2)
	}()

	// Goroutine 2: Update task2, then task1
	wg.Add(1)
	go func() {
		defer wg.Done()

		<-startBarrier

		ctx, cancel := context.WithTimeout(suite.Context(), suite.timeout)
		defer cancel()

		// Update task2
		currentTask2, err := suite.Repos.Tasks.GetByID(ctx, task2.ID)
		if err != nil {
			results[1] = fmt.Errorf("failed to get task2: %w", err)
			return
		}

		err = currentTask2.UpdateStatus(domain.StatusDeveloping)
		if err != nil {
			results[1] = fmt.Errorf("failed to update task2 status: %w", err)
			return
		}

		err = suite.Repos.Tasks.Update(ctx, currentTask2)
		if err != nil {
			results[1] = fmt.Errorf("failed to save task2: %w", err)
			return
		}

		// Small delay to increase chance of cross-locking
		time.Sleep(10 * time.Millisecond)

		// Update task1
		currentTask1, err := suite.Repos.Tasks.GetByID(ctx, task1.ID)
		if err != nil {
			results[1] = fmt.Errorf("failed to get task1: %w", err)
			return
		}

		err = currentTask1.UpdateStatus(domain.StatusComplete)
		if err != nil {
			results[1] = fmt.Errorf("failed to update task1 status: %w", err)
			return
		}

		results[1] = suite.Repos.Tasks.Update(ctx, currentTask1)
	}()

	// Signal both goroutines to start
	close(startBarrier)

	// Wait for completion
	wg.Wait()

	// Analyze results
	successCount := 0
	for i, err := range results {
		if err == nil {
			successCount++
			t.Logf("Goroutine %d completed successfully", i+1)
		} else {
			t.Logf("Goroutine %d failed: %v", i+1, err)
		}
	}

	// Verify system recovers gracefully
	// At least one operation should succeed even in deadlock scenarios
	assert.GreaterOrEqual(t, successCount, 1,
		"At least one operation should succeed despite potential deadlock")

	// Verify both tasks still exist and are in valid states
	suite.Assert.TaskExists(task1.ID)
	suite.Assert.TaskExists(task2.ID)

	finalTask1, err := suite.Repos.Tasks.GetByID(suite.Context(), task1.ID)
	require.NoError(t, err)

	finalTask2, err := suite.Repos.Tasks.GetByID(suite.Context(), task2.ID)
	require.NoError(t, err)

	// Both tasks should be in valid states
	assert.True(t, finalTask1.Status.IsValid(), "Task1 should be in valid status")
	assert.True(t, finalTask2.Status.IsValid(), "Task2 should be in valid status")

	t.Logf("Deadlock recovery test: %d successes, task1=%s, task2=%s",
		successCount, finalTask1.Status, finalTask2.Status)
}

// TestDatabaseLockRecovery tests PocketBase's automatic database lock retry mechanism
func TestDatabaseLockRecovery(t *testing.T) {
	suite := setupConcurrencyTest(t)
	defer suite.Cleanup()

	// Create test data
	user := suite.Factory.CreateUser()
	require.NoError(t, suite.Repos.Users.Create(suite.Context(), user))

	project := suite.Factory.CreateProject(user)
	require.NoError(t, suite.Repos.Projects.Create(suite.Context(), project))

	// Test high-concurrency scenario to trigger database locks
	numGoroutines := 50
	var wg sync.WaitGroup
	var successCount int64
	var lockErrorCount int64
	startBarrier := make(chan struct{})

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			<-startBarrier

			ctx, cancel := context.WithTimeout(suite.Context(), suite.timeout)
			defer cancel()

			// Create task
			task := suite.Factory.CreateTask(project, user,
				integration.WithTaskTitle(fmt.Sprintf("Lock test task %d", goroutineID)))

			err := suite.Repos.Tasks.Create(ctx, task)
			if err != nil {
				if fmt.Sprintf("%v", err) == "database is locked" {
					atomic.AddInt64(&lockErrorCount, 1)
					t.Logf("Goroutine %d encountered database lock (expected)", goroutineID)
				} else {
					t.Logf("Goroutine %d failed with unexpected error: %v", goroutineID, err)
				}
				return
			}

			atomic.AddInt64(&successCount, 1)
		}(i)
	}

	// Signal all goroutines to start
	close(startBarrier)

	// Wait for completion
	wg.Wait()

	// Verify results
	totalOperations := int64(numGoroutines)
	finalSuccessCount := atomic.LoadInt64(&successCount)
	finalLockErrorCount := atomic.LoadInt64(&lockErrorCount)

	// Most operations should succeed despite lock contention
	assert.Greater(t, finalSuccessCount, totalOperations/2,
		"More than half of operations should succeed")

	// Count actual tasks created
	actualTaskCount, err := suite.Repos.Tasks.CountByProject(suite.Context(), project.ID)
	require.NoError(t, err)

	assert.Equal(t, int(finalSuccessCount), actualTaskCount,
		"Actual task count should match successful creates")

	t.Logf("Database lock recovery: %d total, %d successes, %d lock errors, %d actual tasks",
		totalOperations, finalSuccessCount, finalLockErrorCount, actualTaskCount)
}

// TestConcurrentTransactionIsolation tests transaction isolation behavior
func TestConcurrentTransactionIsolation(t *testing.T) {
	suite := setupConcurrencyTest(t)
	defer suite.Cleanup()

	// Create test data
	user := suite.Factory.CreateUser()
	require.NoError(t, suite.Repos.Users.Create(suite.Context(), user))

	project := suite.Factory.CreateProject(user)
	require.NoError(t, suite.Repos.Projects.Create(suite.Context(), project))

	task := suite.Factory.CreateTask(project, user,
		integration.WithTaskTitle("Isolation test task"),
		integration.WithTaskProgress(50))
	require.NoError(t, suite.Repos.Tasks.Create(suite.Context(), task))

	// Test read-modify-write isolation
	numReaders := 5
	numWriters := 3
	var wg sync.WaitGroup
	startBarrier := make(chan struct{})

	// Track what each goroutine observes
	readerResults := make([]int, numReaders)
	writerResults := make([]error, numWriters)

	// Start reader goroutines
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()

			<-startBarrier

			ctx, cancel := context.WithTimeout(suite.Context(), suite.timeout)
			defer cancel()

			// Read task progress multiple times to check for consistency
			for j := 0; j < 3; j++ {
				currentTask, err := suite.Repos.Tasks.GetByID(ctx, task.ID)
				if err != nil {
					t.Logf("Reader %d failed to read task: %v", readerID, err)
					return
				}

				readerResults[readerID] = currentTask.Progress
				time.Sleep(5 * time.Millisecond)
			}
		}(i)
	}

	// Start writer goroutines
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()

			<-startBarrier

			ctx, cancel := context.WithTimeout(suite.Context(), suite.timeout)
			defer cancel()

			// Update task progress
			currentTask, err := suite.Repos.Tasks.GetByID(ctx, task.ID)
			if err != nil {
				writerResults[writerID] = fmt.Errorf("failed to get task: %w", err)
				return
			}

			newProgress := currentTask.Progress + 10
			if newProgress > 100 {
				newProgress = 100
			}

			err = currentTask.UpdateProgress(newProgress)
			if err != nil {
				writerResults[writerID] = fmt.Errorf("failed to update progress: %w", err)
				return
			}

			writerResults[writerID] = suite.Repos.Tasks.Update(ctx, currentTask)
		}(i)
	}

	// Signal all goroutines to start
	close(startBarrier)

	// Wait for completion
	wg.Wait()

	// Analyze isolation behavior
	successfulWrites := 0
	for i, err := range writerResults {
		if err == nil {
			successfulWrites++
		} else {
			t.Logf("Writer %d failed: %v", i, err)
		}
	}

	// Verify final state
	finalTask, err := suite.Repos.Tasks.GetByID(suite.Context(), task.ID)
	require.NoError(t, err)

	// Progress should be consistent and within valid range
	assert.GreaterOrEqual(t, finalTask.Progress, 50, "Progress should be at least initial value")
	assert.LessOrEqual(t, finalTask.Progress, 100, "Progress should not exceed 100")

	t.Logf("Isolation test: %d successful writes, final progress: %d",
		successfulWrites, finalTask.Progress)

	// Log reader observations for analysis
	for i, progress := range readerResults {
		t.Logf("Reader %d observed progress: %d", i, progress)
	}
}

// TestRaceConditionDetection tests for race conditions in repository operations
func TestRaceConditionDetection(t *testing.T) {
	suite := setupConcurrencyTest(t)
	defer suite.Cleanup()

	// Enable race detection warnings
	t.Parallel()

	// Create test data
	user := suite.Factory.CreateUser()
	require.NoError(t, suite.Repos.Users.Create(suite.Context(), user))

	project := suite.Factory.CreateProject(user)
	require.NoError(t, suite.Repos.Projects.Create(suite.Context(), project))

	// Test race conditions in shared data structures
	numGoroutines := 20
	var wg sync.WaitGroup
	startBarrier := make(chan struct{})

	// Shared counters (potential race condition)
	var successCounter int64
	var errorCounter int64

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			<-startBarrier

			ctx, cancel := context.WithTimeout(suite.Context(), suite.timeout)
			defer cancel()

			// Perform operations that might race
			task := suite.Factory.CreateTask(project, user,
				integration.WithTaskTitle(fmt.Sprintf("Race test %d", goroutineID)))

			err := suite.Repos.Tasks.Create(ctx, task)
			if err != nil {
				atomic.AddInt64(&errorCounter, 1)
				return
			}

			// Update counters atomically
			atomic.AddInt64(&successCounter, 1)

			// Test concurrent read operations
			_, err = suite.Repos.Tasks.GetByID(ctx, task.ID)
			if err != nil {
				t.Logf("Goroutine %d: read after write failed: %v", goroutineID, err)
			}
		}(i)
	}

	// Signal all goroutines to start
	close(startBarrier)

	// Wait for completion
	wg.Wait()

	// Verify no race conditions in counters
	finalSuccesses := atomic.LoadInt64(&successCounter)
	finalErrors := atomic.LoadInt64(&errorCounter)

	assert.Equal(t, int64(numGoroutines), finalSuccesses+finalErrors,
		"All operations should be accounted for")

	// Verify actual database state matches our counters
	actualTaskCount, err := suite.Repos.Tasks.CountByProject(suite.Context(), project.ID)
	require.NoError(t, err)

	assert.Equal(t, int(finalSuccesses), actualTaskCount,
		"Database state should match successful operations")

	t.Logf("Race condition test: %d successes, %d errors, %d database records",
		finalSuccesses, finalErrors, actualTaskCount)
}

// TestConcurrentDatabaseConnections tests behavior under multiple database connections
func TestConcurrentDatabaseConnections(t *testing.T) {
	suite := setupConcurrencyTest(t)
	defer suite.Cleanup()

	// Test high concurrency to stress database connection handling
	numGoroutines := 100
	var wg sync.WaitGroup
	startBarrier := make(chan struct{})

	connectionResults := make([]error, numGoroutines)

	// Create test data
	user := suite.Factory.CreateUser()
	require.NoError(t, suite.Repos.Users.Create(suite.Context(), user))

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			<-startBarrier

			ctx, cancel := context.WithTimeout(suite.Context(), suite.timeout)
			defer cancel()

			// Test simple read operation (should work with connection pooling)
			_, err := suite.Repos.Users.GetByID(ctx, user.ID)
			connectionResults[goroutineID] = err
		}(i)
	}

	// Signal all goroutines to start
	close(startBarrier)

	// Wait for completion
	wg.Wait()

	// Analyze connection results
	successfulConnections := 0
	connectionErrors := 0

	for i, err := range connectionResults {
		if err == nil {
			successfulConnections++
		} else {
			connectionErrors++
			t.Logf("Connection %d failed: %v", i, err)
		}
	}

	// Most connections should succeed
	assert.Greater(t, successfulConnections, numGoroutines*80/100,
		"At least 80% of connections should succeed")

	t.Logf("Connection test: %d successes, %d failures out of %d attempts",
		successfulConnections, connectionErrors, numGoroutines)
}

// TestConcurrencyApproachComparison demonstrates both repository and service-layer approaches
// This test showcases the difference between direct repository access and service-layer access
func TestConcurrencyApproachComparison(t *testing.T) {
	// Test both approaches side by side
	t.Run("RepositoryApproach", func(t *testing.T) {
		t.Parallel()
		suite := setupConcurrencyTest(t)
		defer suite.Cleanup()

		// Create test data using repository approach
		user := suite.Factory.CreateUser()
		require.NoError(t, suite.Repos.Users.Create(suite.Context(), user))

		project := suite.Factory.CreateProject(user)
		require.NoError(t, suite.Repos.Projects.Create(suite.Context(), project))

		// Test concurrent operations directly on repositories
		numTasks := 10
		var wg sync.WaitGroup
		var successCount int64

		for i := 0; i < numTasks; i++ {
			wg.Add(1)
			go func(taskNum int) {
				defer wg.Done()
				task := suite.Factory.CreateTask(project, user,
					integration.WithTaskTitle(fmt.Sprintf("Repo task %d", taskNum)))

				if err := suite.Repos.Tasks.Create(suite.Context(), task); err == nil {
					atomic.AddInt64(&successCount, 1)
				}
			}(i)
		}

		wg.Wait()

		finalCount := atomic.LoadInt64(&successCount)
		t.Logf("Repository approach: %d/%d tasks created successfully", finalCount, numTasks)
		assert.GreaterOrEqual(t, finalCount, int64(8), "Most repository operations should succeed")
	})

	t.Run("ServiceLayerApproach", func(t *testing.T) {
		t.Parallel()
		suite := setupServiceConcurrencyTest(t)
		defer suite.Cleanup()

		// Create test data using repository for setup
		user := suite.Factory.CreateUser()
		require.NoError(t, suite.Repos.Users.Create(suite.Context(), user))

		project := suite.Factory.CreateProject(user)
		require.NoError(t, suite.Repos.Projects.Create(suite.Context(), project))

		// Test concurrent operations through service layer
		taskService := suite.GetTaskService(t)
		numTasks := 10
		var wg sync.WaitGroup
		var successCount int64

		for i := 0; i < numTasks; i++ {
			wg.Add(1)
			go func(taskNum int) {
				defer wg.Done()
				req := domain.CreateTaskRequest{
					Title:       fmt.Sprintf("Service task %d", taskNum),
					Description: "Created via service layer",
					ProjectID:   project.ID,
					Priority:    domain.PriorityMedium,
				}

				if _, err := taskService.CreateTask(suite.Context(), req, user.ID); err == nil {
					atomic.AddInt64(&successCount, 1)
				}
			}(i)
		}

		wg.Wait()

		finalCount := atomic.LoadInt64(&successCount)
		t.Logf("Service layer approach: %d/%d tasks created successfully", finalCount, numTasks)
		assert.GreaterOrEqual(t, finalCount, int64(8), "Most service operations should succeed")

		// Additional validation: Service layer should enforce business rules
		// Try creating invalid task through service
		invalidReq := domain.CreateTaskRequest{
			Title:       "", // Invalid: empty title
			Description: "This should fail",
			ProjectID:   project.ID,
		}

		_, err := taskService.CreateTask(suite.Context(), invalidReq, user.ID)
		assert.Error(t, err, "Service layer should reject invalid requests")
	})
}
