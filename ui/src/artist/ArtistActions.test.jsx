import React from 'react'
import { render, fireEvent, waitFor, screen } from '@testing-library/react'
import { TestContext } from 'ra-test'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import ArtistActions from './ArtistActions'
import subsonic from '../subsonic'
import { ThemeProvider, createMuiTheme } from '@material-ui/core/styles'

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
  beforeEach(() => {
    vi.clearAllMocks()
    subsonic.getSimilarSongs2.mockResolvedValue({
      json: {
        'subsonic-response': {
          status: 'ok',
          similarSongs2: { song: [{ id: 'rec1' }] },
        },
      },
    })
    subsonic.getTopSongs.mockResolvedValue({
      json: {
        'subsonic-response': {
          status: 'ok',
          topSongs: { song: [{ id: 'rec1' }] },
        },
      },
    })
  })

  it('shuffles songs when Shuffle is clicked', async () => {
    const theme = createMuiTheme()
    render(
      <TestContext>
        <ThemeProvider theme={theme}>
          <ArtistActions record={{ id: 'ar1' }} />
        </ThemeProvider>
      </TestContext>,
    )

    fireEvent.click(screen.getByText('resources.artist.actions.shuffle'))
    await waitFor(() =>
      expect(mockGetList).toHaveBeenCalledWith('song', {
        pagination: { page: 1, perPage: 500 },
        sort: { field: 'random', order: 'ASC' },
        filter: { album_artist_id: 'ar1', missing: false },
      }),
    )
    expect(mockDispatch).toHaveBeenCalled()
  })

  it('starts radio when Radio is clicked', async () => {
    const theme = createMuiTheme()
    render(
      <TestContext>
        <ThemeProvider theme={theme}>
          <ArtistActions record={{ id: 'ar1' }} />
        </ThemeProvider>
      </TestContext>,
    )

    fireEvent.click(screen.getByText('resources.artist.actions.radio'))
    await waitFor(() =>
      expect(subsonic.getSimilarSongs2).toHaveBeenCalledWith('ar1', 100),
    )
    expect(mockDispatch).toHaveBeenCalled()
  })

  it('plays top songs when Play is clicked', async () => {
    const theme = createMuiTheme()
    render(
      <TestContext>
        <ThemeProvider theme={theme}>
          <ArtistActions record={{ id: 'ar1', name: 'Artist' }} />
        </ThemeProvider>
      </TestContext>,
    )

    fireEvent.click(screen.getByText('resources.artist.actions.play'))
    await waitFor(() =>
      expect(subsonic.getTopSongs).toHaveBeenCalledWith('Artist', 50),
    )
    expect(mockDispatch).toHaveBeenCalled()
  })
})
