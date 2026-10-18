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

const responseFor = (value, lang = 'en', source) => ({
  json: {
    'subsonic-response': {
      lyricsList: {
        structuredLyrics: [
          {
            kind: 'main',
            lang,
            synced: true,
            line: [{ start: 0, value }],
            ...(source ? { source } : {}),
          },
        ],
      },
    },
  },
})

const responseForEntries = (entries) => ({
  json: {
    'subsonic-response': {
      lyricsList: { structuredLyrics: entries },
    },
  },
})

const sourceEntry = (provider, value) => ({
  kind: 'main',
  lang: 'en',
  synced: true,
  line: [{ start: 0, value }],
  source: { type: 'plugin', provider, format: 'lrc' },
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
    vi.useRealTimers()
    localStorage.clear()
    clearEnhancedLyricsCache()
  })

  it('keeps readable lyrics when a background refresh returns an empty result', async () => {
    subsonic.getLyricsBySongId
      .mockResolvedValueOnce(responseFor('Readable lyrics'))
      .mockResolvedValueOnce(responseForEntries([]))
    const { result, rerender } = renderLyrics({ trackId: 'song-empty-refresh' })
    await expectLyric(result, 'Readable lyrics')
    act(() => result.current.refresh())
    await waitFor(() => expect(result.current.refreshing).toBe(false))
    expect(result.current.layers.main.line[0].value).toBe('Readable lyrics')
    rerender({ trackId: 'song-empty-refresh', requested: false })
    rerender({ trackId: 'song-empty-refresh', requested: true })
    expect(result.current.layers.main.line[0].value).toBe('Readable lyrics')
    expect(subsonic.getLyricsBySongId).toHaveBeenCalledTimes(2)
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

  it('preserves source provenance on the selected lyric layer', async () => {
    const source = {
      type: 'plugin',
      name: 'Better Lyrics',
      provider: 'kugou',
      format: 'lrc',
    }
    subsonic.getLyricsBySongId.mockResolvedValueOnce(
      responseFor('Sourced lyrics', 'en', source),
    )

    const { result } = renderLyrics({ trackId: 'song-source' })

    await expectLyric(result, 'Sourced lyrics')
    expect(result.current.layers.main.source).toEqual(source)
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

  it('reports a cooldown countdown and retries the current song when it expires', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-08-23T06:00:00Z'))
    const cooldownError = new Error('better lyrics API cooldown active for 2s')
    subsonic.getLyricsBySongId
      .mockRejectedValueOnce(cooldownError)
      .mockResolvedValueOnce(responseFor('Recovered after cooldown'))

    const { result } = renderLyrics({ trackId: 'rate-limited-song' })
    await act(async () => {
      await Promise.resolve()
    })

    expect(result.current.error).toBe(cooldownError)
    expect(result.current.retryAfterSeconds).toBe(2)

    await act(async () => {
      vi.advanceTimersByTime(2_000)
      await Promise.resolve()
    })

    expect(subsonic.getLyricsBySongId).toHaveBeenCalledTimes(2)
    expect(result.current.layers.main?.line[0].value).toBe(
      'Recovered after cooldown',
    )
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

  it('exposes variants in response order with auto selection', async () => {
    subsonic.getLyricsBySongId.mockResolvedValueOnce(
      responseForEntries([
        sourceEntry('NetEase', 'NetEase line'),
        sourceEntry('LRCLib', 'LRCLib line'),
      ]),
    )

    const { result } = renderLyrics({ trackId: 'song-variants' })

    await expectLyric(result, 'NetEase line')
    expect(result.current.variants.map((variant) => variant.key)).toEqual([
      'netease',
      'lrclib',
    ])
  })

  it('re-derives layers for a selected source without refetching', async () => {
    subsonic.getLyricsBySongId.mockResolvedValueOnce(
      responseForEntries([
        sourceEntry('NetEase', 'NetEase line'),
        sourceEntry('LRCLib', 'LRCLib line'),
      ]),
    )

    const { result, rerender } = renderLyrics({ trackId: 'song-select' })

    await expectLyric(result, 'NetEase line')

    rerender({ trackId: 'song-select', selectedSourceKey: 'lrclib' })
    await expectLyric(result, 'LRCLib line')

    expect(subsonic.getLyricsBySongId).toHaveBeenCalledTimes(1)
  })

  it('falls back to auto when the selected source disappears', async () => {
    subsonic.getLyricsBySongId
      .mockResolvedValueOnce(
        responseForEntries([
          sourceEntry('NetEase', 'NetEase line'),
          sourceEntry('LRCLib', 'LRCLib line'),
        ]),
      )
      .mockResolvedValueOnce(
        responseForEntries([sourceEntry('NetEase', 'NetEase line')]),
      )

    const { result } = renderLyrics({
      trackId: 'song-fallback',
      selectedSourceKey: 'lrclib',
    })

    await expectLyric(result, 'LRCLib line')

    act(() => {
      result.current.refresh()
    })

    await expectLyric(result, 'NetEase line')
    expect(result.current.variants.map((variant) => variant.key)).toEqual([
      'netease',
    ])
    expect(result.current.error).toBeNull()
  })

  it('refreshes variants without blanking the current lyrics', async () => {
    const nextRequest = createDeferred()
    subsonic.getLyricsBySongId
      .mockResolvedValueOnce(
        responseForEntries([sourceEntry('NetEase', 'NetEase line')]),
      )
      .mockReturnValueOnce(nextRequest.promise)

    const { result } = renderLyrics({ trackId: 'song-refresh' })

    await expectLyric(result, 'NetEase line')

    act(() => {
      result.current.refresh()
    })

    await waitFor(() => expect(result.current.refreshing).toBe(true))
    expect(result.current.layers.main.line[0].value).toBe('NetEase line')

    act(() => {
      nextRequest.resolve(
        responseForEntries([
          sourceEntry('NetEase', 'NetEase line'),
          sourceEntry('LRCLib', 'LRCLib line'),
        ]),
      )
    })

    await waitFor(() => expect(result.current.variants).toHaveLength(2))
    expect(result.current.refreshing).toBe(false)
    expect(result.current.layers.main.line[0].value).toBe('NetEase line')
    expect(subsonic.getLyricsBySongId).toHaveBeenCalledTimes(2)
  })

  it('coalesces a refresh with a request that is still in flight', async () => {
    const pending = createDeferred()
    subsonic.getLyricsBySongId.mockReturnValueOnce(pending.promise)

    const { result } = renderLyrics({ trackId: 'song-coalesce' })

    expect(subsonic.getLyricsBySongId).toHaveBeenCalledTimes(1)

    act(() => {
      result.current.refresh()
    })
    expect(subsonic.getLyricsBySongId).toHaveBeenCalledTimes(1)

    act(() => {
      pending.resolve(
        responseForEntries([sourceEntry('NetEase', 'NetEase line')]),
      )
    })

    await expectLyric(result, 'NetEase line')
    expect(subsonic.getLyricsBySongId).toHaveBeenCalledTimes(1)
  })

  it('keeps lyric references stable when a refresh returns the same payload', async () => {
    subsonic.getLyricsBySongId.mockImplementation(async () =>
      responseForEntries([
        sourceEntry('NetEase', 'NetEase line'),
        sourceEntry('LRCLib', 'LRCLib line'),
      ]),
    )

    const { result } = renderLyrics({ trackId: 'song-stable' })

    await expectLyric(result, 'NetEase line')
    const layers = result.current.layers
    const variants = result.current.variants

    act(() => {
      result.current.refresh()
    })
    await waitFor(() => expect(result.current.refreshing).toBe(false))

    expect(subsonic.getLyricsBySongId).toHaveBeenCalledTimes(2)
    expect(result.current.layers).toBe(layers)
    expect(result.current.variants).toBe(variants)
  })

  it('auto-retries a failed request with a bounded backoff', async () => {
    vi.useFakeTimers()
    subsonic.getLyricsBySongId
      .mockRejectedValueOnce(
        new Error('plugin call failed: BiniLyrics search returned HTTP 502'),
      )
      .mockResolvedValueOnce(
        responseForEntries([sourceEntry('NetEase', 'Recovered line')]),
      )

    const { result } = renderLyrics({ trackId: 'song-auto-retry' })
    await act(async () => {
      await Promise.resolve()
      await Promise.resolve()
    })

    expect(result.current.error).not.toBeNull()
    expect(subsonic.getLyricsBySongId).toHaveBeenCalledTimes(1)

    await act(async () => {
      vi.advanceTimersByTime(5_000)
    })
    await act(async () => {
      await Promise.resolve()
      await Promise.resolve()
    })

    expect(subsonic.getLyricsBySongId).toHaveBeenCalledTimes(2)
    expect(result.current.error).toBeNull()
    expect(result.current.layers.main.line[0].value).toBe('Recovered line')
  })

  it('stops auto-retrying after the backoff schedule is exhausted', async () => {
    vi.useFakeTimers()
    subsonic.getLyricsBySongId.mockRejectedValue(new Error('boom'))

    const { result } = renderLyrics({ trackId: 'song-auto-retry-capped' })
    await act(async () => {
      await Promise.resolve()
      await Promise.resolve()
    })
    expect(result.current.error).not.toBeNull()

    for (const delay of [5_000, 15_000, 45_000, 90_000, 180_000]) {
      await act(async () => {
        vi.advanceTimersByTime(delay)
      })
      await act(async () => {
        await Promise.resolve()
        await Promise.resolve()
      })
    }
    expect(subsonic.getLyricsBySongId).toHaveBeenCalledTimes(6)

    await act(async () => {
      vi.advanceTimersByTime(600_000)
    })
    expect(subsonic.getLyricsBySongId).toHaveBeenCalledTimes(6)
  })

  it('keeps current lyrics when a refresh fails', async () => {
    subsonic.getLyricsBySongId
      .mockResolvedValueOnce(
        responseForEntries([sourceEntry('NetEase', 'NetEase line')]),
      )
      .mockRejectedValueOnce(new Error('refresh failed'))

    const { result } = renderLyrics({ trackId: 'song-refresh-fail' })

    await expectLyric(result, 'NetEase line')

    act(() => {
      result.current.refresh()
    })

    await waitFor(() => expect(result.current.refreshing).toBe(false))
    expect(result.current.error).toBeNull()
    expect(result.current.layers.main.line[0].value).toBe('NetEase line')
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
