import React from 'react'
import { render, fireEvent, screen } from '@testing-library/react'
import { TestContext } from 'ra-test'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ThemeProvider, createTheme } from '@material-ui/core/styles'
import { AlbumContextMenu, ArtistContextMenu } from './ContextMenus'

const mockDispatch = vi.fn()
vi.mock('react-redux', () => ({ useDispatch: () => mockDispatch }))

const { mockConfig } = vi.hoisted(() => ({
  mockConfig: {
    enableSharing: true,
    enableDownloads: true,
    enableFavourites: false,
  },
}))
vi.mock('../config', () => ({ default: mockConfig }))

vi.mock('react-admin', async (importOriginal) => {
  const actual = await importOriginal()
  return {
    ...actual,
    useNotify: () => vi.fn(),
    useDataProvider: () => ({ getList: vi.fn() }),
    useTranslate: () => (x) => x,
  }
})

describe('ContextMenus', () => {
  const renderMenu = (Menu, record) => {
    render(
      <TestContext>
        <ThemeProvider theme={createTheme()}>
          <Menu record={record} />
        </ThemeProvider>
      </TestContext>,
    )
    fireEvent.click(screen.getByLabelText('more'))
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mockConfig.enableSharing = true
    mockConfig.enableDownloads = true
  })

  describe('ArtistContextMenu', () => {
    const withAlbumArtist = {
      id: 'ar1',
      name: 'Artist',
      stats: { albumartist: { songCount: 3, albumCount: 1, size: 1024 } },
    }

    it('shows the album-artist size on the download item', () => {
      renderMenu(ArtistContextMenu, withAlbumArtist)
      expect(screen.getByText('ra.action.download (1 KB)')).toBeInTheDocument()
    })

    it('hides download and share for artists with no album-artist content', () => {
      renderMenu(ArtistContextMenu, { id: 'ar1', name: 'Artist', stats: {} })
      expect(screen.queryByText(/ra\.action\.download/)).not.toBeInTheDocument()
      expect(screen.queryByText('ra.action.share')).not.toBeInTheDocument()
    })
  })

  describe('AlbumContextMenu', () => {
    it('uses the total size on the album download item', () => {
      renderMenu(AlbumContextMenu, {
        id: 'al1',
        name: 'Album',
        duration: 100,
        size: 1024 * 1024,
      })
      expect(screen.getByText('ra.action.download (1 MB)')).toBeInTheDocument()
    })
  })
})
