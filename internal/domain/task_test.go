package domain_test

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"simple-easy-tasks/internal/domain"
)

func TestNewTask(t *testing.T) {
	title := "Test Task"
	description := "Test Description"
	projectID := "proj-123"
	reporterID := "user-456"

	task := domain.NewTask(title, description, projectID, reporterID)

	if task.Title != title {
		t.Errorf("Expected title %s, got %s", title, task.Title)
	}

	if task.Description != description {
		t.Errorf("Expected description %s, got %s", description, task.Description)
	}

	if task.ProjectID != projectID {
		t.Errorf("Expected project ID %s, got %s", projectID, task.ProjectID)
	}

	if task.ReporterID != reporterID {
		t.Errorf("Expected reporter ID %s, got %s", reporterID, task.ReporterID)
	}

	if task.Status != domain.StatusBacklog {
		t.Errorf("Expected default status %s, got %s", domain.StatusBacklog, task.Status)
	}

	if task.Priority != domain.PriorityMedium {
		t.Errorf("Expected default priority %s, got %s", domain.PriorityMedium, task.Priority)
	}

	if task.TimeSpent != 0 {
		t.Errorf("Expected time spent to be 0, got %f", task.TimeSpent)
	}

	if task.Progress != 0 {
		t.Errorf("Expected progress to be 0, got %d", task.Progress)
	}
}

func TestTask_ValidateRequiredFields(t *testing.T) {
	tests := []struct {
		name        string
		title       string
		projectID   string
		reporterID  string
		errorCode   string
		shouldError bool
	}{
		{
			name: "valid fields", title: "Test Task", projectID: "proj-123",
			reporterID: "user-456", shouldError: false, errorCode: "",
		},
		{
			name: "empty title", title: "", projectID: "proj-123",
			reporterID: "user-456", shouldError: true, errorCode: "title",
		},
		{
			name: "whitespace title", title: "   ", projectID: "proj-123",
			reporterID: "user-456", shouldError: true, errorCode: "title",
		},
		{
			name: "empty project ID", title: "Test Task", projectID: "",
			reporterID: "user-456", shouldError: true, errorCode: "project_id",
		},
		{
			name: "whitespace project ID", title: "Test Task", projectID: "   ",
			reporterID: "user-456", shouldError: true, errorCode: "project_id",
		},
		{
			name: "empty reporter ID", title: "Test Task", projectID: "proj-123",
			reporterID: "", shouldError: true, errorCode: "reporter_id",
		},
		{
			name: "whitespace reporter ID", title: "Test Task", projectID: "proj-123",
			reporterID: "   ", shouldError: true, errorCode: "reporter_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := domain.NewTask(tt.title, "description", tt.projectID, tt.reporterID)

			err := task.Validate()

			if tt.shouldError {
				if err == nil {
					t.Error("Expected validation error but got nil")
					return
				}

				domainErr, ok := err.(*domain.Error)
				if !ok {
					t.Errorf("Expected domain.Error, got %T", err)
					return
				}

				if domainErr.Code != tt.errorCode {
					t.Errorf("Expected error code %s, got %s", tt.errorCode, domainErr.Code)
				}
			} else if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestTask_ValidateDateRules(t *testing.T) {
	now := time.Now().UTC()
	yesterday := now.AddDate(0, 0, -1)
	tomorrow := now.AddDate(0, 0, 1)

	tests := []struct {
		dueDate     *time.Time
		startDate   *time.Time
		name        string
		description string
		shouldError bool
	}{
		{
			name: "no dates", dueDate: nil, startDate: nil, shouldError: false,
			description: "should pass with no dates",
		},
		{
			name: "future due date", dueDate: &tomorrow, startDate: nil, shouldError: false,
			description: "should pass with future due date",
		},
		{
			name: "today due date", dueDate: &now, startDate: nil, shouldError: false,
			description: "should pass with today's due date",
		},
		{
			name: "past due date", dueDate: &yesterday, startDate: nil, shouldError: true,
			description: "should fail with past due date",
		},
		{
			name: "valid date range", dueDate: &tomorrow, startDate: &now, shouldError: false,
			description: "should pass when start <= due",
		},
		{
			name: "invalid date range", dueDate: &now, startDate: &tomorrow, shouldError: true,
			description: "should fail when start > due",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := domain.NewTask("Test Task", "description", "proj-123", "user-456")
			task.DueDate = tt.dueDate
			task.StartDate = tt.startDate

			err := task.Validate()

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected validation error: %s", tt.description)
					return
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error (%s), got: %v", tt.description, err)
				}
			}
		})
	}
}

func TestTask_UpdateProgress(t *testing.T) {
	tests := []struct {
		name             string
		initialStatus    domain.TaskStatus
		expectedStatus   domain.TaskStatus
		progress         int
		shouldError      bool
		shouldTransition bool
	}{
		{
			name: "valid progress from developing", initialStatus: domain.StatusDeveloping,
			progress: 50, shouldError: false, expectedStatus: domain.StatusDeveloping, shouldTransition: false,
		},
		{
			name: "100% from developing - no auto transition", initialStatus: domain.StatusDeveloping,
			progress: 100, shouldError: false, expectedStatus: domain.StatusDeveloping, shouldTransition: false,
		},
		{
			name: "100% from review - should auto transition", initialStatus: domain.StatusReview,
			progress: 100, shouldError: false, expectedStatus: domain.StatusComplete, shouldTransition: true,
		},
		{
			name: "negative progress", initialStatus: domain.StatusDeveloping,
			progress: -1, shouldError: true, expectedStatus: domain.StatusDeveloping, shouldTransition: false,
		},
		{
			name: "progress over 100", initialStatus: domain.StatusDeveloping,
			progress: 101, shouldError: true, expectedStatus: domain.StatusDeveloping, shouldTransition: false,
		},
		{
			name: "zero progress", initialStatus: domain.StatusDeveloping,
			progress: 0, shouldError: false, expectedStatus: domain.StatusDeveloping, shouldTransition: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := domain.NewTask("Test Task", "description", "proj-123", "user-456")
			task.Status = tt.initialStatus

			err := task.UpdateProgress(tt.progress)

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got nil")
					return
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
					return
				}

				if task.Progress != tt.progress {
					t.Errorf("Expected progress %d, got %d", tt.progress, task.Progress)
				}

				if task.Status != tt.expectedStatus {
					t.Errorf("Expected status %s, got %s", tt.expectedStatus, task.Status)
				}
			}
		})
	}
}

func TestTask_AddTimeSpent(t *testing.T) {
	tests := []struct {
		name        string
		description string
		hours       float64
		shouldError bool
	}{
		{name: "positive hours", hours: 2.5, shouldError: false, description: "should accept positive hours"},
		{name: "zero hours", hours: 0.0, shouldError: false, description: "should accept zero hours"},
		{name: "negative hours", hours: -1.0, shouldError: true, description: "should reject negative hours"},
		{name: "NaN hours", hours: math.NaN(), shouldError: true, description: "should reject NaN"},
		{name: "positive infinity", hours: math.Inf(1), shouldError: true, description: "should reject positive infinity"},
		{name: "negative infinity", hours: math.Inf(-1), shouldError: true, description: "should reject negative infinity"},
		{name: "very small positive", hours: 0.01, shouldError: false, description: "should accept small positive values"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := domain.NewTask("Test Task", "description", "proj-123", "user-456")
			initialTime := task.TimeSpent

			err := task.AddTimeSpent(tt.hours)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error: %s", tt.description)
					return
				}

				if task.TimeSpent != initialTime {
					t.Errorf("Time spent should not change on error. Expected %f, got %f", initialTime, task.TimeSpent)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error (%s), got: %v", tt.description, err)
					return
				}

				expected := initialTime + tt.hours
				if task.TimeSpent != expected {
					t.Errorf("Expected time spent %f, got %f", expected, task.TimeSpent)
				}
			}
		})
	}
}

func TestTask_CanTransitionTo(t *testing.T) {
	tests := []struct {
		from          domain.TaskStatus
		to            domain.TaskStatus
		canTransition bool
	}{
		// From Backlog
		{domain.StatusBacklog, domain.StatusTodo, true},
		{domain.StatusBacklog, domain.StatusDeveloping, true},
		{domain.StatusBacklog, domain.StatusReview, false},
		{domain.StatusBacklog, domain.StatusComplete, false},
		{domain.StatusBacklog, domain.StatusBacklog, false},

		// From Todo
		{domain.StatusTodo, domain.StatusBacklog, true},
		{domain.StatusTodo, domain.StatusDeveloping, true},
		{domain.StatusTodo, domain.StatusReview, false},
		{domain.StatusTodo, domain.StatusComplete, false},
		{domain.StatusTodo, domain.StatusTodo, false},

		// From Developing
		{domain.StatusDeveloping, domain.StatusTodo, true},
		{domain.StatusDeveloping, domain.StatusReview, true},
		{domain.StatusDeveloping, domain.StatusBacklog, true},
		{domain.StatusDeveloping, domain.StatusComplete, false},
		{domain.StatusDeveloping, domain.StatusDeveloping, false},

		// From Review
		{domain.StatusReview, domain.StatusDeveloping, true},
		{domain.StatusReview, domain.StatusComplete, true},
		{domain.StatusReview, domain.StatusTodo, true},
		{domain.StatusReview, domain.StatusBacklog, false},
		{domain.StatusReview, domain.StatusReview, false},

		// From Complete
		{domain.StatusComplete, domain.StatusReview, true},
		{domain.StatusComplete, domain.StatusTodo, true},
		{domain.StatusComplete, domain.StatusBacklog, false},
		{domain.StatusComplete, domain.StatusDeveloping, false},
		{domain.StatusComplete, domain.StatusComplete, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.from)+"_to_"+string(tt.to), func(t *testing.T) {
			task := domain.NewTask("Test Task", "description", "proj-123", "user-456")
			task.Status = tt.from

			result := task.CanTransitionTo(tt.to)

			if result != tt.canTransition {
				t.Errorf("Expected %v for transition from %s to %s, got %v",
					tt.canTransition, tt.from, tt.to, result)
			}
		})
	}
}

func TestTask_SetParentTask(t *testing.T) {
	tests := []struct {
		name        string
		taskID      string
		parentID    string
		description string
		shouldError bool
	}{
		{
			name: "valid parent", taskID: "task-123", parentID: "parent-456",
			shouldError: false, description: "should accept valid parent ID",
		},
		{
			name: "self as parent", taskID: "task-123", parentID: "task-123",
			shouldError: true, description: "should reject self as parent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := domain.NewTask("Test Task", "description", "proj-123", "user-456")
			task.ID = tt.taskID

			err := task.SetParentTask(tt.parentID)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error: %s", tt.description)
					return
				}

				domainErr, ok := err.(*domain.Error)
				if !ok {
					t.Errorf("Expected domain.Error, got %T", err)
					return
				}

				if domainErr.Code != "circular_dependency" {
					t.Errorf("Expected error code 'circular_dependency', got %s", domainErr.Code)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error (%s), got: %v", tt.description, err)
					return
				}

				if task.ParentTaskID == nil || *task.ParentTaskID != tt.parentID {
					t.Errorf("Expected parent ID %s, got %v", tt.parentID, task.ParentTaskID)
				}
			}
		})
	}
}

func TestTask_GetColumnPositionMap(t *testing.T) {
	tests := []struct {
		setupFunc   func(*domain.Task)
		expected    map[string]int
		name        string
		description string
		shouldError bool
	}{
		{
			name:        "nil column position",
			setupFunc:   func(t *domain.Task) { t.ColumnPosition = nil },
			expected:    map[string]int{},
			shouldError: false,
			description: "should return empty map for nil",
		},
		{
			name:        "empty column position",
			setupFunc:   func(t *domain.Task) { t.ColumnPosition = json.RawMessage{} },
			expected:    map[string]int{},
			shouldError: false,
			description: "should return empty map for empty bytes",
		},
		{
			name:        "null JSON",
			setupFunc:   func(t *domain.Task) { t.ColumnPosition = json.RawMessage("null") },
			expected:    map[string]int{},
			shouldError: false,
			description: "should return empty map for null JSON",
		},
		{
			name: "valid JSON",
			setupFunc: func(t *domain.Task) {
				t.ColumnPosition = json.RawMessage(`{"col1": 1, "col2": 2}`)
			},
			expected:    map[string]int{"col1": 1, "col2": 2},
			shouldError: false,
			description: "should parse valid JSON",
		},
		{
			name: "invalid JSON",
			setupFunc: func(t *domain.Task) {
				t.ColumnPosition = json.RawMessage(`{invalid}`)
			},
			expected:    nil,
			shouldError: true,
			description: "should error on invalid JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := domain.NewTask("Test Task", "description", "proj-123", "user-456")
			tt.setupFunc(task)

			result, err := task.GetColumnPositionMap()

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error: %s", tt.description)
					return
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error (%s), got: %v", tt.description, err)
					return
				}

				if len(result) != len(tt.expected) {
					t.Errorf("Expected map length %d, got %d", len(tt.expected), len(result))
					return
				}

				for key, expectedValue := range tt.expected {
					if actualValue, exists := result[key]; !exists || actualValue != expectedValue {
						t.Errorf("Expected %s: %d, got %s: %d", key, expectedValue, key, actualValue)
					}
				}
			}
		})
	}
}

func TestTask_GetCustomFieldsMap(t *testing.T) {
	tests := []struct {
		setupFunc   func(*domain.Task)
		expected    map[string]interface{}
		name        string
		description string
		shouldError bool
	}{
		{
			name:        "nil custom fields",
			setupFunc:   func(t *domain.Task) { t.CustomFields = nil },
			expected:    map[string]interface{}{},
			shouldError: false,
			description: "should return empty map for nil",
		},
		{
			name:        "empty custom fields",
			setupFunc:   func(t *domain.Task) { t.CustomFields = json.RawMessage{} },
			expected:    map[string]interface{}{},
			shouldError: false,
			description: "should return empty map for empty bytes",
		},
		{
			name:        "null JSON",
			setupFunc:   func(t *domain.Task) { t.CustomFields = json.RawMessage("null") },
			expected:    map[string]interface{}{},
			shouldError: false,
			description: "should return empty map for null JSON",
		},
		{
			name: "valid JSON",
			setupFunc: func(t *domain.Task) {
				t.CustomFields = json.RawMessage(`{"field1": "value1", "field2": 42}`)
			},
			expected:    map[string]interface{}{"field1": "value1", "field2": float64(42)},
			shouldError: false,
			description: "should parse valid JSON",
		},
		{
			name: "invalid JSON",
			setupFunc: func(t *domain.Task) {
				t.CustomFields = json.RawMessage(`{invalid}`)
			},
			expected:    nil,
			shouldError: true,
			description: "should error on invalid JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := domain.NewTask("Test Task", "description", "proj-123", "user-456")
			tt.setupFunc(task)

			result, err := task.GetCustomFieldsMap()

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error: %s", tt.description)
					return
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error (%s), got: %v", tt.description, err)
					return
				}

				if len(result) != len(tt.expected) {
					t.Errorf("Expected map length %d, got %d", len(tt.expected), len(result))
					return
				}

				for key, expectedValue := range tt.expected {
					if actualValue, exists := result[key]; !exists {
						t.Errorf("Expected key %s to exist", key)
					} else {
						// Handle the fact that JSON unmarshaling converts numbers to float64
						switch ev := expectedValue.(type) {
						case float64:
							if av, ok := actualValue.(float64); !ok || av != ev {
								t.Errorf("Expected %s: %v, got %s: %v", key, expectedValue, key, actualValue)
							}
						default:
							if actualValue != expectedValue {
								t.Errorf("Expected %s: %v, got %s: %v", key, expectedValue, key, actualValue)
							}
						}
					}
				}
			}
		})
	}
}

func TestTask_IsOverdue(t *testing.T) {
	now := time.Now().UTC()
	pastTime := now.Add(-time.Hour)
	futureTime := now.Add(time.Hour)

	tests := []struct {
		name     string
		dueDate  *time.Time
		status   domain.TaskStatus
		expected bool
	}{
		{"no due date", nil, domain.StatusDeveloping, false},
		{"future due date", &futureTime, domain.StatusDeveloping, false},
		{"past due date - incomplete", &pastTime, domain.StatusDeveloping, true},
		{"past due date - complete", &pastTime, domain.StatusComplete, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := domain.NewTask("Test Task", "description", "proj-123", "user-456")
			task.DueDate = tt.dueDate
			task.Status = tt.status

			result := task.IsOverdue()

			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
