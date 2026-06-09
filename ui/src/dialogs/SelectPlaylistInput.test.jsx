import * as React from 'react'
import { TestContext } from 'ra-test'
import { DataProviderContext } from 'react-admin'
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react'
import { SelectPlaylistInput } from './SelectPlaylistInput'
import { describe, beforeAll, afterEach, it, expect, vi } from 'vitest'

const mockPlaylists = [
  { id: 'playlist-1', name: 'Rock Classics', ownerId: 'admin' },
  { id: 'playlist-2', name: 'Jazz Collection', ownerId: 'admin' },
  { id: 'playlist-3', name: 'Electronic Beats', ownerId: 'admin' },
  { id: 'playlist-4', name: 'Chill Vibes', ownerId: 'user2' }, // Not writable by admin
]

const mockIndexedData = {
  'playlist-1': { id: 'playlist-1', name: 'Rock Classics', ownerId: 'admin' },
  'playlist-2': { id: 'playlist-2', name: 'Jazz Collection', ownerId: 'admin' },
  'playlist-3': {
    id: 'playlist-3',
    name: 'Electronic Beats',
    ownerId: 'admin',
  },
  'playlist-4': { id: 'playlist-4', name: 'Chill Vibes', ownerId: 'user2' },
}

const createTestComponent = (
  mockDataProvider = null,
  onChangeMock = vi.fn(),
  playlists = mockPlaylists,
  indexedData = mockIndexedData,
) => {
  const dataProvider = mockDataProvider || {
    getList: vi.fn().mockResolvedValue({
      data: playlists,
      total: playlists.length,
    }),
  }

  return render(
    <DataProviderContext.Provider value={dataProvider}>
      <TestContext
        initialState={{
          admin: {
            ui: { optimistic: false },
            resources: {
              playlist: {
                data: indexedData,
                list: {
                  cachedRequests: {
                    '{"pagination":{"page":1,"perPage":-1},"sort":{"field":"name","order":"ASC"},"filter":{"smart":false}}':
                      {
                        ids: Object.keys(indexedData),
                        total: Object.keys(indexedData).length,
                      },
                  },
                },
              },
            },
          },
        }}
      >
        <SelectPlaylistInput onChange={onChangeMock} />
      </TestContext>
    </DataProviderContext.Provider>,
  )
}

describe('SelectPlaylistInput', () => {
  beforeAll(() => localStorage.setItem('userId', 'admin'))
  afterEach(cleanup)

  describe('Basic Functionality', () => {
    it('should render search field and playlist list', async () => {
      const onChangeMock = vi.fn()
      createTestComponent(null, onChangeMock)

      await waitFor(() => {
        expect(screen.getByRole('textbox')).toBeInTheDocument()
        expect(screen.getByText('Rock Classics')).toBeInTheDocument()
        expect(screen.getByText('Jazz Collection')).toBeInTheDocument()
        expect(screen.getByText('Electronic Beats')).toBeInTheDocument()
      })

      // Should not show playlists not owned by admin (not writable)
      expect(screen.queryByText('Chill Vibes')).not.toBeInTheDocument()
    })

    it('should filter playlists based on search input', async () => {
      const onChangeMock = vi.fn()
      createTestComponent(null, onChangeMock)

      await waitFor(() => {
        expect(screen.getByText('Rock Classics')).toBeInTheDocument()
      })

      const searchInput = screen.getByRole('textbox')
      fireEvent.change(searchInput, { target: { value: 'rock' } })

      await waitFor(() => {
        expect(screen.getByText('Rock Classics')).toBeInTheDocument()
        expect(screen.queryByText('Jazz Collection')).not.toBeInTheDocument()
        expect(screen.queryByText('Electronic Beats')).not.toBeInTheDocument()
      })
    })

    it('should handle case-insensitive search', async () => {
      const onChangeMock = vi.fn()
      createTestComponent(null, onChangeMock)

      await waitFor(() => {
        expect(screen.getByText('Jazz Collection')).toBeInTheDocument()
      })

      const searchInput = screen.getByRole('textbox')
      fireEvent.change(searchInput, { target: { value: 'JAZZ' } })

      await waitFor(() => {
        expect(screen.getByText('Jazz Collection')).toBeInTheDocument()
        expect(screen.queryByText('Rock Classics')).not.toBeInTheDocument()
      })
    })
  })

  describe('Playlist Selection', () => {
    it('should select and deselect playlists by clicking', async () => {
      const onChangeMock = vi.fn()
      createTestComponent(null, onChangeMock)

      await waitFor(() => {
        expect(screen.getByText('Rock Classics')).toBeInTheDocument()
      })

      // Select first playlist
      const rockPlaylist = screen.getByText('Rock Classics')
      fireEvent.click(rockPlaylist)

      await waitFor(() => {
        expect(onChangeMock).toHaveBeenCalledWith([
          { id: 'playlist-1', name: 'Rock Classics', ownerId: 'admin' },
        ])
      })

      // Select second playlist
      const jazzPlaylist = screen.getByText('Jazz Collection')
      fireEvent.click(jazzPlaylist)

      await waitFor(() => {
        expect(onChangeMock).toHaveBeenCalledWith([
          { id: 'playlist-1', name: 'Rock Classics', ownerId: 'admin' },
          { id: 'playlist-2', name: 'Jazz Collection', ownerId: 'admin' },
        ])
      })

      // Deselect first playlist
      fireEvent.click(rockPlaylist)

      await waitFor(() => {
        expect(onChangeMock).toHaveBeenCalledWith([
          { id: 'playlist-2', name: 'Jazz Collection', ownerId: 'admin' },
        ])
      })
    })

    it('should show selected playlists as chips', async () => {
      const onChangeMock = vi.fn()
      createTestComponent(null, onChangeMock)

      await waitFor(() => {
        expect(screen.getByText('Rock Classics')).toBeInTheDocument()
      })

      // Select a playlist
      const rockPlaylist = screen.getByText('Rock Classics')
      fireEvent.click(rockPlaylist)

      await waitFor(() => {
        // Should show the selected playlist as a chip
        const chips = screen.getAllByText('Rock Classics')
        expect(chips.length).toBeGreaterThan(1) // One in list, one in chip
      })
    })

    it('should remove selected playlists via chip remove button', async () => {
      const onChangeMock = vi.fn()
      createTestComponent(null, onChangeMock)

      await waitFor(() => {
        expect(screen.getByText('Rock Classics')).toBeInTheDocument()
      })

      // Select a playlist
      const rockPlaylist = screen.getByText('Rock Classics')
      fireEvent.click(rockPlaylist)

      await waitFor(() => {
        // Should show selected playlist as chip
        const chips = screen.getAllByText('Rock Classics')
        expect(chips.length).toBeGreaterThan(1)
      })

      // Find and click the remove button (translation key)
      const removeButton = screen.getByText('×')
      fireEvent.click(removeButton)

      await waitFor(() => {
        expect(onChangeMock).toHaveBeenCalledWith([])
        // Should only have one instance (in the list) after removal
        const remainingChips = screen.getAllByText('Rock Classics')
        expect(remainingChips.length).toBe(1)
      })
    })
  })

  describe('Create New Playlist', () => {
    it('should create new playlist by pressing Enter', async () => {
      const onChangeMock = vi.fn()
      createTestComponent(null, onChangeMock)

      await waitFor(() => {
        expect(screen.getByRole('textbox')).toBeInTheDocument()
      })

      const searchInput = screen.getByRole('textbox')
      fireEvent.change(searchInput, { target: { value: 'My New Playlist' } })
      fireEvent.keyDown(searchInput, { key: 'Enter' })

      await waitFor(() => {
        expect(onChangeMock).toHaveBeenCalledWith([{ name: 'My New Playlist' }])
      })

      // Input should be cleared after creating
      expect(searchInput.value).toBe('')
    })

    it('should create new playlist by clicking add button', async () => {
      const onChangeMock = vi.fn()
      createTestComponent(null, onChangeMock)

      await waitFor(() => {
        expect(screen.getByRole('textbox')).toBeInTheDocument()
      })

      const searchInput = screen.getByRole('textbox')
      fireEvent.change(searchInput, { target: { value: 'Another Playlist' } })

      // Find the add button by the translation key title
      const addButton = screen.getByTitle(
        'resources.playlist.actions.addNewPlaylist',
      )
      fireEvent.click(addButton)

      await waitFor(() => {
        expect(onChangeMock).toHaveBeenCalledWith([
          { name: 'Another Playlist' },
        ])
      })
    })

    it('should not show create option for existing playlist names', async () => {
      const onChangeMock = vi.fn()
      createTestComponent(null, onChangeMock)

      await waitFor(() => {
        expect(screen.getByRole('textbox')).toBeInTheDocument()
      })

      const searchInput = screen.getByRole('textbox')
      fireEvent.change(searchInput, { target: { value: 'Rock Classics' } })

      await waitFor(() => {
        expect(
          screen.queryByText('resources.playlist.actions.addNewPlaylist'),
        ).not.toBeInTheDocument()
      })
    })

    it('should not create playlist with empty name', async () => {
      const onChangeMock = vi.fn()
      createTestComponent(null, onChangeMock)

      await waitFor(() => {
        expect(screen.getByRole('textbox')).toBeInTheDocument()
      })

      const searchInput = screen.getByRole('textbox')
      fireEvent.change(searchInput, { target: { value: '   ' } }) // Only spaces
      fireEvent.keyDown(searchInput, { key: 'Enter' })

      // Should not call onChange
      expect(onChangeMock).not.toHaveBeenCalled()
    })

    it('should show create options in appropriate contexts', async () => {
      const onChangeMock = vi.fn()
      createTestComponent(null, onChangeMock)

      await waitFor(() => {
        expect(screen.getByRole('textbox')).toBeInTheDocument()
      })

      const searchInput = screen.getByRole('textbox')

      // When typing a new name, should show create options
      fireEvent.change(searchInput, { target: { value: 'My New Playlist' } })

      await waitFor(() => {
        // Should show the add button in the search field
        expect(
          screen.getByTitle('resources.playlist.actions.addNewPlaylist'),
        ).toBeInTheDocument()
        // Should also show hint in empty message when no matches
        expect(
          screen.getByText('resources.playlist.actions.pressEnterToCreate'),
        ).toBeInTheDocument()
      })
    })
  })

  describe('Mixed Operations', () => {
    it('should handle selecting existing playlists and creating new ones', async () => {
      const onChangeMock = vi.fn()
      createTestComponent(null, onChangeMock)

      await waitFor(() => {
        expect(screen.getByText('Rock Classics')).toBeInTheDocument()
      })

      // Select existing playlist
      const rockPlaylist = screen.getByText('Rock Classics')
      fireEvent.click(rockPlaylist)

      await waitFor(() => {
        expect(onChangeMock).toHaveBeenCalledWith([
          { id: 'playlist-1', name: 'Rock Classics', ownerId: 'admin' },
        ])
      })

      // Create new playlist
      const searchInput = screen.getByRole('textbox')
      fireEvent.change(searchInput, { target: { value: 'New Mix' } })
      fireEvent.keyDown(searchInput, { key: 'Enter' })

      await waitFor(() => {
        expect(onChangeMock).toHaveBeenCalledWith([
          { id: 'playlist-1', name: 'Rock Classics', ownerId: 'admin' },
          { name: 'New Mix' },
        ])
      })
    })

    it('should maintain selections when searching', async () => {
      const onChangeMock = vi.fn()
      createTestComponent(null, onChangeMock)

      await waitFor(() => {
        expect(screen.getByText('Rock Classics')).toBeInTheDocument()
      })

      // Select a playlist
      const rockPlaylist = screen.getByText('Rock Classics')
      fireEvent.click(rockPlaylist)

      // Filter the list
      const searchInput = screen.getByRole('textbox')
      fireEvent.change(searchInput, { target: { value: 'jazz' } })

      await waitFor(() => {
        // Should still show selected playlists section
        // Rock Classics should still be visible as a selected chip even though filtered out
        expect(screen.getByText('Rock Classics')).toBeInTheDocument() // In selected chips
        expect(screen.getByText('Jazz Collection')).toBeInTheDocument()
      })
    })
  })

  describe('Empty States', () => {
    it('should show empty message when no playlists exist', async () => {
      const onChangeMock = vi.fn()
      createTestComponent(null, onChangeMock, [], {})

      await waitFor(() => {
        expect(
          screen.getByText('resources.playlist.message.noPlaylists'),
        ).toBeInTheDocument()
      })
    })

    it('should show "no results" message when search returns no matches', async () => {
      const onChangeMock = vi.fn()
      createTestComponent(null, onChangeMock)

      await waitFor(() => {
        expect(screen.getByRole('textbox')).toBeInTheDocument()
      })

      const searchInput = screen.getByRole('textbox')
      fireEvent.change(searchInput, {
        target: { value: 'NonExistentPlaylist' },
      })

      await waitFor(() => {
        expect(
          screen.getByText('resources.playlist.message.noPlaylistsFound'),
        ).toBeInTheDocument()
        expect(
          screen.getByText('resources.playlist.actions.pressEnterToCreate'),
        ).toBeInTheDocument()
      })
    })
  })

  describe('Keyboard Navigation', () => {
    it('should not create playlist on Enter if input is empty', async () => {
      const onChangeMock = vi.fn()
      createTestComponent(null, onChangeMock)

      await waitFor(() => {
        expect(screen.getByRole('textbox')).toBeInTheDocument()
      })

      const searchInput = screen.getByRole('textbox')
      fireEvent.keyDown(searchInput, { key: 'Enter' })

      expect(onChangeMock).not.toHaveBeenCalled()
    })

    it('should handle other keys without side effects', async () => {
      const onChangeMock = vi.fn()
      createTestComponent(null, onChangeMock)

      await waitFor(() => {
        expect(screen.getByRole('textbox')).toBeInTheDocument()
      })

      const searchInput = screen.getByRole('textbox')
      fireEvent.change(searchInput, { target: { value: 'test' } })
      fireEvent.keyDown(searchInput, { key: 'ArrowDown' })
      fireEvent.keyDown(searchInput, { key: 'Tab' })
      fireEvent.keyDown(searchInput, { key: 'Escape' })

      // Should not create playlist or trigger onChange
      expect(onChangeMock).not.toHaveBeenCalled()
      expect(searchInput.value).toBe('test')
    })
  })

  describe('Integration Scenarios', () => {
    it('should handle complex workflow: search, select, create, remove', async () => {
      const onChangeMock = vi.fn()
      createTestComponent(null, onChangeMock)

      await waitFor(() => {
        expect(screen.getByText('Rock Classics')).toBeInTheDocument()
      })

      // Search and select existing playlist
      const searchInput = screen.getByRole('textbox')
      fireEvent.change(searchInput, { target: { value: 'rock' } })

      const rockPlaylist = screen.getByText('Rock Classics')
      fireEvent.click(rockPlaylist)

      // Clear search and create new playlist
      fireEvent.change(searchInput, { target: { value: 'My Custom Mix' } })
      fireEvent.keyDown(searchInput, { key: 'Enter' })

      await waitFor(() => {
        expect(onChangeMock).toHaveBeenCalledWith([
          { id: 'playlist-1', name: 'Rock Classics', ownerId: 'admin' },
          { name: 'My Custom Mix' },
        ])
      })

      // Remove the first selected playlist via chip
      const removeButtons = screen.getAllByText('×')
      fireEvent.click(removeButtons[0])

      await waitFor(() => {
        expect(onChangeMock).toHaveBeenCalledWith([{ name: 'My Custom Mix' }])
      })
    })
  })
})
