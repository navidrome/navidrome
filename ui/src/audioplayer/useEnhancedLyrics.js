import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import subsonic from '../subsonic'
import {
  collectLyricVariants,
  getPreferredLyricLanguage,
  hasStructuredLyricContent,
  selectLyricLayers,
  variantKey,
} from './lyrics'

export const emptyLyricLayers = Object.freeze({
  main: null,
  translation: null,
  pronunciation: null,
})

const MAX_LYRIC_CACHE_ENTRIES = 75
const NEGATIVE_CACHE_TTL_MS = 30_000
const COOLDOWN_PATTERN = /(?:cooldown active for|retry after)\s+(\d+)s/gi
// Transient plugin failures (provider 5xx) can clear on their own: the plugin's
// background sweep fills its per-source caches after the failed fast pass, so a
// bounded backoff recovers without a manual track switch. Server-reported
// cooldowns use their own schedule instead.
const AUTO_RETRY_DELAYS_MS = [5_000, 15_000, 45_000, 90_000, 180_000]

const cache = new Map()
const inFlight = new Map()
const retryWaits = new Map()

const readRetryWait = (cacheKey) => {
  const waiting = retryWaits.get(cacheKey)
  if (waiting && waiting.retryAt > Date.now()) return waiting
  retryWaits.delete(cacheKey)
  return null
}

const rememberRetryWait = (
  cacheKey,
  delay,
  error = null,
  receivedAt = Date.now(),
) => {
  const previous = readRetryWait(cacheKey)
  const retryAt = Math.max(previous?.retryAt || 0, receivedAt + delay)
  const waiting = { retryAt, error: error || previous?.error || null }
  retryWaits.delete(cacheKey)
  retryWaits.set(cacheKey, waiting)
  while (retryWaits.size > MAX_LYRIC_CACHE_ENTRIES) {
    retryWaits.delete(retryWaits.keys().next().value)
  }
  return waiting
}

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

const rememberLyrics = (
  cacheKey,
  structuredLyrics,
  expiresAt = null,
  sourcesIncomplete = false,
) => {
  cache.delete(cacheKey)
  cache.set(cacheKey, { structuredLyrics, expiresAt, sourcesIncomplete })
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

// Servers re-serialize identical payloads per request, so a refreshed response
// only replaces the previous one when its content actually changed. Keeping the
// reference stable stops downstream memoization (and the lyrics panel's scroll
// reset) from churning on every progressive refresh.
const sameStructuredLyrics = (left, right) => {
  if (left === right) return true
  if (!left || !right) return false
  try {
    return JSON.stringify(left) === JSON.stringify(right)
  } catch {
    return false
  }
}

// A partial discovery must not discard a working source just because its
// provider failed this time. Complete responses can still remove stale sources.
const mergePartialSources = (previous, incoming) => {
  const group = (entries) => {
    const sources = new Map()
    for (const entry of entries || []) {
      const key = variantKey(entry.source)
      if (!sources.has(key)) sources.set(key, [])
      sources.get(key).push(entry)
    }
    return sources
  }
  const sources = group(incoming)
  const order = [...sources.keys()]
  const cachedSources = group(previous)
  const cachedOrder = [...cachedSources.keys()]
  cachedOrder.forEach((key, index) => {
    if (sources.has(key)) return
    const next = cachedOrder.slice(index + 1).find((key) => sources.has(key))
    const position = next == null ? order.length : order.indexOf(next)
    order.splice(position, 0, key)
    sources.set(key, cachedSources.get(key))
  })
  return order.flatMap((key) => sources.get(key))
}

const createLyricsRequest = ({ trackId, cacheKey, allSources }) => {
  const controller = new AbortController()
  const entry = {
    controller,
    consumers: 0,
    settled: false,
    settledAt: null,
    promise: null,
  }
  entry.promise = subsonic
    .getLyricsBySongId(trackId, { signal: controller.signal, allSources })
    .then((response) => {
      const incoming = normalizeStructuredLyrics(readStructuredLyrics(response))
      const sourcesIncomplete =
        response?.json?.['subsonic-response']?.lyricsList?.sourcesIncomplete ===
        true
      const structuredLyrics = sourcesIncomplete
        ? mergePartialSources(cache.get(cacheKey)?.structuredLyrics, incoming)
        : incoming
      const hasAnyLyric = structuredLyrics.some(hasStructuredLyricContent)
      rememberLyrics(
        cacheKey,
        structuredLyrics,
        hasAnyLyric ? null : Date.now() + NEGATIVE_CACHE_TTL_MS,
        sourcesIncomplete,
      )
      const seconds =
        response?.json?.['subsonic-response']?.lyricsList?.retryAfterSeconds
      const retryAt =
        Number.isFinite(seconds) && seconds > 0
          ? rememberRetryWait(cacheKey, seconds * 1000).retryAt
          : null
      if (retryAt == null) retryWaits.delete(cacheKey)
      return { structuredLyrics, sourcesIncomplete, retryAt }
    })
    .finally(() => {
      entry.settled = true
      entry.settledAt = Date.now()
      if (inFlight.get(cacheKey) === entry) inFlight.delete(cacheKey)
    })
  inFlight.set(cacheKey, entry)
  return entry
}

const acquireLyricsRequest = ({ trackId, cacheKey, allSources }) => {
  const existing = inFlight.get(cacheKey)
  const entry =
    existing && !existing.controller.signal.aborted
      ? existing
      : createLyricsRequest({ trackId, cacheKey, allSources })
  entry.consumers += 1
  let released = false
  return {
    promise: entry.promise,
    get settledAt() {
      return entry.settledAt
    },
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
  const [autoRetry, setAutoRetry] = useState(null)
  const [, setCountdownTick] = useState(0)

  // The refresh path reads the currently displayed state when deciding whether
  // to keep it visible, so mirror it through a ref updated before the request
  // effect runs.
  const stateRef = useRef(state)
  useEffect(() => {
    stateRef.current = state
  }, [state])
  const handledRefreshRef = useRef(refreshToken)
  const autoRetryRef = useRef({ cacheKey: null, attempts: 0 })

  const retry = useCallback(() => {
    if (readRetryWait(cacheKey)) return
    cache.delete(cacheKey)
    autoRetryRef.current = { cacheKey, attempts: 0 }
    setRequestGeneration((current) => current + 1)
  }, [cacheKey])

  const refresh = useCallback(() => {
    if (!trackId || disabled || !requested) return false
    if (readRetryWait(cacheKey)) return false
    // Let the current lookup finish. A later poll or menu opening can discover
    // additional sources without stranding slow responses behind the interval.
    const entry = inFlight.get(cacheKey)
    if (entry && !entry.settled) return false
    setRefreshToken((current) => current + 1)
    return true
  }, [cacheKey, disabled, requested, trackId])

  useEffect(() => {
    if (!trackId || disabled || !requested) {
      handledRefreshRef.current = refreshToken
      setAutoRetry(null)
      setState(emptyLyricsState(cacheKey))
      return undefined
    }

    const isRefresh = refreshToken !== handledRefreshRef.current
    handledRefreshRef.current = refreshToken

    const waiting = readRetryWait(cacheKey)
    if (waiting) {
      const cached = readCachedLyrics(cacheKey)
      setState({
        ...emptyLyricsState(cacheKey),
        structuredLyrics: cached,
        sourcesIncomplete:
          cache.get(cacheKey)?.sourcesIncomplete || Boolean(waiting.error),
        error: cached?.length ? null : waiting.error,
        retryAt: waiting.retryAt,
      })
      return undefined
    }

    if (!isRefresh) {
      const cached = readCachedLyrics(cacheKey)
      if (cached !== null) {
        setState({
          cacheKey,
          structuredLyrics: cached,
          sourcesIncomplete: cache.get(cacheKey)?.sourcesIncomplete || false,
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
    const request = acquireLyricsRequest({
      trackId,
      cacheKey,
      allSources: isRefresh,
    })
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
      .then(({ structuredLyrics, sourcesIncomplete, retryAt }) => {
        if (!active) return
        autoRetryRef.current = { cacheKey, attempts: 0 }
        setAutoRetry(null)
        if (keepingContent && structuredLyrics.length === 0) {
          rememberLyrics(cacheKey, stateRef.current.structuredLyrics)
          setState((current) => ({ ...current, refreshing: false }))
          return
        }
        setState((current) => ({
          cacheKey,
          structuredLyrics:
            current.cacheKey === cacheKey &&
            sameStructuredLyrics(current.structuredLyrics, structuredLyrics)
              ? current.structuredLyrics
              : structuredLyrics,
          loading: false,
          refreshing: false,
          error: null,
          retryAt,
          sourcesIncomplete,
        }))
      })
      .catch((error) => {
        if (!active || error?.name === 'AbortError') return
        const cooldownDelay = retryDelayFromError(error)
        if (keepingContent && stateRef.current.cacheKey === cacheKey) {
          // Keep readable lyrics while honoring both cooldowns and transient
          // failure backoff. Polling and menu refreshes share this deadline.
          const attempts =
            autoRetryRef.current.cacheKey === cacheKey
              ? autoRetryRef.current.attempts
              : 0
          autoRetryRef.current = { cacheKey, attempts: attempts + 1 }
          const delay =
            cooldownDelay ??
            AUTO_RETRY_DELAYS_MS[
              Math.min(attempts, AUTO_RETRY_DELAYS_MS.length - 1)
            ]
          const waiting = rememberRetryWait(
            cacheKey,
            delay,
            error,
            request.settledAt,
          )
          setState((current) =>
            current.cacheKey === cacheKey
              ? {
                  ...current,
                  refreshing: false,
                  retryAt: waiting.retryAt,
                  sourcesIncomplete: true,
                }
              : current,
          )
          return
        }
        cache.delete(cacheKey)
        const waiting =
          cooldownDelay == null
            ? null
            : rememberRetryWait(
                cacheKey,
                cooldownDelay,
                error,
                request.settledAt,
              )
        if (cooldownDelay != null) {
          autoRetryRef.current = { cacheKey, attempts: 0 }
          setAutoRetry(null)
        } else {
          const attempts =
            autoRetryRef.current.cacheKey === cacheKey
              ? autoRetryRef.current.attempts
              : 0
          autoRetryRef.current = { cacheKey, attempts: attempts + 1 }
          setAutoRetry(
            attempts < AUTO_RETRY_DELAYS_MS.length
              ? { cacheKey, at: Date.now() + AUTO_RETRY_DELAYS_MS[attempts] }
              : null,
          )
        }
        setState({
          cacheKey,
          structuredLyrics: null,
          loading: false,
          refreshing: false,
          error,
          retryAt: waiting?.retryAt || null,
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
    const retryTimer = window.setTimeout(() => {
      const waiting = readRetryWait(cacheKey)
      if (waiting) {
        setState((current) =>
          current.cacheKey === cacheKey
            ? { ...current, retryAt: waiting.retryAt }
            : current,
        )
        return
      }
      if (state.structuredLyrics?.length) refresh()
      else retry()
    }, retryDelay)
    const countdownTimer = window.setInterval(
      () => setCountdownTick((current) => current + 1),
      1_000,
    )
    return () => {
      window.clearTimeout(retryTimer)
      window.clearInterval(countdownTimer)
    }
  }, [
    cacheKey,
    disabled,
    requested,
    retry,
    refresh,
    state.cacheKey,
    state.retryAt,
    state.structuredLyrics,
  ])

  useEffect(() => {
    if (
      !autoRetry ||
      autoRetry.cacheKey !== cacheKey ||
      disabled ||
      !requested
    ) {
      return undefined
    }

    const retryDelay = Math.max(0, autoRetry.at - Date.now())
    const retryTimer = window.setTimeout(() => {
      setAutoRetry(null)
      setRequestGeneration((current) => current + 1)
    }, retryDelay)
    return () => window.clearTimeout(retryTimer)
  }, [autoRetry, cacheKey, disabled, requested])

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
    sourcesIncomplete: Boolean(state.sourcesIncomplete),
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
  retryWaits.clear()
  inFlight.forEach((entry) => entry.controller.abort())
  inFlight.clear()
}

export default useEnhancedLyrics
