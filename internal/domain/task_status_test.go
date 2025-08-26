package domain

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTask_UpdateStatus_WithValidTransition(t *testing.T) {
	task := NewTask("Test Task", "Description", "project-1", "user-1")
	task.Status = StatusTodo

	err := task.UpdateStatus(StatusDeveloping)
	assert.NoError(t, err)
	assert.Equal(t, StatusDeveloping, task.Status)
}

func TestTask_UpdateStatus_WithInvalidTransition(t *testing.T) {
	task := NewTask("Test Task", "Description", "project-1", "user-1")
	task.Status = StatusTodo

	err := task.UpdateStatus(StatusComplete)
	assert.Error(t, err)

	// Verify it's a domain error with proper details
	domainErr, ok := err.(*Error)
	require.True(t, ok)
	assert.Equal(t, ConflictError, domainErr.Type)
	assert.Equal(t, "invalid_transition", domainErr.Code)

	// Check that helpful details are included
	assert.Contains(t, domainErr.Details, "current_status")
	assert.Contains(t, domainErr.Details, "requested_status")
	assert.Contains(t, domainErr.Details, "allowed_transitions")

	// Status should remain unchanged
	assert.Equal(t, StatusTodo, task.Status)
}

func TestTask_UpdateStatus_WithInvalidStatus(t *testing.T) {
	task := NewTask("Test Task", "Description", "project-1", "user-1")

	err := task.UpdateStatus("invalid_status")
	assert.Error(t, err)

	domainErr, ok := err.(*Error)
	require.True(t, ok)
	assert.Equal(t, ValidationError, domainErr.Type)
	assert.Equal(t, "status", domainErr.Code)
}

func TestTask_UpdateProgress_AutoTransitionSuccess(t *testing.T) {
	task := NewTask("Test Task", "Description", "project-1", "user-1")
	task.Status = StatusReview // Can transition to complete

	err := task.UpdateProgress(100)
	assert.NoError(t, err)
	assert.Equal(t, 100, task.Progress)
	assert.Equal(t, StatusComplete, task.Status) // Should auto-transition
}

func TestTask_UpdateProgress_AutoTransitionFailure(t *testing.T) {
	task := NewTask("Test Task", "Description", "project-1", "user-1")
	task.Status = StatusBacklog // Cannot transition directly to complete

	err := task.UpdateProgress(100)
	assert.NoError(t, err) // Progress update should still succeed
	assert.Equal(t, 100, task.Progress)
	assert.Equal(t, StatusBacklog, task.Status) // Status should remain unchanged

	// Check that transition failure is recorded in custom fields
	var customData map[string]interface{}
	err = json.Unmarshal(task.CustomFields, &customData)
	assert.NoError(t, err)

	failedTransition, exists := customData["failed_auto_transition"]
	assert.True(t, exists)

	transitionData, ok := failedTransition.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "backlog", transitionData["from_status"])
	assert.Equal(t, "complete", transitionData["to_status"])
	assert.Equal(t, "progress_completion", transitionData["trigger"])
	assert.Equal(t, float64(100), transitionData["progress_at_time"])
	assert.NotEmpty(t, transitionData["error"])
}

func TestTask_CanTransitionTo_ValidTransitions(t *testing.T) {
	validTransitions := map[TaskStatus][]TaskStatus{
		StatusBacklog:    {StatusTodo, StatusDeveloping},
		StatusTodo:       {StatusBacklog, StatusDeveloping},
		StatusDeveloping: {StatusTodo, StatusReview, StatusBacklog},
		StatusReview:     {StatusDeveloping, StatusComplete, StatusTodo},
		StatusComplete:   {StatusReview, StatusTodo},
	}

	for fromStatus, toStatuses := range validTransitions {
		task := &Task{Status: fromStatus}
		for _, toStatus := range toStatuses {
			assert.True(t, task.CanTransitionTo(toStatus),
				"Should be able to transition from %s to %s", fromStatus, toStatus)
		}
	}
}

func TestTask_CanTransitionTo_InvalidTransitions(t *testing.T) {
	// Test some invalid transitions
	invalidTransitions := map[TaskStatus][]TaskStatus{
		StatusBacklog:  {StatusReview, StatusComplete},
		StatusTodo:     {StatusReview, StatusComplete},
		StatusComplete: {StatusBacklog, StatusDeveloping},
	}

	for fromStatus, toStatuses := range invalidTransitions {
		task := &Task{Status: fromStatus}
		for _, toStatus := range toStatuses {
			assert.False(t, task.CanTransitionTo(toStatus),
				"Should not be able to transition from %s to %s", fromStatus, toStatus)
		}
	}
}
