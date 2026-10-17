import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import subsonic from '../subsonic'
import {
  collectLyricVariants,
  getPreferredLyricLanguage,
  hasStructuredLyricContent,
  selectLyricLayers,
} from './lyrics'

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

const normalizeStructuredLyrics = (lyrics) =>
  Array.isArray(lyrics) ? lyrics : []

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

const rememberLyrics = (cacheKey, structuredLyrics, expiresAt = null) => {
  cache.delete(cacheKey)
  cache.set(cacheKey, { structuredLyrics, expiresAt })
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
  return cached.structuredLyrics
}

const createLyricsRequest = ({ trackId, cacheKey }) => {
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
      const structuredLyrics = normalizeStructuredLyrics(
        readStructuredLyrics(response),
      )
      const hasAnyLyric = structuredLyrics.some(hasStructuredLyricContent)
      rememberLyrics(
        cacheKey,
        structuredLyrics,
        hasAnyLyric ? null : Date.now() + NEGATIVE_CACHE_TTL_MS,
      )
      return structuredLyrics
    })
    .finally(() => {
      entry.settled = true
      if (inFlight.get(cacheKey) === entry) inFlight.delete(cacheKey)
    })
  inFlight.set(cacheKey, entry)
  return entry
}

const acquireLyricsRequest = ({ trackId, cacheKey }) => {
  const existing = inFlight.get(cacheKey)
  const entry =
    existing && !existing.controller.signal.aborted
      ? existing
      : createLyricsRequest({ trackId, cacheKey })
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

const emptyLyricsState = (cacheKey) => ({
  cacheKey,
  structuredLyrics: null,
  loading: false,
  refreshing: false,
  error: null,
  retryAt: null,
})

const useEnhancedLyrics = ({
  trackId,
  updatedAt,
  disabled = false,
  requested = true,
  selectedSourceKey = null,
}) => {
  const preferredLanguage = getPreferredLyricLanguage()
  const cacheKey = useMemo(
    () => buildCacheKey({ trackId, preferredLanguage, updatedAt }),
    [preferredLanguage, trackId, updatedAt],
  )
  const [state, setState] = useState(() => emptyLyricsState(cacheKey))
  const [requestGeneration, setRequestGeneration] = useState(0)
  const [refreshToken, setRefreshToken] = useState(0)
  const [, setCountdownTick] = useState(0)

  // The refresh path reads the currently displayed state when deciding whether
  // to keep it visible, so mirror it through a ref updated before the request
  // effect runs.
  const stateRef = useRef(state)
  useEffect(() => {
    stateRef.current = state
  }, [state])
  const handledRefreshRef = useRef(refreshToken)

  const retry = useCallback(() => {
    cache.delete(cacheKey)
    setRequestGeneration((current) => current + 1)
  }, [cacheKey])

  const refresh = useCallback(() => {
    if (!trackId || disabled || !requested) return
    // Coalesce with an unsettled request: its answer is at least as fresh as
    // one started now, and restarting it would strand slow responses behind
    // the poll interval.
    const entry = inFlight.get(cacheKey)
    if (entry && !entry.settled) return
    setRefreshToken((current) => current + 1)
  }, [cacheKey, disabled, requested, trackId])

  useEffect(() => {
    if (!trackId || disabled || !requested) {
      handledRefreshRef.current = refreshToken
      setState(emptyLyricsState(cacheKey))
      return undefined
    }

    const isRefresh = refreshToken !== handledRefreshRef.current
    handledRefreshRef.current = refreshToken

    if (!isRefresh) {
      const cached = readCachedLyrics(cacheKey)
      if (cached) {
        setState({
          cacheKey,
          structuredLyrics: cached,
          loading: false,
          refreshing: false,
          error: null,
          retryAt: null,
        })
        return undefined
      }
    }

    // Progressive refreshes keep the currently shown lyrics; a refresh landing
    // while a request is in flight coalesces with it.
    const keepingContent = Boolean(
      isRefresh &&
      stateRef.current.cacheKey === cacheKey &&
      stateRef.current.structuredLyrics?.length,
    )
    let active = true
    const request = acquireLyricsRequest({ trackId, cacheKey })
    setState((current) =>
      keepingContent
        ? { ...current, refreshing: true }
        : {
            cacheKey,
            structuredLyrics: null,
            loading: true,
            refreshing: false,
            error: null,
            retryAt: null,
          },
    )

    request.promise
      .then((structuredLyrics) => {
        if (!active) return
        setState({
          cacheKey,
          structuredLyrics,
          loading: false,
          refreshing: false,
          error: null,
          retryAt: null,
        })
      })
      .catch((error) => {
        if (!active || error?.name === 'AbortError') return
        if (keepingContent && stateRef.current.cacheKey === cacheKey) {
          // A failed progressive refresh must not blank working lyrics.
          setState((current) =>
            current.cacheKey === cacheKey
              ? { ...current, refreshing: false }
              : current,
          )
          return
        }
        cache.delete(cacheKey)
        setState({
          cacheKey,
          structuredLyrics: null,
          loading: false,
          refreshing: false,
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
  }, [cacheKey, disabled, refreshToken, requestGeneration, requested, trackId])

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

  const structuredLyrics =
    state.cacheKey === cacheKey ? state.structuredLyrics : null
  const variants = useMemo(
    () => collectLyricVariants(structuredLyrics || []),
    [structuredLyrics],
  )
  const layers = useMemo(
    () =>
      structuredLyrics && structuredLyrics.length > 0
        ? selectLyricLayers(
            structuredLyrics,
            preferredLanguage,
            selectedSourceKey,
          )
        : emptyLyricLayers,
    [preferredLanguage, selectedSourceKey, structuredLyrics],
  )

  if (state.cacheKey !== cacheKey) {
    return {
      variants: [],
      layers: emptyLyricLayers,
      loading: Boolean(trackId && !disabled && requested),
      refreshing: false,
      error: null,
      retryAfterSeconds: null,
      retry,
      refresh,
    }
  }
  return {
    variants,
    layers,
    loading: state.loading,
    refreshing: state.refreshing,
    error: state.error,
    retryAfterSeconds:
      state.retryAt == null
        ? null
        : Math.max(1, Math.ceil((state.retryAt - Date.now()) / 1_000)),
    retry,
    refresh,
  }
}

export const clearEnhancedLyricsCache = () => {
  cache.clear()
  inFlight.forEach((entry) => entry.controller.abort())
  inFlight.clear()
}

export default useEnhancedLyrics
