import { act, waitFor } from '@testing-library/react'
import { renderHook } from '@testing-library/react-hooks'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import subsonic from '../subsonic'
import { clearEnhancedLyricsCache } from './useEnhancedLyrics'
import usePlayerLyrics from './usePlayerLyrics'

vi.mock('../subsonic', () => ({
  default: {
    getLyricsBySongId: vi.fn(),
  },
}))

const translate = (key) => key

describe('usePlayerLyrics', () => {
  beforeEach(() => {
    localStorage.setItem('locale', 'en')
    clearEnhancedLyricsCache()
    subsonic.getLyricsBySongId.mockReset()
  })

  afterEach(() => {
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
})
