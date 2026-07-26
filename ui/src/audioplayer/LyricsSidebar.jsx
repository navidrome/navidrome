import { alpha, makeStyles } from '@material-ui/core/styles'
import React, { useCallback, useEffect, useRef, useState } from 'react'
import LyricsLayerControls from './LyricsLayerControls'
import LyricsPanel from './LyricsPanel'
import {
  LYRICS_SIDEBAR_MAX_WIDTH,
  LYRICS_SIDEBAR_MIN_WIDTH,
  LYRICS_SIDEBAR_TRANSITION_MS,
  LYRICS_SIDEBAR_WIDTH_STEP,
  clampSidebarWidth,
  loadSidebarWidth,
  notifySidebarWidthChange,
  saveSidebarWidth,
} from './lyricsSidebarWidth'
import useEnterExitTransition from './useEnterExitTransition'

const useStyles = makeStyles((theme) => ({
  sidebar: {
    position: 'fixed',
    top: 48,
    right: 0,
    bottom: 80,
    minHeight: 0,
    minWidth: LYRICS_SIDEBAR_MIN_WIDTH,
    maxWidth: LYRICS_SIDEBAR_MAX_WIDTH,
    display: 'flex',
    flexDirection: 'column',
    color: theme.palette.text.primary,
    backgroundColor: theme.palette.background.default,
    backgroundImage: 'none',
    borderLeft: `1px solid ${alpha(theme.palette.divider, 0.1)}`,
    boxShadow: 'none',
    zIndex: theme.zIndex.appBar - 1,
    transition: `transform ${LYRICS_SIDEBAR_TRANSITION_MS}ms cubic-bezier(0.22, 1, 0.36, 1), opacity ${LYRICS_SIDEBAR_TRANSITION_MS}ms cubic-bezier(0.22, 1, 0.36, 1)`,
    willChange: 'transform, opacity',
    '@media (prefers-reduced-motion: reduce)': {
      transition: 'none',
    },
    '@media (hover: hover) and (pointer: fine)': {
      '&:hover $resizer::after': {
        background: alpha(theme.palette.primary.main, 0.32),
      },
    },
  },
  resizer: {
    position: 'absolute',
    top: 0,
    bottom: 0,
    left: -6,
    width: 12,
    padding: 0,
    border: 0,
    background: 'transparent',
    cursor: 'ew-resize',
    zIndex: 3,
    '&::after': {
      content: '""',
      position: 'absolute',
      top: 0,
      bottom: 0,
      left: 5,
      width: 2,
      background: 'transparent',
      transition: 'background 160ms ease',
    },
    '&:focus::after': {
      background: alpha(theme.palette.primary.main, 0.48),
    },
    '@media (hover: hover) and (pointer: fine)': {
      '&:hover::after': {
        background: alpha(theme.palette.primary.main, 0.48),
      },
    },
    '&:focus': {
      outline: 'none',
    },
  },
  panel: {
    flex: 1,
    minHeight: 0,
    position: 'relative',
  },
}))

const LyricsSidebar = ({
  visible,
  mainLyric,
  translationLyric,
  pronunciationLyric,
  showTranslation,
  showPronunciation,
  translationEnabled,
  pronunciationEnabled,
  onToggleTranslation,
  onTogglePronunciation,
  audioInstance,
  loading = false,
  error = null,
  labels = {},
}) => {
  const [width, setWidth] = useState(loadSidebarWidth)
  const sidebarRef = useRef(null)
  const resizerRef = useRef(null)
  const widthRef = useRef(width)
  const resizeCleanupRef = useRef(null)
  const { rendered, entered } = useEnterExitTransition(
    visible,
    LYRICS_SIDEBAR_TRANSITION_MS,
  )
  const [isResizing, setIsResizing] = useState(false)
  const classes = useStyles()

  useEffect(() => {
    widthRef.current = width
  }, [width])

  useEffect(
    () => () => {
      resizeCleanupRef.current?.()
      resizeCleanupRef.current = null
    },
    [],
  )

  useEffect(() => {
    if (entered || typeof document === 'undefined') return
    const activeElement = document.activeElement
    if (activeElement && sidebarRef.current?.contains(activeElement)) {
      activeElement.blur()
    }
  }, [entered])

  const applyWidth = useCallback((next) => {
    const resolvedWidth = clampSidebarWidth(
      typeof next === 'function' ? next(widthRef.current) : next,
    )
    widthRef.current = resolvedWidth
    if (sidebarRef.current) {
      sidebarRef.current.style.width = `${resolvedWidth}px`
    }
    resizerRef.current?.setAttribute('aria-valuenow', String(resolvedWidth))
    notifySidebarWidthChange(resolvedWidth)
    return resolvedWidth
  }, [])

  const updateWidth = useCallback(
    (next, { persist = false } = {}) => {
      const resolvedWidth = applyWidth(next)
      setWidth(resolvedWidth)
      if (persist) saveSidebarWidth(resolvedWidth)
    },
    [applyWidth],
  )

  const handleResizePointerDown = useCallback(
    (event) => {
      event.preventDefault()
      resizeCleanupRef.current?.()

      const target = event.currentTarget
      const pointerId = event.pointerId
      const startX = event.clientX
      const startWidth = widthRef.current
      let latestWidth = startWidth
      let cleanedUp = false
      const canCapture =
        pointerId != null && typeof target.setPointerCapture === 'function'
      const listenerTarget = window
      setIsResizing(true)

      const handlePointerMove = (moveEvent) => {
        latestWidth = applyWidth(startWidth + startX - moveEvent.clientX)
      }

      const cleanupResize = ({ persist = false } = {}) => {
        if (cleanedUp) return
        cleanedUp = true
        listenerTarget.removeEventListener('pointermove', handlePointerMove)
        listenerTarget.removeEventListener('pointerup', handlePointerUp)
        listenerTarget.removeEventListener('pointercancel', handlePointerCancel)
        target.removeEventListener('lostpointercapture', handlePointerCancel)
        if (
          canCapture &&
          typeof target.releasePointerCapture === 'function' &&
          (!target.hasPointerCapture || target.hasPointerCapture(pointerId))
        ) {
          try {
            target.releasePointerCapture(pointerId)
          } catch {
            // Ignore stale pointer capture state.
          }
        }
        setWidth(latestWidth)
        if (persist) saveSidebarWidth(latestWidth)
        setIsResizing(false)
        if (resizeCleanupRef.current === cleanupResize) {
          resizeCleanupRef.current = null
        }
      }

      const handlePointerUp = () => cleanupResize({ persist: true })
      const handlePointerCancel = () => cleanupResize()

      resizeCleanupRef.current = cleanupResize
      listenerTarget.addEventListener('pointermove', handlePointerMove)
      listenerTarget.addEventListener('pointerup', handlePointerUp)
      listenerTarget.addEventListener('pointercancel', handlePointerCancel)
      target.addEventListener('lostpointercapture', handlePointerCancel)
      if (canCapture) {
        try {
          target.setPointerCapture(pointerId)
        } catch {
          // Window-level listeners keep resize working when capture is unavailable.
        }
      }
    },
    [applyWidth],
  )

  const handleResizeKeyDown = useCallback(
    (event) => {
      if (event.key === 'ArrowLeft') {
        event.preventDefault()
        updateWidth((current) => current + LYRICS_SIDEBAR_WIDTH_STEP, {
          persist: true,
        })
      } else if (event.key === 'ArrowRight') {
        event.preventDefault()
        updateWidth((current) => current - LYRICS_SIDEBAR_WIDTH_STEP, {
          persist: true,
        })
      } else if (event.key === 'Home') {
        event.preventDefault()
        updateWidth(LYRICS_SIDEBAR_MIN_WIDTH, { persist: true })
      } else if (event.key === 'End') {
        event.preventDefault()
        updateWidth(LYRICS_SIDEBAR_MAX_WIDTH, { persist: true })
      }
    },
    [updateWidth],
  )

  if (!rendered) return null

  return (
    <aside
      className={classes.sidebar}
      data-testid="lyrics-sidebar"
      ref={sidebarRef}
      style={{
        width: widthRef.current,
        transform: entered ? 'translateX(0)' : 'translateX(100%)',
        opacity: entered ? 1 : 0,
        pointerEvents: entered ? 'auto' : 'none',
      }}
      aria-label={labels.title || 'Lyrics'}
      aria-hidden={!entered}
      data-resizing={isResizing ? 'true' : 'false'}
    >
      <button
        type="button"
        ref={resizerRef}
        className={classes.resizer}
        data-testid="lyrics-sidebar-resizer"
        aria-label={labels.resize || 'Resize lyrics sidebar'}
        aria-orientation="vertical"
        aria-valuemin={LYRICS_SIDEBAR_MIN_WIDTH}
        aria-valuemax={LYRICS_SIDEBAR_MAX_WIDTH}
        aria-valuenow={width}
        role="separator"
        onPointerDown={handleResizePointerDown}
        onKeyDown={handleResizeKeyDown}
      />
      <div className={classes.panel}>
        <LyricsLayerControls
          showPronunciation={showPronunciation}
          showTranslation={showTranslation}
          pronunciationEnabled={pronunciationEnabled}
          translationEnabled={translationEnabled}
          onTogglePronunciation={onTogglePronunciation}
          onToggleTranslation={onToggleTranslation}
          labels={labels}
          testId="lyrics-sidebar-floating-controls"
        />
        <LyricsPanel
          visible={visible}
          mainLyric={mainLyric}
          translationLyric={translationLyric}
          pronunciationLyric={pronunciationLyric}
          showTranslation={showTranslation}
          showPronunciation={showPronunciation}
          audioInstance={audioInstance}
          loading={loading}
          error={error}
          labels={labels}
        />
      </div>
    </aside>
  )
}

export default LyricsSidebar
