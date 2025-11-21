import React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { describe, it, beforeEach, vi } from 'vitest'
import { Provider } from 'react-redux'
import { createStore, combineReducers } from 'redux'
import { activityReducer } from '../reducers'
import NowPlayingPanel from './NowPlayingPanel'
import subsonic from '../subsonic'

vi.mock('../subsonic', () => ({
  default: {
    getNowPlaying: vi.fn(),
    getAvatarUrl: vi.fn(() => '/avatar'),
    getCoverArtUrl: vi.fn(() => '/cover'),
  },
}))

// Create a mock for useMediaQuery
const mockUseMediaQuery = vi.fn()

vi.mock('react-admin', async (importOriginal) => {
  const actual = await importOriginal()
  const redux = await import('react-redux')
  return {
    ...actual,
    useTranslate: () => (x) => x,
    useSelector: redux.useSelector,
    useDispatch: redux.useDispatch,
    Link: ({ to, children, onClick, ...props }) => (
      <a
        href={to}
        onClick={(e) => {
          e.preventDefault() // Prevent navigation in tests
          if (onClick) onClick(e)
        }}
        {...props}
      >
        {children}
      </a>
    ),
  }
})

// Mock the specific Material-UI hooks we need
vi.mock('@material-ui/core/useMediaQuery', () => ({
  default: () => mockUseMediaQuery(),
}))

vi.mock('@material-ui/core/styles/useTheme', () => ({
  default: () => ({
    breakpoints: {
      down: () => '(max-width:959.95px)', // Mock breakpoint string
    },
  }),
}))

describe('<NowPlayingPanel />', () => {
  const createMockStore = (overrides = {}) => {
    const defaultState = {
      activity: {
        nowPlayingCount: 1,
        serverStart: { startTime: Date.now() }, // Server is up by default
        streamReconnected: 0,
        ...overrides,
      },
    }
    return createStore(
      combineReducers({ activity: activityReducer }),
      defaultState,
    )
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mockUseMediaQuery.mockReturnValue(false) // Default to large screen

    subsonic.getNowPlaying.mockResolvedValue({
      json: {
        'subsonic-response': {
          status: 'ok',
          nowPlaying: {
            entry: [
              {
                playerId: 1,
                username: 'u1',
                playerName: 'Chrome Browser',
                title: 'Song',
                albumArtist: 'Artist',
                albumId: 'album1',
                albumArtistId: 'artist1',
                minutesAgo: 2,
              },
            ],
          },
        },
      },
    })
  })

  it('fetches and displays entries when opened', async () => {
    const store = createMockStore()
    render(
      <Provider store={store}>
        <NowPlayingPanel />
      </Provider>,
    )

    // Wait for initial fetch to complete
    await waitFor(() => {
      expect(subsonic.getNowPlaying).toHaveBeenCalled()
    })

    fireEvent.click(screen.getByRole('button'))
    await waitFor(() => {
      expect(screen.getByText('Artist')).toBeInTheDocument()
      expect(screen.getByRole('link', { name: 'Artist' })).toHaveAttribute(
        'href',
        '/artist/artist1/show',
      )
    })
  })

  it('displays player name after username', async () => {
    const store = createMockStore()
    render(
      <Provider store={store}>
        <NowPlayingPanel />
      </Provider>,
    )

    // Wait for initial fetch to complete
    await waitFor(() => {
      expect(subsonic.getNowPlaying).toHaveBeenCalled()
    })

    fireEvent.click(screen.getByRole('button'))
    await waitFor(() => {
      expect(
        screen.getByText('u1 (Chrome Browser) • nowPlaying.minutesAgo'),
      ).toBeInTheDocument()
    })
  })

  it('handles entries without player name', async () => {
    subsonic.getNowPlaying.mockResolvedValueOnce({
      json: {
        'subsonic-response': {
          status: 'ok',
          nowPlaying: {
            entry: [
              {
                playerId: 1,
                username: 'u1',
                title: 'Song',
                albumArtist: 'Artist',
                albumId: 'album1',
                albumArtistId: 'artist1',
                minutesAgo: 2,
              },
            ],
          },
        },
      },
    })

    const store = createMockStore()
    render(
      <Provider store={store}>
        <NowPlayingPanel />
      </Provider>,
    )

    // Wait for initial fetch to complete
    await waitFor(() => {
      expect(subsonic.getNowPlaying).toHaveBeenCalled()
    })

    fireEvent.click(screen.getByRole('button'))
    await waitFor(() => {
      expect(screen.getByText('u1 • nowPlaying.minutesAgo')).toBeInTheDocument()
    })
  })

  it('shows empty message when no entries', async () => {
    subsonic.getNowPlaying.mockResolvedValueOnce({
      json: {
        'subsonic-response': { status: 'ok', nowPlaying: { entry: [] } },
      },
    })
    const store = createMockStore({ nowPlayingCount: 0 })
    render(
      <Provider store={store}>
        <NowPlayingPanel />
      </Provider>,
    )

    // Wait for initial fetch
    await waitFor(() => {
      expect(subsonic.getNowPlaying).toHaveBeenCalled()
    })

    fireEvent.click(screen.getByRole('button'))
    await waitFor(() => {
      expect(screen.getByText('nowPlaying.empty')).toBeInTheDocument()
    })
  })

  it('does not close panel when artist link is clicked on large screens', async () => {
    mockUseMediaQuery.mockReturnValue(false) // Simulate large screen

    const store = createMockStore()
    render(
      <Provider store={store}>
        <NowPlayingPanel />
      </Provider>,
    )

    // Wait for initial fetch to complete
    await waitFor(() => {
      expect(subsonic.getNowPlaying).toHaveBeenCalled()
    })

    // Open the panel
    fireEvent.click(screen.getByRole('button'))
    await waitFor(() => {
      expect(screen.getByText('Artist')).toBeInTheDocument()
    })

    // Check that the popover is open
    expect(screen.getByRole('presentation')).toBeInTheDocument()

    // Click the artist link
    fireEvent.click(screen.getByRole('link', { name: 'Artist' }))

    // Panel should remain open (popover should still be in document)
    expect(screen.getByRole('presentation')).toBeInTheDocument()
    expect(screen.getByText('Artist')).toBeInTheDocument()
  })

  it('does not fetch on mount when server is down', () => {
    const store = createMockStore({
      nowPlayingCount: 1,
      serverStart: { startTime: null }, // Server is down
    })
    render(
      <Provider store={store}>
        <NowPlayingPanel />
      </Provider>,
    )

    // Should not have made initial fetch request due to server being down
    expect(subsonic.getNowPlaying).not.toHaveBeenCalled()
  })

  it('does not fetch on stream reconnection when server is down', () => {
    const store = createMockStore({
      nowPlayingCount: 1,
      serverStart: { startTime: null }, // Server is down
      streamReconnected: Date.now(), // Stream reconnected
    })
    render(
      <Provider store={store}>
        <NowPlayingPanel />
      </Provider>,
    )

    // Should not have made fetch request due to server being down
    expect(subsonic.getNowPlaying).not.toHaveBeenCalled()
  })

  it('does not double-fetch on server reconnection', () => {
    const initialStore = createMockStore({
      nowPlayingCount: 1,
      serverStart: { startTime: null }, // Server initially down
      streamReconnected: 0,
    })
    const { rerender } = render(
      <Provider store={initialStore}>
        <NowPlayingPanel />
      </Provider>,
    )

    // Clear initial (empty) calls
    vi.clearAllMocks()

    // Simulate server coming back up with stream reconnection (both state changes happen)
    const reconnectedStore = createMockStore({
      nowPlayingCount: 1,
      serverStart: { startTime: Date.now() }, // Server back up
      streamReconnected: Date.now(), // Stream reconnected
    })
    rerender(
      <Provider store={reconnectedStore}>
        <NowPlayingPanel />
      </Provider>,
    )

    // Should only make one call despite both serverUp and streamReconnected changing
    expect(subsonic.getNowPlaying).toHaveBeenCalledTimes(1)
  })

  it('skips polling when server is down', () => {
    vi.useFakeTimers()

    const store = createMockStore({
      nowPlayingCount: 1,
      serverStart: { startTime: null }, // Server is down
    })
    render(
      <Provider store={store}>
        <NowPlayingPanel />
      </Provider>,
    )

    // Clear initial mount fetch
    vi.clearAllMocks()

    // Advance time by 70 seconds to trigger polling interval
    vi.advanceTimersByTime(70000)

    // Should not have made any additional requests due to server being down
    expect(subsonic.getNowPlaying).not.toHaveBeenCalled()

    vi.useRealTimers()
  })

  it('resumes polling when server comes back up', () => {
    vi.useFakeTimers()

    const store = createMockStore({
      nowPlayingCount: 1,
      serverStart: { startTime: null }, // Server is down
    })
    const { rerender } = render(
      <Provider store={store}>
        <NowPlayingPanel />
      </Provider>,
    )

    // Clear initial mount fetch
    vi.clearAllMocks()

    // Advance time - should not poll when server is down
    vi.advanceTimersByTime(70000)
    expect(subsonic.getNowPlaying).not.toHaveBeenCalled()

    // Update state to indicate server is back up
    const updatedStore = createMockStore({
      nowPlayingCount: 1,
      serverStart: { startTime: Date.now() }, // Server is back up
    })
    rerender(
      <Provider store={updatedStore}>
        <NowPlayingPanel />
      </Provider>,
    )

    // Clear the fetch that happens due to initial mount of rerender
    vi.clearAllMocks()

    // Advance time again - should now poll since server is up
    vi.advanceTimersByTime(70000)
    expect(subsonic.getNowPlaying).toHaveBeenCalled()

    vi.useRealTimers()
  })
})
