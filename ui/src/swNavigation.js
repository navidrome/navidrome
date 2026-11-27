// Lives outside sw.js so it can be unit tested: sw.js only loads inside a
// service worker, where workbox arrives via importScripts.
export const createNavigationHandler =
  (networkOnly, offlineFallback) => async (params) => {
    try {
      // Attempt a network request.
      const response = await networkOnly.handle(params)
      // A 5xx reaches us as a normal response, but carries no usable app
      return response.status >= 500 ? offlineFallback() : response
    } catch (error) {
      // If it fails, return the cached HTML.
      return offlineFallback()
    }
  }
