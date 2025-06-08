import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect, beforeEach, vi } from 'vitest'
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
    render(<AudioTitle audioInfo={audioInfo} gainInfo={{}} isMobile={false} />)
    const link = screen.getByRole('link')
    expect(link.getAttribute('href')).toBe('/playlist/playlist-1/show')
  })

  it('falls back to album link when no playlistId', () => {
    const audioInfo = {
      trackId: 'track-1',
      song: { ...baseSong, playlistId: undefined },
    }
    render(<AudioTitle audioInfo={audioInfo} gainInfo={{}} isMobile={false} />)
    const link = screen.getByRole('link')
    expect(link.getAttribute('href')).toBe('/album/album-1/show')
  })
})
