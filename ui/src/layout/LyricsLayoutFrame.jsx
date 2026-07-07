import React, { useEffect, useState } from 'react'
import { makeStyles } from '@material-ui/core/styles'
import LyricsSidebar from '../audioplayer/LyricsSidebar'
import { useLyricsLayout } from '../audioplayer/LyricsLayoutContext'
import {
  LYRICS_SIDEBAR_WIDTH_EVENT,
  clampSidebarWidth,
  loadSidebarWidth,
} from '../audioplayer/lyricsSidebarWidth'

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
  const [sidebarWidth, setSidebarWidth] = useState(loadSidebarWidth)
  const classes = useStyles()

  useEffect(() => {
    if (!hasVisibleLyrics) return undefined

    const handleSidebarWidth = (event) => {
      const next = clampSidebarWidth(event.detail?.width)
      setSidebarWidth((current) => (current === next ? current : next))
    }

    setSidebarWidth(loadSidebarWidth())
    window.addEventListener(LYRICS_SIDEBAR_WIDTH_EVENT, handleSidebarWidth)
    return () =>
      window.removeEventListener(LYRICS_SIDEBAR_WIDTH_EVENT, handleSidebarWidth)
  }, [hasVisibleLyrics])

  return (
    <div
      className={classes.frame}
      data-testid="lyrics-layout-frame"
      data-lyrics-sidebar-visible={hasVisibleLyrics ? 'true' : 'false'}
      style={{
        marginRight: hasVisibleLyrics ? sidebarWidth : 0,
      }}
    >
      {children}
      {desktopLyricsProps && <LyricsSidebar {...desktopLyricsProps} />}
    </div>
  )
}

export default LyricsLayoutFrame
