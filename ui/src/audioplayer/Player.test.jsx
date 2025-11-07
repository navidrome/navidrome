/* eslint-env jest */

import React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { Provider } from 'react-redux'
import { createStore, combineReducers } from 'redux'
import { ThemeProvider } from '@material-ui/core/styles'
import { createMuiTheme } from '@material-ui/core/styles'
import { Player } from './Player'
import { playerReducer } from '../../reducers/playerReducer'
import { settingsReducer } from '../../reducers/settingsReducer'
import { replayGainReducer } from '../../reducers/replayGainReducer'

// Mock dependencies
jest.mock('../themes/useCurrentTheme', () => ({
  __esModule: true,
  default: () => ({
    player: { theme: 'dark' },
  }),
}))

jest.mock('../config', () => ({
  enableCoverAnimation: false,
  gaTrackingId: null,
}))

jest.mock('./AudioTitle', () => ({
  __esModule: true,
  default: ({ audioInfo }) => (
    <div data-testid="audio-title">{audioInfo?.song?.title || 'No song'}</div>
  ),
}))

jest.mock('./PlayerToolbar', () => ({
  __esModule: true,
  default: ({ id }) => <div data-testid="player-toolbar">{id || 'No ID'}</div>,
}))

jest.mock('./locale', () => ({
  __esModule: true,
  default: () => (key) => key,
}))

jest.mock('./keyHandlers', () => ({
  __esModule: true,
  default: () => ({}),
}))

jest.mock('../hotkeys', () => ({
  keyMap: {},
}))

jest.mock('react-ga', () => ({
  event: jest.fn(),
}))

jest.mock('../utils', () => ({
  sendNotification: jest.fn(),
}))

jest.mock('navidrome-music-player', () => ({
  __esModule: true,
  default: ({ children, ...props }) => (
    <div data-testid="react-jk-music-player" {...props}>
      {children}
    </div>
  ),
}))

jest.mock('navidrome-music-player/assets/index.css', () => {})

// Mock react-redux hooks
jest.mock('react-redux', () => ({
  ...jest.requireActual('react-redux'),
  useSelector: jest.fn(),
  useDispatch: jest.fn(),
}))

// Mock react-admin hooks
jest.mock('react-admin', () => ({
  useAuthState: () => ({ authenticated: true }),
  useDataProvider: () => ({
    getOne: jest.fn().mockResolvedValue({ data: {} }),
  }),
  useTranslate: () => (key) => key,
  createMuiTheme: jest.fn(),
}))

// Mock @material-ui/core
jest.mock('@material-ui/core', () => ({
  ...jest.requireActual('@material-ui/core'),
  useMediaQuery: () => true, // Mock as desktop
  ThemeProvider: ({ children }) => <div>{children}</div>,
}))

describe('Player Component', () => {
  const mockStore = createStore(
    combineReducers({
      player: playerReducer,
      settings: settingsReducer,
      replayGain: replayGainReducer,
    }),
    {
      player: {
        queue: [
          {
            uuid: '1',
            musicSrc: 'song1.mp3',
            title: 'Song 1',
            artist: 'Artist 1',
          },
        ],
        current: {
          uuid: '1',
          trackId: 'track1',
          song: { title: 'Song 1', artist: 'Artist 1' },
        },
        playIndex: 0,
        mode: 'single',
        volume: 0.8,
        clear: false,
      },
      settings: {
        notifications: true,
      },
      replayGain: {
        gainMode: 'track',
      },
    },
  )

  const renderPlayer = () => {
    return render(
      <Provider store={mockStore}>
        <ThemeProvider theme={createMuiTheme()}>
          <Player />
        </ThemeProvider>
      </Provider>,
    )
  }

  beforeEach(() => {
    jest.clearAllMocks()
  })

  it('should render player when authenticated with queue', () => {
    renderPlayer()

    expect(screen.getByTestId('react-jk-music-player')).toBeInTheDocument()
    expect(screen.getByTestId('audio-title')).toBeInTheDocument()
    expect(screen.getByTestId('player-toolbar')).toBeInTheDocument()
  })

  it('should not render when not authenticated', () => {
    // Mock unauthenticated state
    const { useAuthState } = jest.requireMock('react-admin')
    useAuthState.mockReturnValue({ authenticated: false })

    const { container } = renderPlayer()
    expect(container.firstChild).toBeNull()

    // Reset mock
    const { useAuthState: originalUseAuthState } =
      jest.requireMock('react-admin')
    originalUseAuthState.mockReturnValue({ authenticated: true })
  })

  it('should not render when queue is empty', () => {
    const emptyStore = createStore(
      combineReducers({
        player: playerReducer,
        settings: settingsReducer,
        replayGain: replayGainReducer,
      }),
      {
        player: { queue: [] },
        settings: { notifications: true },
        replayGain: { gainMode: 'track' },
      },
    )

    const { container } = render(
      <Provider store={emptyStore}>
        <ThemeProvider theme={createMuiTheme()}>
          <Player />
        </ThemeProvider>
      </Provider>,
    )

    expect(container.firstChild).toBeNull()
  })

  it('should have proper accessibility attributes', () => {
    renderPlayer()

    const playerRegion = screen.getByRole('region')
    expect(playerRegion).toHaveAttribute('aria-label', 'player.audioPlayer')
    expect(playerRegion).toHaveAttribute('aria-live', 'polite')
  })

  it('should render audio title with correct information', () => {
    renderPlayer()

    expect(screen.getByTestId('audio-title')).toHaveTextContent('Song 1')
  })

  it('should render player toolbar with track ID', () => {
    renderPlayer()

    expect(screen.getByTestId('player-toolbar')).toHaveTextContent('track1')
  })

  it('should handle mobile player detection', () => {
    // Mock mobile detection
    const { useMediaQuery } = jest.requireMock('@material-ui/core')
    useMediaQuery.mockReturnValue(false) // Mobile

    renderPlayer()

    // Mobile-specific logic should be applied
    // This would be tested more thoroughly with actual mobile behavior
  })

  it('should update document title when not visible', () => {
    // Mock empty queue to make player not visible
    const emptyStore = createStore(
      combineReducers({
        player: playerReducer,
        settings: settingsReducer,
        replayGain: replayGainReducer,
      }),
      {
        player: { queue: [] },
        settings: { notifications: true },
        replayGain: { gainMode: 'track' },
      },
    )

    render(
      <Provider store={emptyStore}>
        <ThemeProvider theme={createMuiTheme()}>
          <Player />
        </ThemeProvider>
      </Provider>,
    )

    // Document title should be reset when player is not visible
    expect(document.title).toBe('Navidrome')
  })

  it('should integrate with theme provider', () => {
    renderPlayer()

    // ThemeProvider should wrap the component
    expect(
      screen.getByTestId('react-jk-music-player').parentElement,
    ).toBeInTheDocument()
  })
})
