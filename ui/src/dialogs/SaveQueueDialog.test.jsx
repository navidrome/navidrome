import * as React from 'react'
import { TestContext } from 'ra-test'
import { DataProviderContext } from 'react-admin'
import {
  cleanup,
  fireEvent,
  render,
  waitFor,
  screen,
} from '@testing-library/react'
import { SaveQueueDialog } from './SaveQueueDialog'
import { describe, afterEach, it, expect, vi, beforeAll } from 'vitest'

const queue = [{ trackId: 'song-1' }, { trackId: 'song-2' }]

const createTestUtils = (mockDataProvider) =>
  render(
    <DataProviderContext.Provider value={mockDataProvider}>
      <TestContext
        initialState={{
          saveQueueDialog: { open: true },
          player: { queue },
          admin: { ui: { optimistic: false } },
        }}
      >
        <SaveQueueDialog />
      </TestContext>
    </DataProviderContext.Provider>,
  )

// Mock useHistory to update window.location.hash on push
vi.mock('react-router-dom', async (importOriginal) => {
  const actual = await importOriginal()
  return {
    ...actual,
    useHistory: () => ({
      push: (url) => {
        window.location.hash = `#${url}`
      },
    }),
  }
})

beforeAll(() => {
  // No need to patch pushState anymore
})

describe('SaveQueueDialog', () => {
  afterEach(cleanup)

  it('creates playlist and saves queue', async () => {
    const mockDataProvider = {
      create: vi
        .fn()
        .mockResolvedValueOnce({ data: { id: 'created-id' } })
        .mockResolvedValueOnce({ data: { id: 'pt-id' } }),
    }

    createTestUtils(mockDataProvider)

    fireEvent.change(screen.getByRole('textbox'), {
      target: { value: 'my playlist' },
    })
    fireEvent.click(screen.getByTestId('save-queue-save'))

    await waitFor(() => {
      expect(mockDataProvider.create).toHaveBeenNthCalledWith(1, 'playlist', {
        data: { name: 'my playlist' },
      })
    })
    await waitFor(() => {
      expect(mockDataProvider.create).toHaveBeenNthCalledWith(
        2,
        'playlistTrack',
        {
          data: { ids: ['song-1', 'song-2'] },
          filter: { playlist_id: 'created-id' },
        },
      )
    })
    await waitFor(() => {
      expect(window.location.hash).toBe('#/playlist/created-id/show')
    })
  })

  it('disables save button when name is empty', () => {
    const mockDataProvider = { create: vi.fn() }
    createTestUtils(mockDataProvider)
    expect(screen.getByTestId('save-queue-save')).toBeDisabled()
  })
})
