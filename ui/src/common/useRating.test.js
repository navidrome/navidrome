import { renderHook, act } from '@testing-library/react-hooks'
import { vi, describe, it, expect, beforeEach } from 'vitest'
import { useRating } from './useRating'
import subsonic from '../subsonic'
import { useDataProvider } from 'react-admin'

const mockRefresh = vi.fn()

vi.mock('../subsonic', () => ({
  default: {
    setRating: vi.fn(() => Promise.resolve()),
  },
}))

vi.mock('react-admin', async () => {
  const actual = await vi.importActual('react-admin')
  return {
    ...actual,
    useDataProvider: vi.fn(),
    useNotify: vi.fn(() => vi.fn()),
    useRefresh: vi.fn(() => mockRefresh),
  }
})

describe('useRating', () => {
  let getOne
  beforeEach(() => {
    getOne = vi.fn((resource, params) =>
      Promise.resolve({ data: { id: params.id } }),
    )
    useDataProvider.mockReturnValue({ getOne })
    vi.clearAllMocks()
  })

  it('returns rating value from record', () => {
    const record = { id: 'sg-1', rating: 3 }
    const { result } = renderHook(() => useRating('song', record))
    const [rate, rating, loading] = result.current
    expect(rating).toBe(3)
    expect(loading).toBe(false)
    expect(typeof rate).toBe('function')
  })

  it('sets rating using targetId and calls setRating API', async () => {
    const record = { id: 'sg-1', rating: 0 }
    const { result } = renderHook(() => useRating('song', record))
    await act(async () => {
      await result.current[0](4, 'sg-1')
    })
    expect(subsonic.setRating).toHaveBeenCalledWith('sg-1', 4)
    expect(getOne).toHaveBeenCalledWith('song', { id: 'sg-1' })
  })

  it('handles zero rating (unrate)', async () => {
    const record = { id: 'sg-1', rating: 5 }
    const { result } = renderHook(() => useRating('song', record))
    await act(async () => {
      await result.current[0](0, 'sg-1')
    })
    expect(subsonic.setRating).toHaveBeenCalledWith('sg-1', 0)
  })

  describe('playlist track scenarios', () => {
    it('refreshes the song and reloads the list for playlist tracks', async () => {
      const record = {
        id: '1',
        mediaFileId: 'sg-1',
        playlistId: 'pl-1',
        rating: 2,
      }
      const { result } = renderHook(() => useRating('playlistTrack', record))
      await act(async () => {
        await result.current[0](5, 'sg-1')
      })

      // Should rate using the media file ID
      expect(subsonic.setRating).toHaveBeenCalledWith('sg-1', 5)

      // The row is a position in the playlist, so it cannot be refetched by id:
      // rating can drop the track out of a smart playlist and shift every row up
      expect(getOne).toHaveBeenCalledTimes(1)
      expect(getOne).toHaveBeenCalledWith('song', { id: 'sg-1' })
      expect(getOne).not.toHaveBeenCalledWith(
        'playlistTrack',
        expect.anything(),
      )
      expect(mockRefresh).toHaveBeenCalled()
    })

    it('reloads the list even when the song refresh fails', async () => {
      getOne.mockImplementation(() => Promise.reject(new Error('boom')))
      const record = {
        id: '5',
        mediaFileId: 'sg-10',
        playlistId: 'pl-123',
        rating: 1,
      }
      const { result } = renderHook(() => useRating('playlistTrack', record))
      await act(async () => {
        await result.current[0](3, 'sg-10')
      })

      expect(subsonic.setRating).toHaveBeenCalledWith('sg-10', 3)
      expect(mockRefresh).toHaveBeenCalled()
    })

    it('only refreshes original resource when no mediaFileId present', async () => {
      const record = { id: 'sg-1', rating: 4 }
      const { result } = renderHook(() => useRating('song', record))
      await act(async () => {
        await result.current[0](2, 'sg-1')
      })

      // Should only refresh the original resource (song), without reloading the list
      expect(getOne).toHaveBeenCalledTimes(1)
      expect(getOne).toHaveBeenCalledWith('song', { id: 'sg-1' })
      expect(mockRefresh).not.toHaveBeenCalled()
    })

    it('does not include playlist_id filter for non-playlist resources', async () => {
      const record = { id: 'sg-1', rating: 0 }
      const { result } = renderHook(() => useRating('song', record))
      await act(async () => {
        await result.current[0](5, 'sg-1')
      })

      // Should refresh without any filter
      expect(getOne).toHaveBeenCalledWith('song', { id: 'sg-1' })
    })
  })

  describe('component integration scenarios', () => {
    it('handles mediaFileId fallback correctly for playlist tracks', async () => {
      const record = {
        id: 'pt-1',
        mediaFileId: 'sg-1',
        playlistId: 'pl-1',
        rating: 0,
      }
      const { result } = renderHook(() => useRating('playlistTrack', record))

      // Simulate RatingField component behavior: uses mediaFileId || record.id
      const targetId = record.mediaFileId || record.id
      await act(async () => {
        await result.current[0](4, targetId)
      })

      expect(subsonic.setRating).toHaveBeenCalledWith('sg-1', 4)
    })

    it('handles regular song rating without mediaFileId', async () => {
      const record = { id: 'sg-1', rating: 2 }
      const { result } = renderHook(() => useRating('song', record))

      // Simulate RatingField component behavior: uses mediaFileId || record.id
      const targetId = record.mediaFileId || record.id
      await act(async () => {
        await result.current[0](5, targetId)
      })

      expect(subsonic.setRating).toHaveBeenCalledWith('sg-1', 5)
      expect(getOne).toHaveBeenCalledTimes(1)
      expect(getOne).toHaveBeenCalledWith('song', { id: 'sg-1' })
    })
  })
})
