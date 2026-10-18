import { act, waitFor } from '@testing-library/react'
import { renderHook } from '@testing-library/react-hooks'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import subsonic from '../subsonic'
import { downloadLyrics } from './lyricsExport'
import { clearEnhancedLyricsCache } from './useEnhancedLyrics'
import usePlayerLyrics from './usePlayerLyrics'

vi.mock('../subsonic', () => ({
  default: {
    getLyricsBySongId: vi.fn(),
  },
}))

vi.mock('./lyricsExport', () => ({
  downloadLyrics: vi.fn(),
}))

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
    downloadLyrics.mockReset()
  })

  afterEach(() => {
    vi.useRealTimers()
    localStorage.clear()
    clearEnhancedLyricsCache()
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
    localStorage.setItem('lyrics.source:song-1', 'lrclib')
    subsonic.getLyricsBySongId.mockResolvedValue(variantsResponse())

    const { result } = renderPlayerLyrics()
    await showLyrics(result)

    expect(result.current.desktopLyricsProps.selectedSourceKey).toBe('lrclib')
    expect(result.current.desktopLyricsProps.mainLyric.line[0].value).toBe(
      'LRCLib line',
    )

    act(() => result.current.desktopLyricsProps.onSelectSource('netease'))
    expect(result.current.desktopLyricsProps.selectedSourceKey).toBe('netease')
    expect(localStorage.getItem('lyrics.source:song-1')).toBe('netease')

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
    localStorage.setItem('lyrics.offset:song-1:netease', '-350')
    subsonic.getLyricsBySongId.mockResolvedValue(variantsResponse())

    const { result } = renderPlayerLyrics()
    await showLyrics(result)

    expect(result.current.desktopLyricsProps.offsetMs).toBe(-350)
    expect(result.current.desktopLyricsProps.offsetEnabled).toBe(true)

    act(() => result.current.desktopLyricsProps.onOffsetChange(-100))
    expect(localStorage.getItem('lyrics.offset:song-1:netease')).toBe('-100')

    act(() => result.current.desktopLyricsProps.onSelectSource('lrclib'))
    await waitFor(() =>
      expect(result.current.desktopLyricsProps.offsetMs).toBe(0),
    )

    act(() => result.current.desktopLyricsProps.onOffsetChange(200))
    expect(localStorage.getItem('lyrics.offset:song-1:lrclib')).toBe('200')
    expect(localStorage.getItem('lyrics.offset:song-1:netease')).toBe('-100')
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

  it('stops discovery when lyrics are hidden and does not poll local lyrics', async () => {
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
    await act(async () => vi.advanceTimersByTime(15_000))
    expect(subsonic.getLyricsBySongId).toHaveBeenCalledTimes(1)
    act(() => result.current.toolbarLyricsProps.onToggleLyrics())
    await act(async () => vi.advanceTimersByTime(60_000))
    expect(subsonic.getLyricsBySongId).toHaveBeenCalledTimes(1)
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
    localStorage.setItem('lyrics.source:song-1', 'lrclib')
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
    expect(result.current.desktopLyricsProps.selectedSourceKey).toBe('lrclib')
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

  it('triggers a download for the selected variant with its format and display fields', async () => {
    subsonic.getLyricsBySongId.mockResolvedValue(variantsResponse())

    const { result } = renderPlayerLyrics()
    await showLyrics(result)

    act(() => result.current.desktopLyricsProps.onDownload())

    expect(downloadLyrics).toHaveBeenCalledWith(
      expect.objectContaining({ displayArtist: 'Artist' }),
      expect.objectContaining({
        format: 'elrc',
        artist: 'Artist',
        title: 'Title',
        label: 'NetEase',
      }),
    )
  })
})
