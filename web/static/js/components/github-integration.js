/**
 * GitHub Integration UI Component
 * Handles OAuth flow, repository selection, and integration settings
 */
class GitHubIntegration {
    constructor(options = {}) {
        this.projectId = options.projectId;
        this.apiBaseUrl = options.apiBaseUrl || '/api/v1';
        this.currentIntegration = null;
        this.accessToken = null;
        this.repositories = [];
        
        this.init();
    }

    init() {
        this.bindEvents();
        this.checkExistingIntegration();
    }

    bindEvents() {
        // Connect button
        const connectBtn = document.getElementById('github-connect-btn');
        if (connectBtn) {
            connectBtn.addEventListener('click', () => this.initiateConnection());
        }

        // Modal events
        const cancelBtn = document.getElementById('cancel-repo-selection');
        if (cancelBtn) {
            cancelBtn.addEventListener('click', () => this.closeRepoModal());
        }

        // Repository search
        const searchInput = document.getElementById('repo-search');
        if (searchInput) {
            searchInput.addEventListener('input', (e) => this.filterRepositories(e.target.value));
        }

        // Settings form
        const settingsForm = document.getElementById('github-settings-form');
        if (settingsForm) {
            settingsForm.addEventListener('submit', (e) => this.saveSettings(e));
        }

        // Retry connection
        const retryBtn = document.getElementById('retry-connection');
        if (retryBtn) {
            retryBtn.addEventListener('click', () => this.retryConnection());
        }

        // Handle OAuth callback if present in URL
        this.handleOAuthCallback();
    }

    async checkExistingIntegration() {
        if (!this.projectId) return;

        try {
            const response = await this.apiCall(`/github/integrations/project/${this.projectId}`);
            if (response.ok) {
                const integration = await response.json();
                this.currentIntegration = integration;
                this.showConnectedState(integration);
            }
        } catch (error) {
            console.log('No existing integration found');
        }
    }

    async initiateConnection() {
        try {
            this.showLoading(true);
            
            const response = await this.apiCall('/github/auth', {
                method: 'POST',
                body: JSON.stringify({
                    project_id: this.projectId
                })
            });

            if (!response.ok) {
                throw new Error('Failed to initiate GitHub authentication');
            }

            const data = await response.json();
            
            // Store state for OAuth callback verification
            sessionStorage.setItem('github_oauth_state', data.state);
            sessionStorage.setItem('github_oauth_project_id', this.projectId || '');
            
            // Redirect to GitHub OAuth
            window.location.href = data.auth_url;
            
        } catch (error) {
            this.showError('Failed to connect to GitHub: ' + error.message);
            this.showLoading(false);
        }
    }

    handleOAuthCallback() {
        const urlParams = new URLSearchParams(window.location.search);
        const code = urlParams.get('code');
        const state = urlParams.get('state');
        const error = urlParams.get('error');

        if (error) {
            this.showError('GitHub authorization failed: ' + error);
            return;
        }

        if (code && state) {
            const storedState = sessionStorage.getItem('github_oauth_state');
            const storedProjectId = sessionStorage.getItem('github_oauth_project_id');
            
            if (state !== storedState) {
                this.showError('Invalid OAuth state. Please try again.');
                return;
            }

            this.handleCallback(code, state, storedProjectId);
            
            // Clean up session storage
            sessionStorage.removeItem('github_oauth_state');
            sessionStorage.removeItem('github_oauth_project_id');
            
            // Clean up URL
            const newUrl = window.location.origin + window.location.pathname;
            window.history.replaceState({}, document.title, newUrl);
        }
    }

    async handleCallback(code, state, projectId) {
        try {
            this.showLoading(true);
            
            const response = await this.apiCall('/github/callback?' + new URLSearchParams({
                code,
                state
            }));

            if (!response.ok) {
                throw new Error('OAuth callback failed');
            }

            const data = await response.json();
            this.accessToken = data.access_token;
            
            // Show repository selection
            await this.showRepositorySelection();
            
        } catch (error) {
            this.showError('Failed to complete GitHub authentication: ' + error.message);
        } finally {
            this.showLoading(false);
        }
    }

    async showRepositorySelection() {
        const modal = document.getElementById('repo-selection-modal');
        const loading = document.getElementById('repo-loading');
        const repoList = document.getElementById('repository-list');
        
        modal.classList.remove('hidden');
        loading.style.display = 'flex';
        
        try {
            await this.loadRepositories();
            this.renderRepositories();
            loading.style.display = 'none';
        } catch (error) {
            this.showError('Failed to load repositories: ' + error.message);
            this.closeRepoModal();
        }
    }

    async loadRepositories(page = 1) {
        const response = await this.apiCall('/github/repositories?' + new URLSearchParams({
            page: page.toString(),
            per_page: '50'
        }), {
            headers: {
                'X-GitHub-Token': this.accessToken
            }
        });

        if (!response.ok) {
            throw new Error('Failed to load repositories');
        }

        const data = await response.json();
        this.repositories = data.repositories || [];
    }

    renderRepositories(filter = '') {
        const repoList = document.getElementById('repository-list');
        const template = document.getElementById('repo-item-template');
        
        // Clear existing items except loading
        const existingItems = repoList.querySelectorAll('.repository-item');
        existingItems.forEach(item => item.remove());
        
        const filteredRepos = this.repositories.filter(repo => 
            repo.full_name.toLowerCase().includes(filter.toLowerCase()) ||
            (repo.description && repo.description.toLowerCase().includes(filter.toLowerCase()))
        );

        filteredRepos.forEach(repo => {
            const item = this.createRepositoryItem(repo, template);
            repoList.appendChild(item);
        });

        if (filteredRepos.length === 0) {
            const noResults = document.createElement('div');
            noResults.className = 'p-4 text-center text-gray-500';
            noResults.textContent = filter ? 'No repositories match your search.' : 'No repositories found.';
            repoList.appendChild(noResults);
        }
    }

    createRepositoryItem(repo, template) {
        const item = template.content.cloneNode(true);
        const container = item.querySelector('.repository-item');
        
        // Set repository data
        item.querySelector('.repo-name').textContent = repo.full_name;
        item.querySelector('.repo-description').textContent = repo.description || 'No description';
        
        // Visibility badge
        const visibilityBadge = item.querySelector('.repo-visibility');
        visibilityBadge.textContent = repo.private ? 'Private' : 'Public';
        visibilityBadge.classList.add(
            repo.private ? 'bg-red-100' : 'bg-green-100',
            repo.private ? 'text-red-800' : 'text-green-800'
        );
        
        // Language
        if (repo.language) {
            item.querySelector('.language-name').textContent = repo.language;
            const languageColor = this.getLanguageColor(repo.language);
            item.querySelector('.language-color').style.backgroundColor = languageColor;
        } else {
            item.querySelector('.repo-language').style.display = 'none';
        }
        
        // Stars
        item.querySelector('.star-count').textContent = repo.stargazers_count;
        
        // Updated time
        item.querySelector('.update-time').textContent = this.formatRelativeTime(repo.updated_at);
        
        // Select button
        const selectBtn = item.querySelector('.select-repo-btn');
        selectBtn.addEventListener('click', () => this.selectRepository(repo));
        
        // Clickable row
        container.addEventListener('click', (e) => {
            if (e.target !== selectBtn) {
                this.selectRepository(repo);
            }
        });
        
        return item;
    }

    async selectRepository(repo) {
        try {
            this.showLoading(true);
            this.closeRepoModal();
            
            const [owner, name] = repo.full_name.split('/');
            
            const response = await this.apiCall('/github/integrations', {
                method: 'POST',
                body: JSON.stringify({
                    access_token: this.accessToken,
                    project_id: this.projectId,
                    repo_owner: owner,
                    repo_name: name
                })
            });

            if (!response.ok) {
                throw new Error('Failed to create integration');
            }

            const integration = await response.json();
            this.currentIntegration = integration;
            this.showConnectedState(integration);
            
        } catch (error) {
            this.showError('Failed to connect repository: ' + error.message);
        } finally {
            this.showLoading(false);
        }
    }

    showConnectedState(integration) {
        const connectSection = document.querySelector('.github-connect-section');
        const connectionStatus = document.getElementById('connection-status');
        const settingsPanel = document.getElementById('github-settings');
        const connectBtn = document.getElementById('github-connect-btn');
        
        // Update connect button
        connectBtn.innerHTML = `
            <svg class="-ml-1 mr-2 h-4 w-4" fill="currentColor" viewBox="0 0 20 20">
                <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd"/>
            </svg>
            Connected to ${integration.repo_owner}/${integration.repo_name}
        `;
        connectBtn.classList.replace('bg-gray-900', 'bg-green-600');
        connectBtn.classList.replace('hover:bg-gray-800', 'hover:bg-green-700');
        connectBtn.disabled = true;
        
        // Show connection status
        connectionStatus.classList.remove('hidden');
        
        // Show settings panel
        settingsPanel.classList.remove('hidden');
        this.populateSettings(integration.settings);
    }

    populateSettings(settings) {
        document.getElementById('auto-link-commits').checked = settings.auto_link_commits;
        document.getElementById('auto-link-prs').checked = settings.auto_link_prs;
        document.getElementById('auto-create-branches').checked = settings.auto_create_branches;
        document.getElementById('sync-labels').checked = settings.sync_labels;
        
        // Webhook events
        const webhookEvents = settings.webhook_events || [];
        document.querySelectorAll('input[name="webhook_events"]').forEach(checkbox => {
            checkbox.checked = webhookEvents.includes(checkbox.value);
        });
    }

    async saveSettings(event) {
        event.preventDefault();
        
        if (!this.currentIntegration) {
            this.showError('No integration found to update');
            return;
        }
        
        try {
            const formData = new FormData(event.target);
            const settings = {
                auto_link_commits: formData.get('auto_link_commits') === 'on',
                auto_link_prs: formData.get('auto_link_prs') === 'on',
                auto_create_branches: formData.get('auto_create_branches') === 'on',
                sync_labels: formData.get('sync_labels') === 'on',
                webhook_events: formData.getAll('webhook_events')
            };
            
            const response = await this.apiCall(`/github/integrations/${this.currentIntegration.id}/settings`, {
                method: 'PUT',
                body: JSON.stringify({ settings })
            });

            if (!response.ok) {
                throw new Error('Failed to save settings');
            }

            this.showSuccess('Settings saved successfully');
            
        } catch (error) {
            this.showError('Failed to save settings: ' + error.message);
        }
    }

    filterRepositories(query) {
        this.renderRepositories(query);
    }

    closeRepoModal() {
        const modal = document.getElementById('repo-selection-modal');
        modal.classList.add('hidden');
    }

    retryConnection() {
        const errorPanel = document.getElementById('github-error');
        errorPanel.classList.add('hidden');
        this.initiateConnection();
    }

    // Utility methods

    async apiCall(endpoint, options = {}) {
        const url = this.apiBaseUrl + endpoint;
        const defaultOptions = {
            headers: {
                'Content-Type': 'application/json',
                ...options.headers
            }
        };
        
        return fetch(url, { ...defaultOptions, ...options });
    }

    showLoading(show) {
        // Implement loading state UI
        const connectBtn = document.getElementById('github-connect-btn');
        if (connectBtn) {
            connectBtn.disabled = show;
            if (show) {
                connectBtn.innerHTML = `
                    <div class="animate-spin rounded-full h-4 w-4 border-b-2 border-white"></div>
                    <span class="ml-2">Connecting...</span>
                `;
            }
        }
    }

    showError(message) {
        const errorPanel = document.getElementById('github-error');
        const errorMessage = document.getElementById('error-message');
        
        if (errorPanel && errorMessage) {
            errorMessage.textContent = message;
            errorPanel.classList.remove('hidden');
        } else {
            console.error('GitHub Integration Error:', message);
        }
    }

    showSuccess(message) {
        // Create a success toast
        const toast = document.createElement('div');
        toast.className = 'fixed top-4 right-4 bg-green-500 text-white px-6 py-3 rounded-md shadow-lg z-50';
        toast.textContent = message;
        
        document.body.appendChild(toast);
        
        setTimeout(() => {
            toast.remove();
        }, 3000);
    }

    getLanguageColor(language) {
        const colors = {
            'JavaScript': '#f1e05a',
            'TypeScript': '#2b7489',
            'Python': '#3572A5',
            'Java': '#b07219',
            'Go': '#00ADD8',
            'Rust': '#dea584',
            'C++': '#f34b7d',
            'C': '#555555',
            'PHP': '#4F5D95',
            'Ruby': '#701516',
            'Swift': '#ffac45',
            'Kotlin': '#F18E33',
            'C#': '#239120',
            'HTML': '#e34c26',
            'CSS': '#1572B6',
            'Shell': '#89e051',
            'Dockerfile': '#384d54'
        };
        return colors[language] || '#586069';
    }

    formatRelativeTime(dateString) {
        const now = new Date();
        const date = new Date(dateString);
        const diffInSeconds = (now - date) / 1000;
        
        if (diffInSeconds < 60) {
            return 'just now';
        } else if (diffInSeconds < 3600) {
            return `${Math.floor(diffInSeconds / 60)} minutes ago`;
        } else if (diffInSeconds < 86400) {
            return `${Math.floor(diffInSeconds / 3600)} hours ago`;
        } else if (diffInSeconds < 604800) {
            return `${Math.floor(diffInSeconds / 86400)} days ago`;
        } else {
            return date.toLocaleDateString();
        }
    }
}

// Auto-initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    // Check if GitHub integration components are present
    if (document.getElementById('github-connect-btn')) {
        // Get project ID from data attribute or global variable
        const projectId = document.body.dataset.projectId || window.currentProjectId;
        
        window.githubIntegration = new GitHubIntegration({
            projectId: projectId,
            apiBaseUrl: '/api/v1'
        });
    }
});

// Export for use in other modules
if (typeof module !== 'undefined' && module.exports) {
    module.exports = GitHubIntegration;
}