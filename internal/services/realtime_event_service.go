package services

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ericfisherdev/simple-easy-tasks/internal/domain"
)

// RealtimeEventService manages real-time event broadcasting and subscription
type RealtimeEventService struct {
	eventBroadcaster EventBroadcaster
	subscriptions    map[string]*EventSubscription
	subscriptionsMux sync.RWMutex
	eventQueue       chan *domain.TaskEvent
	eventHandlers    map[domain.TaskEventType][]RealtimeEventHandler
	handlersMux      sync.RWMutex
	wsHandler        WebSocketBroadcaster // Interface for WebSocket broadcasting
}

// EventSubscription represents an active event subscription
type EventSubscription struct {
	ID         string                 `json:"id"`
	UserID     string                 `json:"user_id"`
	ProjectID  string                 `json:"project_id,omitempty"`
	EventTypes []domain.TaskEventType `json:"event_types"`
	Filters    map[string]interface{} `json:"filters,omitempty"`
	Channel    chan *domain.TaskEvent `json:"-"`
	CreatedAt  time.Time              `json:"created_at"`
	LastEvent  *time.Time             `json:"last_event,omitempty"`
	Active     bool                   `json:"active"`
}

// RealtimeEventHandler represents a function that handles specific event types
type RealtimeEventHandler func(ctx context.Context, event *domain.TaskEvent) error

// WebSocketBroadcaster interface for WebSocket integration
type WebSocketBroadcaster interface {
	SendToUser(userID string, eventType domain.TaskEventType, data interface{})
	SendToProject(projectID string, eventType domain.TaskEventType, data interface{})
	BroadcastToAll(eventType domain.TaskEventType, data interface{})
	GetActiveConnections() int
}

// RealtimeEventConfig configures the real-time event service
type RealtimeEventConfig struct {
	QueueSize            int
	MaxSubscriptions     int
	EventRetention       time.Duration
	BroadcastTimeout     time.Duration
	EnableEventHistory   bool
	EnableMetrics        bool
	WebSocketIntegration bool
}

// EventMetrics tracks event system performance
type EventMetrics struct {
	EventsProcessed       int64                          `json:"events_processed"`
	EventsByType          map[domain.TaskEventType]int64 `json:"events_by_type"`
	ActiveSubscriptions   int                            `json:"active_subscriptions"`
	SubscriptionsByType   map[domain.TaskEventType]int   `json:"subscriptions_by_type"`
	AverageProcessingTime time.Duration                  `json:"average_processing_time"`
	ErrorCount            int64                          `json:"error_count"`
	LastEventTime         time.Time                      `json:"last_event_time"`
	WebSocketConnections  int                            `json:"websocket_connections"`
}

// NewRealtimeEventService creates a new real-time event service
func NewRealtimeEventService(eventBroadcaster EventBroadcaster, config RealtimeEventConfig) *RealtimeEventService {
	service := &RealtimeEventService{
		eventBroadcaster: eventBroadcaster,
		subscriptions:    make(map[string]*EventSubscription),
		eventQueue:       make(chan *domain.TaskEvent, config.QueueSize),
		eventHandlers:    make(map[domain.TaskEventType][]RealtimeEventHandler),
	}

	// Start event processing goroutine
	go service.processEvents()

	// Register default event handlers
	service.registerDefaultHandlers()

	return service
}

// SetWebSocketBroadcaster sets the WebSocket broadcaster for real-time updates
func (s *RealtimeEventService) SetWebSocketBroadcaster(ws WebSocketBroadcaster) {
	s.wsHandler = ws
}

// PublishTaskEvent publishes a task event to the real-time system
func (s *RealtimeEventService) PublishTaskEvent(_ context.Context, event *domain.TaskEvent) error {
	if event == nil {
		return domain.NewValidationError("INVALID_EVENT", "Event cannot be nil", nil)
	}

	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Validate event
	if err := s.validateEvent(event); err != nil {
		return err
	}

	// Add to event queue
	select {
	case s.eventQueue <- event:
		log.Printf("Event queued: %s for task %s", event.Type, event.TaskID)
		return nil
	case <-time.After(5 * time.Second):
		return domain.NewInternalError("EVENT_QUEUE_FULL", "Event queue is full", nil)
	}
}

// Subscribe creates a new event subscription
func (s *RealtimeEventService) Subscribe(_ context.Context, req *SubscriptionRequest) (*EventSubscription, error) {
	if err := s.validateSubscriptionRequest(req); err != nil {
		return nil, err
	}

	subscription := &EventSubscription{
		ID:         generateSubscriptionID(),
		UserID:     req.UserID,
		ProjectID:  req.ProjectID,
		EventTypes: req.EventTypes,
		Filters:    req.Filters,
		Channel:    make(chan *domain.TaskEvent, 100),
		CreatedAt:  time.Now(),
		Active:     true,
	}

	s.subscriptionsMux.Lock()
	s.subscriptions[subscription.ID] = subscription
	s.subscriptionsMux.Unlock()

	log.Printf("Created event subscription %s for user %s", subscription.ID, subscription.UserID)
	return subscription, nil
}

// Unsubscribe removes an event subscription
func (s *RealtimeEventService) Unsubscribe(_ context.Context, subscriptionID, userID string) error {
	s.subscriptionsMux.Lock()
	defer s.subscriptionsMux.Unlock()

	subscription, exists := s.subscriptions[subscriptionID]
	if !exists {
		return domain.NewNotFoundError("SUBSCRIPTION_NOT_FOUND", "Subscription not found")
	}

	if subscription.UserID != userID {
		return domain.NewAuthorizationError("UNAUTHORIZED", "Cannot unsubscribe from other user's subscription")
	}

	// Close channel and mark as inactive
	close(subscription.Channel)
	subscription.Active = false
	delete(s.subscriptions, subscriptionID)

	log.Printf("Removed event subscription %s for user %s", subscriptionID, userID)
	return nil
}

// GetSubscriptions returns active subscriptions for a user
func (s *RealtimeEventService) GetSubscriptions(_ context.Context, userID string) ([]*EventSubscription, error) {
	s.subscriptionsMux.RLock()
	defer s.subscriptionsMux.RUnlock()

	var userSubscriptions []*EventSubscription
	for _, subscription := range s.subscriptions {
		if subscription.UserID == userID && subscription.Active {
			userSubscriptions = append(userSubscriptions, subscription)
		}
	}

	return userSubscriptions, nil
}

// RegisterEventHandler registers a handler for a specific event type
func (s *RealtimeEventService) RegisterEventHandler(eventType domain.TaskEventType, handler RealtimeEventHandler) {
	s.handlersMux.Lock()
	defer s.handlersMux.Unlock()

	if s.eventHandlers[eventType] == nil {
		s.eventHandlers[eventType] = make([]RealtimeEventHandler, 0)
	}

	s.eventHandlers[eventType] = append(s.eventHandlers[eventType], handler)
	log.Printf("Registered event handler for type %s", eventType)
}

// processEvents processes events from the queue
func (s *RealtimeEventService) processEvents() {
	for event := range s.eventQueue {
		start := time.Now()

		// Process the event
		if err := s.handleEvent(context.Background(), event); err != nil {
			log.Printf("Error processing event %s: %v", event.EventID, err)
		}

		// Broadcast to subscriptions
		s.broadcastToSubscriptions(event)

		// Broadcast via WebSocket if configured
		if s.wsHandler != nil {
			s.broadcastViaWebSocket(event)
		}

		// Record processing time
		processingTime := time.Since(start)
		log.Printf("Processed event %s in %v", event.EventID, processingTime)
	}
}

// handleEvent processes an event through registered handlers
func (s *RealtimeEventService) handleEvent(ctx context.Context, event *domain.TaskEvent) error {
	s.handlersMux.RLock()
	handlers, exists := s.eventHandlers[event.Type]
	s.handlersMux.RUnlock()

	if !exists || len(handlers) == 0 {
		log.Printf("No handlers registered for event type %s", event.Type)
		return nil
	}

	// Execute all handlers for this event type
	for _, handler := range handlers {
		if err := handler(ctx, event); err != nil {
			log.Printf("Handler error for event %s: %v", event.EventID, err)
			// Continue processing other handlers
		}
	}

	return nil
}

// broadcastToSubscriptions sends events to matching subscriptions
func (s *RealtimeEventService) broadcastToSubscriptions(event *domain.TaskEvent) {
	s.subscriptionsMux.RLock()
	defer s.subscriptionsMux.RUnlock()

	sentCount := 0
	for _, subscription := range s.subscriptions {
		if !subscription.Active {
			continue
		}

		// Check if subscription matches this event
		if s.subscriptionMatchesEvent(subscription, event) {
			select {
			case subscription.Channel <- event:
				now := time.Now()
				subscription.LastEvent = &now
				sentCount++
			default:
				// Channel is full, mark subscription as inactive
				log.Printf("Subscription %s channel full, marking as inactive", subscription.ID)
				subscription.Active = false
				close(subscription.Channel)
			}
		}
	}

	if sentCount > 0 {
		log.Printf("Broadcasted event %s to %d subscriptions", event.EventID, sentCount)
	}
}

// broadcastViaWebSocket sends events via WebSocket connections
func (s *RealtimeEventService) broadcastViaWebSocket(event *domain.TaskEvent) {
	if s.wsHandler == nil {
		return
	}

	// Send to specific project
	if event.ProjectID != "" {
		s.wsHandler.SendToProject(event.ProjectID, event.Type, event)
	}

	// Send to specific user if different from creator
	if event.UserID != "" {
		s.wsHandler.SendToUser(event.UserID, event.Type, event)
	}

	log.Printf("Broadcasted event %s via WebSocket", event.EventID)
}

// subscriptionMatchesEvent checks if a subscription matches an event
func (s *RealtimeEventService) subscriptionMatchesEvent(subscription *EventSubscription, event *domain.TaskEvent) bool {
	// Check event type match
	eventTypeMatches := false
	for _, eventType := range subscription.EventTypes {
		if eventType == event.Type {
			eventTypeMatches = true
			break
		}
	}

	if !eventTypeMatches {
		return false
	}

	// Check project filter
	if subscription.ProjectID != "" && subscription.ProjectID != event.ProjectID {
		return false
	}

	// Check additional filters
	if len(subscription.Filters) > 0 {
		return s.checkEventFilters(subscription.Filters, event)
	}

	return true
}

// checkEventFilters applies additional filters to events
func (s *RealtimeEventService) checkEventFilters(filters map[string]interface{}, _ *domain.TaskEvent) bool {
	// Check assignee filter
	if assigneeID, exists := filters["assignee_id"]; exists {
		if assigneeStr, ok := assigneeID.(string); ok {
			// This would require looking up the task to check assignee
			// For now, we'll assume it matches
			_ = assigneeStr
		}
	}

	// Check priority filter
	if priority, exists := filters["priority"]; exists {
		if priorityStr, ok := priority.(string); ok {
			// This would require looking up the task to check priority
			// For now, we'll assume it matches
			_ = priorityStr
		}
	}

	// Add more filter types as needed
	return true
}

// validateEvent validates an event before processing
func (s *RealtimeEventService) validateEvent(event *domain.TaskEvent) error {
	if event.EventID == "" {
		return domain.NewValidationError("MISSING_EVENT_ID", "Event ID is required", nil)
	}

	if event.TaskID == "" {
		return domain.NewValidationError("MISSING_TASK_ID", "Task ID is required", nil)
	}

	if !event.Type.IsValid() {
		return domain.NewValidationError("INVALID_EVENT_TYPE", "Invalid event type", map[string]interface{}{
			"event_type": string(event.Type),
		})
	}

	return nil
}

// validateSubscriptionRequest validates a subscription request
func (s *RealtimeEventService) validateSubscriptionRequest(req *SubscriptionRequest) error {
	if req.UserID == "" {
		return domain.NewValidationError("MISSING_USER_ID", "User ID is required", nil)
	}

	if len(req.EventTypes) == 0 {
		return domain.NewValidationError("MISSING_EVENT_TYPES", "At least one event type is required", nil)
	}

	for _, eventType := range req.EventTypes {
		if !eventType.IsValid() {
			return domain.NewValidationError("INVALID_EVENT_TYPE", "Invalid event type", map[string]interface{}{
				"event_type": string(eventType),
			})
		}
	}

	return nil
}

// registerDefaultHandlers registers default event handlers
func (s *RealtimeEventService) registerDefaultHandlers() {
	// Task creation handler
	s.RegisterEventHandler(domain.TaskCreated, func(_ context.Context, event *domain.TaskEvent) error {
		log.Printf("Task created: %s in project %s", event.TaskID, event.ProjectID)
		return nil
	})

	// Task update handler
	s.RegisterEventHandler(domain.TaskUpdated, func(_ context.Context, event *domain.TaskEvent) error {
		log.Printf("Task updated: %s", event.TaskID)
		return nil
	})

	// Task moved handler
	s.RegisterEventHandler(domain.TaskMoved, func(_ context.Context, event *domain.TaskEvent) error {
		log.Printf("Task moved: %s", event.TaskID)
		return nil
	})

	// Task assigned handler
	s.RegisterEventHandler(domain.TaskAssigned, func(_ context.Context, event *domain.TaskEvent) error {
		log.Printf("Task assigned: %s", event.TaskID)
		return nil
	})

	// Task deleted handler
	s.RegisterEventHandler(domain.TaskDeleted, func(_ context.Context, event *domain.TaskEvent) error {
		log.Printf("Task deleted: %s", event.TaskID)
		return nil
	})

	// Task commented handler
	s.RegisterEventHandler(domain.TaskCommented, func(_ context.Context, event *domain.TaskEvent) error {
		log.Printf("Task commented: %s", event.TaskID)
		return nil
	})
}

// GetMetrics returns event system metrics
func (s *RealtimeEventService) GetMetrics() *EventMetrics {
	s.subscriptionsMux.RLock()
	defer s.subscriptionsMux.RUnlock()

	metrics := &EventMetrics{
		ActiveSubscriptions: len(s.subscriptions),
		SubscriptionsByType: make(map[domain.TaskEventType]int),
		EventsByType:        make(map[domain.TaskEventType]int64),
	}

	// Count subscriptions by type
	for _, subscription := range s.subscriptions {
		if subscription.Active {
			for _, eventType := range subscription.EventTypes {
				metrics.SubscriptionsByType[eventType]++
			}
		}
	}

	// Get WebSocket connection count if available
	if s.wsHandler != nil {
		metrics.WebSocketConnections = s.wsHandler.GetActiveConnections()
	}

	return metrics
}

// Cleanup removes inactive subscriptions and performs maintenance
func (s *RealtimeEventService) Cleanup() {
	s.subscriptionsMux.Lock()
	defer s.subscriptionsMux.Unlock()

	cleanedUp := 0
	for id, subscription := range s.subscriptions {
		// Remove inactive subscriptions older than 1 hour
		if !subscription.Active && time.Since(subscription.CreatedAt) > time.Hour {
			delete(s.subscriptions, id)
			cleanedUp++
		}

		// Remove subscriptions without recent activity (older than 24 hours)
		if subscription.LastEvent == nil && time.Since(subscription.CreatedAt) > 24*time.Hour {
			subscription.Active = false
			close(subscription.Channel)
			delete(s.subscriptions, id)
			cleanedUp++
		}
	}

	if cleanedUp > 0 {
		log.Printf("Cleaned up %d inactive subscriptions", cleanedUp)
	}
}

// StartCleanupWorker starts a goroutine that periodically cleans up subscriptions
func (s *RealtimeEventService) StartCleanupWorker() {
	go func() {
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			s.Cleanup()
		}
	}()
}

// SubscriptionRequest represents a request to create an event subscription
type SubscriptionRequest struct {
	UserID     string                 `json:"user_id"`
	ProjectID  string                 `json:"project_id,omitempty"`
	EventTypes []domain.TaskEventType `json:"event_types"`
	Filters    map[string]interface{} `json:"filters,omitempty"`
}

// Helper function to generate subscription IDs
func generateSubscriptionID() string {
	return fmt.Sprintf("sub_%d_%s", time.Now().UnixNano(), randString(8))
}

// Helper function to generate random strings
func randString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
