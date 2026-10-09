import React from 'react'
import { render, fireEvent, waitFor, screen } from '@testing-library/react'
import { TestContext } from 'ra-test'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import ArtistActions from './ArtistActions'
import subsonic from '../subsonic'
import {
  openShareMenu,
  openDownloadMenu,
  DOWNLOAD_MENU_ARTIST,
} from '../actions'
import { ThemeProvider, createTheme } from '@material-ui/core/styles'

const mockDispatch = vi.fn()
vi.mock('react-redux', () => ({ useDispatch: () => mockDispatch }))

vi.mock('../subsonic', () => ({
  default: { getSimilarSongs2: vi.fn(), getTopSongs: vi.fn() },
}))

const { mockConfig, mockPermissions } = vi.hoisted(() => ({
  mockConfig: { enableSharing: true, enableDownloads: true },
  mockPermissions: { value: 'admin' },
}))
vi.mock('../config', () => ({ default: mockConfig }))

const mockNotify = vi.fn()
const mockGetList = vi.fn().mockResolvedValue({ data: [{ id: 's1' }] })
const mockRefreshMetadata = vi.fn().mockResolvedValue({ data: { id: 'ar1' } })

vi.mock('react-admin', async (importOriginal) => {
  const actual = await importOriginal()
  return {
    ...actual,
    useNotify: () => mockNotify,
    useDataProvider: () => ({
      getList: mockGetList,
      refreshMetadata: mockRefreshMetadata,
    }),
    usePermissions: () => ({ permissions: mockPermissions.value }),
    useTranslate: () => (x) => x,
  }
})

describe('ArtistActions', () => {
  const defaultRecord = {
    id: 'ar1',
    name: 'Artist',
    stats: { albumartist: { songCount: 3, albumCount: 1, size: 1024 } },
  }

  const renderArtistActions = (record = defaultRecord) => {
    const theme = createTheme()
    return render(
      <TestContext>
        <ThemeProvider theme={theme}>
          <ArtistActions record={record} />
        </ThemeProvider>
      </TestContext>,
    )
  }

  const clickActionButton = (actionKey) => {
    fireEvent.click(screen.getByText(`resources.artist.actions.${actionKey}`))
  }

  beforeEach(() => {
    vi.clearAllMocks()
    // Mock console.error to suppress error logging in tests
    vi.spyOn(console, 'error').mockImplementation(() => {})
    mockConfig.enableSharing = true
    mockConfig.enableDownloads = true
    mockPermissions.value = 'admin'

    const songWithReplayGain = {
      id: 'rec1',
      replayGain: {
        albumGain: -5,
        albumPeak: 1,
        trackGain: -6,
        trackPeak: 0.8,
      },
    }

    subsonic.getSimilarSongs2.mockResolvedValue({
      json: {
        'subsonic-response': {
          status: 'ok',
          similarSongs2: { song: [songWithReplayGain] },
        },
      },
    })
    subsonic.getTopSongs.mockResolvedValue({
      json: {
        'subsonic-response': {
          status: 'ok',
          topSongs: { song: [songWithReplayGain] },
        },
      },
    })
  })

  describe('Shuffle action', () => {
    it('shuffles songs when clicked', async () => {
      renderArtistActions()
      clickActionButton('shuffle')

      await waitFor(() =>
        expect(mockGetList).toHaveBeenCalledWith('song', {
          pagination: { page: 1, perPage: 500 },
          sort: { field: 'random', order: 'ASC' },
          filter: { album_artist_id: 'ar1', missing: false },
        }),
      )
      expect(mockDispatch).toHaveBeenCalled()
    })
  })

  describe('Radio action', () => {
    it('starts radio when clicked', async () => {
      renderArtistActions()
      clickActionButton('radio')

      await waitFor(() =>
        expect(subsonic.getSimilarSongs2).toHaveBeenCalledWith('ar1', 100),
      )
      expect(mockDispatch).toHaveBeenCalled()
    })

    it('maps replaygain info', async () => {
      renderArtistActions()
      clickActionButton('radio')

      await waitFor(() =>
        expect(subsonic.getSimilarSongs2).toHaveBeenCalledWith('ar1', 100),
      )
      const action = mockDispatch.mock.calls[0][0]
      expect(action.data.rec1).toMatchObject({
        rgAlbumGain: -5,
        rgAlbumPeak: 1,
        rgTrackGain: -6,
        rgTrackPeak: 0.8,
      })
    })
  })

  describe('Play action', () => {
    it('plays top songs when clicked', async () => {
      renderArtistActions()
      clickActionButton('topSongs')

      await waitFor(() =>
        expect(subsonic.getTopSongs).toHaveBeenCalledWith('Artist', 100),
      )
      expect(mockDispatch).toHaveBeenCalled()
    })

    it('maps replaygain info for top songs', async () => {
      renderArtistActions()
      clickActionButton('topSongs')

      await waitFor(() =>
        expect(subsonic.getTopSongs).toHaveBeenCalledWith('Artist', 100),
      )
      const action = mockDispatch.mock.calls[0][0]
      expect(action.data.rec1).toMatchObject({
        rgAlbumGain: -5,
        rgAlbumPeak: 1,
        rgTrackGain: -6,
        rgTrackPeak: 0.8,
      })
    })

    it('handles API rejection', async () => {
      subsonic.getTopSongs.mockRejectedValue(new Error('Network error'))

      renderArtistActions()
      clickActionButton('topSongs')

      await waitFor(() =>
        expect(subsonic.getTopSongs).toHaveBeenCalledWith('Artist', 100),
      )
      expect(mockNotify).toHaveBeenCalledWith('ra.page.error', 'warning')
      expect(mockDispatch).not.toHaveBeenCalled()
    })

    it('handles failed API response', async () => {
      subsonic.getTopSongs.mockResolvedValue({
        json: {
          'subsonic-response': {
            status: 'failed',
            error: { code: 40, message: 'Wrong username or password' },
          },
        },
      })

      renderArtistActions()
      clickActionButton('topSongs')

      await waitFor(() =>
        expect(subsonic.getTopSongs).toHaveBeenCalledWith('Artist', 100),
      )
      expect(mockNotify).toHaveBeenCalledWith('ra.page.error', 'warning')
      expect(mockDispatch).not.toHaveBeenCalled()
    })

    it('handles empty song list', async () => {
      subsonic.getTopSongs.mockResolvedValue({
        json: {
          'subsonic-response': {
            status: 'ok',
            topSongs: { song: [] },
          },
        },
      })

      renderArtistActions()
      clickActionButton('topSongs')

      await waitFor(() =>
        expect(subsonic.getTopSongs).toHaveBeenCalledWith('Artist', 100),
      )
      expect(mockNotify).toHaveBeenCalledWith(
        'message.noTopSongsFound',
        'warning',
      )
      expect(mockDispatch).not.toHaveBeenCalled()
    })

    it('handles missing topSongs property', async () => {
      subsonic.getTopSongs.mockResolvedValue({
        json: {
          'subsonic-response': {
            status: 'ok',
            // topSongs property is missing
          },
        },
      })

      renderArtistActions()
      clickActionButton('topSongs')

      await waitFor(() =>
        expect(subsonic.getTopSongs).toHaveBeenCalledWith('Artist', 100),
      )
      expect(mockNotify).toHaveBeenCalledWith(
        'message.noTopSongsFound',
        'warning',
      )
      expect(mockDispatch).not.toHaveBeenCalled()
    })
  })

  describe('Share action', () => {
    it('shows the share button and dispatches openShareMenu when clicked', () => {
      renderArtistActions()
      fireEvent.click(screen.getByText('ra.action.share'))
      expect(mockDispatch).toHaveBeenCalledWith(
        openShareMenu(['ar1'], 'artist', 'Artist'),
      )
    })

    it('hides the share button when sharing is disabled', () => {
      mockConfig.enableSharing = false
      renderArtistActions()
      expect(screen.queryByText('ra.action.share')).not.toBeInTheDocument()
    })
  })

  describe('Download action', () => {
    it('shows the download button with album-artist size and dispatches openDownloadMenu when clicked', () => {
      renderArtistActions()
      expect(screen.getByText('ra.action.download (1 KB)')).toBeInTheDocument()
      fireEvent.click(screen.getByText(/ra\.action\.download/))
      expect(mockDispatch).toHaveBeenCalledWith(
        openDownloadMenu(defaultRecord, DOWNLOAD_MENU_ARTIST),
      )
    })

    it('hides the download button when downloads are disabled', () => {
      mockConfig.enableDownloads = false
      renderArtistActions()
      expect(screen.queryByText(/ra\.action\.download/)).not.toBeInTheDocument()
    })
  })

  describe('Album-artist gating', () => {
    it('hides Share and Download for artists with no album-artist content', () => {
      renderArtistActions({ id: 'ar1', name: 'Artist', stats: {} })
      expect(screen.queryByText('ra.action.share')).not.toBeInTheDocument()
      expect(screen.queryByText(/ra\.action\.download/)).not.toBeInTheDocument()
    })

    it('hides Share and Download for a missing artist', () => {
      renderArtistActions({ ...defaultRecord, missing: true })
      expect(screen.queryByText('ra.action.share')).not.toBeInTheDocument()
      expect(screen.queryByText(/ra\.action\.download/)).not.toBeInTheDocument()
    })
  })

  describe('Refresh metadata action', () => {
    const refreshLabel = 'resources.album.actions.refresh'

    it('refreshes the artist metadata for admins', async () => {
      renderArtistActions()
      fireEvent.click(screen.getByRole('button', { name: refreshLabel }))

      await waitFor(() =>
        expect(mockRefreshMetadata).toHaveBeenCalledWith('artist', 'ar1'),
      )
    })

    it('hides the action for non-admin users', () => {
      mockPermissions.value = 'regular'
      renderArtistActions()
      expect(
        screen.queryByRole('button', { name: refreshLabel }),
      ).not.toBeInTheDocument()
    })
  })
})
