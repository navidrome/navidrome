import { act, waitFor } from '@testing-library/react'
import { renderHook } from '@testing-library/react-hooks'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import subsonic from '../subsonic'
import { clearEnhancedLyricsCache } from './useEnhancedLyrics'
import usePlayerLyrics from './usePlayerLyrics'

vi.mock('../subsonic', () => ({
  default: {
    getLyricsBySongId: vi.fn(),
    getLyricsVariants: vi.fn(),
    getLyricsOriginal: vi.fn(),
    updateLyricsPreference: vi.fn(),
  },
}))

const translate = (key, options = {}) =>
  key.replace(/%\{(\w+)\}/g, (match, name) =>
    options[name] == null ? '' : String(options[name]),
  )

const identity = (recording, overrides = {}) => ({
  version: 1,
  instance: 'plugin-instance',
  recording,
  language: 'en',
  role: 'main',
  format: 'lrc',
  ...overrides,
})

const providerEntry = (
  provider,
  value,
  selection,
  variant = identity('r-a@0123456789abcdef0123456789abcdef'),
) => ({
  kind: 'main',
  lang: 'en',
  synced: true,
  line: [{ start: 1000, value }],
  source: {
    type: 'plugin',
    name: 'Better Lyrics',
    pluginId: 'plugin-id',
    provider,
    format: variant.format,
  },
  selection,
  variant,
})

const localEntry = (value, local) => ({
  kind: 'main',
  lang: 'en',
  synced: true,
  line: [{ start: 1000, value }],
  source: { type: local.kind, format: local.format },
  local,
})

const providerSelection = (
  token,
  variant = identity('r-a@0123456789abcdef0123456789abcdef'),
  status = 'pinned',
) => ({
  kind: 'provider',
  plugin: 'plugin-id',
  provider: 'LRCLib',
  identity: variant,
  selection: token,
  status,
})

const localSelection = (local, status = 'pinned') => ({
  kind: 'local',
  local,
  status,
})

const lyricsResponse = (structuredLyrics, personal = null) => ({
  json: {
    'subsonic-response': {
      lyricsList: {
        structuredLyrics,
        ...(personal ? { personal } : {}),
      },
    },
  },
})

const mutationResponse = (personal) => ({
  json: { 'subsonic-response': { lyricsPersonal: personal } },
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

const local = {
  version: 1,
  token: 'local:v1:0123456789abcdef0123456789abcdef',
  kind: 'sidecar',
  format: 'lrc',
  language: 'en',
  role: 'main',
}

const embedded = {
  version: 1,
  token: 'local:v1:fedcba9876543210fedcba9876543210',
  kind: 'embedded',
  format: '',
  language: 'en',
  role: 'main',
}

const renderPlayerLyrics = (props = {}) =>
  renderHook(
    (nextProps) =>
      usePlayerLyrics({
        authIdentity: 'user-a',
        trackId: 'song-1',
        trackUpdatedAt: undefined,
        isRadio: false,
        audioInstance: null,
        isDesktop: true,
        translate,
        ...nextProps,
      }),
    { initialProps: props },
  )

const showLyrics = async (result) => {
  act(() => result.current.toolbarLyricsProps.onToggleLyrics())
  await waitFor(() =>
    expect(result.current.desktopLyricsProps.mainLyric).not.toBeNull(),
  )
}

describe('usePlayerLyrics personal corrections', () => {
  beforeEach(() => {
    localStorage.setItem('locale', 'en')
    clearEnhancedLyricsCache()
    subsonic.getLyricsBySongId.mockReset()
    subsonic.getLyricsVariants.mockReset()
    subsonic.getLyricsOriginal.mockReset()
    subsonic.updateLyricsPreference.mockReset()
    subsonic.updateLyricsPreference.mockImplementation((_id, payload) =>
      Promise.resolve(mutationResponse({ revision: 2, ...payload })),
    )
  })

  afterEach(() => {
    vi.useRealTimers()
    localStorage.clear()
    clearEnhancedLyricsCache()
  })

  it('pins a provider variant over the automatic order', async () => {
    const netease = providerEntry(
      'NetEase',
      'NetEase line',
      'token-netease',
      identity('r-n@0123456789abcdef0123456789abcdef', { format: 'lrc' }),
    )
    const lrclib = providerEntry(
      'LRCLib',
      'LRCLib line',
      'token-lrclib',
      identity('r-l@0123456789abcdef0123456789abcdef', { format: 'elrc' }),
    )
    subsonic.getLyricsBySongId.mockResolvedValue(
      lyricsResponse([netease, lrclib], {
        revision: 4,
        selection: providerSelection(
          'token-lrclib',
          identity('r-l@0123456789abcdef0123456789abcdef', { format: 'elrc' }),
        ),
      }),
    )

    const { result } = renderPlayerLyrics()
    await showLyrics(result)

    expect(result.current.desktopLyricsProps.mainLyric.line[0].value).toBe(
      'LRCLib line',
    )
    expect(result.current.desktopLyricsProps.selectedSourceKey).toBe(
      'selection:token-lrclib',
    )
  })

  it('pins and offsets a local sidecar variant from durable state', async () => {
    const embeddedEntry = localEntry('Embedded line', embedded)
    const sidecarEntry = localEntry('Sidecar line', local)
    subsonic.getLyricsBySongId.mockResolvedValue(
      lyricsResponse([embeddedEntry, sidecarEntry], {
        revision: 7,
        selection: localSelection(local),
        offset: [{ kind: 'local', local, offsetMs: 500 }],
      }),
    )

    const { result } = renderPlayerLyrics()
    await showLyrics(result)

    expect(result.current.desktopLyricsProps.mainLyric.line[0].value).toBe(
      'Sidecar line',
    )
    expect(result.current.desktopLyricsProps.selectedSourceKey).toBe(
      `local:${local.token}`,
    )
    expect(result.current.desktopLyricsProps.offsetMs).toBe(500)
  })

  it('does not select a sibling when the stored local choice is unavailable', async () => {
    const embeddedEntry = localEntry('Embedded line', embedded)
    const sidecarEntry = localEntry('Sidecar line', local)
    subsonic.getLyricsBySongId.mockResolvedValue(
      lyricsResponse([embeddedEntry, sidecarEntry], {
        revision: 3,
        selection: localSelection(local, 'unavailable'),
      }),
    )

    const { result } = renderPlayerLyrics()
    await showLyrics(result)

    expect(result.current.desktopLyricsProps.mainLyric.line[0].value).toBe(
      'Embedded line',
    )
    expect(result.current.desktopLyricsProps.personalNotice).toContain(
      'player.lyricsPersonalFallbackText',
    )
  })

  it('posts a deliberate local selection, not Auto, and leaves unscoped legacy keys intact', async () => {
    localStorage.setItem('lyrics.source:song-1', 'legacy-lrclib')
    localStorage.setItem('lyrics.offset:song-1:legacy-lrclib', '250')
    const embeddedEntry = localEntry('Embedded line', embedded)
    const sidecarEntry = localEntry('Sidecar line', local)
    subsonic.getLyricsBySongId.mockResolvedValue(
      lyricsResponse([embeddedEntry, sidecarEntry], { revision: 2 }),
    )

    const { result } = renderPlayerLyrics()
    await showLyrics(result)

    act(() =>
      result.current.desktopLyricsProps.onSelectSource(`local:${local.token}`),
    )
    await waitFor(() =>
      expect(subsonic.updateLyricsPreference).toHaveBeenCalledWith('song-1', {
        action: 'select',
        local: local.token,
      }),
    )
    expect(localStorage.getItem('lyrics.source:song-1')).toBe('legacy-lrclib')
    expect(localStorage.getItem('lyrics.offset:song-1:legacy-lrclib')).toBe(
      '250',
    )
  })

  it('returns to automatic and resets the offset through durable actions', async () => {
    const embeddedEntry = localEntry('Embedded line', embedded)
    const sidecarEntry = localEntry('Sidecar line', local)
    subsonic.getLyricsBySongId.mockResolvedValue(
      lyricsResponse([embeddedEntry, sidecarEntry], {
        revision: 5,
        selection: localSelection(local),
        offset: [{ kind: 'local', local, offsetMs: -250 }],
      }),
    )
    subsonic.updateLyricsPreference.mockImplementation((_id, payload) =>
      Promise.resolve(
        mutationResponse({
          revision: 6,
          ...(payload.action === 'auto' ? {} : {}),
        }),
      ),
    )

    const { result } = renderPlayerLyrics()
    await showLyrics(result)
    expect(result.current.desktopLyricsProps.offsetMs).toBe(-250)

    act(() => result.current.desktopLyricsProps.onOffsetChange(0))
    await waitFor(() =>
      expect(subsonic.updateLyricsPreference).toHaveBeenCalledWith('song-1', {
        action: 'offset',
        local: local.token,
        offsetMs: 0,
      }),
    )

    act(() => result.current.desktopLyricsProps.onSelectSource(null))
    await waitFor(() =>
      expect(subsonic.updateLyricsPreference).toHaveBeenCalledWith('song-1', {
        action: 'auto',
      }),
    )
  })

  it('coalesces rapid offset clicks to the latest local action', async () => {
    const embeddedEntry = localEntry('Embedded line', embedded)
    const sidecarEntry = localEntry('Sidecar line', local)
    subsonic.getLyricsBySongId.mockResolvedValue(
      lyricsResponse([embeddedEntry, sidecarEntry], {
        revision: 2,
        selection: localSelection(local),
      }),
    )

    const { result } = renderPlayerLyrics()
    await showLyrics(result)
    act(() => result.current.desktopLyricsProps.onOffsetChange(50))
    act(() => result.current.desktopLyricsProps.onOffsetChange(100))
    act(() => result.current.desktopLyricsProps.onOffsetChange(-500))
    await waitFor(() =>
      expect(subsonic.updateLyricsPreference).toHaveBeenCalledTimes(1),
    )
    expect(subsonic.updateLyricsPreference).toHaveBeenCalledWith('song-1', {
      action: 'offset',
      local: local.token,
      offsetMs: -500,
    })
  })

  it('rejects a provider recording and restores it by token', async () => {
    const lrclib = providerEntry('LRCLib', 'LRCLib line', 'token-lrclib')
    const netease = providerEntry('NetEase', 'NetEase line', 'token-netease')
    subsonic.getLyricsBySongId.mockResolvedValue(
      lyricsResponse([netease, lrclib], { revision: 2 }),
    )
    subsonic.getLyricsVariants.mockResolvedValue(
      lyricsResponse([], {
        revision: 3,
        rejection: [
          {
            token: 'reject-token',
            plugin: 'plugin-id',
            provider: 'LRCLib',
            recording: 'r-a',
          },
        ],
      }),
    )

    const { result } = renderPlayerLyrics()
    await showLyrics(result)
    act(() =>
      result.current.desktopLyricsProps.onRejectVariant(
        result.current.desktopLyricsProps.variants.find(
          (variant) => variant.key === 'selection:token-lrclib',
        ),
      ),
    )
    await waitFor(() =>
      expect(subsonic.updateLyricsPreference).toHaveBeenCalledWith('song-1', {
        action: 'reject',
        selection: 'token-lrclib',
      }),
    )

    act(() =>
      result.current.desktopLyricsProps.onRestoreRejection('reject-token'),
    )
    await waitFor(() =>
      expect(subsonic.updateLyricsPreference).toHaveBeenCalledWith('song-1', {
        action: 'restore',
        rejection: 'reject-token',
      }),
    )
  })

  it('filters a rejected recording from a partial merge', async () => {
    const lrclib = providerEntry(
      'LRCLib',
      'LRCLib line',
      'token-lrclib',
      identity('r-l@0123456789abcdef0123456789abcdef'),
    )
    const netease = providerEntry(
      'NetEase',
      'NetEase line',
      'token-netease',
      identity('r-n@0123456789abcdef0123456789abcdef'),
    )
    subsonic.getLyricsBySongId.mockResolvedValueOnce(
      lyricsResponse([netease, lrclib], { revision: 2 }),
    )

    const { result } = renderPlayerLyrics()
    await showLyrics(result)
    expect(result.current.desktopLyricsProps.variants).toHaveLength(2)

    // The partial response is marked incomplete, which normally merges the
    // cached LRCLib source back in. The rejection must win.
    const partial = lyricsResponse([netease], {
      revision: 3,
      rejection: [
        {
          token: 'reject-token',
          plugin: 'plugin-id',
          provider: 'LRCLib',
          instance: 'plugin-instance',
          recording: 'r-l',
        },
      ],
    })
    partial.json['subsonic-response'].lyricsList.sourcesIncomplete = true
    subsonic.getLyricsBySongId.mockResolvedValueOnce(partial)

    act(() => result.current.desktopLyricsProps.onSourceMenuOpenChange(true))
    await waitFor(() =>
      expect(result.current.desktopLyricsProps.variants).toHaveLength(1),
    )
    expect(result.current.desktopLyricsProps.mainLyric.line[0].value).toBe(
      'NetEase line',
    )
  })

  it('does not retain stale lyrics that the listener has now rejected', async () => {
    const lrclib = providerEntry(
      'LRCLib',
      'LRCLib line',
      'token-lrclib',
      identity('r-l@0123456789abcdef0123456789abcdef'),
    )
    subsonic.getLyricsBySongId.mockResolvedValueOnce(
      lyricsResponse([lrclib], { revision: 2 }),
    )
    const { result } = renderPlayerLyrics()
    await showLyrics(result)
    expect(result.current.desktopLyricsProps.mainLyric.line[0].value).toBe(
      'LRCLib line',
    )

    const empty = lyricsResponse([], {
      revision: 3,
      rejection: [
        {
          token: 'reject-token',
          plugin: 'plugin-id',
          provider: 'LRCLib',
          instance: 'plugin-instance',
          recording: 'r-l',
        },
      ],
    })
    empty.json['subsonic-response'].lyricsList.sourcesIncomplete = true
    subsonic.getLyricsBySongId.mockResolvedValueOnce(empty)
    act(() => result.current.desktopLyricsProps.onSourceMenuOpenChange(true))

    await waitFor(() =>
      expect(result.current.desktopLyricsProps.mainLyric).toBeNull(),
    )
  })

  it('resets personal state when the browser session switches accounts', async () => {
    const lrclib = providerEntry('LRCLib', 'LRCLib line', 'token-lrclib')
    subsonic.getLyricsBySongId.mockResolvedValue(
      lyricsResponse([lrclib], {
        revision: 2,
        selection: providerSelection('token-lrclib'),
      }),
    )

    const { result, rerender } = renderPlayerLyrics()
    await showLyrics(result)
    expect(result.current.desktopLyricsProps.selectedSourceKey).toBe(
      'selection:token-lrclib',
    )

    subsonic.getLyricsBySongId.mockResolvedValue(
      lyricsResponse([lrclib], { revision: 0 }),
    )
    rerender({ authIdentity: 'user-b' })
    await waitFor(() =>
      expect(result.current.desktopLyricsProps.selectedSourceKey).toBeNull(),
    )
    expect(result.current.desktopLyricsProps.mainLyric.line[0].value).toBe(
      'LRCLib line',
    )
  })

  it('ignores a late write response from the previous account', async () => {
    const lrclib = providerEntry('LRCLib', 'LRCLib line', 'token-lrclib')
    subsonic.getLyricsBySongId.mockResolvedValue(
      lyricsResponse([lrclib], { revision: 2 }),
    )
    const pending = deferred()
    subsonic.updateLyricsPreference.mockReturnValue(pending.promise)

    const { result, rerender } = renderPlayerLyrics()
    await showLyrics(result)
    act(() =>
      result.current.desktopLyricsProps.onSelectSource(
        'selection:token-lrclib',
      ),
    )
    rerender({ authIdentity: 'user-b' })
    await act(async () => {
      pending.resolve(
        mutationResponse({
          revision: 3,
          selection: providerSelection('token-lrclib'),
        }),
      )
      await Promise.resolve()
    })
    expect(result.current.desktopLyricsProps.selectedSourceKey).toBeNull()
  })
})
