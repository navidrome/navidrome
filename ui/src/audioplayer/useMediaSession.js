import { useCallback, useEffect, useRef } from 'react'
import subsonic from '../subsonic'

const ARTWORK_SIZES = [96, 192, 300, 512]
const DEFAULT_SKIP_SECONDS = 10
const MEDIA_SESSION_ACTIONS = [
  'play',
  'pause',
  'previoustrack',
  'nexttrack',
  'seekbackward',
  'seekforward',
  'seekto',
]

export const toAbsoluteArtworkUrl = (rawUrl, baseHref) => {
  if (!rawUrl || typeof rawUrl !== 'string') {
    return ''
  }
  const base =
    baseHref || (typeof window !== 'undefined' ? window.location?.href : '')
  if (!base) {
    return rawUrl
  }
  try {
    return new URL(rawUrl, base).href
  } catch {
    return rawUrl
  }
}

export const buildMediaSessionArtwork = (item, baseHref) => {
  if (!item || item.isRadio) {
    return []
  }

  const song = item.song
  const trackId = item.trackId || song?.mediaFileId || song?.id

  if (trackId && song?.album) {
    const coverRecord = {
      id: trackId,
      album: song.album,
      ...(song.imageHash && { imageHash: song.imageHash }),
      ...(song.updatedAt && { updatedAt: song.updatedAt }),
    }
    const entries = ARTWORK_SIZES.map((size) => {
      const url = subsonic.getCoverArtUrl(coverRecord, size)
      const src = toAbsoluteArtworkUrl(url, baseHref)
      return src ? { src, sizes: `${size}x${size}` } : null
    }).filter(Boolean)

    if (entries.length > 0) {
      return entries
    }
  }

  const fallbackSrc = toAbsoluteArtworkUrl(item.cover, baseHref)
  return fallbackSrc ? [{ src: fallbackSrc }] : []
}

export const resolveActiveMediaItem = (playerState) => {
  const queue = playerState?.queue || []
  if (queue.length === 0) {
    return null
  }
  if (playerState.playIndex != null && queue[playerState.playIndex]) {
    return queue[playerState.playIndex]
  }
  if (
    playerState.current?.uuid &&
    queue.some((it) => it.uuid === playerState.current.uuid)
  ) {
    return playerState.current
  }
  const savedIdx =
    playerState.savedPlayIndex >= 0 ? playerState.savedPlayIndex : 0
  return queue[savedIdx] || queue[0] || null
}

/**
 * Writes a MediaSession property while preserving compatibility with browser
 * extensions (such as KDE Plasma Browser Integration) that define an own
 * property setter on `navigator.mediaSession` without forwarding to the native
 * `MediaSession.prototype` setter. Calling both keeps extension-backed MPRIS
 * consumers (e.g. KDE Connect) and the browser's native OS MediaSession in sync.
 */
export const setMediaSessionProperty = (mediaSession, prop, value) => {
  if (!mediaSession) {
    return
  }
  try {
    mediaSession[prop] = value
  } catch {
    // Ignore non-writable test or browser environments
  }
  if (Object.prototype.hasOwnProperty.call(mediaSession, prop)) {
    const proto = Object.getPrototypeOf(mediaSession)
    const nativeSetter = proto
      ? Object.getOwnPropertyDescriptor(proto, prop)?.set
      : undefined
    if (typeof nativeSetter === 'function') {
      try {
        nativeSetter.call(mediaSession, value)
      } catch {
        // Ignore if mediaSession is a plain mock object without native brand
      }
    }
  }
}

const safeSetActionHandler = (mediaSession, action, handler) => {
  if (typeof mediaSession?.setActionHandler !== 'function') {
    return
  }
  try {
    mediaSession.setActionHandler(action, handler)
  } catch {
    // Unsupported MediaSession action in this browser
  }
}

const syncPositionState = (mediaSession, audio) => {
  if (!mediaSession || typeof mediaSession.setPositionState !== 'function') {
    return
  }
  if (!audio) {
    return
  }
  const duration = Number(audio.duration)
  if (!Number.isFinite(duration) || duration <= 0) {
    return
  }
  const rawRate = Number(audio.playbackRate)
  const playbackRate = Number.isFinite(rawRate) && rawRate > 0 ? rawRate : 1
  const rawPos = Number(audio.currentTime)
  const position = Number.isFinite(rawPos)
    ? Math.max(0, Math.min(rawPos, duration))
    : 0

  try {
    mediaSession.setPositionState({
      duration,
      playbackRate,
      position,
    })
  } catch {
    // Ignore transient duration/position race errors
  }
}

export const useMediaSession = (audioInstance, playerState) => {
  const lastMetadataKeyRef = useRef('')

  const syncTrackMetadata = useCallback((item) => {
    if (typeof navigator === 'undefined' || !('mediaSession' in navigator)) {
      return
    }
    const ms = navigator.mediaSession
    if (!ms) {
      return
    }

    if (!item || item.isRadio) {
      if (lastMetadataKeyRef.current !== '') {
        lastMetadataKeyRef.current = ''
        setMediaSessionProperty(ms, 'metadata', null)
        setMediaSessionProperty(ms, 'playbackState', 'none')
      }
      return
    }

    const song = item.song || {}
    const title = song.title || item.name || ''
    const artist = song.artist || item.singer || ''
    const album = song.album || ''
    const artwork = buildMediaSessionArtwork(item)
    const primaryArtwork = artwork[artwork.length - 1]?.src || ''
    const nextKey = `${title}\0${artist}\0${album}\0${primaryArtwork}`

    if (nextKey === lastMetadataKeyRef.current && ms.metadata != null) {
      return
    }
    lastMetadataKeyRef.current = nextKey

    const metadataInit = {
      title,
      artist,
      album,
      ...(artwork.length > 0 && { artwork }),
    }
    const metadataObj =
      typeof window !== 'undefined' &&
      typeof window.MediaMetadata === 'function'
        ? new window.MediaMetadata(metadataInit)
        : metadataInit

    setMediaSessionProperty(ms, 'metadata', metadataObj)
  }, [])

  const activeItem = resolveActiveMediaItem(playerState)
  const isActiveTrack = Boolean(activeItem && !activeItem.isRadio)

  useEffect(() => {
    syncTrackMetadata(activeItem)
  }, [activeItem, syncTrackMetadata])

  useEffect(() => {
    if (typeof navigator === 'undefined' || !('mediaSession' in navigator)) {
      return undefined
    }
    const ms = navigator.mediaSession
    if (!ms || !audioInstance || !isActiveTrack) {
      MEDIA_SESSION_ACTIONS.forEach((action) =>
        safeSetActionHandler(ms, action, null),
      )
      return undefined
    }

    safeSetActionHandler(ms, 'play', () => {
      if (audioInstance.paused) {
        if (typeof audioInstance.togglePlay === 'function') {
          audioInstance.togglePlay()
        } else {
          audioInstance.play?.()
        }
      }
    })

    safeSetActionHandler(ms, 'pause', () => {
      if (!audioInstance.paused) {
        if (typeof audioInstance.togglePlay === 'function') {
          audioInstance.togglePlay()
        } else {
          audioInstance.pause?.()
        }
      }
    })

    safeSetActionHandler(ms, 'previoustrack', () => {
      audioInstance.playPrev?.()
    })

    safeSetActionHandler(ms, 'nexttrack', () => {
      audioInstance.playNext?.()
    })

    safeSetActionHandler(ms, 'seekbackward', (details) => {
      const skip = Number(details?.seekOffset) || DEFAULT_SKIP_SECONDS
      audioInstance.currentTime = Math.max(
        (Number(audioInstance.currentTime) || 0) - skip,
        0,
      )
      syncPositionState(ms, audioInstance)
    })

    safeSetActionHandler(ms, 'seekforward', (details) => {
      const skip = Number(details?.seekOffset) || DEFAULT_SKIP_SECONDS
      const current = Number(audioInstance.currentTime) || 0
      const duration = Number(audioInstance.duration)
      audioInstance.currentTime =
        Number.isFinite(duration) && duration > 0
          ? Math.min(current + skip, duration)
          : current + skip
      syncPositionState(ms, audioInstance)
    })

    safeSetActionHandler(ms, 'seekto', (details) => {
      const target = Number(details?.seekTime)
      if (!Number.isFinite(target)) {
        return
      }
      if (details?.fastSeek && typeof audioInstance.fastSeek === 'function') {
        audioInstance.fastSeek(target)
      } else {
        audioInstance.currentTime = Math.max(0, target)
      }
      syncPositionState(ms, audioInstance)
    })

    const handlePlaying = () => {
      setMediaSessionProperty(ms, 'playbackState', 'playing')
      syncPositionState(ms, audioInstance)
    }

    const handlePaused = () => {
      setMediaSessionProperty(ms, 'playbackState', 'paused')
      syncPositionState(ms, audioInstance)
    }

    const handlePositionChange = () => {
      syncPositionState(ms, audioInstance)
    }

    if (typeof audioInstance.addEventListener === 'function') {
      audioInstance.addEventListener('play', handlePlaying)
      audioInstance.addEventListener('playing', handlePlaying)
      audioInstance.addEventListener('pause', handlePaused)
      audioInstance.addEventListener('ended', handlePaused)
      audioInstance.addEventListener('loadedmetadata', handlePositionChange)
      audioInstance.addEventListener('durationchange', handlePositionChange)
      audioInstance.addEventListener('ratechange', handlePositionChange)
      audioInstance.addEventListener('seeked', handlePositionChange)
    }

    setMediaSessionProperty(
      ms,
      'playbackState',
      audioInstance.paused ? 'paused' : 'playing',
    )
    syncPositionState(ms, audioInstance)

    return () => {
      MEDIA_SESSION_ACTIONS.forEach((action) =>
        safeSetActionHandler(ms, action, null),
      )
      if (typeof audioInstance.removeEventListener === 'function') {
        audioInstance.removeEventListener('play', handlePlaying)
        audioInstance.removeEventListener('playing', handlePlaying)
        audioInstance.removeEventListener('pause', handlePaused)
        audioInstance.removeEventListener('ended', handlePaused)
        audioInstance.removeEventListener(
          'loadedmetadata',
          handlePositionChange,
        )
        audioInstance.removeEventListener(
          'durationchange',
          handlePositionChange,
        )
        audioInstance.removeEventListener('ratechange', handlePositionChange)
        audioInstance.removeEventListener('seeked', handlePositionChange)
      }
    }
  }, [audioInstance, isActiveTrack])

  return { syncTrackMetadata }
}

export default useMediaSession
