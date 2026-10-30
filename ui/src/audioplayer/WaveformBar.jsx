import React, { useCallback, useEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { drawWaveform, loadWaveform } from './waveform'

const PROGRESS_BAR_SELECTOR =
  '.react-jinke-music-player-main .music-player-panel .progress-bar'
const MAX_LOCATE_ATTEMPTS = 60
const FALLBACK_COLOR = '#31c27c'

// The player library styles the slider track per theme, so the waveform reuses its color.
const trackColor = (progressBar) => {
  const track = progressBar.querySelector('.rc-slider-track')
  const color = track && window.getComputedStyle(track).backgroundColor
  if (!color || color === 'transparent' || color === 'rgba(0, 0, 0, 0)') {
    return FALLBACK_COLOR
  }
  return color
}

const canvasStyle = {
  position: 'absolute',
  left: 0,
  top: 0,
  width: '100%',
  height: '100%',
  pointerEvents: 'none',
  zIndex: 77,
}

// WaveformBar draws the current song's waveform inside the desktop player's progress bar. The
// slider stays on top of it, so seeking keeps working as before. onAvailable reports whether a
// waveform is being shown, so the player can hide the regular bar.
const WaveformBar = ({ trackId, audioInstance, onAvailable }) => {
  const [progressBar, setProgressBar] = useState(null)
  const [peaks, setPeaks] = useState(null)
  const canvasRef = useRef(null)

  // The progress bar is rendered by the player library, so it may not exist yet, and it can be
  // replaced when the player re-mounts.
  useEffect(() => {
    let frame
    let attempts = 0
    const locate = () => {
      const el = document.querySelector(PROGRESS_BAR_SELECTOR)
      if (el) {
        setProgressBar(el)
      } else if (++attempts < MAX_LOCATE_ATTEMPTS) {
        frame = requestAnimationFrame(locate)
      }
    }
    locate()
    return () => cancelAnimationFrame(frame)
  }, [trackId])

  useEffect(() => {
    setPeaks(null)
    if (!trackId) {
      return
    }
    let cancelled = false
    loadWaveform(trackId)
      .then((p) => !cancelled && setPeaks(p))
      .catch(() => {})
    return () => {
      cancelled = true
    }
  }, [trackId])

  const available = !!(peaks && progressBar)
  useEffect(() => {
    onAvailable(available)
  }, [available, onAvailable])
  useEffect(() => () => onAvailable(false), [onAvailable])

  const draw = useCallback(() => {
    const canvas = canvasRef.current
    if (!canvas || !peaks || !progressBar) {
      return
    }
    const width = canvas.clientWidth
    const height = canvas.clientHeight
    const dpr = window.devicePixelRatio || 1
    if (canvas.width !== width * dpr || canvas.height !== height * dpr) {
      canvas.width = width * dpr
      canvas.height = height * dpr
    }
    const ctx = canvas.getContext('2d')
    if (!ctx) {
      return
    }
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
    const duration = audioInstance?.duration
    const progress =
      duration > 0 && Number.isFinite(duration)
        ? Math.min(1, audioInstance.currentTime / duration)
        : 0
    drawWaveform(ctx, peaks, width, height, progress, trackColor(progressBar))
  }, [peaks, progressBar, audioInstance])

  useEffect(() => {
    draw()
    if (!audioInstance) {
      return
    }
    const events = ['timeupdate', 'seeked', 'durationchange', 'emptied']
    events.forEach((e) => audioInstance.addEventListener(e, draw))
    return () =>
      events.forEach((e) => audioInstance.removeEventListener(e, draw))
  }, [audioInstance, draw])

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas || typeof ResizeObserver === 'undefined') {
      return
    }
    const observer = new ResizeObserver(() => draw())
    observer.observe(canvas)
    return () => observer.disconnect()
  }, [draw, available])

  if (!available) {
    return null
  }
  return createPortal(
    <canvas
      ref={canvasRef}
      className="nd-waveform"
      style={canvasStyle}
      data-testid="waveform"
    />,
    progressBar,
  )
}

export default WaveformBar
