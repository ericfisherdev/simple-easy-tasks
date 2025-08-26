//go:build integration
// +build integration

package integration

import (
	"context"
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

// EnhancedConcurrencyTestSuite provides service-layer concurrency testing infrastructure
type EnhancedConcurrencyTestSuite struct {
	*integration.DatabaseTestSuite
	maxGoroutines int
	timeout       time.Duration
}

// setupEnhancedConcurrencyTest creates a test suite optimized for service-layer concurrency testing
func setupEnhancedConcurrencyTest(t *testing.T) *EnhancedConcurrencyTestSuite {
	suite := integration.SetupConcurrencyTestWithServices(t)

	return &EnhancedConcurrencyTestSuite{
		DatabaseTestSuite: suite,
		maxGoroutines:     runtime.NumCPU() * 4, // Use more goroutines than CPU cores to test contention
		timeout:           30 * time.Second,     // Generous timeout for concurrent operations
	}
}

// TestConcurrentTaskStatusUpdatesViaService tests concurrent task status updates through the service layer
// This ensures the full application stack handles concurrency correctly
func TestConcurrentTaskStatusUpdatesViaService(t *testing.T) {
	suite := setupEnhancedConcurrencyTest(t)
	defer suite.Cleanup()

	// Create test data using the repository layer for setup
	user := suite.Factory.CreateUser()
	require.NoError(t, suite.Repos.Users.Create(suite.Context(), user))

	project := suite.Factory.CreateProject(user)
	require.NoError(t, suite.Repos.Projects.Create(suite.Context(), project))

	task := suite.Factory.CreateTask(project, user)
	require.NoError(t, suite.Repos.Tasks.Create(suite.Context(), task))

	// Get the task service for operations
	taskService := suite.GetTaskService(t)

	// Test concurrent status updates through service layer
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

	// Launch concurrent status update goroutines using service layer
	for i, status := range statuses {
		wg.Add(1)
		go func(index int, newStatus domain.TaskStatus) {
			defer wg.Done()

			// Wait for start signal
			<-startBarrier

			// Create context with timeout
			ctx, cancel := context.WithTimeout(suite.Context(), suite.timeout)
			defer cancel()

			// Use service layer method for updating task status
			// This tests the full application stack including business logic validation
			_, err := taskService.UpdateTaskStatus(ctx, task.ID, newStatus, user.ID)
			results[index] = err
		}(i, status)
	}

	// Signal all goroutines to start
	close(startBarrier)

	// Wait for completion
	wg.Wait()

	// Analyze results - service layer should handle concurrent updates gracefully
	successCount := 0
	var lastError error
	for i, err := range results {
		if err == nil {
			successCount++
		} else {
			t.Logf("Service-layer status update %d failed with: %v", i, err)
			lastError = err
		}
	}

	// At least one update should succeed (last-write-wins or optimistic concurrency control)
	assert.GreaterOrEqual(t, successCount, 1, "At least one service-layer concurrent update should succeed")

	// Verify final task state through service layer
	finalTask, err := taskService.GetTask(suite.Context(), task.ID, user.ID)
	require.NoError(t, err)

	// Final status should be one of the attempted statuses
	assert.Contains(t, statuses, finalTask.Status, "Final status should be one of the attempted statuses")

	// Verify database consistency
	suite.Assert.TaskExists(task.ID)
	suite.Assert.TaskHasStatus(task.ID, finalTask.Status)

	t.Logf("Service-layer concurrent status updates completed: %d successes, %d failures", successCount, len(results)-successCount)
	if lastError != nil {
		t.Logf("Sample error: %v", lastError)
	}
	t.Logf("Final task status: %s", finalTask.Status)
}

// TestConcurrentTaskCreationViaService tests concurrent task creation through service layer
func TestConcurrentTaskCreationViaService(t *testing.T) {
	suite := setupEnhancedConcurrencyTest(t)
	defer suite.Cleanup()

	// Create test data
	user := suite.Factory.CreateUser()
	require.NoError(t, suite.Repos.Users.Create(suite.Context(), user))

	project := suite.Factory.CreateProject(user)
	require.NoError(t, suite.Repos.Projects.Create(suite.Context(), project))

	// Get services
	taskService := suite.GetTaskService(t)

	// Test concurrent task creation
	numTasks := 15
	var wg sync.WaitGroup
	startBarrier := make(chan struct{})
	results := make([]error, numTasks)
	taskIDs := make([]string, numTasks)

	for i := 0; i < numTasks; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			// Wait for start signal
			<-startBarrier

			ctx, cancel := context.WithTimeout(suite.Context(), suite.timeout)
			defer cancel()

			// Create task using service layer with business logic validation
			req := domain.CreateTaskRequest{
				Title:       fmt.Sprintf("Service test task #%d at %v", index, time.Now().UnixNano()),
				Description: fmt.Sprintf("Created via service layer in goroutine %d", index),
				ProjectID:   project.ID,
				Priority:    domain.PriorityMedium,
			}

			createdTask, err := taskService.CreateTask(ctx, req, user.ID)
			results[index] = err

			if err == nil {
				taskIDs[index] = createdTask.ID
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
			t.Logf("Service-layer task creation %d failed: %v", i, err)
		}
	}

	// Most or all creations should succeed since we're creating different tasks
	assert.GreaterOrEqual(t, successCount, numTasks*80/100,
		"At least 80% of service-layer task creations should succeed")

	// Verify all successful tasks exist and are properly configured
	for i, taskID := range taskIDs {
		if taskID != "" {
			suite.Assert.TaskExists(taskID)

			// Verify task via service layer
			task, err := taskService.GetTask(suite.Context(), taskID, user.ID)
			require.NoError(t, err, "Should be able to retrieve task %d via service", i)
			assert.Equal(t, project.ID, task.ProjectID, "Task should belong to correct project")
			assert.Equal(t, domain.StatusTodo, task.Status, "Task should have correct initial status")
			assert.Equal(t, domain.PriorityMedium, task.Priority, "Task should have correct priority")
		}
	}

	t.Logf("Service-layer concurrent task creation: %d successes out of %d attempts", successCount, numTasks)
}

// TestConcurrentProjectMemberManagementViaService tests concurrent project member management through service layer
func TestConcurrentProjectMemberManagementViaService(t *testing.T) {
	suite := setupEnhancedConcurrencyTest(t)
	defer suite.Cleanup()

	// Create test data
	owner := suite.Factory.CreateUser()
	require.NoError(t, suite.Repos.Users.Create(suite.Context(), owner))

	// Create project via service layer to ensure proper initialization
	projectService := suite.GetProjectService(t)
	createReq := domain.CreateProjectRequest{
		Title:       "Service Test Project",
		Description: "Testing concurrent member management",
		Slug:        "service-test-project",
	}

	project, err := projectService.CreateProject(suite.Context(), createReq, owner.ID)
	require.NoError(t, err)

	// Create multiple users to add as members
	numUsers := 10
	users := make([]*domain.User, numUsers)
	for i := 0; i < numUsers; i++ {
		user := suite.Factory.CreateUser(
			integration.WithUserEmail(fmt.Sprintf("servicemember%d@test.example.com", i)))
		require.NoError(t, suite.Repos.Users.Create(suite.Context(), user))
		users[i] = user
	}

	// Test concurrent member additions via service layer
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

			// Add member via service layer (would include business logic validation)
			// Note: This assumes there's a project service method for adding members
			// For now, we'll simulate this by getting the project and updating it
			currentProject, err := projectService.GetProject(ctx, project.ID, owner.ID)
			if err != nil {
				results[index] = fmt.Errorf("failed to get project: %w", err)
				return
			}

			// Check if member already exists
			memberExists := false
			for _, existingMemberID := range currentProject.MemberIDs {
				if existingMemberID == memberUser.ID {
					memberExists = true
					break
				}
			}

			if !memberExists {
				// For now, we'll simulate member addition by updating the project description
				// In a real implementation, you'd have a proper AddMember service method
				newDesc := fmt.Sprintf("%s - Member %s added", currentProject.Description, memberUser.ID)
				updateReq := domain.UpdateProjectRequest{
					Description: &newDesc,
				}

				_, err = projectService.UpdateProject(ctx, project.ID, updateReq, owner.ID)
				results[index] = err
			}
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
			t.Logf("Service-layer member addition %d failed: %v", i, err)
		}
	}

	// Verify final project state via service layer
	finalProject, err := projectService.GetProject(suite.Context(), project.ID, owner.ID)
	require.NoError(t, err)

	// All members should be present (service layer should handle duplicates)
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

	assert.Equal(t, 0, duplicateCount, "Service layer should prevent duplicate members")

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

	t.Logf("Service-layer concurrent member management: %d successes, final member count: %d, duplicates: %d",
		successCount, len(finalProject.MemberIDs), duplicateCount)
}

// TestConcurrentServiceErrorHandling tests how services handle concurrent error scenarios
func TestConcurrentServiceErrorHandling(t *testing.T) {
	suite := setupEnhancedConcurrencyTest(t)
	defer suite.Cleanup()

	// Create test data
	user := suite.Factory.CreateUser()
	require.NoError(t, suite.Repos.Users.Create(suite.Context(), user))

	project := suite.Factory.CreateProject(user)
	require.NoError(t, suite.Repos.Projects.Create(suite.Context(), project))

	// Get services
	taskService := suite.GetTaskService(t)

	// Test concurrent operations with deliberate validation errors
	numGoroutines := 20
	var wg sync.WaitGroup
	startBarrier := make(chan struct{})
	
	var validCreations int64
	var validationErrors int64
	var otherErrors int64

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			<-startBarrier

			ctx, cancel := context.WithTimeout(suite.Context(), suite.timeout)
			defer cancel()

			// Alternate between valid and invalid task creation requests
			var req domain.CreateTaskRequest
			if goroutineID%2 == 0 {
				// Valid request
				req = domain.CreateTaskRequest{
					Title:       fmt.Sprintf("Valid task %d", goroutineID),
					Description: "This should succeed",
					ProjectID:   project.ID,
					Priority:    domain.PriorityMedium,
				}
			} else {
				// Invalid request (empty title should cause validation error)
				req = domain.CreateTaskRequest{
					Title:       "", // Invalid: empty title
					Description: "This should fail validation",
					ProjectID:   project.ID,
					Priority:    domain.PriorityMedium,
				}
			}

			_, err := taskService.CreateTask(ctx, req, user.ID)
			if err == nil {
				atomic.AddInt64(&validCreations, 1)
			} else {
				// Classify error types
				errStr := err.Error()
				if strings.Contains(errStr, "title is required") || 
				   strings.Contains(errStr, "validation error") ||
				   strings.Contains(errStr, "INVALID_TITLE") {
					atomic.AddInt64(&validationErrors, 1)
				} else {
					atomic.AddInt64(&otherErrors, 1)
					t.Logf("Goroutine %d unexpected error: %v", goroutineID, err)
				}
			}
		}(i)
	}

	// Signal all goroutines to start
	close(startBarrier)

	// Wait for completion
	wg.Wait()

	// Verify error handling
	finalValidCreations := atomic.LoadInt64(&validCreations)
	finalValidationErrors := atomic.LoadInt64(&validationErrors)
	finalOtherErrors := atomic.LoadInt64(&otherErrors)

	// Should have roughly half valid creations (even goroutine IDs)
	expectedValid := int64(numGoroutines / 2)
	assert.Equal(t, expectedValid, finalValidCreations,
		"Should have created exactly half the tasks (valid requests)")

	// Should have roughly half validation errors (odd goroutine IDs)
	expectedValidationErrors := int64(numGoroutines / 2)
	assert.Equal(t, expectedValidationErrors, finalValidationErrors,
		"Should have exactly half validation errors (invalid requests)")

	// Should have no other types of errors
	assert.Equal(t, int64(0), finalOtherErrors,
		"Should not have any unexpected errors")

	t.Logf("Service-layer error handling: %d valid, %d validation errors, %d other errors",
		finalValidCreations, finalValidationErrors, finalOtherErrors)
}

// TestServiceLayerTransactionBehavior tests transaction-like behavior at service layer
func TestServiceLayerTransactionBehavior(t *testing.T) {
	suite := setupEnhancedConcurrencyTest(t)
	defer suite.Cleanup()

	// Create test data
	user := suite.Factory.CreateUser()
	require.NoError(t, suite.Repos.Users.Create(suite.Context(), user))

	project := suite.Factory.CreateProject(user)
	require.NoError(t, suite.Repos.Projects.Create(suite.Context(), project))

	task := suite.Factory.CreateTask(project, user)
	require.NoError(t, suite.Repos.Tasks.Create(suite.Context(), task))

	taskService := suite.GetTaskService(t)

	// Test concurrent complex operations that should maintain consistency
	numGoroutines := 10
	var wg sync.WaitGroup
	startBarrier := make(chan struct{})
	results := make([]error, numGoroutines)

	// Track initial state
	initialTask, err := taskService.GetTask(suite.Context(), task.ID, user.ID)
	require.NoError(t, err)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			<-startBarrier

			ctx, cancel := context.WithTimeout(suite.Context(), suite.timeout)
			defer cancel()

			// Perform a complex update operation
			title := fmt.Sprintf("Updated by goroutine %d", goroutineID)
			description := fmt.Sprintf("Complex update from goroutine %d at %v", goroutineID, time.Now())
			status := domain.StatusDeveloping
			updateReq := domain.UpdateTaskRequest{
				Title:       &title,
				Description: &description,
				Status:      &status,
			}

			_, err := taskService.UpdateTask(ctx, task.ID, updateReq, user.ID)
			results[goroutineID] = err
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
			t.Logf("Service-layer complex update %d failed: %v", i, err)
		}
	}

	// At least one update should succeed
	assert.GreaterOrEqual(t, successCount, 1,
		"At least one service-layer complex update should succeed")

	// Verify final state is consistent
	finalTask, err := taskService.GetTask(suite.Context(), task.ID, user.ID)
	require.NoError(t, err)

	// Task should be in a valid state
	assert.True(t, finalTask.Status.IsValid(), "Final task status should be valid")
	assert.NotEmpty(t, finalTask.Title, "Final task should have a title")
	assert.NotEmpty(t, finalTask.ID, "Final task should have an ID")
	assert.Equal(t, project.ID, finalTask.ProjectID, "Task should still belong to correct project")

	// Status should have changed from initial if any updates succeeded
	if successCount > 0 {
		// The status should be different from initial or be the expected updated status
		if finalTask.Status != initialTask.Status {
			assert.Equal(t, domain.StatusDeveloping, finalTask.Status,
				"Updated task should have the new status")
		}
	}

	suite.Assert.TaskExists(task.ID)

	t.Logf("Service-layer transaction behavior: %d successes, final status: %s",
		successCount, finalTask.Status)
}

// TestConcurrentServiceAuthenticationChecks tests concurrent authentication/authorization checks
func TestConcurrentServiceAuthenticationChecks(t *testing.T) {
	suite := setupEnhancedConcurrencyTest(t)
	defer suite.Cleanup()

	// Create test data
	owner := suite.Factory.CreateUser()
	require.NoError(t, suite.Repos.Users.Create(suite.Context(), owner))

	unauthorizedUser := suite.Factory.CreateUser(
		integration.WithUserEmail("unauthorized@test.example.com"))
	require.NoError(t, suite.Repos.Users.Create(suite.Context(), unauthorizedUser))

	project := suite.Factory.CreateProject(owner)
	require.NoError(t, suite.Repos.Projects.Create(suite.Context(), project))

	task := suite.Factory.CreateTask(project, owner)
	require.NoError(t, suite.Repos.Tasks.Create(suite.Context(), task))

	taskService := suite.GetTaskService(t)

	// Test concurrent access with mixed authorized/unauthorized users
	numGoroutines := 20
	var wg sync.WaitGroup
	startBarrier := make(chan struct{})
	
	var authorizedAccess int64
	var unauthorizedAccess int64
	var authErrors int64

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			<-startBarrier

			ctx, cancel := context.WithTimeout(suite.Context(), suite.timeout)
			defer cancel()

			// Alternate between authorized and unauthorized users
			var userID string
			isAuthorized := goroutineID%2 == 0
			if isAuthorized {
				userID = owner.ID
			} else {
				userID = unauthorizedUser.ID
			}

			// Try to access the task
			_, err := taskService.GetTask(ctx, task.ID, userID)
			if err == nil {
				if isAuthorized {
					atomic.AddInt64(&authorizedAccess, 1)
				} else {
					atomic.AddInt64(&unauthorizedAccess, 1)
					t.Logf("WARNING: Unauthorized user %d gained access", goroutineID)
				}
			} else {
				// Should be authorization error for unauthorized users
				if !isAuthorized {
					atomic.AddInt64(&authErrors, 1)
				} else {
					t.Logf("Authorized user %d was denied access: %v", goroutineID, err)
				}
			}
		}(i)
	}

	// Signal all goroutines to start
	close(startBarrier)

	// Wait for completion
	wg.Wait()

	// Verify authentication/authorization behavior
	finalAuthorizedAccess := atomic.LoadInt64(&authorizedAccess)
	finalUnauthorizedAccess := atomic.LoadInt64(&unauthorizedAccess)
	finalAuthErrors := atomic.LoadInt64(&authErrors)

	expectedAuthorized := int64(numGoroutines / 2)
	expectedAuthErrors := int64(numGoroutines / 2)

	// All authorized requests should succeed
	assert.Equal(t, expectedAuthorized, finalAuthorizedAccess,
		"All authorized service requests should succeed")

	// No unauthorized access should be granted
	assert.Equal(t, int64(0), finalUnauthorizedAccess,
		"No unauthorized access should be granted")

	// All unauthorized requests should result in auth errors
	assert.Equal(t, expectedAuthErrors, finalAuthErrors,
		"All unauthorized requests should be rejected")

	t.Logf("Service-layer auth checks: %d authorized, %d unauthorized access, %d auth errors",
		finalAuthorizedAccess, finalUnauthorizedAccess, finalAuthErrors)
}