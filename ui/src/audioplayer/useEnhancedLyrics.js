import { useCallback, useEffect, useMemo, useState } from 'react'
import subsonic from '../subsonic'
import { getPreferredLyricLanguage, selectLyricLayers } from './lyrics'

export const emptyLyricLayers = Object.freeze({
  main: null,
  translation: null,
  pronunciation: null,
})

const MAX_LYRIC_CACHE_ENTRIES = 75
const NEGATIVE_CACHE_TTL_MS = 30_000

const cache = new Map()
const inFlight = new Map()

const normalizeLyricLayers = (layers) => ({
  main: layers?.main || null,
  translation: layers?.translation || null,
  pronunciation: layers?.pronunciation || null,
})

const readStructuredLyrics = (response) =>
  response?.json?.['subsonic-response']?.lyricsList?.structuredLyrics || []

const buildCacheKey = ({ trackId, preferredLanguage, updatedAt }) =>
  [trackId || '', updatedAt || '', preferredLanguage || ''].join('\u0000')

const rememberLyrics = (cacheKey, layers, expiresAt = null) => {
  cache.delete(cacheKey)
  cache.set(cacheKey, { layers, expiresAt })
  while (cache.size > MAX_LYRIC_CACHE_ENTRIES) {
    const oldestCacheKey = cache.keys().next().value
    cache.delete(oldestCacheKey)
  }
}

const readCachedLyrics = (cacheKey) => {
  const cached = cache.get(cacheKey)
  if (!cached) return null
  if (cached.expiresAt != null && cached.expiresAt <= Date.now()) {
    cache.delete(cacheKey)
    return null
  }
  cache.delete(cacheKey)
  cache.set(cacheKey, cached)
  return cached.layers
}

const createLyricsRequest = ({ trackId, preferredLanguage, cacheKey }) => {
  const controller = new AbortController()
  const entry = {
    controller,
    consumers: 0,
    settled: false,
    promise: null,
  }
  entry.promise = subsonic
    .getLyricsBySongId(trackId, { signal: controller.signal })
    .then((response) => {
      const selected = normalizeLyricLayers(
        selectLyricLayers(readStructuredLyrics(response), preferredLanguage),
      )
      const hasAnyLayer = Boolean(
        selected.main || selected.translation || selected.pronunciation,
      )
      rememberLyrics(
        cacheKey,
        selected,
        hasAnyLayer ? null : Date.now() + NEGATIVE_CACHE_TTL_MS,
      )
      return selected
    })
    .finally(() => {
      entry.settled = true
      if (inFlight.get(cacheKey) === entry) inFlight.delete(cacheKey)
    })
  inFlight.set(cacheKey, entry)
  return entry
}

const acquireLyricsRequest = ({ trackId, preferredLanguage, cacheKey }) => {
  const existing = inFlight.get(cacheKey)
  const entry =
    existing && !existing.controller.signal.aborted
      ? existing
      : createLyricsRequest({ trackId, preferredLanguage, cacheKey })
  entry.consumers += 1
  let released = false
  return {
    promise: entry.promise,
    release: () => {
      if (released) return
      released = true
      entry.consumers = Math.max(0, entry.consumers - 1)
      if (!entry.settled && entry.consumers === 0) {
        if (inFlight.get(cacheKey) === entry) inFlight.delete(cacheKey)
        entry.controller.abort()
      }
    },
  }
}

const useEnhancedLyrics = ({
  trackId,
  updatedAt,
  disabled = false,
  requested = true,
}) => {
  const preferredLanguage = getPreferredLyricLanguage()
  const cacheKey = useMemo(
    () => buildCacheKey({ trackId, preferredLanguage, updatedAt }),
    [preferredLanguage, trackId, updatedAt],
  )
  const [state, setState] = useState(() => ({
    cacheKey,
    layers: emptyLyricLayers,
    loading: false,
    error: null,
  }))
  const [requestGeneration, setRequestGeneration] = useState(0)

  const retry = useCallback(() => {
    cache.delete(cacheKey)
    setRequestGeneration((current) => current + 1)
  }, [cacheKey])

  useEffect(() => {
    if (!trackId || disabled || !requested) {
      setState({
        cacheKey,
        layers: emptyLyricLayers,
        loading: false,
        error: null,
      })
      return undefined
    }

    const cached = readCachedLyrics(cacheKey)
    if (cached) {
      setState({ cacheKey, layers: cached, loading: false, error: null })
      return undefined
    }

    let active = true
    const request = acquireLyricsRequest({
      trackId,
      preferredLanguage,
      cacheKey,
    })
    setState({
      cacheKey,
      layers: emptyLyricLayers,
      loading: true,
      error: null,
    })

    request.promise
      .then((layers) => {
        if (!active) return
        setState({ cacheKey, layers, loading: false, error: null })
      })
      .catch((error) => {
        if (!active || error?.name === 'AbortError') return
        cache.delete(cacheKey)
        setState({
          cacheKey,
          layers: emptyLyricLayers,
          loading: false,
          error,
        })
      })

    return () => {
      active = false
      request.release()
    }
  }, [
    cacheKey,
    disabled,
    preferredLanguage,
    requestGeneration,
    requested,
    trackId,
  ])

  if (state.cacheKey !== cacheKey) {
    return {
      layers: emptyLyricLayers,
      loading: Boolean(trackId && !disabled && requested),
      error: null,
      retry,
    }
  }
  return {
    layers: state.layers,
    loading: state.loading,
    error: state.error,
    retry,
  }
}

export const clearEnhancedLyricsCache = () => {
  cache.clear()
  inFlight.forEach((entry) => entry.controller.abort())
  inFlight.clear()
}

export default useEnhancedLyrics
