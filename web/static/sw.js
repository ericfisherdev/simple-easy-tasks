/**
 * Service Worker for Simple Easy Tasks
 * Provides offline functionality, background sync, and push notifications
 */

const CACHE_NAME = 'simple-easy-tasks-v1';
const RUNTIME_CACHE = 'runtime-cache-v1';

// Files to cache on install
const urlsToCache = [
    '/',
    '/static/css/input.css',
    '/static/css/task-board.css',
    '/static/css/htmx-styles.css',
    '/static/js/animations.js',
    '/static/js/components/task-board.js',
    '/static/js/htmx-config.js',
    '/offline.html',
    // Add any other critical assets
];

// Install event - cache critical resources
self.addEventListener('install', event => {
    console.log('Service Worker: Install event triggered');
    
    event.waitUntil(
        caches.open(CACHE_NAME)
            .then(cache => {
                console.log('Service Worker: Caching files');
                return cache.addAll(urlsToCache);
            })
            .then(() => {
                console.log('Service Worker: Files cached successfully');
                return self.skipWaiting();
            })
            .catch(error => {
                console.error('Service Worker: Failed to cache files', error);
            })
    );
});

// Activate event - clean up old caches
self.addEventListener('activate', event => {
    console.log('Service Worker: Activate event triggered');
    
    event.waitUntil(
        caches.keys()
            .then(cacheNames => {
                return Promise.all(
                    cacheNames.map(cacheName => {
                        if (cacheName !== CACHE_NAME && cacheName !== RUNTIME_CACHE) {
                            console.log('Service Worker: Deleting old cache', cacheName);
                            return caches.delete(cacheName);
                        }
                    })
                );
            })
            .then(() => {
                console.log('Service Worker: Claiming clients');
                return self.clients.claim();
            })
    );
});

// Fetch event - serve from cache with network fallback
self.addEventListener('fetch', event => {
    const { request } = event;
    const url = new URL(request.url);
    
    // Skip non-http requests
    if (!request.url.startsWith('http')) {
        return;
    }
    
    // Handle different types of requests
    if (request.method === 'GET') {
        if (url.pathname.startsWith('/api/')) {
            // API requests - network first with cache fallback
            event.respondWith(networkFirstStrategy(request));
        } else if (url.pathname.startsWith('/static/')) {
            // Static assets - cache first with network fallback
            event.respondWith(cacheFirstStrategy(request));
        } else {
            // HTML pages - network first with offline fallback
            event.respondWith(networkFirstWithOfflineFallback(request));
        }
    } else {
        // POST, PUT, DELETE requests - handle offline actions
        event.respondWith(handleOfflineActions(request));
    }
});

// Network first strategy for API requests
async function networkFirstStrategy(request) {
    try {
        const networkResponse = await fetch(request);
        
        if (networkResponse.ok) {
            // Cache successful API responses for offline access
            const cache = await caches.open(RUNTIME_CACHE);
            cache.put(request, networkResponse.clone());
        }
        
        return networkResponse;
    } catch (error) {
        console.log('Service Worker: Network failed, trying cache', error);
        
        const cachedResponse = await caches.match(request);
        if (cachedResponse) {
            return cachedResponse;
        }
        
        // Return offline response for API requests
        return new Response(
            JSON.stringify({
                error: 'Offline',
                message: 'This request is not available offline'
            }),
            {
                status: 503,
                headers: { 'Content-Type': 'application/json' }
            }
        );
    }
}

// Cache first strategy for static assets
async function cacheFirstStrategy(request) {
    const cachedResponse = await caches.match(request);
    
    if (cachedResponse) {
        return cachedResponse;
    }
    
    try {
        const networkResponse = await fetch(request);
        
        if (networkResponse.ok) {
            const cache = await caches.open(CACHE_NAME);
            cache.put(request, networkResponse.clone());
        }
        
        return networkResponse;
    } catch (error) {
        console.log('Service Worker: Failed to fetch asset', error);
        throw error;
    }
}

// Network first with offline fallback for HTML pages
async function networkFirstWithOfflineFallback(request) {
    try {
        const networkResponse = await fetch(request);
        
        if (networkResponse.ok) {
            const cache = await caches.open(RUNTIME_CACHE);
            cache.put(request, networkResponse.clone());
        }
        
        return networkResponse;
    } catch (error) {
        console.log('Service Worker: Network failed for page, trying cache', error);
        
        const cachedResponse = await caches.match(request);
        if (cachedResponse) {
            return cachedResponse;
        }
        
        // Return offline page
        return caches.match('/offline.html');
    }
}

// Handle offline actions (POST, PUT, DELETE)
async function handleOfflineActions(request) {
    try {
        return await fetch(request);
    } catch (error) {
        console.log('Service Worker: Offline action detected', request.method, request.url);
        
        // Store the request for background sync
        if ('serviceWorker' in navigator && 'sync' in window.ServiceWorkerRegistration.prototype) {
            await storeOfflineAction(request);
            
            // Request a background sync
            await self.registration.sync.register('background-sync');
            
            return new Response(
                JSON.stringify({
                    success: true,
                    message: 'Action queued for when you\'re back online',
                    queued: true
                }),
                {
                    status: 200,
                    headers: { 'Content-Type': 'application/json' }
                }
            );
        } else {
            return new Response(
                JSON.stringify({
                    error: 'Offline',
                    message: 'This action cannot be completed offline'
                }),
                {
                    status: 503,
                    headers: { 'Content-Type': 'application/json' }
                }
            );
        }
    }
}

// Store offline actions in IndexedDB
async function storeOfflineAction(request) {
    const action = {
        id: Date.now(),
        url: request.url,
        method: request.method,
        headers: Object.fromEntries(request.headers.entries()),
        body: request.method !== 'GET' ? await request.text() : null,
        timestamp: Date.now()
    };
    
    try {
        const db = await openOfflineDB();
        const transaction = db.transaction(['offline_actions'], 'readwrite');
        const store = transaction.objectStore('offline_actions');
        await store.add(action);
        console.log('Service Worker: Stored offline action', action);
    } catch (error) {
        console.error('Service Worker: Failed to store offline action', error);
    }
}

// Open IndexedDB for offline actions
function openOfflineDB() {
    return new Promise((resolve, reject) => {
        const request = indexedDB.open('OfflineActions', 1);
        
        request.onerror = () => reject(request.error);
        request.onsuccess = () => resolve(request.result);
        
        request.onupgradeneeded = event => {
            const db = event.target.result;
            const store = db.createObjectStore('offline_actions', { keyPath: 'id' });
            store.createIndex('timestamp', 'timestamp', { unique: false });
        };
    });
}

// Background sync event
self.addEventListener('sync', event => {
    console.log('Service Worker: Background sync triggered', event.tag);
    
    if (event.tag === 'background-sync') {
        event.waitUntil(syncOfflineActions());
    }
});

// Sync offline actions when back online
async function syncOfflineActions() {
    console.log('Service Worker: Syncing offline actions');
    
    try {
        const db = await openOfflineDB();
        const transaction = db.transaction(['offline_actions'], 'readonly');
        const store = transaction.objectStore('offline_actions');
        const actions = await getAllFromStore(store);
        
        console.log(`Service Worker: Found ${actions.length} offline actions to sync`);
        
        for (const action of actions) {
            try {
                const response = await fetch(action.url, {
                    method: action.method,
                    headers: action.headers,
                    body: action.body
                });
                
                if (response.ok) {
                    // Remove successful action
                    await removeOfflineAction(action.id);
                    console.log('Service Worker: Synced offline action', action.id);
                    
                    // Notify the client about successful sync
                    await notifyClient('offline-action-synced', {
                        action: action,
                        success: true
                    });
                } else {
                    console.log('Service Worker: Failed to sync action', action.id, response.status);
                }
            } catch (error) {
                console.error('Service Worker: Error syncing action', action.id, error);
            }
        }
    } catch (error) {
        console.error('Service Worker: Error during background sync', error);
    }
}

// Helper function to get all items from IndexedDB store
function getAllFromStore(store) {
    return new Promise((resolve, reject) => {
        const request = store.getAll();
        request.onerror = () => reject(request.error);
        request.onsuccess = () => resolve(request.result);
    });
}

// Remove offline action from IndexedDB
async function removeOfflineAction(id) {
    try {
        const db = await openOfflineDB();
        const transaction = db.transaction(['offline_actions'], 'readwrite');
        const store = transaction.objectStore('offline_actions');
        await store.delete(id);
    } catch (error) {
        console.error('Service Worker: Failed to remove offline action', error);
    }
}

// Notify client about events
async function notifyClient(type, data) {
    const clients = await self.clients.matchAll();
    
    clients.forEach(client => {
        client.postMessage({
            type: type,
            data: data
        });
    });
}

// Push notification event
self.addEventListener('push', event => {
    console.log('Service Worker: Push notification received');
    
    let notificationData = {
        title: 'Simple Easy Tasks',
        body: 'You have new updates!',
        icon: '/static/icons/icon-192x192.png',
        badge: '/static/icons/badge-72x72.png',
        tag: 'default'
    };
    
    if (event.data) {
        try {
            const data = event.data.json();
            notificationData = { ...notificationData, ...data };
        } catch (error) {
            console.error('Service Worker: Error parsing push data', error);
        }
    }
    
    event.waitUntil(
        self.registration.showNotification(notificationData.title, {
            body: notificationData.body,
            icon: notificationData.icon,
            badge: notificationData.badge,
            tag: notificationData.tag,
            data: notificationData.data || {},
            actions: [
                {
                    action: 'open',
                    title: 'View'
                },
                {
                    action: 'close',
                    title: 'Dismiss'
                }
            ]
        })
    );
});

// Notification click event
self.addEventListener('notificationclick', event => {
    console.log('Service Worker: Notification clicked', event.notification.tag);
    
    event.notification.close();
    
    if (event.action === 'open' || event.action === '') {
        event.waitUntil(
            clients.matchAll().then(clientList => {
                // Try to find an existing window to focus
                for (const client of clientList) {
                    if (client.url === '/' && 'focus' in client) {
                        return client.focus();
                    }
                }
                
                // Open a new window
                if (clients.openWindow) {
                    return clients.openWindow('/');
                }
            })
        );
    }
});

console.log('Service Worker: Loaded successfully');