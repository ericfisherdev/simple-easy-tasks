package services

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"simple-easy-tasks/internal/domain"
)

// EventBroadcaster defines the interface for broadcasting task events to subscribed clients
type EventBroadcaster interface {
	// BroadcastEvent broadcasts a task event to all relevant subscribers
	BroadcastEvent(ctx context.Context, event *domain.TaskEvent) error

	// Subscribe adds a new event subscription
	Subscribe(ctx context.Context, subscription *domain.EventSubscription) error

	// Unsubscribe removes an event subscription
	Unsubscribe(ctx context.Context, subscriptionID string) error

	// GetSubscription retrieves a subscription by ID
	GetSubscription(ctx context.Context, subscriptionID string) (*domain.EventSubscription, error)

	// GetUserSubscriptions retrieves all subscriptions for a user
	GetUserSubscriptions(ctx context.Context, userID string) ([]*domain.EventSubscription, error)

	// GetActiveSubscriptionCount returns the number of active subscriptions
	GetActiveSubscriptionCount() int

	// Cleanup removes inactive or expired subscriptions
	Cleanup(ctx context.Context) error
}

// EventHandler is a callback function for processing events
type EventHandler func(event *domain.TaskEvent, subscription *domain.EventSubscription) error

// eventBroadcaster implements the EventBroadcaster interface
type eventBroadcaster struct {
	subscriptions     map[string]*domain.EventSubscription
	userSubscriptions map[string][]string // userID -> []subscriptionID
	eventHandlers     []EventHandler
	mu                sync.RWMutex
	logger            *slog.Logger
	pbService         *PocketBaseService

	// Configuration
	maxSubscriptionsPerUser int
	subscriptionTimeout     time.Duration
	eventQueueSize          int
}

// EventBroadcasterConfig holds configuration for the event broadcaster
type EventBroadcasterConfig struct {
	MaxSubscriptionsPerUser int           // Maximum subscriptions per user (default: 10)
	SubscriptionTimeout     time.Duration // Timeout for inactive subscriptions (default: 1 hour)
	EventQueueSize          int           // Size of event queue per subscription (default: 100)
	Logger                  *slog.Logger  // Logger instance
}

// NewEventBroadcaster creates a new event broadcaster service
func NewEventBroadcaster(pbService *PocketBaseService, config EventBroadcasterConfig) EventBroadcaster {
	if config.MaxSubscriptionsPerUser <= 0 {
		config.MaxSubscriptionsPerUser = 10
	}
	if config.SubscriptionTimeout <= 0 {
		config.SubscriptionTimeout = time.Hour
	}
	if config.EventQueueSize <= 0 {
		config.EventQueueSize = 100
	}
	if config.Logger == nil {
		config.Logger = slog.Default()
	}

	return &eventBroadcaster{
		subscriptions:           make(map[string]*domain.EventSubscription),
		userSubscriptions:       make(map[string][]string),
		eventHandlers:           make([]EventHandler, 0),
		logger:                  config.Logger,
		pbService:               pbService,
		maxSubscriptionsPerUser: config.MaxSubscriptionsPerUser,
		subscriptionTimeout:     config.SubscriptionTimeout,
		eventQueueSize:          config.EventQueueSize,
	}
}

// BroadcastEvent broadcasts a task event to all relevant subscribers
func (b *eventBroadcaster) BroadcastEvent(ctx context.Context, event *domain.TaskEvent) error {
	if event == nil {
		return domain.NewValidationError("NIL_EVENT", "Event cannot be nil", nil)
	}

	if err := event.Validate(); err != nil {
		return err
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	matchingSubscriptions := b.findMatchingSubscriptions(event)
	if len(matchingSubscriptions) == 0 {
		b.logger.Debug("No matching subscriptions found for event",
			"event_type", event.Type,
			"task_id", event.TaskID,
			"project_id", event.ProjectID)
		return nil
	}

	b.logger.Info("Broadcasting event to subscribers",
		"event_type", event.Type,
		"task_id", event.TaskID,
		"project_id", event.ProjectID,
		"subscribers", len(matchingSubscriptions))

	// Broadcast to PocketBase realtime subscribers
	if err := b.broadcastToPocketBase(ctx, event); err != nil {
		b.logger.Error("Failed to broadcast to PocketBase",
			"error", err,
			"event_id", event.EventID)
	}

	// Process event handlers for all matching subscriptions
	for _, subscription := range matchingSubscriptions {
		subscription.UpdateActivity()

		// Update the subscription in the map to persist the activity change
		b.subscriptions[subscription.ID] = subscription

		for _, handler := range b.eventHandlers {
			if err := handler(event, subscription); err != nil {
				b.logger.Error("Event handler failed",
					"error", err,
					"subscription_id", subscription.ID,
					"event_id", event.EventID)
			}
		}
	}

	return nil
}

// Subscribe adds a new event subscription
func (b *eventBroadcaster) Subscribe(_ context.Context, subscription *domain.EventSubscription) error {
	if subscription == nil {
		return domain.NewValidationError("NIL_SUBSCRIPTION", "Subscription cannot be nil", nil)
	}

	if err := subscription.Validate(); err != nil {
		return err
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// Check user subscription limit
	userSubs := b.userSubscriptions[subscription.UserID]
	if len(userSubs) >= b.maxSubscriptionsPerUser {
		return domain.NewValidationError(
			"SUBSCRIPTION_LIMIT_EXCEEDED",
			fmt.Sprintf("User has reached maximum subscription limit of %d", b.maxSubscriptionsPerUser),
			nil,
		)
	}

	// Add subscription
	b.subscriptions[subscription.ID] = subscription
	b.userSubscriptions[subscription.UserID] = append(userSubs, subscription.ID)

	b.logger.Info("Added event subscription",
		"subscription_id", subscription.ID,
		"user_id", subscription.UserID,
		"project_id", subscription.ProjectID,
		"event_types", subscription.EventTypes)

	return nil
}

// Unsubscribe removes an event subscription
func (b *eventBroadcaster) Unsubscribe(_ context.Context, subscriptionID string) error {
	if subscriptionID == "" {
		return domain.NewValidationError("INVALID_SUBSCRIPTION_ID", "Subscription ID cannot be empty", nil)
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	subscription, exists := b.subscriptions[subscriptionID]
	if !exists {
		return domain.NewNotFoundError("SUBSCRIPTION_NOT_FOUND", "Subscription not found")
	}

	// Remove from user subscriptions
	userSubs := b.userSubscriptions[subscription.UserID]
	for i, subID := range userSubs {
		if subID == subscriptionID {
			b.userSubscriptions[subscription.UserID] = append(userSubs[:i], userSubs[i+1:]...)
			break
		}
	}

	// Remove main subscription
	delete(b.subscriptions, subscriptionID)

	b.logger.Info("Removed event subscription",
		"subscription_id", subscriptionID,
		"user_id", subscription.UserID)

	return nil
}

// GetSubscription retrieves a subscription by ID
func (b *eventBroadcaster) GetSubscription(
	_ context.Context, subscriptionID string,
) (*domain.EventSubscription, error) {
	if subscriptionID == "" {
		return nil, domain.NewValidationError("INVALID_SUBSCRIPTION_ID", "Subscription ID cannot be empty", nil)
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	subscription, exists := b.subscriptions[subscriptionID]
	if !exists {
		return nil, domain.NewNotFoundError("SUBSCRIPTION_NOT_FOUND", "Subscription not found")
	}

	return subscription, nil
}

// GetUserSubscriptions retrieves all subscriptions for a user
func (b *eventBroadcaster) GetUserSubscriptions(_ context.Context, userID string) ([]*domain.EventSubscription, error) {
	if userID == "" {
		return nil, domain.NewValidationError("INVALID_USER_ID", "User ID cannot be empty", nil)
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	userSubIDs, exists := b.userSubscriptions[userID]
	if !exists {
		return []*domain.EventSubscription{}, nil
	}

	subscriptions := make([]*domain.EventSubscription, 0, len(userSubIDs))
	for _, subID := range userSubIDs {
		if subscription, exists := b.subscriptions[subID]; exists {
			subscriptions = append(subscriptions, subscription)
		}
	}

	return subscriptions, nil
}

// GetActiveSubscriptionCount returns the number of active subscriptions
func (b *eventBroadcaster) GetActiveSubscriptionCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	activeCount := 0
	for _, subscription := range b.subscriptions {
		if subscription.Active {
			activeCount++
		}
	}

	return activeCount
}

// Cleanup removes inactive or expired subscriptions
func (b *eventBroadcaster) Cleanup(_ context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	cutoffTime := time.Now().UTC().Add(-b.subscriptionTimeout)
	removedCount := 0

	for subID, subscription := range b.subscriptions {
		if subscription.LastActivity.Before(cutoffTime) {
			// Remove from user subscriptions
			userSubs := b.userSubscriptions[subscription.UserID]
			for i, userSubID := range userSubs {
				if userSubID == subID {
					b.userSubscriptions[subscription.UserID] = append(userSubs[:i], userSubs[i+1:]...)
					break
				}
			}

			// Remove main subscription
			delete(b.subscriptions, subID)
			removedCount++
		}
	}

	if removedCount > 0 {
		b.logger.Info("Cleaned up expired subscriptions",
			"removed_count", removedCount,
			"total_remaining", len(b.subscriptions))
	}

	return nil
}

// AddEventHandler adds a custom event handler
func (b *eventBroadcaster) AddEventHandler(handler EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.eventHandlers = append(b.eventHandlers, handler)
}

// findMatchingSubscriptions finds all subscriptions that should receive an event
func (b *eventBroadcaster) findMatchingSubscriptions(event *domain.TaskEvent) []*domain.EventSubscription {
	var matchingSubscriptions []*domain.EventSubscription

	for _, subscription := range b.subscriptions {
		if subscription.MatchesEvent(event) {
			matchingSubscriptions = append(matchingSubscriptions, subscription)
		}
	}

	return matchingSubscriptions
}

// broadcastToPocketBase sends the event to PocketBase's real-time system
func (b *eventBroadcaster) broadcastToPocketBase(_ context.Context, event *domain.TaskEvent) error {
	if b.pbService == nil || b.pbService.app == nil {
		return domain.NewInternalError("POCKETBASE_UNAVAILABLE", "PocketBase service is not available", nil)
	}

	// Convert event to JSON for logging/debugging
	_, err := event.ToJSON()
	if err != nil {
		return err
	}

	// Create subscription message for PocketBase's realtime system
	message := map[string]interface{}{
		"action": "create", // PocketBase action type
		"record": map[string]interface{}{
			"id":         event.EventID,
			"type":       event.Type,
			"task_id":    event.TaskID,
			"project_id": event.ProjectID,
			"user_id":    event.UserID,
			"data":       string(event.Data),
			"timestamp":  event.Timestamp.Unix(),
		},
	}

	// Use PocketBase's subscription broker to broadcast
	broker := b.pbService.app.SubscriptionsBroker()
	if broker == nil {
		return domain.NewInternalError("BROKER_UNAVAILABLE", "Subscription broker is not available", nil)
	}

	// Send to all clients subscribed to task events
	// The actual broadcasting will depend on PocketBase's internal implementation
	// For now, we log the event - in production this would integrate with PocketBase's SSE system
	b.logger.Debug("Would broadcast to PocketBase SSE clients",
		"event_type", event.Type,
		"event_id", event.EventID,
		"message", message)

	return nil
}

// EventBroadcasterHealthChecker provides health checking for the event broadcaster
type EventBroadcasterHealthChecker struct {
	broadcaster EventBroadcaster
}

// NewEventBroadcasterHealthChecker creates a health checker for the event broadcaster
func NewEventBroadcasterHealthChecker(broadcaster EventBroadcaster) *EventBroadcasterHealthChecker {
	return &EventBroadcasterHealthChecker{broadcaster: broadcaster}
}

// Name returns the checker name
func (h *EventBroadcasterHealthChecker) Name() string {
	return "event_broadcaster"
}

// Check performs the event broadcaster health check
func (h *EventBroadcasterHealthChecker) Check(_ context.Context) HealthCheck {
	if h.broadcaster == nil {
		return HealthCheck{
			Name:   "event_broadcaster",
			Status: HealthStatusUnhealthy,
			Error:  "Event broadcaster is not available",
		}
	}

	activeSubscriptions := h.broadcaster.GetActiveSubscriptionCount()

	return HealthCheck{
		Name:    "event_broadcaster",
		Status:  HealthStatusHealthy,
		Message: "Event broadcaster is operational",
		Details: map[string]interface{}{
			"active_subscriptions": activeSubscriptions,
		},
	}
}
