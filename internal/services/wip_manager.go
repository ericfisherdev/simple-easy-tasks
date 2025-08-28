package services

import (
	"context"
	"fmt"

	"github.com/ericfisherdev/simple-easy-tasks/internal/domain"
	"github.com/ericfisherdev/simple-easy-tasks/internal/repository"
)

// WIPManager handles Work-in-Progress limit enforcement
type WIPManager interface {
	// ValidateWIPLimit checks if moving a task would violate WIP limits
	ValidateWIPLimit(ctx context.Context, projectID string, targetStatus domain.TaskStatus) error

	// GetWIPLimits retrieves WIP limits for a project column
	GetWIPLimits(ctx context.Context, projectID string, status domain.TaskStatus) (*WIPLimits, error)

	// SetWIPLimits configures WIP limits for a project column
	SetWIPLimits(ctx context.Context, projectID string, status domain.TaskStatus, limits WIPLimits) error

	// GetWIPStatus returns current WIP status for all columns
	GetWIPStatus(ctx context.Context, projectID string) (map[domain.TaskStatus]*WIPStatus, error)

	// CheckWIPViolations identifies columns that are violating WIP limits
	CheckWIPViolations(ctx context.Context, projectID string) ([]*WIPViolation, error)
}

// WIPStatus represents the current WIP status for a column
type WIPStatus struct {
	Status        domain.TaskStatus `json:"status"`
	CurrentCount  int               `json:"current_count"`
	Limits        *WIPLimits        `json:"limits"`
	IsViolating   bool              `json:"is_violating"`
	ViolationType string            `json:"violation_type,omitempty"` // "soft" or "hard"
}

// WIPViolation represents a WIP limit violation
type WIPViolation struct {
	ProjectID     string            `json:"project_id"`
	Status        domain.TaskStatus `json:"status"`
	CurrentCount  int               `json:"current_count"`
	Limit         int               `json:"limit"`
	ViolationType string            `json:"violation_type"` // "soft" or "hard"
	Severity      string            `json:"severity"`       // "warning" or "error"
}

// WIPOverrideReason represents reasons for overriding WIP limits
type WIPOverrideReason string

const (
	// OverrideEmergency allows bypassing WIP limits for emergency tasks
	OverrideEmergency WIPOverrideReason = "emergency"
	// OverrideHotfix allows bypassing WIP limits for critical hotfixes
	OverrideHotfix WIPOverrideReason = "hotfix"
	// OverrideBlocker allows bypassing WIP limits for blocking issues
	OverrideBlocker WIPOverrideReason = "blocker"
	// OverrideManagement allows bypassing WIP limits for management requests\n	OverrideManagement WIPOverrideReason = "management"
)

// WIPOverride represents a WIP limit override request
type WIPOverride struct {
	Reason    WIPOverrideReason `json:"reason"`
	Comment   string            `json:"comment"`
	UserID    string            `json:"user_id"`
	ExpiresAt *string           `json:"expires_at,omitempty"`
}

// wipManager implements WIP limit management
type wipManager struct {
	taskRepo    repository.TaskRepository
	projectRepo repository.ProjectRepository
	// In a real implementation, you'd have a WIP configuration repository
	// wipConfigRepo repository.WIPConfigRepository
}

// NewWIPManager creates a new WIP manager
func NewWIPManager(
	taskRepo repository.TaskRepository,
	projectRepo repository.ProjectRepository,
) WIPManager {
	return &wipManager{
		taskRepo:    taskRepo,
		projectRepo: projectRepo,
	}
}

// ValidateWIPLimit checks if moving a task would violate WIP limits
func (wm *wipManager) ValidateWIPLimit(ctx context.Context, projectID string, targetStatus domain.TaskStatus) error {
	if projectID == "" {
		return domain.NewValidationError("INVALID_PROJECT_ID", "Project ID cannot be empty", nil)
	}

	// Get WIP limits for the target column
	limits, err := wm.GetWIPLimits(ctx, projectID, targetStatus)
	if err != nil {
		return err
	}

	// If WIP limits are not enabled, allow the move
	if !limits.Enabled {
		return nil
	}

	// Count current tasks in the target column
	currentCount, err := wm.getColumnTaskCount(ctx, projectID, targetStatus)
	if err != nil {
		return err
	}

	// Check hard limit violation
	if limits.HardLimit > 0 && currentCount >= limits.HardLimit {
		return domain.NewConflictError("WIP_HARD_LIMIT_VIOLATED",
			fmt.Sprintf("Moving task would violate hard WIP limit (%d) for %s column",
				limits.HardLimit, string(targetStatus)))
	}

	// Check soft limit violation (warning)
	if limits.SoftLimit > 0 && currentCount >= limits.SoftLimit {
		// Soft limit violation - could be allowed with warning or override
		return domain.NewValidationError("WIP_SOFT_LIMIT_VIOLATED",
			fmt.Sprintf("Moving task would exceed soft WIP limit (%d) for %s column",
				limits.SoftLimit, string(targetStatus)),
			map[string]interface{}{
				"current_count":      currentCount,
				"soft_limit":         limits.SoftLimit,
				"column":             string(targetStatus),
				"override_available": true,
			})
	}

	return nil
}

// GetWIPLimits retrieves WIP limits for a project column
func (wm *wipManager) GetWIPLimits(
	ctx context.Context, projectID string, status domain.TaskStatus,
) (*WIPLimits, error) {
	if projectID == "" {
		return nil, domain.NewValidationError("INVALID_PROJECT_ID", "Project ID cannot be empty", nil)
	}

	// Validate project exists
	_, err := wm.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, domain.NewNotFoundError("PROJECT_NOT_FOUND", "Project not found")
	}

	// TODO: In a real implementation, retrieve from WIP configuration storage
	// For now, return default limits based on column type
	limits := wm.getDefaultWIPLimits(status)

	return limits, nil
}

// SetWIPLimits configures WIP limits for a project column
func (wm *wipManager) SetWIPLimits(ctx context.Context, projectID string, _ domain.TaskStatus, limits WIPLimits) error {
	if projectID == "" {
		return domain.NewValidationError("INVALID_PROJECT_ID", "Project ID cannot be empty", nil)
	}

	// Validate project exists
	_, err := wm.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return domain.NewNotFoundError("PROJECT_NOT_FOUND", "Project not found")
	}

	// Validate limits
	if err := wm.validateWIPLimits(limits); err != nil {
		return err
	}

	// TODO: In a real implementation, store in WIP configuration storage
	// For now, this is a placeholder
	// return wm.wipConfigRepo.SetLimits(ctx, projectID, status, limits)

	return nil
}

// GetWIPStatus returns current WIP status for all columns
func (wm *wipManager) GetWIPStatus(ctx context.Context, projectID string) (map[domain.TaskStatus]*WIPStatus, error) {
	if projectID == "" {
		return nil, domain.NewValidationError("INVALID_PROJECT_ID", "Project ID cannot be empty", nil)
	}

	// Validate project exists
	_, err := wm.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, domain.NewNotFoundError("PROJECT_NOT_FOUND", "Project not found")
	}

	statuses := []domain.TaskStatus{
		domain.StatusBacklog,
		domain.StatusTodo,
		domain.StatusDeveloping,
		domain.StatusReview,
		domain.StatusComplete,
	}

	wipStatus := make(map[domain.TaskStatus]*WIPStatus)

	for _, status := range statuses {
		// Get current task count
		currentCount, err := wm.getColumnTaskCount(ctx, projectID, status)
		if err != nil {
			return nil, err
		}

		// Get WIP limits
		limits, err := wm.GetWIPLimits(ctx, projectID, status)
		if err != nil {
			return nil, err
		}

		// Determine violation status
		isViolating := false
		violationType := ""

		if limits.Enabled {
			if limits.HardLimit > 0 && currentCount >= limits.HardLimit {
				isViolating = true
				violationType = "hard"
			} else if limits.SoftLimit > 0 && currentCount >= limits.SoftLimit {
				isViolating = true
				violationType = "soft"
			}
		}

		wipStatus[status] = &WIPStatus{
			Status:        status,
			CurrentCount:  currentCount,
			Limits:        limits,
			IsViolating:   isViolating,
			ViolationType: violationType,
		}
	}

	return wipStatus, nil
}

// CheckWIPViolations identifies columns that are violating WIP limits
func (wm *wipManager) CheckWIPViolations(ctx context.Context, projectID string) ([]*WIPViolation, error) {
	wipStatus, err := wm.GetWIPStatus(ctx, projectID)
	if err != nil {
		return nil, err
	}

	var violations []*WIPViolation

	for status, statusInfo := range wipStatus {
		if statusInfo.IsViolating {
			violation := &WIPViolation{
				ProjectID:     projectID,
				Status:        status,
				CurrentCount:  statusInfo.CurrentCount,
				ViolationType: statusInfo.ViolationType,
			}

			// Set limit and severity based on violation type
			if statusInfo.ViolationType == "hard" {
				violation.Limit = statusInfo.Limits.HardLimit
				violation.Severity = "error"
			} else {
				violation.Limit = statusInfo.Limits.SoftLimit
				violation.Severity = "warning"
			}

			violations = append(violations, violation)
		}
	}

	return violations, nil
}

// Helper methods

// getColumnTaskCount counts non-archived tasks in a specific column
func (wm *wipManager) getColumnTaskCount(ctx context.Context, projectID string, status domain.TaskStatus) (int, error) {
	filters := repository.TaskFilters{
		Status: []domain.TaskStatus{status},
		Limit:  10000, // Large limit to count all tasks
	}

	tasks, err := wm.taskRepo.GetByProject(ctx, projectID, filters)
	if err != nil {
		return 0, domain.NewInternalError("WIP_COUNT_FAILED", "Failed to count column tasks", err)
	}

	// Count only non-archived tasks
	count := 0
	for _, task := range tasks {
		if !task.IsArchived() {
			count++
		}
	}

	return count, nil
}

// getDefaultWIPLimits returns sensible default WIP limits based on column type
func (wm *wipManager) getDefaultWIPLimits(status domain.TaskStatus) *WIPLimits {
	defaults := map[domain.TaskStatus]*WIPLimits{
		domain.StatusBacklog: {
			SoftLimit: 50,
			HardLimit: 100,
			Enabled:   false, // Backlog typically doesn't have WIP limits
		},
		domain.StatusTodo: {
			SoftLimit: 10,
			HardLimit: 15,
			Enabled:   false,
		},
		domain.StatusDeveloping: {
			SoftLimit: 5,
			HardLimit: 8,
			Enabled:   true, // Development typically has WIP limits
		},
		domain.StatusReview: {
			SoftLimit: 3,
			HardLimit: 5,
			Enabled:   true,
		},
		domain.StatusComplete: {
			SoftLimit: 0,
			HardLimit: 0,
			Enabled:   false, // Complete column doesn't need limits
		},
	}

	if limits, exists := defaults[status]; exists {
		return limits
	}

	// Fallback default
	return &WIPLimits{
		SoftLimit: 5,
		HardLimit: 10,
		Enabled:   false,
	}
}

// validateWIPLimits validates WIP limit configuration
func (wm *wipManager) validateWIPLimits(limits WIPLimits) error {
	if limits.Enabled {
		if limits.SoftLimit < 0 {
			return domain.NewValidationError("INVALID_SOFT_LIMIT", "Soft limit cannot be negative", nil)
		}
		if limits.HardLimit < 0 {
			return domain.NewValidationError("INVALID_HARD_LIMIT", "Hard limit cannot be negative", nil)
		}
		if limits.SoftLimit > limits.HardLimit && limits.HardLimit > 0 {
			return domain.NewValidationError("INVALID_LIMITS", "Soft limit cannot exceed hard limit", nil)
		}
	}
	return nil
}
