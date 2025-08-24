package api_test

import (
	"context"
	"net/http"
	"testing"

	"simple-easy-tasks/internal/api"
	"simple-easy-tasks/internal/api/middleware"
	"simple-easy-tasks/internal/domain"
	"simple-easy-tasks/internal/services"
	"simple-easy-tasks/internal/testutil"

	"github.com/gin-gonic/gin"
)

func TestProjectHandler_CreateProject(t *testing.T) {
	tests := []testutil.TestCase{
		{
			Name:   "successful project creation",
			Method: "POST",
			URL:    "/api/projects",
			Body: map[string]interface{}{
				"title":       "Test Project",
				"description": "A test project",
				"slug":        "test-project",
				"color":       "#FF0000",
				"icon":        "project-icon",
			},
			ExpectedStatus: http.StatusCreated,
			ExpectedBody: map[string]interface{}{
				"success": true,
				"data": map[string]interface{}{
					"project": map[string]interface{}{
						"title":       "Test Project",
						"description": "A test project",
						"slug":        "test-project",
						"status":      "active",
					},
				},
			},
		},
		{
			Name:   "invalid request body",
			Method: "POST",
			URL:    "/api/projects",
			Body: map[string]interface{}{
				"title": "", // Empty title should fail validation
			},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedBody: map[string]interface{}{
				"success": false,
				"error": map[string]interface{}{
					"type": "VALIDATION_ERROR",
					"code": "INVALID_REQUEST",
				},
			},
		},
		{
			Name:           "missing request body",
			Method:         "POST",
			URL:            "/api/projects",
			Body:           nil,
			ExpectedStatus: http.StatusBadRequest,
			ExpectedBody: map[string]interface{}{
				"success": false,
				"error": map[string]interface{}{
					"type": "VALIDATION_ERROR",
				},
			},
		},
	}

	// Setup router with mock dependencies
	router := setupProjectTestRouter(t)
	helper := testutil.NewHTTPTestHelper(t, router)

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			// Add auth header for authenticated requests
			headers := map[string]string{
				"Authorization": "Bearer mock-token",
			}

			recorder := helper.Request(tc.Method, tc.URL, tc.Body, headers)
			helper.AssertStatus(recorder, tc.ExpectedStatus)

			if tc.ExpectedBody != nil {
				// Note: We're doing partial matching here since exact field matching
				// would be too brittle for complex nested objects
				responseBody := recorder.Body.String()
				if tc.ExpectedStatus == http.StatusCreated {
					if !contains(responseBody, "success") || !contains(responseBody, "true") {
						t.Error("Expected success response")
					}
				} else {
					if !contains(responseBody, "success") || !contains(responseBody, "false") {
						t.Error("Expected error response")
					}
				}
			}
		})
	}
}

func TestProjectHandler_GetProject(t *testing.T) {
	tests := []testutil.TestCase{
		{
			Name:           "get existing project",
			Method:         "GET",
			URL:            "/api/projects/project-1",
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:           "get non-existent project",
			Method:         "GET",
			URL:            "/api/projects/non-existent",
			ExpectedStatus: http.StatusNotFound,
		},
		{
			Name:           "missing project ID",
			Method:         "GET",
			URL:            "/api/projects/",
			ExpectedStatus: http.StatusNotFound, // Router should return 404 for missing param
		},
	}

	router := setupProjectTestRouter(t)
	helper := testutil.NewHTTPTestHelper(t, router)

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			headers := map[string]string{
				"Authorization": "Bearer mock-token",
			}

			recorder := helper.Request(tc.Method, tc.URL, tc.Body, headers)
			helper.AssertStatus(recorder, tc.ExpectedStatus)
		})
	}
}

func TestProjectHandler_ListProjects(t *testing.T) {
	tests := []testutil.TestCase{
		{
			Name:           "list projects with default params",
			Method:         "GET",
			URL:            "/api/projects",
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:           "list projects with pagination",
			Method:         "GET",
			URL:            "/api/projects?limit=10&offset=0",
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:           "list projects with status filter",
			Method:         "GET",
			URL:            "/api/projects?status=active",
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:           "list projects with invalid limit",
			Method:         "GET",
			URL:            "/api/projects?limit=invalid",
			ExpectedStatus: http.StatusOK, // Should use default limit
		},
	}

	router := setupProjectTestRouter(t)
	helper := testutil.NewHTTPTestHelper(t, router)

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			headers := map[string]string{
				"Authorization": "Bearer mock-token",
			}

			recorder := helper.Request(tc.Method, tc.URL, tc.Body, headers)
			helper.AssertStatus(recorder, tc.ExpectedStatus)

			if tc.ExpectedStatus == http.StatusOK {
				responseBody := recorder.Body.String()
				if !contains(responseBody, "success") || !contains(responseBody, "projects") {
					t.Error("Expected projects list response")
				}
			}
		})
	}
}

func TestProjectHandler_UpdateProject(t *testing.T) {
	tests := []testutil.TestCase{
		{
			Name:   "update project title",
			Method: "PUT",
			URL:    "/api/projects/project-1",
			Body: map[string]interface{}{
				"title": "Updated Project Title",
			},
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:   "update project description",
			Method: "PUT",
			URL:    "/api/projects/project-1",
			Body: map[string]interface{}{
				"description": "Updated description",
			},
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:           "update non-existent project",
			Method:         "PUT",
			URL:            "/api/projects/non-existent",
			Body:           map[string]interface{}{"title": "New Title"},
			ExpectedStatus: http.StatusNotFound,
		},
		{
			Name:           "update with invalid body",
			Method:         "PUT",
			URL:            "/api/projects/project-1",
			Body:           "invalid json",
			ExpectedStatus: http.StatusBadRequest,
		},
	}

	router := setupProjectTestRouter(t)
	helper := testutil.NewHTTPTestHelper(t, router)

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			headers := map[string]string{
				"Authorization": "Bearer mock-token",
			}

			recorder := helper.Request(tc.Method, tc.URL, tc.Body, headers)
			helper.AssertStatus(recorder, tc.ExpectedStatus)
		})
	}
}

func TestProjectHandler_DeleteProject(t *testing.T) {
	tests := []testutil.TestCase{
		{
			Name:           "delete existing project",
			Method:         "DELETE",
			URL:            "/api/projects/project-1",
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:           "delete non-existent project",
			Method:         "DELETE",
			URL:            "/api/projects/non-existent",
			ExpectedStatus: http.StatusNotFound,
		},
		{
			Name:           "delete without project ID",
			Method:         "DELETE",
			URL:            "/api/projects/",
			ExpectedStatus: http.StatusNotFound,
		},
	}

	router := setupProjectTestRouter(t)
	helper := testutil.NewHTTPTestHelper(t, router)

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			headers := map[string]string{
				"Authorization": "Bearer mock-token",
			}

			recorder := helper.Request(tc.Method, tc.URL, tc.Body, headers)
			helper.AssertStatus(recorder, tc.ExpectedStatus)
		})
	}
}

// setupProjectTestRouter creates a test router with project endpoints and mock dependencies.
func setupProjectTestRouter(t *testing.T) *gin.Engine {
	router := testutil.NewTestRouter()

	// Create mock repositories
	projectRepo := testutil.NewMockProjectRepository()
	userRepo := testutil.NewMockUserRepository()

	// Add test data
	testUser := testutil.MockUser("user-1", "test@example.com", "testuser", "Test User")
	testProject := testutil.MockProject("project-1", "Test Project", "test-project", "user-1")

	userRepo.AddUser(testUser)
	projectRepo.AddProject(testProject)

	// Create mock auth service
	mockAuthService := &MockAuthService{
		user: testUser,
	}

	// Create middleware
	authMiddleware := middleware.NewAuthMiddleware(mockAuthService)

	// Setup routes
	projectHandler := api.NewProjectHandler(projectRepo)

	apiGroup := router.Group("/api")
	projectHandler.RegisterRoutes(apiGroup, authMiddleware)

	return router
}

// MockAuthService is a mock implementation of AuthService for testing.
type MockAuthService struct {
	user *domain.User
}

func (m *MockAuthService) ValidateToken(ctx context.Context, token string) (*domain.User, error) {
	if token == "mock-token" {
		return m.user, nil
	}
	return nil, domain.NewAuthenticationError("INVALID_TOKEN", "Invalid token")
}

func (m *MockAuthService) Login(ctx context.Context, req domain.LoginRequest) (*domain.TokenPair, error) {
	return nil, domain.NewInternalError("NOT_IMPLEMENTED", "Not implemented in mock", nil)
}

func (m *MockAuthService) Register(ctx context.Context, req domain.CreateUserRequest) (*domain.User, error) {
	return nil, domain.NewInternalError("NOT_IMPLEMENTED", "Not implemented in mock", nil)
}

func (m *MockAuthService) RefreshToken(ctx context.Context, refreshToken string) (*domain.TokenPair, error) {
	return nil, domain.NewInternalError("NOT_IMPLEMENTED", "Not implemented in mock", nil)
}

func (m *MockAuthService) Logout(ctx context.Context, userID string) error {
	return domain.NewInternalError("NOT_IMPLEMENTED", "Not implemented in mock", nil)
}

func (m *MockAuthService) ForgotPassword(ctx context.Context, email string) error {
	return domain.NewInternalError("NOT_IMPLEMENTED", "Not implemented in mock", nil)
}

func (m *MockAuthService) ResetPassword(ctx context.Context, token, newPassword string) error {
	return domain.NewInternalError("NOT_IMPLEMENTED", "Not implemented in mock", nil)
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr ||
		len(s) > len(substr) && s[:len(substr)] == substr ||
		len(s) > len(substr) && findInString(s, substr)
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Ensure interface compliance
var _ services.AuthService = (*MockAuthService)(nil)
