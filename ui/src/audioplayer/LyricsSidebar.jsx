import IconButton from '@material-ui/core/IconButton'
import Tooltip from '@material-ui/core/Tooltip'
import { alpha, makeStyles, useTheme } from '@material-ui/core/styles'
import RecordVoiceOverIcon from '@material-ui/icons/RecordVoiceOver'
import TranslateIcon from '@material-ui/icons/Translate'
import clsx from 'clsx'
import React, { useCallback, useEffect, useRef, useState } from 'react'
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
    width: (props) => props.width,
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
  controls: {
    position: 'absolute',
    bottom: theme.spacing(1),
    right: theme.spacing(0.75),
    zIndex: 2,
    display: 'flex',
    alignItems: 'center',
    gap: theme.spacing(0.25),
    padding: theme.spacing(0.25),
    borderRadius: theme.shape.borderRadius * 2,
    backgroundColor: 'transparent',
  },
  controlButton: {
    padding: theme.spacing(0.75),
    color: alpha(theme.palette.text.primary, 0.58),
    backgroundColor: 'transparent',
    transition:
      'color 160ms ease, background-color 160ms ease, transform 160ms cubic-bezier(0.23, 1, 0.32, 1)',
    '@media (hover: hover) and (pointer: fine)': {
      '&:hover': {
        color: theme.palette.text.primary,
        backgroundColor: alpha(theme.palette.primary.main, 0.08),
      },
      '&$controlActive:hover': {
        color: theme.palette.primary.main,
      },
    },
    '&:focus-visible': {
      color: theme.palette.text.primary,
      backgroundColor: alpha(theme.palette.primary.main, 0.1),
    },
    '&:active:not(:disabled)': {
      transform: 'scale(0.97)',
    },
    '&$controlActive': {
      color: theme.palette.primary.main,
    },
    '&$controlActive:focus-visible': {
      color: theme.palette.primary.main,
    },
    '&:disabled': {
      color: alpha(theme.palette.text.primary, 0.28),
    },
    '@media (prefers-reduced-motion: reduce)': {
      transition: 'none',
      '&:active:not(:disabled)': {
        transform: 'none',
      },
    },
  },
  controlActive: {},
  panel: {
    flex: 1,
    minHeight: 0,
    position: 'relative',
  },
}))

const LayerButton = ({
  active,
  classes,
  disabled,
  label,
  onClick,
  testId,
  children,
}) => {
  const theme = useTheme()
  const activeStyle =
    active && !disabled ? { color: theme.palette.primary.main } : undefined

  return (
    <Tooltip title={label}>
      <span>
        <IconButton
          size="small"
          onClick={onClick}
          disabled={disabled}
          aria-label={label}
          aria-pressed={active}
          data-testid={testId}
          style={activeStyle}
          className={clsx(classes.controlButton, {
            [classes.controlActive]: active && !disabled,
          })}
        >
          {children}
        </IconButton>
      </span>
    </Tooltip>
  )
}

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
}) => {
  const [width, setWidth] = useState(loadSidebarWidth)
  const sidebarRef = useRef(null)
  const widthRef = useRef(width)
  const resizeCleanupRef = useRef(null)
  const { rendered, entered } = useEnterExitTransition(
    visible,
    LYRICS_SIDEBAR_TRANSITION_MS,
  )
  const [isResizing, setIsResizing] = useState(false)
  const classes = useStyles({ width })

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

  const updateWidth = useCallback((next, { persist = false } = {}) => {
    const resolvedWidth = clampSidebarWidth(
      typeof next === 'function' ? next(widthRef.current) : next,
    )
    widthRef.current = resolvedWidth
    setWidth(resolvedWidth)
    notifySidebarWidthChange(resolvedWidth)
    if (persist) saveSidebarWidth(resolvedWidth)
  }, [])

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
        latestWidth = clampSidebarWidth(startWidth + startX - moveEvent.clientX)
        updateWidth(latestWidth)
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
    [updateWidth],
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
        width,
        transform: entered ? 'translateX(0)' : 'translateX(100%)',
        opacity: entered ? 1 : 0,
        pointerEvents: entered ? 'auto' : 'none',
      }}
      aria-label="Lyrics"
      aria-hidden={!entered}
      data-resizing={isResizing ? 'true' : 'false'}
    >
      <button
        type="button"
        className={classes.resizer}
        data-testid="lyrics-sidebar-resizer"
        aria-label="Resize lyrics sidebar"
        aria-orientation="vertical"
        aria-valuemin={LYRICS_SIDEBAR_MIN_WIDTH}
        aria-valuemax={LYRICS_SIDEBAR_MAX_WIDTH}
        aria-valuenow={width}
        role="separator"
        onPointerDown={handleResizePointerDown}
        onKeyDown={handleResizeKeyDown}
      />
      <div className={classes.panel}>
        <div
          className={classes.controls}
          data-testid="lyrics-sidebar-floating-controls"
        >
          <LayerButton
            active={showPronunciation}
            classes={classes}
            disabled={!pronunciationEnabled}
            label={
              showPronunciation ? 'Hide pronunciation' : 'Show pronunciation'
            }
            onClick={onTogglePronunciation}
            testId="toggle-pronunciation-button"
          >
            <RecordVoiceOverIcon fontSize="small" />
          </LayerButton>
          <LayerButton
            active={showTranslation}
            classes={classes}
            disabled={!translationEnabled}
            label={showTranslation ? 'Hide translation' : 'Show translation'}
            onClick={onToggleTranslation}
            testId="toggle-translation-button"
          >
            <TranslateIcon fontSize="small" />
          </LayerButton>
        </div>
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
        />
      </div>
    </aside>
  )
}

export default LyricsSidebar
