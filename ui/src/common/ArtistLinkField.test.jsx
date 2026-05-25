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

  describe('creditedAs', () => {
    it('renders creditedAs as the link text when present', () => {
      const record = {
        artist: 'PAS',
        participants: {
          artist: [
            {
              id: 'canon-1',
              name: 'Planetary Assault Systems',
              creditedAs: 'PAS',
            },
          ],
        },
      }

      render(<ArtistLinkField record={record} source="artist" />)

      expect(screen.getByText('PAS')).toBeInTheDocument()
      expect(
        screen.queryByText('Planetary Assault Systems'),
      ).not.toBeInTheDocument()
    })

    it('sets a title tooltip with the canonical name when creditedAs differs', () => {
      const record = {
        artist: 'PAS',
        participants: {
          artist: [
            {
              id: 'canon-1',
              name: 'Planetary Assault Systems',
              creditedAs: 'PAS',
            },
          ],
        },
      }

      render(<ArtistLinkField record={record} source="artist" />)

      const link = screen.getByRole('link')
      expect(link).toHaveAttribute('title', 'Planetary Assault Systems')
    })

    it('falls back to name when creditedAs is missing', () => {
      const record = {
        artist: 'Some Artist',
        participants: {
          artist: [{ id: 'canon-2', name: 'Some Artist' }],
        },
      }

      render(<ArtistLinkField record={record} source="artist" />)

      expect(screen.getByText('Some Artist')).toBeInTheDocument()
      const link = screen.getByRole('link')
      expect(link).not.toHaveAttribute('title')
    })

    it('does not set a tooltip when creditedAs equals name', () => {
      const record = {
        artist: 'Same Name',
        participants: {
          artist: [
            { id: 'canon-3', name: 'Same Name', creditedAs: 'Same Name' },
          ],
        },
      }

      render(<ArtistLinkField record={record} source="artist" />)

      const link = screen.getByRole('link')
      expect(link).not.toHaveAttribute('title')
    })

    it('inline-linkifies when displayArtist holds the canonical name (Picard-default tagging)', () => {
      // Picard "use standardized artist names" mode: ARTIST tag carries the
      // canonical name, ARTIST_CREDIT carries the credit. The display string
      // is the canonical name; parseAndReplaceArtists must match on `name`
      // (not `creditedAs`) to embed the link inline. The link text itself
      // still renders the credit via ALink.
      const record = {
        artist: 'Planetary Assault Systems',
        participants: {
          artist: [
            {
              id: 'canon-1',
              name: 'Planetary Assault Systems',
              creditedAs: 'PAS',
            },
          ],
        },
      }

      render(<ArtistLinkField record={record} source="artist" />)

      const link = screen.getByRole('link')
      // Link text is the credit, tooltip carries canonical
      expect(link).toHaveTextContent('PAS')
      expect(link).toHaveAttribute('title', 'Planetary Assault Systems')
      // The original canonical string should not appear as raw plain text
      // (it was replaced by the link)
      expect(
        screen.queryByText(/^Planetary Assault Systems$/),
      ).not.toBeInTheDocument()
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
