import { act, waitFor } from '@testing-library/react'
import { renderHook } from '@testing-library/react-hooks'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import subsonic from '../subsonic'
import useEnhancedLyrics, {
  clearEnhancedLyricsCache,
  emptyLyricLayers,
} from './useEnhancedLyrics'

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

const renderLyrics = (initialProps) =>
  renderHook((props) => useEnhancedLyrics(props), { initialProps })

const expectLyric = (result, value) =>
  waitFor(() => expect(result.current.layers.main?.line[0].value).toBe(value))

describe('useEnhancedLyrics', () => {
  beforeEach(() => {
    localStorage.setItem('locale', 'en')
    clearEnhancedLyricsCache()
    subsonic.getLyricsBySongId.mockReset()
  })

  afterEach(() => {
    localStorage.clear()
    clearEnhancedLyricsCache()
  })

  it('fetches enhanced structured lyrics and caches them by track id', async () => {
    subsonic.getLyricsBySongId
      .mockResolvedValueOnce(responseFor('Track one'))
      .mockResolvedValueOnce(responseFor('Track two'))

    const { result, rerender } = renderLyrics({ trackId: 'song-1' })

    await expectLyric(result, 'Track one')
    expect(subsonic.getLyricsBySongId).toHaveBeenCalledWith(
      'song-1',
      expect.objectContaining({ signal: expect.any(Object) }),
    )

    rerender({ trackId: 'song-2' })
    await expectLyric(result, 'Track two')

    rerender({ trackId: 'song-1' })
    await expectLyric(result, 'Track one')
    expect(subsonic.getLyricsBySongId).toHaveBeenCalledTimes(2)
  })

  it('clears previous lyrics while loading an uncached track', async () => {
    const nextRequest = createDeferred()
    subsonic.getLyricsBySongId
      .mockResolvedValueOnce(responseFor('Track one'))
      .mockReturnValueOnce(nextRequest.promise)

    const { result, rerender } = renderLyrics({ trackId: 'song-1' })

    await expectLyric(result, 'Track one')

    rerender({ trackId: 'song-2' })

    expect(result.current.loading).toBe(true)
    expect(result.current.layers).toBe(emptyLyricLayers)

    nextRequest.resolve(responseFor('Track two'))
    await expectLyric(result, 'Track two')
  })

  it('stays empty when disabled or not requested', () => {
    const { result } = renderLyrics({
      trackId: 'song-1',
      disabled: true,
      requested: false,
    })

    expect(result.current.layers).toBe(emptyLyricLayers)
    expect(result.current.loading).toBe(false)
    expect(subsonic.getLyricsBySongId).not.toHaveBeenCalled()
  })

  it('resets layers and retries the same lyrics identity after an error', async () => {
    const error = new Error('lyrics failed')
    subsonic.getLyricsBySongId
      .mockRejectedValueOnce(error)
      .mockResolvedValueOnce(responseFor('Recovered lyrics'))

    const { result } = renderLyrics({ trackId: 'song-error' })

    await waitFor(() => expect(result.current.error).toBe(error))
    expect(result.current.layers).toBe(emptyLyricLayers)
    expect(result.current.loading).toBe(false)

    act(() => {
      result.current.retry()
    })

    await expectLyric(result, 'Recovered lyrics')
    expect(subsonic.getLyricsBySongId).toHaveBeenCalledTimes(2)
  })

  it('keeps cached lyrics separate by preferred language', async () => {
    subsonic.getLyricsBySongId
      .mockResolvedValueOnce(responseFor('English lyrics', 'en'))
      .mockResolvedValueOnce(responseFor('Japanese lyrics', 'ja'))

    const { result, rerender } = renderLyrics({ trackId: 'song-1' })

    await expectLyric(result, 'English lyrics')

    localStorage.setItem('locale', 'ja')
    rerender({ trackId: 'song-1' })

    await expectLyric(result, 'Japanese lyrics')
    expect(subsonic.getLyricsBySongId).toHaveBeenCalledTimes(2)
  })

  it('invalidates cache entries when the track update identity changes', async () => {
    subsonic.getLyricsBySongId
      .mockResolvedValueOnce(responseFor('Old lyrics'))
      .mockResolvedValueOnce(responseFor('Updated lyrics'))

    const { result, rerender } = renderLyrics({
      trackId: 'song-1',
      updatedAt: 'old',
    })

    await expectLyric(result, 'Old lyrics')
    rerender({ trackId: 'song-1', updatedAt: 'new' })
    await expectLyric(result, 'Updated lyrics')
    expect(subsonic.getLyricsBySongId).toHaveBeenCalledTimes(2)
  })

  it('deduplicates concurrent consumers and aborts only after the last one leaves', () => {
    let signal
    subsonic.getLyricsBySongId.mockImplementation((_trackId, options) => {
      signal = options.signal
      return new Promise((_resolve, reject) => {
        signal.addEventListener('abort', () => {
          reject(new DOMException('Aborted', 'AbortError'))
        })
      })
    })

    const first = renderLyrics({ trackId: 'shared-song' })
    const second = renderLyrics({ trackId: 'shared-song' })

    expect(subsonic.getLyricsBySongId).toHaveBeenCalledTimes(1)
    first.unmount()
    expect(signal.aborted).toBe(false)
    second.unmount()
    expect(signal.aborted).toBe(true)
  })

  it('starts a fresh request after an abandoned request is aborted', () => {
    const signals = []
    subsonic.getLyricsBySongId.mockImplementation((_trackId, options) => {
      signals.push(options.signal)
      return new Promise((_resolve, reject) => {
        options.signal.addEventListener('abort', () => {
          reject(new DOMException('Aborted', 'AbortError'))
        })
      })
    })

    const first = renderLyrics({ trackId: 'retry-song' })
    first.unmount()
    expect(signals[0].aborted).toBe(true)

    const second = renderLyrics({ trackId: 'retry-song' })
    expect(subsonic.getLyricsBySongId).toHaveBeenCalledTimes(2)
    second.unmount()
  })
})
