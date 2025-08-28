//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ericfisherdev/simple-easy-tasks/internal/domain"
	"github.com/ericfisherdev/simple-easy-tasks/internal/repository"
)

func TestTaskRepository_EnhancedFeatures_Integration(t *testing.T) {
	// Setup SHARED test suite for all subtests (performance optimization)
	suite := SetupDatabaseTest(t)
	defer suite.Cleanup()

	// Get repositories from suite
	taskRepo := suite.Repos.Tasks
	userRepo := suite.Repos.Users
	projectRepo := suite.Repos.Projects

	// Helper function to create test users and projects
	setupTestData := func(t *testing.T, suite *DatabaseTestSuite) (*domain.User, *domain.User, *domain.User, *domain.Project, *domain.Project) {
		// Create users
		owner := suite.Factory.CreateUser(
			WithUserEmail("owner@enhanced.test.com"),
			WithUserUsername("enhancedowner"),
			WithUserName("Enhanced Owner"),
		)
		assignee1 := suite.Factory.CreateUser(
			WithUserEmail("assignee1@enhanced.test.com"),
			WithUserUsername("assignee1"),
			WithUserName("Assignee One"),
		)
		assignee2 := suite.Factory.CreateUser(
			WithUserEmail("assignee2@enhanced.test.com"),
			WithUserUsername("assignee2"),
			WithUserName("Assignee Two"),
		)

		// Save users to database
		require.NoError(t, userRepo.Create(context.Background(), owner))
		require.NoError(t, userRepo.Create(context.Background(), assignee1))
		require.NoError(t, userRepo.Create(context.Background(), assignee2))

		// Create projects
		project1 := suite.Factory.CreateProject(owner,
			WithProjectTitle("Enhanced Test Project 1"),
			WithProjectSlug("enhanced-test-1"),
		)
		project2 := suite.Factory.CreateProject(owner,
			WithProjectTitle("Enhanced Test Project 2"),
			WithProjectSlug("enhanced-test-2"),
		)
		require.NoError(t, projectRepo.Create(context.Background(), project1))
		require.NoError(t, projectRepo.Create(context.Background(), project2))

		return owner, assignee1, assignee2, project1, project2
	}

	// ===========================================
	// GET BY PROJECT WITH ADVANCED FILTERING TESTS
	// ===========================================

	t.Run("GetByProject_WithStatusFilter_ReturnsFilteredTasks", func(t *testing.T) {
		require.NoError(t, suite.Reset())
		owner, assignee1, _, project1, _ := setupTestData(t, suite)

		// Create tasks with different statuses
		todoTask := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("TODO Task"),
			WithTaskStatus(domain.StatusTodo),
		)
		require.NoError(t, taskRepo.Create(context.Background(), todoTask))

		devTask := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Developing Task"),
			WithTaskStatus(domain.StatusDeveloping),
			WithTaskAssignee(assignee1.ID),
		)
		require.NoError(t, taskRepo.Create(context.Background(), devTask))

		completeTask := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Complete Task"),
			WithTaskStatus(domain.StatusComplete),
		)
		require.NoError(t, taskRepo.Create(context.Background(), completeTask))

		// Test filtering by TODO status only
		filters := repository.TaskFilters{
			Status: []domain.TaskStatus{domain.StatusTodo},
			Limit:  10,
		}
		tasks, err := taskRepo.GetByProject(context.Background(), project1.ID, filters)
		require.NoError(t, err)
		assert.Len(t, tasks, 1)
		assert.Equal(t, domain.StatusTodo, tasks[0].Status)
		assert.Equal(t, "TODO Task", tasks[0].Title)

		// Test filtering by multiple statuses
		filters.Status = []domain.TaskStatus{domain.StatusTodo, domain.StatusDeveloping}
		tasks, err = taskRepo.GetByProject(context.Background(), project1.ID, filters)
		require.NoError(t, err)
		assert.Len(t, tasks, 2)

		statuses := make([]domain.TaskStatus, len(tasks))
		for i, task := range tasks {
			statuses[i] = task.Status
		}
		assert.Contains(t, statuses, domain.StatusTodo)
		assert.Contains(t, statuses, domain.StatusDeveloping)
	})

	t.Run("GetByProject_WithPriorityFilter_ReturnsFilteredTasks", func(t *testing.T) {
		require.NoError(t, suite.Reset())
		owner, _, _, project1, _ := setupTestData(t, suite)

		// Create tasks with different priorities
		highTask := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("High Priority Task"),
			WithTaskPriority(domain.PriorityHigh),
		)
		require.NoError(t, taskRepo.Create(context.Background(), highTask))

		mediumTask := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Medium Priority Task"),
			WithTaskPriority(domain.PriorityMedium),
		)
		require.NoError(t, taskRepo.Create(context.Background(), mediumTask))

		lowTask := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Low Priority Task"),
			WithTaskPriority(domain.PriorityLow),
		)
		require.NoError(t, taskRepo.Create(context.Background(), lowTask))

		// Test filtering by high priority
		filters := repository.TaskFilters{
			Priority: []domain.TaskPriority{domain.PriorityHigh},
			Limit:    10,
		}
		tasks, err := taskRepo.GetByProject(context.Background(), project1.ID, filters)
		require.NoError(t, err)
		assert.Len(t, tasks, 1)
		assert.Equal(t, domain.PriorityHigh, tasks[0].Priority)

		// Test filtering by multiple priorities
		filters.Priority = []domain.TaskPriority{domain.PriorityHigh, domain.PriorityLow}
		tasks, err = taskRepo.GetByProject(context.Background(), project1.ID, filters)
		require.NoError(t, err)
		assert.Len(t, tasks, 2)

		priorities := make([]domain.TaskPriority, len(tasks))
		for i, task := range tasks {
			priorities[i] = task.Priority
		}
		assert.Contains(t, priorities, domain.PriorityHigh)
		assert.Contains(t, priorities, domain.PriorityLow)
	})

	t.Run("GetByProject_WithAssigneeFilter_ReturnsFilteredTasks", func(t *testing.T) {
		require.NoError(t, suite.Reset())
		owner, assignee1, assignee2, project1, _ := setupTestData(t, suite)

		// Create tasks with different assignees
		assignedTask1 := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Task Assigned to User 1"),
			WithTaskAssignee(assignee1.ID),
		)
		require.NoError(t, taskRepo.Create(context.Background(), assignedTask1))

		assignedTask2 := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Task Assigned to User 2"),
			WithTaskAssignee(assignee2.ID),
		)
		require.NoError(t, taskRepo.Create(context.Background(), assignedTask2))

		unassignedTask := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Unassigned Task"),
		)
		require.NoError(t, taskRepo.Create(context.Background(), unassignedTask))

		// Test filtering by specific assignee
		filters := repository.TaskFilters{
			AssigneeID: &assignee1.ID,
			Limit:      10,
		}
		tasks, err := taskRepo.GetByProject(context.Background(), project1.ID, filters)
		require.NoError(t, err)
		assert.Len(t, tasks, 1)
		assert.Equal(t, assignee1.ID, *tasks[0].AssigneeID)

		// Test filtering for unassigned tasks
		emptyAssignee := ""
		filters.AssigneeID = &emptyAssignee
		tasks, err = taskRepo.GetByProject(context.Background(), project1.ID, filters)
		require.NoError(t, err)
		assert.Len(t, tasks, 1)
		assert.Nil(t, tasks[0].AssigneeID)
	})

	t.Run("GetByProject_WithTagsFilter_ReturnsFilteredTasks", func(t *testing.T) {
		require.NoError(t, suite.Reset())
		owner, _, _, project1, _ := setupTestData(t, suite)

		// Create tasks with different tags
		bugTask := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Bug Task"),
		)
		bugTask.Tags = []string{"bug", "critical"}
		require.NoError(t, taskRepo.Create(context.Background(), bugTask))

		featureTask := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Feature Task"),
		)
		featureTask.Tags = []string{"feature", "enhancement"}
		require.NoError(t, taskRepo.Create(context.Background(), featureTask))

		docTask := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Documentation Task"),
		)
		docTask.Tags = []string{"documentation", "critical"}
		require.NoError(t, taskRepo.Create(context.Background(), docTask))

		// Test filtering by single tag
		filters := repository.TaskFilters{
			Tags:  []string{"bug"},
			Limit: 10,
		}
		tasks, err := taskRepo.GetByProject(context.Background(), project1.ID, filters)
		require.NoError(t, err)
		assert.Len(t, tasks, 1)
		assert.Contains(t, tasks[0].Tags, "bug")

		// Test filtering by tag that appears in multiple tasks
		filters.Tags = []string{"critical"}
		tasks, err = taskRepo.GetByProject(context.Background(), project1.ID, filters)
		require.NoError(t, err)
		assert.Len(t, tasks, 2) // Both bug and doc tasks have "critical"

		// Test filtering by multiple tags (OR operation)
		filters.Tags = []string{"feature", "documentation"}
		tasks, err = taskRepo.GetByProject(context.Background(), project1.ID, filters)
		require.NoError(t, err)
		assert.Len(t, tasks, 2) // Feature task and doc task
	})

	t.Run("GetByProject_WithDateRangeFilter_ReturnsFilteredTasks", func(t *testing.T) {
		require.NoError(t, suite.Reset())
		owner, _, _, project1, _ := setupTestData(t, suite)

		now := time.Now()
		tomorrow := now.Add(24 * time.Hour)
		dayAfterTomorrow := now.Add(48 * time.Hour)
		threeDaysLater := now.Add(72 * time.Hour)

		// Create tasks with different due dates (all in future to pass validation)
		soonDueTask := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Soon Due Task"),
			WithTaskDueDate(tomorrow),
		)
		require.NoError(t, taskRepo.Create(context.Background(), soonDueTask))

		laterDueTask := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Later Due Task"),
			WithTaskDueDate(dayAfterTomorrow),
		)
		require.NoError(t, taskRepo.Create(context.Background(), laterDueTask))

		muchLaterTask := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Much Later Task"),
			WithTaskDueDate(threeDaysLater),
		)
		require.NoError(t, taskRepo.Create(context.Background(), muchLaterTask))

		noDueTask := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("No Due Date Task"),
		)
		require.NoError(t, taskRepo.Create(context.Background(), noDueTask))

		// Test filtering by due before (should find tomorrow and day after tomorrow tasks)
		beforeThreeDays := threeDaysLater.Add(-1 * time.Hour)
		filters := repository.TaskFilters{
			DueBefore: &beforeThreeDays,
			Limit:     10,
		}
		tasks, err := taskRepo.GetByProject(context.Background(), project1.ID, filters)
		require.NoError(t, err)
		assert.Len(t, tasks, 2) // Soon due and later due tasks

		// Test filtering by due after (should find all future tasks)
		afterNow := now.Add(1 * time.Hour)
		filters = repository.TaskFilters{
			DueAfter: &afterNow,
			Limit:    10,
		}
		tasks, err = taskRepo.GetByProject(context.Background(), project1.ID, filters)
		require.NoError(t, err)
		assert.Len(t, tasks, 3) // All tasks with due dates

		// Test filtering with both bounds (should find middle task only)
		afterTomorrow := tomorrow.Add(1 * time.Hour)
		beforeThreeDaysAgain := threeDaysLater.Add(-1 * time.Hour)
		filters = repository.TaskFilters{
			DueAfter:  &afterTomorrow,
			DueBefore: &beforeThreeDaysAgain,
			Limit:     10,
		}
		tasks, err = taskRepo.GetByProject(context.Background(), project1.ID, filters)
		require.NoError(t, err)
		assert.Len(t, tasks, 1)
		assert.Equal(t, "Later Due Task", tasks[0].Title)
	})

	t.Run("GetByProject_WithSearchFilter_ReturnsFilteredTasks", func(t *testing.T) {
		require.NoError(t, suite.Reset())
		owner, _, _, project1, _ := setupTestData(t, suite)

		// Create tasks with searchable content
		authTask := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Authentication Bug Fix"),
			WithTaskDescription("Fix OAuth authentication issues"),
		)
		require.NoError(t, taskRepo.Create(context.Background(), authTask))

		apiTask := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("API Development"),
			WithTaskDescription("Develop REST API endpoints"),
		)
		require.NoError(t, taskRepo.Create(context.Background(), apiTask))

		dbTask := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Database Migration"),
			WithTaskDescription("Migrate authentication database"),
		)
		require.NoError(t, taskRepo.Create(context.Background(), dbTask))

		// Test search in title
		filters := repository.TaskFilters{
			Search: "API",
			Limit:  10,
		}
		tasks, err := taskRepo.GetByProject(context.Background(), project1.ID, filters)
		require.NoError(t, err)
		assert.Len(t, tasks, 1)
		assert.Equal(t, "API Development", tasks[0].Title)

		// Test search in description
		filters.Search = "authentication"
		tasks, err = taskRepo.GetByProject(context.Background(), project1.ID, filters)
		require.NoError(t, err)
		assert.Len(t, tasks, 2) // Both auth task and db task mention authentication
	})

	t.Run("GetByProject_WithArchivedFilter_ReturnsFilteredTasks", func(t *testing.T) {
		require.NoError(t, suite.Reset())
		owner, _, _, project1, _ := setupTestData(t, suite)

		// Create active and archived tasks
		activeTask := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Active Task"),
		)
		require.NoError(t, taskRepo.Create(context.Background(), activeTask))

		archivedTask := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Archived Task"),
		)
		require.NoError(t, taskRepo.Create(context.Background(), archivedTask))
		require.NoError(t, taskRepo.ArchiveTask(context.Background(), archivedTask.ID))

		// Test filtering for active tasks only
		archivedFalse := false
		filters := repository.TaskFilters{
			Archived: &archivedFalse,
			Limit:    10,
		}
		tasks, err := taskRepo.GetByProject(context.Background(), project1.ID, filters)
		require.NoError(t, err)
		assert.Len(t, tasks, 1)
		assert.Equal(t, "Active Task", tasks[0].Title)
		assert.False(t, tasks[0].Archived)

		// Test filtering for archived tasks only
		archivedTrue := true
		filters.Archived = &archivedTrue
		tasks, err = taskRepo.GetByProject(context.Background(), project1.ID, filters)
		require.NoError(t, err)
		assert.Len(t, tasks, 1)
		assert.Equal(t, "Archived Task", tasks[0].Title)
		assert.True(t, tasks[0].Archived)
	})

	t.Run("GetByProject_WithComplexMultipleFilters_ReturnsCorrectTasks", func(t *testing.T) {
		require.NoError(t, suite.Reset())
		owner, assignee1, assignee2, project1, _ := setupTestData(t, suite)

		now := time.Now()
		tomorrow := now.Add(24 * time.Hour)

		// Create various tasks to test complex filtering
		matchingTask := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Urgent Bug Fix"),
			WithTaskStatus(domain.StatusTodo),
			WithTaskPriority(domain.PriorityHigh),
			WithTaskAssignee(assignee1.ID),
			WithTaskDueDate(tomorrow),
		)
		matchingTask.Tags = []string{"bug", "urgent"}
		require.NoError(t, taskRepo.Create(context.Background(), matchingTask))

		nonMatchingTask1 := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Feature Request"),
			WithTaskStatus(domain.StatusDeveloping), // Different status
			WithTaskPriority(domain.PriorityHigh),
			WithTaskAssignee(assignee1.ID),
			WithTaskDueDate(tomorrow),
		)
		nonMatchingTask1.Tags = []string{"feature", "urgent"}
		require.NoError(t, taskRepo.Create(context.Background(), nonMatchingTask1))

		nonMatchingTask2 := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Documentation Update"),
			WithTaskStatus(domain.StatusTodo),
			WithTaskPriority(domain.PriorityLow), // Different priority
			WithTaskAssignee(assignee2.ID),       // Different assignee
		)
		nonMatchingTask2.Tags = []string{"documentation"}
		require.NoError(t, taskRepo.Create(context.Background(), nonMatchingTask2))

		// Complex filter: TODO status, high priority, specific assignee, with "bug" tag, due tomorrow
		futureDate := now.Add(48 * time.Hour)
		filters := repository.TaskFilters{
			Status:     []domain.TaskStatus{domain.StatusTodo},
			Priority:   []domain.TaskPriority{domain.PriorityHigh},
			AssigneeID: &assignee1.ID,
			Tags:       []string{"bug"},
			DueBefore:  &futureDate,
			Limit:      10,
		}

		tasks, err := taskRepo.GetByProject(context.Background(), project1.ID, filters)
		require.NoError(t, err)
		assert.Len(t, tasks, 1)
		assert.Equal(t, "Urgent Bug Fix", tasks[0].Title)
		assert.Equal(t, domain.StatusTodo, tasks[0].Status)
		assert.Equal(t, domain.PriorityHigh, tasks[0].Priority)
		assert.Equal(t, assignee1.ID, *tasks[0].AssigneeID)
		assert.Contains(t, tasks[0].Tags, "bug")
	})

	t.Run("GetByProject_WithSortingOptions_ReturnsSortedTasks", func(t *testing.T) {
		require.NoError(t, suite.Reset())
		owner, _, _, project1, _ := setupTestData(t, suite)

		// Create tasks with different properties for sorting
		task1 := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("A First Task"),
			WithTaskPriority(domain.PriorityLow),
			WithTaskProgress(30),
		)
		task1.Position = 3
		require.NoError(t, taskRepo.Create(context.Background(), task1))

		time.Sleep(10 * time.Millisecond) // Ensure different creation times

		task2 := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("B Second Task"),
			WithTaskPriority(domain.PriorityHigh),
			WithTaskProgress(80),
		)
		task2.Position = 1
		require.NoError(t, taskRepo.Create(context.Background(), task2))

		time.Sleep(10 * time.Millisecond)

		task3 := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("C Third Task"),
			WithTaskPriority(domain.PriorityMedium),
			WithTaskProgress(50),
		)
		task3.Position = 2
		require.NoError(t, taskRepo.Create(context.Background(), task3))

		// Test sorting by title ascending
		filters := repository.TaskFilters{
			SortBy:    repository.SortByTitle,
			SortOrder: repository.SortOrderAsc,
			Limit:     10,
		}
		tasks, err := taskRepo.GetByProject(context.Background(), project1.ID, filters)
		require.NoError(t, err)
		assert.Len(t, tasks, 3)
		assert.Equal(t, "A First Task", tasks[0].Title)
		assert.Equal(t, "B Second Task", tasks[1].Title)
		assert.Equal(t, "C Third Task", tasks[2].Title)

		// Test sorting by title descending
		filters.SortOrder = repository.SortOrderDesc
		tasks, err = taskRepo.GetByProject(context.Background(), project1.ID, filters)
		require.NoError(t, err)
		assert.Len(t, tasks, 3)
		assert.Equal(t, "C Third Task", tasks[0].Title)
		assert.Equal(t, "B Second Task", tasks[1].Title)
		assert.Equal(t, "A First Task", tasks[2].Title)

		// Test sorting by position ascending
		filters.SortBy = repository.SortByPosition
		filters.SortOrder = repository.SortOrderAsc
		tasks, err = taskRepo.GetByProject(context.Background(), project1.ID, filters)
		require.NoError(t, err)
		assert.Len(t, tasks, 3)
		assert.Equal(t, 1, tasks[0].Position) // task2
		assert.Equal(t, 2, tasks[1].Position) // task3
		assert.Equal(t, 3, tasks[2].Position) // task1

		// Test sorting by progress descending
		filters.SortBy = repository.SortByProgress
		filters.SortOrder = repository.SortOrderDesc
		tasks, err = taskRepo.GetByProject(context.Background(), project1.ID, filters)
		require.NoError(t, err)
		assert.Len(t, tasks, 3)
		assert.Equal(t, 80, tasks[0].Progress) // task2
		assert.Equal(t, 50, tasks[1].Progress) // task3
		assert.Equal(t, 30, tasks[2].Progress) // task1
	})

	t.Run("GetByProject_WithPagination_ReturnsCorrectPages", func(t *testing.T) {
		require.NoError(t, suite.Reset())
		owner, _, _, project1, _ := setupTestData(t, suite)

		// Create 10 tasks for pagination testing
		for i := 0; i < 10; i++ {
			task := suite.Factory.CreateTask(project1, owner,
				WithTaskTitle(fmt.Sprintf("Pagination Task %02d", i+1)),
			)
			task.Position = i + 1
			require.NoError(t, taskRepo.Create(context.Background(), task))
		}

		// Test first page
		filters := repository.TaskFilters{
			SortBy:    repository.SortByPosition,
			SortOrder: repository.SortOrderAsc,
			Limit:     3,
			Offset:    0,
		}
		page1, err := taskRepo.GetByProject(context.Background(), project1.ID, filters)
		require.NoError(t, err)
		assert.Len(t, page1, 3)
		assert.Equal(t, "Pagination Task 01", page1[0].Title)
		assert.Equal(t, "Pagination Task 02", page1[1].Title)
		assert.Equal(t, "Pagination Task 03", page1[2].Title)

		// Test second page
		filters.Offset = 3
		page2, err := taskRepo.GetByProject(context.Background(), project1.ID, filters)
		require.NoError(t, err)
		assert.Len(t, page2, 3)
		assert.Equal(t, "Pagination Task 04", page2[0].Title)
		assert.Equal(t, "Pagination Task 05", page2[1].Title)
		assert.Equal(t, "Pagination Task 06", page2[2].Title)

		// Test last partial page
		filters.Offset = 9
		lastPage, err := taskRepo.GetByProject(context.Background(), project1.ID, filters)
		require.NoError(t, err)
		assert.Len(t, lastPage, 1)
		assert.Equal(t, "Pagination Task 10", lastPage[0].Title)
	})

	// ===========================================
	// SUBTASKS AND DEPENDENCIES TESTS
	// ===========================================

	t.Run("GetSubtasks_WithValidParent_ReturnsSubtasks", func(t *testing.T) {
		require.NoError(t, suite.Reset())
		owner, _, _, project1, _ := setupTestData(t, suite)

		// Create parent task
		parentTask := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Parent Task"),
		)
		require.NoError(t, taskRepo.Create(context.Background(), parentTask))

		// Create subtasks
		subtask1 := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Subtask 1"),
			WithTaskParent(parentTask.ID),
		)
		subtask1.Position = 1
		require.NoError(t, taskRepo.Create(context.Background(), subtask1))

		subtask2 := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Subtask 2"),
			WithTaskParent(parentTask.ID),
		)
		subtask2.Position = 2
		require.NoError(t, taskRepo.Create(context.Background(), subtask2))

		// Create unrelated task
		unrelatedTask := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Unrelated Task"),
		)
		require.NoError(t, taskRepo.Create(context.Background(), unrelatedTask))

		// Get subtasks
		subtasks, err := taskRepo.GetSubtasks(context.Background(), parentTask.ID)
		require.NoError(t, err)
		assert.Len(t, subtasks, 2)

		// Verify subtasks are returned in position order
		assert.Equal(t, "Subtask 1", subtasks[0].Title)
		assert.Equal(t, "Subtask 2", subtasks[1].Title)
		assert.Equal(t, parentTask.ID, *subtasks[0].ParentTaskID)
		assert.Equal(t, parentTask.ID, *subtasks[1].ParentTaskID)
	})

	t.Run("GetSubtasks_WithNoSubtasks_ReturnsEmptySlice", func(t *testing.T) {
		require.NoError(t, suite.Reset())
		owner, _, _, project1, _ := setupTestData(t, suite)

		// Create parent task with no subtasks
		parentTask := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Childless Parent"),
		)
		require.NoError(t, taskRepo.Create(context.Background(), parentTask))

		subtasks, err := taskRepo.GetSubtasks(context.Background(), parentTask.ID)
		require.NoError(t, err)
		assert.Empty(t, subtasks)
	})

	t.Run("GetSubtasks_WithEmptyParentID_ReturnsError", func(t *testing.T) {
		_, err := taskRepo.GetSubtasks(context.Background(), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parent task ID cannot be empty")
	})

	t.Run("GetSubtasks_WithNonexistentParent_ReturnsEmptySlice", func(t *testing.T) {
		subtasks, err := taskRepo.GetSubtasks(context.Background(), "nonexistent123")
		require.NoError(t, err)
		assert.Empty(t, subtasks)
	})

	t.Run("GetDependencies_WithValidTask_ReturnsDependencyTasks", func(t *testing.T) {
		require.NoError(t, suite.Reset())
		owner, _, _, project1, _ := setupTestData(t, suite)

		// Create dependency tasks
		dep1 := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Dependency Task 1"),
			WithTaskStatus(domain.StatusComplete),
		)
		require.NoError(t, taskRepo.Create(context.Background(), dep1))

		dep2 := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Dependency Task 2"),
			WithTaskStatus(domain.StatusDeveloping),
		)
		require.NoError(t, taskRepo.Create(context.Background(), dep2))

		// Create main task with dependencies
		mainTask := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Main Task"),
		)
		mainTask.Dependencies = []string{dep1.ID, dep2.ID}
		require.NoError(t, taskRepo.Create(context.Background(), mainTask))

		// Create unrelated task
		unrelatedTask := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Unrelated Task"),
		)
		require.NoError(t, taskRepo.Create(context.Background(), unrelatedTask))

		// Get dependencies
		dependencies, err := taskRepo.GetDependencies(context.Background(), mainTask.ID)
		require.NoError(t, err)
		assert.Len(t, dependencies, 2)

		// Verify correct dependencies are returned
		titles := make([]string, len(dependencies))
		for i, dep := range dependencies {
			titles[i] = dep.Title
		}
		assert.Contains(t, titles, "Dependency Task 1")
		assert.Contains(t, titles, "Dependency Task 2")
	})

	t.Run("GetDependencies_WithNoDependencies_ReturnsEmptySlice", func(t *testing.T) {
		require.NoError(t, suite.Reset())
		owner, _, _, project1, _ := setupTestData(t, suite)

		// Create task with no dependencies
		independentTask := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Independent Task"),
		)
		require.NoError(t, taskRepo.Create(context.Background(), independentTask))

		dependencies, err := taskRepo.GetDependencies(context.Background(), independentTask.ID)
		require.NoError(t, err)
		assert.Empty(t, dependencies)
	})

	t.Run("GetDependencies_WithEmptyTaskID_ReturnsError", func(t *testing.T) {
		_, err := taskRepo.GetDependencies(context.Background(), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "task ID cannot be empty")
	})

	t.Run("GetDependencies_WithNonexistentTask_ReturnsError", func(t *testing.T) {
		_, err := taskRepo.GetDependencies(context.Background(), "nonexistent123")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get task")
	})

	t.Run("GetDependencies_WithPartiallyValidDependencies_ReturnsValidOnes", func(t *testing.T) {
		require.NoError(t, suite.Reset())
		owner, _, _, project1, _ := setupTestData(t, suite)

		// Create valid dependency task
		validDep := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Valid Dependency"),
		)
		require.NoError(t, taskRepo.Create(context.Background(), validDep))

		// Create main task with mix of valid and invalid dependency IDs
		mainTask := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Main Task"),
		)
		mainTask.Dependencies = []string{validDep.ID, "nonexistent123", "alsoinvalid456"}
		require.NoError(t, taskRepo.Create(context.Background(), mainTask))

		// Get dependencies - should return only valid ones
		dependencies, err := taskRepo.GetDependencies(context.Background(), mainTask.ID)
		require.NoError(t, err)
		assert.Len(t, dependencies, 1)
		assert.Equal(t, "Valid Dependency", dependencies[0].Title)
	})

	// ===========================================
	// KANBAN MOVE FUNCTIONALITY TESTS
	// ===========================================

	t.Run("Move_ValidTransition_Success", func(t *testing.T) {
		require.NoError(t, suite.Reset())
		owner, _, _, project1, _ := setupTestData(t, suite)

		// Create task in TODO status
		task := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Task to Move"),
			WithTaskStatus(domain.StatusTodo),
		)
		task.Position = 5
		require.NoError(t, taskRepo.Create(context.Background(), task))

		originalUpdatedAt := task.UpdatedAt
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps

		// Move to DEVELOPING status with new position
		err := taskRepo.Move(context.Background(), task.ID, domain.StatusDeveloping, 3)
		require.NoError(t, err)

		// Verify the move
		updated, err := taskRepo.GetByID(context.Background(), task.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.StatusDeveloping, updated.Status)
		assert.Equal(t, 3, updated.Position)
		assert.True(t, updated.UpdatedAt.After(originalUpdatedAt))
	})

	t.Run("Move_InvalidTransition_ReturnsError", func(t *testing.T) {
		require.NoError(t, suite.Reset())
		owner, _, _, project1, _ := setupTestData(t, suite)

		// Create task in TODO status
		task := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Task with Invalid Transition"),
			WithTaskStatus(domain.StatusTodo),
		)
		require.NoError(t, taskRepo.Create(context.Background(), task))

		// Try to move directly from TODO to COMPLETE (invalid transition)
		err := taskRepo.Move(context.Background(), task.ID, domain.StatusComplete, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot transition from")

		// Verify task status hasn't changed
		unchanged, err := taskRepo.GetByID(context.Background(), task.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.StatusTodo, unchanged.Status)
	})

	t.Run("Move_EmptyTaskID_ReturnsError", func(t *testing.T) {
		err := taskRepo.Move(context.Background(), "", domain.StatusDeveloping, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "task ID cannot be empty")
	})

	t.Run("Move_InvalidStatus_ReturnsError", func(t *testing.T) {
		err := taskRepo.Move(context.Background(), "task123", domain.TaskStatus("invalid"), 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid task status")
	})

	t.Run("Move_NegativePosition_ReturnsError", func(t *testing.T) {
		err := taskRepo.Move(context.Background(), "task123", domain.StatusDeveloping, -1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "position cannot be negative")
	})

	t.Run("Move_NonexistentTask_ReturnsError", func(t *testing.T) {
		err := taskRepo.Move(context.Background(), "nonexistent123", domain.StatusDeveloping, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to find task for move")
	})

	t.Run("Move_ValidComplexTransition_Success", func(t *testing.T) {
		require.NoError(t, suite.Reset())
		owner, _, _, project1, _ := setupTestData(t, suite)

		// Create task and move through valid workflow
		task := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Complex Workflow Task"),
			WithTaskStatus(domain.StatusBacklog),
		)
		require.NoError(t, taskRepo.Create(context.Background(), task))

		// Move from BACKLOG -> TODO
		err := taskRepo.Move(context.Background(), task.ID, domain.StatusTodo, 1)
		require.NoError(t, err)

		// Move from TODO -> DEVELOPING
		err = taskRepo.Move(context.Background(), task.ID, domain.StatusDeveloping, 2)
		require.NoError(t, err)

		// Move from DEVELOPING -> REVIEW
		err = taskRepo.Move(context.Background(), task.ID, domain.StatusReview, 3)
		require.NoError(t, err)

		// Move from REVIEW -> COMPLETE
		err = taskRepo.Move(context.Background(), task.ID, domain.StatusComplete, 4)
		require.NoError(t, err)

		// Verify final state
		final, err := taskRepo.GetByID(context.Background(), task.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.StatusComplete, final.Status)
		assert.Equal(t, 4, final.Position)
	})

	// ===========================================
	// BULK OPERATIONS TESTS
	// ===========================================

	t.Run("BulkUpdateStatus_ValidTransitions_Success", func(t *testing.T) {
		require.NoError(t, suite.Reset())
		owner, _, _, project1, _ := setupTestData(t, suite)

		// Create multiple tasks in TODO status
		var taskIDs []string
		for i := 0; i < 3; i++ {
			task := suite.Factory.CreateTask(project1, owner,
				WithTaskTitle(fmt.Sprintf("Bulk Update Task %d", i+1)),
				WithTaskStatus(domain.StatusTodo),
			)
			require.NoError(t, taskRepo.Create(context.Background(), task))
			taskIDs = append(taskIDs, task.ID)
		}

		// Bulk update to DEVELOPING status
		err := taskRepo.BulkUpdateStatus(context.Background(), taskIDs, domain.StatusDeveloping)
		require.NoError(t, err)

		// Verify all tasks were updated
		for i, taskID := range taskIDs {
			updated, err := taskRepo.GetByID(context.Background(), taskID)
			require.NoError(t, err, "Failed to get task %d", i)
			assert.Equal(t, domain.StatusDeveloping, updated.Status, "Task %d status not updated", i)
		}
	})

	t.Run("BulkUpdateStatus_EmptySlice_NoOp", func(t *testing.T) {
		err := taskRepo.BulkUpdateStatus(context.Background(), []string{}, domain.StatusDeveloping)
		require.NoError(t, err) // Should be no-op without error
	})

	t.Run("BulkUpdateStatus_InvalidStatus_ReturnsError", func(t *testing.T) {
		taskIDs := []string{"task1", "task2"}
		err := taskRepo.BulkUpdateStatus(context.Background(), taskIDs, domain.TaskStatus("invalid"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid task status")
	})

	t.Run("BulkUpdateStatus_EmptyTaskID_ReturnsError", func(t *testing.T) {
		taskIDs := []string{"validtask123", ""}
		err := taskRepo.BulkUpdateStatus(context.Background(), taskIDs, domain.StatusDeveloping)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "task ID")
		assert.Contains(t, err.Error(), "is empty")
	})

	t.Run("BulkUpdateStatus_NonexistentTask_ReturnsError", func(t *testing.T) {
		taskIDs := []string{"nonexistent123"}
		err := taskRepo.BulkUpdateStatus(context.Background(), taskIDs, domain.StatusDeveloping)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to find task")
	})

	t.Run("BulkUpdateStatus_InvalidTransition_ReturnsError", func(t *testing.T) {
		require.NoError(t, suite.Reset())
		owner, _, _, project1, _ := setupTestData(t, suite)

		// Create task in TODO status
		task := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Invalid Transition Task"),
			WithTaskStatus(domain.StatusTodo),
		)
		require.NoError(t, taskRepo.Create(context.Background(), task))

		// Try to bulk update to invalid transition (TODO -> COMPLETE)
		err := taskRepo.BulkUpdateStatus(context.Background(), []string{task.ID}, domain.StatusComplete)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot transition from")

		// Verify task status unchanged
		unchanged, err := taskRepo.GetByID(context.Background(), task.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.StatusTodo, unchanged.Status)
	})

	t.Run("BulkUpdateStatus_MixedValidInvalidTransitions_FailsAll", func(t *testing.T) {
		require.NoError(t, suite.Reset())
		owner, _, _, project1, _ := setupTestData(t, suite)

		// Create one task with valid transition
		validTask := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Valid Transition Task"),
			WithTaskStatus(domain.StatusTodo),
		)
		require.NoError(t, taskRepo.Create(context.Background(), validTask))

		// Create one task with invalid transition
		invalidTask := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Invalid Transition Task"),
			WithTaskStatus(domain.StatusComplete), // Can't transition COMPLETE -> DEVELOPING
		)
		require.NoError(t, taskRepo.Create(context.Background(), invalidTask))

		// Try bulk update - should fail due to invalid transition
		taskIDs := []string{validTask.ID, invalidTask.ID}
		err := taskRepo.BulkUpdateStatus(context.Background(), taskIDs, domain.StatusDeveloping)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot transition from")

		// Verify neither task was updated (all-or-nothing semantics)
		validTaskCheck, err := taskRepo.GetByID(context.Background(), validTask.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.StatusTodo, validTaskCheck.Status, "Valid task should not be updated if batch fails")

		invalidTaskCheck, err := taskRepo.GetByID(context.Background(), invalidTask.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.StatusComplete, invalidTaskCheck.Status, "Invalid task should remain unchanged")
	})

	// ===========================================
	// GET TASKS BY FILTER TESTS
	// ===========================================

	t.Run("GetTasksByFilter_WithoutProjectScope_ReturnsTasksFromAllProjects", func(t *testing.T) {
		require.NoError(t, suite.Reset())
		owner, _, _, project1, project2 := setupTestData(t, suite)

		// Create tasks in different projects
		task1 := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Project 1 Task"),
			WithTaskStatus(domain.StatusTodo),
		)
		require.NoError(t, taskRepo.Create(context.Background(), task1))

		task2 := suite.Factory.CreateTask(project2, owner,
			WithTaskTitle("Project 2 Task"),
			WithTaskStatus(domain.StatusTodo),
		)
		require.NoError(t, taskRepo.Create(context.Background(), task2))

		task3 := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Another Project 1 Task"),
			WithTaskStatus(domain.StatusDeveloping),
		)
		require.NoError(t, taskRepo.Create(context.Background(), task3))

		// Filter by status across all projects
		filters := repository.TaskFilters{
			Status: []domain.TaskStatus{domain.StatusTodo},
			Limit:  10,
		}
		tasks, err := taskRepo.GetTasksByFilter(context.Background(), filters)
		require.NoError(t, err)
		assert.Len(t, tasks, 2) // Both TODO tasks from both projects

		projectIDs := make([]string, len(tasks))
		for i, task := range tasks {
			projectIDs[i] = task.ProjectID
		}
		assert.Contains(t, projectIDs, project1.ID)
		assert.Contains(t, projectIDs, project2.ID)
	})

	t.Run("GetTasksByFilter_WithReporterFilter_ReturnsTasksByReporter", func(t *testing.T) {
		require.NoError(t, suite.Reset())
		owner, assignee1, _, project1, _ := setupTestData(t, suite)

		// Create tasks by different reporters
		ownerTask := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Owner Created Task"),
		)
		require.NoError(t, taskRepo.Create(context.Background(), ownerTask))

		assigneeTask := suite.Factory.CreateTask(project1, assignee1,
			WithTaskTitle("Assignee Created Task"),
		)
		require.NoError(t, taskRepo.Create(context.Background(), assigneeTask))

		// Filter by reporter
		filters := repository.TaskFilters{
			ReporterID: &assignee1.ID,
			Limit:      10,
		}
		tasks, err := taskRepo.GetTasksByFilter(context.Background(), filters)
		require.NoError(t, err)
		assert.Len(t, tasks, 1)
		assert.Equal(t, "Assignee Created Task", tasks[0].Title)
		assert.Equal(t, assignee1.ID, tasks[0].ReporterID)
	})

	t.Run("GetTasksByFilter_WithParentTaskFilters_ReturnsCorrectTasks", func(t *testing.T) {
		require.NoError(t, suite.Reset())
		owner, _, _, project1, _ := setupTestData(t, suite)

		// Create parent task
		parentTask := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Parent Task"),
		)
		require.NoError(t, taskRepo.Create(context.Background(), parentTask))

		// Create subtask
		subtask := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Subtask"),
			WithTaskParent(parentTask.ID),
		)
		require.NoError(t, taskRepo.Create(context.Background(), subtask))

		// Create standalone task
		standaloneTask := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Standalone Task"),
		)
		require.NoError(t, taskRepo.Create(context.Background(), standaloneTask))

		// Filter for tasks with parents
		hasParentTrue := true
		filters := repository.TaskFilters{
			HasParent: &hasParentTrue,
			Limit:     10,
		}
		tasks, err := taskRepo.GetTasksByFilter(context.Background(), filters)
		require.NoError(t, err)
		assert.Len(t, tasks, 1)
		assert.Equal(t, "Subtask", tasks[0].Title)
		assert.NotNil(t, tasks[0].ParentTaskID)

		// Filter for tasks without parents
		hasParentFalse := false
		filters.HasParent = &hasParentFalse
		tasks, err = taskRepo.GetTasksByFilter(context.Background(), filters)
		require.NoError(t, err)
		assert.Len(t, tasks, 2) // Parent and standalone tasks
		for _, task := range tasks {
			assert.Nil(t, task.ParentTaskID)
		}

		// Filter by specific parent ID
		filters.HasParent = nil
		filters.ParentID = &parentTask.ID
		tasks, err = taskRepo.GetTasksByFilter(context.Background(), filters)
		require.NoError(t, err)
		assert.Len(t, tasks, 1)
		assert.Equal(t, "Subtask", tasks[0].Title)
		assert.Equal(t, parentTask.ID, *tasks[0].ParentTaskID)
	})

	t.Run("GetTasksByFilter_WithLimitExceeding1000_UsesDefault", func(t *testing.T) {
		require.NoError(t, suite.Reset())
		owner, _, _, project1, _ := setupTestData(t, suite)

		// Create a few test tasks
		for i := 0; i < 5; i++ {
			task := suite.Factory.CreateTask(project1, owner,
				WithTaskTitle(fmt.Sprintf("Limit Test Task %d", i+1)),
			)
			require.NoError(t, taskRepo.Create(context.Background(), task))
		}

		// Request with excessive limit
		filters := repository.TaskFilters{
			Limit: 2000, // Exceeds max of 1000
		}
		tasks, err := taskRepo.GetTasksByFilter(context.Background(), filters)
		require.NoError(t, err)
		assert.Len(t, tasks, 5) // Should return all tasks, using default limit behavior
	})

	t.Run("GetTasksByFilter_WithNegativeOffset_UsesZero", func(t *testing.T) {
		require.NoError(t, suite.Reset())
		owner, _, _, project1, _ := setupTestData(t, suite)

		// Create test tasks
		for i := 0; i < 3; i++ {
			task := suite.Factory.CreateTask(project1, owner,
				WithTaskTitle(fmt.Sprintf("Offset Test Task %d", i+1)),
			)
			task.Position = i + 1
			require.NoError(t, taskRepo.Create(context.Background(), task))
		}

		// Request with negative offset
		filters := repository.TaskFilters{
			Offset:    -10,
			Limit:     2,
			SortBy:    repository.SortByPosition,
			SortOrder: repository.SortOrderAsc,
		}
		tasks, err := taskRepo.GetTasksByFilter(context.Background(), filters)
		require.NoError(t, err)
		assert.Len(t, tasks, 2) // Should return first 2 tasks
		assert.Equal(t, "Offset Test Task 1", tasks[0].Title)
		assert.Equal(t, "Offset Test Task 2", tasks[1].Title)
	})

	// ===========================================
	// EDGE CASES AND ERROR SCENARIOS
	// ===========================================

	t.Run("GetByProject_WithEmptyProjectID_ReturnsError", func(t *testing.T) {
		filters := repository.TaskFilters{Limit: 10}
		_, err := taskRepo.GetByProject(context.Background(), "", filters)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project ID cannot be empty")
	})

	t.Run("GetTasksByFilter_WithInvalidSortField_UsesDefault", func(t *testing.T) {
		require.NoError(t, suite.Reset())
		owner, _, _, project1, _ := setupTestData(t, suite)

		// Create test task
		task := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Sort Test Task"),
		)
		require.NoError(t, taskRepo.Create(context.Background(), task))

		// Use invalid sort field
		filters := repository.TaskFilters{
			SortBy: "invalid_field",
			Limit:  10,
		}
		tasks, err := taskRepo.GetTasksByFilter(context.Background(), filters)
		require.NoError(t, err)
		assert.Len(t, tasks, 1)
		// Should not error and return results using default sort
	})

	t.Run("GetTasksByFilter_WithEmptyFilters_ReturnsAllTasks", func(t *testing.T) {
		require.NoError(t, suite.Reset())
		owner, _, _, project1, _ := setupTestData(t, suite)

		// Create test tasks
		for i := 0; i < 3; i++ {
			task := suite.Factory.CreateTask(project1, owner,
				WithTaskTitle(fmt.Sprintf("Empty Filter Task %d", i+1)),
			)
			require.NoError(t, taskRepo.Create(context.Background(), task))
		}

		// Use empty filters
		filters := repository.TaskFilters{
			Limit: 10,
		}
		tasks, err := taskRepo.GetTasksByFilter(context.Background(), filters)
		require.NoError(t, err)
		assert.Len(t, tasks, 3) // Should return all tasks
	})

	t.Run("GetByProject_WithComplexFilterCombinations_HandlesEdgeCases", func(t *testing.T) {
		require.NoError(t, suite.Reset())
		owner, assignee1, _, project1, _ := setupTestData(t, suite)

		now := time.Now()
		pastDate := now.Add(-24 * time.Hour)
		futureDate := now.Add(24 * time.Hour)

		// Create task that matches some but not all filter criteria
		partialMatchTask := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("Partial Match Task"),
			WithTaskStatus(domain.StatusTodo),       // Matches status filter
			WithTaskPriority(domain.PriorityMedium), // Doesn't match priority filter
			WithTaskAssignee(assignee1.ID),          // Matches assignee filter
			WithTaskDueDate(futureDate),             // Doesn't match date filter
		)
		partialMatchTask.Tags = []string{"test", "partial"} // Partially matches tags
		require.NoError(t, taskRepo.Create(context.Background(), partialMatchTask))

		// Create task that doesn't match any criteria
		noMatchTask := suite.Factory.CreateTask(project1, owner,
			WithTaskTitle("No Match Task"),
			WithTaskStatus(domain.StatusComplete), // Different status
			WithTaskPriority(domain.PriorityLow),  // Different priority
			WithTaskDueDate(pastDate),             // Different date range
		)
		noMatchTask.Tags = []string{"different", "unrelated"}
		require.NoError(t, taskRepo.Create(context.Background(), noMatchTask))

		// Apply restrictive filters that require ALL conditions to be met
		filters := repository.TaskFilters{
			Status:     []domain.TaskStatus{domain.StatusTodo},
			Priority:   []domain.TaskPriority{domain.PriorityHigh}, // Won't match partialMatchTask
			AssigneeID: &assignee1.ID,
			Tags:       []string{"urgent"}, // Won't match partialMatchTask
			DueBefore:  &pastDate,          // Won't match partialMatchTask
			Limit:      10,
		}

		tasks, err := taskRepo.GetByProject(context.Background(), project1.ID, filters)
		require.NoError(t, err)
		assert.Empty(t, tasks) // No tasks should match ALL criteria
	})
}
