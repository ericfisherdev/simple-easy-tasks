package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/ericfisherdev/simple-easy-tasks/internal/domain"
)

// APIClient handles communication with the Simple Easy Tasks API
type APIClient struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

// NewAPIClient creates a new API client
func NewAPIClient(baseURL, token string) *APIClient {
	return &APIClient{
		BaseURL: baseURL,
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewAPIClientFromProfile creates an API client from a profile
func NewAPIClientFromProfile(profile *Profile) *APIClient {
	return NewAPIClient(profile.ServerURL, profile.Token)
}

// APIError represents an API error response
type APIError struct {
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
	Details    string `json:"details"`
}

func (e *APIError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("API error (%d): %s - %s", e.StatusCode, e.Message, e.Details)
	}
	return fmt.Sprintf("API error (%d): %s", e.StatusCode, e.Message)
}

// doRequest performs an HTTP request with authentication
func (c *APIClient) doRequest(method, endpoint string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader

	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	url := fmt.Sprintf("%s%s", c.BaseURL, endpoint)
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// handleResponse processes the HTTP response and handles errors
func (c *APIClient) handleResponse(resp *http.Response, result interface{}) error {
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		apiError := &APIError{
			StatusCode: resp.StatusCode,
		}

		// Try to parse error response
		var errorResp map[string]interface{}
		if json.Unmarshal(body, &errorResp) == nil {
			if msg, ok := errorResp["error"].(string); ok {
				apiError.Message = msg
			} else if msg, ok := errorResp["message"].(string); ok {
				apiError.Message = msg
			}
			if details, ok := errorResp["details"].(string); ok {
				apiError.Details = details
			}
		}

		if apiError.Message == "" {
			apiError.Message = string(body)
		}

		return apiError
	}

	if result != nil && len(body) > 0 {
		if err := json.Unmarshal(body, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// Health checks the API health
func (c *APIClient) Health() error {
	resp, err := c.doRequest("GET", "/api/health", nil)
	if err != nil {
		return err
	}

	var healthResp map[string]interface{}
	return c.handleResponse(resp, &healthResp)
}

// Login authenticates with email and password
func (c *APIClient) Login(email, password string) (*LoginResponse, error) {
	loginReq := map[string]string{
		"email":    email,
		"password": password,
	}

	resp, err := c.doRequest("POST", "/api/auth/login", loginReq)
	if err != nil {
		return nil, err
	}

	var loginResp LoginResponse
	err = c.handleResponse(resp, &loginResp)
	if err != nil {
		return nil, err
	}

	// Update client token
	c.Token = loginResp.Token

	return &loginResp, nil
}

// LoginResponse represents the response from login
type LoginResponse struct {
	Token string      `json:"token"`
	User  domain.User `json:"user"`
}

// GetProjects retrieves all projects
func (c *APIClient) GetProjects() ([]domain.Project, error) {
	resp, err := c.doRequest("GET", "/api/projects", nil)
	if err != nil {
		return nil, err
	}

	var projects []domain.Project
	err = c.handleResponse(resp, &projects)
	return projects, err
}

// GetProject retrieves a specific project
func (c *APIClient) GetProject(projectID string) (*domain.Project, error) {
	endpoint := fmt.Sprintf("/api/projects/%s", projectID)
	resp, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var project domain.Project
	err = c.handleResponse(resp, &project)
	return &project, err
}

// CreateProject creates a new project
func (c *APIClient) CreateProject(req *CreateProjectRequest) (*domain.Project, error) {
	resp, err := c.doRequest("POST", "/api/projects", req)
	if err != nil {
		return nil, err
	}

	var project domain.Project
	err = c.handleResponse(resp, &project)
	return &project, err
}

// CreateProjectRequest represents a project creation request
type CreateProjectRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// GetTasks retrieves tasks for a project
func (c *APIClient) GetTasks(projectID string, options *TaskListOptions) ([]domain.Task, error) {
	endpoint := fmt.Sprintf("/api/projects/%s/tasks", projectID)

	// Add query parameters
	if options != nil {
		params := url.Values{}
		if len(options.Status) > 0 {
			for _, status := range options.Status {
				params.Add("status", status)
			}
		}
		if options.Assignee != "" {
			params.Add("assignee", options.Assignee)
		}
		if len(options.Priority) > 0 {
			for _, priority := range options.Priority {
				params.Add("priority", priority)
			}
		}
		if options.Search != "" {
			params.Add("search", options.Search)
		}
		if options.Limit > 0 {
			params.Add("limit", fmt.Sprintf("%d", options.Limit))
		}

		if paramStr := params.Encode(); paramStr != "" {
			endpoint += "?" + paramStr
		}
	}

	resp, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var tasks []domain.Task
	err = c.handleResponse(resp, &tasks)
	return tasks, err
}

// TaskListOptions represents options for listing tasks
type TaskListOptions struct {
	Status   []string
	Assignee string
	Priority []string
	Tags     []string
	Search   string
	Limit    int
}

// CreateTask creates a new task
func (c *APIClient) CreateTask(projectID string, req *CreateTaskRequest) (*domain.Task, error) {
	endpoint := fmt.Sprintf("/api/projects/%s/tasks", projectID)
	resp, err := c.doRequest("POST", endpoint, req)
	if err != nil {
		return nil, err
	}

	var task domain.Task
	err = c.handleResponse(resp, &task)
	return &task, err
}

// CreateTaskRequest represents a task creation request
type CreateTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
	AssigneeID  string `json:"assignee_id,omitempty"`
	Status      string `json:"status,omitempty"`
}

// UpdateTask updates an existing task
func (c *APIClient) UpdateTask(projectID, taskID string, req *UpdateTaskRequest) (*domain.Task, error) {
	endpoint := fmt.Sprintf("/api/projects/%s/tasks/%s", projectID, taskID)
	resp, err := c.doRequest("PUT", endpoint, req)
	if err != nil {
		return nil, err
	}

	var task domain.Task
	err = c.handleResponse(resp, &task)
	return &task, err
}

// UpdateTaskRequest represents a task update request
type UpdateTaskRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Priority    *string `json:"priority,omitempty"`
	Status      *string `json:"status,omitempty"`
	AssigneeID  *string `json:"assignee_id,omitempty"`
}

// DeleteTask deletes a task
func (c *APIClient) DeleteTask(projectID, taskID string) error {
	endpoint := fmt.Sprintf("/api/projects/%s/tasks/%s", projectID, taskID)
	resp, err := c.doRequest("DELETE", endpoint, nil)
	if err != nil {
		return err
	}

	return c.handleResponse(resp, nil)
}

// TestConnection tests the connection to the API
func (c *APIClient) TestConnection() error {
	return c.Health()
}
