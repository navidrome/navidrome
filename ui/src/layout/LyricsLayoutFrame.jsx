import React, { useEffect, useState } from 'react'
import clsx from 'clsx'
import { makeStyles } from '@material-ui/core/styles'
import LyricsSidebar from '../audioplayer/LyricsSidebar'
import { useLyricsLayout } from '../audioplayer/LyricsLayoutContext'
import { LYRICS_SIDEBAR_TRANSITION_MS } from '../audioplayer/lyricsSidebarWidth'

const DESKTOP_PLAYER_HEIGHT = 80
const DESKTOP_APP_BAR_HEIGHT = 48

const useStyles = makeStyles(
  (theme) => ({
    frame: {
      width: '100%',
      minWidth: 0,
    },
    frameWithLyrics: {
      display: 'flex',
      alignItems: 'stretch',
      gap: 0,
      height: `calc(100vh - ${DESKTOP_APP_BAR_HEIGHT + DESKTOP_PLAYER_HEIGHT}px)`,
      minHeight: 0,
      overflow: 'hidden',
      backgroundColor: theme.palette.background.default,
    },
    content: {
      minWidth: 0,
      width: '100%',
    },
    contentWithLyrics: {
      flex: '1 1 auto',
      overflowY: 'auto',
      overflowX: 'hidden',
      overscrollBehavior: 'contain',
    },
  }),
  { name: 'NDLyricsLayoutFrame' },
)

const LyricsLayoutFrame = ({ children }) => {
  const classes = useStyles()
  const { desktopLyricsProps } = useLyricsLayout()
  const hasVisibleLyrics = Boolean(desktopLyricsProps?.visible)
  const [layoutActive, setLayoutActive] = useState(hasVisibleLyrics)

  useEffect(() => {
    if (hasVisibleLyrics) {
      setLayoutActive(true)
      return undefined
    }

    const timer = window.setTimeout(() => {
      setLayoutActive(false)
    }, LYRICS_SIDEBAR_TRANSITION_MS)
    return () => window.clearTimeout(timer)
  }, [hasVisibleLyrics])

  const hasLyricsHost = Boolean(
    desktopLyricsProps && (hasVisibleLyrics || layoutActive),
  )

  return (
    <div
      className={clsx(classes.frame, {
        [classes.frameWithLyrics]: hasLyricsHost,
      })}
      data-testid="lyrics-layout-frame"
      data-lyrics-layout-active={hasLyricsHost ? 'true' : 'false'}
      data-lyrics-sidebar-visible={hasVisibleLyrics ? 'true' : 'false'}
    >
      <div
        className={clsx(classes.content, {
          [classes.contentWithLyrics]: hasLyricsHost,
        })}
        data-testid="lyrics-layout-content"
      >
        {children}
      </div>
      {desktopLyricsProps && <LyricsSidebar {...desktopLyricsProps} />}
    </div>
  )
}

export default LyricsLayoutFrame
