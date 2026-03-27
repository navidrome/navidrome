import {
  resolveLyricsOverlayState,
  togglePronunciationPreference,
} from './lyricsOverlayState'

describe('Player lyrics state helpers', () => {
  it('keeps the lyrics window preference across track changes in the session', () => {
    const visibleOnCurrentTrack = resolveLyricsOverlayState({
      karaokeVisiblePreference: true,
      translationPreference: false,
      pronunciationPreference: null,
      hasKaraokeLyric: true,
      hasTranslationLyric: true,
      hasPronunciationLyric: true,
    })
    expect(visibleOnCurrentTrack.karaokeVisible).toBe(true)

    const hiddenForTrackWithoutLyrics = resolveLyricsOverlayState({
      karaokeVisiblePreference: true,
      translationPreference: false,
      pronunciationPreference: null,
      hasKaraokeLyric: false,
      hasTranslationLyric: false,
      hasPronunciationLyric: false,
    })
    expect(hiddenForTrackWithoutLyrics.karaokeVisible).toBe(false)

    const restoredOnNextLyricsTrack = resolveLyricsOverlayState({
      karaokeVisiblePreference: true,
      translationPreference: false,
      pronunciationPreference: null,
      hasKaraokeLyric: true,
      hasTranslationLyric: false,
      hasPronunciationLyric: false,
    })
    expect(restoredOnNextLyricsTrack.karaokeVisible).toBe(true)
  })

  it('restores translation and pronunciation preferences after tracks without those layers', () => {
    const initialState = resolveLyricsOverlayState({
      karaokeVisiblePreference: false,
      translationPreference: false,
      pronunciationPreference: null,
      hasKaraokeLyric: true,
      hasTranslationLyric: true,
      hasPronunciationLyric: true,
    })
    expect(initialState.showTranslation).toBe(false)
    expect(initialState.showPronunciation).toBe(true)

    const translationPreference = true
    const pronunciationPreference = togglePronunciationPreference(null, true)
    expect(pronunciationPreference).toBe(false)

    const hiddenOnTrackWithoutAuxLayers = resolveLyricsOverlayState({
      karaokeVisiblePreference: false,
      translationPreference,
      pronunciationPreference,
      hasKaraokeLyric: true,
      hasTranslationLyric: false,
      hasPronunciationLyric: false,
    })
    expect(hiddenOnTrackWithoutAuxLayers.showTranslation).toBe(false)
    expect(hiddenOnTrackWithoutAuxLayers.showPronunciation).toBe(false)

    const restoredOnTrackWithAuxLayers = resolveLyricsOverlayState({
      karaokeVisiblePreference: false,
      translationPreference,
      pronunciationPreference,
      hasKaraokeLyric: true,
      hasTranslationLyric: true,
      hasPronunciationLyric: true,
    })
    expect(restoredOnTrackWithAuxLayers.showTranslation).toBe(true)
    expect(restoredOnTrackWithAuxLayers.showPronunciation).toBe(false)
  })
})
