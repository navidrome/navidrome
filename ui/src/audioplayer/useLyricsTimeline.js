import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from 'react'
import {
  KARAOKE_ANIMATION_PRELOAD_MS,
  KARAOKE_ANIMATION_SYNC_DRIFT_MS,
  KARAOKE_CHARACTER_WAVE_MS,
  KARAOKE_CLOCK_DRIFT_RESET_MS,
  KARAOKE_HIGHLIGHT_LEAD_MS,
  KARAOKE_LINE_RELEASE_MS,
} from './lyricsKaraokeConstants'
import {
  GECKO_REDRAW_KEYFRAMES,
  GECKO_REDRAW_MS,
  buildCharacterWavePlan,
  buildLetterMaskSweep,
  characterAnimationKeyframes,
  getTokenDurationMs,
  isGecko,
} from './lyricsKaraokeMotion'
import {
  buildLyricsTimeline,
  getTimelineScrollTarget,
  LyricTimelineCursor,
  sameIndexes,
  tokenProgressAt,
} from './lyricsTimeline'

const mediaTimeMs = (audio) => {
  const seconds = Number(audio?.currentTime)
  return Number.isFinite(seconds) && seconds >= 0 ? seconds * 1000 : 0
}

const mediaDurationMs = (audio) => {
  const seconds = Number(audio?.duration)
  return Number.isFinite(seconds) && seconds > 0 ? seconds * 1000 : null
}

const GECKO = isGecko()

const nowMs = () => (typeof performance !== 'undefined' ? performance.now() : 0)

// The sweep and letter wave are native animations. The audio clock only nudges
// them when they drift, so the browser interpolates every frame itself instead
// of receiving a new inline transform per glyph per frame.
const setAnimationsRate = (record, rate) => {
  const next = Number(rate) > 0 ? Number(rate) : 1
  if (record.rate === next) return
  const now = nowMs()
  if (record.rate != null && record.animationSyncedAt != null) {
    record.animationTime =
      (record.animationTime ?? 0) +
      (now - record.animationSyncedAt) * record.rate
  }
  record.animationSyncedAt = now
  record.rate = next
  record.animations?.forEach((animation) => {
    animation.playbackRate = next
  })
}

const predictedAnimationTime = (record, now) => {
  if (record.animationTime == null) return null
  const elapsed =
    record.animationSyncedAt == null
      ? 0
      : (now - record.animationSyncedAt) * (record.rate ?? 1)
  return record.animationTime + elapsed
}

const setAnimationsPlaying = (record, playing) => {
  if (record.playing === playing) return
  record.playing = playing
  record.animations?.forEach((animation) => {
    if (playing) animation.play()
    else animation.pause()
  })
}

const createTokenAnimations = (record, context) => {
  const animations = []
  const letterCount = record.characters.length
  record.characters.forEach((node, index) => {
    if (typeof node.animate !== 'function') return
    const sweep = buildLetterMaskSweep(
      record.animationDuration,
      letterCount,
      index,
    )
    animations.push(
      node.animate(sweep.keyframes, {
        duration: sweep.durationMs,
        delay: sweep.delayMs,
        easing: 'linear',
        fill: 'both',
      }),
    )
  })

  if (record.waveEnabled) {
    const plan = buildCharacterWavePlan(record.window, letterCount)
    record.characters.forEach((node, index) => {
      if (typeof node.animate !== 'function') return
      const timing = plan[index]
      animations.push(
        node.animate(characterAnimationKeyframes, {
          duration: timing.durationMs,
          delay: timing.delayMs,
          fill: 'both',
        }),
      )
      if (GECKO) {
        animations.push(
          node.animate(GECKO_REDRAW_KEYFRAMES, {
            duration: GECKO_REDRAW_MS,
            iterations: Math.max(
              1,
              Math.ceil(timing.durationMs / GECKO_REDRAW_MS),
            ),
            delay: timing.delayMs,
            composite: 'add',
          }),
        )
      }
    })
  }

  record.animations = animations
  record.animationTime = null
  record.animationSyncedAt = null
  record.playing = null
  record.rate = null
  setAnimationsRate(record, context.rate)
}

const syncTokenAnimations = (record, timeMs, context, force = false) => {
  if (!record.animations?.length) return
  setAnimationsRate(record, context.rate)
  const now = nowMs()
  const next = Math.max(0, Number(timeMs) || 0)
  const predicted = predictedAnimationTime(record, now)
  const drift = predicted == null ? Infinity : Math.abs(predicted - next)
  const wasPlaying = record.playing
  if (force || drift > KARAOKE_ANIMATION_SYNC_DRIFT_MS) {
    record.animationTime = next
    record.animationSyncedAt = now
    record.animations.forEach((animation) => {
      animation.currentTime = next
    })
  }
  setAnimationsPlaying(record, context.playing)
  if (context.playing && wasPlaying !== true) {
    // play() rewinds an animation that already finished. Put every animation back
    // on the sampled clock so resuming a paused song never replays a line.
    record.animationTime = next
    record.animationSyncedAt = now
    record.animations.forEach((animation) => {
      animation.currentTime = next
    })
  }
}

const applyTokenState = (record, state, context) => {
  record.state = state
  record.node.dataset.lyricsState = state

  if (state === 'future') {
    syncTokenAnimations(record, 0, { ...context, playing: false }, true)
    return
  }
  if (state === 'release' || state === 'inactive-past') {
    syncTokenAnimations(
      record,
      record.animationDuration + KARAOKE_CHARACTER_WAVE_MS,
      { ...context, playing: false },
      true,
    )
  }
}

const resetToken = (record, state, context) => {
  applyTokenState(record, state, context)
}

const setTokenPresentation = (record, time, context, force = false) => {
  const progress = tokenProgressAt(record.window, time)
  const state =
    progress <= 0 ? 'future' : progress >= 1 ? 'completed' : 'active'
  const changed = record.state !== state
  if (changed || state === 'active') {
    applyTokenState(record, state, context)
  }

  if (state === 'active' || state === 'completed') {
    const start = Number(record.window?.start) || 0
    syncTokenAnimations(record, time - start, context, force || changed)
  }
}

const canListenToMedia = (audio) =>
  Boolean(
    audio &&
    typeof audio.addEventListener === 'function' &&
    typeof audio.removeEventListener === 'function',
  )

const useLyricsTimeline = ({
  lines,
  audioInstance,
  visible,
  reducedMotion,
}) => {
  const [durationMs, setDurationMs] = useState(() =>
    mediaDurationMs(audioInstance),
  )

  useEffect(() => {
    const updateDuration = () => setDurationMs(mediaDurationMs(audioInstance))
    updateDuration()
    if (!canListenToMedia(audioInstance)) return undefined
    audioInstance.addEventListener('loadedmetadata', updateDuration)
    audioInstance.addEventListener('durationchange', updateDuration)
    return () => {
      audioInstance.removeEventListener('loadedmetadata', updateDuration)
      audioInstance.removeEventListener('durationchange', updateDuration)
    }
  }, [audioInstance])

  const timeline = useMemo(
    () => buildLyricsTimeline(lines, { durationMs }),
    [durationMs, lines],
  )
  const cursorRef = useRef(new LyricTimelineCursor(timeline))
  const lineNodesRef = useRef(new Map())
  const tokenRecordsRef = useRef(new Map())
  const lineTokenKeysRef = useRef(new Map())
  const releaseIndexesRef = useRef(new Set())
  const frameRef = useRef(0)
  const resetInterpolationRef = useRef(null)
  const lastPublishedIndexesRef = useRef([])
  const lastPublishedPrimaryIndexRef = useRef(-1)
  const lastScrollTargetRef = useRef(-1)
  const lastAppliedTimeRef = useRef(0)
  const mediaContextRef = useRef({ playing: false, rate: 1 })
  const [activeState, setActiveState] = useState(() => ({
    indexes: [],
    primaryIndex: -1,
  }))
  const [scrollTargetIndex, setScrollTargetIndex] = useState(-1)

  const setLineState = useCallback((lineIndex, phase) => {
    const node = lineNodesRef.current.get(lineIndex)
    if (!node) return
    node.dataset.active = phase === 'active' ? 'true' : 'false'
    node.dataset.lifecycle = phase
    node.dataset.highlightActive = phase === 'active' ? 'true' : 'false'
    node.dataset.raised = phase === 'idle' ? 'false' : 'true'
  }, [])

  const resetLineTokens = useCallback((lineIndex, state = 'future') => {
    lineTokenKeysRef.current.get(lineIndex)?.forEach((key) => {
      const record = tokenRecordsRef.current.get(key)
      if (record) resetToken(record, state, mediaContextRef.current)
    })
  }, [])

  const ensureLineAnimations = useCallback((lineIndex) => {
    lineTokenKeysRef.current.get(lineIndex)?.forEach((key) => {
      const record = tokenRecordsRef.current.get(key)
      if (record && !record.animations) {
        createTokenAnimations(record, mediaContextRef.current)
      }
    })
  }, [])

  const preloadUpcomingLine = useCallback(
    (current) => {
      let nextIndex = -1
      let nextStart = Infinity
      timeline.windows.forEach((window) => {
        if (!window.valid || window.start == null) return
        const delta = window.start - current
        if (
          delta > 0 &&
          delta <= KARAOKE_ANIMATION_PRELOAD_MS &&
          window.start < nextStart
        ) {
          nextStart = window.start
          nextIndex = window.lineIndex
        }
      })
      if (nextIndex >= 0) ensureLineAnimations(nextIndex)
    },
    [ensureLineAnimations, timeline.windows],
  )

  const updateLineTokens = useCallback((lineIndex, time, force = false) => {
    lineTokenKeysRef.current.get(lineIndex)?.forEach((key) => {
      const record = tokenRecordsRef.current.get(key)
      if (record)
        setTokenPresentation(record, time, mediaContextRef.current, force)
    })
  }, [])

  const publishActiveState = useCallback((indexes, nextPrimaryIndex) => {
    if (
      sameIndexes(lastPublishedIndexesRef.current, indexes) &&
      lastPublishedPrimaryIndexRef.current === nextPrimaryIndex
    ) {
      return
    }
    lastPublishedIndexesRef.current = indexes
    lastPublishedPrimaryIndexRef.current = nextPrimaryIndex
    setActiveState({ indexes, primaryIndex: nextPrimaryIndex })
  }, [])

  const publishScrollTarget = useCallback(
    (time) => {
      const nextIndex = getTimelineScrollTarget(timeline, time)
      if (nextIndex === lastScrollTargetRef.current) return
      lastScrollTargetRef.current = nextIndex
      setScrollTargetIndex(nextIndex)
    },
    [timeline],
  )

  const apply = useCallback(
    (time, forceSeek = false) => {
      const current = Number.isFinite(Number(time))
        ? Math.max(0, Number(time))
        : 0
      const lead = reducedMotion ? 0 : KARAOKE_HIGHLIGHT_LEAD_MS
      const rate = Number(audioInstance?.playbackRate)
      mediaContextRef.current = {
        playing: Boolean(
          audioInstance &&
          !audioInstance.paused &&
          (typeof document === 'undefined' ||
            document.visibilityState === 'visible'),
        ),
        rate: Number.isFinite(rate) && rate > 0 ? rate : 1,
      }
      lastAppliedTimeRef.current = current
      const cursor = cursorRef.current
      const previousIndexes = cursor.lastIndexes
      const result = cursor.update(current, forceSeek)

      if (forceSeek) {
        releaseIndexesRef.current.clear()
        timeline.windows.forEach((window) => {
          if (!window.valid || current < window.start) {
            setLineState(window.lineIndex, 'idle')
            resetLineTokens(window.lineIndex, 'future')
            return
          }
          if (current >= window.end) {
            if (current < window.end + KARAOKE_LINE_RELEASE_MS) {
              releaseIndexesRef.current.add(window.lineIndex)
              setLineState(window.lineIndex, 'release')
              resetLineTokens(window.lineIndex, 'release')
            } else {
              setLineState(window.lineIndex, 'past')
              resetLineTokens(window.lineIndex, 'inactive-past')
            }
          }
        })
      } else if (result.changed) {
        previousIndexes.forEach((lineIndex) => {
          if (cursor.active.has(lineIndex)) return
          const window = timeline.windows[lineIndex]
          if (
            window?.valid &&
            current >= window.end &&
            current < window.end + KARAOKE_LINE_RELEASE_MS
          ) {
            releaseIndexesRef.current.add(lineIndex)
            setLineState(lineIndex, 'release')
            resetLineTokens(lineIndex, 'release')
          } else {
            setLineState(lineIndex, 'past')
            resetLineTokens(lineIndex, 'inactive-past')
          }
        })
      }

      result.indexes.forEach((lineIndex) => {
        releaseIndexesRef.current.delete(lineIndex)
        setLineState(lineIndex, 'active')
        ensureLineAnimations(lineIndex)
        updateLineTokens(lineIndex, current + lead, forceSeek)
      })

      releaseIndexesRef.current.forEach((lineIndex) => {
        const window = timeline.windows[lineIndex]
        if (
          !window?.valid ||
          current >= window.end + KARAOKE_LINE_RELEASE_MS ||
          current < window.end
        ) {
          releaseIndexesRef.current.delete(lineIndex)
          const isPast = current >= (window?.end ?? Infinity)
          setLineState(lineIndex, isPast ? 'past' : 'idle')
          resetLineTokens(lineIndex, isPast ? 'inactive-past' : 'future')
          return
        }
      })

      if (result.changed || forceSeek) {
        preloadUpcomingLine(current)
        publishActiveState(result.indexes, result.primaryIndex)
      }
      publishScrollTarget(current)
      return result
    },
    [
      audioInstance,
      ensureLineAnimations,
      preloadUpcomingLine,
      publishActiveState,
      publishScrollTarget,
      reducedMotion,
      resetLineTokens,
      setLineState,
      timeline.windows,
      updateLineTokens,
    ],
  )

  const registerLine = useCallback(
    (lineIndex, node) => {
      if (!node) {
        lineNodesRef.current.delete(lineIndex)
        return
      }
      lineNodesRef.current.set(lineIndex, node)
      const window = timeline.windows[lineIndex]
      const time = lastAppliedTimeRef.current
      if (cursorRef.current.active.has(lineIndex)) {
        setLineState(lineIndex, 'active')
      } else if (
        window?.valid &&
        time >= window.end &&
        time < window.end + KARAOKE_LINE_RELEASE_MS
      ) {
        setLineState(lineIndex, 'release')
      } else if (window?.valid && time >= window.end) {
        setLineState(lineIndex, 'past')
      } else {
        setLineState(lineIndex, 'idle')
      }
    },
    [setLineState, timeline.windows],
  )

  const registerToken = useCallback(
    (key, descriptor, node) => {
      const existing = tokenRecordsRef.current.get(key)
      if (!node) {
        if (existing) {
          existing.animations?.forEach((animation) => animation.cancel?.())
          tokenRecordsRef.current.delete(key)
          const keys = lineTokenKeysRef.current.get(existing.lineIndex)
          keys?.delete(key)
          if (keys?.size === 0) {
            lineTokenKeysRef.current.delete(existing.lineIndex)
          }
        }
        return
      }

      const characterNodes = reducedMotion
        ? []
        : Array.from(node.querySelectorAll('[data-lyrics-character="true"]'))
      const record = {
        key,
        node,
        lineIndex: descriptor.lineIndex,
        window: descriptor.window,
        waveEnabled: descriptor.waveEnabled,
        animationDuration: getTokenDurationMs(descriptor.window),
        characters: characterNodes,
        animations: null,
        animationTime: null,
        animationSyncedAt: null,
        playing: null,
        rate: null,
        state: null,
      }
      tokenRecordsRef.current.set(key, record)
      if (!lineTokenKeysRef.current.has(record.lineIndex)) {
        lineTokenKeysRef.current.set(record.lineIndex, new Set())
      }
      lineTokenKeysRef.current.get(record.lineIndex).add(key)

      if (cursorRef.current.active.has(record.lineIndex)) {
        createTokenAnimations(record, mediaContextRef.current)
        setTokenPresentation(
          record,
          lastAppliedTimeRef.current +
            (reducedMotion ? 0 : KARAOKE_HIGHLIGHT_LEAD_MS),
          mediaContextRef.current,
          true,
        )
      } else {
        const lineWindow = timeline.windows[record.lineIndex]
        if (
          lineWindow?.valid &&
          lastAppliedTimeRef.current >= lineWindow.end &&
          lastAppliedTimeRef.current < lineWindow.end + KARAOKE_LINE_RELEASE_MS
        ) {
          resetToken(record, 'release', mediaContextRef.current)
        } else {
          resetToken(
            record,
            lineWindow?.valid && lastAppliedTimeRef.current >= lineWindow.end
              ? 'inactive-past'
              : 'future',
            mediaContextRef.current,
          )
        }
      }
    },
    [reducedMotion, timeline.windows],
  )

  const getLineNode = useCallback(
    (lineIndex) => lineNodesRef.current.get(lineIndex) || null,
    [],
  )

  useLayoutEffect(() => {
    lineNodesRef.current.forEach((node) => {
      node.dataset.active = 'false'
      node.dataset.lifecycle = 'idle'
      node.dataset.highlightActive = 'false'
      node.dataset.raised = 'false'
    })
    tokenRecordsRef.current.forEach((record) =>
      resetToken(record, 'future', mediaContextRef.current),
    )
    cursorRef.current = new LyricTimelineCursor(timeline)
    releaseIndexesRef.current.clear()
    lastPublishedIndexesRef.current = []
    lastPublishedPrimaryIndexRef.current = -1
    lastScrollTargetRef.current = -1
    setActiveState({ indexes: [], primaryIndex: -1 })
    setScrollTargetIndex(-1)
    apply(mediaTimeMs(audioInstance), true)
  }, [apply, audioInstance, timeline])

  useEffect(() => {
    if (
      !audioInstance ||
      !timeline.events.length ||
      !canListenToMedia(audioInstance)
    ) {
      return undefined
    }
    const seek = () => {
      resetInterpolationRef.current?.()
      apply(mediaTimeMs(audioInstance), true)
    }
    audioInstance.addEventListener('seeking', seek)
    audioInstance.addEventListener('seeked', seek)
    return () => {
      audioInstance.removeEventListener('seeking', seek)
      audioInstance.removeEventListener('seeked', seek)
    }
  }, [apply, audioInstance, timeline.events.length])

  useEffect(() => {
    if (!audioInstance || !timeline.events.length || !visible) return undefined
    if (!canListenToMedia(audioInstance)) {
      apply(mediaTimeMs(audioInstance), true)
      return undefined
    }

    let cancelled = false
    let anchorAudioMs = mediaTimeMs(audioInstance)
    let anchorPerfMs = performance.now()
    let lastFrameTime = anchorAudioMs

    const stop = () => {
      if (frameRef.current) window.cancelAnimationFrame(frameRef.current)
      frameRef.current = 0
    }

    const resetAnchor = () => {
      anchorAudioMs = mediaTimeMs(audioInstance)
      anchorPerfMs = performance.now()
      lastFrameTime = anchorAudioMs
    }
    resetInterpolationRef.current = resetAnchor

    const readInterpolatedTime = () => {
      const observed = mediaTimeMs(audioInstance)
      const now = performance.now()
      const rate = Number(audioInstance.playbackRate)
      const canInterpolate =
        !audioInstance.paused &&
        !audioInstance.seeking &&
        Number.isFinite(rate) &&
        rate > 0
      if (!canInterpolate) {
        anchorAudioMs = observed
        anchorPerfMs = now
        lastFrameTime = observed
        return observed
      }
      const predicted = anchorAudioMs + (now - anchorPerfMs) * rate
      const drift = observed - predicted
      let current =
        Math.abs(drift) > KARAOKE_CLOCK_DRIFT_RESET_MS ? observed : predicted
      if (Math.abs(drift) > KARAOKE_CLOCK_DRIFT_RESET_MS) {
        anchorAudioMs = observed
        anchorPerfMs = now
      }
      const backwards = lastFrameTime - current
      if (backwards > 0) {
        current = lastFrameTime
      }
      lastFrameTime = Math.max(0, current)
      return lastFrameTime
    }

    const tick = () => {
      if (
        cancelled ||
        audioInstance.paused ||
        window.document.visibilityState !== 'visible'
      ) {
        stop()
        return
      }
      apply(readInterpolatedTime())
      frameRef.current = window.requestAnimationFrame(tick)
    }

    const start = () => {
      if (
        frameRef.current ||
        audioInstance.paused ||
        window.document.visibilityState !== 'visible'
      ) {
        return
      }
      resetAnchor()
      frameRef.current = window.requestAnimationFrame(tick)
    }

    const syncAndStop = () => {
      stop()
      apply(mediaTimeMs(audioInstance), true)
    }

    const visibility = () => {
      if (window.document.visibilityState !== 'visible') {
        mediaContextRef.current = {
          ...mediaContextRef.current,
          playing: false,
        }
        tokenRecordsRef.current.forEach((record) =>
          setAnimationsPlaying(record, false),
        )
        stop()
        return
      }
      apply(mediaTimeMs(audioInstance), true)
      start()
    }

    audioInstance.addEventListener('play', start)
    audioInstance.addEventListener('pause', syncAndStop)
    window.document.addEventListener('visibilitychange', visibility)
    apply(mediaTimeMs(audioInstance), true)
    start()

    return () => {
      cancelled = true
      stop()
      if (resetInterpolationRef.current === resetAnchor) {
        resetInterpolationRef.current = null
      }
      audioInstance.removeEventListener('play', start)
      audioInstance.removeEventListener('pause', syncAndStop)
      window.document.removeEventListener('visibilitychange', visibility)
    }
  }, [apply, audioInstance, timeline.events.length, visible])

  const syncNow = useCallback(
    (time = mediaTimeMs(audioInstance), forceSeek = true) =>
      apply(time, forceSeek),
    [apply, audioInstance],
  )

  useEffect(
    () => () => {
      tokenRecordsRef.current.forEach((record) => {
        record.animations?.forEach((animation) => animation.cancel?.())
      })
    },
    [],
  )

  return {
    activeIndexes: activeState.indexes,
    primaryIndex: activeState.primaryIndex,
    scrollTargetIndex,
    registerLine,
    registerToken,
    getLineNode,
    syncNow,
    timeline,
  }
}

export default useLyricsTimeline
