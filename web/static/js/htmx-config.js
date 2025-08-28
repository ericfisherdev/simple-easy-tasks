// HTMX Configuration for Simple Easy Tasks
document.addEventListener('DOMContentLoaded', function() {
    // Configure HTMX defaults
    htmx.config.defaultSwapStyle = 'outerHTML';
    htmx.config.defaultSwapDelay = 0;
    htmx.config.defaultSettleDelay = 20;
    htmx.config.useTemplateFragments = true;
    
    // Global request configuration
    htmx.on('htmx:configRequest', function(evt) {
        // Add CSRF token if available
        const csrfToken = document.querySelector('meta[name="csrf-token"]');
        if (csrfToken) {
            evt.detail.headers['X-CSRFToken'] = csrfToken.content;
        }
        
        // Set content type for API requests
        if (evt.detail.path.startsWith('/api/')) {
            evt.detail.headers['Content-Type'] = 'application/json';
        }
    });
    
    // Global loading states
    htmx.on('htmx:beforeRequest', function(evt) {
        showLoading(evt.target);
    });
    
    htmx.on('htmx:afterRequest', function(evt) {
        hideLoading(evt.target);
        
        // Handle errors
        if (evt.detail.failed) {
            showError(evt.target, 'Request failed. Please try again.');
        }
    });
    
    // Handle successful responses
    htmx.on('htmx:afterSettle', function(evt) {
        // Reinitialize any new interactive elements
        initializeNewElements(evt.target);
    });
    
    // Boost navigation for better UX
    htmx.on('htmx:pushedIntoHistory', function(evt) {
        // Update page title if available
        const title = document.querySelector('title');
        if (title && evt.detail.path !== '/') {
            title.textContent = `${title.textContent.split(' - ')[0]} - ${evt.detail.path}`;
        }
    });
});

// Loading state management
function showLoading(element) {
    element.classList.add('htmx-request');
    
    // Add loading indicator
    const loadingIndicator = document.createElement('div');
    loadingIndicator.className = 'htmx-loading-indicator';
    loadingIndicator.innerHTML = '<div class="spinner"></div>';
    element.appendChild(loadingIndicator);
}

function hideLoading(element) {
    element.classList.remove('htmx-request');
    
    // Remove loading indicator
    const loadingIndicator = element.querySelector('.htmx-loading-indicator');
    if (loadingIndicator) {
        loadingIndicator.remove();
    }
}

// Error handling
function showError(element, message) {
    const errorDiv = document.createElement('div');
    errorDiv.className = 'error-message bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-4';
    errorDiv.textContent = message;
    
    // Insert error message
    element.parentNode.insertBefore(errorDiv, element);
    
    // Auto-remove after 5 seconds
    setTimeout(() => {
        errorDiv.remove();
    }, 5000);
}

// Initialize new elements added via HTMX
function initializeNewElements(container) {
    // Initialize any drag-and-drop elements
    const draggableElements = container.querySelectorAll('.draggable');
    draggableElements.forEach(initializeDragAndDrop);
    
    // Initialize tooltips
    const tooltipElements = container.querySelectorAll('[data-tooltip]');
    tooltipElements.forEach(initializeTooltip);
}

// Drag and drop initialization
function initializeDragAndDrop(element) {
    element.draggable = true;
    
    element.addEventListener('dragstart', function(e) {
        e.dataTransfer.setData('text/plain', element.dataset.taskId);
        element.classList.add('dragging');
    });
    
    element.addEventListener('dragend', function() {
        element.classList.remove('dragging');
    });
}

// Tooltip initialization
function initializeTooltip(element) {
    const tooltip = document.createElement('div');
    tooltip.className = 'tooltip hidden absolute bg-gray-900 text-white text-xs rounded py-1 px-2 z-10';
    tooltip.textContent = element.dataset.tooltip;
    document.body.appendChild(tooltip);
    
    element.addEventListener('mouseenter', function(e) {
        const rect = e.target.getBoundingClientRect();
        tooltip.style.left = rect.left + 'px';
        tooltip.style.top = (rect.bottom + 5) + 'px';
        tooltip.classList.remove('hidden');
    });
    
    element.addEventListener('mouseleave', function() {
        tooltip.classList.add('hidden');
    });
}

// Form validation helpers
function validateForm(form) {
    const requiredFields = form.querySelectorAll('[required]');
    let isValid = true;
    
    requiredFields.forEach(field => {
        if (!field.value.trim()) {
            showFieldError(field, 'This field is required');
            isValid = false;
        } else {
            clearFieldError(field);
        }
    });
    
    return isValid;
}

function showFieldError(field, message) {
    clearFieldError(field);
    
    const errorDiv = document.createElement('div');
    errorDiv.className = 'field-error text-red-600 text-sm mt-1';
    errorDiv.textContent = message;
    
    field.parentNode.insertBefore(errorDiv, field.nextSibling);
    field.classList.add('border-red-500');
}

function clearFieldError(field) {
    const existingError = field.parentNode.querySelector('.field-error');
    if (existingError) {
        existingError.remove();
    }
    field.classList.remove('border-red-500');
}

// Debounce utility for search
function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}