import { useEffect, useRef, useState } from 'react'
import subsonic from '../subsonic'
import { getPreferredLyricLanguage, selectLyricLayers } from './lyrics'

export const emptyLyricLayers = Object.freeze({
  main: null,
  translation: null,
  pronunciation: null,
})

const normalizeLyricLayers = (layers) => ({
  main: layers?.main || null,
  translation: layers?.translation || null,
  pronunciation: layers?.pronunciation || null,
})

const readStructuredLyrics = (response) =>
  response?.json?.['subsonic-response']?.lyricsList?.structuredLyrics || []

const MAX_LYRIC_CACHE_ENTRIES = 75

const buildCacheKey = (trackId, preferredLanguage) =>
  `${trackId || ''}\u0000${preferredLanguage || ''}`

const rememberLyrics = (cache, cacheKey, layers) => {
  cache.delete(cacheKey)
  cache.set(cacheKey, layers)
  while (cache.size > MAX_LYRIC_CACHE_ENTRIES) {
    const oldestCacheKey = cache.keys().next().value
    cache.delete(oldestCacheKey)
  }
}

const useEnhancedLyrics = (trackId, disabled = false) => {
  const cacheRef = useRef(new Map())
  const requestIdRef = useRef(0)
  const [layers, setLayers] = useState(emptyLyricLayers)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)
  const preferredLanguage = getPreferredLyricLanguage()

  useEffect(() => {
    requestIdRef.current += 1
    const requestId = requestIdRef.current
    let cancelled = false

    if (!trackId || disabled) {
      setLayers(emptyLyricLayers)
      setLoading(false)
      setError(null)
      return () => {
        cancelled = true
      }
    }

    const cacheKey = buildCacheKey(trackId, preferredLanguage)
    const cached = cacheRef.current.get(cacheKey)
    if (cached) {
      rememberLyrics(cacheRef.current, cacheKey, cached)
      setLayers(cached)
      setLoading(false)
      setError(null)
      return () => {
        cancelled = true
      }
    }

    setLayers(emptyLyricLayers)
    setLoading(true)
    setError(null)

    subsonic
      .getLyricsBySongId(trackId)
      .then((response) => {
        if (cancelled || requestIdRef.current !== requestId) return
        const selected = normalizeLyricLayers(
          selectLyricLayers(readStructuredLyrics(response), preferredLanguage),
        )
        rememberLyrics(cacheRef.current, cacheKey, selected)
        setLayers(selected)
        setError(null)
      })
      .catch((err) => {
        if (cancelled || requestIdRef.current !== requestId) return
        cacheRef.current.delete(cacheKey)
        setLayers(emptyLyricLayers)
        setError(err)
      })
      .finally(() => {
        if (!cancelled && requestIdRef.current === requestId) {
          setLoading(false)
        }
      })

    return () => {
      cancelled = true
    }
  }, [disabled, preferredLanguage, trackId])

  return { layers, loading, error }
}

export default useEnhancedLyrics
