package services

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"simple-easy-tasks/internal/domain"
)

// SubscriptionManager defines the interface for managing real-time subscriptions
type SubscriptionManager interface {
	// CreateSubscription creates a new event subscription for a user
	CreateSubscription(ctx context.Context, req CreateSubscriptionRequest) (*domain.EventSubscription, error)

	// UpdateSubscription updates an existing subscription
	UpdateSubscription(ctx context.Context, subscriptionID string, req UpdateSubscriptionRequest) (*domain.EventSubscription, error)

	// DeleteSubscription removes a subscription
	DeleteSubscription(ctx context.Context, subscriptionID string, userID string) error

	// GetSubscription retrieves a subscription by ID
	GetSubscription(ctx context.Context, subscriptionID string, userID string) (*domain.EventSubscription, error)

	// ListUserSubscriptions lists all subscriptions for a user
	ListUserSubscriptions(ctx context.Context, userID string) ([]*domain.EventSubscription, error)

	// ValidateSubscriptionAccess checks if a user can access a subscription
	ValidateSubscriptionAccess(ctx context.Context, subscriptionID string, userID string) error

	// StartCleanupRoutine starts background cleanup of expired subscriptions
	StartCleanupRoutine(ctx context.Context, interval time.Duration)
}

// CreateSubscriptionRequest represents a request to create a new subscription
type CreateSubscriptionRequest struct {
	UserID     string                    `json:"user_id" binding:"required"`
	ProjectID  *string                   `json:"project_id,omitempty"`
	EventTypes []domain.TaskEventType    `json:"event_types" binding:"required,min=1"`
	Filters    map[string]string         `json:"filters,omitempty"`
}

// UpdateSubscriptionRequest represents a request to update a subscription
type UpdateSubscriptionRequest struct {
	EventTypes *[]domain.TaskEventType `json:"event_types,omitempty"`
	Filters    *map[string]string      `json:"filters,omitempty"`
	Active     *bool                   `json:"active,omitempty"`
}

// Validate validates the create subscription request
func (r *CreateSubscriptionRequest) Validate() error {
	if r.UserID == "" {
		return domain.NewValidationError("INVALID_USER_ID", "User ID cannot be empty", nil)
	}

	if len(r.EventTypes) == 0 {
		return domain.NewValidationError("NO_EVENT_TYPES", "At least one event type must be specified", nil)
	}

	for _, eventType := range r.EventTypes {
		if !eventType.IsValid() {
			return domain.NewValidationError("INVALID_EVENT_TYPE", fmt.Sprintf("Invalid event type: %s", eventType), nil)
		}
	}

	return nil
}

// subscriptionManager implements the SubscriptionManager interface
type subscriptionManager struct {
	broadcaster   EventBroadcaster
	projectRepo   ProjectRepository  // For project access validation
	userRepo      UserRepository    // For user validation
	logger        *slog.Logger
	mu            sync.RWMutex

	// Cleanup configuration
	cleanupInterval   time.Duration
	cleanupRunning    bool
	cleanupStop       chan struct{}
}

// ProjectRepository interface for project access validation
type ProjectRepository interface {
	GetByID(ctx context.Context, id string) (*domain.Project, error)
}

// UserRepository interface for user validation  
type UserRepository interface {
	GetByID(ctx context.Context, id string) (*domain.User, error)
}

// SubscriptionManagerConfig holds configuration for the subscription manager
type SubscriptionManagerConfig struct {
	Logger          *slog.Logger
	CleanupInterval time.Duration // Default cleanup interval (default: 5 minutes)
}

// NewSubscriptionManager creates a new subscription manager
func NewSubscriptionManager(
	broadcaster EventBroadcaster,
	projectRepo ProjectRepository,
	userRepo UserRepository,
	config SubscriptionManagerConfig,
) SubscriptionManager {
	if config.Logger == nil {
		config.Logger = slog.Default()
	}
	if config.CleanupInterval <= 0 {
		config.CleanupInterval = 5 * time.Minute
	}

	return &subscriptionManager{
		broadcaster:     broadcaster,
		projectRepo:     projectRepo,
		userRepo:        userRepo,
		logger:          config.Logger,
		cleanupInterval: config.CleanupInterval,
		cleanupStop:     make(chan struct{}),
	}
}

// CreateSubscription creates a new event subscription for a user
func (m *subscriptionManager) CreateSubscription(ctx context.Context, req CreateSubscriptionRequest) (*domain.EventSubscription, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Validate user exists
	_, err := m.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		return nil, domain.NewNotFoundError("USER_NOT_FOUND", "User not found")
	}

	// Validate project access if project ID is specified
	if req.ProjectID != nil {
		if err := m.validateProjectAccess(ctx, *req.ProjectID, req.UserID); err != nil {
			return nil, err
		}
	}

	// Create subscription
	subscription := domain.NewEventSubscription(req.UserID, req.ProjectID, req.EventTypes)
	if req.Filters != nil {
		subscription.Filters = req.Filters
	}

	// Add to broadcaster
	if err := m.broadcaster.Subscribe(ctx, subscription); err != nil {
		return nil, err
	}

	m.logger.Info("Created new subscription",
		"subscription_id", subscription.ID,
		"user_id", subscription.UserID,
		"project_id", subscription.ProjectID,
		"event_types", subscription.EventTypes)

	return subscription, nil
}

// UpdateSubscription updates an existing subscription
func (m *subscriptionManager) UpdateSubscription(ctx context.Context, subscriptionID string, req UpdateSubscriptionRequest) (*domain.EventSubscription, error) {
	if subscriptionID == "" {
		return nil, domain.NewValidationError("INVALID_SUBSCRIPTION_ID", "Subscription ID cannot be empty", nil)
	}

	// Get existing subscription
	subscription, err := m.broadcaster.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}

	// Update fields
	if req.EventTypes != nil {
		// Validate event types
		for _, eventType := range *req.EventTypes {
			if !eventType.IsValid() {
				return nil, domain.NewValidationError("INVALID_EVENT_TYPE", fmt.Sprintf("Invalid event type: %s", eventType), nil)
			}
		}
		subscription.EventTypes = *req.EventTypes
	}

	if req.Filters != nil {
		subscription.Filters = *req.Filters
	}

	if req.Active != nil {
		subscription.Active = *req.Active
	}

	// Validate updated subscription
	if err := subscription.Validate(); err != nil {
		return nil, err
	}

	m.logger.Info("Updated subscription",
		"subscription_id", subscriptionID,
		"user_id", subscription.UserID)

	return subscription, nil
}

// DeleteSubscription removes a subscription
func (m *subscriptionManager) DeleteSubscription(ctx context.Context, subscriptionID string, userID string) error {
	if err := m.ValidateSubscriptionAccess(ctx, subscriptionID, userID); err != nil {
		return err
	}

	if err := m.broadcaster.Unsubscribe(ctx, subscriptionID); err != nil {
		return err
	}

	m.logger.Info("Deleted subscription",
		"subscription_id", subscriptionID,
		"user_id", userID)

	return nil
}

// GetSubscription retrieves a subscription by ID
func (m *subscriptionManager) GetSubscription(ctx context.Context, subscriptionID string, userID string) (*domain.EventSubscription, error) {
	if err := m.ValidateSubscriptionAccess(ctx, subscriptionID, userID); err != nil {
		return nil, err
	}

	return m.broadcaster.GetSubscription(ctx, subscriptionID)
}

// ListUserSubscriptions lists all subscriptions for a user
func (m *subscriptionManager) ListUserSubscriptions(ctx context.Context, userID string) ([]*domain.EventSubscription, error) {
	if userID == "" {
		return nil, domain.NewValidationError("INVALID_USER_ID", "User ID cannot be empty", nil)
	}

	// Validate user exists
	_, err := m.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, domain.NewNotFoundError("USER_NOT_FOUND", "User not found")
	}

	return m.broadcaster.GetUserSubscriptions(ctx, userID)
}

// ValidateSubscriptionAccess checks if a user can access a subscription
func (m *subscriptionManager) ValidateSubscriptionAccess(ctx context.Context, subscriptionID string, userID string) error {
	if subscriptionID == "" {
		return domain.NewValidationError("INVALID_SUBSCRIPTION_ID", "Subscription ID cannot be empty", nil)
	}

	if userID == "" {
		return domain.NewValidationError("INVALID_USER_ID", "User ID cannot be empty", nil)
	}

	// Get subscription
	subscription, err := m.broadcaster.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return err
	}

	// Check ownership
	if subscription.UserID != userID {
		return domain.NewAuthorizationError("ACCESS_DENIED", "You don't have access to this subscription")
	}

	return nil
}

// StartCleanupRoutine starts background cleanup of expired subscriptions
func (m *subscriptionManager) StartCleanupRoutine(ctx context.Context, interval time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cleanupRunning {
		m.logger.Warn("Cleanup routine is already running")
		return
	}

	if interval <= 0 {
		interval = m.cleanupInterval
	}

	m.cleanupRunning = true

	go func() {
		defer func() {
			m.mu.Lock()
			m.cleanupRunning = false
			m.mu.Unlock()
		}()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		m.logger.Info("Started subscription cleanup routine", "interval", interval)

		for {
			select {
			case <-ctx.Done():
				m.logger.Info("Stopping subscription cleanup routine due to context cancellation")
				return

			case <-m.cleanupStop:
				m.logger.Info("Stopping subscription cleanup routine")
				return

			case <-ticker.C:
				if err := m.broadcaster.Cleanup(ctx); err != nil {
					m.logger.Error("Failed to cleanup subscriptions", "error", err)
				}
			}
		}
	}()
}

// StopCleanupRoutine stops the background cleanup routine
func (m *subscriptionManager) StopCleanupRoutine() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.cleanupRunning {
		return
	}

	select {
	case m.cleanupStop <- struct{}{}:
		m.logger.Info("Sent stop signal to cleanup routine")
	default:
		// Channel might be full or cleanup already stopped
	}
}

// validateProjectAccess checks if a user has access to a project
func (m *subscriptionManager) validateProjectAccess(ctx context.Context, projectID string, userID string) error {
	project, err := m.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return domain.NewNotFoundError("PROJECT_NOT_FOUND", "Project not found")
	}

	if !project.HasAccess(userID) {
		return domain.NewAuthorizationError("ACCESS_DENIED", "You don't have access to this project")
	}

	return nil
}

// SubscriptionManagerHealthChecker provides health checking for subscription manager
type SubscriptionManagerHealthChecker struct {
	manager SubscriptionManager
}

// NewSubscriptionManagerHealthChecker creates a health checker
func NewSubscriptionManagerHealthChecker(manager SubscriptionManager) *SubscriptionManagerHealthChecker {
	return &SubscriptionManagerHealthChecker{manager: manager}
}

// Name returns the checker name
func (h *SubscriptionManagerHealthChecker) Name() string {
	return "subscription_manager"
}

// Check performs the health check
func (h *SubscriptionManagerHealthChecker) Check(ctx context.Context) HealthCheck {
	if h.manager == nil {
		return HealthCheck{
			Name:   "subscription_manager",
			Status: HealthStatusUnhealthy,
			Error:  "Subscription manager is not available",
		}
	}

	return HealthCheck{
		Name:    "subscription_manager",
		Status:  HealthStatusHealthy,
		Message: "Subscription manager is operational",
		Details: map[string]interface{}{
			"service": "ready",
		},
	}
}

// SubscriptionFilter helps create common subscription filters
type SubscriptionFilter struct {
	filters map[string]string
}

// NewSubscriptionFilter creates a new subscription filter builder
func NewSubscriptionFilter() *SubscriptionFilter {
	return &SubscriptionFilter{
		filters: make(map[string]string),
	}
}

// ByTaskID filters events for a specific task
func (f *SubscriptionFilter) ByTaskID(taskID string) *SubscriptionFilter {
	f.filters["task_id"] = taskID
	return f
}

// ByUserID filters events for a specific user
func (f *SubscriptionFilter) ByUserID(userID string) *SubscriptionFilter {
	f.filters["user_id"] = userID
	return f
}

// ByAssignee filters events for tasks assigned to a specific user
func (f *SubscriptionFilter) ByAssignee(assigneeID string) *SubscriptionFilter {
	f.filters["assignee_id"] = assigneeID
	return f
}

// ByStatus filters events for tasks with a specific status
func (f *SubscriptionFilter) ByStatus(status domain.TaskStatus) *SubscriptionFilter {
	f.filters["status"] = string(status)
	return f
}

// Build returns the constructed filters
func (f *SubscriptionFilter) Build() map[string]string {
	result := make(map[string]string)
	for k, v := range f.filters {
		result[k] = v
	}
	return result
}