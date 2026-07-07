import React from 'react'
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react'
import { ThemeProvider, createTheme } from '@material-ui/core/styles'
import { afterEach, describe, expect, it, vi } from 'vitest'
import LyricsSidebar from './LyricsSidebar'
import {
  LYRICS_SIDEBAR_MAX_WIDTH,
  LYRICS_SIDEBAR_MIN_WIDTH,
  LYRICS_SIDEBAR_STORAGE_KEY,
  LYRICS_SIDEBAR_TRANSITION_MS,
  clampSidebarWidth,
} from './lyricsSidebarWidth'

const theme = createTheme({
  palette: {
    primary: {
      main: '#35aa66',
    },
    background: {
      default: '#101820',
      paper: '#ffffff',
    },
  },
})
const lyric = {
  synced: true,
  line: [{ start: 0, end: 1000, value: 'Main line' }],
}
const originalPointerEvent = window.PointerEvent

const renderSidebar = (props = {}) =>
  render(
    <ThemeProvider theme={theme}>
      <LyricsSidebar
        visible
        mainLyric={lyric}
        showTranslation
        showPronunciation
        translationEnabled
        pronunciationEnabled
        onToggleTranslation={vi.fn()}
        onTogglePronunciation={vi.fn()}
        {...props}
      />
    </ThemeProvider>,
  )

describe('<LyricsSidebar />', () => {
  afterEach(() => {
    vi.useRealTimers()
    cleanup()
    localStorage.clear()
    document.body.className = ''
    Object.defineProperty(window, 'PointerEvent', {
      configurable: true,
      writable: true,
      value: originalPointerEvent,
    })
  })

  it('renders as an embedded sidebar without global layout side effects', () => {
    const { unmount } = renderSidebar()

    expect(document.body.className).toBe('')
    const sidebar = screen.getByTestId('lyrics-sidebar')
    expect(sidebar).toHaveStyle({
      transform: 'translateX(0)',
      opacity: '1',
    })
    const sidebarStyle = window.getComputedStyle(sidebar)
    expect(sidebarStyle.position).toBe('fixed')
    expect(sidebarStyle.top).toBe('48px')
    expect(sidebarStyle.bottom).toBe('80px')
    expect(sidebarStyle.backgroundColor).toBe('rgb(16, 24, 32)')
    expect(sidebarStyle.backgroundImage).toBe('none')
    expect(sidebarStyle.borderLeftWidth).toBe('1px')
    expect(sidebarStyle.boxShadow).toBe('none')
    expect(sidebarStyle.zIndex).toBe('1099')
    expect(screen.queryByRole('heading', { name: 'Lyrics' })).toBeNull()
    expect(screen.queryByTestId('lyrics-sidebar-header')).toBeNull()
    expect(screen.queryByTestId('close-lyrics-button')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Close lyrics')).not.toBeInTheDocument()

    const resizer = screen.getByTestId('lyrics-sidebar-resizer')
    const resizerStyle = window.getComputedStyle(resizer)
    expect(resizerStyle.backgroundColor).toBe('rgba(0, 0, 0, 0)')
    expect(resizerStyle.borderTopWidth).toBe('0px')

    unmount()

    expect(document.body.className).toBe('')
  })

  it('keeps the sidebar mounted while sliding closed', () => {
    vi.useFakeTimers()
    const { rerender } = renderSidebar()

    rerender(
      <ThemeProvider theme={theme}>
        <LyricsSidebar
          visible={false}
          mainLyric={lyric}
          showTranslation
          showPronunciation
          translationEnabled
          pronunciationEnabled
          onToggleTranslation={vi.fn()}
          onTogglePronunciation={vi.fn()}
        />
      </ThemeProvider>,
    )

    expect(screen.getByTestId('lyrics-sidebar')).toHaveStyle({
      transform: 'translateX(100%)',
      opacity: '0',
    })
    expect(document.body.className).toBe('')

    vi.advanceTimersByTime(LYRICS_SIDEBAR_TRANSITION_MS)

    expect(screen.queryByTestId('lyrics-sidebar')).toBeNull()
    vi.useRealTimers()
  })

  it('clamps persisted and keyboard-resized widths', () => {
    localStorage.setItem(LYRICS_SIDEBAR_STORAGE_KEY, '999')
    renderSidebar()

    const sidebar = screen.getByTestId('lyrics-sidebar')
    const resizer = screen.getByTestId('lyrics-sidebar-resizer')

    expect(sidebar).toHaveStyle({ width: `${LYRICS_SIDEBAR_MAX_WIDTH}px` })
    expect(resizer).toHaveAttribute(
      'aria-valuenow',
      String(LYRICS_SIDEBAR_MAX_WIDTH),
    )

    fireEvent.keyDown(resizer, { key: 'Home' })

    expect(sidebar).toHaveStyle({ width: `${LYRICS_SIDEBAR_MIN_WIDTH}px` })
    expect(localStorage.getItem(LYRICS_SIDEBAR_STORAGE_KEY)).toBe(
      String(LYRICS_SIDEBAR_MIN_WIDTH),
    )

    fireEvent.keyDown(resizer, { key: 'End' })

    expect(sidebar).toHaveStyle({ width: `${LYRICS_SIDEBAR_MAX_WIDTH}px` })
    expect(localStorage.getItem(LYRICS_SIDEBAR_STORAGE_KEY)).toBe(
      String(LYRICS_SIDEBAR_MAX_WIDTH),
    )
  })

  it('clamps pointer resizing from the left separator', async () => {
    Object.defineProperty(window, 'PointerEvent', {
      configurable: true,
      writable: true,
      value: MouseEvent,
    })
    renderSidebar()

    const sidebar = screen.getByTestId('lyrics-sidebar')
    const resizer = screen.getByTestId('lyrics-sidebar-resizer')

    fireEvent.pointerDown(resizer, { clientX: 500 })
    window.dispatchEvent(new MouseEvent('pointermove', { clientX: -100 }))
    await waitFor(() =>
      expect(sidebar).toHaveStyle({ width: `${LYRICS_SIDEBAR_MAX_WIDTH}px` }),
    )
    expect(localStorage.getItem(LYRICS_SIDEBAR_STORAGE_KEY)).toBeNull()

    window.dispatchEvent(new MouseEvent('pointermove', { clientX: 1000 }))
    await waitFor(() =>
      expect(sidebar).toHaveStyle({ width: `${LYRICS_SIDEBAR_MIN_WIDTH}px` }),
    )

    window.dispatchEvent(new MouseEvent('pointerup'))
    expect(localStorage.getItem(LYRICS_SIDEBAR_STORAGE_KEY)).toBe(
      String(LYRICS_SIDEBAR_MIN_WIDTH),
    )
  })

  it('cleans pointer resizing on cancellation and unmount', async () => {
    Object.defineProperty(window, 'PointerEvent', {
      configurable: true,
      writable: true,
      value: MouseEvent,
    })
    const { unmount } = renderSidebar()

    const sidebar = screen.getByTestId('lyrics-sidebar')
    const resizer = screen.getByTestId('lyrics-sidebar-resizer')

    fireEvent.pointerDown(resizer, { clientX: 500 })
    window.dispatchEvent(new MouseEvent('pointermove', { clientX: -100 }))
    await waitFor(() =>
      expect(sidebar).toHaveStyle({ width: `${LYRICS_SIDEBAR_MAX_WIDTH}px` }),
    )

    window.dispatchEvent(new MouseEvent('pointercancel'))
    await waitFor(() =>
      expect(sidebar).toHaveAttribute('data-resizing', 'false'),
    )

    window.dispatchEvent(new MouseEvent('pointermove', { clientX: 1000 }))
    expect(sidebar).toHaveStyle({ width: `${LYRICS_SIDEBAR_MAX_WIDTH}px` })
    expect(localStorage.getItem(LYRICS_SIDEBAR_STORAGE_KEY)).toBeNull()

    fireEvent.pointerDown(resizer, { clientX: 500 })
    window.dispatchEvent(new MouseEvent('pointermove', { clientX: 1000 }))
    await waitFor(() =>
      expect(sidebar).toHaveStyle({ width: `${LYRICS_SIDEBAR_MIN_WIDTH}px` }),
    )

    unmount()
    window.dispatchEvent(new MouseEvent('pointerup'))

    expect(localStorage.getItem(LYRICS_SIDEBAR_STORAGE_KEY)).toBeNull()
  })

  it('keeps resizing through window listeners when pointer capture fails', async () => {
    Object.defineProperty(window, 'PointerEvent', {
      configurable: true,
      writable: true,
      value: MouseEvent,
    })
    renderSidebar()

    const sidebar = screen.getByTestId('lyrics-sidebar')
    const resizer = screen.getByTestId('lyrics-sidebar-resizer')
    resizer.setPointerCapture = vi.fn(() => {
      throw new Error('pointer capture unavailable')
    })
    resizer.releasePointerCapture = vi.fn()

    fireEvent.pointerDown(resizer, { clientX: 500, pointerId: 1 })
    window.dispatchEvent(new MouseEvent('pointermove', { clientX: -100 }))

    await waitFor(() =>
      expect(sidebar).toHaveStyle({ width: `${LYRICS_SIDEBAR_MAX_WIDTH}px` }),
    )

    window.dispatchEvent(new MouseEvent('pointerup'))

    expect(localStorage.getItem(LYRICS_SIDEBAR_STORAGE_KEY)).toBe(
      String(LYRICS_SIDEBAR_MAX_WIDTH),
    )
  })

  it('blurs focus inside the sidebar when exit transition hides it', () => {
    vi.useFakeTimers()
    const { rerender } = renderSidebar()
    const resizer = screen.getByTestId('lyrics-sidebar-resizer')
    resizer.focus()
    expect(document.activeElement).toBe(resizer)

    rerender(
      <ThemeProvider theme={theme}>
        <LyricsSidebar
          visible={false}
          mainLyric={lyric}
          showTranslation
          showPronunciation
          translationEnabled
          pronunciationEnabled
          onToggleTranslation={vi.fn()}
          onTogglePronunciation={vi.fn()}
        />
      </ThemeProvider>,
    )

    expect(document.activeElement).not.toBe(resizer)
    vi.advanceTimersByTime(LYRICS_SIDEBAR_TRANSITION_MS)
  })

  it('uses icon toggle buttons with pressed and disabled states', () => {
    const onTogglePronunciation = vi.fn()
    const onToggleTranslation = vi.fn()
    renderSidebar({
      showPronunciation: true,
      showTranslation: false,
      pronunciationEnabled: true,
      translationEnabled: false,
      onTogglePronunciation,
      onToggleTranslation,
    })

    const pronunciation = screen.getByTestId('toggle-pronunciation-button')
    const translation = screen.getByTestId('toggle-translation-button')
    const controls = screen.getByTestId('lyrics-sidebar-floating-controls')

    expect(controls).toContainElement(pronunciation)
    expect(controls).toContainElement(translation)
    expect(pronunciation).toHaveAttribute('aria-label', 'Hide pronunciation')
    expect(pronunciation).toHaveAttribute('aria-pressed', 'true')
    expect(window.getComputedStyle(pronunciation).color).toBe(
      'rgb(53, 170, 102)',
    )
    expect(pronunciation.querySelector('svg')).toBeInTheDocument()
    fireEvent.click(pronunciation)
    expect(onTogglePronunciation).toHaveBeenCalledTimes(1)

    expect(translation).toHaveAttribute('aria-label', 'Show translation')
    expect(translation).toHaveAttribute('aria-pressed', 'false')
    expect(translation).toBeDisabled()
    expect(window.getComputedStyle(translation).color).not.toBe(
      'rgb(53, 170, 102)',
    )
    fireEvent.click(translation)
    expect(onToggleTranslation).not.toHaveBeenCalled()
  })

  it('stays mounted without re-entering when an open sidebar receives no lyrics', () => {
    const { rerender } = renderSidebar()

    const sidebar = screen.getByTestId('lyrics-sidebar')
    expect(sidebar).toHaveStyle({ transform: 'translateX(0)' })
    expect(screen.getByTestId('karaoke-lyrics-panel')).toBeInTheDocument()

    rerender(
      <ThemeProvider theme={theme}>
        <LyricsSidebar
          visible
          mainLyric={null}
          showTranslation={false}
          showPronunciation={false}
          translationEnabled={false}
          pronunciationEnabled={false}
          onToggleTranslation={vi.fn()}
          onTogglePronunciation={vi.fn()}
        />
      </ThemeProvider>,
    )

    expect(screen.getByTestId('lyrics-sidebar')).toHaveStyle({
      transform: 'translateX(0)',
    })
    expect(screen.getByTestId('karaoke-lyrics-panel')).toBeInTheDocument()
    expect(screen.getByTestId('lyrics-empty-state')).toHaveTextContent(
      'No lyrics available',
    )
    expect(screen.queryByTestId('lyrics-line-group')).toBeNull()
    expect(screen.getByTestId('toggle-pronunciation-button')).toBeDisabled()
    expect(screen.getByTestId('toggle-translation-button')).toBeDisabled()
  })

  it('exposes fixed min and max clamp values', () => {
    expect(clampSidebarWidth(10)).toBe(LYRICS_SIDEBAR_MIN_WIDTH)
    expect(clampSidebarWidth(999)).toBe(LYRICS_SIDEBAR_MAX_WIDTH)
  })
})
