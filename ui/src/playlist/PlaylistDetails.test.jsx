import React from 'react'
import { render, screen, cleanup } from '@testing-library/react'
import PlaylistDetails from './PlaylistDetails'
import { usePermissions } from 'react-admin'
import { useMediaQuery } from '@material-ui/core'

vi.mock('react-admin', () => ({
  usePermissions: vi.fn(),
  useTranslate: () => (key, opts) => {
    if (key === 'resources.playlist.byOwner') {
      return `by ${opts.name}`
    }
    if (key === 'resources.song.name') {
      return opts.smart_count === 1 ? 'Song' : 'Songs'
    }
    return key
  },
  useRecordContext: (props) => props.record || {},
}))

vi.mock('@material-ui/core', async (importOriginal) => {
  const actual = await importOriginal()
  return { ...actual, useMediaQuery: vi.fn() }
})

describe('<PlaylistDetails />', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useMediaQuery.mockReturnValue(false)
  })

  afterEach(cleanup)

  const baseRecord = {
    id: 'pl1',
    name: 'My Playlist',
    songCount: 1,
    duration: 60,
    size: 1024,
    ownerName: 'Owner',
    public: false,
  }

  it('shows owner for admin users', () => {
    usePermissions.mockReturnValue({ permissions: 'admin' })
    render(<PlaylistDetails record={baseRecord} />)
    expect(screen.getByText('by Owner')).toBeInTheDocument()
  })

  it('shows owner for public playlists', () => {
    usePermissions.mockReturnValue({ permissions: 'user' })
    render(<PlaylistDetails record={{ ...baseRecord, public: true }} />)
    expect(screen.getByText('by Owner')).toBeInTheDocument()
  })

  it('hides owner for private playlists when not admin', () => {
    usePermissions.mockReturnValue({ permissions: 'user' })
    render(<PlaylistDetails record={baseRecord} />)
    expect(screen.queryByText('by Owner')).toBeNull()
  })
})
