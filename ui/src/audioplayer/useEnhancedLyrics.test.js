import { waitFor } from '@testing-library/react'
import { renderHook } from '@testing-library/react-hooks'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import subsonic from '../subsonic'
import useEnhancedLyrics, { emptyLyricLayers } from './useEnhancedLyrics'

vi.mock('../subsonic', () => ({
  default: {
    getLyricsBySongId: vi.fn(),
  },
}))

const responseFor = (value) => ({
  json: {
    'subsonic-response': {
      lyricsList: {
        structuredLyrics: [
          {
            kind: 'main',
            lang: 'en',
            synced: true,
            line: [{ start: 0, value }],
          },
        ],
      },
    },
  },
})

describe('useEnhancedLyrics', () => {
  beforeEach(() => {
    localStorage.setItem('locale', 'en')
    subsonic.getLyricsBySongId.mockReset()
  })

  afterEach(() => {
    localStorage.clear()
  })

  it('fetches enhanced structured lyrics and caches them by track id', async () => {
    subsonic.getLyricsBySongId
      .mockResolvedValueOnce(responseFor('Track one'))
      .mockResolvedValueOnce(responseFor('Track two'))

    const { result, rerender } = renderHook(
      ({ trackId }) => useEnhancedLyrics(trackId),
      { initialProps: { trackId: 'song-1' } },
    )

    await waitFor(() =>
      expect(result.current.layers.main?.line[0].value).toBe('Track one'),
    )
    expect(subsonic.getLyricsBySongId).toHaveBeenCalledWith('song-1')

    rerender({ trackId: 'song-2' })
    await waitFor(() =>
      expect(result.current.layers.main?.line[0].value).toBe('Track two'),
    )

    rerender({ trackId: 'song-1' })
    await waitFor(() =>
      expect(result.current.layers.main?.line[0].value).toBe('Track one'),
    )
    expect(subsonic.getLyricsBySongId).toHaveBeenCalledTimes(2)
  })

  it('stays empty when disabled', () => {
    const { result } = renderHook(() => useEnhancedLyrics('song-1', true))

    expect(result.current.layers).toBe(emptyLyricLayers)
    expect(result.current.loading).toBe(false)
    expect(subsonic.getLyricsBySongId).not.toHaveBeenCalled()
  })
})
