package api_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/ericfisherdev/simple-easy-tasks/internal/api"
	"github.com/ericfisherdev/simple-easy-tasks/internal/api/middleware"
	"github.com/ericfisherdev/simple-easy-tasks/internal/domain"
	"github.com/ericfisherdev/simple-easy-tasks/internal/repository"
	"github.com/ericfisherdev/simple-easy-tasks/internal/services"
	"github.com/ericfisherdev/simple-easy-tasks/internal/testutil"

	"github.com/gin-gonic/gin"
)

func TestTaskHandler_ListTasks(t *testing.T) {
	tests := []testutil.TestCase{
		{
			Name:           "list tasks successfully",
			Method:         "GET",
			URL:            "/api/projects/project-1/tasks",
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:           "list tasks with status filter",
			Method:         "GET",
			URL:            "/api/projects/project-1/tasks?status=todo",
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:           "list tasks with pagination",
			Method:         "GET",
			URL:            "/api/projects/project-1/tasks?limit=10&offset=0",
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:           "list tasks with priority filter",
			Method:         "GET",
			URL:            "/api/projects/project-1/tasks?priority=high",
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:           "list tasks with assignee filter",
			Method:         "GET",
			URL:            "/api/projects/project-1/tasks?assignee=user-1",
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:           "list tasks with search",
			Method:         "GET",
			URL:            "/api/projects/project-1/tasks?search=test",
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:           "missing project ID",
			Method:         "GET",
			URL:            "/api/projects//tasks",
			ExpectedStatus: http.StatusBadRequest,
		},
		{
			Name:           "project not found",
			Method:         "GET",
			URL:            "/api/projects/non-existent/tasks",
			ExpectedStatus: http.StatusNotFound,
		},
	}

	router := setupTaskTestRouter(t)
	helper := testutil.NewHTTPTestHelper(t, router)

	for _, tc := range tests {
		t.Run(tc.Name, func(_ *testing.T) {
			headers := map[string]string{
				"Authorization": "Bearer mock-token",
			}

			recorder := helper.Request(tc.Method, tc.URL, tc.Body, headers)
			helper.AssertStatus(recorder, tc.ExpectedStatus)

			if tc.ExpectedStatus == http.StatusOK {
				responseBody := recorder.Body.String()
				if !contains(responseBody, "success") || !contains(responseBody, "true") {
					t.Error("Expected success response")
				}
				if !contains(responseBody, "tasks") {
					t.Error("Expected tasks in response")
				}
			}
		})
	}
}

func TestTaskHandler_CreateTask(t *testing.T) {
	tests := []testutil.TestCase{
		{
			Name:   "successful task creation",
			Method: "POST",
			URL:    "/api/projects/project-1/tasks",
			Body: map[string]interface{}{
				"title":       "New Test Task",
				"description": "A new test task",
				"priority":    "medium",
				"assignee_id": "user-1",
			},
			ExpectedStatus: http.StatusCreated,
		},
		{
			Name:   "create task with minimal fields",
			Method: "POST",
			URL:    "/api/projects/project-1/tasks",
			Body: map[string]interface{}{
				"title": "Minimal Task",
			},
			ExpectedStatus: http.StatusCreated,
		},
		{
			Name:   "create task with due date",
			Method: "POST",
			URL:    "/api/projects/project-1/tasks",
			Body: map[string]interface{}{
				"title":    "Task with Due Date",
				"due_date": time.Now().Add(7 * 24 * time.Hour).Format(time.RFC3339),
			},
			ExpectedStatus: http.StatusCreated,
		},
		{
			Name:   "invalid request body - missing title",
			Method: "POST",
			URL:    "/api/projects/project-1/tasks",
			Body: map[string]interface{}{
				"description": "Task without title",
			},
			ExpectedStatus: http.StatusBadRequest,
		},
		{
			Name:   "invalid request body - empty title",
			Method: "POST",
			URL:    "/api/projects/project-1/tasks",
			Body: map[string]interface{}{
				"title": "",
			},
			ExpectedStatus: http.StatusBadRequest,
		},
		{
			Name:           "missing project ID",
			Method:         "POST",
			URL:            "/api/projects//tasks",
			Body:           map[string]interface{}{"title": "Test"},
			ExpectedStatus: http.StatusBadRequest,
		},
		{
			Name:           "missing request body",
			Method:         "POST",
			URL:            "/api/projects/project-1/tasks",
			Body:           nil,
			ExpectedStatus: http.StatusBadRequest,
		},
		{
			Name:           "invalid JSON",
			Method:         "POST",
			URL:            "/api/projects/project-1/tasks",
			Body:           "invalid json",
			ExpectedStatus: http.StatusBadRequest,
		},
	}

	router := setupTaskTestRouter(t)
	helper := testutil.NewHTTPTestHelper(t, router)

	for _, tc := range tests {
		t.Run(tc.Name, func(_ *testing.T) {
			headers := map[string]string{
				"Authorization": "Bearer mock-token",
			}

			recorder := helper.Request(tc.Method, tc.URL, tc.Body, headers)
			helper.AssertStatus(recorder, tc.ExpectedStatus)

			if tc.ExpectedStatus == http.StatusCreated {
				responseBody := recorder.Body.String()
				if !contains(responseBody, "success") || !contains(responseBody, "true") {
					t.Error("Expected success response")
				}
				if !contains(responseBody, "task") {
					t.Error("Expected task in response")
				}
			}
		})
	}
}

func TestTaskHandler_GetTask(t *testing.T) {
	tests := []testutil.TestCase{
		{
			Name:           "get existing task",
			Method:         "GET",
			URL:            "/api/projects/project-1/tasks/task-1",
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:           "get non-existent task",
			Method:         "GET",
			URL:            "/api/projects/project-1/tasks/non-existent",
			ExpectedStatus: http.StatusNotFound,
		},
		{
			Name:           "task from different project",
			Method:         "GET",
			URL:            "/api/projects/project-2/tasks/task-1",
			ExpectedStatus: http.StatusNotFound,
		},
		{
			Name:           "missing project ID",
			Method:         "GET",
			URL:            "/api/projects//tasks/task-1",
			ExpectedStatus: http.StatusBadRequest,
		},
		{
			Name:           "missing task ID",
			Method:         "GET",
			URL:            "/api/projects/project-1/tasks/",
			ExpectedStatus: http.StatusMovedPermanently,
		},
	}

	router := setupTaskTestRouter(t)
	helper := testutil.NewHTTPTestHelper(t, router)

	for _, tc := range tests {
		t.Run(tc.Name, func(_ *testing.T) {
			headers := map[string]string{
				"Authorization": "Bearer mock-token",
			}

			recorder := helper.Request(tc.Method, tc.URL, tc.Body, headers)
			helper.AssertStatus(recorder, tc.ExpectedStatus)

			if tc.ExpectedStatus == http.StatusOK {
				responseBody := recorder.Body.String()
				if !contains(responseBody, "success") || !contains(responseBody, "true") {
					t.Error("Expected success response")
				}
				if !contains(responseBody, "task") {
					t.Error("Expected task in response")
				}
			}
		})
	}
}

func TestTaskHandler_UpdateTask(t *testing.T) {
	tests := []testutil.TestCase{
		{
			Name:   "update task title",
			Method: "PUT",
			URL:    "/api/projects/project-1/tasks/task-1",
			Body: map[string]interface{}{
				"title": "Updated Task Title",
			},
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:   "update task status",
			Method: "PUT",
			URL:    "/api/projects/project-1/tasks/task-1",
			Body: map[string]interface{}{
				"status": "developing",
			},
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:   "update task priority",
			Method: "PUT",
			URL:    "/api/projects/project-1/tasks/task-1",
			Body: map[string]interface{}{
				"priority": "high",
			},
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:   "update multiple fields",
			Method: "PUT",
			URL:    "/api/projects/project-1/tasks/task-1",
			Body: map[string]interface{}{
				"title":       "Updated Title",
				"description": "Updated Description",
				"priority":    "low",
			},
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:           "update non-existent task",
			Method:         "PUT",
			URL:            "/api/projects/project-1/tasks/non-existent",
			Body:           map[string]interface{}{"title": "Updated"},
			ExpectedStatus: http.StatusNotFound,
		},
		{
			Name:           "missing project ID",
			Method:         "PUT",
			URL:            "/api/projects//tasks/task-1",
			Body:           map[string]interface{}{"title": "Updated"},
			ExpectedStatus: http.StatusBadRequest,
		},
		{
			Name:           "missing task ID",
			Method:         "PUT",
			URL:            "/api/projects/project-1/tasks/",
			Body:           map[string]interface{}{"title": "Updated"},
			ExpectedStatus: http.StatusNotFound,
		},
		{
			Name:           "invalid JSON",
			Method:         "PUT",
			URL:            "/api/projects/project-1/tasks/task-1",
			Body:           "invalid json",
			ExpectedStatus: http.StatusBadRequest,
		},
	}

	router := setupTaskTestRouter(t)
	helper := testutil.NewHTTPTestHelper(t, router)

	for _, tc := range tests {
		t.Run(tc.Name, func(_ *testing.T) {
			headers := map[string]string{
				"Authorization": "Bearer mock-token",
			}

			recorder := helper.Request(tc.Method, tc.URL, tc.Body, headers)
			helper.AssertStatus(recorder, tc.ExpectedStatus)

			if tc.ExpectedStatus == http.StatusOK {
				responseBody := recorder.Body.String()
				if !contains(responseBody, "success") || !contains(responseBody, "true") {
					t.Error("Expected success response")
				}
			}
		})
	}
}

func TestTaskHandler_DeleteTask(t *testing.T) {
	tests := []testutil.TestCase{
		{
			Name:           "delete existing task",
			Method:         "DELETE",
			URL:            "/api/projects/project-1/tasks/task-1",
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:           "delete non-existent task",
			Method:         "DELETE",
			URL:            "/api/projects/project-1/tasks/non-existent",
			ExpectedStatus: http.StatusNotFound,
		},
		{
			Name:           "missing project ID",
			Method:         "DELETE",
			URL:            "/api/projects//tasks/task-1",
			ExpectedStatus: http.StatusBadRequest,
		},
		{
			Name:           "missing task ID",
			Method:         "DELETE",
			URL:            "/api/projects/project-1/tasks/",
			ExpectedStatus: http.StatusNotFound,
		},
	}

	router := setupTaskTestRouter(t)
	helper := testutil.NewHTTPTestHelper(t, router)

	for _, tc := range tests {
		t.Run(tc.Name, func(_ *testing.T) {
			headers := map[string]string{
				"Authorization": "Bearer mock-token",
			}

			recorder := helper.Request(tc.Method, tc.URL, tc.Body, headers)
			helper.AssertStatus(recorder, tc.ExpectedStatus)

			if tc.ExpectedStatus == http.StatusOK {
				responseBody := recorder.Body.String()
				if !contains(responseBody, "success") || !contains(responseBody, "true") {
					t.Error("Expected success response")
				}
			}
		})
	}
}

func TestTaskHandler_MoveTask(t *testing.T) {
	tests := []testutil.TestCase{
		{
			Name:   "move task to different status",
			Method: "POST",
			URL:    "/api/projects/project-1/tasks/task-1/move",
			Body: map[string]interface{}{
				"new_status":   "developing",
				"new_position": 1,
			},
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:   "move task with position only",
			Method: "POST",
			URL:    "/api/projects/project-1/tasks/task-1/move",
			Body: map[string]interface{}{
				"new_status":   "todo",
				"new_position": 5,
			},
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:   "invalid status",
			Method: "POST",
			URL:    "/api/projects/project-1/tasks/task-1/move",
			Body: map[string]interface{}{
				"new_status":   "invalid",
				"new_position": 1,
			},
			ExpectedStatus: http.StatusBadRequest,
		},
		{
			Name:   "missing required fields",
			Method: "POST",
			URL:    "/api/projects/project-1/tasks/task-1/move",
			Body: map[string]interface{}{
				"new_position": 1,
			},
			ExpectedStatus: http.StatusBadRequest,
		},
		{
			Name:           "move non-existent task",
			Method:         "POST",
			URL:            "/api/projects/project-1/tasks/non-existent/move",
			Body:           map[string]interface{}{"new_status": "todo", "new_position": 1},
			ExpectedStatus: http.StatusNotFound,
		},
		{
			Name:           "missing project ID",
			Method:         "POST",
			URL:            "/api/projects//tasks/task-1/move",
			Body:           map[string]interface{}{"new_status": "todo", "new_position": 1},
			ExpectedStatus: http.StatusBadRequest,
		},
	}

	router := setupTaskTestRouter(t)
	helper := testutil.NewHTTPTestHelper(t, router)

	for _, tc := range tests {
		t.Run(tc.Name, func(_ *testing.T) {
			headers := map[string]string{
				"Authorization": "Bearer mock-token",
			}

			recorder := helper.Request(tc.Method, tc.URL, tc.Body, headers)
			helper.AssertStatus(recorder, tc.ExpectedStatus)

			if tc.ExpectedStatus == http.StatusOK {
				responseBody := recorder.Body.String()
				if !contains(responseBody, "success") || !contains(responseBody, "true") {
					t.Error("Expected success response")
				}
			}
		})
	}
}

func TestTaskHandler_UpdateTaskStatus(t *testing.T) {
	tests := []testutil.TestCase{
		{
			Name:   "update task status",
			Method: "PUT",
			URL:    "/api/projects/project-1/tasks/task-1/status",
			Body: map[string]interface{}{
				"status": "developing",
			},
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:   "update to complete status",
			Method: "PUT",
			URL:    "/api/projects/project-1/tasks/task-1/status",
			Body: map[string]interface{}{
				"status": "complete",
			},
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:   "invalid status",
			Method: "PUT",
			URL:    "/api/projects/project-1/tasks/task-1/status",
			Body: map[string]interface{}{
				"status": "invalid",
			},
			ExpectedStatus: http.StatusBadRequest,
		},
		{
			Name:           "missing status field",
			Method:         "PUT",
			URL:            "/api/projects/project-1/tasks/task-1/status",
			Body:           map[string]interface{}{},
			ExpectedStatus: http.StatusBadRequest,
		},
		{
			Name:           "update status of non-existent task",
			Method:         "PUT",
			URL:            "/api/projects/project-1/tasks/non-existent/status",
			Body:           map[string]interface{}{"status": "todo"},
			ExpectedStatus: http.StatusNotFound,
		},
	}

	router := setupTaskTestRouter(t)
	helper := testutil.NewHTTPTestHelper(t, router)

	for _, tc := range tests {
		t.Run(tc.Name, func(_ *testing.T) {
			headers := map[string]string{
				"Authorization": "Bearer mock-token",
			}

			recorder := helper.Request(tc.Method, tc.URL, tc.Body, headers)
			helper.AssertStatus(recorder, tc.ExpectedStatus)
		})
	}
}

func TestTaskHandler_AssignTask(t *testing.T) {
	tests := []testutil.TestCase{
		{
			Name:   "assign task to user",
			Method: "POST",
			URL:    "/api/projects/project-1/tasks/task-1/assign",
			Body: map[string]interface{}{
				"assignee_id": "user-2",
			},
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:           "missing assignee ID",
			Method:         "POST",
			URL:            "/api/projects/project-1/tasks/task-1/assign",
			Body:           map[string]interface{}{},
			ExpectedStatus: http.StatusBadRequest,
		},
		{
			Name:           "assign non-existent task",
			Method:         "POST",
			URL:            "/api/projects/project-1/tasks/non-existent/assign",
			Body:           map[string]interface{}{"assignee_id": "user-2"},
			ExpectedStatus: http.StatusNotFound,
		},
	}

	router := setupTaskTestRouter(t)
	helper := testutil.NewHTTPTestHelper(t, router)

	for _, tc := range tests {
		t.Run(tc.Name, func(_ *testing.T) {
			headers := map[string]string{
				"Authorization": "Bearer mock-token",
			}

			recorder := helper.Request(tc.Method, tc.URL, tc.Body, headers)
			helper.AssertStatus(recorder, tc.ExpectedStatus)
		})
	}
}

func TestTaskHandler_UnassignTask(t *testing.T) {
	tests := []testutil.TestCase{
		{
			Name:           "unassign task",
			Method:         "DELETE",
			URL:            "/api/projects/project-1/tasks/task-1/assign",
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:           "unassign non-existent task",
			Method:         "DELETE",
			URL:            "/api/projects/project-1/tasks/non-existent/assign",
			ExpectedStatus: http.StatusNotFound,
		},
	}

	router := setupTaskTestRouter(t)
	helper := testutil.NewHTTPTestHelper(t, router)

	for _, tc := range tests {
		t.Run(tc.Name, func(_ *testing.T) {
			headers := map[string]string{
				"Authorization": "Bearer mock-token",
			}

			recorder := helper.Request(tc.Method, tc.URL, tc.Body, headers)
			helper.AssertStatus(recorder, tc.ExpectedStatus)
		})
	}
}

func TestTaskHandler_DuplicateTask(t *testing.T) {
	tests := []testutil.TestCase{
		{
			Name:           "duplicate task with default options",
			Method:         "POST",
			URL:            "/api/projects/project-1/tasks/task-1/duplicate",
			Body:           map[string]interface{}{},
			ExpectedStatus: http.StatusCreated,
		},
		{
			Name:   "duplicate task with custom options",
			Method: "POST",
			URL:    "/api/projects/project-1/tasks/task-1/duplicate",
			Body: map[string]interface{}{
				"new_title":           "Duplicated Task",
				"include_subtasks":    true,
				"include_attachments": false,
				"reset_progress":      true,
			},
			ExpectedStatus: http.StatusCreated,
		},
		{
			Name:           "duplicate non-existent task",
			Method:         "POST",
			URL:            "/api/projects/project-1/tasks/non-existent/duplicate",
			Body:           map[string]interface{}{},
			ExpectedStatus: http.StatusNotFound,
		},
	}

	router := setupTaskTestRouter(t)
	helper := testutil.NewHTTPTestHelper(t, router)

	for _, tc := range tests {
		t.Run(tc.Name, func(_ *testing.T) {
			headers := map[string]string{
				"Authorization": "Bearer mock-token",
			}

			recorder := helper.Request(tc.Method, tc.URL, tc.Body, headers)
			helper.AssertStatus(recorder, tc.ExpectedStatus)

			if tc.ExpectedStatus == http.StatusCreated {
				responseBody := recorder.Body.String()
				if !contains(responseBody, "success") || !contains(responseBody, "true") {
					t.Error("Expected success response")
				}
			}
		})
	}
}

func TestTaskHandler_GetTaskHistory(t *testing.T) {
	tests := []testutil.TestCase{
		{
			Name:           "get task history",
			Method:         "GET",
			URL:            "/api/projects/project-1/tasks/task-1/history",
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:           "get task history with pagination",
			Method:         "GET",
			URL:            "/api/projects/project-1/tasks/task-1/history?limit=10&offset=0",
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:           "get history of non-existent task",
			Method:         "GET",
			URL:            "/api/projects/project-1/tasks/non-existent/history",
			ExpectedStatus: http.StatusNotFound,
		},
	}

	router := setupTaskTestRouter(t)
	helper := testutil.NewHTTPTestHelper(t, router)

	for _, tc := range tests {
		t.Run(tc.Name, func(_ *testing.T) {
			headers := map[string]string{
				"Authorization": "Bearer mock-token",
			}

			recorder := helper.Request(tc.Method, tc.URL, tc.Body, headers)
			helper.AssertStatus(recorder, tc.ExpectedStatus)

			if tc.ExpectedStatus == http.StatusOK {
				responseBody := recorder.Body.String()
				if !contains(responseBody, "history") {
					t.Error("Expected history in response")
				}
			}
		})
	}
}

func TestTaskHandler_LogTimeSpent(t *testing.T) {
	tests := []testutil.TestCase{
		{
			Name:   "log time spent",
			Method: "POST",
			URL:    "/api/projects/project-1/tasks/task-1/time-log",
			Body: map[string]interface{}{
				"hours":       2.5,
				"description": "Worked on feature implementation",
			},
			ExpectedStatus: http.StatusCreated,
		},
		{
			Name:   "log time with logged_at timestamp",
			Method: "POST",
			URL:    "/api/projects/project-1/tasks/task-1/time-log",
			Body: map[string]interface{}{
				"hours":     1.0,
				"logged_at": time.Now().Format(time.RFC3339),
			},
			ExpectedStatus: http.StatusCreated,
		},
		{
			Name:   "invalid hours - negative",
			Method: "POST",
			URL:    "/api/projects/project-1/tasks/task-1/time-log",
			Body: map[string]interface{}{
				"hours": -1.0,
			},
			ExpectedStatus: http.StatusBadRequest,
		},
		{
			Name:   "missing hours field",
			Method: "POST",
			URL:    "/api/projects/project-1/tasks/task-1/time-log",
			Body: map[string]interface{}{
				"description": "Missing hours",
			},
			ExpectedStatus: http.StatusBadRequest,
		},
		{
			Name:           "log time for non-existent task",
			Method:         "POST",
			URL:            "/api/projects/project-1/tasks/non-existent/time-log",
			Body:           map[string]interface{}{"hours": 2.0},
			ExpectedStatus: http.StatusNotFound,
		},
	}

	router := setupTaskTestRouter(t)
	helper := testutil.NewHTTPTestHelper(t, router)

	for _, tc := range tests {
		t.Run(tc.Name, func(_ *testing.T) {
			headers := map[string]string{
				"Authorization": "Bearer mock-token",
			}

			recorder := helper.Request(tc.Method, tc.URL, tc.Body, headers)
			helper.AssertStatus(recorder, tc.ExpectedStatus)

			if tc.ExpectedStatus == http.StatusCreated {
				responseBody := recorder.Body.String()
				if !contains(responseBody, "time_log") {
					t.Error("Expected time_log in response")
				}
			}
		})
	}
}

// setupTaskTestRouter creates a test router with task endpoints and mock dependencies.
func setupTaskTestRouter(_ *testing.T) *gin.Engine {
	router := testutil.NewTestRouter()

	// Create mock repositories
	taskRepo := testutil.NewMockTaskRepository()
	projectRepo := testutil.NewMockProjectRepository()
	userRepo := testutil.NewMockUserRepository()

	// Add test data
	testUser := testutil.MockUser("user-1", "test@example.com", "testuser", "Test User")
	testUser2 := testutil.MockUser("user-2", "test2@example.com", "testuser2", "Test User 2")
	testProject := testutil.MockProject("project-1", "Test Project", "test-project", "user-1")
	testTask := testutil.MockTask("task-1", "Test Task", "project-1", "user-1")
	testTask2 := testutil.MockTask("task-2", "Test Task 2", "project-1", "user-1")
	testTask3 := testutil.MockTask("task-3", "Test Task 3", "project-1", "user-1")

	// Set up dependency relationship for testing (task-1 depends on task-2)
	testTask.Dependencies = []string{"task-2"}

	userRepo.AddUser(testUser)
	userRepo.AddUser(testUser2)
	projectRepo.AddProject(testProject)
	taskRepo.AddTask(testTask)
	taskRepo.AddTask(testTask2)
	taskRepo.AddTask(testTask3)

	// Create mock services
	mockAuthService := &MockAuthService{
		user: testUser,
	}

	mockTaskService := &MockTaskService{
		tasks:       []*domain.Task{testTask, testTask2, testTask3},
		projectRepo: projectRepo,
		userRepo:    userRepo,
	}

	// Create middleware
	authMiddleware := middleware.NewAuthMiddleware(mockAuthService)

	// Setup routes
	taskHandler := api.NewTaskHandler(mockTaskService, taskRepo)

	apiGroup := router.Group("/api")
	taskHandler.RegisterRoutes(apiGroup, authMiddleware)

	return router
}

// MockTaskService is a mock implementation of TaskService for testing.
type MockTaskService struct {
	tasks       []*domain.Task
	projectRepo *testutil.MockProjectRepository
	userRepo    *testutil.MockUserRepository
}

func (m *MockTaskService) CreateTask(
	_ context.Context, req domain.CreateTaskRequest, userID string,
) (*domain.Task, error) {
	// Basic validation
	if req.Title == "" {
		return nil, domain.NewValidationError("INVALID_TITLE", "Title is required", nil)
	}

	// Check project exists
	_, err := m.projectRepo.GetByID(context.Background(), req.ProjectID)
	if err != nil {
		return nil, domain.NewNotFoundError("PROJECT_NOT_FOUND", "Project not found")
	}

	task := &domain.Task{
		ID:          "new-task-id",
		Title:       req.Title,
		Description: req.Description,
		ProjectID:   req.ProjectID,
		ReporterID:  userID,
		Status:      domain.StatusTodo,
		Priority:    req.Priority,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	m.tasks = append(m.tasks, task)
	return task, nil
}

func (m *MockTaskService) GetTask(_ context.Context, taskID string, _ string) (*domain.Task, error) {
	for _, task := range m.tasks {
		if task.ID == taskID {
			return task, nil
		}
	}
	return nil, domain.NewNotFoundError("TASK_NOT_FOUND", "Task not found")
}

func (m *MockTaskService) UpdateTask(
	_ context.Context, taskID string, req domain.UpdateTaskRequest, _ string,
) (*domain.Task, error) {
	for _, task := range m.tasks {
		if task.ID == taskID {
			if req.Title != nil {
				task.Title = *req.Title
			}
			if req.Description != nil {
				task.Description = *req.Description
			}
			if req.Status != nil {
				task.Status = *req.Status
			}
			if req.Priority != nil {
				task.Priority = *req.Priority
			}
			task.UpdatedAt = time.Now()
			return task, nil
		}
	}
	return nil, domain.NewNotFoundError("TASK_NOT_FOUND", "Task not found")
}

func (m *MockTaskService) DeleteTask(_ context.Context, taskID string, _ string) error {
	for i, task := range m.tasks {
		if task.ID == taskID {
			// Remove from slice
			m.tasks = append(m.tasks[:i], m.tasks[i+1:]...)
			return nil
		}
	}
	return domain.NewNotFoundError("TASK_NOT_FOUND", "Task not found")
}

func (m *MockTaskService) ListProjectTasks(
	_ context.Context, projectID string, _ string, _, _ int,
) ([]*domain.Task, error) {
	var projectTasks []*domain.Task
	for _, task := range m.tasks {
		if task.ProjectID == projectID {
			projectTasks = append(projectTasks, task)
		}
	}
	return projectTasks, nil
}

func (m *MockTaskService) ListUserTasks(_ context.Context, _ string, _, _ int) ([]*domain.Task, error) {
	return m.tasks, nil
}

func (m *MockTaskService) AssignTask(
	_ context.Context, taskID string, assigneeID string, _ string,
) (*domain.Task, error) {
	for _, task := range m.tasks {
		if task.ID == taskID {
			task.AssigneeID = &assigneeID
			task.UpdatedAt = time.Now()
			return task, nil
		}
	}
	return nil, domain.NewNotFoundError("TASK_NOT_FOUND", "Task not found")
}

func (m *MockTaskService) UnassignTask(_ context.Context, taskID string, _ string) (*domain.Task, error) {
	for _, task := range m.tasks {
		if task.ID == taskID {
			task.AssigneeID = nil
			task.UpdatedAt = time.Now()
			return task, nil
		}
	}
	return nil, domain.NewNotFoundError("TASK_NOT_FOUND", "Task not found")
}

func (m *MockTaskService) UpdateTaskStatus(
	_ context.Context, taskID string, status domain.TaskStatus, _ string,
) (*domain.Task, error) {
	for _, task := range m.tasks {
		if task.ID == taskID {
			task.Status = status
			task.UpdatedAt = time.Now()
			return task, nil
		}
	}
	return nil, domain.NewNotFoundError("TASK_NOT_FOUND", "Task not found")
}

func (m *MockTaskService) MoveTask(_ context.Context, req services.MoveTaskRequest, _ string) error {
	// Validate status
	if !req.NewStatus.IsValid() {
		return domain.NewValidationError("INVALID_STATUS", "Invalid task status", nil)
	}

	for _, task := range m.tasks {
		if task.ID == req.TaskID && task.ProjectID == req.ProjectID {
			task.Status = req.NewStatus
			task.Position = req.NewPosition
			task.UpdatedAt = time.Now()
			return nil
		}
	}
	return domain.NewNotFoundError("TASK_NOT_FOUND", "Task not found")
}

func (m *MockTaskService) GetProjectTasksFiltered(
	ctx context.Context, projectID string, _ repository.TaskFilters, _ string,
) ([]*domain.Task, error) {
	// Check if project exists
	_, err := m.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, domain.NewNotFoundError("PROJECT_NOT_FOUND", "Project not found")
	}

	var filteredTasks []*domain.Task
	for _, task := range m.tasks {
		if task.ProjectID == projectID {
			filteredTasks = append(filteredTasks, task)
		}
	}
	return filteredTasks, nil
}

func (m *MockTaskService) GetSubtasks(_ context.Context, _ string, _ string) ([]*domain.Task, error) {
	return []*domain.Task{}, nil
}

func (m *MockTaskService) GetTaskDependencies(_ context.Context, _ string, _ string) ([]*domain.Task, error) {
	return []*domain.Task{}, nil
}

func (m *MockTaskService) DuplicateTask(
	_ context.Context, taskID string, options services.DuplicationOptions, userID string,
) (*domain.Task, error) {
	for _, task := range m.tasks {
		if task.ID == taskID {
			newTitle := options.NewTitle
			if newTitle == "" {
				newTitle = "Copy of " + task.Title
			}

			newTask := &domain.Task{
				ID:          "duplicated-task-id",
				Title:       newTitle,
				Description: task.Description,
				ProjectID:   task.ProjectID,
				ReporterID:  userID,
				Status:      domain.StatusTodo,
				Priority:    task.Priority,
				Progress:    0,
				TimeSpent:   0.0,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
			m.tasks = append(m.tasks, newTask)
			return newTask, nil
		}
	}
	return nil, domain.NewNotFoundError("TASK_NOT_FOUND", "Task not found")
}

func (m *MockTaskService) CreateFromTemplate(_ context.Context, _ string, _ string, _ string) (*domain.Task, error) {
	return nil, domain.NewInternalError("NOT_IMPLEMENTED", "Not implemented in mock", nil)
}

func (m *MockTaskService) CreateSubtask(
	_ context.Context, parentTaskID string, req domain.CreateTaskRequest, userID string,
) (*domain.Task, error) {
	// Basic validation
	if req.Title == "" {
		return nil, domain.NewValidationError("INVALID_TITLE", "Title is required", nil)
	}

	// Find parent task
	var parentTask *domain.Task
	for _, task := range m.tasks {
		if task.ID == parentTaskID {
			parentTask = task
			break
		}
	}

	if parentTask == nil {
		return nil, domain.NewNotFoundError("PARENT_TASK_NOT_FOUND", "Parent task not found")
	}

	// Create subtask
	subtask := &domain.Task{
		ID:           "subtask-" + parentTaskID + "-1",
		Title:        req.Title,
		Description:  req.Description,
		ProjectID:    parentTask.ProjectID,
		ParentTaskID: &parentTaskID,
		Status:       domain.StatusTodo,
		Priority:     domain.PriorityMedium,
		ReporterID:   userID,
		Progress:     0,
		TimeSpent:    0.0,
		Position:     0,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Add to tasks slice
	m.tasks = append(m.tasks, subtask)

	return subtask, nil
}

func (m *MockTaskService) AddDependency(_ context.Context, taskID string, dependencyID string, _ string) error {
	// Find the task
	var task *domain.Task
	for _, t := range m.tasks {
		if t.ID == taskID {
			task = t
			break
		}
	}

	if task == nil {
		return domain.NewNotFoundError("TASK_NOT_FOUND", "Task not found")
	}

	// Find the dependency task
	var dependencyTask *domain.Task
	for _, t := range m.tasks {
		if t.ID == dependencyID {
			dependencyTask = t
			break
		}
	}

	if dependencyTask == nil {
		return domain.NewNotFoundError("DEPENDENCY_NOT_FOUND", "Dependency task not found")
	}

	// Prevent circular dependencies
	if taskID == dependencyID {
		return domain.NewConflictError("CIRCULAR_DEPENDENCY", "Task cannot depend on itself")
	}

	// Check if dependency already exists
	for _, depID := range task.Dependencies {
		if depID == dependencyID {
			return domain.NewConflictError("DEPENDENCY_EXISTS", "Dependency already exists")
		}
	}

	// Add dependency
	task.Dependencies = append(task.Dependencies, dependencyID)
	task.UpdatedAt = time.Now()

	return nil
}

func (m *MockTaskService) RemoveDependency(_ context.Context, taskID string, dependencyID string, _ string) error {
	// Find the task
	var task *domain.Task
	for _, t := range m.tasks {
		if t.ID == taskID {
			task = t
			break
		}
	}

	if task == nil {
		return domain.NewNotFoundError("TASK_NOT_FOUND", "Task not found")
	}

	// Find and remove the dependency
	dependencyIndex := -1
	for i, depID := range task.Dependencies {
		if depID == dependencyID {
			dependencyIndex = i
			break
		}
	}

	if dependencyIndex == -1 {
		return domain.NewNotFoundError("DEPENDENCY_NOT_FOUND", "Dependency not found")
	}

	// Remove dependency from slice
	task.Dependencies = append(task.Dependencies[:dependencyIndex], task.Dependencies[dependencyIndex+1:]...)
	task.UpdatedAt = time.Now()

	return nil
}

func TestTaskHandler_CreateSubtask(t *testing.T) {
	tests := []testutil.TestCase{
		{
			Name:   "successful subtask creation",
			Method: "POST",
			URL:    "/api/projects/project-1/tasks/task-1/subtasks",
			Body: map[string]interface{}{
				"title":       "Test Subtask",
				"description": "Test subtask description",
			},
			ExpectedStatus: http.StatusCreated,
		},
		{
			Name:           "missing title",
			Method:         "POST",
			URL:            "/api/projects/project-1/tasks/task-1/subtasks",
			Body:           map[string]interface{}{},
			ExpectedStatus: http.StatusBadRequest,
		},
		{
			Name:           "missing parent task ID",
			Method:         "POST",
			URL:            "/api/projects/project-1/tasks//subtasks",
			ExpectedStatus: http.StatusBadRequest,
		},
		{
			Name:   "parent task not found",
			Method: "POST",
			URL:    "/api/projects/project-1/tasks/non-existent/subtasks",
			Body: map[string]interface{}{
				"title": "Test Subtask",
			},
			ExpectedStatus: http.StatusNotFound,
		},
	}

	router := setupTaskTestRouter(t)
	helper := testutil.NewHTTPTestHelper(t, router)

	for _, tc := range tests {
		t.Run(tc.Name, func(_ *testing.T) {
			headers := map[string]string{
				"Authorization": "Bearer mock-token",
			}

			recorder := helper.Request(tc.Method, tc.URL, tc.Body, headers)
			helper.AssertStatus(recorder, tc.ExpectedStatus)

			if tc.ExpectedStatus == http.StatusCreated {
				responseBody := recorder.Body.String()
				if !contains(responseBody, "success") || !contains(responseBody, "true") {
					t.Error("Expected success response")
				}
				if !contains(responseBody, "task") {
					t.Error("Expected task in response")
				}
			}
		})
	}
}

func TestTaskHandler_ListSubtasks(t *testing.T) {
	tests := []testutil.TestCase{
		{
			Name:           "list subtasks successfully",
			Method:         "GET",
			URL:            "/api/projects/project-1/tasks/task-1/subtasks",
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:           "missing parent task ID",
			Method:         "GET",
			URL:            "/api/projects/project-1/tasks//subtasks",
			ExpectedStatus: http.StatusBadRequest,
		},
		{
			Name:           "parent task not found",
			Method:         "GET",
			URL:            "/api/projects/project-1/tasks/non-existent/subtasks",
			ExpectedStatus: http.StatusNotFound,
		},
	}

	router := setupTaskTestRouter(t)
	helper := testutil.NewHTTPTestHelper(t, router)

	for _, tc := range tests {
		t.Run(tc.Name, func(_ *testing.T) {
			headers := map[string]string{
				"Authorization": "Bearer mock-token",
			}

			recorder := helper.Request(tc.Method, tc.URL, tc.Body, headers)
			helper.AssertStatus(recorder, tc.ExpectedStatus)

			if tc.ExpectedStatus == http.StatusOK {
				responseBody := recorder.Body.String()
				if !contains(responseBody, "success") || !contains(responseBody, "true") {
					t.Error("Expected success response")
				}
				if !contains(responseBody, "subtasks") {
					t.Error("Expected subtasks in response")
				}
			}
		})
	}
}

func TestTaskHandler_AddDependency(t *testing.T) {
	tests := []testutil.TestCase{
		{
			Name:   "successfully add dependency",
			Method: "POST",
			URL:    "/api/projects/project-1/tasks/task-1/dependencies",
			Body: map[string]interface{}{
				"dependency_id": "task-3",
			},
			ExpectedStatus: http.StatusCreated,
		},
		{
			Name:           "missing dependency ID",
			Method:         "POST",
			URL:            "/api/projects/project-1/tasks/task-1/dependencies",
			Body:           map[string]interface{}{},
			ExpectedStatus: http.StatusBadRequest,
		},
		{
			Name:           "missing task ID",
			Method:         "POST",
			URL:            "/api/projects/project-1/tasks//dependencies",
			ExpectedStatus: http.StatusBadRequest,
		},
		{
			Name:   "task not found",
			Method: "POST",
			URL:    "/api/projects/project-1/tasks/non-existent/dependencies",
			Body: map[string]interface{}{
				"dependency_id": "task-3",
			},
			ExpectedStatus: http.StatusNotFound,
		},
	}

	router := setupTaskTestRouter(t)
	helper := testutil.NewHTTPTestHelper(t, router)

	for _, tc := range tests {
		t.Run(tc.Name, func(_ *testing.T) {
			headers := map[string]string{
				"Authorization": "Bearer mock-token",
			}

			recorder := helper.Request(tc.Method, tc.URL, tc.Body, headers)
			helper.AssertStatus(recorder, tc.ExpectedStatus)

			if tc.ExpectedStatus == http.StatusCreated {
				responseBody := recorder.Body.String()
				if !contains(responseBody, "success") || !contains(responseBody, "true") {
					t.Error("Expected success response")
				}
				if !contains(responseBody, "dependency_id") {
					t.Error("Expected dependency_id in response")
				}
			}
		})
	}
}

func TestTaskHandler_RemoveDependency(t *testing.T) {
	tests := []testutil.TestCase{
		{
			Name:           "successfully remove dependency",
			Method:         "DELETE",
			URL:            "/api/projects/project-1/tasks/task-1/dependencies/task-2",
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:           "missing dependency ID",
			Method:         "DELETE",
			URL:            "/api/projects/project-1/tasks/task-1/dependencies/",
			ExpectedStatus: http.StatusNotFound,
		},
		{
			Name:           "missing task ID",
			Method:         "DELETE",
			URL:            "/api/projects/project-1/tasks//dependencies/task-2",
			ExpectedStatus: http.StatusBadRequest,
		},
		{
			Name:           "task not found",
			Method:         "DELETE",
			URL:            "/api/projects/project-1/tasks/non-existent/dependencies/task-2",
			ExpectedStatus: http.StatusNotFound,
		},
	}

	router := setupTaskTestRouter(t)
	helper := testutil.NewHTTPTestHelper(t, router)

	for _, tc := range tests {
		t.Run(tc.Name, func(_ *testing.T) {
			headers := map[string]string{
				"Authorization": "Bearer mock-token",
			}

			recorder := helper.Request(tc.Method, tc.URL, tc.Body, headers)
			helper.AssertStatus(recorder, tc.ExpectedStatus)

			if tc.ExpectedStatus == http.StatusOK {
				responseBody := recorder.Body.String()
				if !contains(responseBody, "success") || !contains(responseBody, "true") {
					t.Error("Expected success response")
				}
				if !contains(responseBody, "dependency_id") {
					t.Error("Expected dependency_id in response")
				}
			}
		})
	}
}

// Note: Helper functions contains() and findInString() are defined in project_handler_test.go

// Ensure interface compliance
var _ services.TaskService = (*MockTaskService)(nil)
