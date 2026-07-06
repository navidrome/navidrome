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

const useEnhancedLyrics = (trackId, disabled = false) => {
  const cacheRef = useRef(new Map())
  const requestIdRef = useRef(0)
  const [layers, setLayers] = useState(emptyLyricLayers)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)

  useEffect(() => {
    requestIdRef.current += 1
    const requestId = requestIdRef.current

    if (!trackId || disabled) {
      setLayers(emptyLyricLayers)
      setLoading(false)
      setError(null)
      return undefined
    }

    const cached = cacheRef.current.get(trackId)
    if (cached) {
      setLayers(cached)
      setLoading(false)
      setError(null)
      return undefined
    }

    setLoading(true)
    setError(null)

    subsonic
      .getLyricsBySongId(trackId)
      .then((response) => {
        if (requestIdRef.current !== requestId) return
        const selected = normalizeLyricLayers(
          selectLyricLayers(
            readStructuredLyrics(response),
            getPreferredLyricLanguage(),
          ),
        )
        cacheRef.current.set(trackId, selected)
        setLayers(selected)
        setError(null)
      })
      .catch((err) => {
        if (requestIdRef.current !== requestId) return
        cacheRef.current.delete(trackId)
        setLayers(emptyLyricLayers)
        setError(err)
      })
      .finally(() => {
        if (requestIdRef.current === requestId) {
          setLoading(false)
        }
      })

    return undefined
  }, [disabled, trackId])

  return { layers, loading, error }
}

export default useEnhancedLyrics
