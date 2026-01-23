import React from 'react'
import { render, fireEvent, waitFor, screen } from '@testing-library/react'
import { TestContext } from 'ra-test'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import ArtistActions from './ArtistActions'
import subsonic from '../subsonic'
import { ThemeProvider, createTheme } from '@material-ui/core/styles'

const mockDispatch = vi.fn()
vi.mock('react-redux', () => ({ useDispatch: () => mockDispatch }))

vi.mock('../subsonic', () => ({
  default: { getSimilarSongs2: vi.fn(), getTopSongs: vi.fn() },
}))

const mockNotify = vi.fn()
const mockGetList = vi.fn().mockResolvedValue({ data: [{ id: 's1' }] })

vi.mock('react-admin', async (importOriginal) => {
  const actual = await importOriginal()
  return {
    ...actual,
    useNotify: () => mockNotify,
    useDataProvider: () => ({ getList: mockGetList }),
    useTranslate: () => (x) => x,
  }
})

describe('ArtistActions', () => {
  const defaultRecord = { id: 'ar1', name: 'Artist' }

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
})
