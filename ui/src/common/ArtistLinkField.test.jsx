import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect, beforeEach, vi } from 'vitest'
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
  withWidth: () => (Component) => {
    const WithWidthComponent = (props) => <Component {...props} width="md" />
    WithWidthComponent.displayName = `WithWidth(${Component.displayName || Component.name || 'Component'})`
    return WithWidthComponent
  },
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

  describe('when rendering artists', () => {
    it('renders artists from participants when available', () => {
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

    it('falls back to record[source] when participants not available', () => {
      const record = {
        artist: 'Fallback Artist',
        artistId: '123',
      }

      render(<ArtistLinkField record={record} source="artist" />)

      expect(screen.getByText('Fallback Artist')).toBeInTheDocument()
    })

    it('handles empty artists array', () => {
      const record = {
        participants: {
          artist: [],
        },
      }

      render(<ArtistLinkField record={record} source="artist" />)

      expect(intersperse).toHaveBeenCalledWith([], ' • ')
    })
  })

  describe('when handling remixers', () => {
    it('adds remixers when showing artist role', () => {
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

    it('limits remixers to maximum of 2', () => {
      const record = {
        participants: {
          artist: [{ id: '1', name: 'Artist 1' }],
          remixer: [
            { id: '2', name: 'Remixer 1' },
            { id: '3', name: 'Remixer 2' },
            { id: '4', name: 'Remixer 3' },
          ],
        },
      }

      render(<ArtistLinkField record={record} source="artist" />)

      expect(screen.getByText('Artist 1')).toBeInTheDocument()
      expect(screen.getByText('Remixer 1')).toBeInTheDocument()
      expect(screen.getByText('Remixer 2')).toBeInTheDocument()
      expect(screen.queryByText('Remixer 3')).not.toBeInTheDocument()
    })

    it('deduplicates artists and remixers', () => {
      const record = {
        participants: {
          artist: [{ id: '1', name: 'Duplicate Person' }],
          remixer: [{ id: '1', name: 'Duplicate Person' }],
        },
      }

      render(<ArtistLinkField record={record} source="artist" />)

      const links = screen.getAllByRole('link')
      expect(links).toHaveLength(1)
      expect(links[0]).toHaveTextContent('Duplicate Person')
    })
  })

  describe('when using parseAndReplaceArtists', () => {
    it('uses parseAndReplaceArtists when role is albumartist', () => {
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

    it('uses parseAndReplaceArtists when role is artist', () => {
      const record = {
        artist: 'Main Artist',
        participants: {
          artist: [{ id: '1', name: 'Main Artist' }],
        },
      }

      render(<ArtistLinkField record={record} source="artist" />)

      expect(screen.getByText('Main Artist')).toBeInTheDocument()
      expect(screen.getByRole('link')).toHaveAttribute('href', '/artist/1')
    })

    it('adds remixers after parseAndReplaceArtists for artist role', () => {
      const record = {
        artist: 'Main Artist',
        participants: {
          artist: [{ id: '1', name: 'Main Artist' }],
          remixer: [{ id: '2', name: 'Remixer 1' }],
        },
      }

      render(<ArtistLinkField record={record} source="artist" />)

      const links = screen.getAllByRole('link')
      expect(links).toHaveLength(2)
      expect(links[0]).toHaveAttribute('href', '/artist/1')
      expect(links[1]).toHaveAttribute('href', '/artist/2')
    })
  })

  describe('when handling artist deduplication', () => {
    it('deduplicates artists with the same id', () => {
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

    it('aggregates subroles for the same artist', () => {
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
  })

  describe('when limiting displayed artists', () => {
    it('limits the number of artists displayed', () => {
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
  })
})
