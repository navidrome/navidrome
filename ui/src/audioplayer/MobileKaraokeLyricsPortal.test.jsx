import React from 'react'
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import MobileKaraokeLyricsPortal, {
  MOBILE_KARAOKE_LYRICS_ACTIVE_CLASS,
  MOBILE_KARAOKE_LYRICS_ENTERED_CLASS,
  MOBILE_KARAOKE_LYRICS_HOST_SELECTOR,
  MOBILE_KARAOKE_LYRICS_LAYER_CLASS,
  MOBILE_KARAOKE_LYRICS_TRANSITION_MS,
} from './MobileKaraokeLyricsPortal'
import usePlayerLyrics from './usePlayerLyrics'

const { defaultLyricsResponse, useEnhancedLyricsMock } = vi.hoisted(() => {
  const defaultLyricsResponse = {
    layers: {
      main: {
        synced: true,
        line: [{ start: 0, end: 1000, value: 'Persistent panel lyrics' }],
      },
      translation: null,
      pronunciation: null,
    },
    loading: false,
    error: null,
    retry: vi.fn(),
  }
  return {
    defaultLyricsResponse,
    useEnhancedLyricsMock: vi.fn(() => defaultLyricsResponse),
  }
})

vi.mock('./useEnhancedLyrics', () => ({
  default: useEnhancedLyricsMock,
}))

const createHost = () => {
  const host = document.createElement('div')
  host.className = MOBILE_KARAOKE_LYRICS_HOST_SELECTOR.slice(1)
  document.body.appendChild(host)
  return host
}

const portal = (text, active = true) => (
  <MobileKaraokeLyricsPortal active={active}>
    <span>{text}</span>
  </MobileKaraokeLyricsPortal>
)

const MobileLyricsHarness = ({ trackId = 'track-1', isRadio = false }) => {
  const { toolbarLyricsProps, mobileLyricsSurface } = usePlayerLyrics({
    trackId,
    trackUpdatedAt: 'now',
    isRadio,
    audioInstance: null,
    isDesktop: false,
    translate: (key) => key,
  })
  return (
    <>
      <button
        type="button"
        onClick={toolbarLyricsProps.onToggleLyrics}
        disabled={toolbarLyricsProps.lyricsDisabled}
      >
        Toggle lyrics
      </button>
      {mobileLyricsSurface}
    </>
  )
}

describe('<MobileKaraokeLyricsPortal />', () => {
  afterEach(() => {
    vi.useRealTimers()
    useEnhancedLyricsMock.mockReset()
    useEnhancedLyricsMock.mockImplementation(() => defaultLyricsResponse)
    cleanup()
    document.body.innerHTML = ''
  })

  it('mounts active lyrics into the mobile cover host and cleans up the class', () => {
    vi.useFakeTimers()
    const host = createHost()
    const { rerender } = render(portal('Inline lyrics'))

    expect(within(host).getByText('Inline lyrics')).toBeInTheDocument()
    expect(host).toHaveClass(MOBILE_KARAOKE_LYRICS_ACTIVE_CLASS)
    expect(host).toHaveClass(MOBILE_KARAOKE_LYRICS_ENTERED_CLASS)
    expect(
      host.querySelector(`.${MOBILE_KARAOKE_LYRICS_LAYER_CLASS}`),
    ).toHaveAttribute('data-entered', 'true')
    expect(
      host.querySelector(`.${MOBILE_KARAOKE_LYRICS_LAYER_CLASS}`),
    ).toHaveStyle({ pointerEvents: 'auto' })

    rerender(portal('Inline lyrics', false))

    expect(within(host).getByText('Inline lyrics')).toBeInTheDocument()
    expect(host).toHaveClass(MOBILE_KARAOKE_LYRICS_ACTIVE_CLASS)
    expect(host).not.toHaveClass(MOBILE_KARAOKE_LYRICS_ENTERED_CLASS)
    expect(
      host.querySelector(`.${MOBILE_KARAOKE_LYRICS_LAYER_CLASS}`),
    ).toHaveAttribute('data-entered', 'false')
    expect(
      host.querySelector(`.${MOBILE_KARAOKE_LYRICS_LAYER_CLASS}`),
    ).toHaveStyle({ pointerEvents: 'none' })

    act(() => {
      vi.advanceTimersByTime(MOBILE_KARAOKE_LYRICS_TRANSITION_MS)
    })

    expect(screen.queryByText('Inline lyrics')).not.toBeInTheDocument()
    expect(host).not.toHaveClass(MOBILE_KARAOKE_LYRICS_ACTIVE_CLASS)
    expect(host).not.toHaveClass(MOBILE_KARAOKE_LYRICS_ENTERED_CLASS)
  })

  it('attaches when the mobile cover host appears after activation', async () => {
    vi.useFakeTimers()

    render(portal('Late lyrics'))

    expect(screen.queryByText('Late lyrics')).not.toBeInTheDocument()

    const host = createHost()

    await waitFor(() =>
      expect(within(host).getByText('Late lyrics')).toBeInTheDocument(),
    )
    expect(host).toHaveClass(MOBILE_KARAOKE_LYRICS_ACTIVE_CLASS)
  })

  it('moves active lyrics when the mobile cover host is replaced', async () => {
    const firstHost = createHost()

    render(portal('Persistent lyrics'))
    expect(within(firstHost).getByText('Persistent lyrics')).toBeInTheDocument()

    const secondHost = createHost()
    firstHost.replaceWith(secondHost)

    await waitFor(() =>
      expect(
        within(secondHost).getByText('Persistent lyrics'),
      ).toBeInTheDocument(),
    )
    expect(firstHost).not.toHaveClass(MOBILE_KARAOKE_LYRICS_ACTIVE_CLASS)
    expect(secondHost).toHaveClass(MOBILE_KARAOKE_LYRICS_ACTIVE_CLASS)
  })

  it('keeps panel contents rendered for the complete mobile exit', async () => {
    vi.useFakeTimers()
    const host = createHost()
    const { rerender } = render(<MobileLyricsHarness />)

    fireEvent.click(screen.getByRole('button', { name: 'Toggle lyrics' }))
    expect(
      within(host).getByText('Persistent panel lyrics'),
    ).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Toggle lyrics' }))

    expect(useEnhancedLyricsMock).toHaveBeenLastCalledWith(
      expect.objectContaining({ requested: false }),
    )
    rerender(<MobileLyricsHarness trackId="track-2" />)
    const trackTwoRequests = useEnhancedLyricsMock.mock.calls
      .filter(([options]) => options.trackId === 'track-2')
      .map(([options]) => options.requested)
    expect(trackTwoRequests.length).toBeGreaterThan(0)
    expect(trackTwoRequests).not.toContain(true)
    expect(
      within(host).getByText('Persistent panel lyrics'),
    ).toBeInTheDocument()
    expect(
      host.querySelector(`.${MOBILE_KARAOKE_LYRICS_LAYER_CLASS}`),
    ).toHaveStyle({ pointerEvents: 'none' })

    act(() => {
      vi.advanceTimersByTime(MOBILE_KARAOKE_LYRICS_TRANSITION_MS)
    })

    expect(
      screen.queryByText('Persistent panel lyrics'),
    ).not.toBeInTheDocument()
  })

  it('centers working pronunciation and translation toggles below mobile lyrics', () => {
    useEnhancedLyricsMock.mockImplementation(() => ({
      ...defaultLyricsResponse,
      layers: {
        ...defaultLyricsResponse.layers,
        pronunciation: {
          synced: true,
          line: [{ start: 0, end: 1000, value: 'Spoken lyrics' }],
        },
        translation: {
          synced: true,
          line: [{ start: 0, end: 1000, value: 'Translated lyrics' }],
        },
      },
    }))
    const host = createHost()
    render(<MobileLyricsHarness />)

    fireEvent.click(screen.getByRole('button', { name: 'Toggle lyrics' }))

    const controls = within(host).getByTestId('lyrics-mobile-layer-controls')
    expect(controls).toHaveAttribute('data-placement', 'mobile')
    expect(controls).toHaveStyle({
      bottom: '8px',
      left: '50%',
      transform: 'translateX(-50%)',
    })

    const pronunciation = within(host).getByTestId(
      'toggle-pronunciation-button',
    )
    const translation = within(host).getByTestId('toggle-translation-button')
    expect(pronunciation).toHaveAttribute(
      'aria-label',
      'player.hideLyricsPronunciationText',
    )
    expect(pronunciation).toHaveAttribute('aria-pressed', 'true')
    expect(translation).toHaveAttribute(
      'aria-label',
      'player.hideLyricsTranslationText',
    )
    expect(translation).toHaveAttribute('aria-pressed', 'true')

    fireEvent.click(pronunciation)
    fireEvent.click(translation)

    expect(pronunciation).toHaveAttribute('aria-pressed', 'false')
    expect(translation).toHaveAttribute('aria-pressed', 'false')
    expect(within(host).queryByText('Spoken lyrics')).not.toBeInTheDocument()
    expect(
      within(host).queryByText('Translated lyrics'),
    ).not.toBeInTheDocument()
  })

  it('keeps open lyrics closable after playback switches to radio', () => {
    vi.useFakeTimers()
    const host = createHost()
    const { rerender } = render(<MobileLyricsHarness />)
    const toggle = screen.getByRole('button', { name: 'Toggle lyrics' })

    fireEvent.click(toggle)
    expect(
      within(host).getByText('Persistent panel lyrics'),
    ).toBeInTheDocument()

    rerender(<MobileLyricsHarness isRadio />)
    expect(toggle).toBeEnabled()
    fireEvent.click(toggle)

    act(() => {
      vi.advanceTimersByTime(MOBILE_KARAOKE_LYRICS_TRANSITION_MS)
    })
    expect(
      screen.queryByText('Persistent panel lyrics'),
    ).not.toBeInTheDocument()
    expect(toggle).toBeDisabled()
  })
})
