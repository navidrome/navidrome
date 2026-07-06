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

const rememberLyrics = (cache, trackId, layers) => {
  cache.delete(trackId)
  cache.set(trackId, layers)
  while (cache.size > MAX_LYRIC_CACHE_ENTRIES) {
    const oldestTrackId = cache.keys().next().value
    cache.delete(oldestTrackId)
  }
}

const useEnhancedLyrics = (trackId, disabled = false) => {
  const cacheRef = useRef(new Map())
  const requestIdRef = useRef(0)
  const [layers, setLayers] = useState(emptyLyricLayers)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)

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

    const cached = cacheRef.current.get(trackId)
    if (cached) {
      rememberLyrics(cacheRef.current, trackId, cached)
      setLayers(cached)
      setLoading(false)
      setError(null)
      return () => {
        cancelled = true
      }
    }

    setLoading(true)
    setError(null)

    subsonic
      .getLyricsBySongId(trackId)
      .then((response) => {
        if (cancelled || requestIdRef.current !== requestId) return
        const selected = normalizeLyricLayers(
          selectLyricLayers(
            readStructuredLyrics(response),
            getPreferredLyricLanguage(),
          ),
        )
        rememberLyrics(cacheRef.current, trackId, selected)
        setLayers(selected)
        setError(null)
      })
      .catch((err) => {
        if (cancelled || requestIdRef.current !== requestId) return
        cacheRef.current.delete(trackId)
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
  }, [disabled, trackId])

  return { layers, loading, error }
}

export default useEnhancedLyrics
