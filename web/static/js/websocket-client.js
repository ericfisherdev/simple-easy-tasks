/**
 * WebSocket Client with Auto-Reconnection
 * Manages WebSocket connections with automatic reconnection, heartbeat, and event handling
 */

class WebSocketClient {
    constructor(options = {}) {
        this.url = options.url || this.getWebSocketURL();
        this.token = options.token || this.getAuthToken();
        this.projectId = options.projectId || null;
        this.autoReconnect = options.autoReconnect !== false;
        this.heartbeatInterval = options.heartbeatInterval || 30000; // 30 seconds
        this.reconnectDelay = options.reconnectDelay || 1000; // Start with 1 second
        this.maxReconnectDelay = options.maxReconnectDelay || 30000; // Max 30 seconds
        this.maxReconnectAttempts = options.maxReconnectAttempts || -1; // Infinite
        this.debug = options.debug || false;

        this.ws = null;
        this.isConnected = false;
        this.reconnectAttempts = 0;
        this.heartbeatTimer = null;
        this.reconnectTimer = null;
        this.subscriptions = new Map();
        this.eventHandlers = new Map();
        this.messageQueue = [];
        
        // Connection state callbacks
        this.onConnect = options.onConnect || (() => {});
        this.onDisconnect = options.onDisconnect || (() => {});
        this.onError = options.onError || (() => {});
        this.onReconnecting = options.onReconnecting || (() => {});
        
        this.init();
    }

    /**
     * Initialize the WebSocket connection
     */
    init() {
        this.log('Initializing WebSocket client');
        this.connect();
    }

    /**
     * Establish WebSocket connection
     */
    connect() {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.log('WebSocket already connected');
            return;
        }

        const wsUrl = `${this.url}?token=${encodeURIComponent(this.token)}${this.projectId ? `&project_id=${this.projectId}` : ''}`;
        this.log(`Connecting to: ${wsUrl}`);

        try {
            this.ws = new WebSocket(wsUrl);
            this.setupEventListeners();
        } catch (error) {
            this.log('Failed to create WebSocket connection:', error);
            this.handleConnectionError(error);
        }
    }

    /**
     * Setup WebSocket event listeners
     */
    setupEventListeners() {
        this.ws.onopen = (event) => {
            this.log('WebSocket connected');
            this.isConnected = true;
            this.reconnectAttempts = 0;
            this.reconnectDelay = 1000;
            
            this.startHeartbeat();
            this.processMessageQueue();
            this.restoreSubscriptions();
            this.onConnect(event);
        };

        this.ws.onmessage = (event) => {
            try {
                const message = JSON.parse(event.data);
                this.handleMessage(message);
            } catch (error) {
                this.log('Failed to parse WebSocket message:', error);
            }
        };

        this.ws.onclose = (event) => {
            this.log(`WebSocket closed: ${event.code} - ${event.reason}`);
            this.isConnected = false;
            this.stopHeartbeat();
            
            if (!event.wasClean && this.autoReconnect) {
                this.scheduleReconnect();
            }
            
            this.onDisconnect(event);
        };

        this.ws.onerror = (error) => {
            this.log('WebSocket error:', error);
            this.handleConnectionError(error);
        };
    }

    /**
     * Handle incoming WebSocket messages
     */
    handleMessage(message) {
        this.log('Received message:', message);

        switch (message.type) {
            case 'pong':
                this.log('Received heartbeat pong');
                break;
                
            case 'subscription_created':
                this.handleSubscriptionCreated(message);
                break;
                
            case 'subscription_deleted':
                this.handleSubscriptionDeleted(message);
                break;
                
            case 'event':
                this.handleRealtimeEvent(message);
                break;
                
            case 'error':
                this.log('Server error:', message.error);
                this.triggerEvent('error', { error: message.error, message: message });
                break;
                
            case 'project_joined':
                this.handleProjectJoined(message);
                break;
                
            case 'project_left':
                this.handleProjectLeft(message);
                break;
                
            default:
                this.log('Unknown message type:', message.type);
                this.triggerEvent('unknown_message', message);
                break;
        }
    }

    /**
     * Handle subscription creation confirmation
     */
    handleSubscriptionCreated(message) {
        const { subscription_id, project_id, event_types } = message.data;
        this.subscriptions.set(subscription_id, {
            id: subscription_id,
            projectId: project_id,
            eventTypes: event_types,
            active: true
        });
        this.triggerEvent('subscription_created', message.data);
    }

    /**
     * Handle subscription deletion confirmation
     */
    handleSubscriptionDeleted(message) {
        const { subscription_id } = message.data;
        this.subscriptions.delete(subscription_id);
        this.triggerEvent('subscription_deleted', message.data);
    }

    /**
     * Handle real-time events
     */
    handleRealtimeEvent(message) {
        const { action, data } = message;
        this.triggerEvent('realtime_event', { action, data });
        this.triggerEvent(`event_${action}`, data);
    }

    /**
     * Handle project join confirmation
     */
    handleProjectJoined(message) {
        this.projectId = message.data.project_id;
        this.triggerEvent('project_joined', message.data);
    }

    /**
     * Handle project leave confirmation
     */
    handleProjectLeft(message) {
        this.projectId = null;
        this.triggerEvent('project_left', message.data);
    }

    /**
     * Send a message to the server
     */
    send(message) {
        if (this.isConnected && this.ws.readyState === WebSocket.OPEN) {
            const messageWithTimestamp = {
                ...message,
                timestamp: new Date().toISOString()
            };
            this.ws.send(JSON.stringify(messageWithTimestamp));
            this.log('Sent message:', messageWithTimestamp);
        } else {
            this.log('WebSocket not connected, queuing message:', message);
            this.messageQueue.push(message);
        }
    }

    /**
     * Subscribe to events
     */
    subscribe(eventTypes, projectId = null, filters = {}) {
        const subscribeMessage = {
            type: 'subscribe',
            data: {
                event_types: eventTypes,
                project_id: projectId || this.projectId,
                filters: filters
            }
        };
        
        this.send(subscribeMessage);
        return this.generateSubscriptionId();
    }

    /**
     * Unsubscribe from events
     */
    unsubscribe(subscriptionId) {
        const unsubscribeMessage = {
            type: 'unsubscribe',
            data: {
                subscription_id: subscriptionId
            }
        };
        
        this.send(unsubscribeMessage);
        this.subscriptions.delete(subscriptionId);
    }

    /**
     * Join a project room
     */
    joinProject(projectId) {
        const joinMessage = {
            type: 'project_join',
            data: {
                project_id: projectId
            }
        };
        
        this.send(joinMessage);
        this.projectId = projectId;
    }

    /**
     * Leave a project room
     */
    leaveProject() {
        const leaveMessage = {
            type: 'project_leave',
            data: {}
        };
        
        this.send(leaveMessage);
        this.projectId = null;
    }

    /**
     * Send a task update event
     */
    sendTaskUpdate(taskId, updateData) {
        const updateMessage = {
            type: 'task_update',
            data: {
                task_id: taskId,
                ...updateData
            }
        };
        
        this.send(updateMessage);
    }

    /**
     * Start heartbeat to keep connection alive
     */
    startHeartbeat() {
        this.stopHeartbeat(); // Clear any existing timer
        
        this.heartbeatTimer = setInterval(() => {
            if (this.isConnected) {
                this.send({ type: 'ping' });
            }
        }, this.heartbeatInterval);
    }

    /**
     * Stop heartbeat timer
     */
    stopHeartbeat() {
        if (this.heartbeatTimer) {
            clearInterval(this.heartbeatTimer);
            this.heartbeatTimer = null;
        }
    }

    /**
     * Schedule automatic reconnection
     */
    scheduleReconnect() {
        if (!this.autoReconnect) {
            return;
        }

        if (this.maxReconnectAttempts > 0 && this.reconnectAttempts >= this.maxReconnectAttempts) {
            this.log('Max reconnection attempts reached');
            this.triggerEvent('max_reconnect_attempts_reached');
            return;
        }

        this.reconnectAttempts++;
        this.log(`Scheduling reconnect attempt ${this.reconnectAttempts} in ${this.reconnectDelay}ms`);
        
        this.onReconnecting({
            attempt: this.reconnectAttempts,
            delay: this.reconnectDelay
        });

        this.reconnectTimer = setTimeout(() => {
            this.log(`Reconnect attempt ${this.reconnectAttempts}`);
            this.connect();
        }, this.reconnectDelay);

        // Exponential backoff with jitter
        this.reconnectDelay = Math.min(
            this.reconnectDelay * 2 + Math.random() * 1000,
            this.maxReconnectDelay
        );
    }

    /**
     * Process queued messages after reconnection
     */
    processMessageQueue() {
        while (this.messageQueue.length > 0) {
            const message = this.messageQueue.shift();
            this.send(message);
        }
    }

    /**
     * Restore subscriptions after reconnection
     */
    restoreSubscriptions() {
        for (const [_, subscription] of this.subscriptions) {
            if (subscription.active) {
                this.subscribe(subscription.eventTypes, subscription.projectId);
            }
        }
    }

    /**
     * Handle connection errors
     */
    handleConnectionError(error) {
        this.onError(error);
        
        if (this.autoReconnect && !this.reconnectTimer) {
            this.scheduleReconnect();
        }
    }

    /**
     * Register event handler
     */
    on(eventName, handler) {
        if (!this.eventHandlers.has(eventName)) {
            this.eventHandlers.set(eventName, []);
        }
        this.eventHandlers.get(eventName).push(handler);
    }

    /**
     * Unregister event handler
     */
    off(eventName, handler) {
        if (this.eventHandlers.has(eventName)) {
            const handlers = this.eventHandlers.get(eventName);
            const index = handlers.indexOf(handler);
            if (index > -1) {
                handlers.splice(index, 1);
            }
        }
    }

    /**
     * Trigger event handlers
     */
    triggerEvent(eventName, data = {}) {
        if (this.eventHandlers.has(eventName)) {
            this.eventHandlers.get(eventName).forEach(handler => {
                try {
                    handler(data);
                } catch (error) {
                    this.log('Error in event handler:', error);
                }
            });
        }
    }

    /**
     * Get connection status
     */
    getStatus() {
        return {
            isConnected: this.isConnected,
            readyState: this.ws ? this.ws.readyState : WebSocket.CLOSED,
            reconnectAttempts: this.reconnectAttempts,
            subscriptions: Array.from(this.subscriptions.values()),
            queuedMessages: this.messageQueue.length
        };
    }

    /**
     * Manually disconnect
     */
    disconnect() {
        this.log('Manually disconnecting WebSocket');
        this.autoReconnect = false;
        
        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer);
            this.reconnectTimer = null;
        }
        
        this.stopHeartbeat();
        
        if (this.ws) {
            this.ws.close(1000, 'Manual disconnect');
        }
    }

    /**
     * Reconnect manually
     */
    reconnect() {
        this.log('Manual reconnect triggered');
        this.autoReconnect = true;
        this.reconnectAttempts = 0;
        this.reconnectDelay = 1000;
        
        if (this.ws) {
            this.ws.close();
        }
        
        setTimeout(() => this.connect(), 100);
    }

    /**
     * Helper methods
     */
    
    getWebSocketURL() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const host = window.location.host;
        return `${protocol}//${host}/api/ws/connect`;
    }

    getAuthToken() {
        // Try to get token from localStorage, sessionStorage, or cookie
        return localStorage.getItem('access_token') || 
               sessionStorage.getItem('access_token') || 
               this.getCookie('access_token') || 
               '';
    }

    getCookie(name) {
        const value = `; ${document.cookie}`;
        const parts = value.split(`; ${name}=`);
        if (parts.length === 2) {
            return parts.pop().split(';').shift();
        }
        return null;
    }

    generateSubscriptionId() {
        return `sub_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }

    log(...args) {
        if (this.debug) {
            console.log('[WebSocketClient]', ...args);
        }
    }
}

/**
 * WebSocket Client Manager
 * Provides a singleton interface for managing WebSocket connections
 */
class WebSocketClientManager {
    constructor() {
        this.client = null;
        this.eventHandlers = new Map();
    }

    /**
     * Initialize WebSocket client
     */
    init(options = {}) {
        if (this.client) {
            this.client.disconnect();
        }

        this.client = new WebSocketClient({
            debug: true,
            ...options,
            onConnect: (event) => {
                this.triggerEvent('connect', event);
                if (options.onConnect) options.onConnect(event);
            },
            onDisconnect: (event) => {
                this.triggerEvent('disconnect', event);
                if (options.onDisconnect) options.onDisconnect(event);
            },
            onError: (error) => {
                this.triggerEvent('error', error);
                if (options.onError) options.onError(error);
            },
            onReconnecting: (data) => {
                this.triggerEvent('reconnecting', data);
                if (options.onReconnecting) options.onReconnecting(data);
            }
        });

        // Forward client events
        this.client.on('realtime_event', (data) => this.triggerEvent('realtime_event', data));
        this.client.on('subscription_created', (data) => this.triggerEvent('subscription_created', data));
        this.client.on('subscription_deleted', (data) => this.triggerEvent('subscription_deleted', data));
        this.client.on('project_joined', (data) => this.triggerEvent('project_joined', data));
        this.client.on('project_left', (data) => this.triggerEvent('project_left', data));

        return this.client;
    }

    /**
     * Get the current client instance
     */
    getClient() {
        return this.client;
    }

    /**
     * Check if client is connected
     */
    isConnected() {
        return this.client ? this.client.isConnected : false;
    }

    /**
     * Subscribe to task events for a project
     */
    subscribeToProject(projectId, eventTypes = null) {
        if (!this.client) {
            throw new Error('WebSocket client not initialized');
        }

        const events = eventTypes || [
            'task_created',
            'task_updated', 
            'task_moved',
            'task_assigned',
            'task_deleted',
            'task_commented'
        ];

        return this.client.subscribe(events, projectId);
    }

    /**
     * Send task update
     */
    sendTaskUpdate(taskId, updateData) {
        if (this.client) {
            this.client.sendTaskUpdate(taskId, updateData);
        }
    }

    /**
     * Join project
     */
    joinProject(projectId) {
        if (this.client) {
            this.client.joinProject(projectId);
        }
    }

    /**
     * Leave project
     */
    leaveProject() {
        if (this.client) {
            this.client.leaveProject();
        }
    }

    /**
     * Event handling
     */
    on(eventName, handler) {
        if (!this.eventHandlers.has(eventName)) {
            this.eventHandlers.set(eventName, []);
        }
        this.eventHandlers.get(eventName).push(handler);
    }

    off(eventName, handler) {
        if (this.eventHandlers.has(eventName)) {
            const handlers = this.eventHandlers.get(eventName);
            const index = handlers.indexOf(handler);
            if (index > -1) {
                handlers.splice(index, 1);
            }
        }
    }

    triggerEvent(eventName, data = {}) {
        if (this.eventHandlers.has(eventName)) {
            this.eventHandlers.get(eventName).forEach(handler => {
                try {
                    handler(data);
                } catch (error) {
                    console.error('Error in WebSocket event handler:', error);
                }
            });
        }
    }

    /**
     * Get connection status
     */
    getStatus() {
        return this.client ? this.client.getStatus() : { isConnected: false };
    }

    /**
     * Disconnect
     */
    disconnect() {
        if (this.client) {
            this.client.disconnect();
        }
    }
}

// Create global instance
window.WebSocketManager = new WebSocketClientManager();

// Export classes for module usage
if (typeof module !== 'undefined' && module.exports) {
    module.exports = { WebSocketClient, WebSocketClientManager };
}