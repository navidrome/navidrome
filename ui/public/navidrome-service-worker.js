import { clientsClaim } from 'workbox-core'
import { NetworkOnly } from 'workbox-strategies'
import { registerRoute, NavigationRoute } from 'workbox-routing'

clientsClaim()
self.skipWaiting()

const CACHE_NAME = 'offline-html'
// This assumes /offline.html is a URL for your self-contained
// (no external images or styles) offline page.
const FALLBACK_HTML_URL = './offline.html'
// Populate the cache with the offline HTML page when the
// service worker is installed.
self.addEventListener('install', async (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME).then((cache) => cache.add(FALLBACK_HTML_URL))
  )
})

const networkOnly = new NetworkOnly()
const navigationHandler = async (params) => {
  try {
    // Attempt a network request.
    return await networkOnly.handle(params)
  } catch (error) {
    // If it fails, return the cached HTML.
    return caches.match(FALLBACK_HTML_URL, {
      cacheName: CACHE_NAME,
    })
  }
}

// Register this strategy to handle all navigations.
registerRoute(new NavigationRoute(navigationHandler))
