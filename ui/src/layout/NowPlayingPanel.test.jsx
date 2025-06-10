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
    const store = createStore(combineReducers({ activity: activityReducer }), {
      activity: { nowPlayingCount: 1 },
    })
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
    const store = createStore(combineReducers({ activity: activityReducer }), {
      activity: { nowPlayingCount: 1 },
    })
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

    const store = createStore(combineReducers({ activity: activityReducer }), {
      activity: { nowPlayingCount: 1 },
    })
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
    const store = createStore(combineReducers({ activity: activityReducer }), {
      activity: { nowPlayingCount: 0 },
    })
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

    const store = createStore(combineReducers({ activity: activityReducer }), {
      activity: { nowPlayingCount: 1 },
    })
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
})
