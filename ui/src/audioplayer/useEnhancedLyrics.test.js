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

const responseFor = (value, lang = 'en') => ({
  json: {
    'subsonic-response': {
      lyricsList: {
        structuredLyrics: [
          {
            kind: 'main',
            lang,
            synced: true,
            line: [{ start: 0, value }],
          },
        ],
      },
    },
  },
})

const createDeferred = () => {
  let resolve
  let reject
  const promise = new Promise((resolvePromise, rejectPromise) => {
    resolve = resolvePromise
    reject = rejectPromise
  })
  return { promise, resolve, reject }
}

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

  it('clears previous lyrics while loading an uncached track', async () => {
    const nextRequest = createDeferred()
    subsonic.getLyricsBySongId
      .mockResolvedValueOnce(responseFor('Track one'))
      .mockReturnValueOnce(nextRequest.promise)

    const { result, rerender } = renderHook(
      ({ trackId }) => useEnhancedLyrics(trackId),
      { initialProps: { trackId: 'song-1' } },
    )

    await waitFor(() =>
      expect(result.current.layers.main?.line[0].value).toBe('Track one'),
    )

    rerender({ trackId: 'song-2' })

    await waitFor(() => expect(result.current.loading).toBe(true))
    expect(result.current.layers).toBe(emptyLyricLayers)

    nextRequest.resolve(responseFor('Track two'))
    await waitFor(() =>
      expect(result.current.layers.main?.line[0].value).toBe('Track two'),
    )
  })

  it('stays empty when disabled', () => {
    const { result } = renderHook(() => useEnhancedLyrics('song-1', true))

    expect(result.current.layers).toBe(emptyLyricLayers)
    expect(result.current.loading).toBe(false)
    expect(subsonic.getLyricsBySongId).not.toHaveBeenCalled()
  })

  it('resets layers and retries after a lyrics request error', async () => {
    const error = new Error('lyrics failed')
    subsonic.getLyricsBySongId
      .mockRejectedValueOnce(error)
      .mockResolvedValueOnce(responseFor('Recovered lyrics'))

    const { result, rerender } = renderHook(
      ({ trackId }) => useEnhancedLyrics(trackId),
      { initialProps: { trackId: 'song-error' } },
    )

    await waitFor(() => expect(result.current.error).toBe(error))
    expect(result.current.layers).toBe(emptyLyricLayers)
    expect(result.current.loading).toBe(false)

    rerender({ trackId: null })
    rerender({ trackId: 'song-error' })

    await waitFor(() =>
      expect(result.current.layers.main?.line[0].value).toBe(
        'Recovered lyrics',
      ),
    )
    expect(subsonic.getLyricsBySongId).toHaveBeenCalledTimes(2)
  })

  it('keeps cached lyrics separate by preferred language', async () => {
    subsonic.getLyricsBySongId
      .mockResolvedValueOnce(responseFor('English lyrics', 'en'))
      .mockResolvedValueOnce(responseFor('Japanese lyrics', 'ja'))

    const { result, rerender } = renderHook(
      ({ trackId }) => useEnhancedLyrics(trackId),
      { initialProps: { trackId: 'song-1' } },
    )

    await waitFor(() =>
      expect(result.current.layers.main?.line[0].value).toBe('English lyrics'),
    )

    localStorage.setItem('locale', 'ja')
    rerender({ trackId: 'song-1' })

    await waitFor(() =>
      expect(result.current.layers.main?.line[0].value).toBe('Japanese lyrics'),
    )
    expect(subsonic.getLyricsBySongId).toHaveBeenCalledTimes(2)
  })
})
