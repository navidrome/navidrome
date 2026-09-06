import React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { ThemeProvider, createTheme } from '@material-ui/core/styles'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import AlbumActions from './AlbumActions'

const { mockConfig, mockPermissions } = vi.hoisted(() => ({
  mockConfig: {
    enableSharing: false,
    enableDownloads: false,
    losslessFormats: 'FLAC,WAV',
  },
  mockPermissions: { value: 'admin' },
}))
vi.mock('../config', () => ({ default: mockConfig }))

vi.mock('react-redux', () => ({
  useDispatch: () => vi.fn(),
  useSelector: () => ({}),
}))

const mockRefreshMetadata = vi.fn()

vi.mock('react-admin', async (importOriginal) => {
  const actual = await importOriginal()
  return {
    ...actual,
    useNotify: () => vi.fn(),
    useDataProvider: () => ({ refreshMetadata: mockRefreshMetadata }),
    usePermissions: () => ({ permissions: mockPermissions.value }),
    useTranslate: () => (x) => x,
  }
})

describe('AlbumActions', () => {
  const record = { id: 'al1', name: 'Album', size: 1024 }
  const refreshLabel = 'resources.album.actions.refresh'

  beforeEach(() => {
    vi.clearAllMocks()
    mockPermissions.value = 'admin'
    mockRefreshMetadata.mockResolvedValue({ data: { id: 'al1' } })
  })

  const renderAlbumActions = () =>
    render(
      <ThemeProvider theme={createTheme()}>
        <AlbumActions record={record} ids={[]} data={{}} />
      </ThemeProvider>,
    )

  it('refreshes the album metadata for admins', async () => {
    renderAlbumActions()
    fireEvent.click(screen.getByRole('button', { name: refreshLabel }))

    await waitFor(() =>
      expect(mockRefreshMetadata).toHaveBeenCalledWith('album', 'al1'),
    )
  })

  it('hides the action for non-admin users', () => {
    mockPermissions.value = 'regular'
    renderAlbumActions()
    expect(
      screen.queryByRole('button', { name: refreshLabel }),
    ).not.toBeInTheDocument()
  })
})
