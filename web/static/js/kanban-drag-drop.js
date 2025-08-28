/**
 * HTMX Kanban Drag and Drop Integration
 * Provides drag-and-drop functionality for task cards with HTMX integration
 */

class KanbanDragDrop {
    constructor(options = {}) {
        this.boardSelector = options.boardSelector || '.kanban-board';
        this.columnSelector = options.columnSelector || '.kanban-column';
        this.cardSelector = options.cardSelector || '.task-card';
        this.dragHandleSelector = options.dragHandleSelector || '.drag-handle';
        this.dropZoneSelector = options.dropZoneSelector || '.drop-zone';
        
        this.apiEndpoint = options.apiEndpoint || '/api/projects';
        this.csrfToken = options.csrfToken || this.getCSRFToken();
        
        this.onTaskMoved = options.onTaskMoved || (() => {});
        this.onDragStart = options.onDragStart || (() => {});
        this.onDragEnd = options.onDragEnd || (() => {});
        this.onError = options.onError || ((error) => console.error('Drag drop error:', error));
        
        this.draggedElement = null;
        this.sourceColumn = null;
        this.dragPlaceholder = null;
        
        this.isWebSocketEnabled = options.enableWebSocket !== false;
        this.websocketManager = window.WebSocketManager;
        
        this.init();
    }

    /**
     * Initialize drag and drop functionality
     */
    init() {
        this.setupDragAndDrop();
        this.setupDropZones();
        this.setupWebSocketIntegration();
        this.setupTouchEvents();
        
        console.log('Kanban drag and drop initialized');
    }

    /**
     * Setup drag and drop event listeners
     */
    setupDragAndDrop() {
        document.addEventListener('dragstart', (e) => this.handleDragStart(e));
        document.addEventListener('dragend', (e) => this.handleDragEnd(e));
        document.addEventListener('dragover', (e) => this.handleDragOver(e));
        document.addEventListener('drop', (e) => this.handleDrop(e));
        document.addEventListener('dragenter', (e) => this.handleDragEnter(e));
        document.addEventListener('dragleave', (e) => this.handleDragLeave(e));
        
        this.makeTaskCardsDraggable();
    }

    /**
     * Make task cards draggable
     */
    makeTaskCardsDraggable() {
        document.querySelectorAll(this.cardSelector).forEach(card => {
            if (!card.hasAttribute('draggable')) {
                card.draggable = true;
                card.setAttribute('data-draggable', 'true');
                
                // Add drag handle styling
                const dragHandle = card.querySelector(this.dragHandleSelector) || card;
                dragHandle.style.cursor = 'grab';
                
                dragHandle.addEventListener('mousedown', () => {
                    dragHandle.style.cursor = 'grabbing';
                });
                
                dragHandle.addEventListener('mouseup', () => {
                    dragHandle.style.cursor = 'grab';
                });
            }
        });
    }

    /**
     * Setup drop zones
     */
    setupDropZones() {
        document.querySelectorAll(this.columnSelector).forEach(column => {
            column.addEventListener('dragover', (e) => e.preventDefault());
            column.addEventListener('drop', (e) => this.handleColumnDrop(e));
            
            // Create visual drop zone
            if (!column.querySelector('.drop-zone-indicator')) {
                const dropZone = document.createElement('div');
                dropZone.className = 'drop-zone-indicator';
                dropZone.style.cssText = `
                    height: 4px;
                    background: #3b82f6;
                    margin: 8px 0;
                    border-radius: 2px;
                    opacity: 0;
                    transition: opacity 0.2s ease;
                `;
                column.appendChild(dropZone);
            }
        });
    }

    /**
     * Setup WebSocket integration for real-time updates
     */
    setupWebSocketIntegration() {
        if (!this.isWebSocketEnabled || !this.websocketManager) {
            return;
        }

        this.websocketManager.on('event_task_moved', (data) => {
            this.handleRemoteTaskMove(data);
        });

        this.websocketManager.on('event_task_updated', (data) => {
            this.handleRemoteTaskUpdate(data);
        });
    }

    /**
     * Setup touch events for mobile devices
     */
    setupTouchEvents() {
        let touchStartY = 0;
        let touchElement = null;
        
        document.addEventListener('touchstart', (e) => {
            if (e.target.closest(this.cardSelector)) {
                touchStartY = e.touches[0].clientY;
                touchElement = e.target.closest(this.cardSelector);
                touchElement.style.transform = 'scale(1.05)';
                touchElement.style.zIndex = '1000';
            }
        });

        document.addEventListener('touchmove', (e) => {
            if (touchElement) {
                e.preventDefault();
                const touch = e.touches[0];
                const deltaY = touch.clientY - touchStartY;
                
                touchElement.style.transform = `translateY(${deltaY}px) scale(1.05)`;
                
                // Find drop target
                const elementBelow = document.elementFromPoint(touch.clientX, touch.clientY);
                this.highlightDropTarget(elementBelow);
            }
        });

        document.addEventListener('touchend', (e) => {
            if (touchElement) {
                const touch = e.changedTouches[0];
                const dropTarget = document.elementFromPoint(touch.clientX, touch.clientY);
                
                this.handleTouchDrop(touchElement, dropTarget);
                
                touchElement.style.transform = '';
                touchElement.style.zIndex = '';
                touchElement = null;
                
                this.clearDropHighlights();
            }
        });
    }

    /**
     * Handle drag start
     */
    handleDragStart(e) {
        const card = e.target.closest(this.cardSelector);
        if (!card) return;

        this.draggedElement = card;
        this.sourceColumn = card.closest(this.columnSelector);
        
        // Create placeholder
        this.dragPlaceholder = this.createPlaceholder(card);
        
        // Set drag data
        e.dataTransfer.effectAllowed = 'move';
        e.dataTransfer.setData('text/html', card.outerHTML);
        e.dataTransfer.setData('application/json', JSON.stringify({
            taskId: card.dataset.taskId,
            sourceColumnId: this.sourceColumn.dataset.columnId,
            sourcePosition: this.getCardPosition(card)
        }));

        // Add drag styling
        setTimeout(() => {
            card.style.opacity = '0.5';
            this.showDropZones();
        }, 0);

        this.onDragStart({ card, sourceColumn: this.sourceColumn });
    }

    /**
     * Handle drag end
     */
    handleDragEnd(e) {
        const card = e.target.closest(this.cardSelector);
        if (!card) return;

        // Reset styling
        card.style.opacity = '';
        
        // Remove placeholder
        if (this.dragPlaceholder && this.dragPlaceholder.parentNode) {
            this.dragPlaceholder.parentNode.removeChild(this.dragPlaceholder);
        }
        
        this.hideDropZones();
        this.clearDropHighlights();
        
        this.onDragEnd({ card });
        
        // Reset state
        this.draggedElement = null;
        this.sourceColumn = null;
        this.dragPlaceholder = null;
    }

    /**
     * Handle drag over
     */
    handleDragOver(e) {
        e.preventDefault();
        e.dataTransfer.dropEffect = 'move';
        
        const column = e.target.closest(this.columnSelector);
        if (column && this.draggedElement) {
            this.updatePlaceholderPosition(e, column);
        }
    }

    /**
     * Handle drag enter
     */
    handleDragEnter(e) {
        const column = e.target.closest(this.columnSelector);
        if (column) {
            this.highlightDropTarget(column);
        }
    }

    /**
     * Handle drag leave
     */
    handleDragLeave(e) {
        const column = e.target.closest(this.columnSelector);
        if (column && !column.contains(e.relatedTarget)) {
            this.removeDropHighlight(column);
        }
    }

    /**
     * Handle drop
     */
    handleDrop(e) {
        e.preventDefault();
        
        const dropTarget = e.target.closest(this.columnSelector);
        if (!dropTarget || !this.draggedElement) return;

        const dragData = JSON.parse(e.dataTransfer.getData('application/json'));
        const newPosition = this.calculateDropPosition(e, dropTarget);
        
        this.moveTask({
            taskId: dragData.taskId,
            sourceColumnId: dragData.sourceColumnId,
            targetColumnId: dropTarget.dataset.columnId,
            sourcePosition: dragData.sourcePosition,
            targetPosition: newPosition
        });
    }

    /**
     * Handle column drop (for cards dropped directly on column)
     */
    handleColumnDrop(e) {
        const column = e.currentTarget;
        const dropZone = column.querySelector('.drop-zone-indicator');
        
        if (dropZone) {
            dropZone.style.opacity = '0';
        }
    }

    /**
     * Handle touch drop for mobile devices
     */
    handleTouchDrop(draggedCard, dropTarget) {
        const targetColumn = dropTarget?.closest(this.columnSelector);
        if (!targetColumn || !draggedCard) return;

        const taskId = draggedCard.dataset.taskId;
        const sourceColumn = draggedCard.closest(this.columnSelector);
        const sourceColumnId = sourceColumn.dataset.columnId;
        const targetColumnId = targetColumn.dataset.columnId;
        
        const sourcePosition = this.getCardPosition(draggedCard);
        const targetPosition = this.calculateTouchDropPosition(dropTarget, targetColumn);

        this.moveTask({
            taskId,
            sourceColumnId,
            targetColumnId,
            sourcePosition,
            targetPosition
        });
    }

    /**
     * Move task via API
     */
    async moveTask(moveData) {
        const { taskId, sourceColumnId, targetColumnId, sourcePosition, targetPosition } = moveData;
        
        // Optimistic update
        this.performOptimisticMove(moveData);
        
        try {
            const projectId = this.getProjectId();
            const response = await this.apiRequest(`${this.apiEndpoint}/${projectId}/tasks/${taskId}/move`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-CSRF-Token': this.csrfToken
                },
                body: JSON.stringify({
                    new_status: this.columnIdToStatus(targetColumnId),
                    new_position: targetPosition
                })
            });

            if (!response.ok) {
                throw new Error(`API request failed: ${response.status} ${response.statusText}`);
            }

            const result = await response.json();
            
            if (!result.success) {
                throw new Error(result.error?.message || 'Task move failed');
            }

            // Notify WebSocket clients
            if (this.websocketManager?.isConnected()) {
                this.websocketManager.sendTaskUpdate(taskId, {
                    action: 'moved',
                    from_column: sourceColumnId,
                    to_column: targetColumnId,
                    from_position: sourcePosition,
                    to_position: targetPosition
                });
            }

            this.onTaskMoved({
                taskId,
                sourceColumnId,
                targetColumnId,
                sourcePosition,
                targetPosition,
                response: result
            });

        } catch (error) {
            console.error('Failed to move task:', error);
            
            // Revert optimistic update
            this.revertOptimisticMove(moveData);
            
            this.onError(error);
            this.showErrorNotification('Failed to move task. Please try again.');
        }
    }

    /**
     * Perform optimistic UI update
     */
    performOptimisticMove(moveData) {
        const { taskId, targetColumnId, targetPosition } = moveData;
        const card = document.querySelector(`[data-task-id="${taskId}"]`);
        const targetColumn = document.querySelector(`[data-column-id="${targetColumnId}"]`);
        
        if (!card || !targetColumn) return;

        // Store original state for potential revert
        card.dataset.originalParent = card.parentNode.dataset.columnId;
        card.dataset.originalPosition = this.getCardPosition(card);

        // Move card in DOM
        const targetCards = Array.from(targetColumn.querySelectorAll(this.cardSelector));
        const insertBefore = targetCards[targetPosition];
        
        if (insertBefore) {
            targetColumn.insertBefore(card, insertBefore);
        } else {
            targetColumn.appendChild(card);
        }

        // Update card visual state if moving between statuses
        this.updateCardStatus(card, this.columnIdToStatus(targetColumnId));
        
        // Add move animation
        card.style.transform = 'scale(0.98)';
        setTimeout(() => {
            card.style.transform = '';
        }, 200);
    }

    /**
     * Revert optimistic move on error
     */
    revertOptimisticMove(moveData) {
        const { taskId } = moveData;
        const card = document.querySelector(`[data-task-id="${taskId}"]`);
        
        if (!card) return;

        const originalParent = document.querySelector(`[data-column-id="${card.dataset.originalParent}"]`);
        const originalPosition = parseInt(card.dataset.originalPosition);

        if (originalParent) {
            const siblings = Array.from(originalParent.querySelectorAll(this.cardSelector));
            const insertBefore = siblings[originalPosition];
            
            if (insertBefore) {
                originalParent.insertBefore(card, insertBefore);
            } else {
                originalParent.appendChild(card);
            }

            // Restore original status
            this.updateCardStatus(card, this.columnIdToStatus(originalParent.dataset.columnId));
        }

        // Clean up stored data
        delete card.dataset.originalParent;
        delete card.dataset.originalPosition;
    }

    /**
     * Handle remote task moves from WebSocket
     */
    handleRemoteTaskMove(data) {
        const { task_id, from_column, to_column, to_position } = data.data;
        const card = document.querySelector(`[data-task-id="${task_id}"]`);
        
        if (!card) return;

        // Don't process moves we initiated
        if (card.dataset.localMove) {
            delete card.dataset.localMove;
            return;
        }

        const targetColumn = document.querySelector(`[data-column-id="${to_column}"]`);
        if (!targetColumn) return;

        // Animate remote move
        card.style.transition = 'transform 0.3s ease, opacity 0.3s ease';
        card.style.transform = 'scale(1.05)';
        card.style.opacity = '0.8';

        setTimeout(() => {
            const targetCards = Array.from(targetColumn.querySelectorAll(this.cardSelector));
            const insertBefore = targetCards[to_position];
            
            if (insertBefore) {
                targetColumn.insertBefore(card, insertBefore);
            } else {
                targetColumn.appendChild(card);
            }

            this.updateCardStatus(card, this.columnIdToStatus(to_column));

            // Reset animation
            card.style.transform = '';
            card.style.opacity = '';
            setTimeout(() => {
                card.style.transition = '';
            }, 300);

        }, 150);
    }

    /**
     * Handle remote task updates from WebSocket
     */
    handleRemoteTaskUpdate(data) {
        const taskData = data.data;
        if (!taskData || !taskData.task_id) return;

        const card = document.querySelector(`[data-task-id="${taskData.task_id}"]`);
        if (!card) return;

        // Update card content with new data
        this.updateCardContent(card, taskData);
        
        // Add update animation
        card.style.animation = 'pulse 0.5s ease';
        setTimeout(() => {
            card.style.animation = '';
        }, 500);
    }

    /**
     * Utility Methods
     */

    createPlaceholder(card) {
        const placeholder = document.createElement('div');
        placeholder.className = 'drag-placeholder';
        placeholder.style.cssText = `
            height: ${card.offsetHeight}px;
            background: rgba(59, 130, 246, 0.1);
            border: 2px dashed #3b82f6;
            border-radius: 8px;
            margin: 8px 0;
            opacity: 0.7;
        `;
        
        card.parentNode.insertBefore(placeholder, card.nextSibling);
        return placeholder;
    }

    updatePlaceholderPosition(e, column) {
        if (!this.dragPlaceholder) return;

        const cards = Array.from(column.querySelectorAll(this.cardSelector));
        const afterElement = this.getDragAfterElement(column, e.clientY);

        if (afterElement == null) {
            column.appendChild(this.dragPlaceholder);
        } else {
            column.insertBefore(this.dragPlaceholder, afterElement);
        }
    }

    getDragAfterElement(container, y) {
        const draggableElements = [...container.querySelectorAll(`${this.cardSelector}:not(.dragging)`)];

        return draggableElements.reduce((closest, child) => {
            const box = child.getBoundingClientRect();
            const offset = y - box.top - box.height / 2;

            if (offset < 0 && offset > closest.offset) {
                return { offset: offset, element: child };
            } else {
                return closest;
            }
        }, { offset: Number.NEGATIVE_INFINITY }).element;
    }

    calculateDropPosition(e, column) {
        const cards = Array.from(column.querySelectorAll(this.cardSelector));
        const y = e.clientY;

        for (let i = 0; i < cards.length; i++) {
            const cardRect = cards[i].getBoundingClientRect();
            if (y < cardRect.top + cardRect.height / 2) {
                return i;
            }
        }

        return cards.length;
    }

    calculateTouchDropPosition(dropTarget, column) {
        const cards = Array.from(column.querySelectorAll(this.cardSelector));
        
        if (dropTarget.classList.contains(this.cardSelector.slice(1))) {
            const targetIndex = cards.indexOf(dropTarget);
            return targetIndex >= 0 ? targetIndex : cards.length;
        }
        
        return cards.length;
    }

    getCardPosition(card) {
        const column = card.closest(this.columnSelector);
        if (!column) return 0;
        
        const cards = Array.from(column.querySelectorAll(this.cardSelector));
        return cards.indexOf(card);
    }

    highlightDropTarget(element) {
        const column = element?.closest(this.columnSelector);
        if (!column) return;

        column.classList.add('drag-over');
        const dropZone = column.querySelector('.drop-zone-indicator');
        if (dropZone) {
            dropZone.style.opacity = '1';
        }
    }

    removeDropHighlight(column) {
        column.classList.remove('drag-over');
        const dropZone = column.querySelector('.drop-zone-indicator');
        if (dropZone) {
            dropZone.style.opacity = '0';
        }
    }

    clearDropHighlights() {
        document.querySelectorAll(this.columnSelector).forEach(column => {
            this.removeDropHighlight(column);
        });
    }

    showDropZones() {
        document.querySelectorAll(this.columnSelector).forEach(column => {
            column.classList.add('drop-zone-active');
        });
    }

    hideDropZones() {
        document.querySelectorAll(this.columnSelector).forEach(column => {
            column.classList.remove('drop-zone-active');
        });
    }

    updateCardStatus(card, status) {
        const statusElement = card.querySelector('.task-status');
        if (statusElement) {
            statusElement.textContent = this.formatStatus(status);
            statusElement.className = `task-status status-${status}`;
        }

        card.dataset.status = status;
    }

    updateCardContent(card, taskData) {
        // Update title
        const titleElement = card.querySelector('.task-title');
        if (titleElement && taskData.title) {
            titleElement.textContent = taskData.title;
        }

        // Update description
        const descElement = card.querySelector('.task-description');
        if (descElement && taskData.description) {
            descElement.textContent = taskData.description;
        }

        // Update priority
        const priorityElement = card.querySelector('.task-priority');
        if (priorityElement && taskData.priority) {
            priorityElement.textContent = this.formatPriority(taskData.priority);
            priorityElement.className = `task-priority priority-${taskData.priority}`;
        }

        // Update assignee
        const assigneeElement = card.querySelector('.task-assignee');
        if (assigneeElement && taskData.assignee_name) {
            assigneeElement.textContent = taskData.assignee_name;
        }
    }

    columnIdToStatus(columnId) {
        const statusMap = {
            'backlog': 'backlog',
            'todo': 'todo', 
            'doing': 'developing',
            'review': 'review',
            'done': 'complete'
        };
        return statusMap[columnId] || columnId;
    }

    formatStatus(status) {
        const statusMap = {
            'backlog': 'Backlog',
            'todo': 'To Do',
            'developing': 'In Progress', 
            'review': 'Review',
            'complete': 'Done'
        };
        return statusMap[status] || status;
    }

    formatPriority(priority) {
        const priorityMap = {
            'low': 'Low',
            'medium': 'Medium',
            'high': 'High',
            'critical': 'Critical'
        };
        return priorityMap[priority] || priority;
    }

    getProjectId() {
        return document.querySelector('[data-project-id]')?.dataset.projectId ||
               new URLSearchParams(window.location.search).get('project_id') ||
               window.location.pathname.match(/\/projects\/([^\/]+)/)?.[1];
    }

    getCSRFToken() {
        return document.querySelector('meta[name="csrf-token"]')?.getAttribute('content') ||
               document.querySelector('[data-csrf-token]')?.dataset.csrfToken ||
               '';
    }

    async apiRequest(url, options) {
        return fetch(url, {
            ...options,
            headers: {
                'Authorization': `Bearer ${this.getAuthToken()}`,
                ...options.headers
            }
        });
    }

    getAuthToken() {
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

    showErrorNotification(message) {
        // Simple error notification - in production use toast/notification library
        const notification = document.createElement('div');
        notification.className = 'error-notification';
        notification.style.cssText = `
            position: fixed;
            top: 20px;
            right: 20px;
            background: #ef4444;
            color: white;
            padding: 12px 16px;
            border-radius: 4px;
            z-index: 10000;
            animation: slideIn 0.3s ease;
        `;
        notification.textContent = message;
        
        document.body.appendChild(notification);
        
        setTimeout(() => {
            notification.remove();
        }, 5000);
    }

    /**
     * Public API Methods
     */

    refreshCards() {
        this.makeTaskCardsDraggable();
    }

    enableDragDrop() {
        document.querySelectorAll(this.cardSelector).forEach(card => {
            card.draggable = true;
        });
    }

    disableDragDrop() {
        document.querySelectorAll(this.cardSelector).forEach(card => {
            card.draggable = false;
        });
    }

    destroy() {
        // Remove event listeners and clean up
        document.removeEventListener('dragstart', this.handleDragStart);
        document.removeEventListener('dragend', this.handleDragEnd);
        document.removeEventListener('dragover', this.handleDragOver);
        document.removeEventListener('drop', this.handleDrop);
        document.removeEventListener('dragenter', this.handleDragEnter);
        document.removeEventListener('dragleave', this.handleDragLeave);
        
        if (this.websocketManager) {
            this.websocketManager.off('event_task_moved', this.handleRemoteTaskMove);
            this.websocketManager.off('event_task_updated', this.handleRemoteTaskUpdate);
        }
    }
}

// Auto-initialize if on kanban board page
document.addEventListener('DOMContentLoaded', () => {
    if (document.querySelector('.kanban-board')) {
        window.kanbanDragDrop = new KanbanDragDrop({
            onTaskMoved: (data) => {
                console.log('Task moved:', data);
                
                // Trigger HTMX events if available
                if (window.htmx) {
                    htmx.trigger(document.body, 'task:moved', data);
                }
            },
            onError: (error) => {
                console.error('Kanban error:', error);
                
                // Trigger HTMX error event if available
                if (window.htmx) {
                    htmx.trigger(document.body, 'task:move-error', { error });
                }
            }
        });
    }
});

// CSS animations
const style = document.createElement('style');
style.textContent = `
    @keyframes slideIn {
        from { transform: translateX(100%); }
        to { transform: translateX(0); }
    }
    
    @keyframes pulse {
        0%, 100% { transform: scale(1); }
        50% { transform: scale(1.02); }
    }
    
    .drag-over {
        background-color: rgba(59, 130, 246, 0.05);
        border: 2px dashed #3b82f6;
    }
    
    .drop-zone-active {
        min-height: 100px;
    }
    
    .task-card[draggable="true"] {
        cursor: grab;
        transition: transform 0.2s ease, box-shadow 0.2s ease;
    }
    
    .task-card[draggable="true"]:hover {
        transform: translateY(-2px);
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
    }
    
    .task-card.dragging {
        opacity: 0.5;
        transform: rotate(5deg);
    }
`;
document.head.appendChild(style);

// Export for module usage
if (typeof module !== 'undefined' && module.exports) {
    module.exports = KanbanDragDrop;
}