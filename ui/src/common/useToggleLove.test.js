import { renderHook, act } from '@testing-library/react-hooks'
import { vi, describe, it, expect, beforeEach } from 'vitest'
import { useToggleLove } from './useToggleLove'
import subsonic from '../subsonic'
import { useDataProvider, useRefresh } from 'react-admin'

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
    useRefresh: vi.fn(),
    useNotify: vi.fn(() => vi.fn()),
  }
})

describe('useToggleLove', () => {
  let getOne
  let refresh
  beforeEach(() => {
    getOne = vi.fn(() => Promise.resolve())
    refresh = vi.fn()
    useDataProvider.mockReturnValue({ getOne })
    useRefresh.mockReturnValue(refresh)
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
    it('refreshes the list and song without refetching the positional playlist track', async () => {
      const record = {
        id: 'pt-1',
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

      expect(getOne).toHaveBeenCalledTimes(1)
      expect(getOne).toHaveBeenCalledWith('song', { id: 'sg-1' })
      expect(refresh).toHaveBeenCalledTimes(1)
    })

    it('refreshes the list after loving a playlist track', async () => {
      const record = {
        id: 'pt-5',
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

      // Should unstar using the media file ID
      expect(subsonic.unstar).toHaveBeenCalledWith('sg-10')

      expect(getOne).toHaveBeenCalledWith('song', { id: 'sg-10' })
      expect(refresh).toHaveBeenCalledTimes(1)
    })

    it('only refreshes original resource when no mediaFileId present', async () => {
      const record = { id: 'sg-1', starred: false }
      const { result } = renderHook(() => useToggleLove('song', record))
      await act(async () => {
        await result.current[0]()
      })

      // Should only refresh the original resource (song)
      expect(getOne).toHaveBeenCalledTimes(1)
      expect(getOne).toHaveBeenCalledWith('song', { id: 'sg-1' })
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
