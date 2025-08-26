//go:build integration
// +build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"simple-easy-tasks/internal/domain"
)

func TestTaskRepository_Integration(t *testing.T) {
	// Setup SHARED test suite for all subtests (performance optimization)
	suite := SetupDatabaseTest(t)
	defer suite.Cleanup()

	// Get repositories from suite
	taskRepo := suite.Repos.Tasks
	userRepo := suite.Repos.Users
	projectRepo := suite.Repos.Projects

	// Helper function to create test users and projects
	setupTestData := func(t *testing.T, suite *DatabaseTestSuite) (*domain.User, *domain.User, *domain.Project) {
		// Create users
		owner := suite.Factory.CreateUser(
			WithUserEmail("owner@task.test.com"),
			WithUserUsername("taskowner"),
			WithUserName("Task Owner"),
		)
		assignee := suite.Factory.CreateUser(
			WithUserEmail("assignee@task.test.com"),
			WithUserUsername("taskassignee"),
			WithUserName("Task Assignee"),
		)

		// Save users to database
		require.NoError(t, userRepo.Create(context.Background(), owner))
		require.NoError(t, userRepo.Create(context.Background(), assignee))

		// Create project
		project := suite.Factory.CreateProject(owner,
			WithProjectTitle("Task Integration Test Project"),
			WithProjectSlug("task-integration-test"),
		)
		require.NoError(t, projectRepo.Create(context.Background(), project))

		return owner, assignee, project
	}

	// ===========================================
	// CORE CRUD OPERATIONS TESTS
	// ===========================================

	t.Run("Create_ValidTask_Success", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, assignee, project := setupTestData(t, suite)

		// Create task with all fields populated
		task := suite.Factory.CreateTask(project, owner,
			WithTaskTitle("Integration Test Task"),
			WithTaskDescription("This is a comprehensive test task"),
			WithTaskStatus(domain.StatusTodo),
			WithTaskPriority(domain.PriorityHigh),
			WithTaskAssignee(assignee.ID),
			WithTaskDueDate(time.Now().Add(7*24*time.Hour)),
			WithTaskProgress(25),
		)

		// Set complex fields
		task.Tags = []string{"integration", "testing", "crud"}
		task.Dependencies = []string{}
		task.Attachments = []string{"file1.pdf", "screenshot.png"}

		customFields := map[string]interface{}{
			"complexity":      "medium",
			"estimated_hours": 8.5,
			"sprint":          "sprint-1",
		}
		customFieldsJSON, _ := json.Marshal(customFields)
		task.CustomFields = customFieldsJSON

		columnPosition := map[string]int{
			"todo":   1,
			"kanban": 5,
		}
		columnPosJSON, _ := json.Marshal(columnPosition)
		task.ColumnPosition = columnPosJSON

		// Act
		err := taskRepo.Create(context.Background(), task)

		// Assert
		require.NoError(t, err)
		assert.NotEmpty(t, task.ID)
		assert.False(t, task.CreatedAt.IsZero())
		assert.False(t, task.UpdatedAt.IsZero())

		// Verify the task can be retrieved and all fields match
		retrieved, err := taskRepo.GetByID(context.Background(), task.ID)
		require.NoError(t, err)
		assert.Equal(t, task.Title, retrieved.Title)
		assert.Equal(t, task.Description, retrieved.Description)
		assert.Equal(t, task.Status, retrieved.Status)
		assert.Equal(t, task.Priority, retrieved.Priority)
		assert.Equal(t, task.ProjectID, retrieved.ProjectID)
		assert.Equal(t, task.ReporterID, retrieved.ReporterID)
		assert.Equal(t, *task.AssigneeID, *retrieved.AssigneeID)
		assert.Equal(t, task.Progress, retrieved.Progress)
		assert.Equal(t, task.Tags, retrieved.Tags)
		assert.Equal(t, task.Attachments, retrieved.Attachments)
		assert.JSONEq(t, string(task.CustomFields), string(retrieved.CustomFields))
		assert.JSONEq(t, string(task.ColumnPosition), string(retrieved.ColumnPosition))
	})

	t.Run("Create_RequiredFieldsOnly_Success", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupTestData(t, suite)

		// Create minimal task with only required fields
		task := suite.Factory.CreateTask(project, owner,
			WithTaskTitle("Minimal Task"),
		)
		// Clear optional fields to test required-only scenario
		task.AssigneeID = nil
		task.DueDate = nil
		task.ParentTaskID = nil

		err := taskRepo.Create(context.Background(), task)

		require.NoError(t, err)
		assert.NotEmpty(t, task.ID)

		// Verify retrieval
		retrieved, err := taskRepo.GetByID(context.Background(), task.ID)
		require.NoError(t, err)
		assert.Equal(t, task.Title, retrieved.Title)
		assert.Nil(t, retrieved.AssigneeID)
		assert.Nil(t, retrieved.DueDate)
		assert.Nil(t, retrieved.ParentTaskID)
	})

	t.Run("Create_InvalidProjectID_ReturnsError", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupTestData(t, suite)

		// Create task with non-existent project ID
		task := suite.Factory.CreateTask(project, owner,
			WithTaskTitle("Invalid Project Task"),
		)
		task.ProjectID = "nonexistent123"

		err := taskRepo.Create(context.Background(), task)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to save task record")
	})

	t.Run("Create_InvalidReporterID_ReturnsError", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupTestData(t, suite)

		// Create task with non-existent reporter ID
		task := suite.Factory.CreateTask(project, owner,
			WithTaskTitle("Invalid Reporter Task"),
		)
		task.ReporterID = "nonexistent123"

		err := taskRepo.Create(context.Background(), task)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to save task record")
	})

	t.Run("Create_ValidationErrors_ReturnsError", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupTestData(t, suite)

		// Test empty title
		task := suite.Factory.CreateTask(project, owner)
		task.Title = ""

		err := taskRepo.Create(context.Background(), task)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "validation failed")

		// Test title too long
		task.Title = strings.Repeat("a", 201) // Exceeds 200 char limit
		err = taskRepo.Create(context.Background(), task)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "validation failed")

		// Test invalid progress
		task.Title = "Valid Title"
		task.Progress = 101 // Invalid progress > 100
		err = taskRepo.Create(context.Background(), task)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "validation failed")
	})

	t.Run("GetByID_ExistingTask_ReturnsTask", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupTestData(t, suite)

		// Create and save task
		task := suite.Factory.CreateTask(project, owner,
			WithTaskTitle("Get By ID Test"),
		)
		require.NoError(t, taskRepo.Create(context.Background(), task))

		// Act
		retrieved, err := taskRepo.GetByID(context.Background(), task.ID)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, task.ID, retrieved.ID)
		assert.Equal(t, task.Title, retrieved.Title)
	})

	t.Run("GetByID_NonExistentTask_ReturnsError", func(t *testing.T) {
		_, err := taskRepo.GetByID(context.Background(), "nonexistent123")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to find task by ID")
	})

	t.Run("GetByID_EmptyID_ReturnsError", func(t *testing.T) {
		_, err := taskRepo.GetByID(context.Background(), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "task ID cannot be empty")
	})

	t.Run("Update_ValidChanges_Success", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, assignee, project := setupTestData(t, suite)

		// Create and save task
		task := suite.Factory.CreateTask(project, owner,
			WithTaskTitle("Original Title"),
			WithTaskStatus(domain.StatusTodo),
		)
		require.NoError(t, taskRepo.Create(context.Background(), task))

		originalCreated := task.CreatedAt
		originalID := task.ID

		// Update task
		task.Title = "Updated Title"
		task.Description = "Updated description"
		task.Status = domain.StatusDeveloping
		task.Priority = domain.PriorityHigh
		task.AssigneeID = &assignee.ID
		task.Progress = 50

		err := taskRepo.Update(context.Background(), task)

		require.NoError(t, err)
		assert.Equal(t, originalID, task.ID)
		assert.Equal(t, originalCreated, task.CreatedAt)
		assert.True(t, task.UpdatedAt.After(originalCreated))

		// Verify changes persisted
		retrieved, err := taskRepo.GetByID(context.Background(), task.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Title", retrieved.Title)
		assert.Equal(t, "Updated description", retrieved.Description)
		assert.Equal(t, domain.StatusDeveloping, retrieved.Status)
		assert.Equal(t, domain.PriorityHigh, retrieved.Priority)
		assert.Equal(t, assignee.ID, *retrieved.AssigneeID)
		assert.Equal(t, 50, retrieved.Progress)
	})

	t.Run("Update_EmptyID_ReturnsError", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupTestData(t, suite)

		task := suite.Factory.CreateTask(project, owner)
		task.ID = ""

		err := taskRepo.Update(context.Background(), task)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "task ID cannot be empty for update")
	})

	t.Run("Update_NonExistentTask_ReturnsError", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupTestData(t, suite)

		task := suite.Factory.CreateTask(project, owner,
			WithTaskID("nonexistent123"),
		)

		err := taskRepo.Update(context.Background(), task)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to find task for update")
	})

	t.Run("Delete_ExistingTask_Success", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupTestData(t, suite)

		// Create and save task
		task := suite.Factory.CreateTask(project, owner,
			WithTaskTitle("Task to Delete"),
		)
		require.NoError(t, taskRepo.Create(context.Background(), task))

		// Verify task exists
		_, err := taskRepo.GetByID(context.Background(), task.ID)
		require.NoError(t, err)

		// Act
		err = taskRepo.Delete(context.Background(), task.ID)
		require.NoError(t, err)

		// Verify task no longer exists
		_, err = taskRepo.GetByID(context.Background(), task.ID)
		require.Error(t, err)
	})

	t.Run("Delete_NonExistentTask_ReturnsError", func(t *testing.T) {
		err := taskRepo.Delete(context.Background(), "nonexistent123")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to find task for deletion")
	})

	t.Run("Delete_EmptyID_ReturnsError", func(t *testing.T) {
		err := taskRepo.Delete(context.Background(), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "task ID cannot be empty")
	})

	// ===========================================
	// QUERY OPERATIONS TESTS
	// ===========================================

	t.Run("ListByProject_WithTasks_ReturnsOrderedTasks", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupTestData(t, suite)

		// Create multiple tasks with different positions
		tasks := make([]*domain.Task, 5)
		for i := 0; i < 5; i++ {
			task := suite.Factory.CreateTask(project, owner,
				WithTaskTitle(fmt.Sprintf("Task %d", i)),
			)
			task.Position = i + 1 // Position 1-5
			require.NoError(t, taskRepo.Create(context.Background(), task))
			tasks[i] = task
		}

		// Act - Get all tasks for project
		retrieved, err := taskRepo.ListByProject(context.Background(), project.ID, 0, 10)

		// Assert
		require.NoError(t, err)
		assert.Len(t, retrieved, 5)

		// Verify ordering by position (ascending)
		for i := 0; i < len(retrieved)-1; i++ {
			assert.True(t, retrieved[i].Position <= retrieved[i+1].Position,
				"Tasks should be ordered by position")
		}
	})

	t.Run("ListByProject_WithPagination_ReturnsCorrectPage", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupTestData(t, suite)

		// Create 7 tasks
		for i := 0; i < 7; i++ {
			task := suite.Factory.CreateTask(project, owner,
				WithTaskTitle(fmt.Sprintf("Paginated Task %d", i)),
			)
			task.Position = i
			require.NoError(t, taskRepo.Create(context.Background(), task))
		}

		// Get first page (3 items)
		page1, err := taskRepo.ListByProject(context.Background(), project.ID, 0, 3)
		require.NoError(t, err)
		assert.Len(t, page1, 3)

		// Get second page (3 items)
		page2, err := taskRepo.ListByProject(context.Background(), project.ID, 3, 3)
		require.NoError(t, err)
		assert.Len(t, page2, 3)

		// Get third page (1 item)
		page3, err := taskRepo.ListByProject(context.Background(), project.ID, 6, 3)
		require.NoError(t, err)
		assert.Len(t, page3, 1)

		// Verify no overlap
		allIDs := make(map[string]bool)
		for _, task := range append(append(page1, page2...), page3...) {
			assert.False(t, allIDs[task.ID], "No duplicate task IDs across pages")
			allIDs[task.ID] = true
		}
		assert.Len(t, allIDs, 7, "Should have all 7 unique tasks")
	})

	t.Run("ListByProject_EmptyProjectID_ReturnsError", func(t *testing.T) {
		_, err := taskRepo.ListByProject(context.Background(), "", 0, 10)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project ID cannot be empty")
	})

	t.Run("ListByAssignee_WithTasks_ReturnsTasks", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, assignee, project := setupTestData(t, suite)

		// Create tasks - some assigned, some not
		assignedTasks := make([]*domain.Task, 3)
		for i := 0; i < 3; i++ {
			task := suite.Factory.CreateTask(project, owner,
				WithTaskTitle(fmt.Sprintf("Assigned Task %d", i)),
				WithTaskAssignee(assignee.ID),
			)
			require.NoError(t, taskRepo.Create(context.Background(), task))
			assignedTasks[i] = task
		}

		// Create unassigned task
		unassignedTask := suite.Factory.CreateTask(project, owner,
			WithTaskTitle("Unassigned Task"),
		)
		require.NoError(t, taskRepo.Create(context.Background(), unassignedTask))

		// Act
		retrieved, err := taskRepo.ListByAssignee(context.Background(), assignee.ID, 0, 10)

		// Assert
		require.NoError(t, err)
		assert.Len(t, retrieved, 3, "Should only return assigned tasks")

		for _, task := range retrieved {
			assert.Equal(t, assignee.ID, *task.AssigneeID)
		}
	})

	t.Run("ListByAssignee_EmptyAssigneeID_ReturnsError", func(t *testing.T) {
		_, err := taskRepo.ListByAssignee(context.Background(), "", 0, 10)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "assignee ID cannot be empty")
	})

	t.Run("ListByStatus_WithTasks_ReturnsTasksWithStatus", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupTestData(t, suite)

		// Create tasks with different statuses
		statuses := []domain.TaskStatus{
			domain.StatusTodo,
			domain.StatusDeveloping,
			domain.StatusTodo,
			domain.StatusComplete,
		}

		for i, status := range statuses {
			task := suite.Factory.CreateTask(project, owner,
				WithTaskTitle(fmt.Sprintf("Status Task %d", i)),
				WithTaskStatus(status),
			)
			require.NoError(t, taskRepo.Create(context.Background(), task))
		}

		// Act - Get only TODO tasks
		todoTasks, err := taskRepo.ListByStatus(context.Background(), domain.StatusTodo, 0, 10)

		// Assert
		require.NoError(t, err)
		assert.Len(t, todoTasks, 2, "Should return 2 TODO tasks")

		for _, task := range todoTasks {
			assert.Equal(t, domain.StatusTodo, task.Status)
		}
	})

	t.Run("ListByStatus_InvalidStatus_ReturnsError", func(t *testing.T) {
		_, err := taskRepo.ListByStatus(context.Background(), domain.TaskStatus("invalid"), 0, 10)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid task status")
	})

	t.Run("ListByCreator_WithTasks_ReturnsTasksCreatedByUser", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, assignee, project := setupTestData(t, suite)

		// Create tasks with different reporters
		ownerTasks := make([]*domain.Task, 2)
		for i := 0; i < 2; i++ {
			task := suite.Factory.CreateTask(project, owner,
				WithTaskTitle(fmt.Sprintf("Owner Task %d", i)),
			)
			require.NoError(t, taskRepo.Create(context.Background(), task))
			ownerTasks[i] = task
		}

		// Create task reported by assignee
		assigneeTask := suite.Factory.CreateTask(project, assignee,
			WithTaskTitle("Assignee Task"),
		)
		require.NoError(t, taskRepo.Create(context.Background(), assigneeTask))

		// Act
		ownerCreatedTasks, err := taskRepo.ListByCreator(context.Background(), owner.ID, 0, 10)

		// Assert
		require.NoError(t, err)
		assert.Len(t, ownerCreatedTasks, 2)

		for _, task := range ownerCreatedTasks {
			assert.Equal(t, owner.ID, task.ReporterID)
		}
	})

	t.Run("Search_ByTitle_ReturnsMatchingTasks", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupTestData(t, suite)

		// Create tasks with different titles
		searchableTasks := []struct {
			title       string
			description string
		}{
			{"Bug Fix Authentication", "Fix login issues"},
			{"Feature User Dashboard", "Create user dashboard"},
			{"Bug Fix Database", "Fix database connection"},
			{"Documentation Update", "Update API docs"},
		}

		for _, taskData := range searchableTasks {
			task := suite.Factory.CreateTask(project, owner,
				WithTaskTitle(taskData.title),
				WithTaskDescription(taskData.description),
			)
			require.NoError(t, taskRepo.Create(context.Background(), task))
		}

		// Search for "Bug" - should match 2 tasks
		bugTasks, err := taskRepo.Search(context.Background(), "Bug", "", 0, 10)
		require.NoError(t, err)
		assert.Len(t, bugTasks, 2)

		// Search for "Dashboard" - should match 1 task
		dashboardTasks, err := taskRepo.Search(context.Background(), "Dashboard", "", 0, 10)
		require.NoError(t, err)
		assert.Len(t, dashboardTasks, 1)
		assert.Contains(t, dashboardTasks[0].Title, "Dashboard")

		// Search for "login" in description - should match 1 task
		loginTasks, err := taskRepo.Search(context.Background(), "login", "", 0, 10)
		require.NoError(t, err)
		assert.Len(t, loginTasks, 1)
		assert.Contains(t, loginTasks[0].Description, "login")
	})

	t.Run("Search_WithProjectFilter_ReturnsFilteredResults", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project1 := setupTestData(t, suite)

		// Create second project
		project2 := suite.Factory.CreateProject(owner,
			WithProjectTitle("Project 2"),
			WithProjectSlug("project-2"),
		)
		require.NoError(t, projectRepo.Create(context.Background(), project2))

		// Create tasks in both projects
		task1 := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Search Task Project 1"),
		)
		require.NoError(t, taskRepo.Create(context.Background(), task1))

		task2 := suite.Factory.CreateTask(project2, owner,
			WithTaskTitle("Search Task Project 2"),
		)
		require.NoError(t, taskRepo.Create(context.Background(), task2))

		// Search with project filter
		project1Tasks, err := taskRepo.Search(context.Background(), "Search", project1.ID, 0, 10)
		require.NoError(t, err)
		assert.Len(t, project1Tasks, 1)
		assert.Equal(t, project1.ID, project1Tasks[0].ProjectID)

		// Search without project filter - should find both
		allTasks, err := taskRepo.Search(context.Background(), "Search", "", 0, 10)
		require.NoError(t, err)
		assert.Len(t, allTasks, 2)
	})

	t.Run("Search_EmptyQuery_ReturnsError", func(t *testing.T) {
		_, err := taskRepo.Search(context.Background(), "", "", 0, 10)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "search query cannot be empty")
	})

	// ===========================================
	// COUNT OPERATIONS TESTS
	// ===========================================

	t.Run("Count_WithTasks_ReturnsCorrectCount", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupTestData(t, suite)

		expectedCount := 5
		for i := 0; i < expectedCount; i++ {
			task := suite.Factory.CreateTask(project, owner,
				WithTaskTitle(fmt.Sprintf("Count Task %d", i)),
			)
			require.NoError(t, taskRepo.Create(context.Background(), task))
		}

		count, err := taskRepo.Count(context.Background())
		require.NoError(t, err)
		assert.Equal(t, expectedCount, count)
	})

	t.Run("CountByProject_WithTasks_ReturnsCorrectCount", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project1 := setupTestData(t, suite)

		// Create second project
		project2 := suite.Factory.CreateProject(owner,
			WithProjectTitle("Project 2"),
			WithProjectSlug("project-2"),
		)
		require.NoError(t, projectRepo.Create(context.Background(), project2))

		// Create tasks in project1
		for i := 0; i < 3; i++ {
			task := suite.Factory.CreateTask(project1, owner)
			require.NoError(t, taskRepo.Create(context.Background(), task))
		}

		// Create tasks in project2
		for i := 0; i < 2; i++ {
			task := suite.Factory.CreateTask(project2, owner)
			require.NoError(t, taskRepo.Create(context.Background(), task))
		}

		// Test counts
		count1, err := taskRepo.CountByProject(context.Background(), project1.ID)
		require.NoError(t, err)
		assert.Equal(t, 3, count1)

		count2, err := taskRepo.CountByProject(context.Background(), project2.ID)
		require.NoError(t, err)
		assert.Equal(t, 2, count2)
	})

	t.Run("CountByAssignee_WithTasks_ReturnsCorrectCount", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, assignee, project := setupTestData(t, suite)

		// Create assigned tasks
		for i := 0; i < 4; i++ {
			task := suite.Factory.CreateTask(project, owner,
				WithTaskAssignee(assignee.ID),
			)
			require.NoError(t, taskRepo.Create(context.Background(), task))
		}

		// Create unassigned task
		unassignedTask := suite.Factory.CreateTask(project, owner)
		require.NoError(t, taskRepo.Create(context.Background(), unassignedTask))

		count, err := taskRepo.CountByAssignee(context.Background(), assignee.ID)
		require.NoError(t, err)
		assert.Equal(t, 4, count)
	})

	t.Run("CountByStatus_WithTasks_ReturnsCorrectCount", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupTestData(t, suite)

		// Create tasks with different statuses
		todoTasks := 3
		developingTasks := 2

		for i := 0; i < todoTasks; i++ {
			task := suite.Factory.CreateTask(project, owner,
				WithTaskStatus(domain.StatusTodo),
			)
			require.NoError(t, taskRepo.Create(context.Background(), task))
		}

		for i := 0; i < developingTasks; i++ {
			task := suite.Factory.CreateTask(project, owner,
				WithTaskStatus(domain.StatusDeveloping),
			)
			require.NoError(t, taskRepo.Create(context.Background(), task))
		}

		todoCount, err := taskRepo.CountByStatus(context.Background(), domain.StatusTodo)
		require.NoError(t, err)
		assert.Equal(t, todoTasks, todoCount)

		developingCount, err := taskRepo.CountByStatus(context.Background(), domain.StatusDeveloping)
		require.NoError(t, err)
		assert.Equal(t, developingTasks, developingCount)

		// Test non-existent status count
		completeCount, err := taskRepo.CountByStatus(context.Background(), domain.StatusComplete)
		require.NoError(t, err)
		assert.Equal(t, 0, completeCount)
	})

	t.Run("ExistsByID_ExistingTask_ReturnsTrue", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupTestData(t, suite)

		task := suite.Factory.CreateTask(project, owner)
		require.NoError(t, taskRepo.Create(context.Background(), task))

		exists, err := taskRepo.ExistsByID(context.Background(), task.ID)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("ExistsByID_NonExistentTask_ReturnsFalse", func(t *testing.T) {
		exists, err := taskRepo.ExistsByID(context.Background(), "nonexistent123")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("ExistsByID_EmptyID_ReturnsError", func(t *testing.T) {
		_, err := taskRepo.ExistsByID(context.Background(), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "task ID cannot be empty")
	})

	// ===========================================
	// BULK OPERATIONS TESTS
	// ===========================================

	t.Run("BulkUpdate_ValidTasks_Success", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupTestData(t, suite)

		// Create and save multiple tasks
		tasks := make([]*domain.Task, 3)
		for i := 0; i < 3; i++ {
			task := suite.Factory.CreateTask(project, owner,
				WithTaskTitle(fmt.Sprintf("Bulk Task %d", i)),
				WithTaskStatus(domain.StatusTodo),
			)
			require.NoError(t, taskRepo.Create(context.Background(), task))
			tasks[i] = task
		}

		// Update all tasks
		for _, task := range tasks {
			task.Status = domain.StatusDeveloping
			task.Progress = 25
		}

		err := taskRepo.BulkUpdate(context.Background(), tasks)
		require.NoError(t, err)

		// Verify updates
		for _, task := range tasks {
			retrieved, err := taskRepo.GetByID(context.Background(), task.ID)
			require.NoError(t, err)
			assert.Equal(t, domain.StatusDeveloping, retrieved.Status)
			assert.Equal(t, 25, retrieved.Progress)
		}
	})

	t.Run("BulkUpdate_EmptySlice_NoOp", func(t *testing.T) {
		err := taskRepo.BulkUpdate(context.Background(), []*domain.Task{})
		require.NoError(t, err) // Should be no-op
	})

	t.Run("BulkUpdate_TaskWithoutID_ReturnsError", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupTestData(t, suite)

		task := suite.Factory.CreateTask(project, owner)
		task.ID = "" // Invalid - empty ID

		err := taskRepo.BulkUpdate(context.Background(), []*domain.Task{task})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty ID")
	})

	t.Run("BulkDelete_ValidIDs_Success", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupTestData(t, suite)

		// Create and save tasks
		taskIDs := make([]string, 3)
		for i := 0; i < 3; i++ {
			task := suite.Factory.CreateTask(project, owner,
				WithTaskTitle(fmt.Sprintf("Bulk Delete Task %d", i)),
			)
			require.NoError(t, taskRepo.Create(context.Background(), task))
			taskIDs[i] = task.ID
		}

		// Verify tasks exist
		for _, id := range taskIDs {
			exists, err := taskRepo.ExistsByID(context.Background(), id)
			require.NoError(t, err)
			assert.True(t, exists)
		}

		// Bulk delete
		err := taskRepo.BulkDelete(context.Background(), taskIDs)
		require.NoError(t, err)

		// Verify tasks no longer exist
		for _, id := range taskIDs {
			exists, err := taskRepo.ExistsByID(context.Background(), id)
			require.NoError(t, err)
			assert.False(t, exists)
		}
	})

	t.Run("BulkDelete_EmptySlice_NoOp", func(t *testing.T) {
		err := taskRepo.BulkDelete(context.Background(), []string{})
		require.NoError(t, err) // Should be no-op
	})

	t.Run("BulkDelete_EmptyID_ReturnsError", func(t *testing.T) {
		err := taskRepo.BulkDelete(context.Background(), []string{"valid123", ""})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "task ID")
		assert.Contains(t, err.Error(), "is empty")
	})

	// ===========================================
	// ARCHIVE OPERATIONS TESTS
	// ===========================================

	t.Run("ArchiveTask_ExistingTask_Success", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupTestData(t, suite)

		// Create and save task
		task := suite.Factory.CreateTask(project, owner,
			WithTaskTitle("Task to Archive"),
		)
		require.NoError(t, taskRepo.Create(context.Background(), task))

		// Archive task
		err := taskRepo.ArchiveTask(context.Background(), task.ID)
		require.NoError(t, err)

		// Verify task still exists but is archived
		archived, err := taskRepo.GetByID(context.Background(), task.ID)
		require.NoError(t, err)
		assert.Equal(t, task.ID, archived.ID)
		// Note: Archive status would be checked through custom fields or separate field
	})

	t.Run("UnarchiveTask_ArchivedTask_Success", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupTestData(t, suite)

		// Create, save, and archive task
		task := suite.Factory.CreateTask(project, owner,
			WithTaskTitle("Task to Unarchive"),
		)
		require.NoError(t, taskRepo.Create(context.Background(), task))
		require.NoError(t, taskRepo.ArchiveTask(context.Background(), task.ID))

		// Unarchive task
		err := taskRepo.UnarchiveTask(context.Background(), task.ID)
		require.NoError(t, err)

		// Verify task is unarchived
		unarchived, err := taskRepo.GetByID(context.Background(), task.ID)
		require.NoError(t, err)
		assert.Equal(t, task.ID, unarchived.ID)
	})

	// ===========================================
	// RELATIONSHIP CONSTRAINT TESTS
	// ===========================================

	t.Run("Create_WithInvalidAssigneeID_ReturnsError", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupTestData(t, suite)

		// Create task with non-existent assignee
		invalidAssigneeID := "nonexistent123"
		task := suite.Factory.CreateTask(project, owner,
			WithTaskTitle("Invalid Assignee Task"),
			WithTaskAssignee(invalidAssigneeID),
		)

		err := taskRepo.Create(context.Background(), task)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to save task record")
	})

	t.Run("Create_WithValidRelationships_Success", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, assignee, project := setupTestData(t, suite)

		// Create parent task
		parentTask := suite.Factory.CreateTask(project, owner,
			WithTaskTitle("Parent Task"),
		)
		require.NoError(t, taskRepo.Create(context.Background(), parentTask))

		// Create child task with all valid relationships
		childTask := suite.Factory.CreateTask(project, owner,
			WithTaskTitle("Child Task"),
			WithTaskAssignee(assignee.ID),
			WithTaskParent(parentTask.ID),
		)

		err := taskRepo.Create(context.Background(), childTask)
		require.NoError(t, err)

		// Verify relationships
		retrieved, err := taskRepo.GetByID(context.Background(), childTask.ID)
		require.NoError(t, err)
		assert.Equal(t, project.ID, retrieved.ProjectID)
		assert.Equal(t, owner.ID, retrieved.ReporterID)
		assert.Equal(t, assignee.ID, *retrieved.AssigneeID)
		assert.Equal(t, parentTask.ID, *retrieved.ParentTaskID)
	})

	// ===========================================
	// TASK HIERARCHY TESTS
	// ===========================================

	t.Run("TaskHierarchy_ParentChild_WorksCorrectly", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupTestData(t, suite)

		// Create parent task
		parentTask := suite.Factory.CreateTask(project, owner,
			WithTaskTitle("Parent Task"),
		)
		require.NoError(t, taskRepo.Create(context.Background(), parentTask))

		// Create multiple child tasks
		childTasks := make([]*domain.Task, 3)
		for i := 0; i < 3; i++ {
			child := suite.Factory.CreateTask(project, owner,
				WithTaskTitle(fmt.Sprintf("Child Task %d", i)),
				WithTaskParent(parentTask.ID),
			)
			require.NoError(t, taskRepo.Create(context.Background(), child))
			childTasks[i] = child
		}

		// Verify parent-child relationships
		for _, child := range childTasks {
			retrieved, err := taskRepo.GetByID(context.Background(), child.ID)
			require.NoError(t, err)
			assert.Equal(t, parentTask.ID, *retrieved.ParentTaskID)
		}

		// Verify parent task exists and is correct
		retrievedParent, err := taskRepo.GetByID(context.Background(), parentTask.ID)
		require.NoError(t, err)
		assert.Nil(t, retrievedParent.ParentTaskID, "Parent task should not have a parent")
	})

	t.Run("TaskHierarchy_SelfReference_PreventedByDomain", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupTestData(t, suite)

		// Create task
		task := suite.Factory.CreateTask(project, owner,
			WithTaskTitle("Self-Referencing Task"),
		)
		require.NoError(t, taskRepo.Create(context.Background(), task))

		// Try to set itself as parent (domain validation should prevent this)
		err := task.SetParentTask(task.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be its own parent")
	})

	t.Run("TaskHierarchy_DeepNesting_WorksCorrectly", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupTestData(t, suite)

		// Create a chain of tasks: grandparent -> parent -> child
		grandparent := suite.Factory.CreateTask(project, owner,
			WithTaskTitle("Grandparent Task"),
		)
		require.NoError(t, taskRepo.Create(context.Background(), grandparent))

		parent := suite.Factory.CreateTask(project, owner,
			WithTaskTitle("Parent Task"),
			WithTaskParent(grandparent.ID),
		)
		require.NoError(t, taskRepo.Create(context.Background(), parent))

		child := suite.Factory.CreateTask(project, owner,
			WithTaskTitle("Child Task"),
			WithTaskParent(parent.ID),
		)
		require.NoError(t, taskRepo.Create(context.Background(), child))

		// Verify the hierarchy
		retrievedChild, err := taskRepo.GetByID(context.Background(), child.ID)
		require.NoError(t, err)
		assert.Equal(t, parent.ID, *retrievedChild.ParentTaskID)

		retrievedParent, err := taskRepo.GetByID(context.Background(), parent.ID)
		require.NoError(t, err)
		assert.Equal(t, grandparent.ID, *retrievedParent.ParentTaskID)

		retrievedGrandparent, err := taskRepo.GetByID(context.Background(), grandparent.ID)
		require.NoError(t, err)
		assert.Nil(t, retrievedGrandparent.ParentTaskID)
	})

	t.Run("TaskHierarchy_InvalidParentTaskID_ReturnsError", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupTestData(t, suite)

		// Create task with non-existent parent
		task := suite.Factory.CreateTask(project, owner,
			WithTaskTitle("Invalid Parent Task"),
			WithTaskParent("nonexistent123"),
		)

		err := taskRepo.Create(context.Background(), task)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to save task record")
	})

	// ===========================================
	// DATA INTEGRITY TESTS
	// ===========================================

	t.Run("DataIntegrity_JSONFields_HandledCorrectly", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupTestData(t, suite)

		// Create task with complex JSON data
		task := suite.Factory.CreateTask(project, owner,
			WithTaskTitle("JSON Data Task"),
		)

		// Set complex JSON fields
		customFields := map[string]interface{}{
			"complexity": "high",
			"category":   "bug",
			"sprint":     "sprint-3",
			"metadata": map[string]interface{}{
				"browser":      "chrome",
				"version":      "1.2.3",
				"reporter":     "automated-test",
				"severity":     9,
				"reproducible": true,
			},
			"tags": []string{"critical", "p0", "blocker"},
		}
		customFieldsJSON, err := json.Marshal(customFields)
		require.NoError(t, err)
		task.CustomFields = customFieldsJSON

		columnPosition := map[string]int{
			"backlog":    10,
			"todo":       5,
			"developing": 0,
		}
		columnPosJSON, err := json.Marshal(columnPosition)
		require.NoError(t, err)
		task.ColumnPosition = columnPosJSON

		githubData := map[string]interface{}{
			"issue_number": 123,
			"pr_number":    456,
			"branch":       "feature/task-integration",
			"commits": []map[string]string{
				{"sha": "abc123", "message": "Initial commit"},
				{"sha": "def456", "message": "Bug fixes"},
			},
		}
		githubDataJSON, err := json.Marshal(githubData)
		require.NoError(t, err)
		task.GithubData = githubDataJSON

		task.Tags = []string{"integration", "json", "complex"}
		task.Attachments = []string{"screenshot1.png", "logs.txt", "config.json"}

		// Save and retrieve
		require.NoError(t, taskRepo.Create(context.Background(), task))
		retrieved, err := taskRepo.GetByID(context.Background(), task.ID)
		require.NoError(t, err)

		// Verify JSON fields
		assert.JSONEq(t, string(task.CustomFields), string(retrieved.CustomFields))
		assert.JSONEq(t, string(task.ColumnPosition), string(retrieved.ColumnPosition))
		assert.JSONEq(t, string(task.GithubData), string(retrieved.GithubData))
		assert.Equal(t, task.Tags, retrieved.Tags)
		assert.Equal(t, task.Attachments, retrieved.Attachments)

		// Test domain methods for JSON fields work correctly
		retrievedCustomFields, err := retrieved.GetCustomFieldsMap()
		require.NoError(t, err)
		assert.Equal(t, "high", retrievedCustomFields["complexity"])
		assert.Equal(t, "bug", retrievedCustomFields["category"])

		retrievedColumnPos, err := retrieved.GetColumnPositionMap()
		require.NoError(t, err)
		assert.Equal(t, 10, retrievedColumnPos["backlog"])
		assert.Equal(t, 5, retrievedColumnPos["todo"])
	})

	t.Run("DataIntegrity_DateFields_HandledCorrectly", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupTestData(t, suite)

		// Create task with specific dates
		now := time.Now().UTC()
		dueDate := now.Add(7 * 24 * time.Hour)
		startDate := now.Add(2 * 24 * time.Hour)

		task := suite.Factory.CreateTask(project, owner,
			WithTaskTitle("Date Fields Task"),
			WithTaskDueDate(dueDate),
		)
		task.StartDate = &startDate

		require.NoError(t, taskRepo.Create(context.Background(), task))

		// Retrieve and verify dates
		retrieved, err := taskRepo.GetByID(context.Background(), task.ID)
		require.NoError(t, err)

		assert.True(t, retrieved.DueDate.Equal(dueDate), "Due date should match")
		assert.True(t, retrieved.StartDate.Equal(startDate), "Start date should match")
		assert.False(t, retrieved.CreatedAt.IsZero(), "CreatedAt should be set")
		assert.False(t, retrieved.UpdatedAt.IsZero(), "UpdatedAt should be set")
	})

	t.Run("DataIntegrity_NumericFields_HandledCorrectly", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupTestData(t, suite)

		// Create task with various numeric values
		effortEstimate := 16.5
		task := suite.Factory.CreateTask(project, owner,
			WithTaskTitle("Numeric Fields Task"),
			WithTaskProgress(75),
		)
		task.EffortEstimate = &effortEstimate
		task.TimeSpent = 12.75
		task.Position = 999

		require.NoError(t, taskRepo.Create(context.Background(), task))

		// Retrieve and verify numeric fields
		retrieved, err := taskRepo.GetByID(context.Background(), task.ID)
		require.NoError(t, err)

		assert.Equal(t, 75, retrieved.Progress)
		assert.Equal(t, 16.5, *retrieved.EffortEstimate)
		assert.Equal(t, 12.75, retrieved.TimeSpent)
		assert.Equal(t, 999, retrieved.Position)
	})

	// ===========================================
	// PERFORMANCE AND SCALE TESTS
	// ===========================================

	t.Run("Performance_BulkOperations_WithLargeDataset", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupTestData(t, suite)

		// Create a larger number of tasks for performance testing
		taskCount := 50
		tasks := make([]*domain.Task, taskCount)

		// Measure creation time
		startTime := time.Now()
		for i := 0; i < taskCount; i++ {
			task := suite.Factory.CreateTask(project, owner,
				WithTaskTitle(fmt.Sprintf("Performance Task %d", i)),
				WithTaskProgress(i%101), // 0-100
			)
			task.Position = i
			require.NoError(t, taskRepo.Create(context.Background(), task))
			tasks[i] = task
		}
		creationDuration := time.Since(startTime)
		t.Logf("Created %d tasks in %v (avg: %v per task)",
			taskCount, creationDuration, creationDuration/time.Duration(taskCount))

		// Test bulk update performance
		for _, task := range tasks {
			task.Status = domain.StatusDeveloping
		}

		startTime = time.Now()
		err := taskRepo.BulkUpdate(context.Background(), tasks)
		updateDuration := time.Since(startTime)
		require.NoError(t, err)
		t.Logf("Bulk updated %d tasks in %v (avg: %v per task)",
			taskCount, updateDuration, updateDuration/time.Duration(taskCount))

		// Test query performance with larger dataset
		startTime = time.Now()
		allTasks, err := taskRepo.ListByProject(context.Background(), project.ID, 0, taskCount)
		queryDuration := time.Since(startTime)
		require.NoError(t, err)
		assert.Len(t, allTasks, taskCount)
		t.Logf("Queried %d tasks in %v", taskCount, queryDuration)

		// Performance assertions (adjust thresholds based on requirements)
		assert.Less(t, creationDuration, 10*time.Second, "Task creation should be reasonably fast")
		assert.Less(t, updateDuration, 5*time.Second, "Bulk update should be reasonably fast")
		assert.Less(t, queryDuration, 1*time.Second, "Query should be reasonably fast")
	})

	t.Run("Performance_SearchWithLargeDataset_PerformsWell", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupTestData(t, suite)

		// Create tasks with searchable content
		searchTerms := []string{"bug", "feature", "improvement", "task", "epic"}
		taskCount := 100

		for i := 0; i < taskCount; i++ {
			term := searchTerms[i%len(searchTerms)]
			task := suite.Factory.CreateTask(project, owner,
				WithTaskTitle(fmt.Sprintf("Search %s Task %d", term, i)),
				WithTaskDescription(fmt.Sprintf("This is a %s for testing search functionality", term)),
			)
			require.NoError(t, taskRepo.Create(context.Background(), task))
		}

		// Test search performance
		startTime := time.Now()
		bugTasks, err := taskRepo.Search(context.Background(), "bug", "", 0, 50)
		searchDuration := time.Since(startTime)

		require.NoError(t, err)
		assert.Greater(t, len(bugTasks), 0, "Should find bug tasks")
		t.Logf("Searched %d tasks for 'bug' in %v, found %d matches",
			taskCount, searchDuration, len(bugTasks))

		// Performance assertion
		assert.Less(t, searchDuration, 2*time.Second, "Search should be reasonably fast")

		// Verify search accuracy
		for _, task := range bugTasks {
			assert.True(t,
				strings.Contains(strings.ToLower(task.Title), "bug") ||
					strings.Contains(strings.ToLower(task.Description), "bug"),
				"Search results should contain the search term")
		}
	})

	t.Run("Performance_PaginationWithLargeDataset_PerformsConsistently", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupTestData(t, suite)

		// Create a larger dataset
		taskCount := 200
		for i := 0; i < taskCount; i++ {
			task := suite.Factory.CreateTask(project, owner,
				WithTaskTitle(fmt.Sprintf("Pagination Task %d", i)),
			)
			task.Position = i
			require.NoError(t, taskRepo.Create(context.Background(), task))
		}

		// Test pagination performance across multiple pages
		pageSize := 20
		totalPages := taskCount / pageSize
		var durations []time.Duration

		for page := 0; page < totalPages; page++ {
			offset := page * pageSize
			startTime := time.Now()
			tasks, err := taskRepo.ListByProject(context.Background(), project.ID, offset, pageSize)
			duration := time.Since(startTime)
			durations = append(durations, duration)

			require.NoError(t, err)
			assert.Len(t, tasks, pageSize, fmt.Sprintf("Page %d should have %d tasks", page, pageSize))
		}

		// Calculate average duration
		var totalDuration time.Duration
		for _, d := range durations {
			totalDuration += d
		}
		avgDuration := totalDuration / time.Duration(len(durations))
		t.Logf("Paginated through %d pages (avg: %v per page)", totalPages, avgDuration)

		// Performance assertions
		assert.Less(t, avgDuration, 500*time.Millisecond, "Pagination should be consistently fast")

		// Check consistency - no page should be significantly slower than others
		maxDuration := time.Duration(0)
		for _, d := range durations {
			if d > maxDuration {
				maxDuration = d
			}
		}
		assert.Less(t, maxDuration, avgDuration*3, "No page should be significantly slower than average")
	})

	// ===========================================
	// CONCURRENT OPERATIONS TESTS
	// ===========================================

	t.Run("Concurrency_SimultaneousCreations_AllSucceed", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupTestData(t, suite)

		// Create tasks concurrently
		taskCount := 10
		errChan := make(chan error, taskCount)
		taskChan := make(chan *domain.Task, taskCount)

		for i := 0; i < taskCount; i++ {
			go func(index int) {
				task := suite.Factory.CreateTask(project, owner,
					WithTaskTitle(fmt.Sprintf("Concurrent Task %d", index)),
				)
				err := taskRepo.Create(context.Background(), task)
				errChan <- err
				taskChan <- task
			}(i)
		}

		// Collect results
		var tasks []*domain.Task
		for i := 0; i < taskCount; i++ {
			err := <-errChan
			task := <-taskChan
			require.NoError(t, err, fmt.Sprintf("Task creation %d should succeed", i))
			tasks = append(tasks, task)
		}

		// Verify all tasks were created and are unique
		taskIDs := make(map[string]bool)
		for _, task := range tasks {
			assert.NotEmpty(t, task.ID, "Task should have an ID")
			assert.False(t, taskIDs[task.ID], "Task IDs should be unique")
			taskIDs[task.ID] = true

			// Verify task exists in database
			retrieved, err := taskRepo.GetByID(context.Background(), task.ID)
			require.NoError(t, err)
			assert.Equal(t, task.Title, retrieved.Title)
		}

		assert.Len(t, taskIDs, taskCount, "Should have created all tasks with unique IDs")
	})

	t.Run("Concurrency_SimultaneousUpdates_AllSucceed", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupTestData(t, suite)

		// Create initial tasks
		taskCount := 5
		tasks := make([]*domain.Task, taskCount)
		for i := 0; i < taskCount; i++ {
			task := suite.Factory.CreateTask(project, owner,
				WithTaskTitle(fmt.Sprintf("Concurrent Update Task %d", i)),
			)
			require.NoError(t, taskRepo.Create(context.Background(), task))
			tasks[i] = task
		}

		// Update tasks concurrently
		errChan := make(chan error, taskCount)
		for i, task := range tasks {
			go func(index int, t *domain.Task) {
				t.Title = fmt.Sprintf("Updated Task %d", index)
				t.Progress = (index + 1) * 20
				t.Status = domain.StatusDeveloping
				errChan <- taskRepo.Update(context.Background(), t)
			}(i, task)
		}

		// Collect results
		for i := 0; i < taskCount; i++ {
			err := <-errChan
			require.NoError(t, err, fmt.Sprintf("Task update %d should succeed", i))
		}

		// Verify all updates were applied
		for i, task := range tasks {
			retrieved, err := taskRepo.GetByID(context.Background(), task.ID)
			require.NoError(t, err)
			assert.Equal(t, fmt.Sprintf("Updated Task %d", i), retrieved.Title)
			assert.Equal(t, (i+1)*20, retrieved.Progress)
			assert.Equal(t, domain.StatusDeveloping, retrieved.Status)
		}
	})

	// ===========================================
	// EDGE CASES AND ERROR SCENARIOS
	// ===========================================

	t.Run("EdgeCase_TaskWithMaximumFieldValues_HandledCorrectly", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, assignee, project := setupTestData(t, suite)

		// Create task with maximum allowed field values
		maxTitle := strings.Repeat("a", 200) // Max title length
		maxProgress := 100
		maxPosition := 999999
		maxTimeSpent := 999999.99
		maxEffortEstimate := 999999.99

		// Create very large but valid JSON data
		largeCustomFields := map[string]interface{}{
			"description": strings.Repeat("Large description field. ", 100),
			"metadata": map[string]interface{}{
				"large_array": make([]string, 100),
				"nested": map[string]interface{}{
					"deep_nesting": map[string]string{
						"key1": strings.Repeat("value1", 50),
						"key2": strings.Repeat("value2", 50),
					},
				},
			},
		}
		customFieldsJSON, err := json.Marshal(largeCustomFields)
		require.NoError(t, err)

		// Create task with maximum values
		task := suite.Factory.CreateTask(project, owner,
			WithTaskTitle(maxTitle),
			WithTaskAssignee(assignee.ID),
			WithTaskProgress(maxProgress),
		)
		task.Position = maxPosition
		task.TimeSpent = maxTimeSpent
		task.EffortEstimate = &maxEffortEstimate
		task.CustomFields = customFieldsJSON
		task.Tags = make([]string, 50) // Large number of tags
		for i := 0; i < 50; i++ {
			task.Tags[i] = fmt.Sprintf("tag_%d", i)
		}

		// Should handle maximum values without error
		err = taskRepo.Create(context.Background(), task)
		require.NoError(t, err)

		// Verify retrieval works correctly
		retrieved, err := taskRepo.GetByID(context.Background(), task.ID)
		require.NoError(t, err)
		assert.Equal(t, maxTitle, retrieved.Title)
		assert.Equal(t, maxProgress, retrieved.Progress)
		assert.Equal(t, maxPosition, retrieved.Position)
		assert.Equal(t, maxTimeSpent, retrieved.TimeSpent)
		assert.Equal(t, maxEffortEstimate, *retrieved.EffortEstimate)
		assert.Len(t, retrieved.Tags, 50)
		assert.JSONEq(t, string(task.CustomFields), string(retrieved.CustomFields))
	})

	t.Run("EdgeCase_TaskWithMinimumFieldValues_HandledCorrectly", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupTestData(t, suite)

		// Create task with minimum values
		task := suite.Factory.CreateTask(project, owner,
			WithTaskTitle("A"), // Minimum title length (1 char)
			WithTaskProgress(0),
		)
		task.Position = 0
		task.TimeSpent = 0
		zeroEffort := 0.0
		task.EffortEstimate = &zeroEffort
		task.Tags = []string{}
		task.Dependencies = []string{}
		task.Attachments = []string{}
		task.CustomFields = json.RawMessage("{}")

		err := taskRepo.Create(context.Background(), task)
		require.NoError(t, err)

		// Verify minimum values are handled correctly
		retrieved, err := taskRepo.GetByID(context.Background(), task.ID)
		require.NoError(t, err)
		assert.Equal(t, "A", retrieved.Title)
		assert.Equal(t, 0, retrieved.Progress)
		assert.Equal(t, 0, retrieved.Position)
		assert.Equal(t, 0.0, retrieved.TimeSpent)
		assert.Equal(t, 0.0, *retrieved.EffortEstimate)
		assert.Empty(t, retrieved.Tags)
		assert.Empty(t, retrieved.Dependencies)
		assert.Empty(t, retrieved.Attachments)
	})

	t.Run("EdgeCase_EmptyOptionalFields_HandledCorrectly", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupTestData(t, suite)

		// Create task with explicitly nil/empty optional fields
		task := suite.Factory.CreateTask(project, owner,
			WithTaskTitle("Empty Optionals Task"),
		)
		task.AssigneeID = nil
		task.ParentTaskID = nil
		task.DueDate = nil
		task.StartDate = nil
		task.EffortEstimate = nil
		task.Tags = nil
		task.Dependencies = nil
		task.Attachments = nil
		task.CustomFields = nil
		task.GithubData = nil
		task.ColumnPosition = nil

		err := taskRepo.Create(context.Background(), task)
		require.NoError(t, err)

		// Verify nil values are handled correctly
		retrieved, err := taskRepo.GetByID(context.Background(), task.ID)
		require.NoError(t, err)
		assert.Nil(t, retrieved.AssigneeID)
		assert.Nil(t, retrieved.ParentTaskID)
		assert.Nil(t, retrieved.DueDate)
		assert.Nil(t, retrieved.StartDate)
		assert.Nil(t, retrieved.EffortEstimate)
		assert.Empty(t, retrieved.Tags)
		assert.Empty(t, retrieved.Dependencies)
		assert.Empty(t, retrieved.Attachments)
		assert.Empty(t, retrieved.CustomFields)
		assert.Empty(t, retrieved.GithubData)
		assert.Empty(t, retrieved.ColumnPosition)
	})
}
