import React, { useCallback, useEffect, useRef, useState } from 'react'
import { makeStyles } from '@material-ui/core/styles'
import { setSidebarVisibility } from 'react-admin'
import { useDispatch, useSelector } from 'react-redux'
import LyricsSidebar from '../audioplayer/LyricsSidebar'
import { useLyricsLayout } from '../audioplayer/LyricsLayoutContext'
import {
  LYRICS_ROUTE_MIN_WIDTH,
  LYRICS_SIDEBAR_MIN_WIDTH,
  clampSidebarWidth,
  getSidebarMaxWidth,
  loadSidebarWidth,
  saveSidebarWidth,
} from '../audioplayer/lyricsSidebarWidth'

const MIN_CONTENT_WITH_LYRICS =
  LYRICS_ROUTE_MIN_WIDTH + LYRICS_SIDEBAR_MIN_WIDTH

const useStyles = makeStyles(
  () => ({
    frame: {
      minWidth: 0,
    },
  }),
  { name: 'NDLyricsLayoutFrame' },
)

const LyricsLayoutFrame = ({ children }) => {
  const { desktopLyricsProps } = useLyricsLayout()
  const hasVisibleLyrics = Boolean(desktopLyricsProps?.visible)
  const [preferredSidebarWidth, setPreferredSidebarWidth] =
    useState(loadSidebarWidth)
  const [contentWidth, setContentWidth] = useState(null)
  const frameRef = useRef(null)
  const autoCollapsedRef = useRef(false)
  const manualSidebarOverrideRef = useRef(false)
  const pendingSidebarStateRef = useRef(null)
  const previousSidebarOpenRef = useRef(null)
  const dispatch = useDispatch()
  const sidebarOpen = useSelector(
    (state) => state.admin?.ui?.sidebarOpen ?? false,
  )
  const classes = useStyles()
  const sidebarMaxWidth = getSidebarMaxWidth(contentWidth)
  const sidebarWidth = clampSidebarWidth(preferredSidebarWidth, sidebarMaxWidth)

  useEffect(() => {
    const parent = frameRef.current?.parentElement
    if (!parent) return undefined

    const updateContentWidth = (nextWidth) => {
      const numeric = Number(nextWidth)
      if (!Number.isFinite(numeric) || numeric <= 0) return
      const rounded = Math.round(numeric)
      setContentWidth((current) => (current === rounded ? current : rounded))
    }

    const measureParent = () => {
      updateContentWidth(parent.clientWidth)
    }

    measureParent()
    const ResizeObserverConstructor = window.ResizeObserver
    if (!ResizeObserverConstructor) {
      window.addEventListener('resize', measureParent)
      return () => window.removeEventListener('resize', measureParent)
    }

    const resizeObserver = new ResizeObserverConstructor((entries) => {
      const entry = entries.find((candidate) => candidate.target === parent)
      const borderBoxSize = Array.isArray(entry?.borderBoxSize)
        ? entry.borderBoxSize[0]
        : entry?.borderBoxSize
      updateContentWidth(
        borderBoxSize?.inlineSize ||
          entry?.target?.clientWidth ||
          entry?.contentRect?.width,
      )
    })
    resizeObserver.observe(parent)
    return () => resizeObserver.disconnect()
  }, [])

  useEffect(() => {
    if (hasVisibleLyrics) {
      setPreferredSidebarWidth(loadSidebarWidth())
    }
  }, [hasVisibleLyrics])

  useEffect(() => {
    const previousSidebarOpen = previousSidebarOpenRef.current
    previousSidebarOpenRef.current = sidebarOpen
    if (previousSidebarOpen == null || previousSidebarOpen === sidebarOpen) {
      return
    }

    if (pendingSidebarStateRef.current === sidebarOpen) {
      pendingSidebarStateRef.current = null
      return
    }

    if (hasVisibleLyrics) {
      manualSidebarOverrideRef.current = true
      autoCollapsedRef.current = false
    }
  }, [hasVisibleLyrics, sidebarOpen])

  useEffect(() => {
    if (!hasVisibleLyrics) {
      if (
        autoCollapsedRef.current &&
        !manualSidebarOverrideRef.current &&
        !sidebarOpen
      ) {
        autoCollapsedRef.current = false
        pendingSidebarStateRef.current = true
        dispatch(setSidebarVisibility(true))
      }
      manualSidebarOverrideRef.current = false
      return
    }

    if (
      manualSidebarOverrideRef.current ||
      !Number.isFinite(contentWidth) ||
      contentWidth <= 0
    ) {
      return
    }

    if (sidebarOpen && contentWidth < MIN_CONTENT_WITH_LYRICS) {
      autoCollapsedRef.current = true
      pendingSidebarStateRef.current = false
      dispatch(setSidebarVisibility(false))
      return
    }
  }, [contentWidth, dispatch, hasVisibleLyrics, sidebarOpen])

  const handleSidebarWidthChange = useCallback(
    (nextWidth, { persist = false } = {}) => {
      const resolvedWidth = clampSidebarWidth(nextWidth, sidebarMaxWidth)
      setPreferredSidebarWidth(resolvedWidth)
      if (persist) saveSidebarWidth(resolvedWidth)
    },
    [sidebarMaxWidth],
  )

  return (
    <div
      ref={frameRef}
      className={classes.frame}
      data-testid="lyrics-layout-frame"
      data-lyrics-sidebar-visible={hasVisibleLyrics ? 'true' : 'false'}
      style={{
        marginRight: hasVisibleLyrics ? sidebarWidth : 0,
      }}
    >
      {children}
      {desktopLyricsProps && (
        <LyricsSidebar
          {...desktopLyricsProps}
          width={sidebarWidth}
          maxWidth={sidebarMaxWidth}
          onWidthChange={handleSidebarWidthChange}
        />
      )}
    </div>
  )
}

export default LyricsLayoutFrame
