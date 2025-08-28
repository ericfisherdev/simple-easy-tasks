package services

import (
	"context"
	"fmt"
	"math"

	"simple-easy-tasks/internal/domain"
	"simple-easy-tasks/internal/repository"
)

// PositionManager handles sophisticated position management for drag-and-drop functionality
type PositionManager interface {
	// CalculateNewPosition calculates the optimal position for a task being moved
	CalculateNewPosition(ctx context.Context, req PositionRequest) (int, error)

	// RebalanceColumn rebalances positions in a column to prevent overflow
	RebalanceColumn(ctx context.Context, projectID string, status domain.TaskStatus) error

	// GetPositionBetween calculates a position between two existing positions
	GetPositionBetween(beforePos, afterPos int) int

	// ValidatePosition checks if a position is valid and available
	ValidatePosition(ctx context.Context, projectID string, status domain.TaskStatus, position int) error
}

// PositionRequest contains parameters for position calculation
type PositionRequest struct {
	ProjectID     string            `json:"project_id"`
	TaskID        string            `json:"task_id"`
	TargetStatus  domain.TaskStatus `json:"target_status"`
	InsertAtIndex int               `json:"insert_at_index"` // 0-based index in the target column
	BeforeTaskID  *string           `json:"before_task_id,omitempty"`
	AfterTaskID   *string           `json:"after_task_id,omitempty"`
}

// positionManager implements sophisticated position management
type positionManager struct {
	taskRepo repository.TaskRepository
}

// NewPositionManager creates a new position manager
func NewPositionManager(taskRepo repository.TaskRepository) PositionManager {
	return &positionManager{
		taskRepo: taskRepo,
	}
}

// Position constants for calculation
const (
	MinPosition        = 1000      // Minimum position value
	MaxPosition        = 999999000 // Maximum position value
	DefaultIncrement   = 1000      // Default increment between positions
	RebalanceThreshold = 100       // Minimum space before rebalancing
)

// CalculateNewPosition calculates the optimal position for a task being moved
func (pm *positionManager) CalculateNewPosition(ctx context.Context, req PositionRequest) (int, error) {
	if req.ProjectID == "" {
		return 0, domain.NewValidationError("INVALID_PROJECT_ID", "Project ID cannot be empty", nil)
	}

	if !req.TargetStatus.IsValid() {
		return 0, domain.NewValidationError("INVALID_STATUS", "Invalid target status", nil)
	}

	// Get all tasks in the target column, ordered by position
	filters := repository.TaskFilters{
		Status:    []domain.TaskStatus{req.TargetStatus},
		SortBy:    "position",
		SortOrder: "asc",
		Limit:     1000,
	}

	columnTasks, err := pm.taskRepo.GetByProject(ctx, req.ProjectID, filters)
	if err != nil {
		return 0, domain.NewInternalError("POSITION_CALC_FAILED", "Failed to get column tasks", err)
	}

	// Filter out the task being moved if it's in the same column
	var filteredTasks []*domain.Task
	for _, task := range columnTasks {
		if task.ID != req.TaskID {
			filteredTasks = append(filteredTasks, task)
		}
	}
	columnTasks = filteredTasks

	// Handle empty column
	if len(columnTasks) == 0 {
		return MinPosition, nil
	}

	// Handle insertion at specific index
	if req.InsertAtIndex >= 0 {
		return pm.calculatePositionAtIndex(columnTasks, req.InsertAtIndex)
	}

	// Handle insertion relative to specific tasks
	if req.BeforeTaskID != nil || req.AfterTaskID != nil {
		return pm.calculatePositionRelative(columnTasks, req.BeforeTaskID, req.AfterTaskID)
	}

	// Default: append to end
	lastTask := columnTasks[len(columnTasks)-1]
	return pm.getNextPosition(lastTask.Position), nil
}

// calculatePositionAtIndex calculates position for insertion at a specific index
func (pm *positionManager) calculatePositionAtIndex(columnTasks []*domain.Task, index int) (int, error) {
	numTasks := len(columnTasks)

	// Insert at beginning
	if index <= 0 {
		if numTasks == 0 {
			return MinPosition, nil
		}
		return pm.getPrevPosition(columnTasks[0].Position), nil
	}

	// Insert at end
	if index >= numTasks {
		return pm.getNextPosition(columnTasks[numTasks-1].Position), nil
	}

	// Insert in middle
	beforePos := columnTasks[index-1].Position
	afterPos := columnTasks[index].Position

	return pm.GetPositionBetween(beforePos, afterPos), nil
}

// calculatePositionRelative calculates position relative to specific tasks
func (pm *positionManager) calculatePositionRelative(columnTasks []*domain.Task, beforeTaskID, afterTaskID *string) (int, error) {
	var beforePos, afterPos int
	var beforeFound, afterFound bool

	// Find the reference tasks and their positions
	for _, task := range columnTasks {
		if beforeTaskID != nil && task.ID == *beforeTaskID {
			afterPos = task.Position
			afterFound = true
		}
		if afterTaskID != nil && task.ID == *afterTaskID {
			beforePos = task.Position
			beforeFound = true
		}
	}

	// Validate that reference tasks were found
	if beforeTaskID != nil && !afterFound {
		return 0, domain.NewValidationError("BEFORE_TASK_NOT_FOUND", "Before task not found in column", nil)
	}
	if afterTaskID != nil && !beforeFound {
		return 0, domain.NewValidationError("AFTER_TASK_NOT_FOUND", "After task not found in column", nil)
	}

	// Calculate position between reference tasks
	if beforeFound && afterFound {
		return pm.GetPositionBetween(beforePos, afterPos), nil
	}

	// Position before a specific task
	if afterFound {
		return pm.getPrevPosition(afterPos), nil
	}

	// Position after a specific task
	if beforeFound {
		return pm.getNextPosition(beforePos), nil
	}

	// This shouldn't happen due to validation above
	return MinPosition, nil
}

// GetPositionBetween calculates a position between two existing positions using fractional positioning
func (pm *positionManager) GetPositionBetween(beforePos, afterPos int) int {
	if beforePos >= afterPos {
		return afterPos + DefaultIncrement
	}

	// Calculate midpoint
	midpoint := beforePos + (afterPos-beforePos)/2

	// Ensure minimum gap
	if midpoint <= beforePos {
		midpoint = beforePos + 1
	}
	if midpoint >= afterPos {
		midpoint = afterPos - 1
	}

	// If no space for midpoint, trigger rebalancing
	if midpoint <= beforePos || midpoint >= afterPos {
		// Return a position that will trigger rebalancing
		return beforePos + 1
	}

	return midpoint
}

// getNextPosition calculates the next position after the given position
func (pm *positionManager) getNextPosition(currentPos int) int {
	nextPos := currentPos + DefaultIncrement
	if nextPos > MaxPosition {
		// Position overflow - use default increment from current
		return currentPos + 100
	}
	return nextPos
}

// getPrevPosition calculates the previous position before the given position
func (pm *positionManager) getPrevPosition(currentPos int) int {
	prevPos := currentPos - DefaultIncrement
	if prevPos < MinPosition {
		// Position underflow - use half the current position
		return int(math.Max(float64(MinPosition), float64(currentPos)/2))
	}
	return prevPos
}

// RebalanceColumn rebalances positions in a column to prevent overflow and ensure proper spacing
func (pm *positionManager) RebalanceColumn(ctx context.Context, projectID string, status domain.TaskStatus) error {
	if projectID == "" {
		return domain.NewValidationError("INVALID_PROJECT_ID", "Project ID cannot be empty", nil)
	}

	// Get all tasks in the column, ordered by current position
	filters := repository.TaskFilters{
		Status:    []domain.TaskStatus{status},
		SortBy:    "position",
		SortOrder: "asc",
		Limit:     1000,
	}

	tasks, err := pm.taskRepo.GetByProject(ctx, projectID, filters)
	if err != nil {
		return domain.NewInternalError("REBALANCE_FAILED", "Failed to get tasks for rebalancing", err)
	}

	if len(tasks) == 0 {
		return nil // Nothing to rebalance
	}

	// Calculate new evenly spaced positions
	totalRange := MaxPosition - MinPosition
	increment := totalRange / (len(tasks) + 1)

	// Ensure minimum increment
	if increment < DefaultIncrement {
		increment = DefaultIncrement
	}

	// Update positions
	for i, task := range tasks {
		newPosition := MinPosition + (i+1)*increment

		// Only update if position changed significantly
		if abs(task.Position-newPosition) > RebalanceThreshold {
			task.Position = newPosition

			if err := pm.taskRepo.Update(ctx, task); err != nil {
				return domain.NewInternalError("REBALANCE_UPDATE_FAILED",
					fmt.Sprintf("Failed to update position for task %s", task.ID), err)
			}
		}
	}

	return nil
}

// ValidatePosition checks if a position is valid and doesn't conflict
func (pm *positionManager) ValidatePosition(ctx context.Context, projectID string, status domain.TaskStatus, position int) error {
	if position < MinPosition || position > MaxPosition {
		return domain.NewValidationError("INVALID_POSITION",
			fmt.Sprintf("Position must be between %d and %d", MinPosition, MaxPosition), nil)
	}

	// Check for position conflicts (optional strict validation)
	filters := repository.TaskFilters{
		Status: []domain.TaskStatus{status},
		Limit:  1000,
	}

	tasks, err := pm.taskRepo.GetByProject(ctx, projectID, filters)
	if err != nil {
		return domain.NewInternalError("POSITION_VALIDATION_FAILED", "Failed to validate position", err)
	}

	// Check for exact position conflicts
	for _, task := range tasks {
		if task.Position == position {
			return domain.NewConflictError("POSITION_CONFLICT",
				fmt.Sprintf("Position %d is already occupied", position))
		}
	}

	return nil
}

// ShouldRebalance determines if a column needs rebalancing based on position density
func (pm *positionManager) ShouldRebalance(ctx context.Context, projectID string, status domain.TaskStatus) (bool, error) {
	filters := repository.TaskFilters{
		Status:    []domain.TaskStatus{status},
		SortBy:    "position",
		SortOrder: "asc",
		Limit:     1000,
	}

	tasks, err := pm.taskRepo.GetByProject(ctx, projectID, filters)
	if err != nil {
		return false, err
	}

	if len(tasks) < 2 {
		return false, nil
	}

	// Check if any adjacent positions are too close
	for i := 1; i < len(tasks); i++ {
		gap := tasks[i].Position - tasks[i-1].Position
		if gap < RebalanceThreshold {
			return true, nil
		}
	}

	// Check for position overflow risk
	lastPos := tasks[len(tasks)-1].Position
	if lastPos > MaxPosition-DefaultIncrement {
		return true, nil
	}

	return false, nil
}

// Helper function for absolute value
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
