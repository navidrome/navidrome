import React from 'react'
import { render, waitFor } from '@testing-library/react'
import { RecordContextProvider } from 'react-admin'
import { ThemeProvider, createTheme } from '@material-ui/core/styles'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ArtistDetails } from './ArtistShow'
import subsonic from '../subsonic'

vi.mock('../subsonic', () => ({
  default: { getArtistInfo: vi.fn(), getCoverArtUrl: vi.fn() },
}))

// Not under test here: isolate ArtistDetails from the leaf presentational views.
vi.mock('./DesktopArtistDetails', () => ({ default: () => null }))
vi.mock('./MobileArtistDetails', () => ({ default: () => null }))

const mockGetArtistInfo = subsonic.getArtistInfo

describe('ArtistDetails', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockGetArtistInfo.mockResolvedValue({
      json: {
        'subsonic-response': {
          status: 'ok',
          artistInfo: { biography: 'fetched' },
        },
      },
    })
  })

  const theme = createTheme()

  const wrap = (record) => (
    <ThemeProvider theme={theme}>
      <RecordContextProvider value={record}>
        <ArtistDetails />
      </RecordContextProvider>
    </ThemeProvider>
  )

  const renderDetails = (record) => render(wrap(record))

  it('re-fetches the artist info when the record object changes', async () => {
    const record = { id: 'ar1', name: 'Artist', biography: 'old' }
    const { rerender } = renderDetails(record)
    await waitFor(() => expect(mockGetArtistInfo).toHaveBeenCalledTimes(1))

    rerender(wrap({ ...record, biography: 'new' }))

    await waitFor(() => expect(mockGetArtistInfo).toHaveBeenCalledTimes(2))
  })

  it('does not re-fetch when the same record object is passed again', async () => {
    const record = { id: 'ar1', name: 'Artist', biography: 'old' }
    const { rerender } = renderDetails(record)
    await waitFor(() => expect(mockGetArtistInfo).toHaveBeenCalledTimes(1))

    rerender(wrap(record))

    expect(mockGetArtistInfo).toHaveBeenCalledTimes(1)
  })
})
