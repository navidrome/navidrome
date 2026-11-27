import React from 'react'
import { render, act, cleanup } from '@testing-library/react'
import { useMediaQuery } from '@material-ui/core'
import { useAuthState } from 'react-admin'
import { useSelector } from 'react-redux'
import { Player } from './Player'

// Captures the props the music player is rendered with, and exposes the
// onModeChange callback so a test can simulate the user collapsing the player.
const playerProps = { current: null }

vi.mock('navidrome-music-player', () => ({
  default: (props) => {
    playerProps.current = props
    return <div data-testid="music-player" />
  },
}))
vi.mock('navidrome-music-player/assets/index.css', () => ({}))

vi.mock('@material-ui/core', async () => {
  const actual = await import('@material-ui/core')
  return { ...actual, useMediaQuery: vi.fn() }
})

vi.mock('react-admin', async () => {
  const actual = await import('react-admin')
  return {
    ...actual,
    useAuthState: vi.fn(),
    useDataProvider: () => ({ getOne: vi.fn().mockResolvedValue({}) }),
    useTranslate: () => (key) => key,
  }
})

vi.mock('react-redux', () => ({
  useDispatch: () => vi.fn(),
  useSelector: vi.fn(),
}))

vi.mock('../transcode', () => ({
  detectBrowserProfile: () => ({}),
  decisionService: {
    setProfile: vi.fn(),
    resolveStreamUrl: vi.fn().mockResolvedValue(''),
    prefetchDecisions: vi.fn(),
    invalidateAll: vi.fn(),
  },
}))

vi.mock('./PlayerToolbar', () => ({ default: () => <div /> }))
vi.mock('../subsonic', () => ({ default: { reportPlayback: vi.fn() } }))

const state = {
  player: {
    queue: [{ trackId: 't1', uuid: 'u1', isRadio: false }],
    playIndex: 0,
    volume: 1,
    mode: 'order',
    current: { trackId: 't1' },
    clear: false,
    savedPlayIndex: 0,
  },
  settings: { notifications: false },
  replayGain: { gainMode: 'none' },
  theme: 'dark',
}

describe('<Player />', () => {
  beforeEach(() => {
    playerProps.current = null
    useAuthState.mockReturnValue({ authenticated: true })
    useSelector.mockImplementation((selector) => selector(state))
  })

  afterEach(cleanup)

  // Regression test for #5917: a player collapsed while the window was narrow
  // could never be reopened once the window grew past the desktop breakpoint,
  // because toggleMode is disabled on desktop.
  it('returns to full mode when the viewport grows into desktop', () => {
    useMediaQuery.mockReturnValue(false)
    const { rerender } = render(<Player />)
    expect(playerProps.current.mode).toBe('full')

    // The user collapses the player into the mini circle.
    act(() => playerProps.current.onModeChange('mini'))
    expect(playerProps.current.mode).toBe('mini')

    // The window is widened past the desktop breakpoint.
    useMediaQuery.mockReturnValue(true)
    act(() => {
      rerender(<Player />)
    })

    expect(playerProps.current.mode).toBe('full')
  })

  it('keeps the collapsed mode while the viewport stays narrow', () => {
    useMediaQuery.mockReturnValue(false)
    const { rerender } = render(<Player />)

    act(() => playerProps.current.onModeChange('mini'))
    act(() => {
      rerender(<Player />)
    })

    expect(playerProps.current.mode).toBe('mini')
  })
})
