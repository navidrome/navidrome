import { act, waitFor } from '@testing-library/react'
import { renderHook } from '@testing-library/react-hooks'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import subsonic from '../subsonic'
import { downloadBlob, downloadOriginalLyrics } from './lyricsExport'
import { clearEnhancedLyricsCache } from './useEnhancedLyrics'
import usePlayerLyrics from './usePlayerLyrics'

vi.mock('../subsonic', () => ({
  default: {
    getLyricsBySongId: vi.fn(),
    getLyricsVariants: vi.fn(),
    getLyricsOriginal: vi.fn(),
  },
}))

vi.mock('./lyricsExport', async (importOriginal) => {
  const actual = await importOriginal()
  return {
    ...actual,
    downloadBlob: vi.fn(),
    downloadOriginalLyrics: vi.fn(),
  }
})

const translate = (key) => key

const sourceEntry = (provider, value) => ({
  kind: 'main',
  lang: 'en',
  synced: true,
  displayArtist: 'Artist',
  displayTitle: 'Title',
  line: [{ start: 0, value }],
  source: { type: 'plugin', name: 'Better Lyrics', provider, format: 'elrc' },
})

const variantsResponse = () => ({
  json: {
    'subsonic-response': {
      lyricsList: {
        structuredLyrics: [
          sourceEntry('NetEase', 'NetEase line'),
          sourceEntry('LRCLib', 'LRCLib line'),
        ],
      },
    },
  },
})

const selectedEntry = (
  provider,
  value,
  { selection = `selection-${provider}`, start = 1000, offset = 0 } = {},
) => ({
  ...sourceEntry(provider, value),
  offset,
  selection,
  line: [{ start, value }],
})

const selectedVariantsResponse = () => ({
  json: {
    'subsonic-response': {
      lyricsList: {
        structuredLyrics: [
          selectedEntry('NetEase', 'NetEase line', { offset: 350 }),
          selectedEntry('LRCLib', 'LRCLib line', { start: 2000 }),
        ],
      },
    },
  },
})

const listingResponse = (variants) => ({
  json: {
    'subsonic-response': {
      lyricsVariants: { variant: variants },
    },
  },
})

const deferred = () => {
  let resolve
  let reject
  const promise = new Promise((nextResolve, nextReject) => {
    resolve = nextResolve
    reject = nextReject
  })
  return { promise, resolve, reject }
}

const blobText = (blob) =>
  new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(reader.result)
    reader.onerror = () => reject(reader.error)
    reader.readAsText(blob)
  })

const renderPlayerLyrics = (props = {}) =>
  renderHook(() =>
    usePlayerLyrics({
      trackId: 'song-1',
      trackUpdatedAt: undefined,
      isRadio: false,
      audioInstance: null,
      isDesktop: true,
      translate,
      ...props,
    }),
  )

const showLyrics = async (result) => {
  act(() => result.current.toolbarLyricsProps.onToggleLyrics())
  await waitFor(() =>
    expect(result.current.desktopLyricsProps.mainLyric).not.toBeNull(),
  )
}

describe('usePlayerLyrics', () => {
  beforeEach(() => {
    localStorage.setItem('locale', 'en')
    clearEnhancedLyricsCache()
    subsonic.getLyricsBySongId.mockReset()
    subsonic.getLyricsVariants.mockReset()
    subsonic.getLyricsOriginal.mockReset()
    downloadBlob.mockReset()
    downloadOriginalLyrics.mockReset()
  })

  afterEach(() => {
    vi.useRealTimers()
    localStorage.clear()
    clearEnhancedLyricsCache()
  })

  it('restores an existing provider selection and migrates its timing offset', async () => {
    localStorage.setItem('lyrics.source:song-1', 'lrclib')
    localStorage.setItem('lyrics.offset:song-1:lrclib', '250')
    subsonic.getLyricsBySongId.mockResolvedValue(variantsResponse())
    const { result } = renderPlayerLyrics()
    await showLyrics(result)
    expect(result.current.desktopLyricsProps.mainLyric.line[0].value).toBe(
      'LRCLib line',
    )
    expect(result.current.desktopLyricsProps.offsetMs).toBe(250)
    expect(localStorage.getItem('lyrics.source:song-1')).toBe(
      'plugin:better%20lyrics:lrclib',
    )
    act(() => result.current.desktopLyricsProps.onOffsetChange(0))
    expect(result.current.desktopLyricsProps.offsetMs).toBe(0)
    expect(localStorage.getItem('lyrics.offset:song-1:lrclib')).toBeNull()
  })

  it('toggles the letter wave preference', async () => {
    subsonic.getLyricsBySongId.mockResolvedValue(variantsResponse())
    const { result } = renderPlayerLyrics()
    await showLyrics(result)

    expect(result.current.desktopLyricsProps.letterWave).toBe(false)
    act(() => result.current.desktopLyricsProps.onToggleLetterWave())
    expect(result.current.desktopLyricsProps.letterWave).toBe(true)
    act(() => result.current.desktopLyricsProps.onToggleLetterWave())
    expect(result.current.desktopLyricsProps.letterWave).toBe(false)
  })

  it('loads lyrics for a Subsonic queue entry without embedded lyrics', async () => {
    const current = {
      trackId: 'song-1',
      song: {
        id: 'song-1',
        title: 'Subsonic song',
        artist: 'Artist',
      },
    }
    subsonic.getLyricsBySongId.mockResolvedValue({
      json: {
        'subsonic-response': {
          lyricsList: {
            structuredLyrics: [
              {
                kind: 'main',
                lang: 'en',
                synced: true,
                line: [{ start: 0, value: 'Loaded by track ID' }],
              },
            ],
          },
        },
      },
    })

    const { result } = renderHook(() =>
      usePlayerLyrics({
        trackId: current.trackId,
        trackUpdatedAt: current.song.updatedAt,
        isRadio: false,
        audioInstance: null,
        isDesktop: true,
        translate,
      }),
    )

    expect(current.song).not.toHaveProperty('lyrics')
    expect(subsonic.getLyricsBySongId).not.toHaveBeenCalled()

    act(() => result.current.toolbarLyricsProps.onToggleLyrics())

    await waitFor(() =>
      expect(result.current.desktopLyricsProps.mainLyric?.line[0].value).toBe(
        'Loaded by track ID',
      ),
    )
    expect(subsonic.getLyricsBySongId).toHaveBeenCalledWith(
      'song-1',
      expect.objectContaining({ signal: expect.any(Object) }),
    )
    expect(result.current.toolbarLyricsProps.lyricsActive).toBe(true)
    expect(result.current.desktopLyricsProps.visible).toBe(true)
  })

  it('restores the saved source choice per track and persists changes', async () => {
    localStorage.setItem(
      'lyrics.source:song-1',
      'plugin:better%20lyrics:lrclib',
    )
    subsonic.getLyricsBySongId.mockResolvedValue(variantsResponse())

    const { result } = renderPlayerLyrics()
    await showLyrics(result)

    expect(result.current.desktopLyricsProps.selectedSourceKey).toBe(
      'plugin:better%20lyrics:lrclib',
    )
    expect(result.current.desktopLyricsProps.mainLyric.line[0].value).toBe(
      'LRCLib line',
    )

    act(() =>
      result.current.desktopLyricsProps.onSelectSource(
        'plugin:better%20lyrics:netease',
      ),
    )
    expect(result.current.desktopLyricsProps.selectedSourceKey).toBe(
      'plugin:better%20lyrics:netease',
    )
    expect(localStorage.getItem('lyrics.source:song-1')).toBe(
      'plugin:better%20lyrics:netease',
    )

    act(() => result.current.desktopLyricsProps.onSelectSource(null))
    expect(localStorage.getItem('lyrics.source:song-1')).toBeNull()
    await waitFor(() =>
      expect(result.current.desktopLyricsProps.mainLyric.line[0].value).toBe(
        'NetEase line',
      ),
    )
  })

  it('falls back to auto when the saved source is missing from the response', async () => {
    localStorage.setItem('lyrics.source:song-1', 'gone')
    subsonic.getLyricsBySongId.mockResolvedValue(variantsResponse())

    const { result } = renderPlayerLyrics()
    await showLyrics(result)

    expect(result.current.desktopLyricsProps.mainLyric.line[0].value).toBe(
      'NetEase line',
    )
    expect(result.current.desktopLyricsProps.selectedSourceKey).toBe('gone')
  })

  it('keys the offset by track and resolved source', async () => {
    localStorage.setItem(
      'lyrics.offset:song-1:plugin:better%20lyrics:netease',
      '-350',
    )
    subsonic.getLyricsBySongId.mockResolvedValue(variantsResponse())

    const { result } = renderPlayerLyrics()
    await showLyrics(result)

    expect(result.current.desktopLyricsProps.offsetMs).toBe(-350)
    expect(result.current.desktopLyricsProps.offsetEnabled).toBe(true)

    act(() => result.current.desktopLyricsProps.onOffsetChange(-100))
    expect(
      localStorage.getItem(
        'lyrics.offset:song-1:plugin:better%20lyrics:netease',
      ),
    ).toBe('-100')

    act(() =>
      result.current.desktopLyricsProps.onSelectSource(
        'plugin:better%20lyrics:lrclib',
      ),
    )
    await waitFor(() =>
      expect(result.current.desktopLyricsProps.offsetMs).toBe(0),
    )

    act(() => result.current.desktopLyricsProps.onOffsetChange(200))
    expect(
      localStorage.getItem(
        'lyrics.offset:song-1:plugin:better%20lyrics:lrclib',
      ),
    ).toBe('200')
    expect(
      localStorage.getItem(
        'lyrics.offset:song-1:plugin:better%20lyrics:netease',
      ),
    ).toBe('-100')
  })

  it('recovers a clean empty first pass without opening the source menu', async () => {
    vi.useFakeTimers()
    subsonic.getLyricsBySongId
      .mockResolvedValueOnce({
        json: { 'subsonic-response': { lyricsList: { structuredLyrics: [] } } },
      })
      .mockResolvedValue(variantsResponse())
    const { result } = renderPlayerLyrics()
    await act(async () => {
      result.current.toolbarLyricsProps.onToggleLyrics()
    })
    expect(result.current.desktopLyricsProps.mainLyric).toBeNull()
    expect(result.current.desktopLyricsProps.loading).toBe(true)

    await act(async () => vi.advanceTimersByTime(5_000))
    expect(result.current.desktopLyricsProps.mainLyric.line[0].value).toBe(
      'NetEase line',
    )
    expect(result.current.desktopLyricsProps.loading).toBe(false)
    expect(subsonic.getLyricsBySongId).toHaveBeenCalledTimes(2)
  })

  it('discovers alternatives once for local lyrics and stops when hidden', async () => {
    vi.useFakeTimers()
    const response = variantsResponse()
    response.json['subsonic-response'].lyricsList.structuredLyrics = [
      { ...sourceEntry('local', 'Local line'), source: { type: 'embedded' } },
    ]
    subsonic.getLyricsBySongId.mockResolvedValue(response)
    const { result } = renderPlayerLyrics()
    await act(async () => {
      result.current.toolbarLyricsProps.onToggleLyrics()
    })
    await act(async () => vi.advanceTimersByTime(5_000))
    await act(async () => vi.advanceTimersByTime(10_000))
    expect(subsonic.getLyricsBySongId).toHaveBeenCalledTimes(2)
    act(() => result.current.toolbarLyricsProps.onToggleLyrics())
    await act(async () => vi.advanceTimersByTime(60_000))
    expect(subsonic.getLyricsBySongId).toHaveBeenCalledTimes(2)
  })

  it('keeps local lyrics readable while a slow generic plugin is discovered', async () => {
    vi.useFakeTimers()
    const local = {
      ...sourceEntry('local', 'Local line'),
      source: { type: 'embedded' },
    }
    const responseFor = (entries) => ({
      json: {
        'subsonic-response': { lyricsList: { structuredLyrics: entries } },
      },
    })
    let finishDiscovery
    subsonic.getLyricsBySongId
      .mockResolvedValueOnce(responseFor([local]))
      .mockImplementationOnce(
        () =>
          new Promise((resolve) => {
            finishDiscovery = resolve
          }),
      )
      .mockResolvedValue(variantsResponse())
    const { result } = renderPlayerLyrics()
    await act(async () => {
      result.current.toolbarLyricsProps.onToggleLyrics()
    })
    await act(async () => vi.advanceTimersByTime(5_000))
    expect(subsonic.getLyricsBySongId).toHaveBeenLastCalledWith(
      'song-1',
      expect.objectContaining({ allSources: true }),
    )
    await act(async () => vi.advanceTimersByTime(10_000))
    expect(result.current.desktopLyricsProps.mainLyric.line[0].value).toBe(
      'Local line',
    )
    expect(result.current.desktopLyricsProps.loading).toBe(false)
    expect(result.current.desktopLyricsProps.refreshing).toBe(true)
    expect(subsonic.getLyricsBySongId).toHaveBeenCalledTimes(2)
    const plugin = {
      ...sourceEntry('Shared provider', 'Remote line'),
      source: {
        type: 'plugin',
        name: 'Independent plugin',
        provider: 'Shared provider',
        format: 'lrc',
      },
    }
    await act(async () => finishDiscovery(responseFor([local, plugin])))
    expect(result.current.desktopLyricsProps.variants).toHaveLength(2)
    expect(result.current.desktopLyricsProps.mainLyric.line[0].value).toBe(
      'Local line',
    )
    await act(async () => vi.advanceTimersByTime(5_000))
    expect(subsonic.getLyricsBySongId).toHaveBeenCalledTimes(3)
  })

  it('keeps discovering after a partial response with only local lyrics', async () => {
    vi.useFakeTimers()
    const initial = variantsResponse()
    initial.json['subsonic-response'].lyricsList.structuredLyrics = [
      { ...sourceEntry('local', 'Local words'), source: { type: 'embedded' } },
    ]
    const partial = {
      json: {
        'subsonic-response': {
          lyricsList: {
            ...initial.json['subsonic-response'].lyricsList,
            sourcesIncomplete: true,
          },
        },
      },
    }
    subsonic.getLyricsBySongId
      .mockResolvedValueOnce(initial)
      .mockResolvedValue(partial)
    const { result } = renderPlayerLyrics()
    await act(async () => {
      result.current.toolbarLyricsProps.onToggleLyrics()
    })
    await act(async () => vi.advanceTimersByTime(5_000))
    await act(async () => vi.advanceTimersByTime(5_000))
    expect(subsonic.getLyricsBySongId).toHaveBeenCalledTimes(3)
    expect(result.current.desktopLyricsProps.mainLyric.line[0].value).toBe(
      'Local words',
    )
  })

  it('does not bypass a server cooldown with discovery polling', async () => {
    vi.useFakeTimers()
    subsonic.getLyricsBySongId.mockRejectedValue(
      new Error('provider retry after 120s'),
    )
    const { result } = renderPlayerLyrics()
    await act(async () => {
      result.current.toolbarLyricsProps.onToggleLyrics()
    })
    await act(async () => vi.advanceTimersByTime(60_000))
    expect(subsonic.getLyricsBySongId).toHaveBeenCalledTimes(1)
    expect(result.current.desktopLyricsProps.error).not.toBeNull()
  })

  it('preserves a selected source as automatic discovery updates the winner', async () => {
    vi.useFakeTimers()
    localStorage.setItem(
      'lyrics.source:song-1',
      'plugin:better%20lyrics:lrclib',
    )
    subsonic.getLyricsBySongId.mockResolvedValue(variantsResponse())
    const { result } = renderPlayerLyrics()
    await act(async () => {
      result.current.toolbarLyricsProps.onToggleLyrics()
    })
    const updated = variantsResponse()
    updated.json['subsonic-response'].lyricsList.structuredLyrics.unshift(
      sourceEntry('Better Lyrics', 'New best line'),
    )
    subsonic.getLyricsBySongId.mockResolvedValue(updated)
    await act(async () => vi.advanceTimersByTime(5_000))
    expect(result.current.desktopLyricsProps.selectedSourceKey).toBe(
      'plugin:better%20lyrics:lrclib',
    )
    expect(result.current.desktopLyricsProps.mainLyric.line[0].value).toBe(
      'LRCLib line',
    )
  })

  it('slows an empty search after one minute and discovers delayed results', async () => {
    vi.useFakeTimers()
    subsonic.getLyricsBySongId.mockResolvedValue({
      json: { 'subsonic-response': { lyricsList: { structuredLyrics: [] } } },
    })
    const { result } = renderPlayerLyrics()
    await act(async () => {
      result.current.toolbarLyricsProps.onToggleLyrics()
    })
    for (let tick = 0; tick < 12; tick += 1) {
      await act(async () => vi.advanceTimersByTime(5_000))
    }
    expect(result.current.desktopLyricsProps.loading).toBe(false)
    expect(subsonic.getLyricsBySongId).toHaveBeenCalledTimes(13)
    subsonic.getLyricsBySongId.mockResolvedValue(variantsResponse())
    await act(async () => vi.advanceTimersByTime(25_000))
    expect(subsonic.getLyricsBySongId).toHaveBeenCalledTimes(13)
    await act(async () => vi.advanceTimersByTime(5_000))
    expect(subsonic.getLyricsBySongId).toHaveBeenCalledTimes(14)
    expect(result.current.desktopLyricsProps.mainLyric.line[0].value).toBe(
      'NetEase line',
    )
  })

  it('refreshes every five seconds, then slows while the picker is closed', async () => {
    vi.useFakeTimers()
    subsonic.getLyricsBySongId.mockResolvedValue(variantsResponse())

    const flushMicrotasks = () =>
      act(async () => {
        await Promise.resolve()
        await Promise.resolve()
      })

    const { result } = renderPlayerLyrics()
    act(() => result.current.toolbarLyricsProps.onToggleLyrics())
    await flushMicrotasks()
    expect(subsonic.getLyricsBySongId).toHaveBeenCalledTimes(1)

    act(() => result.current.desktopLyricsProps.onSourceMenuOpenChange(true))
    expect(subsonic.getLyricsBySongId).toHaveBeenCalledTimes(2)

    let expectedCalls = 2
    for (let step = 0; step < 12; step += 1) {
      await flushMicrotasks()
      await act(async () => {
        vi.advanceTimersByTime(5_000)
      })
      expectedCalls += 1
      expect(subsonic.getLyricsBySongId).toHaveBeenCalledTimes(expectedCalls)
    }

    act(() => result.current.desktopLyricsProps.onSourceMenuOpenChange(false))
    await flushMicrotasks()
    await act(async () => {
      vi.advanceTimersByTime(15_000)
    })
    expect(subsonic.getLyricsBySongId).toHaveBeenCalledTimes(14)
  })

  it('downloads a labeled converted variant with its display fields', async () => {
    subsonic.getLyricsBySongId.mockResolvedValue(selectedVariantsResponse())

    const { result } = renderPlayerLyrics()
    await showLyrics(result)

    await act(async () => {
      await result.current.desktopLyricsProps.onDownload('converted')
    })

    expect(downloadBlob).toHaveBeenCalledTimes(1)
    const [blob, filename] = downloadBlob.mock.calls[0]
    expect(filename).toBe('Artist - Title (NetEase converted).lrc')
    expect(blob.type).toBe('text/plain; charset=utf-8')
    expect(await blobText(blob)).toBe('[00:01.35]NetEase line')
  })

  it('binds adjusted output to the selected variant and personal correction', async () => {
    subsonic.getLyricsBySongId.mockResolvedValue(selectedVariantsResponse())

    const { result } = renderPlayerLyrics()
    await showLyrics(result)

    await act(async () => {
      result.current.desktopLyricsProps.onOffsetChange(500)
    })
    await act(async () => {
      await result.current.desktopLyricsProps.onDownload('converted')
    })
    expect(await blobText(downloadBlob.mock.calls[0][0])).toBe(
      '[00:01.35]NetEase line',
    )

    await act(async () => {
      await result.current.desktopLyricsProps.onDownload('adjusted')
    })
    expect(await blobText(downloadBlob.mock.calls[1][0])).toBe(
      '[00:01.85]NetEase line',
    )

    await act(async () => {
      result.current.desktopLyricsProps.onSelectSource(
        result.current.desktopLyricsProps.variants[1].key,
      )
    })
    await act(async () => {
      await result.current.desktopLyricsProps.onDownload('adjusted')
    })
    // LRCLib has no source offset; it must not inherit NetEase's +350/+500.
    expect(await blobText(downloadBlob.mock.calls[2][0])).toBe(
      '[00:02.00]LRCLib line',
    )
  })

  it('downloads an available original without passing it through a serializer', async () => {
    subsonic.getLyricsBySongId.mockResolvedValue(selectedVariantsResponse())
    subsonic.getLyricsVariants.mockResolvedValue(
      listingResponse([
        {
          selection: 'selection-NetEase',
          original: {
            available: true,
            format: 'qrc',
            extension: '.qrc',
          },
        },
      ]),
    )
    const original = {
      blob: new Blob(['exact qrc'], { type: 'application/xml' }),
      filename: 'Artist - Title.qrc',
      mediaType: 'application/xml',
    }
    subsonic.getLyricsOriginal.mockResolvedValue(original)

    const { result } = renderPlayerLyrics()
    await showLyrics(result)
    await act(async () => {
      result.current.desktopLyricsProps.onOffsetChange(500)
      result.current.desktopLyricsProps.onDownloadMenuOpenChange(true)
    })
    await waitFor(() =>
      expect(result.current.desktopLyricsProps.originalInfo?.status).toBe(
        'available',
      ),
    )

    await act(async () => {
      await result.current.desktopLyricsProps.onDownload('original')
    })

    expect(subsonic.getLyricsOriginal).toHaveBeenCalledWith(
      'song-1',
      'selection-NetEase',
      expect.objectContaining({ signal: expect.any(AbortSignal) }),
    )
    expect(downloadOriginalLyrics).toHaveBeenCalledWith(original, {
      artist: 'Artist',
      title: 'Title',
      extension: '.qrc',
    })
  })

  it('does not publish an original after an A to B to A target change', async () => {
    subsonic.getLyricsBySongId.mockResolvedValue(selectedVariantsResponse())
    const pending = deferred()
    subsonic.getLyricsOriginal.mockReturnValue(pending.promise)

    const { result } = renderPlayerLyrics()
    await showLyrics(result)
    let firstDownload
    act(() => {
      firstDownload = result.current.desktopLyricsProps.onDownload('original')
    })
    await waitFor(() => expect(subsonic.getLyricsOriginal).toHaveBeenCalled())

    const secondKey = result.current.desktopLyricsProps.variants[1].key
    await act(async () => {
      result.current.desktopLyricsProps.onSelectSource(secondKey)
    })
    await act(async () => {
      result.current.desktopLyricsProps.onSelectSource(
        result.current.desktopLyricsProps.variants[0].key,
      )
    })

    await act(async () => {
      pending.resolve({
        blob: new Blob(['exact original']),
        filename: 'Artist - Title.qrc',
        mediaType: 'application/xml',
      })
      await firstDownload
    })
    expect(downloadOriginalLyrics).not.toHaveBeenCalled()
  })

  it('ignores a stale listing after the selected variant changes', async () => {
    subsonic.getLyricsBySongId.mockResolvedValue(selectedVariantsResponse())
    const listingA = deferred()
    const listingB = deferred()
    subsonic.getLyricsVariants
      .mockReturnValueOnce(listingA.promise)
      .mockImplementation(() => listingB.promise)

    const { result } = renderPlayerLyrics()
    await showLyrics(result)
    act(() => result.current.desktopLyricsProps.onDownloadMenuOpenChange(true))
    await waitFor(() =>
      expect(subsonic.getLyricsVariants).toHaveBeenCalledTimes(1),
    )
    expect(result.current.desktopLyricsProps.originalInfo?.status).toBe(
      'loading',
    )

    const secondKey = result.current.desktopLyricsProps.variants[1].key
    await act(async () => {
      result.current.desktopLyricsProps.onSelectSource(secondKey)
    })
    await act(async () => {
      result.current.desktopLyricsProps.onDownloadMenuOpenChange(true)
    })
    await waitFor(() =>
      expect(
        subsonic.getLyricsVariants.mock.calls.length,
      ).toBeGreaterThanOrEqual(2),
    )

    await act(async () => {
      listingA.resolve(
        listingResponse([
          {
            selection: 'selection-NetEase',
            original: { available: true, format: 'qrc', extension: '.qrc' },
          },
        ]),
      )
      await listingA.promise
    })
    expect(result.current.desktopLyricsProps.originalInfo).not.toEqual(
      expect.objectContaining({ selection: 'selection-NetEase' }),
    )

    await act(async () => {
      listingB.resolve(
        listingResponse([
          {
            selection: 'selection-LRCLib',
            original: {
              available: false,
              reason: 'source supplies no native document',
            },
          },
        ]),
      )
      await listingB.promise
    })
    expect(result.current.desktopLyricsProps.originalInfo?.status).toBe(
      'unavailable',
    )
    expect(result.current.desktopLyricsProps.originalInfo?.selection).toBe(
      'selection-LRCLib',
    )
  })

  it('does not write or notify after cancellation or unmount', async () => {
    subsonic.getLyricsBySongId.mockResolvedValue(selectedVariantsResponse())
    const pending = deferred()
    subsonic.getLyricsOriginal.mockReturnValue(pending.promise)
    const onExportError = vi.fn()

    const { result, unmount } = renderPlayerLyrics({ onExportError })
    await showLyrics(result)
    let download
    act(() => {
      download = result.current.desktopLyricsProps.onDownload('original')
    })
    await waitFor(() => expect(subsonic.getLyricsOriginal).toHaveBeenCalled())
    unmount()

    await act(async () => {
      pending.reject(new Error('late diagnostic'))
      await download.catch(() => {})
    })
    expect(onExportError).not.toHaveBeenCalled()
    expect(downloadOriginalLyrics).not.toHaveBeenCalled()
  })

  it('does not fall back to converted bytes when original retrieval fails', async () => {
    subsonic.getLyricsBySongId.mockResolvedValue(selectedVariantsResponse())
    subsonic.getLyricsOriginal.mockRejectedValue(new Error('data not found'))
    const onExportError = vi.fn()

    const { result } = renderPlayerLyrics({ onExportError })
    await showLyrics(result)
    await act(async () => {
      await result.current.desktopLyricsProps.onDownload('original')
    })

    expect(downloadOriginalLyrics).not.toHaveBeenCalled()
    expect(downloadBlob).not.toHaveBeenCalled()
    expect(onExportError).toHaveBeenCalledWith('player.downloadLyricsErrorText')
    expect(result.current.desktopLyricsProps.originalInfo).toEqual(
      expect.objectContaining({ status: 'unavailable' }),
    )
  })

  it('keeps conversions working for a legacy response without a selection', async () => {
    subsonic.getLyricsBySongId.mockResolvedValue(variantsResponse())
    const onExportError = vi.fn()

    const { result } = renderPlayerLyrics({ onExportError })
    await showLyrics(result)
    await act(async () => {
      result.current.desktopLyricsProps.onDownloadMenuOpenChange(true)
    })
    await waitFor(() =>
      expect(result.current.desktopLyricsProps.originalInfo?.status).toBe(
        'unavailable',
      ),
    )
    await act(async () => {
      await result.current.desktopLyricsProps.onDownload('converted')
    })

    expect(subsonic.getLyricsVariants).not.toHaveBeenCalled()
    expect(downloadBlob).toHaveBeenCalledTimes(1)
    await act(async () => {
      await result.current.desktopLyricsProps.onDownload('original')
    })
    expect(subsonic.getLyricsOriginal).not.toHaveBeenCalled()
    expect(onExportError).toHaveBeenCalledWith(
      'player.downloadLyricsUnavailableLegacyText',
    )
  })
})

describe('evidence-only lyrics state', () => {
  it('does not remain loading and forwards typed evidence', async () => {
    subsonic.getLyricsBySongId.mockResolvedValue({
      json: {
        'subsonic-response': {
          lyricsList: {
            instrumentalEvidence: [
              { kind: 'instrumental', provider: 'LRCLib' },
            ],
          },
        },
      },
    })
    const { result } = renderPlayerLyrics()
    act(() => result.current.toolbarLyricsProps.onToggleLyrics())
    await waitFor(() =>
      expect(result.current.desktopLyricsProps.evidence).toHaveLength(1),
    )
    expect(result.current.desktopLyricsProps.loading).toBe(false)
    expect(result.current.desktopLyricsProps.mainLyric).toBeNull()
  })
})
