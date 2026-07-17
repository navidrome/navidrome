import React, { useEffect } from 'react'
import { act, cleanup, render, screen } from '@testing-library/react'
import { ThemeProvider, createTheme } from '@material-ui/core/styles'
import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  LyricsLayoutProvider,
  useLyricsLayout,
} from '../audioplayer/LyricsLayoutContext'
import { LYRICS_SIDEBAR_TRANSITION_MS } from '../audioplayer/lyricsSidebarWidth'
import LyricsLayoutFrame from './LyricsLayoutFrame'

const theme = createTheme({
  palette: {
    primary: { main: '#35aa66' },
    background: { default: '#101820' },
  },
})

const lyric = {
  synced: true,
  line: [{ start: 0, end: 1000, value: 'Main line' }],
}

const lyricsProps = {
  visible: true,
  mainLyric: lyric,
  showTranslation: false,
  showPronunciation: false,
  translationEnabled: false,
  pronunciationEnabled: false,
  onToggleTranslation: vi.fn(),
  onTogglePronunciation: vi.fn(),
}

const PublishLyricsProps = ({ props }) => {
  const { setDesktopLyricsProps } = useLyricsLayout()

  useEffect(() => {
    setDesktopLyricsProps(props)
    return () => setDesktopLyricsProps(null)
  }, [props, setDesktopLyricsProps])

  return null
}

const frame = (props) => (
  <ThemeProvider theme={theme}>
    <LyricsLayoutProvider>
      <PublishLyricsProps props={props} />
      <LyricsLayoutFrame>
        <div data-testid="route-content">Albums grid</div>
      </LyricsLayoutFrame>
    </LyricsLayoutProvider>
  </ThemeProvider>
)

const renderFrame = (props) => {
  const view = render(frame(props))
  return {
    ...view,
    rerenderFrame: (nextProps) => view.rerender(frame(nextProps)),
  }
}

describe('<LyricsLayoutFrame />', () => {
  afterEach(() => {
    vi.useRealTimers()
    cleanup()
  })

  it('reserves a native right-hand lyrics pane beside route content', async () => {
    renderFrame(lyricsProps)

    const sidebar = await screen.findByTestId('lyrics-sidebar')
    const frame = screen.getByTestId('lyrics-layout-frame')
    const content = screen.getByTestId('route-content')

    expect(frame).toHaveAttribute('data-lyrics-sidebar-visible', 'true')
    expect(frame).toHaveStyle({ marginRight: '360px' })
    expect(frame).toContainElement(content)
    expect(frame).toContainElement(sidebar)
    expect(window.getComputedStyle(sidebar).position).toBe('fixed')
  })

  it('releases the layout frame after the sidebar exit transition', async () => {
    vi.useFakeTimers()
    const { rerenderFrame } = renderFrame(lyricsProps)

    await screen.findByTestId('lyrics-sidebar')

    rerenderFrame({ ...lyricsProps, visible: false })

    expect(screen.getByTestId('lyrics-layout-frame')).toHaveAttribute(
      'data-lyrics-sidebar-visible',
      'false',
    )
    expect(screen.getByTestId('lyrics-layout-frame')).toHaveStyle({
      marginRight: '0px',
    })
    expect(screen.getByTestId('lyrics-sidebar')).toHaveStyle({
      transform: 'translateX(100%)',
    })

    act(() => {
      vi.advanceTimersByTime(LYRICS_SIDEBAR_TRANSITION_MS)
    })

    expect(screen.getByTestId('lyrics-layout-frame')).toHaveAttribute(
      'data-lyrics-sidebar-visible',
      'false',
    )
    expect(screen.queryByTestId('lyrics-sidebar')).toBeNull()
  })
})
