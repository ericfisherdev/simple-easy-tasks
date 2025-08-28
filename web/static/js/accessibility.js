/**
 * Accessibility Enhancement Module
 * Provides comprehensive keyboard navigation, screen reader support, and accessibility features
 */
class AccessibilityManager {
    constructor() {
        this.focusedElement = null;
        this.focusHistory = [];
        this.announcements = [];
        
        this.init();
    }
    
    init() {
        this.setupKeyboardNavigation();
        this.setupAriaLiveRegions();
        this.setupFocusManagement();
        this.setupColorContrastDetection();
        this.setupReducedMotionSupport();
        
        console.log('Accessibility Manager initialized');
    }
    
    // Setup keyboard navigation
    setupKeyboardNavigation() {
        // Global keyboard shortcuts
        document.addEventListener('keydown', (e) => {
            this.handleGlobalKeydown(e);
        });
        
        // Focus management
        document.addEventListener('focusin', (e) => {
            this.handleFocusIn(e);
        });
        
        document.addEventListener('focusout', (e) => {
            this.handleFocusOut(e);
        });
    }
    
    // Handle global keyboard events
    handleGlobalKeydown(e) {
        const { key, ctrlKey, metaKey, altKey, shiftKey } = e;
        
        // Skip if user is typing in input field
        if (['INPUT', 'TEXTAREA', 'SELECT'].includes(e.target.tagName)) {
            return;
        }
        
        // Global shortcuts
        const modifierKey = ctrlKey || metaKey;
        
        if (modifierKey) {
            switch (key) {
                case '/':
                case 'k':
                    this.focusSearch();
                    e.preventDefault();
                    break;
                case 'n':
                    this.openNewTaskModal();
                    e.preventDefault();
                    break;
                case 'h':
                    this.showKeyboardShortcuts();
                    e.preventDefault();
                    break;
            }
        }
        
        // Navigation shortcuts without modifier
        switch (key) {
            case 'Escape':
                this.handleEscape();
                e.preventDefault();
                break;
            case '?':
                if (shiftKey) {
                    this.showKeyboardShortcuts();
                    e.preventDefault();
                }
                break;
        }
    }
    
    // Board keyboard navigation
    handleBoardKeydown(e) {
        const { key, shiftKey } = e;
        
        switch (key) {
            case 'ArrowRight':
                this.navigateColumns('next');
                e.preventDefault();
                break;
            case 'ArrowLeft':
                this.navigateColumns('previous');
                e.preventDefault();
                break;
            case 'ArrowDown':
                this.navigateToFirstTask();
                e.preventDefault();
                break;
            case 'Tab':
                // Allow normal tab behavior
                break;
            case 'Enter':
            case ' ':
                this.openNewTaskModal();
                e.preventDefault();
                break;
        }
    }
    
    // Column keyboard navigation
    handleColumnKeydown(e, status) {
        const { key, shiftKey, ctrlKey, metaKey } = e;
        
        switch (key) {
            case 'ArrowRight':
                this.navigateColumns('next', status);
                e.preventDefault();
                break;
            case 'ArrowLeft':
                this.navigateColumns('previous', status);
                e.preventDefault();
                break;
            case 'ArrowDown':
                this.navigateToColumnTasks(status, 'first');
                e.preventDefault();
                break;
            case 'ArrowUp':
                this.navigateToBoard();
                e.preventDefault();
                break;
            case 'Enter':
            case ' ':
                this.openNewTaskModalForColumn(status);
                e.preventDefault();
                break;
            case 'Home':
                this.focusFirstColumn();
                e.preventDefault();
                break;
            case 'End':
                this.focusLastColumn();
                e.preventDefault();
                break;
        }
    }
    
    // Task keyboard navigation
    handleTaskKeydown(e, task) {
        const { key, shiftKey, ctrlKey, metaKey } = e;
        
        switch (key) {
            case 'ArrowDown':
                this.navigateToNextTask(task);
                e.preventDefault();
                break;
            case 'ArrowUp':
                this.navigateToPreviousTask(task);
                e.preventDefault();
                break;
            case 'ArrowRight':
                if (ctrlKey || metaKey) {
                    this.moveTaskToNextColumn(task);
                } else {
                    this.navigateColumns('next');
                }
                e.preventDefault();
                break;
            case 'ArrowLeft':
                if (ctrlKey || metaKey) {
                    this.moveTaskToPreviousColumn(task);
                } else {
                    this.navigateColumns('previous');
                }
                e.preventDefault();
                break;
            case 'Enter':
            case ' ':
                this.openTaskDetails(task);
                e.preventDefault();
                break;
            case 'Delete':
            case 'Backspace':
                if (ctrlKey || metaKey) {
                    this.deleteTask(task);
                    e.preventDefault();
                }
                break;
            case 'e':
                if (ctrlKey || metaKey) {
                    this.editTask(task);
                    e.preventDefault();
                }
                break;
            case 'd':
                if (ctrlKey || metaKey) {
                    this.duplicateTask(task);
                    e.preventDefault();
                }
                break;
        }
    }
    
    // Navigation methods
    navigateColumns(direction, currentStatus = null) {
        const columns = document.querySelectorAll('.column');
        const columnArray = Array.from(columns);
        
        let currentIndex = 0;
        if (currentStatus) {
            currentIndex = columnArray.findIndex(col => col.dataset.status === currentStatus);
        }
        
        let nextIndex;
        if (direction === 'next') {
            nextIndex = (currentIndex + 1) % columnArray.length;
        } else {
            nextIndex = currentIndex === 0 ? columnArray.length - 1 : currentIndex - 1;
        }
        
        const nextColumn = columnArray[nextIndex];
        if (nextColumn) {
            this.focusElement(nextColumn);
            this.announceColumnFocus(nextColumn);
        }
    }
    
    navigateToColumnTasks(status, position = 'first') {
        const column = document.querySelector(`[data-status="${status}"]`);
        const tasks = column?.querySelectorAll('.task-card');
        
        if (tasks && tasks.length > 0) {
            const taskToFocus = position === 'first' ? tasks[0] : tasks[tasks.length - 1];
            this.focusElement(taskToFocus);
            this.announceTaskFocus(taskToFocus);
        } else {
            this.announce('No tasks in this column');
        }
    }
    
    navigateToNextTask(currentTask) {
        const currentColumn = currentTask.closest('.column');
        const tasks = Array.from(currentColumn.querySelectorAll('.task-card'));
        const currentIndex = tasks.findIndex(task => task === currentTask);
        
        if (currentIndex < tasks.length - 1) {
            const nextTask = tasks[currentIndex + 1];
            this.focusElement(nextTask);
            this.announceTaskFocus(nextTask);
        } else {
            // Move to next column's first task
            const nextColumn = this.getNextColumn(currentColumn);
            if (nextColumn) {
                this.navigateToColumnTasks(nextColumn.dataset.status, 'first');
            }
        }
    }
    
    navigateToPreviousTask(currentTask) {
        const currentColumn = currentTask.closest('.column');
        const tasks = Array.from(currentColumn.querySelectorAll('.task-card'));
        const currentIndex = tasks.findIndex(task => task === currentTask);
        
        if (currentIndex > 0) {
            const prevTask = tasks[currentIndex - 1];
            this.focusElement(prevTask);
            this.announceTaskFocus(prevTask);
        } else {
            // Move to column header
            this.focusElement(currentColumn);
            this.announceColumnFocus(currentColumn);
        }
    }
    
    // Focus management
    focusElement(element) {
        if (element && typeof element.focus === 'function') {
            element.focus();
            this.focusedElement = element;
            this.addToFocusHistory(element);
        }
    }
    
    addToFocusHistory(element) {
        this.focusHistory.push(element);
        if (this.focusHistory.length > 10) {
            this.focusHistory.shift();
        }
    }
    
    returnToPreviousFocus() {
        if (this.focusHistory.length > 1) {
            this.focusHistory.pop(); // Remove current
            const previous = this.focusHistory.pop();
            if (previous && document.contains(previous)) {
                this.focusElement(previous);
            }
        }
    }
    
    // Setup ARIA live regions
    setupAriaLiveRegions() {
        // Create announcement region
        const announcementRegion = document.createElement('div');
        announcementRegion.id = 'aria-announcements';
        announcementRegion.setAttribute('aria-live', 'polite');
        announcementRegion.setAttribute('aria-atomic', 'true');
        announcementRegion.className = 'sr-only';
        document.body.appendChild(announcementRegion);
        
        // Create status region
        const statusRegion = document.createElement('div');
        statusRegion.id = 'aria-status';
        statusRegion.setAttribute('aria-live', 'assertive');
        statusRegion.setAttribute('aria-atomic', 'true');
        statusRegion.className = 'sr-only';
        document.body.appendChild(statusRegion);
    }
    
    // Announce messages to screen readers
    announce(message, priority = 'polite') {
        const regionId = priority === 'assertive' ? 'aria-status' : 'aria-announcements';
        const region = document.getElementById(regionId);
        
        if (region) {
            // Clear previous announcement
            region.textContent = '';
            
            // Add new announcement after a brief delay
            setTimeout(() => {
                region.textContent = message;
            }, 100);
            
            // Clear announcement after 5 seconds
            setTimeout(() => {
                if (region.textContent === message) {
                    region.textContent = '';
                }
            }, 5000);
        }
        
        console.log(`Accessibility announcement (${priority}): ${message}`);
    }
    
    // Specific announcement methods
    announceColumnFocus(column) {
        const status = column.dataset.status;
        const tasks = column.querySelectorAll('.task-card');
        const count = tasks.length;
        
        this.announce(`${status} column focused. ${count} task${count !== 1 ? 's' : ''}.`);
    }
    
    announceTaskFocus(taskElement) {
        // Get task data from Alpine.js or data attributes
        const taskId = taskElement.dataset.taskId;
        const title = taskElement.querySelector('h4')?.textContent || 'Unknown task';
        
        this.announce(`Task focused: ${title}`);
    }
    
    announceTaskMoved(task, fromStatus, toStatus) {
        this.announce(`Task "${task.title}" moved from ${fromStatus} to ${toStatus}`, 'assertive');
    }
    
    announceTaskCreated(task) {
        this.announce(`New task created: ${task.title}`, 'assertive');
    }
    
    announceTaskDeleted(taskTitle) {
        this.announce(`Task deleted: ${taskTitle}`, 'assertive');
    }
    
    // Action handlers
    focusSearch() {
        const searchInput = document.getElementById('task-search');
        if (searchInput) {
            this.focusElement(searchInput);
            this.announce('Search field focused');
        }
    }
    
    handleEscape() {
        // Close modals, dropdowns, or return focus
        const modal = document.querySelector('[role="dialog"]:not([hidden])');
        if (modal) {
            this.closeModal(modal);
        } else {
            this.returnToPreviousFocus();
        }
    }
    
    showKeyboardShortcuts() {
        const shortcuts = [
            'Ctrl/Cmd + K: Search tasks',
            'Ctrl/Cmd + N: New task',
            'Ctrl/Cmd + H: Show this help',
            'Escape: Close modals or return focus',
            'Arrow keys: Navigate board',
            'Enter/Space: Activate focused element',
            'Ctrl/Cmd + Arrow: Move tasks between columns',
            '?: Show keyboard shortcuts'
        ];
        
        this.announce('Keyboard shortcuts: ' + shortcuts.join(', '));
    }
    
    // Setup focus management
    setupFocusManagement() {
        // Ensure proper focus indicators
        document.addEventListener('focusin', (e) => {
            e.target.classList.add('accessibility-focused');
        });
        
        document.addEventListener('focusout', (e) => {
            e.target.classList.remove('accessibility-focused');
        });
        
        // Handle focus trapping in modals
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Tab') {
                const modal = document.querySelector('[role="dialog"]:not([hidden])');
                if (modal) {
                    this.trapFocusInModal(e, modal);
                }
            }
        });
    }
    
    trapFocusInModal(e, modal) {
        const focusableElements = modal.querySelectorAll(
            'a[href], button, textarea, input, select, [tabindex]:not([tabindex="-1"])'
        );
        
        const firstElement = focusableElements[0];
        const lastElement = focusableElements[focusableElements.length - 1];
        
        if (e.shiftKey && document.activeElement === firstElement) {
            e.preventDefault();
            lastElement.focus();
        } else if (!e.shiftKey && document.activeElement === lastElement) {
            e.preventDefault();
            firstElement.focus();
        }
    }
    
    // Color contrast detection
    setupColorContrastDetection() {
        // Check for high contrast preference
        if (window.matchMedia('(prefers-contrast: high)').matches) {
            document.body.classList.add('high-contrast');
            this.announce('High contrast mode detected');
        }
    }
    
    // Reduced motion support
    setupReducedMotionSupport() {
        if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
            document.body.classList.add('reduced-motion');
            this.announce('Reduced motion mode detected');
        }
    }
    
    // Handle focus events
    handleFocusIn(e) {
        this.focusedElement = e.target;
    }
    
    handleFocusOut(e) {
        // Optional: Handle focus out events
    }
    
    // Utility methods
    getNextColumn(currentColumn) {
        const columns = Array.from(document.querySelectorAll('.column'));
        const currentIndex = columns.indexOf(currentColumn);
        return columns[currentIndex + 1] || columns[0];
    }
    
    getPreviousColumn(currentColumn) {
        const columns = Array.from(document.querySelectorAll('.column'));
        const currentIndex = columns.indexOf(currentColumn);
        return columns[currentIndex - 1] || columns[columns.length - 1];
    }
}

// Make accessibility manager available globally
window.AccessibilityManager = AccessibilityManager;

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.accessibilityManager = new AccessibilityManager();
});

// Add keyboard navigation methods to taskBoard component
document.addEventListener('alpine:init', () => {
    Alpine.data('taskBoard', () => ({
        // ... existing taskBoard methods ...
        
        // Accessibility keyboard handlers
        handleBoardKeydown(e) {
            if (window.accessibilityManager) {
                window.accessibilityManager.handleBoardKeydown(e);
            }
        },
        
        handleColumnKeydown(e, status) {
            if (window.accessibilityManager) {
                window.accessibilityManager.handleColumnKeydown(e, status);
            }
        },
        
        handleTaskKeydown(e, task) {
            if (window.accessibilityManager) {
                window.accessibilityManager.handleTaskKeydown(e, task);
            }
        },
        
        // Enhanced task operations with announcements
        async createTaskWithAnnouncement(formData) {
            const task = await this.createTask(formData);
            if (window.accessibilityManager) {
                window.accessibilityManager.announceTaskCreated(task);
            }
            return task;
        },
        
        async deleteTaskWithAnnouncement(taskId) {
            const task = this.tasks.find(t => t.id === taskId);
            const taskTitle = task?.title || 'Unknown task';
            
            await this.deleteTask(taskId);
            
            if (window.accessibilityManager) {
                window.accessibilityManager.announceTaskDeleted(taskTitle);
            }
        },
        
        async moveTaskWithAnnouncement(taskId, newStatus) {
            const task = this.tasks.find(t => t.id === taskId);
            const oldStatus = task?.status;
            
            await this.moveTask(taskId, newStatus);
            
            if (window.accessibilityManager && task) {
                window.accessibilityManager.announceTaskMoved(task, oldStatus, newStatus);
            }
        }
    }));
});