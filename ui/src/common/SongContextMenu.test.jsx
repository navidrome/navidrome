import React from 'react'
import { render, fireEvent, screen, waitFor } from '@testing-library/react'
import { TestContext } from 'ra-test'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { SongContextMenu } from './SongContextMenu'

vi.mock('../dataProvider', () => ({
  httpClient: vi.fn(),
}))

vi.mock('react-redux', () => ({ useDispatch: () => vi.fn() }))

vi.mock('react-admin', async (importOriginal) => {
  const actual = await importOriginal()
  return {
    ...actual,
    useRedirect: () => (url) => {
      window.location.hash = `#${url}`
    },
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
})
