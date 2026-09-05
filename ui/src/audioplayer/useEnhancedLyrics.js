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
const COOLDOWN_PATTERN = /(?:cooldown active for|retry after)\s+(\d+)s/gi

const cache = new Map()
const inFlight = new Map()

const normalizeLyricLayers = (layers) => ({
  main: layers?.main || null,
  translation: layers?.translation || null,
  pronunciation: layers?.pronunciation || null,
})

const readStructuredLyrics = (response) =>
  response?.json?.['subsonic-response']?.lyricsList?.structuredLyrics || []

const retryDelayFromError = (error) => {
  const serverMessage =
    error?.body?.['subsonic-response']?.error?.message || error?.message || ''
  const matches = [...String(serverMessage).matchAll(COOLDOWN_PATTERN)]
  if (matches.length === 0) return null
  const seconds = Math.max(...matches.map((match) => Number(match[1])))
  return Number.isFinite(seconds) && seconds > 0 ? seconds * 1000 : null
}

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
    retryAt: null,
  }))
  const [requestGeneration, setRequestGeneration] = useState(0)
  const [, setCountdownTick] = useState(0)

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
        retryAt: null,
      })
      return undefined
    }

    const cached = readCachedLyrics(cacheKey)
    if (cached) {
      setState({
        cacheKey,
        layers: cached,
        loading: false,
        error: null,
        retryAt: null,
      })
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
      retryAt: null,
    })

    request.promise
      .then((layers) => {
        if (!active) return
        setState({
          cacheKey,
          layers,
          loading: false,
          error: null,
          retryAt: null,
        })
      })
      .catch((error) => {
        if (!active || error?.name === 'AbortError') return
        cache.delete(cacheKey)
        setState({
          cacheKey,
          layers: emptyLyricLayers,
          loading: false,
          error,
          retryAt: (() => {
            const delay = retryDelayFromError(error)
            return delay == null ? null : Date.now() + delay
          })(),
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

  useEffect(() => {
    if (
      state.cacheKey !== cacheKey ||
      state.retryAt == null ||
      disabled ||
      !requested
    ) {
      return undefined
    }

    const retryDelay = Math.max(0, state.retryAt - Date.now())
    const retryTimer = window.setTimeout(retry, retryDelay)
    const countdownTimer = window.setInterval(
      () => setCountdownTick((current) => current + 1),
      1_000,
    )
    return () => {
      window.clearTimeout(retryTimer)
      window.clearInterval(countdownTimer)
    }
  }, [cacheKey, disabled, requested, retry, state.cacheKey, state.retryAt])

  if (state.cacheKey !== cacheKey) {
    return {
      layers: emptyLyricLayers,
      loading: Boolean(trackId && !disabled && requested),
      error: null,
      retryAfterSeconds: null,
      retry,
    }
  }
  return {
    layers: state.layers,
    loading: state.loading,
    error: state.error,
    retryAfterSeconds:
      state.retryAt == null
        ? null
        : Math.max(1, Math.ceil((state.retryAt - Date.now()) / 1_000)),
    retry,
  }
}

export const clearEnhancedLyricsCache = () => {
  cache.clear()
  inFlight.forEach((entry) => entry.controller.abort())
  inFlight.clear()
}

export default useEnhancedLyrics
