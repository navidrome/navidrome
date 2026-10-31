import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useDispatch, useSelector } from 'react-redux'
import { useInterval } from '../../common'
import { currentPlaying } from '../../actions'
import config from '../../config'
import subsonic from '../../subsonic'
import { CastContext } from './CastContext'
import {
  CAST_SENDER_SDK_URL,
  MAX_CAST_WINDOW_ITEMS,
  buildCastQueueItems,
  buildCastQueueWindow,
  canLoadCastSdk,
  planCastQueueSync,
} from './castUtils'

const SHIMMED_AUDIO_PROPS = [
  'play',
  'pause',
  'load',
  'currentTime',
  'volume',
  'muted',
  'paused',
  'readyState',
  'networkState',
  'src',
  'setAttribute',
  'removeAttribute',
]

const ensureCastSdkLoaded = () => {
  if (typeof window === 'undefined' || typeof document === 'undefined') {
    return
  }
  if (window.cast?.framework?.CastContext && window.chrome?.cast?.media) {
    return
  }
  if (!canLoadCastSdk(config, window)) {
    return
  }

  if (!window.__ND_CAST_CALLBACK_INSTALLED__) {
    const prevCallback = window.__onGCastApiAvailable
    window.__ND_CAST_CALLBACK_INSTALLED__ = true
    window.__onGCastApiAvailable = (isAvailable) => {
      if (typeof prevCallback === 'function') {
        prevCallback(isAvailable)
      }
      if (isAvailable) {
        window.__ND_CAST_API_READY__ = true
        window.dispatchEvent(new Event('nd-cast-api-ready'))
      }
    }
  }

  if (!document.querySelector('script[data-nd-cast-sdk="true"]')) {
    const script = document.createElement('script')
    script.src = CAST_SENDER_SDK_URL
    script.async = true
    script.dataset.ndCastSdk = 'true'
    document.head.appendChild(script)
  }
}

const getRepeatMode = (chromeCastApi, mode) => {
  const repeat = chromeCastApi?.media?.RepeatMode
  if (!repeat) return undefined
  switch (mode) {
    case 'orderLoop':
      return repeat.ALL
    case 'singleLoop':
      return repeat.SINGLE
    case 'shufflePlay':
      return repeat.ALL_AND_SHUFFLE
    default:
      return repeat.OFF
  }
}

const resolveReceiverRepeatMode = (
  chromeCastApi,
  mode,
  windowStartIndex,
  windowEndIndex,
  queueLength,
) => {
  const fullQueueLoaded =
    windowStartIndex === 0 && windowEndIndex === queueLength
  if (fullQueueLoaded || mode === 'singleLoop') {
    return getRepeatMode(chromeCastApi, mode)
  }
  return chromeCastApi?.media?.RepeatMode?.OFF
}

const dispatchMediaEvent = (audio, eventName) => {
  if (audio && typeof audio.dispatchEvent === 'function') {
    try {
      audio.dispatchEvent(new Event(eventName))
    } catch {
      // Ignore synthetic event errors on non-DOM test doubles
    }
  }
}

const CastProvider = ({ audioInstance, enabled = true, children }) => {
  const dispatch = useDispatch()
  const playerState = useSelector((state) => state.player)

  const [castAvailable, setCastAvailable] = useState(false)
  const [isCasting, setIsCasting] = useState(false)
  const [isConnecting, setIsConnecting] = useState(false)
  const [deviceName, setDeviceName] = useState('')
  const [remoteState, setRemoteState] = useState('IDLE')
  const [activeTrackId, setActiveTrackId] = useState(null)
  const [activeIsRadio, setActiveIsRadio] = useState(false)

  const castContextRef = useRef(null)
  const remotePlayerRef = useRef(null)
  const remoteControllerRef = useRef(null)
  const mediaSessionRef = useRef(null)
  const uuidToItemIdRef = useRef(new Map())

  const playerStateRef = useRef(playerState)
  playerStateRef.current = playerState
  const audioInstanceRef = useRef(audioInstance)
  audioInstanceRef.current = audioInstance

  const isCastingRef = useRef(false)
  if (!isCasting && isCastingRef.current) {
    isCastingRef.current = false
  } else if (isCasting) {
    isCastingRef.current = true
  }

  const loadedWindowUuidsRef = useRef([])
  const loadedWindowRef = useRef({ start: 0, end: 0 })
  const activeUuidRef = useRef(null)
  const pendingCastLoadRef = useRef(false)
  const castReportedTrackRef = useRef(null)
  const castReportedStateRef = useRef('IDLE')
  const castLastPosMsRef = useRef(0)
  const seekReportTimerRef = useRef(null)

  const bridgeRef = useRef({
    shimmedAudio: null,
    savedDescriptors: {},
    virtualCurrentTime: 0,
    virtualVolume: 1,
    virtualMuted: false,
    virtualPaused: true,
    virtualSrc: '',
    trackChangedWhileCasting: false,
    seekingUntil: 0,
    lastSeekSentAt: 0,
    seekThrottleTimer: null,
    volumeDebounceTimer: null,
    nativePause: null,
    nativeSetAttribute: null,
    nativeRemoveAttribute: null,
  })

  const recordMediaSessionItems = useCallback((mediaSession) => {
    const items = mediaSession?.items
    if (!Array.isArray(items)) return
    items.forEach((it) => {
      const uuid = it?.media?.customData?.uuid
      if (uuid && it.itemId != null) {
        uuidToItemIdRef.current.set(uuid, it.itemId)
      }
    })
  }, [])

  const getActiveMediaSession = useCallback(() => {
    const session = castContextRef.current?.getCurrentSession?.()
    const media =
      session?.getMediaSession?.() ||
      session?.getSessionObj?.()?.media?.[0] ||
      mediaSessionRef.current
    if (media) {
      mediaSessionRef.current = media
      recordMediaSessionItems(media)
    }
    return media
  }, [recordMediaSessionItems])

  const findCastItemId = useCallback((mediaSession, uuid) => {
    if (!uuid) return undefined
    const direct = mediaSession?.items?.find(
      (it) => it?.media?.customData?.uuid === uuid,
    )
    if (direct && direct.itemId != null) {
      uuidToItemIdRef.current.set(uuid, direct.itemId)
      return direct.itemId
    }
    return uuidToItemIdRef.current.get(uuid)
  }, [])

  const getActiveCastTime = useCallback(() => {
    const rp = remotePlayerRef.current
    if (
      rp &&
      rp.isConnected &&
      Number.isFinite(rp.currentTime) &&
      rp.currentTime > 0
    ) {
      return Math.floor(rp.currentTime)
    }
    return Math.floor(bridgeRef.current.virtualCurrentTime || 0)
  }, [])

  const flushStoppedReport = useCallback(() => {
    const trackId = castReportedTrackRef.current
    if (trackId) {
      subsonic
        .reportPlayback(trackId, castLastPosMsRef.current, 'stopped')
        .catch(() => {})
      castReportedTrackRef.current = null
      castReportedStateRef.current = 'IDLE'
    }
  }, [])

  const seekCastTo = useCallback((seconds) => {
    const bridge = bridgeRef.current
    const queue = playerStateRef.current?.queue || []
    const currentItem = queue.find((it) => it.uuid === activeUuidRef.current)
    if (currentItem?.isRadio) return

    const maxDuration = Number(
      remotePlayerRef.current?.duration || currentItem?.duration || 0,
    )
    const raw = Number.isFinite(Number(seconds)) ? Number(seconds) : 0
    const clamped =
      maxDuration > 0
        ? Math.max(0, Math.min(raw, maxDuration))
        : Math.max(0, raw)

    bridge.virtualCurrentTime = clamped
    bridge.seekingUntil = Date.now() + 800
    castLastPosMsRef.current = Math.floor(clamped * 1000)

    const sendSeek = (targetSec) => {
      const rp = remotePlayerRef.current
      const controller = remoteControllerRef.current
      if (!rp || !controller) return
      rp.currentTime = targetSec
      controller.seek()
      bridge.lastSeekSentAt = Date.now()
    }

    const elapsed = Date.now() - bridge.lastSeekSentAt
    if (elapsed >= 150 && !bridge.seekThrottleTimer) {
      sendSeek(clamped)
    } else if (!bridge.seekThrottleTimer) {
      bridge.seekThrottleTimer = setTimeout(() => {
        bridge.seekThrottleTimer = null
        if (isCastingRef.current) {
          sendSeek(bridge.virtualCurrentTime)
        }
      }, 150)
    }

    if (seekReportTimerRef.current) {
      clearTimeout(seekReportTimerRef.current)
    }
    seekReportTimerRef.current = setTimeout(() => {
      seekReportTimerRef.current = null
      if (
        isCastingRef.current &&
        castReportedTrackRef.current &&
        !currentItem?.isRadio
      ) {
        const stateStr =
          remotePlayerRef.current?.playerState === 'PAUSED'
            ? 'paused'
            : 'playing'
        subsonic
          .reportPlayback(
            castReportedTrackRef.current,
            Math.floor(bridge.virtualCurrentTime * 1000),
            stateStr,
          )
          .catch(() => {})
      }
    }, 250)
  }, [])

  const loadQueueOnCast = useCallback(
    (targetIndex = 0, startSeconds = 0) => {
      const ctx = castContextRef.current
      const session = ctx?.getCurrentSession?.()
      const sessionObj = session?.getSessionObj?.()
      const chromeCastApi = window.chrome?.cast
      const state = playerStateRef.current
      const queue = state?.queue || []

      if (!sessionObj || !chromeCastApi?.media || queue.length === 0) {
        return
      }

      const windowSpec = buildCastQueueWindow(
        queue,
        targetIndex,
        MAX_CAST_WINDOW_ITEMS,
      )
      const queueItems = buildCastQueueItems(
        chromeCastApi,
        windowSpec.items,
        window.location.origin,
      )
      if (queueItems.length === 0) {
        return
      }

      const request = new chromeCastApi.media.QueueLoadRequest(queueItems)
      request.startIndex = Math.min(
        windowSpec.startOffset,
        queueItems.length - 1,
      )
      if (startSeconds > 0 && !windowSpec.items[request.startIndex]?.isRadio) {
        request.currentTime = startSeconds
      }
      const repeatMode = resolveReceiverRepeatMode(
        chromeCastApi,
        state.mode,
        windowSpec.windowStartIndex,
        windowSpec.windowEndIndex,
        queue.length,
      )
      if (repeatMode) {
        request.repeatMode = repeatMode
      }

      uuidToItemIdRef.current.clear()
      mediaSessionRef.current = null
      loadedWindowUuidsRef.current = windowSpec.items.map((item) => item.uuid)
      loadedWindowRef.current = {
        start: windowSpec.windowStartIndex,
        end: windowSpec.windowEndIndex,
      }
      pendingCastLoadRef.current = true

      const targetItem = windowSpec.items[request.startIndex]
      if (targetItem) {
        activeUuidRef.current = targetItem.uuid
        setActiveTrackId(targetItem.trackId || null)
        setActiveIsRadio(Boolean(targetItem.isRadio))
        bridgeRef.current.virtualCurrentTime = startSeconds
        bridgeRef.current.virtualPaused = false
        castLastPosMsRef.current = Math.floor(startSeconds * 1000)

        if (
          state.current?.uuid !== targetItem.uuid ||
          state.playIndex != null
        ) {
          dispatch(
            currentPlaying({
              ...targetItem,
              currentTime: startSeconds,
              duration: targetItem.duration || 0,
              volume: state.volume ?? 1,
            }),
          )
        }
      }

      sessionObj.queueLoad(
        request,
        (media) => {
          if (media) {
            mediaSessionRef.current = media
            recordMediaSessionItems(media)
          }
        },
        () => {
          pendingCastLoadRef.current = false
        },
      )
    },
    [dispatch, recordMediaSessionItems],
  )

  const uninstallAudioShim = useCallback((audio, finalTimeSec = 0) => {
    const bridge = bridgeRef.current
    const target = audio || bridge.shimmedAudio
    if (!target || bridge.shimmedAudio !== target) {
      return
    }

    if (bridge.seekThrottleTimer) {
      clearTimeout(bridge.seekThrottleTimer)
      bridge.seekThrottleTimer = null
    }
    if (bridge.volumeDebounceTimer) {
      clearTimeout(bridge.volumeDebounceTimer)
      bridge.volumeDebounceTimer = null
    }

    const saved = bridge.savedDescriptors || {}
    const nativePause = bridge.nativePause
    const nativeSetAttribute = bridge.nativeSetAttribute
    const changedSrc = bridge.trackChangedWhileCasting ? bridge.virtualSrc : ''

    SHIMMED_AUDIO_PROPS.forEach((prop) => {
      try {
        delete target[prop]
        if (saved[prop]) {
          Object.defineProperty(target, prop, saved[prop])
        }
      } catch {
        // Ignore non-configurable property errors
      }
    })

    bridge.shimmedAudio = null
    bridge.savedDescriptors = {}
    bridge.trackChangedWhileCasting = false

    if (nativePause) {
      try {
        nativePause()
      } catch {
        // Ignore DOM media pause errors
      }
    }

    if (changedSrc && nativeSetAttribute) {
      try {
        nativeSetAttribute('src', changedSrc)
      } catch {
        // Ignore attribute restoration errors
      }
    }

    if (Number.isFinite(finalTimeSec) && finalTimeSec > 0) {
      try {
        target.currentTime = finalTimeSec
      } catch {
        // If metadata for a newly switched track has not loaded yet, apply once loaded
        if (changedSrc && typeof target.addEventListener === 'function') {
          target.addEventListener(
            'loadedmetadata',
            () => {
              try {
                target.currentTime = finalTimeSec
              } catch {
                // Ignore
              }
            },
            { once: true },
          )
        }
      }
    }

    dispatchMediaEvent(target, 'pause')
    dispatchMediaEvent(target, 'timeupdate')
    dispatchMediaEvent(target, 'volumechange')
  }, [])

  const installAudioShim = useCallback(
    (audio, initialTimeSec = 0) => {
      if (!audio) return
      const bridge = bridgeRef.current
      if (bridge.shimmedAudio === audio) {
        return
      }
      if (bridge.shimmedAudio && bridge.shimmedAudio !== audio) {
        uninstallAudioShim(bridge.shimmedAudio, 0)
      }

      const nativePause =
        typeof audio.pause === 'function' ? audio.pause.bind(audio) : null
      const nativeSetAttribute =
        typeof audio.setAttribute === 'function'
          ? audio.setAttribute.bind(audio)
          : null
      const nativeRemoveAttribute =
        typeof audio.removeAttribute === 'function'
          ? audio.removeAttribute.bind(audio)
          : null

      // Pause local HTML5 audio before installing property overrides
      if (nativePause && !audio.paused) {
        try {
          nativePause()
        } catch {
          // Ignore DOM media pause errors
        }
      }

      const initialVol =
        typeof audio.volume === 'number' && Number.isFinite(audio.volume)
          ? audio.volume
          : 1
      const initialMuted = Boolean(audio.muted)
      const initialSrc =
        typeof audio.getAttribute === 'function'
          ? audio.getAttribute('src') || ''
          : audio.src || ''

      const savedDescriptors = {}
      SHIMMED_AUDIO_PROPS.forEach((prop) => {
        const desc = Object.getOwnPropertyDescriptor(audio, prop)
        if (desc) {
          savedDescriptors[prop] = desc
        }
      })

      bridge.shimmedAudio = audio
      bridge.savedDescriptors = savedDescriptors
      bridge.virtualCurrentTime = Number.isFinite(initialTimeSec)
        ? initialTimeSec
        : 0
      bridge.virtualVolume = initialVol
      bridge.virtualMuted = initialMuted
      bridge.virtualPaused = false
      bridge.virtualSrc = initialSrc
      bridge.trackChangedWhileCasting = false
      bridge.seekingUntil = 0
      bridge.nativePause = nativePause
      bridge.nativeSetAttribute = nativeSetAttribute
      bridge.nativeRemoveAttribute = nativeRemoveAttribute

      Object.defineProperties(audio, {
        play: {
          configurable: true,
          value: () => {
            bridge.virtualPaused = false
            if (!pendingCastLoadRef.current) {
              const rp = remotePlayerRef.current
              const controller = remoteControllerRef.current
              if (rp?.playerState === 'PAUSED' && controller?.playOrPause) {
                controller.playOrPause()
              } else if (rp?.playerState === 'IDLE') {
                const pState = playerStateRef.current
                const idx = Math.max(
                  0,
                  (pState?.queue || []).findIndex(
                    (it) => it.uuid === activeUuidRef.current,
                  ),
                )
                loadQueueOnCast(idx, Math.floor(bridge.virtualCurrentTime || 0))
              }
            }
            dispatchMediaEvent(audio, 'play')
            return Promise.resolve()
          },
        },
        pause: {
          configurable: true,
          value: () => {
            const activeStillInQueue =
              Boolean(activeUuidRef.current) &&
              (playerStateRef.current?.queue || []).some(
                (it) => it.uuid === activeUuidRef.current,
              )
            if (!activeStillInQueue || pendingCastLoadRef.current) {
              return
            }
            bridge.virtualPaused = true
            const rp = remotePlayerRef.current
            const controller = remoteControllerRef.current
            if (
              (rp?.playerState === 'PLAYING' ||
                rp?.playerState === 'BUFFERING') &&
              controller?.playOrPause
            ) {
              controller.playOrPause()
            }
            dispatchMediaEvent(audio, 'pause')
          },
        },
        load: {
          configurable: true,
          value: () => {
            setTimeout(() => {
              if (!isCastingRef.current || bridge.shimmedAudio !== audio) {
                return
              }
              dispatchMediaEvent(audio, 'canplay')
              const pState = remotePlayerRef.current?.playerState
              if (pState === 'PLAYING' || !bridge.virtualPaused) {
                dispatchMediaEvent(audio, 'play')
              }
              dispatchMediaEvent(audio, 'timeupdate')
            }, 0)
          },
        },
        currentTime: {
          configurable: true,
          get: () => bridge.virtualCurrentTime,
          set: (val) => {
            seekCastTo(val)
          },
        },
        volume: {
          configurable: true,
          get: () => bridge.virtualVolume,
          set: (val) => {
            const num = Number(val)
            const clamped = Math.max(
              0,
              Math.min(1, Number.isFinite(num) ? num : 1),
            )
            bridge.virtualVolume = clamped
            if (bridge.volumeDebounceTimer) {
              clearTimeout(bridge.volumeDebounceTimer)
            }
            bridge.volumeDebounceTimer = setTimeout(() => {
              bridge.volumeDebounceTimer = null
              if (!isCastingRef.current) return
              const rp = remotePlayerRef.current
              const controller = remoteControllerRef.current
              if (!rp || !controller) return
              const castVol = Math.max(
                0,
                Math.min(1, Math.sqrt(bridge.virtualVolume)),
              )
              if (
                typeof controller.setVolumeLevel === 'function' &&
                Math.abs((rp.volumeLevel ?? 1) - castVol) >= 0.005
              ) {
                rp.volumeLevel = castVol
                controller.setVolumeLevel()
              }
              if (rp.isMuted && castVol > 0 && controller.muteOrUnmute) {
                controller.muteOrUnmute()
              }
            }, 50)
          },
        },
        muted: {
          configurable: true,
          get: () => bridge.virtualMuted,
          set: (val) => {
            const nextMuted = Boolean(val)
            bridge.virtualMuted = nextMuted
            const rp = remotePlayerRef.current
            const controller = remoteControllerRef.current
            if (
              rp &&
              controller?.muteOrUnmute &&
              Boolean(rp.isMuted) !== nextMuted
            ) {
              controller.muteOrUnmute()
            }
          },
        },
        paused: {
          configurable: true,
          get: () => bridge.virtualPaused,
        },
        readyState: {
          configurable: true,
          get: () => 4,
        },
        networkState: {
          configurable: true,
          get: () => 1,
        },
        src: {
          configurable: true,
          get: () => bridge.virtualSrc,
          set: (val) => {
            bridge.virtualSrc = String(val ?? '')
            bridge.trackChangedWhileCasting = true
          },
        },
        setAttribute: {
          configurable: true,
          value: (name, val) => {
            if (String(name).toLowerCase() === 'src') {
              bridge.virtualSrc = String(val ?? '')
              bridge.trackChangedWhileCasting = true
              return
            }
            if (nativeSetAttribute) {
              nativeSetAttribute(name, val)
            }
          },
        },
        removeAttribute: {
          configurable: true,
          value: (name) => {
            if (String(name).toLowerCase() === 'src') {
              bridge.virtualSrc = ''
              bridge.trackChangedWhileCasting = true
              return
            }
            if (nativeRemoveAttribute) {
              nativeRemoveAttribute(name)
            }
          },
        },
      })

      setTimeout(() => {
        if (isCastingRef.current && bridge.shimmedAudio === audio) {
          dispatchMediaEvent(audio, 'canplay')
        }
      }, 0)
    },
    [loadQueueOnCast, seekCastTo, uninstallAudioShim],
  )

  const syncFromRemotePlayer = useCallback(() => {
    const rp = remotePlayerRef.current
    const ctx = castContextRef.current
    if (!rp) return

    const connected = Boolean(rp.isConnected)
    const session = ctx?.getCurrentSession?.()
    const castDeviceName =
      session?.getCastDevice?.()?.friendlyName ||
      session?.getSessionObj?.()?.receiver?.friendlyName ||
      ''

    setDeviceName(castDeviceName)

    if (!connected) {
      return
    }

    const mediaSession = getActiveMediaSession()
    const nextState = rp.playerState || 'IDLE'
    const nextTime = Number.isFinite(rp.currentTime) ? rp.currentTime : 0
    const nextDuration = Number.isFinite(rp.duration) ? rp.duration : 0
    const audio = audioInstanceRef.current
    const bridge = bridgeRef.current

    if (nextState === 'PLAYING' || nextState === 'PAUSED') {
      pendingCastLoadRef.current = false
    }

    setRemoteState(nextState)

    if (typeof rp.volumeLevel === 'number' && Number.isFinite(rp.volumeLevel)) {
      const nextVirtualVol = rp.isMuted
        ? 0
        : Math.max(0, Math.min(1, rp.volumeLevel ** 2))
      bridge.virtualMuted = Boolean(rp.isMuted)
      if (
        !bridge.volumeDebounceTimer &&
        Math.abs(bridge.virtualVolume - nextVirtualVol) >= 0.01
      ) {
        bridge.virtualVolume = nextVirtualVol
        dispatchMediaEvent(audio, 'volumechange')
      }
    }

    const isActivelySeeking = Date.now() < bridge.seekingUntil
    if (!isActivelySeeking) {
      const preservePendingOffset =
        pendingCastLoadRef.current &&
        nextTime === 0 &&
        bridge.virtualCurrentTime > 0
      if (!preservePendingOffset) {
        bridge.virtualCurrentTime = nextTime
        castLastPosMsRef.current = Math.floor(nextTime * 1000)
        dispatchMediaEvent(audio, 'timeupdate')
      }
    }

    if (nextState === 'PLAYING') {
      if (bridge.virtualPaused) {
        bridge.virtualPaused = false
      }
      dispatchMediaEvent(audio, 'play')
    } else if (nextState === 'PAUSED') {
      if (!bridge.virtualPaused) {
        bridge.virtualPaused = true
      }
      dispatchMediaEvent(audio, 'pause')
    }

    const mediaInfo = rp.mediaInfo
    const customData = mediaInfo?.customData || {}
    const queue = playerStateRef.current?.queue || []

    let matchedIdx = -1
    if (customData.uuid) {
      matchedIdx = queue.findIndex((item) => item.uuid === customData.uuid)
    }
    if (matchedIdx < 0 && customData.trackId) {
      matchedIdx = queue.findIndex(
        (item) => item.trackId === customData.trackId,
      )
    }
    const matched = matchedIdx >= 0 ? queue[matchedIdx] : null

    if (matched) {
      if (activeUuidRef.current !== matched.uuid) {
        activeUuidRef.current = matched.uuid
        setActiveTrackId(matched.trackId || null)
        setActiveIsRadio(Boolean(matched.isRadio))
        dispatch(
          currentPlaying({
            ...matched,
            currentTime: nextTime,
            duration: nextDuration || matched.duration || 0,
            volume: playerStateRef.current?.volume ?? 1,
          }),
        )
        if (audio && typeof audio.updatePlayIndex === 'function') {
          audio.updatePlayIndex(matchedIdx)
        }
      }
      if (matched.song?.title && matched.song?.artist) {
        document.title = `${matched.song.title} - ${matched.song.artist} - Navidrome`
      }
    }

    const rawTrackId = matched?.trackId
    const trackId =
      typeof rawTrackId === 'string' && /^[0-9A-Za-z_-]+$/.test(rawTrackId)
        ? rawTrackId
        : null
    const isRadio = Boolean(matched?.isRadio)
    setActiveTrackId(trackId)
    setActiveIsRadio(isRadio)

    if (trackId && !isRadio) {
      const posMs = Math.floor(nextTime * 1000)
      if (nextState === 'PLAYING') {
        if (castReportedTrackRef.current !== trackId) {
          if (castReportedTrackRef.current) {
            subsonic
              .reportPlayback(
                castReportedTrackRef.current,
                castLastPosMsRef.current,
                'stopped',
              )
              .catch(() => {})
          }
          castReportedTrackRef.current = trackId
          castReportedStateRef.current = 'PLAYING'
          subsonic
            .reportPlayback(trackId, posMs, 'starting')
            .then(() => subsonic.reportPlayback(trackId, posMs, 'playing'))
            .catch(() => {})
        } else if (castReportedStateRef.current !== 'PLAYING') {
          castReportedStateRef.current = 'PLAYING'
          subsonic.reportPlayback(trackId, posMs, 'playing').catch(() => {})
        }
      } else if (
        nextState === 'PAUSED' &&
        castReportedTrackRef.current === trackId &&
        castReportedStateRef.current !== 'PAUSED'
      ) {
        castReportedStateRef.current = 'PAUSED'
        subsonic.reportPlayback(trackId, posMs, 'paused').catch(() => {})
      }
    }

    // Auto-advance to the next 40-track window only once when the receiver
    // finishes the last item of a partial window and more songs exist in queue.
    if (
      nextState === 'IDLE' &&
      !pendingCastLoadRef.current &&
      mediaSession?.idleReason ===
        window.chrome?.cast?.media?.IdleReason?.FINISHED
    ) {
      const { end } = loadedWindowRef.current
      if (
        end > 0 &&
        end < queue.length &&
        activeUuidRef.current === queue[end - 1]?.uuid
      ) {
        loadQueueOnCast(end, 0)
      }
    }
  }, [dispatch, getActiveMediaSession, loadQueueOnCast])

  useEffect(() => {
    let cleanupListeners = null

    const initCastSdk = () => {
      const framework = window.cast?.framework
      const chromeCast = window.chrome?.cast
      if (
        !framework?.CastContext ||
        !chromeCast?.media ||
        castContextRef.current
      ) {
        return
      }

      try {
        const ctx = framework.CastContext.getInstance()
        ctx.setOptions({
          receiverApplicationId: chromeCast.media.DEFAULT_MEDIA_RECEIVER_APP_ID,
          autoJoinPolicy: chromeCast.AutoJoinPolicy.ORIGIN_SCOPED,
        })

        const rp = new framework.RemotePlayer()
        const controller = new framework.RemotePlayerController(rp)

        castContextRef.current = ctx
        remotePlayerRef.current = rp
        remoteControllerRef.current = controller
        setCastAvailable(true)

        const handleCastState = (event) => {
          const stateStr = event?.castState || ctx.getCastState?.() || ''
          setIsConnecting(
            stateStr === framework.CastState?.CONNECTING ||
              stateStr === 'CONNECTING',
          )
        }

        const handleSessionState = (event) => {
          const sessionState = event?.sessionState || ''
          const types = framework.SessionState || {}

          if (
            sessionState === types.SESSION_STARTED ||
            sessionState === 'SESSION_STARTED'
          ) {
            const audio = audioInstanceRef.current
            const resumeSec =
              audio && Number.isFinite(audio.currentTime)
                ? Math.floor(audio.currentTime)
                : 0
            isCastingRef.current = true
            installAudioShim(audio, resumeSec)
            setIsCasting(true)
            setIsConnecting(false)

            const pState = playerStateRef.current
            const startIdx =
              pState.playIndex ??
              (pState.savedPlayIndex >= 0 ? pState.savedPlayIndex : 0)
            loadQueueOnCast(startIdx, resumeSec)
            syncFromRemotePlayer()
          } else if (
            sessionState === types.SESSION_RESUMED ||
            sessionState === 'SESSION_RESUMED'
          ) {
            const audio = audioInstanceRef.current
            const resumeSec =
              remotePlayerRef.current?.currentTime ||
              (audio && Number.isFinite(audio.currentTime)
                ? Math.floor(audio.currentTime)
                : 0)
            isCastingRef.current = true
            installAudioShim(audio, resumeSec)
            setIsCasting(true)
            setIsConnecting(false)

            const existingMedia = ctx.getCurrentSession?.()?.getMediaSession?.()
            const pState = playerStateRef.current
            if (!existingMedia || existingMedia.playerState === 'IDLE') {
              const startIdx =
                pState.playIndex ??
                (pState.savedPlayIndex >= 0 ? pState.savedPlayIndex : 0)
              loadQueueOnCast(startIdx, 0)
            } else {
              mediaSessionRef.current = existingMedia
              recordMediaSessionItems(existingMedia)
              const queue = pState.queue || []
              const activeIdx =
                pState.savedPlayIndex >= 0 ? pState.savedPlayIndex : 0
              const winSpec = buildCastQueueWindow(
                queue,
                activeIdx,
                MAX_CAST_WINDOW_ITEMS,
              )
              loadedWindowUuidsRef.current = winSpec.items.map((it) => it.uuid)
              loadedWindowRef.current = {
                start: winSpec.windowStartIndex,
                end: winSpec.windowEndIndex,
              }
            }
            syncFromRemotePlayer()
          } else if (
            sessionState === types.SESSION_ENDED ||
            sessionState === 'SESSION_ENDED'
          ) {
            const finalSec = getActiveCastTime()
            const endedUuid = activeUuidRef.current
            flushStoppedReport()
            pendingCastLoadRef.current = false
            mediaSessionRef.current = null
            uuidToItemIdRef.current.clear()
            loadedWindowUuidsRef.current = []
            activeUuidRef.current = null

            const audio = audioInstanceRef.current
            const queue = playerStateRef.current?.queue || []
            const endedIdx = endedUuid
              ? queue.findIndex((it) => it.uuid === endedUuid)
              : -1
            if (
              endedIdx >= 0 &&
              !bridgeRef.current.trackChangedWhileCasting &&
              endedIdx !== playerStateRef.current?.savedPlayIndex &&
              typeof audio?.updatePlayIndex === 'function'
            ) {
              audio.updatePlayIndex(endedIdx)
            }
            const shouldApplyFinalSec =
              endedIdx >= 0 &&
              !queue[endedIdx]?.isRadio &&
              (bridgeRef.current.trackChangedWhileCasting ||
                endedIdx === playerStateRef.current?.savedPlayIndex ||
                playerStateRef.current?.current?.uuid === endedUuid)

            isCastingRef.current = false
            uninstallAudioShim(audio, shouldApplyFinalSec ? finalSec : 0)

            setIsCasting(false)
            setIsConnecting(false)
            setRemoteState('IDLE')
            setDeviceName('')
            setActiveTrackId(null)
            setActiveIsRadio(false)
          }
        }

        const handleRemoteChange = () => {
          syncFromRemotePlayer()
        }

        ctx.addEventListener(
          framework.CastContextEventType.CAST_STATE_CHANGED,
          handleCastState,
        )
        ctx.addEventListener(
          framework.CastContextEventType.SESSION_STATE_CHANGED,
          handleSessionState,
        )
        controller.addEventListener(
          framework.RemotePlayerEventType.ANY_CHANGE,
          handleRemoteChange,
        )

        // Check if a session is already active at mount time
        const existingSession = ctx.getCurrentSession?.()
        if (existingSession) {
          isCastingRef.current = true
          installAudioShim(
            audioInstanceRef.current,
            remotePlayerRef.current?.currentTime || 0,
          )
          setIsCasting(true)
          syncFromRemotePlayer()
        }

        cleanupListeners = () => {
          ctx.removeEventListener(
            framework.CastContextEventType.CAST_STATE_CHANGED,
            handleCastState,
          )
          ctx.removeEventListener(
            framework.CastContextEventType.SESSION_STATE_CHANGED,
            handleSessionState,
          )
          controller.removeEventListener(
            framework.RemotePlayerEventType.ANY_CHANGE,
            handleRemoteChange,
          )
        }
      } catch {
        setCastAvailable(false)
      }
    }

    if (!enabled) {
      return undefined
    }

    ensureCastSdkLoaded()
    initCastSdk()
    window.addEventListener('nd-cast-api-ready', initCastSdk)

    return () => {
      window.removeEventListener('nd-cast-api-ready', initCastSdk)
      if (seekReportTimerRef.current) {
        clearTimeout(seekReportTimerRef.current)
        seekReportTimerRef.current = null
      }
      uninstallAudioShim(audioInstanceRef.current, 0)
      if (cleanupListeners) {
        cleanupListeners()
      }
      castContextRef.current = null
      remotePlayerRef.current = null
      remoteControllerRef.current = null
      mediaSessionRef.current = null
      uuidToItemIdRef.current = new Map()
      loadedWindowUuidsRef.current = []
      activeUuidRef.current = null
      pendingCastLoadRef.current = false
      isCastingRef.current = false
      setCastAvailable(false)
      setIsCasting(false)
      setIsConnecting(false)
      setRemoteState('IDLE')
      setDeviceName('')
      setActiveTrackId(null)
      setActiveIsRadio(false)
    }
  }, [
    enabled,
    flushStoppedReport,
    getActiveCastTime,
    installAudioShim,
    loadQueueOnCast,
    recordMediaSessionItems,
    syncFromRemotePlayer,
    uninstallAudioShim,
  ])

  // If audioInstance mounts or changes after casting has already started, ensure it is shimmed
  useEffect(() => {
    if (isCasting && audioInstance) {
      installAudioShim(audioInstance, getActiveCastTime())
    }
  }, [isCasting, audioInstance, installAudioShim, getActiveCastTime])

  // Sync repeat/shuffle playMode changes from ReactJkMusicPlayer to Chromecast
  useEffect(() => {
    if (!isCasting) return
    const mediaSession = getActiveMediaSession()
    const { start, end } = loadedWindowRef.current
    const queueLen = (playerState.queue || []).length
    const repeatMode = resolveReceiverRepeatMode(
      window.chrome?.cast,
      playerState.mode,
      start,
      end,
      queueLen,
    )
    if (repeatMode && typeof mediaSession?.queueSetRepeatMode === 'function') {
      mediaSession.queueSetRepeatMode(
        repeatMode,
        () => {},
        () => {},
      )
    }
  }, [
    isCasting,
    playerState.mode,
    playerState.queue,
    playerState.queue.length,
    getActiveMediaSession,
  ])

  // Incremental Chromecast queue synchronization ("Play Next", "Add to Queue",
  // item deletions, drag-and-drop reorders, and explicit track jumps)
  useEffect(() => {
    if (!isCasting) return
    const queue = playerState.queue || []
    const plan = planCastQueueSync(loadedWindowUuidsRef.current, queue, {
      activeUuid: activeUuidRef.current,
      explicitPlayIndex: playerState.playIndex,
      savedPlayIndex: playerState.savedPlayIndex,
      maxItems: MAX_CAST_WINDOW_ITEMS,
    })

    if (plan.action === 'NONE') {
      if (playerState.playIndex != null) {
        const activeItem = queue[playerState.playIndex]
        if (activeItem) {
          dispatch(
            currentPlaying({
              ...activeItem,
              currentTime: getActiveCastTime(),
              duration: activeItem.duration || 0,
              volume: playerState.volume ?? 1,
            }),
          )
        }
      }
      return
    }

    const chromeCastApi = window.chrome?.cast
    const mediaSession = getActiveMediaSession()

    if (plan.action === 'STOP') {
      if (loadedWindowUuidsRef.current.length > 0 || activeUuidRef.current) {
        loadedWindowUuidsRef.current = []
        activeUuidRef.current = null
        uuidToItemIdRef.current.clear()
        flushStoppedReport()
        if (mediaSession?.stop && chromeCastApi?.media?.StopRequest) {
          mediaSession.stop(
            new chromeCastApi.media.StopRequest(),
            () => {},
            () => {},
          )
        }
      }
      return
    }

    if (plan.action === 'LOAD') {
      loadQueueOnCast(plan.targetIndex, plan.startSeconds)
      return
    }

    if (plan.action === 'JUMP') {
      const itemId = findCastItemId(mediaSession, plan.targetUuid)
      activeUuidRef.current = plan.targetUuid
      pendingCastLoadRef.current = true
      bridgeRef.current.virtualCurrentTime = 0
      bridgeRef.current.virtualPaused = false
      setActiveTrackId(plan.targetItem?.trackId || null)
      setActiveIsRadio(Boolean(plan.targetItem?.isRadio))
      dispatch(
        currentPlaying({
          ...plan.targetItem,
          currentTime: 0,
          duration: plan.targetItem?.duration || 0,
          volume: playerState.volume ?? 1,
        }),
      )
      if (
        itemId != null &&
        typeof mediaSession?.queueJumpToItem === 'function'
      ) {
        mediaSession.queueJumpToItem(
          itemId,
          () => {},
          () => loadQueueOnCast(plan.targetIndex, 0),
        )
      } else {
        loadQueueOnCast(plan.targetIndex, 0)
      }
      return
    }

    if (plan.action === 'INSERT') {
      const castItems = buildCastQueueItems(
        chromeCastApi,
        plan.insertedItems,
        window.location.origin,
      )
      const InsertReq = chromeCastApi?.media?.QueueInsertItemsRequest
      if (
        castItems.length > 0 &&
        typeof mediaSession?.queueInsertItems === 'function' &&
        typeof InsertReq === 'function'
      ) {
        if (plan.insertBeforeUuid === null) {
          const req = new InsertReq(castItems)
          loadedWindowUuidsRef.current = plan.nextWindowUuids
          loadedWindowRef.current = {
            start: plan.windowSpec.windowStartIndex,
            end: plan.windowSpec.windowEndIndex,
          }
          mediaSession.queueInsertItems(
            req,
            () => recordMediaSessionItems(getActiveMediaSession()),
            () => loadQueueOnCast(plan.activeIndex, getActiveCastTime()),
          )
          return
        }

        const beforeItemId = findCastItemId(mediaSession, plan.insertBeforeUuid)
        if (beforeItemId != null) {
          const req = new InsertReq(castItems)
          req.insertBefore = beforeItemId
          loadedWindowUuidsRef.current = plan.nextWindowUuids
          loadedWindowRef.current = {
            start: plan.windowSpec.windowStartIndex,
            end: plan.windowSpec.windowEndIndex,
          }
          mediaSession.queueInsertItems(
            req,
            () => recordMediaSessionItems(getActiveMediaSession()),
            () => loadQueueOnCast(plan.activeIndex, getActiveCastTime()),
          )
          return
        }
      }

      loadQueueOnCast(plan.activeIndex, getActiveCastTime())
      return
    }

    if (plan.action === 'REMOVE') {
      const RemoveReq = chromeCastApi?.media?.QueueRemoveItemsRequest
      const itemIds = plan.removedUuids
        .map((u) => findCastItemId(mediaSession, u))
        .filter((id) => id != null)
      if (
        itemIds.length === plan.removedUuids.length &&
        itemIds.length > 0 &&
        typeof mediaSession?.queueRemoveItems === 'function' &&
        typeof RemoveReq === 'function'
      ) {
        loadedWindowUuidsRef.current = plan.nextWindowUuids
        loadedWindowRef.current = {
          start: plan.windowSpec.windowStartIndex,
          end: plan.windowSpec.windowEndIndex,
        }
        mediaSession.queueRemoveItems(
          new RemoveReq(itemIds),
          () => {},
          () => loadQueueOnCast(plan.activeIndex, getActiveCastTime()),
        )
        return
      }

      loadQueueOnCast(plan.activeIndex, getActiveCastTime())
      return
    }

    if (plan.action === 'REORDER') {
      const ReorderReq = chromeCastApi?.media?.QueueReorderItemsRequest
      const itemIds = plan.orderedUuids
        .map((u) => findCastItemId(mediaSession, u))
        .filter((id) => id != null)
      if (
        itemIds.length === plan.orderedUuids.length &&
        itemIds.length > 0 &&
        typeof mediaSession?.queueReorderItems === 'function' &&
        typeof ReorderReq === 'function'
      ) {
        loadedWindowUuidsRef.current = plan.nextWindowUuids
        loadedWindowRef.current = {
          start: plan.windowSpec.windowStartIndex,
          end: plan.windowSpec.windowEndIndex,
        }
        mediaSession.queueReorderItems(
          new ReorderReq(itemIds),
          () => {},
          () => loadQueueOnCast(plan.activeIndex, getActiveCastTime()),
        )
        return
      }

      loadQueueOnCast(plan.activeIndex, getActiveCastTime())
      return
    }

    loadQueueOnCast(plan.activeIndex, getActiveCastTime())
  }, [
    dispatch,
    findCastItemId,
    flushStoppedReport,
    getActiveCastTime,
    getActiveMediaSession,
    isCasting,
    loadQueueOnCast,
    playerState.clear,
    playerState.playIndex,
    playerState.queue,
    playerState.queue.length,
    playerState.savedPlayIndex,
    playerState.volume,
    recordMediaSessionItems,
  ])

  // Forward Next / Prev / Queue Drawer track clicks from ReactJkMusicPlayer to Chromecast
  const onPlayIndexChange = useCallback(
    (playIndex) => {
      if (!isCastingRef.current) return
      const queue = playerStateRef.current?.queue || []
      const target = queue[playIndex]
      if (!target || target.uuid === activeUuidRef.current) {
        return
      }

      activeUuidRef.current = target.uuid
      pendingCastLoadRef.current = true
      bridgeRef.current.virtualCurrentTime = 0
      bridgeRef.current.virtualPaused = false
      setActiveTrackId(target.trackId || null)
      setActiveIsRadio(Boolean(target.isRadio))

      dispatch(
        currentPlaying({
          ...target,
          currentTime: 0,
          duration: target.duration || 0,
          volume: playerStateRef.current?.volume ?? 1,
        }),
      )

      const mediaSession = getActiveMediaSession()
      const itemId = findCastItemId(mediaSession, target.uuid)
      if (
        itemId != null &&
        typeof mediaSession?.queueJumpToItem === 'function'
      ) {
        mediaSession.queueJumpToItem(
          itemId,
          () => {},
          () => loadQueueOnCast(playIndex, 0),
        )
      } else {
        loadQueueOnCast(playIndex, 0)
      }
    },
    [dispatch, findCastItemId, getActiveMediaSession, loadQueueOnCast],
  )

  // Periodic OpenSubsonic reportPlayback heartbeat for Now Playing & Last.fm scrobbling
  useInterval(
    () => {
      if (
        isCasting &&
        remoteState === 'PLAYING' &&
        castReportedTrackRef.current &&
        !activeIsRadio
      ) {
        subsonic
          .reportPlayback(
            castReportedTrackRef.current,
            castLastPosMsRef.current,
            'playing',
          )
          .catch(() => {})
      }
    },
    isCasting && remoteState === 'PLAYING' && activeTrackId && !activeIsRadio
      ? config.playbackReportIntervalMs
      : null,
  )

  const requestSession = useCallback(() => {
    const ctx = castContextRef.current
    if (!ctx?.requestSession) return
    ctx.requestSession().catch(() => {})
  }, [])

  const stopCasting = useCallback(() => {
    const ctx = castContextRef.current
    flushStoppedReport()
    if (ctx?.endCurrentSession) {
      ctx.endCurrentSession(true)
    }
  }, [flushStoppedReport])

  const contextValue = useMemo(
    () => ({
      castAvailable,
      isCasting,
      isConnecting,
      deviceName,
      isCastingRef,
      requestSession,
      stopCasting,
      onPlayIndexChange,
    }),
    [
      castAvailable,
      deviceName,
      isCasting,
      isConnecting,
      onPlayIndexChange,
      requestSession,
      stopCasting,
    ],
  )

  return (
    <CastContext.Provider value={contextValue}>{children}</CastContext.Provider>
  )
}

export default CastProvider
