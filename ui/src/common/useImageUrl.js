import { useEffect, useState, useRef } from 'react'

// Persists across component mount/unmount cycles so that
// React Admin refreshes (which remount list items) don't re-fetch images.
const cache = new Map()
const MAX_CACHE_SIZE = 300
const activeControllers = new Set()

/**
 * Aborts all in-flight image fetches. Call this before navigation/pagination
 * so that pending image requests don't block the browser connection pool.
 */
export const abortAllInFlight = () => {
  for (const controller of activeControllers) {
    controller.abort()
  }
  activeControllers.clear()
}

// Evicts oldest unused entries (Map iterates in insertion order).
const evictIfNeeded = () => {
  if (cache.size <= MAX_CACHE_SIZE) return
  for (const [key, entry] of cache) {
    if (cache.size <= MAX_CACHE_SIZE) break
    if (entry.refCount === 0) {
      if (entry.blobUrl) URL.revokeObjectURL(entry.blobUrl)
      cache.delete(key)
    }
  }
}

/**
 * Loads an image via fetch() with AbortController so that in-flight requests
 * are canceled on unmount (e.g., during pagination). Uses a module-level cache
 * so remounting returns the cached blob URL instantly.
 */
export const useImageUrl = (url) => {
  const cached = url ? cache.get(url) : null
  const [imgUrl, setImgUrl] = useState(cached?.blobUrl || null)
  const [loading, setLoading] = useState(!!url && !cached)
  const [error, setError] = useState(cached?.error || false)
  const abortedRef = useRef(false)

  useEffect(() => {
    abortedRef.current = false

    if (!url) {
      setImgUrl(null)
      setLoading(false)
      setError(false)
      return
    }

    // Re-check: another component's effect may have populated the cache
    // between this component's render and effect execution.
    const entry = cache.get(url)
    if (entry) {
      entry.refCount++
      setImgUrl(entry.blobUrl)
      setLoading(false)
      setError(entry.error || false)
      return () => {
        entry.refCount--
      }
    }

    const controller = new AbortController()
    activeControllers.add(controller)
    setImgUrl(null)
    setLoading(true)
    setError(false)

    fetch(url, { signal: controller.signal })
      .then((res) => {
        if (!res.ok) {
          throw new Error(`HTTP ${res.status}`)
        }
        return res.blob()
      })
      .then((blob) => {
        // Guard against late resolution after abort
        if (abortedRef.current) {
          return
        }
        const objectUrl = URL.createObjectURL(blob)
        // Handle concurrent fetches: if another component already cached
        // this URL, use its entry and discard our blob.
        const existing = cache.get(url)
        if (existing && existing.blobUrl) {
          existing.refCount++
          URL.revokeObjectURL(objectUrl)
          setImgUrl(existing.blobUrl)
        } else {
          cache.set(url, { blobUrl: objectUrl, refCount: 1 })
          evictIfNeeded()
          setImgUrl(objectUrl)
        }
        setLoading(false)
        activeControllers.delete(controller)
      })
      .catch((err) => {
        activeControllers.delete(controller)
        if (err.name === 'AbortError') {
          return // Expected on unmount or URL change
        }
        // Cache the error so repeated mounts don't re-fetch broken URLs
        cache.set(url, { blobUrl: null, error: true, refCount: 0 })
        setError(true)
        setLoading(false)
      })

    return () => {
      abortedRef.current = true
      controller.abort()
      activeControllers.delete(controller)
      const entry = cache.get(url)
      if (entry) {
        entry.refCount--
      }
    }
  }, [url])

  return { imgUrl, loading, error }
}
