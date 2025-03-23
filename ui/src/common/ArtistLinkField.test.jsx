import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, test, expect, beforeEach, vi } from 'vitest'
import { ArtistLinkField } from './ArtistLinkField'
import { intersperse } from '../utils/index.js'

// Mock dependencies
vi.mock('react-redux', () => ({
  useDispatch: vi.fn(() => vi.fn()),
}))

vi.mock('./useGetHandleArtistClick', () => ({
  useGetHandleArtistClick: vi.fn(() => (id) => `/artist/${id}`),
}))

vi.mock('../utils/index.js', () => ({
  intersperse: vi.fn((arr) => arr),
}))

vi.mock('@material-ui/core', () => ({
  withWidth: () => (Component) => (props) => (
    <Component {...props} width="md" />
  ),
}))

vi.mock('react-admin', () => ({
  Link: ({ children, to, ...props }) => (
    <a href={to} {...props}>
      {children}
    </a>
  ),
}))

describe('ArtistLinkField', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  test('renders artists from participants when available', () => {
    const record = {
      participants: {
        artist: [
          { id: '1', name: 'Artist 1' },
          { id: '2', name: 'Artist 2' },
        ],
      },
    }

    render(<ArtistLinkField record={record} source="artist" />)

    expect(screen.getByText('Artist 1')).toBeInTheDocument()
    expect(screen.getByText('Artist 2')).toBeInTheDocument()
  })

  test('falls back to record[source] when participants not available', () => {
    const record = {
      artist: 'Fallback Artist',
      artistId: '123',
    }

    render(<ArtistLinkField record={record} source="artist" />)

    expect(screen.getByText('Fallback Artist')).toBeInTheDocument()
  })

  test('adds remixers when showing artist role', () => {
    const record = {
      participants: {
        artist: [{ id: '1', name: 'Artist 1' }],
        remixer: [{ id: '2', name: 'Remixer 1' }],
      },
    }

    render(<ArtistLinkField record={record} source="artist" />)

    expect(screen.getByText('Artist 1')).toBeInTheDocument()
    expect(screen.getByText('Remixer 1')).toBeInTheDocument()
  })

  test('uses parseAndReplaceArtists when role is albumartist', () => {
    const record = {
      albumArtist: 'Group Artist',
      participants: {
        albumartist: [{ id: '1', name: 'Group Artist' }],
      },
    }

    render(<ArtistLinkField record={record} source="albumArtist" />)

    expect(screen.getByText('Group Artist')).toBeInTheDocument()
    expect(screen.getByRole('link')).toHaveAttribute('href', '/artist/1')
  })

  test('deduplicates artists with the same id', () => {
    const record = {
      participants: {
        artist: [
          { id: '1', name: 'Duplicate Artist' },
          { id: '1', name: 'Duplicate Artist', subRole: 'Vocals' },
        ],
      },
    }

    render(<ArtistLinkField record={record} source="artist" />)

    const links = screen.getAllByRole('link')
    expect(links).toHaveLength(1)
    expect(links[0]).toHaveTextContent('Duplicate Artist (Vocals)')
  })

  test('limits the number of artists displayed', () => {
    const record = {
      participants: {
        artist: [
          { id: '1', name: 'Artist 1' },
          { id: '2', name: 'Artist 2' },
          { id: '3', name: 'Artist 3' },
          { id: '4', name: 'Artist 4' },
        ],
      },
    }

    render(<ArtistLinkField record={record} source="artist" limit={3} />)

    expect(screen.getByText('Artist 1')).toBeInTheDocument()
    expect(screen.getByText('Artist 2')).toBeInTheDocument()
    expect(screen.getByText('Artist 3')).toBeInTheDocument()
    expect(screen.queryByText('Artist 4')).not.toBeInTheDocument()
    expect(screen.getByText('...')).toBeInTheDocument()
  })

  test('aggregates subroles for the same artist', () => {
    const record = {
      participants: {
        artist: [
          { id: '1', name: 'Multi-Role Artist', subRole: 'Vocals' },
          { id: '1', name: 'Multi-Role Artist', subRole: 'Guitar' },
        ],
      },
    }

    render(<ArtistLinkField record={record} source="artist" />)

    expect(
      screen.getByText('Multi-Role Artist (Vocals, Guitar)'),
    ).toBeInTheDocument()
  })

  test('handles empty artists array', () => {
    const record = {
      participants: {
        artist: [],
      },
    }

    render(<ArtistLinkField record={record} source="artist" />)

    expect(intersperse).toHaveBeenCalledWith([], ' • ')
  })
})
