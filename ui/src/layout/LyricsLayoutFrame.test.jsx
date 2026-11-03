import React, { useEffect } from 'react'
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react'
import { ThemeProvider, createTheme } from '@material-ui/core/styles'
import { setSidebarVisibility } from 'react-admin'
import { Provider } from 'react-redux'
import { createStore } from 'redux'
import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  LyricsLayoutProvider,
  useLyricsLayout,
} from '../audioplayer/LyricsLayoutContext'
import {
  LYRICS_SIDEBAR_STORAGE_KEY,
  LYRICS_SIDEBAR_TRANSITION_MS,
} from '../audioplayer/lyricsSidebarWidth'
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

const originalResizeObserver = window.ResizeObserver
const resizeObservers = []

class TestResizeObserver {
  constructor(callback) {
    this.callback = callback
    this.target = null
    this.disconnected = false
    resizeObservers.push(this)
  }

  observe(target) {
    this.target = target
  }

  disconnect() {
    this.disconnected = true
  }
}

const notifyContentWidth = (width) => {
  act(() => {
    resizeObservers.forEach((observer) => {
      if (observer.disconnected || !observer.target) return
      observer.callback([
        {
          target: observer.target,
          contentRect: { width },
          borderBoxSize: [{ inlineSize: width }],
        },
      ])
    })
  })
}

const sidebarVisibilityActionType = setSidebarVisibility(true).type

const createLayoutStore = (sidebarOpen = false) =>
  createStore(
    (state, action) => {
      if (action.type !== sidebarVisibilityActionType) return state
      return {
        ...state,
        admin: {
          ...state.admin,
          ui: {
            ...state.admin.ui,
            sidebarOpen: action.payload,
          },
        },
      }
    },
    { admin: { ui: { sidebarOpen } } },
  )

const PublishLyricsProps = ({ props }) => {
  const { setDesktopLyricsProps } = useLyricsLayout()

  useEffect(() => {
    setDesktopLyricsProps(props)
    return () => setDesktopLyricsProps(null)
  }, [props, setDesktopLyricsProps])

  return null
}

const frame = (props, store) => (
  <Provider store={store}>
    <ThemeProvider theme={theme}>
      <LyricsLayoutProvider>
        <PublishLyricsProps props={props} />
        <LyricsLayoutFrame>
          <div data-testid="route-content">Albums grid</div>
        </LyricsLayoutFrame>
      </LyricsLayoutProvider>
    </ThemeProvider>
  </Provider>
)

const renderFrame = (props, { sidebarOpen = false, store } = {}) => {
  window.ResizeObserver = TestResizeObserver
  const layoutStore = store || createLayoutStore(sidebarOpen)
  const view = render(frame(props, layoutStore))
  return {
    ...view,
    store: layoutStore,
    rerenderFrame: (nextProps) => view.rerender(frame(nextProps, layoutStore)),
  }
}

describe('<LyricsLayoutFrame />', () => {
  afterEach(() => {
    vi.useRealTimers()
    cleanup()
    localStorage.clear()
    resizeObservers.splice(0)
    window.ResizeObserver = originalResizeObserver
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

  it('caps a persisted width against the measured route reserve without overwriting it', async () => {
    localStorage.setItem(LYRICS_SIDEBAR_STORAGE_KEY, '520')
    renderFrame(lyricsProps)
    const sidebar = await screen.findByTestId('lyrics-sidebar')
    const resizer = screen.getByTestId('lyrics-sidebar-resizer')

    notifyContentWidth(755)

    expect(screen.getByTestId('lyrics-layout-frame')).toHaveStyle({
      marginRight: '435px',
    })
    expect(sidebar).toHaveStyle({ width: '435px' })
    expect(resizer).toHaveAttribute('aria-valuemax', '435')
    expect(resizer).toHaveAttribute('aria-valuenow', '435')
    expect(localStorage.getItem(LYRICS_SIDEBAR_STORAGE_KEY)).toBe('520')

    notifyContentWidth(840)

    expect(screen.getByTestId('lyrics-layout-frame')).toHaveStyle({
      marginRight: '520px',
    })
    expect(sidebar).toHaveStyle({ width: '520px' })
    expect(resizer).toHaveAttribute('aria-valuemax', '520')
    expect(resizer).toHaveAttribute('aria-valuenow', '520')
    expect(localStorage.getItem(LYRICS_SIDEBAR_STORAGE_KEY)).toBe('520')
  })

  it('keeps the layout width reserved while the queue obscures lyrics', async () => {
    localStorage.setItem(LYRICS_SIDEBAR_STORAGE_KEY, '520')
    renderFrame({ ...lyricsProps, obscuredByQueue: true })
    const sidebar = await screen.findByTestId('lyrics-sidebar')

    notifyContentWidth(755)

    expect(screen.getByTestId('lyrics-layout-frame')).toHaveStyle({
      marginRight: '435px',
    })
    expect(sidebar).toHaveStyle({
      width: '435px',
      transform: 'translateX(0)',
      pointerEvents: 'none',
    })
    expect(sidebar).toHaveAttribute('aria-hidden', 'true')
    expect(sidebar).toHaveAttribute('inert')
    expect(screen.getByTestId('karaoke-lyrics-panel')).toBeInTheDocument()
  })

  it('collapses an open app menu only when the minimum pane cannot fit', async () => {
    localStorage.setItem(LYRICS_SIDEBAR_STORAGE_KEY, '520')
    const { rerenderFrame, store } = renderFrame(lyricsProps, {
      sidebarOpen: true,
    })
    await screen.findByTestId('lyrics-sidebar')

    notifyContentWidth(570)

    await waitFor(() =>
      expect(store.getState().admin.ui.sidebarOpen).toBe(false),
    )
    expect(screen.getByTestId('lyrics-layout-frame')).toHaveStyle({
      marginRight: '300px',
    })
    expect(screen.getByTestId('lyrics-sidebar-resizer')).toHaveAttribute(
      'aria-valuemax',
      '300',
    )

    notifyContentWidth(755)

    expect(screen.getByTestId('lyrics-layout-frame')).toHaveStyle({
      marginRight: '435px',
    })
    expect(store.getState().admin.ui.sidebarOpen).toBe(false)

    rerenderFrame({ ...lyricsProps, visible: false })

    await waitFor(() =>
      expect(store.getState().admin.ui.sidebarOpen).toBe(true),
    )
  })

  it('keeps the app menu open at the 960px and 1280px layout caps', async () => {
    localStorage.setItem(LYRICS_SIDEBAR_STORAGE_KEY, '520')
    const { store } = renderFrame(lyricsProps, { sidebarOpen: true })
    await screen.findByTestId('lyrics-sidebar')

    notifyContentWidth(720)

    expect(store.getState().admin.ui.sidebarOpen).toBe(true)
    expect(screen.getByTestId('lyrics-layout-frame')).toHaveStyle({
      marginRight: '400px',
    })
    expect(screen.getByTestId('lyrics-sidebar-resizer')).toHaveAttribute(
      'aria-valuemax',
      '400',
    )

    notifyContentWidth(1040)

    expect(screen.getByTestId('lyrics-layout-frame')).toHaveStyle({
      marginRight: '520px',
    })
    expect(screen.getByTestId('lyrics-sidebar-resizer')).toHaveAttribute(
      'aria-valuemax',
      '520',
    )
  })

  it('keeps an automatic collapse until lyrics close when space returns', async () => {
    const { rerenderFrame, store } = renderFrame(lyricsProps, {
      sidebarOpen: true,
    })
    await screen.findByTestId('lyrics-sidebar')
    notifyContentWidth(570)
    await waitFor(() =>
      expect(store.getState().admin.ui.sidebarOpen).toBe(false),
    )

    notifyContentWidth(1040)

    expect(store.getState().admin.ui.sidebarOpen).toBe(false)

    rerenderFrame({ ...lyricsProps, visible: false })

    await waitFor(() =>
      expect(store.getState().admin.ui.sidebarOpen).toBe(true),
    )
  })

  it('does not restore the menu after a manual override', async () => {
    const { rerenderFrame, store } = renderFrame(lyricsProps, {
      sidebarOpen: true,
    })
    await screen.findByTestId('lyrics-sidebar')
    notifyContentWidth(570)
    await waitFor(() =>
      expect(store.getState().admin.ui.sidebarOpen).toBe(false),
    )

    act(() => {
      store.dispatch(setSidebarVisibility(true))
    })
    await waitFor(() =>
      expect(store.getState().admin.ui.sidebarOpen).toBe(true),
    )
    act(() => {
      store.dispatch(setSidebarVisibility(false))
    })
    await waitFor(() =>
      expect(store.getState().admin.ui.sidebarOpen).toBe(false),
    )

    rerenderFrame({ ...lyricsProps, visible: false })

    expect(store.getState().admin.ui.sidebarOpen).toBe(false)
  })

  it('persists deliberate keyboard resizing at the measured cap', async () => {
    localStorage.setItem(LYRICS_SIDEBAR_STORAGE_KEY, '520')
    renderFrame(lyricsProps)
    await screen.findByTestId('lyrics-sidebar')
    notifyContentWidth(755)
    const resizer = screen.getByTestId('lyrics-sidebar-resizer')

    fireEvent.keyDown(resizer, { key: 'Home' })

    expect(screen.getByTestId('lyrics-layout-frame')).toHaveStyle({
      marginRight: '300px',
    })
    expect(localStorage.getItem(LYRICS_SIDEBAR_STORAGE_KEY)).toBe('300')

    fireEvent.keyDown(resizer, { key: 'End' })

    expect(screen.getByTestId('lyrics-layout-frame')).toHaveStyle({
      marginRight: '435px',
    })
    expect(resizer).toHaveAttribute('aria-valuenow', '435')
    expect(localStorage.getItem(LYRICS_SIDEBAR_STORAGE_KEY)).toBe('435')
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
