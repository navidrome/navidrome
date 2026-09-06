import { renderHook, act } from '@testing-library/react-hooks'
import { vi, describe, it, expect, beforeEach } from 'vitest'
import { useToggleLove } from './useToggleLove'
import subsonic from '../subsonic'
import { useDataProvider } from 'react-admin'

const mockRefresh = vi.fn()

vi.mock('../subsonic', () => ({
  default: {
    star: vi.fn(() => Promise.resolve()),
    unstar: vi.fn(() => Promise.resolve()),
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

describe('useToggleLove', () => {
  let getOne
  beforeEach(() => {
    getOne = vi.fn(() => Promise.resolve())
    useDataProvider.mockReturnValue({ getOne })
    vi.clearAllMocks()
  })

  it('uses mediaFileId when present', async () => {
    const record = { id: 'pt-1', mediaFileId: 'sg-1', starred: false }
    const { result } = renderHook(() => useToggleLove('song', record))
    await act(async () => {
      await result.current[0]()
    })
    expect(subsonic.star).toHaveBeenCalledWith('sg-1')
    expect(getOne).toHaveBeenCalledWith('song', { id: 'sg-1' })
  })

  it('falls back to id when mediaFileId not present', async () => {
    const record = { id: 'sg-1', starred: false }
    const { result } = renderHook(() => useToggleLove('song', record))
    await act(async () => {
      await result.current[0]()
    })
    expect(subsonic.star).toHaveBeenCalledWith('sg-1')
    expect(getOne).toHaveBeenCalledWith('song', { id: 'sg-1' })
  })

  it('calls unstar when record is already loved', async () => {
    const record = { id: 'sg-1', starred: true }
    const { result } = renderHook(() => useToggleLove('song', record))
    await act(async () => {
      await result.current[0]()
    })
    expect(subsonic.unstar).toHaveBeenCalledWith('sg-1')
  })

  describe('playlist track scenarios', () => {
    it('refreshes the song and reloads the list for playlist tracks', async () => {
      const record = {
        id: '1',
        mediaFileId: 'sg-1',
        playlistId: 'pl-1',
        starred: false,
      }
      const { result } = renderHook(() =>
        useToggleLove('playlistTrack', record),
      )
      await act(async () => {
        await result.current[0]()
      })

      // Should star using the media file ID
      expect(subsonic.star).toHaveBeenCalledWith('sg-1')

      // The row is a position in the playlist, so it cannot be refetched by id:
      // loving can drop the track out of a smart playlist and shift every row up
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
        starred: true,
      }
      const { result } = renderHook(() =>
        useToggleLove('playlistTrack', record),
      )
      await act(async () => {
        await result.current[0]()
      })

      expect(subsonic.unstar).toHaveBeenCalledWith('sg-10')
      expect(mockRefresh).toHaveBeenCalled()
    })

    it('only refreshes original resource when no mediaFileId present', async () => {
      const record = { id: 'sg-1', starred: false }
      const { result } = renderHook(() => useToggleLove('song', record))
      await act(async () => {
        await result.current[0]()
      })

      // Should only refresh the original resource (song), without reloading the list
      expect(getOne).toHaveBeenCalledTimes(1)
      expect(getOne).toHaveBeenCalledWith('song', { id: 'sg-1' })
      expect(mockRefresh).not.toHaveBeenCalled()
    })

    it('does not include playlist_id filter for non-playlist resources', async () => {
      const record = { id: 'sg-1', starred: false }
      const { result } = renderHook(() => useToggleLove('song', record))
      await act(async () => {
        await result.current[0]()
      })

      // Should refresh without any filter
      expect(getOne).toHaveBeenCalledWith('song', { id: 'sg-1' })
    })
  })
})
