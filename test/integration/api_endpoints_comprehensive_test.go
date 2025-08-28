//go:build integration
// +build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"simple-easy-tasks/internal/api"
	"simple-easy-tasks/internal/api/middleware"
	"simple-easy-tasks/internal/domain"
	"simple-easy-tasks/internal/services"
	"simple-easy-tasks/internal/testutil/integration"
)

// APIEndpointsTestSuite provides comprehensive API endpoint testing
type APIEndpointsTestSuite struct {
	*integration.DatabaseTestSuite
	router      *gin.Engine
	server      *httptest.Server
	authToken   string
	testUser    *domain.User
	testProject *domain.Project
	adminToken  string
	adminUser   *domain.User
}

// setupAPITestSuite initializes the API test suite with router and authentication
func setupAPITestSuite(t *testing.T) *APIEndpointsTestSuite {
	// Setup database test suite
	dbSuite := integration.SetupConcurrencyTestWithServices(t)

	suite := &APIEndpointsTestSuite{
		DatabaseTestSuite: dbSuite,
	}

	// Setup Gin router with all handlers
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add middleware
	router.Use(middleware.RequestIDMiddleware())
	router.Use(middleware.DefaultLoggingMiddleware())
	router.Use(middleware.DefaultRecoveryMiddleware())
	router.Use(middleware.DefaultCORSMiddleware())

	// Get services from DI container
	authService := suite.GetAuthService(t)
	userRepo := suite.GetUserRepository(t)
	projectService := suite.GetProjectService(t)
	projectRepo := suite.GetProjectRepository(t)
	taskService := suite.GetTaskService(t)
	taskRepo := suite.GetTaskRepository(t)
	healthService := suite.GetHealthService(t)
	subscriptionManager := suite.GetSubscriptionManager(t)
	eventBroadcaster := suite.GetEventBroadcaster(t)
	app := suite.GetPocketBaseApp(t)

	// Create auth middleware
	authMiddleware := middleware.NewAuthMiddleware(authService)

	// Initialize handlers
	authHandler := api.NewAuthHandler(authService)
	userHandler := api.NewUserHandler(userRepo)
	projectHandler := api.NewProjectHandler(projectService, projectRepo)
	taskHandler := api.NewTaskHandler(taskService, taskRepo)
	healthHandler := api.NewHealthHandler(healthService.(*services.HealthService))
	realtimeHandler := api.NewRealtimeHandler(subscriptionManager, eventBroadcaster, app)

	// Register routes
	apiGroup := router.Group("/api")
	authHandler.RegisterRoutes(apiGroup, authMiddleware)
	userHandler.RegisterRoutes(apiGroup, authMiddleware)
	projectHandler.RegisterRoutes(apiGroup, authMiddleware)
	taskHandler.RegisterRoutes(apiGroup, authMiddleware)
	realtimeHandler.RegisterRoutes(apiGroup, authMiddleware)

	// Health routes (no /api prefix)
	healthHandler.RegisterRoutes(router)

	suite.router = router
	suite.server = httptest.NewServer(router)

	// Create test users and get auth tokens
	suite.setupTestData(t)

	return suite
}

// setupTestData creates test users, projects and authenticates
func (s *APIEndpointsTestSuite) setupTestData(t *testing.T) {
	ctx := context.Background()

	// Create regular test user
	userReq := domain.CreateUserRequest{
		Email:    "test@example.com",
		Password: "testpassword123",
		Name:     "Test User",
	}

	user, err := s.GetAuthService(t).Register(ctx, userReq)
	require.NoError(t, err)
	s.testUser = user

	// Login to get token
	loginReq := domain.LoginRequest{
		Email:    "test@example.com",
		Password: "testpassword123",
	}

	tokenPair, err := s.GetAuthService(t).Login(ctx, loginReq)
	require.NoError(t, err)
	s.authToken = tokenPair.AccessToken

	// Create admin user
	adminReq := domain.CreateUserRequest{
		Email:    "admin@example.com",
		Password: "adminpassword123",
		Name:     "Admin User",
	}

	adminUser, err := s.GetAuthService(t).Register(ctx, adminReq)
	require.NoError(t, err)

	// Set admin role
	adminUser.Role = domain.AdminRole
	err = s.GetUserRepository(t).Update(ctx, adminUser)
	require.NoError(t, err)
	s.adminUser = adminUser

	// Login admin to get token
	adminLoginReq := domain.LoginRequest{
		Email:    "admin@example.com",
		Password: "adminpassword123",
	}

	adminTokenPair, err := s.GetAuthService(t).Login(ctx, adminLoginReq)
	require.NoError(t, err)
	s.adminToken = adminTokenPair.AccessToken

	// Create test project
	project := &domain.Project{
		Title:       "Test Project",
		Description: "Test project for API testing",
		Slug:        "test-project",
		OwnerID:     user.ID,
		Color:       "#3b82f6",
		Icon:        "📊",
		Status:      domain.ActiveProject,
		Settings:    domain.ProjectSettings{},
		MemberIDs:   []string{},
	}

	err = s.GetProjectRepository(t).Create(ctx, project)
	require.NoError(t, err)
	s.testProject = project
}

// Cleanup closes the test server
func (s *APIEndpointsTestSuite) Cleanup() {
	if s.server != nil {
		s.server.Close()
	}
	s.DatabaseTestSuite.Cleanup()
}

// makeRequest makes an HTTP request to the test server
func (s *APIEndpointsTestSuite) makeRequest(method, path string, body interface{}, token string) (*http.Response, error) {
	var reqBody *bytes.Buffer
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonBody)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req, err := http.NewRequest(method, s.server.URL+path, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	return client.Do(req)
}

// parseResponse parses JSON response body
func (s *APIEndpointsTestSuite) parseResponse(resp *http.Response, v interface{}) error {
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(v)
}

// TestAuthenticationEndpoints tests all authentication endpoints
func TestAuthenticationEndpoints(t *testing.T) {
	suite := setupAPITestSuite(t)
	defer suite.Cleanup()

	t.Run("Login", func(t *testing.T) {
		loginReq := map[string]string{
			"email":    "test@example.com",
			"password": "testpassword123",
		}

		resp, err := suite.makeRequest("POST", "/api/auth/login", loginReq, "")
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = suite.parseResponse(resp, &result)
		require.NoError(t, err)
		assert.True(t, result["success"].(bool))
		assert.Contains(t, result, "data")

		data := result["data"].(map[string]interface{})
		assert.Contains(t, data, "access_token")
		assert.Contains(t, data, "refresh_token")
		assert.Contains(t, data, "expires_at")
	})

	t.Run("Login_InvalidCredentials", func(t *testing.T) {
		loginReq := map[string]string{
			"email":    "test@example.com",
			"password": "wrongpassword",
		}

		resp, err := suite.makeRequest("POST", "/api/auth/login", loginReq, "")
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		var result map[string]interface{}
		err = suite.parseResponse(resp, &result)
		require.NoError(t, err)
		assert.False(t, result["success"].(bool))
		assert.Contains(t, result, "error")
	})

	t.Run("Register", func(t *testing.T) {
		registerReq := map[string]string{
			"email":    "newuser@example.com",
			"password": "newpassword123",
			"name":     "New User",
		}

		resp, err := suite.makeRequest("POST", "/api/auth/register", registerReq, "")
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		err = suite.parseResponse(resp, &result)
		require.NoError(t, err)
		assert.True(t, result["success"].(bool))
		assert.Contains(t, result, "data")

		data := result["data"].(map[string]interface{})
		user := data["user"].(map[string]interface{})
		assert.Equal(t, "newuser@example.com", user["email"])
		assert.Equal(t, "New User", user["name"])
		assert.Equal(t, "user", user["role"])
	})

	t.Run("RefreshToken", func(t *testing.T) {
		// First login to get refresh token
		loginReq := map[string]string{
			"email":    "test@example.com",
			"password": "testpassword123",
		}

		resp, err := suite.makeRequest("POST", "/api/auth/login", loginReq, "")
		require.NoError(t, err)

		var loginResult map[string]interface{}
		err = suite.parseResponse(resp, &loginResult)
		require.NoError(t, err)

		data := loginResult["data"].(map[string]interface{})
		refreshToken := data["refresh_token"].(string)

		// Use refresh token
		refreshReq := map[string]string{
			"refresh_token": refreshToken,
		}

		resp, err = suite.makeRequest("POST", "/api/auth/refresh", refreshReq, "")
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = suite.parseResponse(resp, &result)
		require.NoError(t, err)
		assert.True(t, result["success"].(bool))
	})

	t.Run("GetProfile", func(t *testing.T) {
		resp, err := suite.makeRequest("GET", "/api/auth/me", nil, suite.authToken)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = suite.parseResponse(resp, &result)
		require.NoError(t, err)
		assert.True(t, result["success"].(bool))

		data := result["data"].(map[string]interface{})
		user := data["user"].(map[string]interface{})
		assert.Equal(t, "test@example.com", user["email"])
		assert.Equal(t, "Test User", user["name"])
	})

	t.Run("Logout", func(t *testing.T) {
		resp, err := suite.makeRequest("POST", "/api/auth/logout", nil, suite.authToken)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = suite.parseResponse(resp, &result)
		require.NoError(t, err)
		assert.True(t, result["success"].(bool))
		assert.Contains(t, result["message"], "logged out")
	})

	t.Run("ForgotPassword", func(t *testing.T) {
		forgotReq := map[string]string{
			"email": "test@example.com",
		}

		resp, err := suite.makeRequest("POST", "/api/auth/forgot-password", forgotReq, "")
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = suite.parseResponse(resp, &result)
		require.NoError(t, err)
		assert.True(t, result["success"].(bool))
		assert.Contains(t, result["message"], "password reset")
	})
}

// TestUserEndpoints tests user management endpoints
func TestUserEndpoints(t *testing.T) {
	suite := setupAPITestSuite(t)
	defer suite.Cleanup()

	t.Run("GetProfile", func(t *testing.T) {
		resp, err := suite.makeRequest("GET", "/api/users/profile", nil, suite.authToken)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = suite.parseResponse(resp, &result)
		require.NoError(t, err)
		assert.True(t, result["success"].(bool))
	})

	t.Run("UpdateProfile", func(t *testing.T) {
		updateReq := map[string]interface{}{
			"name": "Updated Test User",
			"preferences": map[string]interface{}{
				"theme":         "dark",
				"notifications": false,
			},
		}

		resp, err := suite.makeRequest("PUT", "/api/users/profile", updateReq, suite.authToken)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = suite.parseResponse(resp, &result)
		require.NoError(t, err)
		assert.True(t, result["success"].(bool))

		data := result["data"].(map[string]interface{})
		user := data["user"].(map[string]interface{})
		assert.Equal(t, "Updated Test User", user["name"])
	})

	t.Run("UpdateAvatar", func(t *testing.T) {
		avatarReq := map[string]string{
			"avatar": "https://example.com/avatar.jpg",
		}

		resp, err := suite.makeRequest("POST", "/api/users/avatar", avatarReq, suite.authToken)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("RemoveAvatar", func(t *testing.T) {
		resp, err := suite.makeRequest("DELETE", "/api/users/avatar", nil, suite.authToken)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("UpdatePreferences", func(t *testing.T) {
		prefsReq := map[string]interface{}{
			"theme":         "light",
			"notifications": true,
			"language":      "en",
		}

		resp, err := suite.makeRequest("PUT", "/api/users/preferences", prefsReq, suite.authToken)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// Admin-only endpoints
	t.Run("ListUsers_Admin", func(t *testing.T) {
		resp, err := suite.makeRequest("GET", "/api/users", nil, suite.adminToken)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = suite.parseResponse(resp, &result)
		require.NoError(t, err)
		assert.True(t, result["success"].(bool))
	})

	t.Run("ListUsers_Forbidden", func(t *testing.T) {
		resp, err := suite.makeRequest("GET", "/api/users", nil, suite.authToken)
		require.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("GetUserByID_Admin", func(t *testing.T) {
		path := fmt.Sprintf("/api/users/%s", suite.testUser.ID)
		resp, err := suite.makeRequest("GET", path, nil, suite.adminToken)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("UpdateUserRole_Admin", func(t *testing.T) {
		roleReq := map[string]string{
			"role": "admin",
		}

		path := fmt.Sprintf("/api/users/%s/role", suite.testUser.ID)
		resp, err := suite.makeRequest("PUT", path, roleReq, suite.adminToken)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

// TestProjectEndpoints tests project management endpoints
func TestProjectEndpoints(t *testing.T) {
	suite := setupAPITestSuite(t)
	defer suite.Cleanup()

	t.Run("ListProjects", func(t *testing.T) {
		resp, err := suite.makeRequest("GET", "/api/projects", nil, suite.authToken)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = suite.parseResponse(resp, &result)
		require.NoError(t, err)
		assert.True(t, result["success"].(bool))

		data := result["data"].(map[string]interface{})
		projects := data["projects"].([]interface{})
		assert.GreaterOrEqual(t, len(projects), 1) // At least the test project
	})

	t.Run("CreateProject", func(t *testing.T) {
		projectReq := map[string]interface{}{
			"title":       "New Test Project",
			"description": "Created via API test",
			"slug":        "new-test-project",
			"color":       "#ef4444",
			"icon":        "🚀",
			"settings": map[string]interface{}{
				"is_private":       false,
				"allow_guest_view": true,
				"enable_comments":  true,
				"custom_fields":    map[string]string{},
				"notifications":    map[string]bool{},
			},
		}

		resp, err := suite.makeRequest("POST", "/api/projects", projectReq, suite.authToken)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		err = suite.parseResponse(resp, &result)
		require.NoError(t, err)
		assert.True(t, result["success"].(bool))

		data := result["data"].(map[string]interface{})
		project := data["project"].(map[string]interface{})
		assert.Equal(t, "New Test Project", project["title"])
		assert.Equal(t, suite.testUser.ID, project["owner_id"])
	})

	t.Run("GetProject", func(t *testing.T) {
		path := fmt.Sprintf("/api/projects/%s", suite.testProject.ID)
		resp, err := suite.makeRequest("GET", path, nil, suite.authToken)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = suite.parseResponse(resp, &result)
		require.NoError(t, err)
		assert.True(t, result["success"].(bool))

		data := result["data"].(map[string]interface{})
		project := data["project"].(map[string]interface{})
		assert.Equal(t, suite.testProject.ID, project["id"])
		assert.Equal(t, suite.testProject.Title, project["title"])
	})

	t.Run("UpdateProject", func(t *testing.T) {
		updateReq := map[string]interface{}{
			"title":       "Updated Project Title",
			"description": "Updated description",
			"color":       "#10b981",
		}

		path := fmt.Sprintf("/api/projects/%s", suite.testProject.ID)
		resp, err := suite.makeRequest("PUT", path, updateReq, suite.authToken)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = suite.parseResponse(resp, &result)
		require.NoError(t, err)
		assert.True(t, result["success"].(bool))
	})

	t.Run("AddMember", func(t *testing.T) {
		memberReq := map[string]string{
			"user_id": suite.adminUser.ID,
		}

		path := fmt.Sprintf("/api/projects/%s/members", suite.testProject.ID)
		resp, err := suite.makeRequest("POST", path, memberReq, suite.authToken)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("RemoveMember", func(t *testing.T) {
		path := fmt.Sprintf("/api/projects/%s/members/%s", suite.testProject.ID, suite.adminUser.ID)
		resp, err := suite.makeRequest("DELETE", path, nil, suite.authToken)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

// TestTaskEndpoints tests task management endpoints
func TestTaskEndpoints(t *testing.T) {
	suite := setupAPITestSuite(t)
	defer suite.Cleanup()

	var createdTaskID string

	t.Run("CreateTask", func(t *testing.T) {
		taskReq := map[string]interface{}{
			"title":          "Test Task",
			"description":    "Task created via API test",
			"priority":       "high",
			"status":         "backlog",
			"assignee_id":    suite.testUser.ID,
			"due_date":       time.Now().Add(7 * 24 * time.Hour).Format(time.RFC3339),
			"tags":           []string{"testing", "api"},
			"time_estimated": 5.0,
		}

		path := fmt.Sprintf("/api/projects/%s/tasks", suite.testProject.ID)
		resp, err := suite.makeRequest("POST", path, taskReq, suite.authToken)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		err = suite.parseResponse(resp, &result)
		require.NoError(t, err)
		assert.True(t, result["success"].(bool))

		data := result["data"].(map[string]interface{})
		task := data["task"].(map[string]interface{})
		createdTaskID = task["id"].(string)
		assert.Equal(t, "Test Task", task["title"])
		assert.Equal(t, "high", task["priority"])
		assert.Equal(t, suite.testProject.ID, task["project_id"])
	})

	t.Run("ListTasks", func(t *testing.T) {
		path := fmt.Sprintf("/api/projects/%s/tasks", suite.testProject.ID)
		resp, err := suite.makeRequest("GET", path, nil, suite.authToken)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = suite.parseResponse(resp, &result)
		require.NoError(t, err)
		assert.True(t, result["success"].(bool))

		data := result["data"].(map[string]interface{})
		tasks := data["tasks"].([]interface{})
		assert.GreaterOrEqual(t, len(tasks), 1)
	})

	t.Run("ListTasks_WithFilters", func(t *testing.T) {
		path := fmt.Sprintf("/api/projects/%s/tasks?status=backlog&priority=high&limit=10", suite.testProject.ID)
		resp, err := suite.makeRequest("GET", path, nil, suite.authToken)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("GetTask", func(t *testing.T) {
		path := fmt.Sprintf("/api/projects/%s/tasks/%s", suite.testProject.ID, createdTaskID)
		resp, err := suite.makeRequest("GET", path, nil, suite.authToken)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = suite.parseResponse(resp, &result)
		require.NoError(t, err)
		assert.True(t, result["success"].(bool))

		data := result["data"].(map[string]interface{})
		task := data["task"].(map[string]interface{})
		assert.Equal(t, createdTaskID, task["id"])
		assert.Equal(t, "Test Task", task["title"])
	})

	t.Run("UpdateTask", func(t *testing.T) {
		updateReq := map[string]interface{}{
			"title":       "Updated Test Task",
			"description": "Updated description",
			"priority":    "critical",
			"status":      "todo",
		}

		path := fmt.Sprintf("/api/projects/%s/tasks/%s", suite.testProject.ID, createdTaskID)
		resp, err := suite.makeRequest("PUT", path, updateReq, suite.authToken)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = suite.parseResponse(resp, &result)
		require.NoError(t, err)
		assert.True(t, result["success"].(bool))
	})

	t.Run("MoveTask", func(t *testing.T) {
		moveReq := map[string]interface{}{
			"new_status":   "developing",
			"new_position": 1,
		}

		path := fmt.Sprintf("/api/projects/%s/tasks/%s/move", suite.testProject.ID, createdTaskID)
		resp, err := suite.makeRequest("POST", path, moveReq, suite.authToken)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("UpdateTaskStatus", func(t *testing.T) {
		statusReq := map[string]string{
			"status": "review",
		}

		path := fmt.Sprintf("/api/projects/%s/tasks/%s/status", suite.testProject.ID, createdTaskID)
		resp, err := suite.makeRequest("PUT", path, statusReq, suite.authToken)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("AssignTask", func(t *testing.T) {
		assignReq := map[string]string{
			"assignee_id": suite.adminUser.ID,
		}

		path := fmt.Sprintf("/api/projects/%s/tasks/%s/assign", suite.testProject.ID, createdTaskID)
		resp, err := suite.makeRequest("POST", path, assignReq, suite.authToken)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("UnassignTask", func(t *testing.T) {
		path := fmt.Sprintf("/api/projects/%s/tasks/%s/assign", suite.testProject.ID, createdTaskID)
		resp, err := suite.makeRequest("DELETE", path, nil, suite.authToken)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("LogTimeSpent", func(t *testing.T) {
		timeReq := map[string]interface{}{
			"hours":       2.5,
			"description": "Worked on implementation",
			"logged_at":   time.Now().Format(time.RFC3339),
		}

		path := fmt.Sprintf("/api/projects/%s/tasks/%s/time-log", suite.testProject.ID, createdTaskID)
		resp, err := suite.makeRequest("POST", path, timeReq, suite.authToken)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("GetTaskHistory", func(t *testing.T) {
		path := fmt.Sprintf("/api/projects/%s/tasks/%s/history", suite.testProject.ID, createdTaskID)
		resp, err := suite.makeRequest("GET", path, nil, suite.authToken)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("DuplicateTask", func(t *testing.T) {
		dupReq := map[string]interface{}{
			"include_subtasks":    false,
			"include_comments":    false,
			"include_attachments": false,
			"reset_progress":      true,
			"reset_time_spent":    true,
		}

		path := fmt.Sprintf("/api/projects/%s/tasks/%s/duplicate", suite.testProject.ID, createdTaskID)
		resp, err := suite.makeRequest("POST", path, dupReq, suite.authToken)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("CreateSubtask", func(t *testing.T) {
		subtaskReq := map[string]interface{}{
			"title":       "Test Subtask",
			"description": "Subtask for testing",
			"priority":    "medium",
		}

		path := fmt.Sprintf("/api/projects/%s/tasks/%s/subtasks", suite.testProject.ID, createdTaskID)
		resp, err := suite.makeRequest("POST", path, subtaskReq, suite.authToken)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("ListSubtasks", func(t *testing.T) {
		path := fmt.Sprintf("/api/projects/%s/tasks/%s/subtasks", suite.testProject.ID, createdTaskID)
		resp, err := suite.makeRequest("GET", path, nil, suite.authToken)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

// TestHealthEndpoints tests health and monitoring endpoints
func TestHealthEndpoints(t *testing.T) {
	suite := setupAPITestSuite(t)
	defer suite.Cleanup()

	t.Run("HealthCheck", func(t *testing.T) {
		resp, err := suite.makeRequest("GET", "/health", nil, "")
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = suite.parseResponse(resp, &result)
		require.NoError(t, err)
		assert.Contains(t, result, "status")
		assert.Contains(t, result, "timestamp")
	})

	t.Run("Liveness", func(t *testing.T) {
		resp, err := suite.makeRequest("GET", "/health/live", nil, "")
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = suite.parseResponse(resp, &result)
		require.NoError(t, err)
		assert.Equal(t, "alive", result["status"])
	})

	t.Run("Readiness", func(t *testing.T) {
		resp, err := suite.makeRequest("GET", "/health/ready", nil, "")
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("DetailedHealth", func(t *testing.T) {
		resp, err := suite.makeRequest("GET", "/health/detailed", nil, "")
		require.NoError(t, err)
		// Should be OK or Service Unavailable depending on dependencies
		assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusServiceUnavailable)
	})

	t.Run("SystemInfo", func(t *testing.T) {
		resp, err := suite.makeRequest("GET", "/health/info", nil, "")
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = suite.parseResponse(resp, &result)
		require.NoError(t, err)
		assert.Contains(t, result, "version")
		assert.Contains(t, result, "uptime")
	})
}

// TestRealtimeEndpoints tests real-time subscription endpoints
func TestRealtimeEndpoints(t *testing.T) {
	suite := setupAPITestSuite(t)
	defer suite.Cleanup()

	var subscriptionID string

	t.Run("CreateSubscription", func(t *testing.T) {
		subReq := map[string]interface{}{
			"project_id":  suite.testProject.ID,
			"event_types": []string{"task_created", "task_updated"},
			"filters": map[string]string{
				"assignee_id": suite.testUser.ID,
			},
		}

		resp, err := suite.makeRequest("POST", "/api/realtime/subscriptions", subReq, suite.authToken)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		err = suite.parseResponse(resp, &result)
		require.NoError(t, err)
		assert.True(t, result["success"].(bool))

		data := result["data"].(map[string]interface{})
		subscriptionID = data["id"].(string)
		assert.Equal(t, suite.testProject.ID, data["project_id"])
		assert.Equal(t, suite.testUser.ID, data["user_id"])
	})

	t.Run("ListSubscriptions", func(t *testing.T) {
		resp, err := suite.makeRequest("GET", "/api/realtime/subscriptions", nil, suite.authToken)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = suite.parseResponse(resp, &result)
		require.NoError(t, err)
		assert.True(t, result["success"].(bool))

		data := result["data"].([]interface{})
		assert.GreaterOrEqual(t, len(data), 1)
	})

	t.Run("GetSubscription", func(t *testing.T) {
		path := fmt.Sprintf("/api/realtime/subscriptions/%s", subscriptionID)
		resp, err := suite.makeRequest("GET", path, nil, suite.authToken)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("UpdateSubscription", func(t *testing.T) {
		updateReq := map[string]interface{}{
			"event_types": []string{"task_created", "task_updated", "task_moved"},
		}

		path := fmt.Sprintf("/api/realtime/subscriptions/%s", subscriptionID)
		resp, err := suite.makeRequest("PUT", path, updateReq, suite.authToken)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("GetActiveConnections", func(t *testing.T) {
		resp, err := suite.makeRequest("GET", "/api/realtime/connections", nil, suite.authToken)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("RealtimeHealth", func(t *testing.T) {
		resp, err := suite.makeRequest("GET", "/api/realtime/health", nil, "")
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("DeleteSubscription", func(t *testing.T) {
		path := fmt.Sprintf("/api/realtime/subscriptions/%s", subscriptionID)
		resp, err := suite.makeRequest("DELETE", path, nil, suite.authToken)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

// TestErrorHandlingAndValidation tests error scenarios and input validation
func TestErrorHandlingAndValidation(t *testing.T) {
	suite := setupAPITestSuite(t)
	defer suite.Cleanup()

	t.Run("UnauthorizedAccess", func(t *testing.T) {
		resp, err := suite.makeRequest("GET", "/api/users/profile", nil, "")
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		var result map[string]interface{}
		err = suite.parseResponse(resp, &result)
		require.NoError(t, err)
		assert.False(t, result["success"].(bool))
		assert.Contains(t, result, "error")
	})

	t.Run("InvalidToken", func(t *testing.T) {
		resp, err := suite.makeRequest("GET", "/api/users/profile", nil, "invalid-token")
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("InvalidRequestBody", func(t *testing.T) {
		invalidReq := "invalid json"
		req, err := http.NewRequest("POST", suite.server.URL+"/api/auth/login", strings.NewReader(invalidReq))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("NotFound", func(t *testing.T) {
		resp, err := suite.makeRequest("GET", "/api/projects/nonexistent", nil, suite.authToken)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("ValidationError", func(t *testing.T) {
		// Try to create task with invalid data
		invalidTaskReq := map[string]interface{}{
			"title": "", // Empty title should fail validation
		}

		path := fmt.Sprintf("/api/projects/%s/tasks", suite.testProject.ID)
		resp, err := suite.makeRequest("POST", path, invalidTaskReq, suite.authToken)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var result map[string]interface{}
		err = suite.parseResponse(resp, &result)
		require.NoError(t, err)
		assert.False(t, result["success"].(bool))

		errorInfo := result["error"].(map[string]interface{})
		assert.Equal(t, "VALIDATION_ERROR", errorInfo["type"])
	})
}

// TestRequestResponseSchemas validates the structure of API responses
func TestRequestResponseSchemas(t *testing.T) {
	suite := setupAPITestSuite(t)
	defer suite.Cleanup()

	t.Run("LoginResponseSchema", func(t *testing.T) {
		loginReq := map[string]string{
			"email":    "test@example.com",
			"password": "testpassword123",
		}

		resp, err := suite.makeRequest("POST", "/api/auth/login", loginReq, "")
		require.NoError(t, err)

		var result map[string]interface{}
		err = suite.parseResponse(resp, &result)
		require.NoError(t, err)

		// Validate response structure
		assert.Contains(t, result, "success")
		assert.True(t, result["success"].(bool))
		assert.Contains(t, result, "data")

		data := result["data"].(map[string]interface{})
		assert.Contains(t, data, "access_token")
		assert.Contains(t, data, "refresh_token")
		assert.Contains(t, data, "expires_at")

		// Validate token format (should be JWT-like)
		accessToken := data["access_token"].(string)
		assert.True(t, len(accessToken) > 0)
		assert.Contains(t, accessToken, ".")
	})

	t.Run("TaskResponseSchema", func(t *testing.T) {
		taskReq := map[string]interface{}{
			"title":       "Schema Test Task",
			"description": "Testing response schema",
			"priority":    "medium",
			"status":      "backlog",
		}

		path := fmt.Sprintf("/api/projects/%s/tasks", suite.testProject.ID)
		resp, err := suite.makeRequest("POST", path, taskReq, suite.authToken)
		require.NoError(t, err)

		var result map[string]interface{}
		err = suite.parseResponse(resp, &result)
		require.NoError(t, err)

		// Validate response structure
		assert.Contains(t, result, "success")
		assert.True(t, result["success"].(bool))
		assert.Contains(t, result, "data")

		data := result["data"].(map[string]interface{})
		task := data["task"].(map[string]interface{})

		// Validate required task fields
		requiredFields := []string{
			"id", "title", "description", "status", "priority",
			"project_id", "reporter_id", "created_at", "updated_at",
		}

		for _, field := range requiredFields {
			assert.Contains(t, task, field, "Task response missing required field: %s", field)
		}

		// Validate data types
		assert.IsType(t, "", task["id"].(string))
		assert.IsType(t, "", task["title"].(string))
		assert.IsType(t, "", task["status"].(string))
		assert.IsType(t, "", task["priority"].(string))
		assert.IsType(t, "", task["project_id"].(string))
	})

	t.Run("ErrorResponseSchema", func(t *testing.T) {
		// Make request that will fail
		resp, err := suite.makeRequest("GET", "/api/users/profile", nil, "invalid-token")
		require.NoError(t, err)

		var result map[string]interface{}
		err = suite.parseResponse(resp, &result)
		require.NoError(t, err)

		// Validate error response structure
		assert.Contains(t, result, "success")
		assert.False(t, result["success"].(bool))
		assert.Contains(t, result, "error")

		errorInfo := result["error"].(map[string]interface{})
		assert.Contains(t, errorInfo, "type")
		assert.Contains(t, errorInfo, "code")
		assert.Contains(t, errorInfo, "message")

		// Validate error types are consistent
		errorType := errorInfo["type"].(string)
		assert.True(t, len(errorType) > 0)
		assert.True(t, strings.Contains(errorType, "_ERROR"))
	})
}
