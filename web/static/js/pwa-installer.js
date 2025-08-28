/**
 * PWA Installation Manager
 * Handles service worker registration, app installation prompts, and offline functionality
 */
class PWAInstaller {
    constructor() {
        this.deferredPrompt = null;
        this.isInstalled = false;
        this.isStandalone = false;
        
        this.init();
    }
    
    async init() {
        this.checkStandaloneMode();
        await this.registerServiceWorker();
        this.setupInstallPrompt();
        this.setupOfflineHandling();
        this.setupNotificationPermission();
        
        console.log('PWA Installer initialized');
    }
    
    // Check if app is running in standalone mode
    checkStandaloneMode() {
        this.isStandalone = window.matchMedia('(display-mode: standalone)').matches ||
                           window.navigator.standalone === true ||
                           document.referrer.includes('android-app://');
        
        if (this.isStandalone) {
            document.body.classList.add('pwa-standalone');
            console.log('App is running in standalone mode');
        }
    }
    
    // Register service worker
    async registerServiceWorker() {
        if ('serviceWorker' in navigator) {
            try {
                const registration = await navigator.serviceWorker.register('/static/sw.js', {
                    scope: '/'
                });
                
                console.log('Service Worker registered successfully:', registration);
                
                // Handle service worker updates
                registration.addEventListener('updatefound', () => {
                    console.log('New service worker available');
                    this.showUpdateAvailable();
                });
                
                // Listen for messages from service worker
                navigator.serviceWorker.addEventListener('message', this.handleServiceWorkerMessage.bind(this));
                
                return registration;
            } catch (error) {
                console.error('Service Worker registration failed:', error);
            }
        } else {
            console.log('Service Worker not supported');
        }
    }
    
    // Handle service worker messages
    handleServiceWorkerMessage(event) {
        const { type, data } = event.data;
        
        switch (type) {
            case 'offline-action-synced':
                this.showSyncNotification(data);
                break;
            case 'cache-updated':
                console.log('Cache updated:', data);
                break;
        }
    }
    
    // Setup app installation prompt
    setupInstallPrompt() {
        window.addEventListener('beforeinstallprompt', (e) => {
            console.log('Install prompt available');
            e.preventDefault();
            this.deferredPrompt = e;
            this.showInstallBanner();
        });
        
        window.addEventListener('appinstalled', () => {
            console.log('App was installed');
            this.isInstalled = true;
            this.hideInstallBanner();
            this.deferredPrompt = null;
            
            // Track installation
            this.trackInstallation();
        });
    }
    
    // Show install banner
    showInstallBanner() {
        // Check if banner should be shown
        if (this.isStandalone || localStorage.getItem('pwa-install-dismissed') === 'true') {
            return;
        }
        
        const banner = this.createInstallBanner();
        document.body.appendChild(banner);
    }
    
    // Create install banner element
    createInstallBanner() {
        const banner = document.createElement('div');
        banner.id = 'pwa-install-banner';
        banner.innerHTML = `
            <div class="fixed bottom-0 left-0 right-0 bg-blue-600 text-white p-4 shadow-lg z-50 md:bottom-4 md:left-4 md:right-auto md:max-w-sm md:rounded-lg" id="install-banner">
                <div class="flex items-center justify-between">
                    <div class="flex-1">
                        <div class="font-medium text-sm">Install Simple Easy Tasks</div>
                        <div class="text-xs opacity-90 mt-1">Get the full app experience with offline access</div>
                    </div>
                    <div class="ml-4 flex items-center space-x-2">
                        <button id="install-app-btn" class="bg-white text-blue-600 px-3 py-1 rounded text-sm font-medium hover:bg-blue-50">
                            Install
                        </button>
                        <button id="dismiss-install-btn" class="text-white/80 hover:text-white p-1">
                            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
                            </svg>
                        </button>
                    </div>
                </div>
            </div>
        `;
        
        // Add event listeners
        banner.querySelector('#install-app-btn').addEventListener('click', () => {
            this.promptInstall();
        });
        
        banner.querySelector('#dismiss-install-btn').addEventListener('click', () => {
            this.dismissInstallBanner();
        });
        
        return banner;
    }
    
    // Prompt app installation
    async promptInstall() {
        if (!this.deferredPrompt) {
            console.log('No install prompt available');
            return;
        }
        
        try {
            this.deferredPrompt.prompt();
            const { outcome } = await this.deferredPrompt.userChoice;
            
            console.log(`User ${outcome} the install prompt`);
            
            if (outcome === 'accepted') {
                this.trackInstallation();
            }
            
            this.deferredPrompt = null;
            this.hideInstallBanner();
        } catch (error) {
            console.error('Error prompting install:', error);
        }
    }
    
    // Dismiss install banner
    dismissInstallBanner() {
        localStorage.setItem('pwa-install-dismissed', 'true');
        this.hideInstallBanner();
    }
    
    // Hide install banner
    hideInstallBanner() {
        const banner = document.getElementById('pwa-install-banner');
        if (banner) {
            banner.remove();
        }
    }
    
    // Setup offline handling
    setupOfflineHandling() {
        // Monitor online/offline status
        window.addEventListener('online', this.handleOnline.bind(this));
        window.addEventListener('offline', this.handleOffline.bind(this));
        
        // Initial status check
        if (!navigator.onLine) {
            this.handleOffline();
        }
    }
    
    // Handle online event
    handleOnline() {
        console.log('App is online');
        this.hideOfflineIndicator();
        this.showOnlineNotification();
        
        // Trigger background sync if available
        if ('serviceWorker' in navigator && 'sync' in window.ServiceWorkerRegistration.prototype) {
            navigator.serviceWorker.ready.then(registration => {
                registration.sync.register('background-sync');
            });
        }
    }
    
    // Handle offline event
    handleOffline() {
        console.log('App is offline');
        this.showOfflineIndicator();
        this.cacheCurrentData();
    }
    
    // Show offline indicator
    showOfflineIndicator() {
        let indicator = document.getElementById('offline-indicator');
        
        if (!indicator) {
            indicator = document.createElement('div');
            indicator.id = 'offline-indicator';
            indicator.innerHTML = `
                <div class="fixed top-0 left-0 right-0 bg-amber-500 text-white px-4 py-2 text-sm font-medium text-center z-50">
                    <div class="flex items-center justify-center space-x-2">
                        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" 
                                  d="M18.364 5.636l-3.536 3.536m0 5.656l3.536 3.536M9.172 9.172L5.636 5.636m3.536 9.192L5.636 18.364"/>
                        </svg>
                        <span>You're offline - changes will be synced when connection returns</span>
                    </div>
                </div>
            `;
            document.body.appendChild(indicator);
            
            // Adjust body padding to account for indicator
            document.body.style.paddingTop = '48px';
        }
    }
    
    // Hide offline indicator
    hideOfflineIndicator() {
        const indicator = document.getElementById('offline-indicator');
        if (indicator) {
            indicator.remove();
            document.body.style.paddingTop = '';
        }
    }
    
    // Show online notification
    showOnlineNotification() {
        this.showNotification('✅ Back Online', 'Your changes are being synced', 'success');
    }
    
    // Cache current data for offline access
    cacheCurrentData() {
        try {
            // Cache current task board data
            if (window.taskBoard && window.taskBoard.tasks) {
                const cacheData = {
                    tasks: window.taskBoard.tasks,
                    columns: window.taskBoard.columns,
                    timestamp: Date.now()
                };
                
                localStorage.setItem('taskboard_cache', JSON.stringify(cacheData));
                console.log('Cached current task data for offline access');
            }
        } catch (error) {
            console.error('Error caching data:', error);
        }
    }
    
    // Setup notification permission
    async setupNotificationPermission() {
        if ('Notification' in window) {
            const permission = await this.requestNotificationPermission();
            console.log('Notification permission:', permission);
        }
    }
    
    // Request notification permission
    async requestNotificationPermission() {
        if (Notification.permission === 'default') {
            return await Notification.requestPermission();
        }
        return Notification.permission;
    }
    
    // Show sync notification
    showSyncNotification(data) {
        this.showNotification(
            'Changes Synced',
            `${data.action.method} ${data.action.url} completed successfully`,
            'success'
        );
    }
    
    // Show update available notification
    showUpdateAvailable() {
        const updateBanner = document.createElement('div');
        updateBanner.id = 'update-banner';
        updateBanner.innerHTML = `
            <div class="fixed top-0 left-0 right-0 bg-blue-600 text-white px-4 py-2 text-sm font-medium z-50">
                <div class="flex items-center justify-between">
                    <span>A new version is available</span>
                    <div class="space-x-2">
                        <button id="update-app-btn" class="bg-white text-blue-600 px-3 py-1 rounded text-xs font-medium">
                            Update
                        </button>
                        <button id="dismiss-update-btn" class="text-white/80 hover:text-white">
                            ✕
                        </button>
                    </div>
                </div>
            </div>
        `;
        
        updateBanner.querySelector('#update-app-btn').addEventListener('click', () => {
            window.location.reload();
        });
        
        updateBanner.querySelector('#dismiss-update-btn').addEventListener('click', () => {
            updateBanner.remove();
        });
        
        document.body.appendChild(updateBanner);
    }
    
    // Generic notification function
    showNotification(title, body, type = 'info') {
        const notification = document.createElement('div');
        notification.className = `fixed top-4 right-4 max-w-sm bg-white border-l-4 rounded-lg shadow-lg p-4 z-50 ${
            type === 'success' ? 'border-green-500' : 
            type === 'error' ? 'border-red-500' : 'border-blue-500'
        }`;
        
        notification.innerHTML = `
            <div class="flex items-start">
                <div class="flex-shrink-0">
                    ${type === 'success' ? '✅' : type === 'error' ? '❌' : 'ℹ️'}
                </div>
                <div class="ml-3 flex-1">
                    <div class="text-sm font-medium text-gray-900">${title}</div>
                    <div class="mt-1 text-sm text-gray-600">${body}</div>
                </div>
                <button class="ml-4 text-gray-400 hover:text-gray-600" onclick="this.parentElement.parentElement.remove()">
                    ✕
                </button>
            </div>
        `;
        
        document.body.appendChild(notification);
        
        // Auto-remove after 5 seconds
        setTimeout(() => {
            if (notification.parentNode) {
                notification.remove();
            }
        }, 5000);
    }
    
    // Track installation
    trackInstallation() {
        // Track installation event
        console.log('App installed - tracking event');
        
        // You can send analytics event here
        if (window.gtag) {
            window.gtag('event', 'pwa_install', {
                event_category: 'engagement',
                event_label: 'PWA Installation'
            });
        }
    }
    
    // Get installation state
    getInstallationState() {
        return {
            isInstalled: this.isInstalled,
            isStandalone: this.isStandalone,
            canInstall: !!this.deferredPrompt,
            isOnline: navigator.onLine
        };
    }
}

// Initialize PWA installer when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.pwaInstaller = new PWAInstaller();
});

// Make PWAInstaller available globally
window.PWAInstaller = PWAInstaller;