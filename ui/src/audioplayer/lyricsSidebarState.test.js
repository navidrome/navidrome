import { describe, expect, it } from 'vitest'
import {
  resolveLyricsSidebarState,
  toggleLayerPreference,
} from './lyricsSidebarState'

describe('lyricsSidebarState', () => {
  it('defaults available layers on and keeps preferences local', () => {
    const resolve = (availability, preferences = {}) =>
      resolveLyricsSidebarState({
        lyricsVisiblePreference: true,
        translationPreference: null,
        pronunciationPreference: null,
        ...availability,
        ...preferences,
      })

    expect(
      resolve({ hasTranslationLyric: true, hasPronunciationLyric: true }),
    ).toEqual({
      lyricsVisible: true,
      showTranslation: true,
      showPronunciation: true,
    })
    expect(
      resolve(
        { hasTranslationLyric: true, hasPronunciationLyric: false },
        { translationPreference: false, pronunciationPreference: true },
      ),
    ).toEqual({
      lyricsVisible: true,
      showTranslation: false,
      showPronunciation: false,
    })

    expect(toggleLayerPreference(null, true)).toBe(false)
    expect(toggleLayerPreference(null, true, false)).toBe(true)
    expect(toggleLayerPreference(false, true)).toBe(true)
    expect(toggleLayerPreference(true, false)).toBe(false)
  })
})
