import React, { useCallback, useState } from 'react'
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
  LYRICS_SIDEBAR_DEFAULT_WIDTH,
  LYRICS_SIDEBAR_MAX_WIDTH,
  LYRICS_SIDEBAR_MIN_WIDTH,
  LYRICS_SIDEBAR_TRANSITION_MS,
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
const source = {
  type: 'plugin',
  name: 'Better Lyrics',
  provider: 'ttml',
  format: 'ttml',
}
const originalPointerEvent = window.PointerEvent

const ControlledLyricsSidebar = ({
  initialWidth = LYRICS_SIDEBAR_DEFAULT_WIDTH,
  maxWidth = LYRICS_SIDEBAR_MAX_WIDTH,
  onWidthChange,
  ...props
}) => {
  const [width, setWidth] = useState(initialWidth)
  const handleWidthChange = useCallback(
    (nextWidth, options) => {
      setWidth(nextWidth)
      onWidthChange?.(nextWidth, options)
    },
    [onWidthChange],
  )

  return (
    <LyricsSidebar
      width={width}
      maxWidth={maxWidth}
      onWidthChange={handleWidthChange}
      {...props}
    />
  )
}

const sidebarView = (props = {}) => (
  <ThemeProvider theme={theme}>
    <ControlledLyricsSidebar
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
  </ThemeProvider>
)

const renderSidebar = (props) => render(sidebarView(props))

describe('<LyricsSidebar />', () => {
  afterEach(() => {
    vi.useRealTimers()
    cleanup()
    localStorage.clear()
    document.body.className = ''
    document
      .querySelectorAll('[data-lyrics-return-target]')
      .forEach((node) => node.remove())
    Object.defineProperty(window, 'PointerEvent', {
      configurable: true,
      writable: true,
      value: originalPointerEvent,
    })
  })

  it('renders as an embedded sidebar without global layout side effects', () => {
    renderSidebar()

    expect(document.body.className).toBe('')
    const sidebar = screen.getByTestId('lyrics-sidebar')
    expect(sidebar).toHaveStyle({
      transform: 'translateX(0)',
      opacity: '1',
    })
    expect(screen.getByTestId('lyrics-sidebar-resizer')).toBeInTheDocument()
  })

  it('keeps the sidebar mounted while sliding closed', () => {
    vi.useFakeTimers()
    const returnTarget = document.createElement('button')
    returnTarget.dataset.lyricsReturnTarget = 'true'
    document.body.appendChild(returnTarget)
    const returnFocusRef = { current: returnTarget }
    const { rerender } = renderSidebar({ returnFocusRef })
    const sidebar = screen.getByTestId('lyrics-sidebar')
    const resizer = screen.getByTestId('lyrics-sidebar-resizer')
    expect(sidebar).not.toHaveAttribute('inert')
    resizer.focus()

    rerender(sidebarView({ visible: false, returnFocusRef }))

    expect(sidebar).toHaveStyle({
      transform: 'translateX(100%)',
      opacity: '0',
    })
    expect(sidebar).toHaveAttribute('aria-hidden', 'true')
    expect(sidebar).toHaveAttribute('inert')
    expect(document.body.className).toBe('')
    expect(document.activeElement).toBe(returnTarget)

    vi.advanceTimersByTime(LYRICS_SIDEBAR_TRANSITION_MS)

    expect(screen.queryByTestId('lyrics-sidebar')).toBeNull()
    vi.useRealTimers()
  })

  it('restores focus if a breakpoint unmounts the open sidebar', () => {
    const returnTarget = document.createElement('button')
    returnTarget.dataset.lyricsReturnTarget = 'true'
    document.body.appendChild(returnTarget)
    const returnFocusRef = { current: returnTarget }
    const { unmount } = renderSidebar({ returnFocusRef })
    const resizer = screen.getByTestId('lyrics-sidebar-resizer')
    resizer.focus()

    unmount()

    expect(document.activeElement).toBe(returnTarget)
  })

  it('keeps lyrics mounted but noninteractive while the queue obscures them', () => {
    const returnTarget = document.createElement('button')
    returnTarget.dataset.lyricsReturnTarget = 'true'
    document.body.appendChild(returnTarget)
    const returnFocusRef = { current: returnTarget }
    const { rerender } = renderSidebar({ returnFocusRef })
    const sidebar = screen.getByTestId('lyrics-sidebar')
    const panel = screen.getByTestId('karaoke-lyrics-panel')
    screen.getByTestId('lyrics-sidebar-resizer').focus()

    rerender(
      sidebarView({
        obscuredByQueue: true,
        returnFocusRef,
      }),
    )

    expect(screen.getByTestId('lyrics-sidebar')).toBe(sidebar)
    expect(screen.getByTestId('karaoke-lyrics-panel')).toBe(panel)
    expect(sidebar).toHaveStyle({
      transform: 'translateX(0)',
      opacity: '1',
      pointerEvents: 'none',
    })
    expect(sidebar).toHaveAttribute('aria-hidden', 'true')
    expect(sidebar).toHaveAttribute('inert')
    expect(document.activeElement).toBe(returnTarget)

    rerender(sidebarView({ returnFocusRef }))

    expect(screen.getByTestId('lyrics-sidebar')).toBe(sidebar)
    expect(screen.getByTestId('karaoke-lyrics-panel')).toBe(panel)
    expect(sidebar).toHaveStyle({ pointerEvents: 'auto' })
    expect(sidebar).toHaveAttribute('aria-hidden', 'false')
    expect(sidebar).not.toHaveAttribute('inert')
  })

  it('closes the source popover when the queue obscures the sidebar', async () => {
    const { rerender } = renderSidebar({ source })

    fireEvent.click(screen.getByRole('button', { name: 'View lyrics source' }))
    expect(screen.getByRole('dialog', { name: 'Lyrics source' })).toBeVisible()

    rerender(sidebarView({ source, obscuredByQueue: true }))

    await waitFor(() =>
      expect(
        screen.queryByRole('dialog', { name: 'Lyrics source' }),
      ).not.toBeInTheDocument(),
    )
  })

  it('clamps controlled keyboard resizing to the available width', () => {
    const onWidthChange = vi.fn()
    renderSidebar({ initialWidth: 999, maxWidth: 435, onWidthChange })

    const sidebar = screen.getByTestId('lyrics-sidebar')
    const resizer = screen.getByTestId('lyrics-sidebar-resizer')

    expect(sidebar).toHaveStyle({ width: '435px' })
    expect(resizer).toHaveAttribute('aria-valuemax', '435')
    expect(resizer).toHaveAttribute('aria-valuenow', '435')

    fireEvent.keyDown(resizer, { key: 'Home' })

    expect(sidebar).toHaveStyle({ width: `${LYRICS_SIDEBAR_MIN_WIDTH}px` })
    expect(onWidthChange).toHaveBeenLastCalledWith(LYRICS_SIDEBAR_MIN_WIDTH, {
      persist: true,
    })

    fireEvent.keyDown(resizer, { key: 'End' })

    expect(sidebar).toHaveStyle({ width: '435px' })
    expect(resizer).toHaveAttribute('aria-valuenow', '435')
    expect(onWidthChange).toHaveBeenLastCalledWith(435, { persist: true })
  })

  it('clamps pointer resizing from the left separator', async () => {
    Object.defineProperty(window, 'PointerEvent', {
      configurable: true,
      writable: true,
      value: MouseEvent,
    })
    const onWidthChange = vi.fn()
    renderSidebar({ maxWidth: 435, onWidthChange })

    const sidebar = screen.getByTestId('lyrics-sidebar')
    const resizer = screen.getByTestId('lyrics-sidebar-resizer')
    resizer.setPointerCapture = vi.fn(() => {
      throw new Error('pointer capture unavailable')
    })
    resizer.releasePointerCapture = vi.fn()

    fireEvent.pointerDown(resizer, { clientX: 500 })
    window.dispatchEvent(new MouseEvent('pointermove', { clientX: -100 }))
    await waitFor(() => expect(sidebar).toHaveStyle({ width: '435px' }))
    expect(resizer).toHaveAttribute('aria-valuemax', '435')
    expect(resizer).toHaveAttribute('aria-valuenow', '435')
    expect(onWidthChange).toHaveBeenLastCalledWith(435, { persist: false })

    window.dispatchEvent(new MouseEvent('pointermove', { clientX: 1000 }))
    await waitFor(() =>
      expect(sidebar).toHaveStyle({ width: `${LYRICS_SIDEBAR_MIN_WIDTH}px` }),
    )

    window.dispatchEvent(new MouseEvent('pointerup'))
    expect(onWidthChange).toHaveBeenLastCalledWith(LYRICS_SIDEBAR_MIN_WIDTH, {
      persist: true,
    })
  })

  it('keeps the live pointer width through an unrelated rerender', async () => {
    Object.defineProperty(window, 'PointerEvent', {
      configurable: true,
      writable: true,
      value: MouseEvent,
    })
    const { rerender } = renderSidebar()
    const sidebar = screen.getByTestId('lyrics-sidebar')
    const resizer = screen.getByTestId('lyrics-sidebar-resizer')

    fireEvent.pointerDown(resizer, { clientX: 500 })
    window.dispatchEvent(new MouseEvent('pointermove', { clientX: 450 }))
    await waitFor(() => expect(sidebar).toHaveStyle({ width: '410px' }))
    expect(resizer).toHaveAttribute('aria-valuenow', '410')

    rerender(sidebarView({ labels: { title: 'Updated lyrics' } }))

    expect(screen.getByTestId('lyrics-sidebar')).toHaveStyle({ width: '410px' })
    expect(screen.getByTestId('lyrics-sidebar-resizer')).toHaveAttribute(
      'aria-valuenow',
      '410',
    )
    window.dispatchEvent(new MouseEvent('pointercancel'))
  })

  it('cleans pointer resizing on cancellation and unmount', async () => {
    Object.defineProperty(window, 'PointerEvent', {
      configurable: true,
      writable: true,
      value: MouseEvent,
    })
    const onWidthChange = vi.fn()
    const { unmount } = renderSidebar({ onWidthChange })

    const sidebar = screen.getByTestId('lyrics-sidebar')
    const resizer = screen.getByTestId('lyrics-sidebar-resizer')

    fireEvent.pointerDown(resizer, { clientX: 500 })
    window.dispatchEvent(new MouseEvent('pointermove', { clientX: -100 }))
    await waitFor(() =>
      expect(sidebar).toHaveStyle({ width: `${LYRICS_SIDEBAR_MAX_WIDTH}px` }),
    )

    window.dispatchEvent(new MouseEvent('pointercancel'))
    const callsAfterCancel = onWidthChange.mock.calls.length

    window.dispatchEvent(new MouseEvent('pointermove', { clientX: 1000 }))
    expect(sidebar).toHaveStyle({ width: `${LYRICS_SIDEBAR_MAX_WIDTH}px` })
    expect(onWidthChange).toHaveBeenCalledTimes(callsAfterCancel)
    expect(onWidthChange).not.toHaveBeenCalledWith(expect.anything(), {
      persist: true,
    })

    fireEvent.pointerDown(resizer, { clientX: 500 })
    window.dispatchEvent(new MouseEvent('pointermove', { clientX: 1000 }))
    await waitFor(() =>
      expect(sidebar).toHaveStyle({ width: `${LYRICS_SIDEBAR_MIN_WIDTH}px` }),
    )

    unmount()
    const callsBeforePointerUp = onWidthChange.mock.calls.length
    window.dispatchEvent(new MouseEvent('pointerup'))

    expect(onWidthChange).toHaveBeenCalledTimes(callsBeforePointerUp)
    expect(onWidthChange).not.toHaveBeenCalledWith(expect.anything(), {
      persist: true,
    })
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
      labels: {
        title: 'Song words',
        resize: 'Resize song words',
        showTranslation: 'Reveal translation',
        hidePronunciation: 'Conceal pronunciation',
      },
    })

    const pronunciation = screen.getByTestId('toggle-pronunciation-button')
    const translation = screen.getByTestId('toggle-translation-button')
    expect(screen.getByLabelText('Song words')).toBeInTheDocument()
    expect(screen.getByLabelText('Resize song words')).toBeInTheDocument()
    expect(pronunciation).toHaveAttribute('aria-label', 'Conceal pronunciation')
    expect(pronunciation).toHaveAttribute('aria-pressed', 'true')
    expect(pronunciation.className).toContain('controlActive')
    fireEvent.click(pronunciation)
    expect(onTogglePronunciation).toHaveBeenCalledTimes(1)

    expect(translation).toHaveAttribute('aria-label', 'Reveal translation')
    expect(translation).toHaveAttribute('aria-pressed', 'false')
    expect(translation).toBeDisabled()
    expect(translation.className).not.toContain('controlActive')
    fireEvent.click(translation)
    expect(onToggleTranslation).not.toHaveBeenCalled()
  })

  it('stays mounted without re-entering when an open sidebar receives no lyrics', () => {
    const { rerender } = renderSidebar()

    const sidebar = screen.getByTestId('lyrics-sidebar')
    expect(sidebar).toHaveStyle({ transform: 'translateX(0)' })
    expect(screen.getByTestId('karaoke-lyrics-panel')).toBeInTheDocument()

    rerender(
      sidebarView({
        mainLyric: null,
        showTranslation: false,
        showPronunciation: false,
        translationEnabled: false,
        pronunciationEnabled: false,
      }),
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
})
