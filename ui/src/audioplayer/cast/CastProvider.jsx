import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useDispatch, useSelector } from 'react-redux'
import { useInterval } from '../../common'
import { currentPlaying } from '../../actions'
import config from '../../config'
import { httpClient } from '../../dataProvider'
import subsonic from '../../subsonic'
import { CastContext } from './CastContext'
import {
  CAST_SENDER_SDK_URL,
  MAX_CAST_WINDOW_ITEMS,
  buildCastQueueItems,
  buildCastQueueWindow,
  canLoadCastSdk,
  getCastStreamProfile,
  planCastQueueSync,
} from './castUtils'

const MIN_CAST_TOKEN_FRESHNESS_MS = 15 * 60 * 1000

const SHIMMED_AUDIO_PROPS = [
  'play',
  'pause',
  'load',
  'currentTime',
  'duration',
  'buffered',
  'volume',
  'muted',
  'paused',
  'readyState',
  'networkState',
  'src',
  'setAttribute',
  'removeAttribute',
]

const resolveEffectivePlayerState = (mediaSession, rp) => {
  const mediaState = mediaSession?.playerState
  const rpState = rp?.playerState
  if (mediaState && mediaState !== 'IDLE') {
    return mediaState
  }
  if (rpState && rpState !== 'IDLE') {
    return rpState
  }
  return mediaState || rpState || 'IDLE'
}

const resolveActiveItemId = (mediaSession) => {
  if (!mediaSession) return null
  return mediaSession.loadingItemId ?? mediaSession.currentItemId ?? null
}

const resolveActiveMediaInfo = (mediaSession, rp, supersededUuids = null) => {
  const isSuperseded = (uuid) =>
    Boolean(uuid && supersededUuids?.size > 0 && supersededUuids.has(uuid))
  const activeItemId = resolveActiveItemId(mediaSession)
  if (activeItemId != null && Array.isArray(mediaSession?.items)) {
    const queueItem = mediaSession.items.find(
      (it) => it?.itemId === activeItemId,
    )
    if (queueItem?.media) {
      return queueItem.media
    }
  }
  const isTransitioningToNextItem =
    mediaSession?.loadingItemId != null &&
    mediaSession?.currentItemId !== mediaSession?.loadingItemId
  if (
    !isTransitioningToNextItem &&
    mediaSession?.media &&
    !isSuperseded(mediaSession.media?.customData?.uuid)
  ) {
    return mediaSession.media
  }
  const rpMedia = rp?.mediaInfo || null
  if (isSuperseded(rpMedia?.customData?.uuid)) {
    return null
  }
  return rpMedia
}

const stopMediaSession = (media, chromeCastApi) => {
  if (!media || typeof media.stop !== 'function') {
    return
  }
  const StopReq = chromeCastApi?.media?.StopRequest
  try {
    media.stop(
      typeof StopReq === 'function' ? new StopReq() : null,
      () => {},
      () => {},
    )
  } catch {
    // Ignore teardown errors on closed media sessions
  }
}

const resolveQueueItemDuration = (item) =>
  Number(item?.duration || item?.song?.duration || 0)

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
  const lastSyncedUiUuidRef = useRef(null)
  const pendingCastLoadRef = useRef(false)
  const queueLoadInFlightRef = useRef(false)
  const supersededMediaRef = useRef(null)
  const supersededUuidsRef = useRef(new Set())
  const pendingJumpRef = useRef(null)
  const receiverUuidRef = useRef(null)
  const receiverRepeatModeRef = useRef(null)
  const jumpWatchdogTimerRef = useRef(null)
  const castReportedTrackRef = useRef(null)
  const castReportedStateRef = useRef('IDLE')
  const castLastPosMsRef = useRef(0)
  const seekReportTimerRef = useRef(null)
  const castTokenCacheRef = useRef(new Map())
  const castOpSeqRef = useRef(0)
  const syncFromRemotePlayerRef = useRef(() => {})
  const attachedMediaRef = useRef({ media: null, listener: null })
  const attachedSessionObjRef = useRef({ sessionObj: null, listener: null })
  const castClockRef = useRef({
    baseTimeSec: 0,
    sampledAtMs: 0,
    playbackRate: 1,
    isPlaying: false,
  })

  const clearJumpWatchdog = useCallback(() => {
    if (jumpWatchdogTimerRef.current) {
      clearTimeout(jumpWatchdogTimerRef.current)
      jumpWatchdogTimerRef.current = null
    }
  }, [])

  const bridgeRef = useRef({
    shimmedAudio: null,
    savedDescriptors: {},
    virtualCurrentTime: 0,
    virtualDuration: 0,
    virtualVolume: 1,
    virtualMuted: false,
    virtualPaused: true,
    virtualSrc: '',
    trackChangedWhileCasting: false,
    seekingUntil: 0,
    lastSeekSentAt: 0,
    seekThrottleTimer: null,
    volumeDebounceTimer: null,
    readyTimer: null,
    suppressAutoResumeOnLoad: false,
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

  const detachMediaSession = useCallback(() => {
    const prev = attachedMediaRef.current
    if (
      prev?.media &&
      prev?.listener &&
      typeof prev.media.removeUpdateListener === 'function'
    ) {
      try {
        prev.media.removeUpdateListener(prev.listener)
      } catch {
        // Ignore teardown errors on closed media sessions
      }
    }
    attachedMediaRef.current = { media: null, listener: null }
  }, [])

  const attachMediaSession = useCallback(
    (media) => {
      if (!media) return
      mediaSessionRef.current = media
      recordMediaSessionItems(media)
      if (attachedMediaRef.current?.media === media) {
        return
      }
      detachMediaSession()
      if (typeof media.addUpdateListener === 'function') {
        const listener = () => {
          recordMediaSessionItems(media)
          syncFromRemotePlayerRef.current()
        }
        try {
          media.addUpdateListener(listener)
          attachedMediaRef.current = { media, listener }
        } catch {
          // Ignore non-standard media mock errors
        }
      }
    },
    [detachMediaSession, recordMediaSessionItems],
  )

  const detachSessionObj = useCallback(() => {
    const prev = attachedSessionObjRef.current
    if (
      prev?.sessionObj &&
      prev?.listener &&
      typeof prev.sessionObj.removeMediaListener === 'function'
    ) {
      try {
        prev.sessionObj.removeMediaListener(prev.listener)
      } catch {
        // Ignore teardown errors on ended Cast sessions
      }
    }
    attachedSessionObjRef.current = { sessionObj: null, listener: null }
  }, [])

  const attachSessionObj = useCallback(
    (sessionObj) => {
      if (
        !sessionObj ||
        attachedSessionObjRef.current?.sessionObj === sessionObj
      ) {
        return
      }
      detachSessionObj()
      if (typeof sessionObj.addMediaListener === 'function') {
        const listener = (media) => {
          if (!isCastingRef.current) return
          if ((playerStateRef.current?.queue || []).length === 0) {
            stopMediaSession(media, window.chrome?.cast)
            return
          }
          queueLoadInFlightRef.current = false
          supersededMediaRef.current = null
          attachMediaSession(media)
          syncFromRemotePlayerRef.current()
        }
        try {
          sessionObj.addMediaListener(listener)
          attachedSessionObjRef.current = { sessionObj, listener }
        } catch {
          // Ignore non-standard session mock errors
        }
      }
    },
    [attachMediaSession, detachSessionObj],
  )

  const getActiveMediaSession = useCallback(() => {
    const session = castContextRef.current?.getCurrentSession?.()
    const sessionObj = session?.getSessionObj?.()
    if (sessionObj) {
      attachSessionObj(sessionObj)
    }
    if (queueLoadInFlightRef.current) {
      return null
    }
    const superseded = supersededMediaRef.current
    const mediaList = Array.isArray(sessionObj?.media)
      ? sessionObj.media.filter((m) => m && m !== superseded)
      : []
    const liveFromList = mediaList.find(
      (m) =>
        m.playerState !== 'IDLE' || m.loadingItemId != null || !m.idleReason,
    )
    const sessionMedia = session?.getMediaSession?.()
    const validSessionMedia =
      sessionMedia && sessionMedia !== superseded ? sessionMedia : null
    const currentRefMedia =
      mediaSessionRef.current && mediaSessionRef.current !== superseded
        ? mediaSessionRef.current
        : null
    const media =
      liveFromList || validSessionMedia || mediaList[0] || currentRefMedia
    if (media) {
      attachMediaSession(media)
    }
    return media
  }, [attachMediaSession, attachSessionObj])

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
    const mediaSession = mediaSessionRef.current
    if (typeof mediaSession?.getEstimatedTime === 'function') {
      const est = Number(mediaSession.getEstimatedTime())
      if (Number.isFinite(est) && est > 0) {
        return Math.floor(est)
      }
    }
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

  const seekCastTo = useCallback(
    (seconds) => {
      const bridge = bridgeRef.current
      const queue = playerStateRef.current?.queue || []
      const currentItem = queue.find((it) => it.uuid === activeUuidRef.current)
      if (currentItem?.isRadio) return

      const maxDuration = Number(
        bridge.virtualDuration ||
          remotePlayerRef.current?.duration ||
          currentItem?.duration ||
          0,
      )
      const raw = Number.isFinite(Number(seconds)) ? Number(seconds) : 0
      const clamped =
        maxDuration > 0
          ? Math.max(0, Math.min(raw, maxDuration))
          : Math.max(0, raw)

      bridge.virtualCurrentTime = clamped
      bridge.seekingUntil = Date.now() + 800
      castLastPosMsRef.current = Math.floor(clamped * 1000)
      castClockRef.current = {
        ...castClockRef.current,
        baseTimeSec: clamped,
        sampledAtMs: Date.now(),
      }
      dispatchMediaEvent(audioInstanceRef.current, 'timeupdate')

      const sendSeek = (targetSec) => {
        const mediaSession = getActiveMediaSession()
        const SeekReq = window.chrome?.cast?.media?.SeekRequest
        if (
          typeof mediaSession?.seek === 'function' &&
          typeof SeekReq === 'function'
        ) {
          const req = new SeekReq()
          req.currentTime = targetSec
          mediaSession.seek(
            req,
            () => syncFromRemotePlayerRef.current(),
            () => {},
          )
          bridge.lastSeekSentAt = Date.now()
          return
        }
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
          const effectiveState = resolveEffectivePlayerState(
            getActiveMediaSession(),
            remotePlayerRef.current,
          )
          const stateStr = effectiveState === 'PAUSED' ? 'paused' : 'playing'
          subsonic
            .reportPlayback(
              castReportedTrackRef.current,
              Math.floor(bridge.virtualCurrentTime * 1000),
              stateStr,
            )
            .catch(() => {})
        }
      }, 250)
    },
    [getActiveMediaSession],
  )

  const fetchCastMediaTokens = useCallback(async (items = []) => {
    const nowMs = Date.now()
    const cache = castTokenCacheRef.current
    const resolved = new Map()
    const pendingPayload = []
    const seenKeys = new Set()

    items.forEach((item) => {
      const profile = getCastStreamProfile(item)
      if (!profile || !profile.id || !profile.key) {
        return
      }
      const cached = cache.get(profile.key)
      if (cached && cached.expiresAtMs - nowMs > MIN_CAST_TOKEN_FRESHNESS_MS) {
        resolved.set(profile.key, cached)
        return
      }
      if (!seenKeys.has(profile.key)) {
        seenKeys.add(profile.key)
        pendingPayload.push({
          key: profile.key,
          id: profile.id,
          isRadio: profile.isRadio,
          format: profile.format,
          maxBitRate: profile.maxBitRate,
        })
      }
    })

    if (pendingPayload.length === 0) {
      return resolved
    }

    let response
    try {
      response = await httpClient('/api/cast/tokens', {
        method: 'POST',
        body: JSON.stringify({ items: pendingPayload }),
      })
    } catch {
      response = await httpClient('/api/cast/tokens', {
        method: 'POST',
        body: JSON.stringify({ items: pendingPayload }),
      })
    }
    const tokens = response?.json?.tokens || {}
    Object.entries(tokens).forEach(([key, entry]) => {
      if (!entry) return
      const expiresAtMs =
        typeof entry.expiresAt === 'number' && entry.expiresAt > 0
          ? entry.expiresAt * 1000
          : Date.now() + 60 * 60 * 1000
      const normalized = {
        streamToken: entry.streamToken || '',
        coverToken: entry.coverToken || '',
        expiresAtMs,
      }
      cache.set(key, normalized)
      resolved.set(key, normalized)
    })
    return resolved
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
      if (windowSpec.items.length === 0) {
        return
      }

      castOpSeqRef.current += 1
      const opSeq = castOpSeqRef.current

      clearJumpWatchdog()
      pendingJumpRef.current = null

      const staleCandidates = [
        activeUuidRef.current,
        receiverUuidRef.current,
        remotePlayerRef.current?.mediaInfo?.customData?.uuid,
        state?.current?.uuid,
      ]
      supersededMediaRef.current =
        mediaSessionRef.current ||
        (Array.isArray(sessionObj.media) ? sessionObj.media[0] : null) ||
        null

      uuidToItemIdRef.current.clear()
      detachMediaSession()
      mediaSessionRef.current = null
      loadedWindowUuidsRef.current = windowSpec.items.map((item) => item.uuid)
      loadedWindowRef.current = {
        start: windowSpec.windowStartIndex,
        end: windowSpec.windowEndIndex,
      }
      queueLoadInFlightRef.current = true
      pendingCastLoadRef.current = true

      const initialStartIdx = Math.min(
        windowSpec.startOffset,
        windowSpec.items.length - 1,
      )
      const targetItem = windowSpec.items[initialStartIdx]
      supersededUuidsRef.current = new Set(
        staleCandidates.filter((u) => u && u !== targetItem?.uuid),
      )

      if (targetItem) {
        const itemDuration = resolveQueueItemDuration(targetItem)
        activeUuidRef.current = targetItem.uuid
        setActiveTrackId(targetItem.trackId || null)
        setActiveIsRadio(Boolean(targetItem.isRadio))
        bridgeRef.current.virtualCurrentTime = startSeconds
        bridgeRef.current.virtualDuration = itemDuration
        bridgeRef.current.virtualPaused = false
        castLastPosMsRef.current = Math.floor(startSeconds * 1000)
        castClockRef.current = {
          baseTimeSec: startSeconds,
          sampledAtMs: Date.now(),
          playbackRate: 1,
          isPlaying: false,
        }
        dispatchMediaEvent(audioInstanceRef.current, 'timeupdate')

        if (state.current?.uuid === targetItem.uuid) {
          lastSyncedUiUuidRef.current = targetItem.uuid
        }

        if (
          state.current?.uuid !== targetItem.uuid ||
          state.playIndex != null
        ) {
          dispatch(
            currentPlaying({
              ...targetItem,
              currentTime: startSeconds,
              duration: itemDuration,
              volume: state.volume ?? 1,
            }),
          )
        }

        if (typeof audioInstanceRef.current?.updatePlayIndex === 'function') {
          lastSyncedUiUuidRef.current = targetItem.uuid
          audioInstanceRef.current.updatePlayIndex(
            windowSpec.windowStartIndex + initialStartIdx,
          )
        } else if (state.playIndex != null) {
          lastSyncedUiUuidRef.current = targetItem.uuid
        }
      }

      fetchCastMediaTokens(windowSpec.items)
        .then((tokenMap) => {
          if (!isCastingRef.current || castOpSeqRef.current !== opSeq) {
            return
          }
          const activeSessionObj = castContextRef.current
            ?.getCurrentSession?.()
            ?.getSessionObj?.()
          if (!activeSessionObj) {
            queueLoadInFlightRef.current = false
            supersededMediaRef.current = null
            supersededUuidsRef.current.clear()
            pendingCastLoadRef.current = false
            return
          }

          const queueItems = buildCastQueueItems(
            chromeCastApi,
            windowSpec.items,
            window.location.origin,
            tokenMap,
          )
          if (queueItems.length === 0) {
            queueLoadInFlightRef.current = false
            supersededMediaRef.current = null
            supersededUuidsRef.current.clear()
            pendingCastLoadRef.current = false
            return
          }

          loadedWindowUuidsRef.current = queueItems
            .map((qi) => qi.media?.customData?.uuid)
            .filter(Boolean)

          const targetQueueIdx = targetItem
            ? queueItems.findIndex(
                (qi) => qi.media?.customData?.uuid === targetItem.uuid,
              )
            : -1
          const request = new chromeCastApi.media.QueueLoadRequest(queueItems)
          request.startIndex = targetQueueIdx >= 0 ? targetQueueIdx : 0
          const selectedSourceItem = windowSpec.items.find(
            (it) =>
              it.uuid ===
              queueItems[request.startIndex]?.media?.customData?.uuid,
          )
          if (startSeconds > 0 && !selectedSourceItem?.isRadio) {
            request.currentTime = startSeconds
          }
          const repeatMode = resolveReceiverRepeatMode(
            chromeCastApi,
            playerStateRef.current?.mode,
            windowSpec.windowStartIndex,
            windowSpec.windowEndIndex,
            (playerStateRef.current?.queue || []).length,
          )
          receiverRepeatModeRef.current = repeatMode || null
          if (repeatMode) {
            request.repeatMode = repeatMode
          }

          activeSessionObj.queueLoad(
            request,
            (media) => {
              if (
                isCastingRef.current &&
                (playerStateRef.current?.queue || []).length === 0
              ) {
                stopMediaSession(media, chromeCastApi)
                return
              }
              if (castOpSeqRef.current !== opSeq) return
              queueLoadInFlightRef.current = false
              supersededMediaRef.current = null
              if (media) {
                attachMediaSession(media)
                syncFromRemotePlayerRef.current()
              }
            },
            () => {
              if (castOpSeqRef.current === opSeq) {
                queueLoadInFlightRef.current = false
                supersededMediaRef.current = null
                supersededUuidsRef.current.clear()
                pendingCastLoadRef.current = false
              }
            },
          )
        })
        .catch(() => {
          if (castOpSeqRef.current === opSeq) {
            queueLoadInFlightRef.current = false
            supersededMediaRef.current = null
            supersededUuidsRef.current.clear()
            pendingCastLoadRef.current = false
          }
        })
    },
    [
      attachMediaSession,
      clearJumpWatchdog,
      detachMediaSession,
      dispatch,
      fetchCastMediaTokens,
    ],
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
    if (bridge.readyTimer) {
      clearTimeout(bridge.readyTimer)
      bridge.readyTimer = null
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
    bridge.virtualDuration = 0
    bridge.suppressAutoResumeOnLoad = false
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

      const pStateForInit = playerStateRef.current
      const initQueue = pStateForInit?.queue || []
      const initQueueItem =
        initQueue.find((it) => it.uuid === activeUuidRef.current) ||
        pStateForInit?.current ||
        initQueue[
          pStateForInit?.playIndex ?? pStateForInit?.savedPlayIndex ?? 0
        ]
      const initialDuration =
        resolveQueueItemDuration(initQueueItem) ||
        (typeof audio.duration === 'number' &&
        Number.isFinite(audio.duration) &&
        audio.duration > 0
          ? audio.duration
          : 0)

      bridge.shimmedAudio = audio
      bridge.savedDescriptors = savedDescriptors
      bridge.virtualCurrentTime = Number.isFinite(initialTimeSec)
        ? initialTimeSec
        : 0
      bridge.virtualDuration = initialDuration
      bridge.virtualVolume = initialVol
      bridge.virtualMuted = initialMuted
      bridge.virtualPaused = false
      bridge.virtualSrc = initialSrc
      bridge.suppressAutoResumeOnLoad = false
      bridge.trackChangedWhileCasting = false
      bridge.seekingUntil = 0
      bridge.nativePause = nativePause
      bridge.nativeSetAttribute = nativeSetAttribute
      bridge.nativeRemoveAttribute = nativeRemoveAttribute

      const scheduleVirtualReady = () => {
        if (bridge.readyTimer) {
          clearTimeout(bridge.readyTimer)
        }
        bridge.readyTimer = setTimeout(() => {
          bridge.readyTimer = null
          if (!isCastingRef.current || bridge.shimmedAudio !== audio) {
            return
          }
          dispatchMediaEvent(audio, 'loadedmetadata')
          dispatchMediaEvent(audio, 'canplay')
          bridge.suppressAutoResumeOnLoad = false
          const effState = resolveEffectivePlayerState(
            getActiveMediaSession(),
            remotePlayerRef.current,
          )
          if (effState === 'PAUSED') {
            bridge.virtualPaused = true
            dispatchMediaEvent(audio, 'pause')
          } else if (effState === 'PLAYING' || !bridge.virtualPaused) {
            dispatchMediaEvent(audio, 'play')
          }
          dispatchMediaEvent(audio, 'timeupdate')
        }, 0)
      }

      Object.defineProperties(audio, {
        play: {
          configurable: true,
          value: () => {
            if (bridge.suppressAutoResumeOnLoad) {
              bridge.suppressAutoResumeOnLoad = false
              bridge.virtualPaused = true
              setTimeout(() => {
                if (
                  isCastingRef.current &&
                  bridge.shimmedAudio === audio &&
                  bridge.virtualPaused
                ) {
                  dispatchMediaEvent(audio, 'pause')
                }
              }, 0)
              return Promise.resolve()
            }
            bridge.virtualPaused = false
            if (!pendingCastLoadRef.current) {
              const mediaSession = getActiveMediaSession()
              const rp = remotePlayerRef.current
              const controller = remoteControllerRef.current
              const effState = resolveEffectivePlayerState(mediaSession, rp)
              if (effState === 'PAUSED') {
                const PlayReq = window.chrome?.cast?.media?.PlayRequest
                if (typeof mediaSession?.play === 'function') {
                  mediaSession.play(
                    typeof PlayReq === 'function' ? new PlayReq() : null,
                    () => syncFromRemotePlayerRef.current(),
                    () => {},
                  )
                } else if (controller?.playOrPause) {
                  controller.playOrPause()
                }
              } else if (
                effState === 'IDLE' &&
                mediaSession?.loadingItemId == null
              ) {
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
            const mediaSession = getActiveMediaSession()
            const rp = remotePlayerRef.current
            const controller = remoteControllerRef.current
            const effState = resolveEffectivePlayerState(mediaSession, rp)
            if (effState === 'PLAYING' || effState === 'BUFFERING') {
              const PauseReq = window.chrome?.cast?.media?.PauseRequest
              if (typeof mediaSession?.pause === 'function') {
                mediaSession.pause(
                  typeof PauseReq === 'function' ? new PauseReq() : null,
                  () => syncFromRemotePlayerRef.current(),
                  () => {},
                )
              } else if (controller?.playOrPause) {
                controller.playOrPause()
              }
            }
            dispatchMediaEvent(audio, 'pause')
          },
        },
        load: {
          configurable: true,
          value: () => {
            scheduleVirtualReady()
          },
        },
        currentTime: {
          configurable: true,
          get: () => bridge.virtualCurrentTime,
          set: (val) => {
            seekCastTo(val)
          },
        },
        duration: {
          configurable: true,
          get: () => bridge.virtualDuration,
        },
        buffered: {
          configurable: true,
          get: () => ({
            length: bridge.virtualDuration > 0 ? 1 : 0,
            start: () => 0,
            end: () => bridge.virtualDuration,
          }),
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
            const nextSrc = String(val ?? '')
            bridge.virtualSrc = nextSrc
            bridge.trackChangedWhileCasting = true
            if (nextSrc) {
              scheduleVirtualReady()
            }
          },
        },
        setAttribute: {
          configurable: true,
          value: (name, val) => {
            if (String(name).toLowerCase() === 'src') {
              const nextSrc = String(val ?? '')
              bridge.virtualSrc = nextSrc
              bridge.trackChangedWhileCasting = true
              if (nextSrc) {
                scheduleVirtualReady()
              }
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

      scheduleVirtualReady()
    },
    [getActiveMediaSession, loadQueueOnCast, seekCastTo, uninstallAudioShim],
  )

  const syncFromRemotePlayer = useCallback(() => {
    const rp = remotePlayerRef.current
    const ctx = castContextRef.current
    if (!rp && !ctx) return

    const session = ctx?.getCurrentSession?.()
    const connected = Boolean(
      rp?.isConnected || (isCastingRef.current && session),
    )
    const castDeviceName =
      session?.getCastDevice?.()?.friendlyName ||
      session?.getSessionObj?.()?.receiver?.friendlyName ||
      ''

    setDeviceName(castDeviceName)

    if (!connected || queueLoadInFlightRef.current) {
      return
    }

    const queue = playerStateRef.current?.queue || []
    if (queue.length === 0) {
      return
    }

    const mediaSession = getActiveMediaSession()
    const nextState = resolveEffectivePlayerState(mediaSession, rp)
    const mediaInfo = resolveActiveMediaInfo(
      mediaSession,
      rp,
      supersededUuidsRef.current,
    )
    const activeItemId = resolveActiveItemId(mediaSession)

    const isTransitioningItem =
      mediaSession?.loadingItemId != null &&
      (mediaSession?.playerState === 'IDLE' ||
        mediaSession?.currentItemId !== mediaSession?.loadingItemId)
    let nextTime = 0
    if (!isTransitioningItem) {
      if (typeof mediaSession?.getEstimatedTime === 'function') {
        const est = Number(mediaSession.getEstimatedTime())
        nextTime = Number.isFinite(est) && est >= 0 ? est : 0
      } else if (
        mediaSession?.playerState &&
        mediaSession.playerState !== 'IDLE' &&
        Number.isFinite(mediaSession?.currentTime)
      ) {
        nextTime = Math.max(0, Number(mediaSession.currentTime))
      } else if (Number.isFinite(rp?.currentTime)) {
        nextTime = Math.max(0, Number(rp.currentTime))
      }
    }

    const mediaInfoDuration = Number(mediaInfo?.duration)
    const nextDuration =
      Number.isFinite(mediaInfoDuration) && mediaInfoDuration > 0
        ? mediaInfoDuration
        : Number.isFinite(rp?.duration) && rp.duration > 0
          ? rp.duration
          : 0
    const audio = audioInstanceRef.current
    const bridge = bridgeRef.current

    if (
      rp &&
      typeof rp.volumeLevel === 'number' &&
      Number.isFinite(rp.volumeLevel)
    ) {
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

    const idleReasons = window.chrome?.cast?.media?.IdleReason
    const isTerminalIdleFailure =
      nextState === 'IDLE' &&
      mediaSession?.loadingItemId == null &&
      ((idleReasons?.ERROR && mediaSession?.idleReason === idleReasons.ERROR) ||
        mediaSession?.idleReason === 'ERROR' ||
        (idleReasons?.CANCELLED &&
          mediaSession?.idleReason === idleReasons.CANCELLED) ||
        mediaSession?.idleReason === 'CANCELLED')

    if (isTerminalIdleFailure) {
      clearJumpWatchdog()
      pendingJumpRef.current = null
      supersededUuidsRef.current.clear()
      pendingCastLoadRef.current = false
    }

    const customData = mediaInfo?.customData || {}

    let matchedIdx = -1
    if (customData.uuid) {
      matchedIdx = queue.findIndex((item) => item.uuid === customData.uuid)
    } else {
      if (activeItemId != null) {
        for (const [uuid, mappedId] of uuidToItemIdRef.current.entries()) {
          if (mappedId === activeItemId) {
            matchedIdx = queue.findIndex((item) => item.uuid === uuid)
            if (matchedIdx >= 0) break
          }
        }
      }
      if (matchedIdx < 0 && customData.trackId) {
        matchedIdx = queue.findIndex(
          (item) => item.trackId === customData.trackId,
        )
      }
    }
    const matched = matchedIdx >= 0 ? queue[matchedIdx] : null

    if (!isTerminalIdleFailure) {
      if (supersededUuidsRef.current.size > 0) {
        if (!matched || supersededUuidsRef.current.has(matched.uuid)) {
          return
        }
        supersededUuidsRef.current.clear()
      }

      const pendingJump = pendingJumpRef.current
      if (pendingJump) {
        if (Date.now() >= pendingJump.expiresAtMs) {
          clearJumpWatchdog()
          pendingJumpRef.current = null
        } else if (
          !matched ||
          (pendingJump.staleUuid && matched.uuid === pendingJump.staleUuid)
        ) {
          return
        } else {
          clearJumpWatchdog()
          if (
            mediaSession?.loadingItemId == null &&
            (nextState === 'PLAYING' || nextState === 'PAUSED')
          ) {
            pendingJumpRef.current = null
            pendingCastLoadRef.current = false
          }
        }
      }
    }

    if (
      !pendingJumpRef.current &&
      supersededUuidsRef.current.size === 0 &&
      mediaSession?.loadingItemId == null &&
      (nextState === 'PLAYING' ||
        nextState === 'PAUSED' ||
        isTerminalIdleFailure)
    ) {
      pendingCastLoadRef.current = false
    }

    setRemoteState(nextState)

    if (matched && mediaSession?.loadingItemId == null) {
      receiverUuidRef.current = matched.uuid
    }
    const resolvedDuration =
      nextDuration || resolveQueueItemDuration(matched) || 0

    if (resolvedDuration > 0 && bridge.virtualDuration !== resolvedDuration) {
      bridge.virtualDuration = resolvedDuration
      dispatchMediaEvent(audio, 'durationchange')
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
        castClockRef.current = {
          baseTimeSec: nextTime,
          sampledAtMs: Date.now(),
          playbackRate:
            Number(mediaSession?.playbackRate) > 0
              ? Number(mediaSession.playbackRate)
              : 1,
          isPlaying: nextState === 'PLAYING',
        }
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

    if (matched) {
      if (activeUuidRef.current !== matched.uuid) {
        activeUuidRef.current = matched.uuid
        setActiveTrackId(matched.trackId || null)
        setActiveIsRadio(Boolean(matched.isRadio))
        dispatch(
          currentPlaying({
            ...matched,
            currentTime: nextTime,
            duration: resolvedDuration,
            volume: playerStateRef.current?.volume ?? 1,
          }),
        )
      }
      if (
        lastSyncedUiUuidRef.current !== matched.uuid &&
        audio &&
        typeof audio.updatePlayIndex === 'function'
      ) {
        lastSyncedUiUuidRef.current = matched.uuid
        bridge.suppressAutoResumeOnLoad = nextState === 'PAUSED'
        audio.updatePlayIndex(matchedIdx)
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
      mediaSession?.loadingItemId == null &&
      (mediaSession?.idleReason ===
        window.chrome?.cast?.media?.IdleReason?.FINISHED ||
        mediaSession?.idleReason === 'FINISHED')
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
  }, [clearJumpWatchdog, dispatch, getActiveMediaSession, loadQueueOnCast])

  syncFromRemotePlayerRef.current = syncFromRemotePlayer

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
            attachSessionObj(ctx.getCurrentSession?.()?.getSessionObj?.())
            installAudioShim(audio, resumeSec)
            setIsCasting(true)
            setIsConnecting(false)

            const pState = playerStateRef.current
            lastSyncedUiUuidRef.current = pState?.current?.uuid || null
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
            attachSessionObj(ctx.getCurrentSession?.()?.getSessionObj?.())
            const existingMedia = getActiveMediaSession()
            const existingTime =
              typeof existingMedia?.getEstimatedTime === 'function'
                ? Math.floor(Number(existingMedia.getEstimatedTime()) || 0)
                : 0
            const resumeSec =
              existingTime ||
              remotePlayerRef.current?.currentTime ||
              (audio && Number.isFinite(audio.currentTime)
                ? Math.floor(audio.currentTime)
                : 0)
            isCastingRef.current = true
            lastSyncedUiUuidRef.current =
              playerStateRef.current?.current?.uuid || null
            installAudioShim(audio, resumeSec)
            setIsCasting(true)
            setIsConnecting(false)

            const pState = playerStateRef.current
            if (
              !existingMedia ||
              (existingMedia.playerState === 'IDLE' &&
                existingMedia.loadingItemId == null)
            ) {
              const startIdx =
                pState.playIndex ??
                (pState.savedPlayIndex >= 0 ? pState.savedPlayIndex : 0)
              loadQueueOnCast(startIdx, 0)
            } else {
              attachMediaSession(existingMedia)
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
            castOpSeqRef.current += 1
            clearJumpWatchdog()
            const finalSec = getActiveCastTime()
            const endedUuid = activeUuidRef.current
            flushStoppedReport()
            queueLoadInFlightRef.current = false
            supersededMediaRef.current = null
            supersededUuidsRef.current.clear()
            pendingJumpRef.current = null
            pendingCastLoadRef.current = false
            detachMediaSession()
            detachSessionObj()
            mediaSessionRef.current = null
            uuidToItemIdRef.current.clear()
            loadedWindowUuidsRef.current = []
            loadedWindowRef.current = { start: 0, end: 0 }
            activeUuidRef.current = null
            receiverUuidRef.current = null
            receiverRepeatModeRef.current = null
            lastSyncedUiUuidRef.current = null
            castClockRef.current = {
              baseTimeSec: 0,
              sampledAtMs: 0,
              playbackRate: 1,
              isPlaying: false,
            }

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
          lastSyncedUiUuidRef.current =
            playerStateRef.current?.current?.uuid || null
          attachSessionObj(existingSession.getSessionObj?.())
          const activeMedia = getActiveMediaSession()
          const initSec =
            typeof activeMedia?.getEstimatedTime === 'function'
              ? Math.floor(Number(activeMedia.getEstimatedTime()) || 0)
              : remotePlayerRef.current?.currentTime || 0
          installAudioShim(audioInstanceRef.current, initSec)
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
      castOpSeqRef.current += 1
      clearJumpWatchdog()
      castTokenCacheRef.current = new Map()
      window.removeEventListener('nd-cast-api-ready', initCastSdk)
      if (seekReportTimerRef.current) {
        clearTimeout(seekReportTimerRef.current)
        seekReportTimerRef.current = null
      }
      flushStoppedReport()
      detachMediaSession()
      detachSessionObj()
      uninstallAudioShim(audioInstanceRef.current, 0)
      if (cleanupListeners) {
        cleanupListeners()
      }
      const activeContext = castContextRef.current
      if (isCastingRef.current && activeContext?.endCurrentSession) {
        try {
          activeContext.endCurrentSession(true)
        } catch {
          // Ignore teardown failures on an already-closed session
        }
      }
      castContextRef.current = null
      remotePlayerRef.current = null
      remoteControllerRef.current = null
      mediaSessionRef.current = null
      uuidToItemIdRef.current = new Map()
      loadedWindowUuidsRef.current = []
      loadedWindowRef.current = { start: 0, end: 0 }
      activeUuidRef.current = null
      receiverUuidRef.current = null
      receiverRepeatModeRef.current = null
      lastSyncedUiUuidRef.current = null
      castClockRef.current = {
        baseTimeSec: 0,
        sampledAtMs: 0,
        playbackRate: 1,
        isPlaying: false,
      }
      queueLoadInFlightRef.current = false
      supersededMediaRef.current = null
      supersededUuidsRef.current.clear()
      pendingJumpRef.current = null
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
    attachMediaSession,
    attachSessionObj,
    clearJumpWatchdog,
    detachMediaSession,
    detachSessionObj,
    enabled,
    flushStoppedReport,
    getActiveCastTime,
    getActiveMediaSession,
    installAudioShim,
    loadQueueOnCast,
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
    receiverRepeatModeRef.current = repeatMode || null
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

  const jumpCastToTrack = useCallback(
    (targetIndex, targetItem) => {
      if (!isCastingRef.current || !targetItem) {
        return
      }
      if (targetItem.uuid === activeUuidRef.current) {
        return
      }

      const windowUuids = loadedWindowUuidsRef.current
      if (
        !Array.isArray(windowUuids) ||
        windowUuids.length === 0 ||
        !windowUuids.includes(targetItem.uuid) ||
        queueLoadInFlightRef.current
      ) {
        loadQueueOnCast(targetIndex, 0)
        return
      }

      const prevActiveUuid = activeUuidRef.current
      const wasPendingLoad =
        pendingCastLoadRef.current || Boolean(pendingJumpRef.current)

      clearJumpWatchdog()
      castOpSeqRef.current += 1
      const opSeq = castOpSeqRef.current

      const rawStaleUuid = receiverUuidRef.current || prevActiveUuid || null
      const staleUuid =
        rawStaleUuid && rawStaleUuid !== targetItem.uuid ? rawStaleUuid : null

      pendingJumpRef.current = {
        targetUuid: targetItem.uuid,
        staleUuid,
        targetIndex,
        opSeq,
        expiresAtMs: Date.now() + 8000,
      }
      activeUuidRef.current = targetItem.uuid
      lastSyncedUiUuidRef.current = targetItem.uuid
      pendingCastLoadRef.current = true

      const targetDuration = resolveQueueItemDuration(targetItem)
      bridgeRef.current.virtualCurrentTime = 0
      bridgeRef.current.virtualDuration = targetDuration
      bridgeRef.current.virtualPaused = false
      castLastPosMsRef.current = 0
      castClockRef.current = {
        baseTimeSec: 0,
        sampledAtMs: Date.now(),
        playbackRate: 1,
        isPlaying: false,
      }
      dispatchMediaEvent(audioInstanceRef.current, 'timeupdate')
      setActiveTrackId(targetItem.trackId || null)
      setActiveIsRadio(Boolean(targetItem.isRadio))

      dispatch(
        currentPlaying({
          ...targetItem,
          currentTime: 0,
          duration: targetDuration,
          volume: playerStateRef.current?.volume ?? 1,
        }),
      )

      const mediaSession = getActiveMediaSession()
      if (!mediaSession) {
        loadQueueOnCast(targetIndex, 0)
        return
      }

      const itemId = findCastItemId(mediaSession, targetItem.uuid)
      const chromeCastApi = window.chrome?.cast
      const QueueJumpReq = chromeCastApi?.media?.QueueJumpRequest

      const onJumpSuccess = () => {
        if (castOpSeqRef.current !== opSeq) return
        syncFromRemotePlayerRef.current()
      }
      const fallbackToLoad = () => {
        if (!isCastingRef.current || castOpSeqRef.current !== opSeq) return
        clearJumpWatchdog()
        loadQueueOnCast(targetIndex, 0)
      }

      const armJumpWatchdog = () => {
        clearJumpWatchdog()
        jumpWatchdogTimerRef.current = setTimeout(() => {
          jumpWatchdogTimerRef.current = null
          if (
            isCastingRef.current &&
            castOpSeqRef.current === opSeq &&
            pendingJumpRef.current?.opSeq === opSeq
          ) {
            loadQueueOnCast(targetIndex, 0)
          }
        }, 4000)
      }

      if (itemId != null) {
        const hasItemsArray = Array.isArray(mediaSession.items)
        const isItemInMediaSlice =
          hasItemsArray &&
          mediaSession.items.some((it) => it?.itemId === itemId)

        if (
          (isItemInMediaSlice || !hasItemsArray) &&
          typeof mediaSession.queueJumpToItem === 'function'
        ) {
          armJumpWatchdog()
          mediaSession.queueJumpToItem(itemId, onJumpSuccess, fallbackToLoad)
          return
        }
        if (
          typeof mediaSession.queueUpdateItems === 'function' &&
          typeof QueueJumpReq === 'function'
        ) {
          const req = new QueueJumpReq()
          req.currentItemId = itemId
          armJumpWatchdog()
          mediaSession.queueUpdateItems(req, onJumpSuccess, fallbackToLoad)
          return
        }
      }

      const fromWinIdx = prevActiveUuid
        ? windowUuids.indexOf(prevActiveUuid)
        : -1
      const toWinIdx = windowUuids.indexOf(targetItem.uuid)
      const canUseRelativeDelta =
        !wasPendingLoad &&
        Boolean(prevActiveUuid) &&
        (!receiverUuidRef.current ||
          prevActiveUuid === receiverUuidRef.current) &&
        receiverRepeatModeRef.current !== 'REPEAT_ALL_AND_SHUFFLE' &&
        fromWinIdx >= 0 &&
        toWinIdx >= 0 &&
        toWinIdx !== fromWinIdx

      if (canUseRelativeDelta) {
        const delta = toWinIdx - fromWinIdx
        if (delta === 1 && typeof mediaSession.queueNext === 'function') {
          armJumpWatchdog()
          mediaSession.queueNext(onJumpSuccess, fallbackToLoad)
          return
        }
        if (delta === -1 && typeof mediaSession.queuePrev === 'function') {
          armJumpWatchdog()
          mediaSession.queuePrev(onJumpSuccess, fallbackToLoad)
          return
        }
        if (
          typeof mediaSession.queueUpdateItems === 'function' &&
          typeof QueueJumpReq === 'function'
        ) {
          const req = new QueueJumpReq()
          req.jump = delta
          armJumpWatchdog()
          mediaSession.queueUpdateItems(req, onJumpSuccess, fallbackToLoad)
          return
        }
      }

      loadQueueOnCast(targetIndex, 0)
    },
    [
      clearJumpWatchdog,
      dispatch,
      findCastItemId,
      getActiveMediaSession,
      loadQueueOnCast,
    ],
  )

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

    if (plan.action === 'STOP') {
      const activeMedia = mediaSessionRef.current || getActiveMediaSession()
      const hadCastQueueState =
        loadedWindowUuidsRef.current.length > 0 ||
        Boolean(activeUuidRef.current) ||
        queueLoadInFlightRef.current ||
        pendingCastLoadRef.current ||
        Boolean(activeMedia && activeMedia.playerState !== 'IDLE')

      if (hadCastQueueState) {
        castOpSeqRef.current += 1
        clearJumpWatchdog()
        queueLoadInFlightRef.current = false
        supersededMediaRef.current = activeMedia || null
        supersededUuidsRef.current.clear()
        pendingJumpRef.current = null
        pendingCastLoadRef.current = false
        loadedWindowUuidsRef.current = []
        loadedWindowRef.current = { start: 0, end: 0 }
        activeUuidRef.current = null
        receiverUuidRef.current = null
        lastSyncedUiUuidRef.current = null
        uuidToItemIdRef.current.clear()
        castLastPosMsRef.current = 0
        castClockRef.current = {
          baseTimeSec: 0,
          sampledAtMs: 0,
          playbackRate: 1,
          isPlaying: false,
        }
        bridgeRef.current.virtualCurrentTime = 0
        bridgeRef.current.virtualDuration = 0
        bridgeRef.current.virtualPaused = true
        flushStoppedReport()
        detachMediaSession()
        mediaSessionRef.current = null
        setRemoteState('IDLE')
        setActiveTrackId(null)
        setActiveIsRadio(false)
        stopMediaSession(activeMedia, chromeCastApi)
      }
      return
    }

    const mediaSession = getActiveMediaSession()

    if (plan.action === 'LOAD') {
      loadQueueOnCast(plan.targetIndex, plan.startSeconds)
      return
    }

    if (plan.action === 'JUMP') {
      jumpCastToTrack(plan.targetIndex, plan.targetItem)
      return
    }

    if (plan.action === 'INSERT') {
      const InsertReq = chromeCastApi?.media?.QueueInsertItemsRequest
      if (
        typeof mediaSession?.queueInsertItems !== 'function' ||
        typeof InsertReq !== 'function'
      ) {
        loadQueueOnCast(plan.activeIndex, getActiveCastTime())
        return
      }

      const beforeItemId =
        plan.insertBeforeUuid === null
          ? null
          : findCastItemId(mediaSession, plan.insertBeforeUuid)
      if (plan.insertBeforeUuid !== null && beforeItemId == null) {
        loadQueueOnCast(plan.activeIndex, getActiveCastTime())
        return
      }

      const opSeq = castOpSeqRef.current
      loadedWindowUuidsRef.current = plan.nextWindowUuids
      loadedWindowRef.current = {
        start: plan.windowSpec.windowStartIndex,
        end: plan.windowSpec.windowEndIndex,
      }

      fetchCastMediaTokens(plan.insertedItems)
        .then((tokenMap) => {
          if (!isCastingRef.current || castOpSeqRef.current !== opSeq) {
            return
          }
          const activeMediaSession = getActiveMediaSession()
          const castItems = buildCastQueueItems(
            chromeCastApi,
            plan.insertedItems,
            window.location.origin,
            tokenMap,
          )
          if (
            castItems.length === 0 ||
            typeof activeMediaSession?.queueInsertItems !== 'function'
          ) {
            loadQueueOnCast(plan.activeIndex, getActiveCastTime())
            return
          }

          const req = new InsertReq(castItems)
          if (plan.insertBeforeUuid !== null) {
            const latestBeforeId =
              findCastItemId(activeMediaSession, plan.insertBeforeUuid) ??
              beforeItemId
            if (latestBeforeId == null) {
              loadQueueOnCast(plan.activeIndex, getActiveCastTime())
              return
            }
            req.insertBefore = latestBeforeId
          }

          activeMediaSession.queueInsertItems(
            req,
            () => recordMediaSessionItems(getActiveMediaSession()),
            () => loadQueueOnCast(plan.activeIndex, getActiveCastTime()),
          )
        })
        .catch(() => {
          if (isCastingRef.current && castOpSeqRef.current === opSeq) {
            loadQueueOnCast(plan.activeIndex, getActiveCastTime())
          }
        })
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
    clearJumpWatchdog,
    detachMediaSession,
    dispatch,
    fetchCastMediaTokens,
    findCastItemId,
    flushStoppedReport,
    getActiveCastTime,
    getActiveMediaSession,
    isCasting,
    jumpCastToTrack,
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
      jumpCastToTrack(playIndex, target)
    },
    [jumpCastToTrack],
  )

  // Poll estimated playback time locally while Chromecast is actively playing.
  // Cast receivers only broadcast MEDIA_STATUS on state/track changes, not
  // every second, and CAF's RemotePlayerController internal timer fails to
  // attach when queueLoad's initial response returns media === null.
  useEffect(() => {
    if (!isCasting || remoteState !== 'PLAYING') {
      return undefined
    }

    const tick = () => {
      const bridge = bridgeRef.current
      if (
        !isCastingRef.current ||
        queueLoadInFlightRef.current ||
        pendingCastLoadRef.current ||
        Boolean(pendingJumpRef.current) ||
        Date.now() < bridge.seekingUntil
      ) {
        return
      }

      const mediaSession = getActiveMediaSession()
      const rp = remotePlayerRef.current
      const effState = resolveEffectivePlayerState(mediaSession, rp)
      if (effState !== 'PLAYING') {
        return
      }

      let nextTime = null
      if (typeof mediaSession?.getEstimatedTime === 'function') {
        const est = Number(mediaSession.getEstimatedTime())
        if (Number.isFinite(est) && est >= 0) {
          nextTime = est
        }
      } else if (
        rp?.isConnected &&
        rp?.playerState === 'PLAYING' &&
        Number.isFinite(rp?.currentTime)
      ) {
        nextTime = Math.max(0, Number(rp.currentTime))
      } else {
        const clock = castClockRef.current
        if (clock.isPlaying && clock.sampledAtMs > 0) {
          const elapsedSec =
            ((Date.now() - clock.sampledAtMs) / 1000) *
            (clock.playbackRate || 1)
          nextTime = Math.max(0, clock.baseTimeSec + elapsedSec)
        }
      }

      if (nextTime == null) {
        return
      }

      if (!activeIsRadio && bridge.virtualDuration > 0) {
        nextTime = Math.min(nextTime, bridge.virtualDuration)
      }

      if (Math.abs(nextTime - bridge.virtualCurrentTime) >= 0.05) {
        bridge.virtualCurrentTime = nextTime
        castLastPosMsRef.current = Math.floor(nextTime * 1000)
        dispatchMediaEvent(audioInstanceRef.current, 'timeupdate')
      }
    }

    const timerId = setInterval(tick, 500)
    return () => clearInterval(timerId)
  }, [activeIsRadio, getActiveMediaSession, isCasting, remoteState])

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
