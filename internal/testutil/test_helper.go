// Package testutil provides testing utilities and helpers.
package testutil

//nolint:gofumpt
import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"simple-easy-tasks/internal/config"
	"simple-easy-tasks/internal/domain"

	"github.com/gin-gonic/gin"
)

// TestConfig returns a test configuration.
func TestConfig() config.Config {
	return config.NewConfig()
}

// NewTestRouter creates a new Gin router for testing.
func NewTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

// TestCase represents a test case for HTTP handlers.
type TestCase struct {
	Body           interface{}
	ExpectedBody   interface{}
	Headers        map[string]string
	SetupFunc      func(t *testing.T)
	CleanupFunc    func(t *testing.T)
	Name           string
	Method         string
	URL            string
	ExpectedStatus int
}

// HTTPTestHelper provides utilities for HTTP testing.
type HTTPTestHelper struct {
	router *gin.Engine
	t      *testing.T
}

// NewHTTPTestHelper creates a new HTTP test helper.
func NewHTTPTestHelper(t *testing.T, router *gin.Engine) *HTTPTestHelper {
	return &HTTPTestHelper{
		router: router,
		t:      t,
	}
}

// Request performs an HTTP request and returns the response.
func (h *HTTPTestHelper) Request(
	method,
	url string,
	body interface{},
	headers map[string]string,
) *httptest.ResponseRecorder {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			h.t.Fatalf("Failed to marshal request body: %v", err)
		}
		bodyReader = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(context.Background(), method, url, bodyReader)
	if err != nil {
		h.t.Fatalf("Failed to create request: %v", err)
	}

	// Set default content type
	if bodyReader != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Set custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	recorder := httptest.NewRecorder()
	h.router.ServeHTTP(recorder, req)
	return recorder
}

// GET performs a GET request.
func (h *HTTPTestHelper) GET(url string, headers map[string]string) *httptest.ResponseRecorder {
	return h.Request("GET", url, nil, headers)
}

// POST performs a POST request.
func (h *HTTPTestHelper) POST(url string, body interface{}, headers map[string]string) *httptest.ResponseRecorder {
	return h.Request("POST", url, body, headers)
}

// PUT performs a PUT request.
func (h *HTTPTestHelper) PUT(url string, body interface{}, headers map[string]string) *httptest.ResponseRecorder {
	return h.Request("PUT", url, body, headers)
}

// DELETE performs a DELETE request.
func (h *HTTPTestHelper) DELETE(url string, headers map[string]string) *httptest.ResponseRecorder {
	return h.Request("DELETE", url, nil, headers)
}

// AssertJSON asserts that the response body matches the expected JSON.
func (h *HTTPTestHelper) AssertJSON(recorder *httptest.ResponseRecorder, expected interface{}) {
	var actualMap map[string]interface{}
	var expectedMap map[string]interface{}

	err := json.Unmarshal(recorder.Body.Bytes(), &actualMap)
	if err != nil {
		h.t.Fatalf("Failed to unmarshal actual response: %v", err)
	}

	expectedBytes, err := json.Marshal(expected)
	if err != nil {
		h.t.Fatalf("Failed to marshal expected response: %v", err)
	}

	err = json.Unmarshal(expectedBytes, &expectedMap)
	if err != nil {
		h.t.Fatalf("Failed to unmarshal expected response: %v", err)
	}

	if !jsonEqual(actualMap, expectedMap) {
		h.t.Errorf("Response body mismatch.\nExpected: %s\nActual: %s",
			string(expectedBytes), recorder.Body.String())
	}
}

// AssertStatus asserts that the response has the expected status code.
func (h *HTTPTestHelper) AssertStatus(recorder *httptest.ResponseRecorder, expectedStatus int) {
	if recorder.Code != expectedStatus {
		h.t.Errorf("Status code mismatch. Expected: %d, Actual: %d", expectedStatus, recorder.Code)
	}
}

// AssertHeader asserts that the response has the expected header value.
func (h *HTTPTestHelper) AssertHeader(recorder *httptest.ResponseRecorder, header, expectedValue string) {
	actualValue := recorder.Header().Get(header)
	if actualValue != expectedValue {
		h.t.Errorf("Header %s mismatch. Expected: %s, Actual: %s", header, expectedValue, actualValue)
	}
}

// RunTestCases runs a slice of test cases.
func (h *HTTPTestHelper) RunTestCases(testCases []TestCase) {
	for _, tc := range testCases {
		h.t.Run(tc.Name, func(t *testing.T) {
			if tc.SetupFunc != nil {
				tc.SetupFunc(t)
			}

			if tc.CleanupFunc != nil {
				defer tc.CleanupFunc(t)
			}

			recorder := h.Request(tc.Method, tc.URL, tc.Body, tc.Headers)

			h.AssertStatus(recorder, tc.ExpectedStatus)

			if tc.ExpectedBody != nil {
				h.AssertJSON(recorder, tc.ExpectedBody)
			}
		})
	}
}

// MockUser creates a mock user for testing.
func MockUser(id, email, username, name string) *domain.User {
	return &domain.User{
		ID:       id,
		Email:    email,
		Username: username,
		Name:     name,
		Role:     domain.RegularUserRole,
		Preferences: domain.UserPreferences{
			Theme:    "light",
			Language: "en",
			Timezone: "UTC",
			Preferences: map[string]string{
				"notifications": "true",
			},
		},
	}
}

// MockProject creates a mock project for testing.
func MockProject(id, title, slug, ownerID string) *domain.Project {
	return &domain.Project{
		ID:          id,
		Title:       title,
		Slug:        slug,
		Description: "Test project description",
		OwnerID:     ownerID,
		Status:      domain.ActiveProject,
		Settings: domain.ProjectSettings{
			CustomFields:   make(map[string]string),
			Notifications:  make(map[string]bool),
			IsPrivate:      false,
			AllowGuestView: true,
			EnableComments: true,
		},
		MemberIDs: []string{},
	}
}

// MockTask creates a mock task for testing.
func MockTask(id, title, projectID, reporterID string) *domain.Task {
	now := time.Now()
	return &domain.Task{
		ID:           id,
		Title:        title,
		Description:  "Test task description",
		ProjectID:    projectID,
		ReporterID:   reporterID,
		Status:       domain.StatusTodo,
		Priority:     domain.PriorityMedium,
		Progress:     0,
		TimeSpent:    0.0,
		Position:     1,
		CreatedAt:    now,
		UpdatedAt:    now,
		Tags:         []string{},
		Dependencies: []string{},
		Attachments:  []string{},
	}
}

// jsonEqual compares two JSON objects for equality.
func jsonEqual(a, b map[string]interface{}) bool {
	aBytes, _ := json.Marshal(a)
	bBytes, _ := json.Marshal(b)
	return bytes.Equal(aBytes, bBytes)
}

// TestContextWithValue creates a test context with a value.
func TestContextWithValue(key, value interface{}) context.Context {
	return context.WithValue(context.Background(), key, value)
}

// TestUserKeyType is the type used for user context key.
type TestUserKeyType string

// TestUserKey is the key used to store user in test context.
const TestUserKey TestUserKeyType = "user"

// TestContextWithUser creates a test context with a user.
func TestContextWithUser(user *domain.User) context.Context {
	return context.WithValue(context.Background(), TestUserKey, user)
}
