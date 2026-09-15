import { useEffect, useState, useRef } from 'react'

// Persists across component mount/unmount cycles so that
// React Admin refreshes (which remount list items) don't re-fetch images.
const cache = new Map()
const MAX_CACHE_SIZE = 300

// Limit concurrent fetches to leave browser connections free for API requests.
// Browsers allow ~6 connections per origin on HTTP/1.1; reserving 2 for API
// calls prevents image fetches from blocking pagination/data requests.
const MAX_CONCURRENT = 4
let activeFetches = 0
const pendingQueue = []

const processQueue = () => {
  while (pendingQueue.length > 0 && activeFetches < MAX_CONCURRENT) {
    const next = pendingQueue.shift()
    next()
  }
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
 * so remounting returns the cached blob URL instantly. A no-store response is a stand-in: it is
 * re-fetched on remount and on a new version, and painted until its replacement arrives.
 */
export const useImageUrl = (url, version) => {
  const cached = url ? cache.get(url) : null
  const [imgUrl, setImgUrl] = useState(cached?.blobUrl || null)
  const [loading, setLoading] = useState(!!url && !cached)
  const [error, setError] = useState(cached?.error || false)
  const abortedRef = useRef(false)

  // Resync during render: otherwise imgUrl holds the previous url's blob for one render, so only
  // fromCache can answer "already cached" - and a remembered failure has no blob to paint.
  const [fromCache, setFromCache] = useState(!!cached?.blobUrl)
  const [trackedUrl, setTrackedUrl] = useState(url)
  if (trackedUrl !== url) {
    setTrackedUrl(url)
    setImgUrl(cached?.blobUrl || null)
    setLoading(!!url && !cached)
    setError(cached?.error || false)
    setFromCache(!!cached?.blobUrl)
  }

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
    if (entry && !entry.transient) {
      entry.refCount++
      setImgUrl(entry.blobUrl)
      setLoading(false)
      setError(entry.error || false)
      return () => {
        entry.refCount--
      }
    }
    const standIn = !!entry

    const controller = new AbortController()
    let queued = true
    if (!standIn) {
      setImgUrl(null)
      setLoading(true)
      setError(false)
    }

    const doFetch = () => {
      queued = false
      activeFetches++
      fetch(url, { signal: controller.signal })
        .then((res) => {
          if (!res.ok) {
            throw new Error(`HTTP ${res.status}`)
          }
          const transient = !!res.headers
            ?.get('Cache-Control')
            ?.includes('no-store')
          return res.blob().then((blob) => ({ blob, transient }))
        })
        .then(({ blob, transient }) => {
          activeFetches--
          processQueue()
          // Guard against late resolution after abort
          if (abortedRef.current) {
            return
          }
          const objectUrl = URL.createObjectURL(blob)
          // Handle concurrent fetches: if another component already cached
          // this URL, use its entry and discard our blob.
          const existing = cache.get(url)
          if (existing?.blobUrl && !existing.transient) {
            existing.refCount++
            URL.revokeObjectURL(objectUrl)
            setImgUrl(existing.blobUrl)
          } else {
            if (existing?.blobUrl && existing.refCount <= 0) {
              URL.revokeObjectURL(existing.blobUrl)
            }
            cache.set(url, { blobUrl: objectUrl, refCount: 1, transient })
            evictIfNeeded()
            setImgUrl(objectUrl)
          }
          setLoading(false)
        })
        .catch((err) => {
          activeFetches--
          processQueue()
          if (err.name === 'AbortError') {
            return // Expected on unmount or URL change
          }
          if (standIn) {
            return // Keep painting the stand-in
          }
          // Cache the error so repeated mounts don't re-fetch broken URLs
          cache.set(url, { blobUrl: null, error: true, refCount: 0 })
          setError(true)
          setLoading(false)
        })
    }

    if (activeFetches < MAX_CONCURRENT) {
      queued = false
      doFetch()
    } else {
      pendingQueue.push(doFetch)
    }

    return () => {
      abortedRef.current = true
      if (queued) {
        // Remove from queue if not yet started
        const idx = pendingQueue.indexOf(doFetch)
        if (idx !== -1) pendingQueue.splice(idx, 1)
      } else {
        controller.abort()
      }
      const entry = cache.get(url)
      if (entry) {
        entry.refCount--
      }
    }
  }, [url, version])

  return { imgUrl, loading, error, fromCache }
}
