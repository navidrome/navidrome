import { useEffect, useState } from 'react'
import {
  KARAOKE_CLOCK_DRIFT_RESET_MS,
  KARAOKE_CLOCK_RESET_THRESHOLD_MS,
  KARAOKE_MONOTONIC_JITTER_MS,
  KARAOKE_RENDER_UPDATE_EPSILON_MS,
} from './lyricsKaraokeConstants'

const usePlaybackClock = (visible, audioInstance) => {
  const [playbackMs, setPlaybackMs] = useState(0)

  useEffect(() => {
    if (!visible || !audioInstance) {
      setPlaybackMs(0)
      return undefined
    }

    let rafId = 0
    let cancelled = false
    let anchorAudioMs = 0
    let anchorPerfMs = null
    let lastRenderMs = 0

    const readPlaybackMs = () => {
      const seconds = Number(audioInstance.currentTime)
      if (!Number.isFinite(seconds) || seconds < 0) return 0
      return seconds * 1000
    }

    const resetAnchor = (perfNow, observedMs) => {
      anchorAudioMs = observedMs
      anchorPerfMs = perfNow
    }

    const tick = () => {
      if (cancelled) return

      const observedMs = readPlaybackMs()
      const perfNow = performance.now()
      const playbackRate = Number(audioInstance.playbackRate)
      const canInterpolate =
        !audioInstance.paused &&
        !audioInstance.seeking &&
        Number.isFinite(playbackRate) &&
        playbackRate > 0
      let nowMs = observedMs

      if (!canInterpolate) {
        resetAnchor(perfNow, observedMs)
      } else if (anchorPerfMs == null) {
        resetAnchor(perfNow, observedMs)
      } else {
        const predicted =
          anchorAudioMs + (perfNow - anchorPerfMs) * playbackRate
        const drift = observedMs - predicted
        if (Math.abs(drift) > KARAOKE_CLOCK_DRIFT_RESET_MS) {
          nowMs = observedMs
          resetAnchor(perfNow, observedMs)
        } else {
          nowMs = predicted
        }
      }

      const backwardsDrift = lastRenderMs - nowMs
      if (canInterpolate && backwardsDrift > KARAOKE_CLOCK_RESET_THRESHOLD_MS) {
        nowMs = observedMs
        resetAnchor(perfNow, observedMs)
      } else if (canInterpolate && backwardsDrift > 0) {
        nowMs = lastRenderMs
      } else if (
        !canInterpolate &&
        backwardsDrift > 0 &&
        backwardsDrift <= KARAOKE_MONOTONIC_JITTER_MS
      ) {
        nowMs = lastRenderMs
      }

      nowMs = Math.max(0, nowMs)
      lastRenderMs = nowMs

      setPlaybackMs((prev) =>
        Math.abs(prev - nowMs) >= KARAOKE_RENDER_UPDATE_EPSILON_MS
          ? nowMs
          : prev,
      )
      rafId = window.requestAnimationFrame(tick)
    }

    const initialMs = readPlaybackMs()
    resetAnchor(performance.now(), initialMs)
    lastRenderMs = initialMs
    setPlaybackMs(initialMs)
    rafId = window.requestAnimationFrame(tick)

    return () => {
      cancelled = true
      if (rafId) window.cancelAnimationFrame(rafId)
    }
  }, [audioInstance, visible])

  return playbackMs
}

export default usePlaybackClock
