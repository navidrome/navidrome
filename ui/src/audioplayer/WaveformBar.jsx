import React, { useCallback, useEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { drawWaveform, loadWaveform } from './waveform'

// The desktop panel and the mobile full-screen player each have a progress bar; only one is shown at a time.
const PROGRESS_BAR_SELECTOR = [
  '.react-jinke-music-player-main .music-player-panel .progress-bar',
  '.react-jinke-music-player-mobile .react-jinke-music-player-mobile-progress-bar',
].join(', ')
const FALLBACK_PLAYED_COLOR = '#31c27c'
const FALLBACK_UNPLAYED_COLOR = '#888888'

const isVisible = (color) =>
  color && color !== 'transparent' && color !== 'rgba(0, 0, 0, 0)'

// Played bars use the slider track color, which the player library styles per theme.
const playedColor = (progressBar) => {
  const track = progressBar.querySelector('.rc-slider-track')
  const color = track && window.getComputedStyle(track).backgroundColor
  return isVisible(color) ? color : FALLBACK_PLAYED_COLOR
}

// Unplayed bars use the color of the time next to the bar, which themes make readable on the
// player background. The slider rail color is often too dark for that.
const unplayedColor = (progressBar) => {
  const time = progressBar.parentElement?.querySelector('.current-time')
  const color = time && window.getComputedStyle(time).color
  return isVisible(color) ? color : FALLBACK_UNPLAYED_COLOR
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

// WaveformBar draws the current song's waveform inside the player's progress bar. The
// slider stays on top of it, so seeking keeps working as before. onAvailable reports whether a
// waveform is being shown, so the player can hide the regular bar.
const WaveformBar = ({ trackId, audioInstance, onAvailable }) => {
  const [progressBar, setProgressBar] = useState(null)
  const [peaks, setPeaks] = useState(null)
  const canvasRef = useRef(null)

  // The progress bar is rendered by the player library, which mounts the mobile player only while
  // it is open and can re-mount the desktop panel, so it is looked up again whenever the DOM changes.
  useEffect(() => {
    const locate = () =>
      setProgressBar(document.querySelector(PROGRESS_BAR_SELECTOR))
    locate()
    const observer = new MutationObserver(locate)
    observer.observe(document.body, { childList: true, subtree: true })
    return () => observer.disconnect()
  }, [])

  useEffect(() => {
    setPeaks(null)
    if (!trackId) {
      return
    }
    // Aborting on track change stops the server from decoding a song that is no longer playing,
    // so skipping through a queue does not hold up the current song's waveform.
    const controller = new AbortController()
    loadWaveform(trackId, controller.signal)
      .then((p) => !controller.signal.aborted && setPeaks(p))
      .catch(() => {})
    return () => controller.abort()
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
    drawWaveform(
      ctx,
      peaks,
      width,
      height,
      progress,
      playedColor(progressBar),
      unplayedColor(progressBar),
    )
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
