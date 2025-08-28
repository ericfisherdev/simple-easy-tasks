/**
 * Task Board Alpine.js Component
 * Provides kanban board functionality with drag-and-drop, real-time updates, and animations
 */
function taskBoard() {
    return {
        // State management
        tasks: [],
        columns: {
            backlog: { title: 'Backlog', tasks: [], color: 'gray' },
            todo: { title: 'Todo', tasks: [], color: 'blue' },
            developing: { title: 'Developing', tasks: [], color: 'yellow' },
            review: { title: 'Review', tasks: [], color: 'purple' },
            complete: { title: 'Complete', tasks: [], color: 'green' }
        },
        draggedTask: null,
        draggedFrom: null,
        isLoading: false,
        error: null,
        
        // Configuration
        projectId: null,
        enableRealtime: true,
        animationDuration: 300,
        
        // Initialization
        init() {
            console.log('Initializing task board...');
            this.projectId = this.$el.dataset.projectId;
            
            if (!this.projectId) {
                console.error('Project ID is required for task board');
                return;
            }
            
            this.loadTasks();
            this.setupDragDrop();
            
            if (this.enableRealtime) {
                this.setupRealtime();
            }
            
            // Setup keyboard shortcuts
            this.setupKeyboardShortcuts();
        },
        
        // Task Management
        async loadTasks() {
            this.isLoading = true;
            this.error = null;
            
            try {
                const response = await fetch(`/api/projects/${this.projectId}/tasks`);
                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }
                
                const data = await response.json();
                this.tasks = data.tasks || [];
                this.organizeTasks();
                
                console.log(`Loaded ${this.tasks.length} tasks`);
            } catch (error) {
                console.error('Failed to load tasks:', error);
                this.error = 'Failed to load tasks. Please try again.';
            } finally {
                this.isLoading = false;
            }
        },
        
        organizeTasks() {
            // Reset all columns
            Object.keys(this.columns).forEach(key => {
                this.columns[key].tasks = [];
            });
            
            // Organize tasks by status
            this.tasks.forEach(task => {
                const status = task.status?.toLowerCase() || 'backlog';
                if (this.columns[status]) {
                    this.columns[status].tasks.push(task);
                }
            });
            
            // Sort tasks by position within each column
            Object.keys(this.columns).forEach(key => {
                this.columns[key].tasks.sort((a, b) => (a.position || 0) - (b.position || 0));
            });
        },
        
        // Task Operations
        async createTask(formData) {
            try {
                const response = await fetch(`/api/projects/${this.projectId}/tasks`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'X-CSRFToken': this.getCSRFToken()
                    },
                    body: JSON.stringify(formData)
                });
                
                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }
                
                const newTask = await response.json();
                this.addTask(newTask);
                
                // Animate task appearance
                this.$nextTick(() => {
                    const taskElement = document.querySelector(`[data-task-id="${newTask.id}"]`);
                    if (taskElement && window.TaskAnimations) {
                        window.TaskAnimations.taskAppear(taskElement);
                    }
                });
                
                return newTask;
            } catch (error) {
                console.error('Failed to create task:', error);
                throw error;
            }
        },
        
        async updateTask(taskId, updates) {
            try {
                const response = await fetch(`/api/projects/${this.projectId}/tasks/${taskId}`, {
                    method: 'PUT',
                    headers: {
                        'Content-Type': 'application/json',
                        'X-CSRFToken': this.getCSRFToken()
                    },
                    body: JSON.stringify(updates)
                });
                
                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }
                
                const updatedTask = await response.json();
                this.replaceTask(updatedTask);
                
                return updatedTask;
            } catch (error) {
                console.error('Failed to update task:', error);
                throw error;
            }
        },
        
        async deleteTask(taskId) {
            try {
                const taskElement = document.querySelector(`[data-task-id="${taskId}"]`);
                
                // Animate task disappearance
                if (taskElement && window.TaskAnimations) {
                    await window.TaskAnimations.taskDisappear(taskElement);
                }
                
                const response = await fetch(`/api/projects/${this.projectId}/tasks/${taskId}`, {
                    method: 'DELETE',
                    headers: {
                        'X-CSRFToken': this.getCSRFToken()
                    }
                });
                
                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }
                
                this.removeTask(taskId);
            } catch (error) {
                console.error('Failed to delete task:', error);
                throw error;
            }
        },
        
        // Task State Management
        addTask(task) {
            this.tasks.push(task);
            this.organizeTasks();
        },
        
        replaceTask(updatedTask) {
            const index = this.tasks.findIndex(t => t.id === updatedTask.id);
            if (index !== -1) {
                this.tasks[index] = updatedTask;
                this.organizeTasks();
            }
        },
        
        removeTask(taskId) {
            this.tasks = this.tasks.filter(t => t.id !== taskId);
            this.organizeTasks();
        },
        
        // Drag and Drop
        setupDragDrop() {
            this.$nextTick(() => {
                if (window.Draggable) {
                    this.initializeGSAPDraggable();
                } else {
                    // Fallback to HTML5 drag and drop
                    this.initializeHTMLDragDrop();
                }
            });
        },
        
        initializeGSAPDraggable() {
            window.Draggable.create('.task-card', {
                type: 'x,y',
                bounds: '.board-container',
                dragClickables: false,
                onDragStart: (e) => this.onDragStart(e.target),
                onDrag: (e) => this.onDrag(e.target),
                onDragEnd: (e) => this.onDragEnd(e.target)
            });
        },
        
        initializeHTMLDragDrop() {
            // HTML5 drag and drop fallback
            document.addEventListener('dragstart', (e) => {
                if (e.target.classList.contains('task-card')) {
                    this.onDragStart(e.target);
                }
            });
            
            document.addEventListener('dragend', (e) => {
                if (e.target.classList.contains('task-card')) {
                    this.onDragEnd(e.target);
                }
            });
        },
        
        onDragStart(element) {
            const taskId = element.dataset.taskId;
            this.draggedTask = this.tasks.find(t => t.id == taskId);
            this.draggedFrom = element.closest('.column').dataset.status;
            
            console.log('Drag started:', this.draggedTask?.title);
            
            // Add dragging visual feedback
            element.classList.add('dragging');
            
            // GSAP animation if available
            if (window.gsap) {
                window.gsap.to(element, {
                    scale: 1.05,
                    rotation: 5,
                    zIndex: 1000,
                    boxShadow: '0 25px 50px -12px rgba(0, 0, 0, 0.25)',
                    duration: 0.2
                });
            }
        },
        
        onDrag(element) {
            // Provide visual feedback for drop zones
            const columns = document.querySelectorAll('.column');
            const elementRect = element.getBoundingClientRect();
            
            columns.forEach(column => {
                const columnRect = column.getBoundingClientRect();
                if (this.isOverlapping(elementRect, columnRect)) {
                    column.classList.add('drop-zone-active');
                } else {
                    column.classList.remove('drop-zone-active');
                }
            });
        },
        
        onDragEnd(element) {
            const dropColumn = this.getDropColumn(element);
            
            element.classList.remove('dragging');
            
            if (dropColumn && dropColumn !== this.draggedFrom) {
                this.moveTask(this.draggedTask.id, dropColumn);
            } else {
                // Animate back to original position
                if (window.gsap) {
                    window.gsap.to(element, {
                        x: 0,
                        y: 0,
                        scale: 1,
                        rotation: 0,
                        duration: 0.3
                    });
                }
            }
            
            // Reset visual states
            if (window.gsap) {
                window.gsap.set(element, { zIndex: 'auto', boxShadow: 'none' });
            }
            
            document.querySelectorAll('.column').forEach(col => {
                col.classList.remove('drop-zone-active');
            });
            
            console.log('Drag ended');
        },
        
        async moveTask(taskId, newStatus) {
            try {
                const response = await fetch(`/api/projects/${this.projectId}/tasks/${taskId}/move`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'X-CSRFToken': this.getCSRFToken()
                    },
                    body: JSON.stringify({ status: newStatus })
                });
                
                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }
                
                const result = await response.json();
                if (result.success) {
                    this.updateTaskStatus(taskId, newStatus);
                    this.animateTaskMove(taskId, newStatus);
                }
            } catch (error) {
                console.error('Failed to move task:', error);
            }
        },
        
        updateTaskStatus(taskId, newStatus) {
            const task = this.tasks.find(t => t.id == taskId);
            if (task) {
                task.status = newStatus;
                this.organizeTasks();
            }
        },
        
        // Utility Functions
        isOverlapping(rect1, rect2) {
            return !(rect1.right < rect2.left || 
                    rect1.left > rect2.right || 
                    rect1.bottom < rect2.top || 
                    rect1.top > rect2.bottom);
        },
        
        getDropColumn(element) {
            const columns = document.querySelectorAll('.column');
            const elementRect = element.getBoundingClientRect();
            
            for (const column of columns) {
                const columnRect = column.getBoundingClientRect();
                if (this.isOverlapping(elementRect, columnRect)) {
                    return column.dataset.status;
                }
            }
            return null;
        },
        
        animateTaskMove(taskId, newStatus) {
            const taskElement = document.querySelector(`[data-task-id="${taskId}"]`);
            const targetColumn = document.querySelector(`[data-status="${newStatus}"] .task-list`);
            
            if (taskElement && targetColumn && window.gsap) {
                window.gsap.to(taskElement, {
                    scale: 0.8,
                    opacity: 0.7,
                    duration: 0.2,
                    onComplete: () => {
                        targetColumn.appendChild(taskElement);
                        window.gsap.fromTo(taskElement, 
                            { scale: 0.8, opacity: 0.7 },
                            { scale: 1, opacity: 1, duration: 0.3 }
                        );
                    }
                });
            }
        },
        
        // Real-time Updates
        setupRealtime() {
            try {
                const eventSource = new EventSource(`/api/projects/${this.projectId}/events`);
                
                eventSource.onmessage = (event) => {
                    try {
                        const data = JSON.parse(event.data);
                        this.handleRealtimeUpdate(data);
                    } catch (error) {
                        console.error('Failed to parse realtime message:', error);
                    }
                };
                
                eventSource.onerror = (error) => {
                    console.error('EventSource failed:', error);
                };
                
                // Cleanup on component destroy
                this.$el.addEventListener('alpine:destroyed', () => {
                    eventSource.close();
                });
                
            } catch (error) {
                console.error('Failed to setup real-time updates:', error);
            }
        },
        
        handleRealtimeUpdate(data) {
            console.log('Realtime update received:', data.type);
            
            switch (data.type) {
                case 'task.created':
                    this.addTask(data.task);
                    break;
                case 'task.updated':
                    this.replaceTask(data.task);
                    break;
                case 'task.moved':
                    this.updateTaskStatus(data.task_id, data.new_status);
                    break;
                case 'task.deleted':
                    this.removeTask(data.task_id);
                    break;
            }
        },
        
        // Keyboard Shortcuts
        setupKeyboardShortcuts() {
            document.addEventListener('keydown', (e) => {
                // Only handle shortcuts when board is focused
                if (!this.$el.contains(document.activeElement)) return;
                
                if (e.ctrlKey || e.metaKey) {
                    switch (e.key) {
                        case 'n':
                            this.openCreateTaskModal();
                            e.preventDefault();
                            break;
                        case 'f':
                            this.focusSearchInput();
                            e.preventDefault();
                            break;
                    }
                }
                
                if (e.key === 'Escape') {
                    this.closeModals();
                    e.preventDefault();
                }
            });
        },
        
        openCreateTaskModal() {
            // Emit event for modal handling
            this.$dispatch('open-modal', { modal: 'create-task' });
        },
        
        focusSearchInput() {
            const searchInput = document.querySelector('#task-search');
            if (searchInput) {
                searchInput.focus();
            }
        },
        
        closeModals() {
            this.$dispatch('close-modals');
        },
        
        // Helper Functions
        getCSRFToken() {
            const meta = document.querySelector('meta[name="csrf-token"]');
            return meta ? meta.getAttribute('content') : '';
        },
        
        formatDate(dateString) {
            if (!dateString) return '';
            const date = new Date(dateString);
            return date.toLocaleDateString();
        },
        
        timeAgo(dateString) {
            if (!dateString) return '';
            const date = new Date(dateString);
            const now = new Date();
            const diffMs = now - date;
            const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
            const diffDays = Math.floor(diffHours / 24);
            
            if (diffHours < 24) {
                return `${diffHours}h ago`;
            } else {
                return `${diffDays}d ago`;
            }
        },
        
        priorityColor(priority) {
            const colors = {
                low: 'green',
                medium: 'yellow',
                high: 'orange',
                critical: 'red'
            };
            return colors[priority] || 'gray';
        },
        
        // Column Management
        getColumnTaskCount(columnKey) {
            return this.columns[columnKey]?.tasks?.length || 0;
        },
        
        getColumnColor(columnKey) {
            return this.columns[columnKey]?.color || 'gray';
        }
    }
}

// Make the component globally available
window.taskBoard = taskBoard;