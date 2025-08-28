package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/pocketbase/pocketbase/core"

	"github.com/ericfisherdev/simple-easy-tasks/internal/api/middleware"
	"github.com/ericfisherdev/simple-easy-tasks/internal/domain"
	"github.com/ericfisherdev/simple-easy-tasks/internal/services"
)

// WebSocketHandler provides WebSocket functionality for real-time features
type WebSocketHandler struct {
	upgrader            websocket.Upgrader
	subscriptionManager services.SubscriptionManager
	eventBroadcaster    services.EventBroadcaster
	app                 core.App
	connectionManager   *ConnectionManager
}

// ConnectionManager manages active WebSocket connections
type ConnectionManager struct {
	connections map[string]*WebSocketConnection
	mutex       sync.RWMutex
	register    chan *WebSocketConnection
	unregister  chan *WebSocketConnection
	broadcast   chan *BroadcastMessage
}

// WebSocketConnection represents a single WebSocket connection
type WebSocketConnection struct {
	ID            string
	UserID        string
	ProjectID     string
	Connection    *websocket.Conn
	Send          chan []byte
	EventTypes    []domain.TaskEventType
	LastPing      time.Time
	Subscriptions map[string]*domain.EventSubscription
	mutex         sync.RWMutex
}

// BroadcastMessage represents a message to broadcast to connections
type BroadcastMessage struct {
	ProjectID    string
	EventType    domain.TaskEventType
	Data         interface{}
	TargetUserID string // Optional: send to specific user only
}

// WebSocketMessage represents incoming/outgoing WebSocket messages
type WebSocketMessage struct {
	Type      string                 `json:"type"`
	Action    string                 `json:"action,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// ConnectionStats provides connection statistics
type ConnectionStats struct {
	TotalConnections     int                          `json:"total_connections"`
	ConnectionsByUser    map[string]int               `json:"connections_by_user"`
	ConnectionsByProject map[string]int               `json:"connections_by_project"`
	EventTypeStats       map[domain.TaskEventType]int `json:"event_type_stats"`
	LastUpdated          time.Time                    `json:"last_updated"`
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(
	subscriptionManager services.SubscriptionManager,
	eventBroadcaster services.EventBroadcaster,
	app core.App,
) *WebSocketHandler {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(_ *http.Request) bool {
			// In production, implement proper origin checking
			return true
		},
	}

	connectionManager := &ConnectionManager{
		connections: make(map[string]*WebSocketConnection),
		register:    make(chan *WebSocketConnection, 256),
		unregister:  make(chan *WebSocketConnection, 256),
		broadcast:   make(chan *BroadcastMessage, 1024),
	}

	handler := &WebSocketHandler{
		upgrader:            upgrader,
		subscriptionManager: subscriptionManager,
		eventBroadcaster:    eventBroadcaster,
		app:                 app,
		connectionManager:   connectionManager,
	}

	// Start the connection manager
	go handler.connectionManager.run()

	return handler
}

// RegisterRoutes registers WebSocket routes
func (h *WebSocketHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware) {
	ws := router.Group("/ws")
	{
		// WebSocket upgrade endpoint
		ws.GET("/connect", authMiddleware.RequireAuth(), h.HandleWebSocketUpgrade)

		// WebSocket management endpoints
		ws.GET("/stats", authMiddleware.RequireAuth(), h.GetConnectionStats)
		ws.POST("/broadcast", authMiddleware.RequireAdmin(), h.BroadcastMessage)
	}
}

// HandleWebSocketUpgrade upgrades HTTP connection to WebSocket
func (h *WebSocketHandler) HandleWebSocketUpgrade(c *gin.Context) {
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

	// Get optional project ID from query parameter
	projectID := c.Query("project_id")

	// Upgrade connection
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Create WebSocket connection
	wsConn := &WebSocketConnection{
		ID:            fmt.Sprintf("%s-%d", user.ID, time.Now().UnixNano()),
		UserID:        user.ID,
		ProjectID:     projectID,
		Connection:    conn,
		Send:          make(chan []byte, 256),
		EventTypes:    h.getDefaultEventTypes(),
		LastPing:      time.Now(),
		Subscriptions: make(map[string]*domain.EventSubscription),
	}

	// Register connection
	h.connectionManager.register <- wsConn

	// Start connection handlers
	go h.handleWebSocketConnection(wsConn)
	go h.handleWebSocketWrites(wsConn)

	log.Printf("WebSocket connection established for user %s", user.ID)
}

// handleWebSocketConnection handles incoming WebSocket messages
func (h *WebSocketHandler) handleWebSocketConnection(conn *WebSocketConnection) {
	defer func() {
		h.connectionManager.unregister <- conn
		_ = conn.Connection.Close()
	}()

	// Set connection timeouts
	_ = conn.Connection.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.Connection.SetPongHandler(func(string) error {
		conn.LastPing = time.Now()
		_ = conn.Connection.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var msg WebSocketMessage
		err := conn.Connection.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error for user %s: %v", conn.UserID, err)
			}
			break
		}

		// Handle different message types
		h.handleWebSocketMessage(conn, &msg)
	}
}

// handleWebSocketWrites handles outgoing WebSocket messages
func (h *WebSocketHandler) handleWebSocketWrites(conn *WebSocketConnection) {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		_ = conn.Connection.Close()
	}()

	for {
		select {
		case message, ok := <-conn.Send:
			_ = conn.Connection.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				_ = conn.Connection.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := conn.Connection.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("WebSocket write error for user %s: %v", conn.UserID, err)
				return
			}

		case <-ticker.C:
			_ = conn.Connection.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.Connection.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleWebSocketMessage processes incoming WebSocket messages
func (h *WebSocketHandler) handleWebSocketMessage(conn *WebSocketConnection, msg *WebSocketMessage) {
	ctx := context.Background()

	switch msg.Type {
	case "subscribe":
		h.handleSubscribeMessage(ctx, conn, msg)
	case "unsubscribe":
		h.handleUnsubscribeMessage(ctx, conn, msg)
	case "ping":
		h.handlePingMessage(conn, msg)
	case "task_update":
		h.handleTaskUpdateMessage(ctx, conn, msg)
	case "project_join":
		h.handleProjectJoinMessage(ctx, conn, msg)
	case "project_leave":
		h.handleProjectLeaveMessage(ctx, conn, msg)
	default:
		h.sendErrorMessage(conn, fmt.Sprintf("Unknown message type: %s", msg.Type))
	}
}

// handleSubscribeMessage handles subscription requests
func (h *WebSocketHandler) handleSubscribeMessage(
	ctx context.Context, conn *WebSocketConnection, msg *WebSocketMessage,
) {
	projectID, _ := msg.Data["project_id"].(string)
	eventTypesRaw, _ := msg.Data["event_types"].([]interface{})

	var eventTypes []domain.TaskEventType
	for _, et := range eventTypesRaw {
		if etStr, ok := et.(string); ok {
			eventType := domain.TaskEventType(etStr)
			if eventType.IsValid() {
				eventTypes = append(eventTypes, eventType)
			}
		}
	}

	if len(eventTypes) == 0 {
		eventTypes = h.getDefaultEventTypes()
	}

	// Create subscription
	subReq := services.CreateSubscriptionRequest{
		UserID:     conn.UserID,
		ProjectID:  &projectID,
		EventTypes: eventTypes,
	}

	subscription, err := h.subscriptionManager.CreateSubscription(ctx, subReq)
	if err != nil {
		h.sendErrorMessage(conn, fmt.Sprintf("Failed to create subscription: %v", err))
		return
	}

	conn.mutex.Lock()
	conn.Subscriptions[subscription.ID] = subscription
	conn.EventTypes = eventTypes
	if projectID != "" {
		conn.ProjectID = projectID
	}
	conn.mutex.Unlock()

	// Send confirmation
	response := WebSocketMessage{
		Type: "subscription_created",
		Data: map[string]interface{}{
			"subscription_id": subscription.ID,
			"project_id":      projectID,
			"event_types":     eventTypes,
		},
		Timestamp: time.Now(),
	}

	h.sendMessage(conn, &response)
	log.Printf("User %s subscribed to events for project %s", conn.UserID, projectID)
}

// handleUnsubscribeMessage handles unsubscription requests
func (h *WebSocketHandler) handleUnsubscribeMessage(
	ctx context.Context, conn *WebSocketConnection, msg *WebSocketMessage,
) {
	subscriptionID, _ := msg.Data["subscription_id"].(string)

	conn.mutex.Lock()
	_, exists := conn.Subscriptions[subscriptionID]
	if exists {
		delete(conn.Subscriptions, subscriptionID)
	}
	conn.mutex.Unlock()

	if !exists {
		h.sendErrorMessage(conn, "Subscription not found")
		return
	}

	// Delete subscription
	err := h.subscriptionManager.DeleteSubscription(ctx, subscriptionID, conn.UserID)
	if err != nil {
		h.sendErrorMessage(conn, fmt.Sprintf("Failed to delete subscription: %v", err))
		return
	}

	// Send confirmation
	response := WebSocketMessage{
		Type: "subscription_deleted",
		Data: map[string]interface{}{
			"subscription_id": subscriptionID,
		},
		Timestamp: time.Now(),
	}

	h.sendMessage(conn, &response)
	log.Printf("User %s unsubscribed from subscription %s", conn.UserID, subscriptionID)
}

// handlePingMessage handles ping messages
func (h *WebSocketHandler) handlePingMessage(conn *WebSocketConnection, _ *WebSocketMessage) {
	conn.LastPing = time.Now()

	response := WebSocketMessage{
		Type: "pong",
		Data: map[string]interface{}{
			"server_time": time.Now().Unix(),
		},
		Timestamp: time.Now(),
	}

	h.sendMessage(conn, &response)
}

// handleTaskUpdateMessage handles real-time task updates
func (h *WebSocketHandler) handleTaskUpdateMessage(
	_ context.Context, conn *WebSocketConnection, msg *WebSocketMessage,
) {
	taskID, _ := msg.Data["task_id"].(string)
	if taskID == "" {
		h.sendErrorMessage(conn, "task_id is required")
		return
	}

	// Create task event for broadcasting
	dataBytes, _ := json.Marshal(msg.Data)
	taskEvent := &domain.TaskEvent{
		EventID:   fmt.Sprintf("evt-%d", time.Now().UnixNano()),
		Type:      domain.TaskUpdated,
		TaskID:    taskID,
		ProjectID: conn.ProjectID,
		UserID:    conn.UserID,
		Data:      dataBytes,
		Timestamp: time.Now(),
	}

	// Broadcast to connection manager
	broadcastMsg := &BroadcastMessage{
		ProjectID: conn.ProjectID,
		EventType: domain.TaskUpdated,
		Data:      taskEvent,
	}

	h.connectionManager.broadcast <- broadcastMsg
	log.Printf("Broadcasting task update for task %s from user %s", taskID, conn.UserID)
}

// handleProjectJoinMessage handles project join events
func (h *WebSocketHandler) handleProjectJoinMessage(
	_ context.Context, conn *WebSocketConnection, msg *WebSocketMessage,
) {
	projectID, _ := msg.Data["project_id"].(string)
	if projectID == "" {
		h.sendErrorMessage(conn, "project_id is required")
		return
	}

	conn.mutex.Lock()
	conn.ProjectID = projectID
	conn.mutex.Unlock()

	response := WebSocketMessage{
		Type: "project_joined",
		Data: map[string]interface{}{
			"project_id": projectID,
			"user_id":    conn.UserID,
		},
		Timestamp: time.Now(),
	}

	h.sendMessage(conn, &response)

	// Notify other users in the project
	broadcastMsg := &BroadcastMessage{
		ProjectID: projectID,
		EventType: domain.TaskEventType("user_joined_project"),
		Data: map[string]interface{}{
			"user_id":    conn.UserID,
			"project_id": projectID,
		},
	}

	h.connectionManager.broadcast <- broadcastMsg
}

// handleProjectLeaveMessage handles project leave events
func (h *WebSocketHandler) handleProjectLeaveMessage(
	_ context.Context, conn *WebSocketConnection, _ *WebSocketMessage,
) {
	projectID := conn.ProjectID

	conn.mutex.Lock()
	conn.ProjectID = ""
	conn.mutex.Unlock()

	response := WebSocketMessage{
		Type: "project_left",
		Data: map[string]interface{}{
			"project_id": projectID,
			"user_id":    conn.UserID,
		},
		Timestamp: time.Now(),
	}

	h.sendMessage(conn, &response)
}

// sendMessage sends a message to a WebSocket connection
func (h *WebSocketHandler) sendMessage(conn *WebSocketConnection, msg *WebSocketMessage) {
	messageBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal WebSocket message: %v", err)
		return
	}

	select {
	case conn.Send <- messageBytes:
	default:
		// Connection buffer is full, close it
		close(conn.Send)
	}
}

// sendErrorMessage sends an error message to a WebSocket connection
func (h *WebSocketHandler) sendErrorMessage(conn *WebSocketConnection, errorMsg string) {
	msg := &WebSocketMessage{
		Type:      "error",
		Error:     errorMsg,
		Timestamp: time.Now(),
	}

	h.sendMessage(conn, msg)
}

// getDefaultEventTypes returns default event types for subscriptions
func (h *WebSocketHandler) getDefaultEventTypes() []domain.TaskEventType {
	return []domain.TaskEventType{
		domain.TaskCreated,
		domain.TaskUpdated,
		domain.TaskMoved,
		domain.TaskDeleted,
		domain.TaskAssigned,
		domain.TaskCommented,
	}
}

// GetConnectionStats returns connection statistics
func (h *WebSocketHandler) GetConnectionStats(c *gin.Context) {
	stats := h.connectionManager.getStats()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
		"message": "Connection statistics retrieved successfully",
	})
}

// BroadcastMessage broadcasts a message to all relevant connections
func (h *WebSocketHandler) BroadcastMessage(c *gin.Context) {
	var req struct {
		ProjectID    string      `json:"project_id"`
		EventType    string      `json:"event_type"`
		Data         interface{} `json:"data"`
		TargetUserID string      `json:"target_user_id,omitempty"`
	}

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

	eventType := domain.TaskEventType(req.EventType)
	if !eventType.IsValid() {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "VALIDATION_ERROR",
				"code":    "INVALID_EVENT_TYPE",
				"message": "Invalid event type",
			},
		})
		return
	}

	broadcastMsg := &BroadcastMessage{
		ProjectID:    req.ProjectID,
		EventType:    eventType,
		Data:         req.Data,
		TargetUserID: req.TargetUserID,
	}

	h.connectionManager.broadcast <- broadcastMsg

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Message broadcasted successfully",
	})
}

// Connection Manager Methods

// run starts the connection manager
func (cm *ConnectionManager) run() {
	for {
		select {
		case conn := <-cm.register:
			cm.mutex.Lock()
			cm.connections[conn.ID] = conn
			cm.mutex.Unlock()
			log.Printf("WebSocket connection registered: %s", conn.ID)

		case conn := <-cm.unregister:
			cm.mutex.Lock()
			if _, exists := cm.connections[conn.ID]; exists {
				delete(cm.connections, conn.ID)
				close(conn.Send)
			}
			cm.mutex.Unlock()
			log.Printf("WebSocket connection unregistered: %s", conn.ID)

		case message := <-cm.broadcast:
			cm.broadcastMessage(message)
		}
	}
}

// broadcastMessage broadcasts a message to relevant connections
func (cm *ConnectionManager) broadcastMessage(msg *BroadcastMessage) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	messageData := WebSocketMessage{
		Type:   "event",
		Action: string(msg.EventType),
		Data: map[string]interface{}{
			"project_id": msg.ProjectID,
			"event_type": msg.EventType,
			"data":       msg.Data,
		},
		Timestamp: time.Now(),
	}

	messageBytes, err := json.Marshal(messageData)
	if err != nil {
		log.Printf("Failed to marshal broadcast message: %v", err)
		return
	}

	sentCount := 0
	for _, conn := range cm.connections {
		shouldSend := false

		// Check if message should be sent to this connection
		switch {
		case msg.TargetUserID != "":
			shouldSend = conn.UserID == msg.TargetUserID
		case msg.ProjectID != "":
			shouldSend = conn.ProjectID == msg.ProjectID
		default:
			shouldSend = true // Broadcast to all
		}

		// Check if connection is subscribed to this event type
		if shouldSend {
			conn.mutex.RLock()
			eventTypeMatches := false
			for _, eventType := range conn.EventTypes {
				if eventType == msg.EventType {
					eventTypeMatches = true
					break
				}
			}
			conn.mutex.RUnlock()

			if eventTypeMatches {
				select {
				case conn.Send <- messageBytes:
					sentCount++
				default:
					// Connection buffer is full, close it
					delete(cm.connections, conn.ID)
					close(conn.Send)
				}
			}
		}
	}

	log.Printf("Broadcast message sent to %d connections", sentCount)
}

// getStats returns connection statistics
func (cm *ConnectionManager) getStats() *ConnectionStats {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	stats := &ConnectionStats{
		TotalConnections:     len(cm.connections),
		ConnectionsByUser:    make(map[string]int),
		ConnectionsByProject: make(map[string]int),
		EventTypeStats:       make(map[domain.TaskEventType]int),
		LastUpdated:          time.Now(),
	}

	for _, conn := range cm.connections {
		conn.mutex.RLock()

		stats.ConnectionsByUser[conn.UserID]++

		if conn.ProjectID != "" {
			stats.ConnectionsByProject[conn.ProjectID]++
		}

		for _, eventType := range conn.EventTypes {
			stats.EventTypeStats[eventType]++
		}

		conn.mutex.RUnlock()
	}

	return stats
}

// CleanupStaleConnections removes connections that haven't pinged recently
func (h *WebSocketHandler) CleanupStaleConnections() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		h.connectionManager.mutex.Lock()
		staleThreshold := time.Now().Add(-2 * time.Minute)

		for id, conn := range h.connectionManager.connections {
			if conn.LastPing.Before(staleThreshold) {
				delete(h.connectionManager.connections, id)
				close(conn.Send)
				_ = conn.Connection.Close()
				log.Printf("Cleaned up stale WebSocket connection: %s", id)
			}
		}

		h.connectionManager.mutex.Unlock()
	}
}

// StartCleanupWorker starts the cleanup worker in a goroutine
func (h *WebSocketHandler) StartCleanupWorker() {
	go h.CleanupStaleConnections()
}

// GetActiveConnections returns the current number of active connections
func (h *WebSocketHandler) GetActiveConnections() int {
	h.connectionManager.mutex.RLock()
	defer h.connectionManager.mutex.RUnlock()
	return len(h.connectionManager.connections)
}

// SendToUser sends a message to all connections for a specific user
func (h *WebSocketHandler) SendToUser(userID string, eventType domain.TaskEventType, data interface{}) {
	broadcastMsg := &BroadcastMessage{
		EventType:    eventType,
		Data:         data,
		TargetUserID: userID,
	}

	h.connectionManager.broadcast <- broadcastMsg
}

// SendToProject sends a message to all connections for a specific project
func (h *WebSocketHandler) SendToProject(projectID string, eventType domain.TaskEventType, data interface{}) {
	broadcastMsg := &BroadcastMessage{
		ProjectID: projectID,
		EventType: eventType,
		Data:      data,
	}

	h.connectionManager.broadcast <- broadcastMsg
}

// BroadcastToAll sends a message to all active connections
func (h *WebSocketHandler) BroadcastToAll(eventType domain.TaskEventType, data interface{}) {
	broadcastMsg := &BroadcastMessage{
		EventType: eventType,
		Data:      data,
	}

	h.connectionManager.broadcast <- broadcastMsg
}
