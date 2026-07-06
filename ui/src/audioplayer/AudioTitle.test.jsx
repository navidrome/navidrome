import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { Provider } from 'react-redux'
import { createStore } from 'redux'
import AudioTitle from './AudioTitle'

vi.mock('@material-ui/core', async () => {
  const actual = await import('@material-ui/core')
  return {
    ...actual,
    useMediaQuery: vi.fn(),
  }
})

vi.mock('react-router-dom', () => ({
  // eslint-disable-next-line react/display-name
  Link: React.forwardRef(({ to, children, ...props }, ref) => (
    <a href={to} ref={ref} {...props}>
      {children}
    </a>
  )),
}))

vi.mock('react-dnd', () => ({
  useDrag: vi.fn(() => [null, () => {}]),
}))

const renderWithStore = (ui, playerState = {}) => {
  const store = createStore(() => ({ player: playerState }))
  return render(<Provider store={store}>{ui}</Provider>)
}

describe('<AudioTitle />', () => {
  const baseSong = {
    id: 'song-1',
    albumId: 'album-1',
    playlistId: 'playlist-1',
    title: 'Test Song',
    artist: 'Artist',
    album: 'Album',
    year: '2020',
  }

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('links to playlist when playlistId is provided', () => {
    const audioInfo = { trackId: 'track-1', song: baseSong }
    renderWithStore(
      <AudioTitle audioInfo={audioInfo} gainInfo={{}} isMobile={false} />,
    )
    const link = screen.getByRole('link')
    expect(link.getAttribute('href')).toBe('/playlist/playlist-1/show')
  })

  it('falls back to album link when no playlistId', () => {
    const audioInfo = {
      trackId: 'track-1',
      song: { ...baseSong, playlistId: undefined },
    }
    renderWithStore(
      <AudioTitle audioInfo={audioInfo} gainInfo={{}} isMobile={false} />,
    )
    const link = screen.getByRole('link')
    expect(link.getAttribute('href')).toBe('/album/album-1/show')
  })

  it('shows live radio title without changing the station link', () => {
    const audioInfo = {
      trackId: 'rd-1',
      isRadio: true,
      radioTitle: 'Artist - Title',
      song: {
        id: 'rd-1',
        title: 'Test Station',
        artist: 'Test Station',
        album: 'https://stream.example.test/radio',
      },
    }
    renderWithStore(
      <AudioTitle audioInfo={audioInfo} gainInfo={{}} isMobile={false} />,
    )

    expect(screen.getByText('Artist - Title')).toBeInTheDocument()
    expect(screen.getByRole('link').getAttribute('href')).toBe(
      '/radio/rd-1/show',
    )
  })

  it('prefers the live redux title over the stale player-library snapshot', () => {
    const audioInfo = {
      trackId: 'radio-1',
      isRadio: true,
      // Note: no radioTitle here — this simulates react-jinke-music-player's
      // stale internal snapshot, which never receives redux updates.
      song: {
        id: 'radio-1',
        title: 'Station Name',
        artist: 'Station Name',
        album: 'https://stream.example.test/radio',
      },
    }
    renderWithStore(
      <AudioTitle audioInfo={audioInfo} gainInfo={{}} isMobile={false} />,
      {
        current: {
          isRadio: true,
          trackId: 'radio-1',
          radioTitle: 'Live Song - Artist',
        },
      },
    )

    expect(screen.getByText('Live Song - Artist')).toBeInTheDocument()
    expect(screen.getByRole('link').getAttribute('href')).toBe(
      '/radio/radio-1/show',
    )
  })

  it('falls back to the station name when redux current is a different radio', () => {
    const audioInfo = {
      trackId: 'radio-1',
      isRadio: true,
      song: {
        id: 'radio-1',
        title: 'Station Name',
        // Artist deliberately differs from title so getByText stays unambiguous
        artist: 'Station Artist',
        album: 'https://stream.example.test/radio',
      },
    }
    renderWithStore(
      <AudioTitle audioInfo={audioInfo} gainInfo={{}} isMobile={false} />,
      {
        current: {
          isRadio: true,
          trackId: 'radio-2',
          radioTitle: 'Some Other Live Title',
        },
      },
    )

    expect(screen.getByText('Station Name')).toBeInTheDocument()
    expect(
      screen.queryByText('Some Other Live Title'),
    ).not.toBeInTheDocument()
  })
})
