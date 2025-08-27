package services

import (
	"context"
	"log/slog"
	"time"

	"simple-easy-tasks/internal/domain"
)

// RealtimeTaskService extends TaskService with real-time event broadcasting capabilities
type RealtimeTaskService interface {
	TaskService
	// GetEventBroadcaster returns the event broadcaster for external use
	GetEventBroadcaster() EventBroadcaster
	// BroadcastTaskCommented broadcasts a task comment event (exported for external use)
	BroadcastTaskCommented(ctx context.Context, task *domain.Task, commentID, comment, authorID string) error
}

// realtimeTaskService wraps the existing task service with real-time capabilities
type realtimeTaskService struct {
	TaskService
	eventBroadcaster EventBroadcaster
	logger           *slog.Logger
}

// NewRealtimeTaskService creates a task service with real-time event broadcasting
func NewRealtimeTaskService(
	taskService TaskService,
	eventBroadcaster EventBroadcaster,
	logger *slog.Logger,
) RealtimeTaskService {
	if logger == nil {
		logger = slog.Default()
	}

	return &realtimeTaskService{
		TaskService:      taskService,
		eventBroadcaster: eventBroadcaster,
		logger:           logger,
	}
}

// GetEventBroadcaster returns the event broadcaster for external use
func (s *realtimeTaskService) GetEventBroadcaster() EventBroadcaster {
	return s.eventBroadcaster
}

// CreateTask creates a new task and broadcasts a creation event
func (s *realtimeTaskService) CreateTask(
	ctx context.Context,
	req domain.CreateTaskRequest,
	userID string,
) (*domain.Task, error) {
	// Create task using base service
	task, err := s.TaskService.CreateTask(ctx, req, userID)
	if err != nil {
		return nil, err
	}

	// Broadcast task creation event
	if err := s.broadcastTaskCreated(ctx, task, userID); err != nil {
		s.logger.Error("Failed to broadcast task creation event",
			"task_id", task.ID,
			"error", err)
		// Don't fail the operation if event broadcasting fails
	}

	return task, nil
}

// UpdateTask updates a task and broadcasts an update event
func (s *realtimeTaskService) UpdateTask(
	ctx context.Context,
	taskID string,
	req domain.UpdateTaskRequest,
	userID string,
) (*domain.Task, error) {
	return s.updateTaskWithBroadcast(ctx, taskID, userID, "task update",
		func() (*domain.Task, error) {
			return s.TaskService.UpdateTask(ctx, taskID, req, userID)
		})
}

// DeleteTask deletes a task and broadcasts a deletion event
func (s *realtimeTaskService) DeleteTask(ctx context.Context, taskID string, userID string) error {
	// Get task before deletion for event data
	task, err := s.GetTask(ctx, taskID, userID)
	if err != nil {
		return err
	}

	// Delete task using base service
	if err := s.TaskService.DeleteTask(ctx, taskID, userID); err != nil {
		return err
	}

	// Broadcast task deletion event
	if err := s.broadcastTaskDeleted(ctx, task, userID); err != nil {
		s.logger.Error("Failed to broadcast task deletion event",
			"task_id", taskID,
			"error", err)
	}

	return nil
}

// AssignTask assigns a task and broadcasts an assignment event
func (s *realtimeTaskService) AssignTask(
	ctx context.Context,
	taskID string,
	assigneeID string,
	userID string,
) (*domain.Task, error) {
	// Get original task for comparison
	originalTask, err := s.GetTask(ctx, taskID, userID)
	if err != nil {
		return nil, err
	}

	// Assign task using base service
	updatedTask, err := s.TaskService.AssignTask(ctx, taskID, assigneeID, userID)
	if err != nil {
		return nil, err
	}

	// Broadcast task assignment event
	if err := s.broadcastTaskAssigned(ctx, originalTask, updatedTask, userID); err != nil {
		s.logger.Error("Failed to broadcast task assignment event",
			"task_id", taskID,
			"error", err)
	}

	return updatedTask, nil
}

// UnassignTask removes assignment from a task and broadcasts an assignment event
func (s *realtimeTaskService) UnassignTask(ctx context.Context, taskID string, userID string) (*domain.Task, error) {
	// Get original task for comparison
	originalTask, err := s.GetTask(ctx, taskID, userID)
	if err != nil {
		return nil, err
	}

	// Unassign task using base service
	updatedTask, err := s.TaskService.UnassignTask(ctx, taskID, userID)
	if err != nil {
		return nil, err
	}

	// Broadcast task assignment event (unassignment is also an assignment event)
	if err := s.broadcastTaskAssigned(ctx, originalTask, updatedTask, userID); err != nil {
		s.logger.Error("Failed to broadcast task unassignment event",
			"task_id", taskID,
			"error", err)
	}

	return updatedTask, nil
}

// UpdateTaskStatus updates a task's status and broadcasts an update event
func (s *realtimeTaskService) UpdateTaskStatus(
	ctx context.Context,
	taskID string,
	status domain.TaskStatus,
	userID string,
) (*domain.Task, error) {
	return s.updateTaskWithBroadcast(ctx, taskID, userID, "task status update",
		func() (*domain.Task, error) {
			return s.TaskService.UpdateTaskStatus(ctx, taskID, status, userID)
		})
}

// MoveTask moves a task between statuses/positions and broadcasts a move event
func (s *realtimeTaskService) MoveTask(ctx context.Context, req MoveTaskRequest, userID string) error {
	// Get original task for comparison
	originalTask, err := s.GetTask(ctx, req.TaskID, userID)
	if err != nil {
		return err
	}

	// Move task using base service
	if moveErr := s.TaskService.MoveTask(ctx, req, userID); moveErr != nil {
		return moveErr
	}

	// Get updated task for event data
	updatedTask, err := s.GetTask(ctx, req.TaskID, userID)
	if err != nil {
		s.logger.Error("Failed to get updated task after move for event broadcasting",
			"task_id", req.TaskID,
			"error", err)
		return nil // Don't fail the move operation
	}

	// Broadcast task move event
	if err := s.broadcastTaskMoved(ctx, originalTask, updatedTask, userID); err != nil {
		s.logger.Error("Failed to broadcast task move event",
			"task_id", req.TaskID,
			"error", err)
	}

	return nil
}

// CreateSubtask creates a subtask and broadcasts a creation event
func (s *realtimeTaskService) CreateSubtask(
	ctx context.Context,
	parentTaskID string,
	req domain.CreateTaskRequest,
	userID string,
) (*domain.Task, error) {
	// Create subtask using base service
	subtask, err := s.TaskService.CreateSubtask(ctx, parentTaskID, req, userID)
	if err != nil {
		return nil, err
	}

	// Broadcast task creation event for subtask
	if err := s.broadcastTaskCreated(ctx, subtask, userID); err != nil {
		s.logger.Error("Failed to broadcast subtask creation event",
			"task_id", subtask.ID,
			"parent_task_id", parentTaskID,
			"error", err)
	}

	return subtask, nil
}

// Event broadcasting helper methods

// broadcastTaskCreated broadcasts a task creation event
func (s *realtimeTaskService) broadcastTaskCreated(ctx context.Context, task *domain.Task, userID string) error {
	eventData := &domain.TaskCreatedData{
		Task: task,
	}

	event, err := domain.NewTaskEvent(domain.TaskCreated, task.ID, task.ProjectID, userID, eventData)
	if err != nil {
		return err
	}

	return s.eventBroadcaster.BroadcastEvent(ctx, event)
}

// broadcastTaskUpdated broadcasts a task update event
func (s *realtimeTaskService) broadcastTaskUpdated(
	ctx context.Context, originalTask, updatedTask *domain.Task, userID string,
) error {
	// Calculate changes
	changes, oldValues := s.calculateTaskChanges(originalTask, updatedTask)

	eventData := &domain.TaskUpdatedData{
		Task:      updatedTask,
		Changes:   changes,
		OldValues: oldValues,
	}

	event, err := domain.NewTaskEvent(domain.TaskUpdated, updatedTask.ID, updatedTask.ProjectID, userID, eventData)
	if err != nil {
		return err
	}

	return s.eventBroadcaster.BroadcastEvent(ctx, event)
}

// broadcastTaskMoved broadcasts a task move event
func (s *realtimeTaskService) broadcastTaskMoved(
	ctx context.Context, originalTask, updatedTask *domain.Task, userID string,
) error {
	eventData := &domain.TaskMovedData{
		Task:        updatedTask,
		OldStatus:   originalTask.Status,
		NewStatus:   updatedTask.Status,
		OldPosition: originalTask.Position,
		NewPosition: updatedTask.Position,
	}

	event, err := domain.NewTaskEvent(domain.TaskMoved, updatedTask.ID, updatedTask.ProjectID, userID, eventData)
	if err != nil {
		return err
	}

	return s.eventBroadcaster.BroadcastEvent(ctx, event)
}

// broadcastTaskAssigned broadcasts a task assignment event
func (s *realtimeTaskService) broadcastTaskAssigned(
	ctx context.Context, originalTask, updatedTask *domain.Task, userID string,
) error {
	eventData := &domain.TaskAssignedData{
		Task:        updatedTask,
		OldAssignee: originalTask.AssigneeID,
		NewAssignee: updatedTask.AssigneeID,
		AssignedBy:  userID,
	}

	event, err := domain.NewTaskEvent(domain.TaskAssigned, updatedTask.ID, updatedTask.ProjectID, userID, eventData)
	if err != nil {
		return err
	}

	return s.eventBroadcaster.BroadcastEvent(ctx, event)
}

// broadcastTaskDeleted broadcasts a task deletion event
func (s *realtimeTaskService) broadcastTaskDeleted(ctx context.Context, task *domain.Task, userID string) error {
	eventData := &domain.TaskDeletedData{
		TaskID:    task.ID,
		TaskTitle: task.Title,
		DeletedBy: userID,
	}

	event, err := domain.NewTaskEvent(domain.TaskDeleted, task.ID, task.ProjectID, userID, eventData)
	if err != nil {
		return err
	}

	return s.eventBroadcaster.BroadcastEvent(ctx, event)
}

// broadcastTaskCommented broadcasts a task comment event
func (s *realtimeTaskService) BroadcastTaskCommented(
	ctx context.Context, task *domain.Task, commentID, comment, authorID string,
) error {
	eventData := &domain.TaskCommentedData{
		Task:      task,
		CommentID: commentID,
		Comment:   comment,
		Author:    authorID,
	}

	event, err := domain.NewTaskEvent(domain.TaskCommented, task.ID, task.ProjectID, authorID, eventData)
	if err != nil {
		return err
	}

	return s.eventBroadcaster.BroadcastEvent(ctx, event)
}

// calculateTaskChanges compares two tasks and returns the changes
func (s *realtimeTaskService) calculateTaskChanges(
	original, updated *domain.Task,
) (map[string]interface{}, map[string]interface{}) {
	changes := make(map[string]interface{})
	oldValues := make(map[string]interface{})

	s.compareBasicFields(original, updated, changes, oldValues)
	s.comparePointerFields(original, updated, changes, oldValues)
	s.compareTags(original, updated, changes, oldValues)

	return changes, oldValues
}

// compareBasicFields compares basic task fields
func (s *realtimeTaskService) compareBasicFields(
	original, updated *domain.Task, changes, oldValues map[string]interface{},
) {
	fieldComparisons := map[string][2]interface{}{
		"title":       {original.Title, updated.Title},
		"description": {original.Description, updated.Description},
		"status":      {original.Status, updated.Status},
		"priority":    {original.Priority, updated.Priority},
		"position":    {original.Position, updated.Position},
		"progress":    {original.Progress, updated.Progress},
		"time_spent":  {original.TimeSpent, updated.TimeSpent},
	}

	for field, values := range fieldComparisons {
		if values[0] != values[1] {
			changes[field] = values[1]
			oldValues[field] = values[0]
		}
	}
}

// comparePointerFields compares pointer fields (assignee, dates)
func (s *realtimeTaskService) comparePointerFields(
	original, updated *domain.Task, changes, oldValues map[string]interface{},
) {
	// Compare assignee
	if s.assigneeChanged(original.AssigneeID, updated.AssigneeID) {
		changes["assignee_id"] = updated.AssigneeID
		oldValues["assignee_id"] = original.AssigneeID
	}

	// Compare dates
	if s.dateChanged(original.DueDate, updated.DueDate) {
		changes["due_date"] = updated.DueDate
		oldValues["due_date"] = original.DueDate
	}

	if s.dateChanged(original.StartDate, updated.StartDate) {
		changes["start_date"] = updated.StartDate
		oldValues["start_date"] = original.StartDate
	}
}

// assigneeChanged checks if assignee pointer values differ
func (s *realtimeTaskService) assigneeChanged(original, updated *string) bool {
	if (original == nil) != (updated == nil) {
		return true
	}
	return original != nil && updated != nil && *original != *updated
}

// dateChanged checks if date pointer values differ
func (s *realtimeTaskService) dateChanged(original, updated *time.Time) bool {
	if (original == nil) != (updated == nil) {
		return true
	}
	return original != nil && updated != nil && !original.Equal(*updated)
}

// compareTags compares tag slices
func (s *realtimeTaskService) compareTags(original, updated *domain.Task, changes, oldValues map[string]interface{}) {
	if len(original.Tags) != len(updated.Tags) {
		changes["tags"] = updated.Tags
		oldValues["tags"] = original.Tags
		return
	}

	for i, tag := range original.Tags {
		if i >= len(updated.Tags) || tag != updated.Tags[i] {
			changes["tags"] = updated.Tags
			oldValues["tags"] = original.Tags
			return
		}
	}
}

// updateTaskWithBroadcast is a helper function to reduce code duplication
// for operations that update a task and broadcast an event
func (s *realtimeTaskService) updateTaskWithBroadcast(
	ctx context.Context,
	taskID string,
	userID string,
	operationName string,
	updateFunc func() (*domain.Task, error),
) (*domain.Task, error) {
	// Get original task for comparison
	originalTask, err := s.GetTask(ctx, taskID, userID)
	if err != nil {
		return nil, err
	}

	// Update task using provided function
	updatedTask, err := updateFunc()
	if err != nil {
		return nil, err
	}

	// Broadcast task update event
	if err := s.broadcastTaskUpdated(ctx, originalTask, updatedTask, userID); err != nil {
		s.logger.Error("Failed to broadcast "+operationName+" event",
			"task_id", taskID,
			"error", err)
	}

	return updatedTask, nil
}
