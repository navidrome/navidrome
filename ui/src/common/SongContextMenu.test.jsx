import React from 'react'
import { render, fireEvent, screen, waitFor } from '@testing-library/react'
import { TestContext } from 'ra-test'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { SongContextMenu } from './SongContextMenu'
import subsonic from '../subsonic'

vi.mock('../dataProvider', () => ({
  httpClient: vi.fn(),
}))

const mockDispatch = vi.fn()
vi.mock('react-redux', () => ({ useDispatch: () => mockDispatch }))

vi.mock('../subsonic', () => ({
  default: { getSimilarSongs2: vi.fn() },
}))

const mockNotify = vi.fn()

vi.mock('react-admin', async (importOriginal) => {
  const actual = await importOriginal()
  return {
    ...actual,
    useRedirect: () => (url) => {
      window.location.hash = `#${url}`
    },
    useNotify: () => mockNotify,
    useDataProvider: () => ({
      getPlaylists: vi.fn().mockResolvedValue({
        data: [{ id: 'pl1', name: 'Pl 1' }],
      }),
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
    subsonic.getSimilarSongs2.mockResolvedValue({
      json: {
        'subsonic-response': {
          status: 'ok',
          similarSongs2: { song: [{ id: 'rec1' }] },
        },
      },
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

  it('calls getSimilarSongs2 when Play Similar is clicked', async () => {
    render(
      <TestContext>
        <SongContextMenu record={{ id: 'song1', size: 1 }} resource="song" />
      </TestContext>,
    )

    fireEvent.click(screen.getAllByRole('button')[1])
    await waitFor(() =>
      screen.getByText(/resources\.song\.actions\.playSimilar/),
    )
    fireEvent.click(screen.getByText(/resources\.song\.actions\.playSimilar/))

    await waitFor(() =>
      expect(subsonic.getSimilarSongs2).toHaveBeenCalledWith('song1', 100),
    )
    expect(mockDispatch).toHaveBeenCalled()
    expect(mockNotify).not.toHaveBeenCalled()
  })

  it('notifies when no similar songs are returned', async () => {
    subsonic.getSimilarSongs2.mockResolvedValueOnce({
      json: {
        'subsonic-response': { status: 'ok', similarSongs2: { song: [] } },
      },
    })

    render(
      <TestContext>
        <SongContextMenu record={{ id: 'song1', size: 1 }} resource="song" />
      </TestContext>,
    )

    fireEvent.click(screen.getAllByRole('button')[1])
    await waitFor(() =>
      screen.getByText(/resources\.song\.actions\.playSimilar/),
    )
    fireEvent.click(screen.getByText(/resources\.song\.actions\.playSimilar/))

    await waitFor(() =>
      expect(subsonic.getSimilarSongs2).toHaveBeenCalledWith('song1', 100),
    )
    expect(mockDispatch).not.toHaveBeenCalled()
    expect(mockNotify).toHaveBeenCalledWith(
      'message.noSimilarSongsFound',
      'warning',
    )
  })
})
