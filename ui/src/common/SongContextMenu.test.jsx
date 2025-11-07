import React from 'react'
import { render, fireEvent, screen, waitFor } from '@testing-library/react'
import { TestContext } from 'ra-test'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { SongContextMenu } from './SongContextMenu'

vi.mock('../dataProvider', () => ({
  httpClient: vi.fn(),
}))

vi.mock('react-redux', () => ({ useDispatch: () => vi.fn() }))

const getPlaylistsMock = vi.fn()

vi.mock('react-admin', async (importOriginal) => {
  const actual = await importOriginal()
  return {
    ...actual,
    useRedirect: () => (url) => {
      window.location.hash = `#${url}`
    },
    useDataProvider: () => ({
      getPlaylists: getPlaylistsMock,
      inspect: vi.fn().mockResolvedValue({
        data: { rawTags: {} },
      }),
    }),
  }
})

describe('SongContextMenu', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    window.location.hash = ''
    getPlaylistsMock.mockResolvedValue({
      data: [{ id: 'pl1', name: 'Pl 1' }],
    })
  })

  it('navigates to playlist when selected', async () => {
    render(
      <TestContext>
        <SongContextMenu record={{ id: 'song1', size: 1 }} resource="song" />
      </TestContext>,
    )
    fireEvent.click(screen.getAllByRole('button')[1])
    await waitFor(() =>
      screen.getByText(/resources\.song\.actions\.showInPlaylist/),
    )
    fireEvent.click(
      screen.getByText(/resources\.song\.actions\.showInPlaylist/),
    )
    await waitFor(() => screen.getByText('Pl 1'))
    fireEvent.click(screen.getByText('Pl 1'))
    expect(window.location.hash).toBe('#/playlist/pl1/show')
  })

  it('stops event propagation when playlist submenu is closed', async () => {
    const mockOnClick = vi.fn()
    render(
      <TestContext>
        <div onClick={mockOnClick}>
          <SongContextMenu record={{ id: 'song1', size: 1 }} resource="song" />
        </div>
      </TestContext>,
    )

    // Open main menu
    fireEvent.click(screen.getAllByRole('button')[1])
    await waitFor(() =>
      screen.getByText(/resources\.song\.actions\.showInPlaylist/),
    )

    // Open playlist submenu
    fireEvent.click(
      screen.getByText(/resources\.song\.actions\.showInPlaylist/),
    )
    await waitFor(() => screen.getByText('Pl 1'))

    // Click outside the playlist submenu (should close it without triggering parent click)
    fireEvent.click(document.body)

    expect(mockOnClick).not.toHaveBeenCalled()
  })

  it('does nothing when "Show in Playlist" is disabled', async () => {
    getPlaylistsMock.mockResolvedValue({ data: [] })
    const mockOnClick = vi.fn()
    render(
      <TestContext>
        <div onClick={mockOnClick}>
          <SongContextMenu record={{ id: 'song1', size: 1 }} resource="song" />
        </div>
      </TestContext>,
    )

    fireEvent.click(screen.getAllByRole('button')[1])
    await waitFor(() =>
      screen.getByText(/resources\.song\.actions\.showInPlaylist/),
    )

    fireEvent.click(
      screen.getByText(/resources\.song\.actions\.showInPlaylist/),
    )
    expect(mockOnClick).not.toHaveBeenCalled()
  })
})
