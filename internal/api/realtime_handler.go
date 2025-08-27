package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pocketbase/pocketbase/core"

	"simple-easy-tasks/internal/api/middleware"
	"simple-easy-tasks/internal/domain"
	"simple-easy-tasks/internal/services"
)

// RealtimeHandler provides HTTP handlers for real-time subscriptions
type RealtimeHandler struct {
	subscriptionManager services.SubscriptionManager
	eventBroadcaster    services.EventBroadcaster
	app                 core.App
}

// NewRealtimeHandler creates a new realtime handler
func NewRealtimeHandler(
	subscriptionManager services.SubscriptionManager,
	eventBroadcaster services.EventBroadcaster,
	app core.App,
) *RealtimeHandler {
	return &RealtimeHandler{
		subscriptionManager: subscriptionManager,
		eventBroadcaster:    eventBroadcaster,
		app:                 app,
	}
}

// RegisterRoutes registers all real-time related routes
func (h *RealtimeHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware) {
	// Subscription management endpoints
	realtime := router.Group("/realtime")
	realtime.Use(authMiddleware.RequireAuth())
	{
		realtime.POST("/subscriptions", h.CreateSubscription)
		realtime.GET("/subscriptions", h.ListSubscriptions)
		realtime.GET("/subscriptions/:id", h.GetSubscription)
		realtime.PUT("/subscriptions/:id", h.UpdateSubscription)
		realtime.DELETE("/subscriptions/:id", h.DeleteSubscription)

		// Real-time connection endpoints
		realtime.GET("/events", h.StreamEvents)
		realtime.GET("/connections", h.GetActiveConnections)
	}

	// Public health check (no auth required)
	router.GET("/realtime/health", h.HealthCheck)

	// PocketBase integration endpoint
	realtime.GET("/pocketbase/tasks", h.PocketBaseTaskEvents)
}

// CreateSubscription creates a new event subscription
func (h *RealtimeHandler) CreateSubscription(c *gin.Context) {
	// Get authenticated user
	user, exists := middleware.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "AUTHENTICATION_ERROR",
				"code":    "USER_NOT_FOUND",
				"message": "User not found in context",
			},
		})
		return
	}

	// Parse request
	var req services.CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "VALIDATION_ERROR",
				"code":    "INVALID_REQUEST",
				"message": "Invalid request format",
				"details": err.Error(),
			},
		})
		return
	}

	// Set user ID from auth context
	req.UserID = user.ID

	// Create subscription
	subscription, err := h.subscriptionManager.CreateSubscription(c.Request.Context(), req)
	if err != nil {
		ErrorResponse(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    subscription,
		"message": "Subscription created successfully",
	})
}

// ListSubscriptions lists all subscriptions for the authenticated user
func (h *RealtimeHandler) ListSubscriptions(c *gin.Context) {
	// Get authenticated user
	user, exists := middleware.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "AUTHENTICATION_ERROR",
				"code":    "USER_NOT_FOUND",
				"message": "User not found in context",
			},
		})
		return
	}

	// Get subscriptions
	subscriptions, err := h.subscriptionManager.ListUserSubscriptions(c.Request.Context(), user.ID)
	if err != nil {
		ErrorResponse(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    subscriptions,
		"message": "Subscriptions retrieved successfully",
	})
}

// GetSubscription retrieves a specific subscription
func (h *RealtimeHandler) GetSubscription(c *gin.Context) {
	// Get authenticated user
	user, exists := middleware.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "AUTHENTICATION_ERROR",
				"code":    "USER_NOT_FOUND",
				"message": "User not found in context",
			},
		})
		return
	}

	subscriptionID := c.Param("id")
	if subscriptionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "VALIDATION_ERROR",
				"code":    "MISSING_SUBSCRIPTION_ID",
				"message": "Subscription ID is required",
			},
		})
		return
	}

	// Get subscription
	subscription, err := h.subscriptionManager.GetSubscription(c.Request.Context(), subscriptionID, user.ID)
	if err != nil {
		ErrorResponse(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    subscription,
		"message": "Subscription retrieved successfully",
	})
}

// UpdateSubscription updates an existing subscription
func (h *RealtimeHandler) UpdateSubscription(c *gin.Context) {
	// Get authenticated user
	user, exists := middleware.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "AUTHENTICATION_ERROR",
				"code":    "USER_NOT_FOUND",
				"message": "User not found in context",
			},
		})
		return
	}

	subscriptionID := c.Param("id")
	if subscriptionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "VALIDATION_ERROR",
				"code":    "MISSING_SUBSCRIPTION_ID",
				"message": "Subscription ID is required",
			},
		})
		return
	}

	// Validate access
	if err := h.subscriptionManager.ValidateSubscriptionAccess(c.Request.Context(), subscriptionID, user.ID); err != nil {
		ErrorResponse(c, err)
		return
	}

	// Parse request
	var req services.UpdateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "VALIDATION_ERROR",
				"code":    "INVALID_REQUEST",
				"message": "Invalid request format",
				"details": err.Error(),
			},
		})
		return
	}

	// Update subscription
	subscription, err := h.subscriptionManager.UpdateSubscription(c.Request.Context(), subscriptionID, req)
	if err != nil {
		ErrorResponse(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    subscription,
		"message": "Subscription updated successfully",
	})
}

// DeleteSubscription deletes a subscription
func (h *RealtimeHandler) DeleteSubscription(c *gin.Context) {
	// Get authenticated user
	user, exists := middleware.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "AUTHENTICATION_ERROR",
				"code":    "USER_NOT_FOUND",
				"message": "User not found in context",
			},
		})
		return
	}

	subscriptionID := c.Param("id")
	if subscriptionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "VALIDATION_ERROR",
				"code":    "MISSING_SUBSCRIPTION_ID",
				"message": "Subscription ID is required",
			},
		})
		return
	}

	// Delete subscription
	if err := h.subscriptionManager.DeleteSubscription(c.Request.Context(), subscriptionID, user.ID); err != nil {
		ErrorResponse(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Subscription deleted successfully",
	})
}

// StreamEvents provides Server-Sent Events stream for real-time updates
func (h *RealtimeHandler) StreamEvents(c *gin.Context) {
	user, exists := middleware.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "AUTHENTICATION_ERROR",
				"code":    "USER_NOT_FOUND",
				"message": "User not found in context",
			},
		})
		return
	}

	subscription, err := h.setupStreamSubscription(c, user)
	if err != nil {
		ErrorResponse(c, err)
		return
	}
	defer h.cleanupSubscription(subscription.ID, user.ID)

	h.setupSSEHeaders(c)
	eventChan := make(chan *domain.TaskEvent, 100)
	defer close(eventChan)

	h.handleSSEStream(c, subscription, eventChan)
}

// setupStreamSubscription creates a subscription for the SSE connection
func (h *RealtimeHandler) setupStreamSubscription(
	c *gin.Context, user *domain.User,
) (*domain.EventSubscription, error) {
	eventTypes := h.parseEventTypes(c.Query("event_types"))

	var projectIDPtr *string
	if projectID := c.Query("project_id"); projectID != "" {
		projectIDPtr = &projectID
	}

	subscriptionReq := services.CreateSubscriptionRequest{
		UserID:     user.ID,
		ProjectID:  projectIDPtr,
		EventTypes: eventTypes,
	}

	return h.subscriptionManager.CreateSubscription(c.Request.Context(), subscriptionReq)
}

// parseEventTypes parses event types from query parameter
func (h *RealtimeHandler) parseEventTypes(eventTypesParam string) []domain.TaskEventType {
	var eventTypes []domain.TaskEventType
	if eventTypesParam != "" {
		typeStrings := strings.Split(eventTypesParam, ",")
		for _, typeStr := range typeStrings {
			eventType := domain.TaskEventType(strings.TrimSpace(typeStr))
			if eventType.IsValid() {
				eventTypes = append(eventTypes, eventType)
			}
		}
	}

	if len(eventTypes) == 0 {
		eventTypes = []domain.TaskEventType{
			domain.TaskCreated,
			domain.TaskUpdated,
			domain.TaskMoved,
			domain.TaskAssigned,
			domain.TaskDeleted,
			domain.TaskCommented,
		}
	}
	return eventTypes
}

// cleanupSubscription removes a subscription safely
func (h *RealtimeHandler) cleanupSubscription(subscriptionID, userID string) {
	if err := h.subscriptionManager.DeleteSubscription(context.Background(), subscriptionID, userID); err != nil {
		// Log error but don't fail
	}
}

// setupSSEHeaders configures headers for Server-Sent Events
func (h *RealtimeHandler) setupSSEHeaders(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("X-Accel-Buffering", "no") // Disable nginx buffering for SSE
}

// handleSSEStream manages the SSE event streaming loop
func (h *RealtimeHandler) handleSSEStream(
	c *gin.Context, subscription *domain.EventSubscription, eventChan chan *domain.TaskEvent,
) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	w := c.Writer
	f, ok := w.(http.Flusher)
	if !ok {
		return
	}

	// Send initial connection message
	connMsg := `{"type":"connected","subscription_id":"` + subscription.ID + `"}`
	if _, err := fmt.Fprintf(w, "data: %s\n\n", connMsg); err != nil {
		return // Connection closed
	}
	f.Flush()

	for {
		select {
		case <-c.Request.Context().Done():
			return

		case <-ticker.C:
			if _, err := fmt.Fprintf(w, ": keepalive\n\n"); err != nil {
				return // Connection closed
			}
			f.Flush()

		case event := <-eventChan:
			eventJSON, err := event.ToJSON()
			if err != nil {
				continue // Skip malformed events
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", string(eventJSON)); err != nil {
				return // Connection closed
			}
			f.Flush()
		}
	}
}

// PocketBaseTaskEvents integrates with PocketBase's built-in real-time system
func (h *RealtimeHandler) PocketBaseTaskEvents(c *gin.Context) {
	// Get authenticated user
	user, exists := middleware.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "AUTHENTICATION_ERROR",
				"code":    "USER_NOT_FOUND",
				"message": "User not found in context",
			},
		})
		return
	}

	// This endpoint leverages PocketBase's built-in SSE functionality
	// For now, we'll provide a simple response indicating PocketBase integration
	// TODO: Implement full PocketBase realtime integration when needed
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "PocketBase realtime integration endpoint",
		"user_id": user.ID,
	})
}

// HealthCheck provides health status for real-time services
func (h *RealtimeHandler) HealthCheck(c *gin.Context) {
	activeSubscriptions := h.eventBroadcaster.GetActiveSubscriptionCount()

	health := map[string]interface{}{
		"status":               "healthy",
		"active_subscriptions": activeSubscriptions,
		"timestamp":            time.Now().UTC(),
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    health,
		"message": "Real-time service is healthy",
	})
}

// TaskEventStreamRequest represents the structure for creating event streams
type TaskEventStreamRequest struct {
	ProjectID  *string                `json:"project_id,omitempty"`
	EventTypes []domain.TaskEventType `json:"event_types,omitempty"`
	Filters    map[string]string      `json:"filters,omitempty"`
}

// ValidateEventStreamRequest validates the event stream request
func ValidateEventStreamRequest(req *TaskEventStreamRequest) error {
	if req.EventTypes != nil {
		for _, eventType := range req.EventTypes {
			if !eventType.IsValid() {
				return fmt.Errorf("invalid event type: %s", eventType)
			}
		}
	}

	return nil
}

// FormatSSEData formats data for Server-Sent Events
func FormatSSEData(event *domain.TaskEvent) string {
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return ""
	}

	return string(eventJSON)
}

// RealtimeMiddleware provides middleware for real-time connections
func RealtimeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Add real-time specific headers
		c.Header("X-Accel-Buffering", "no") // Disable nginx buffering for SSE
		c.Next()
	}
}

// WebSocketUpgrade handles WebSocket upgrade requests (future enhancement)
func (h *RealtimeHandler) WebSocketUpgrade(c *gin.Context) {
	// This would implement WebSocket upgrade logic
	// For now, we focus on SSE which works with PocketBase's existing infrastructure
	c.String(http.StatusNotImplemented, "WebSocket support coming soon")
}

// BroadcastToProject broadcasts an event to all subscribers of a project
func (h *RealtimeHandler) BroadcastToProject(ctx context.Context, projectID string, event *domain.TaskEvent) error {
	// This would be called from task operations to broadcast events
	return h.eventBroadcaster.BroadcastEvent(ctx, event)
}

// GetActiveConnections returns information about active real-time connections
func (h *RealtimeHandler) GetActiveConnections(c *gin.Context) {
	// Get authenticated user
	user, exists := middleware.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "AUTHENTICATION_ERROR",
				"code":    "USER_NOT_FOUND",
				"message": "User not found in context",
			},
		})
		return
	}

	// Get connection information
	activeSubscriptions := h.eventBroadcaster.GetActiveSubscriptionCount()
	userSubscriptions, err := h.subscriptionManager.ListUserSubscriptions(c.Request.Context(), user.ID)
	if err != nil {
		ErrorResponse(c, err)
		return
	}

	connectionInfo := map[string]interface{}{
		"total_active_subscriptions": activeSubscriptions,
		"user_subscriptions":         len(userSubscriptions),
		"subscriptions":              userSubscriptions,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    connectionInfo,
		"message": "Connection information retrieved successfully",
	})
}
