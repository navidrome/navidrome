export const resolveLyricsOverlayState = ({
  karaokeVisiblePreference,
  translationPreference,
  pronunciationPreference,
  hasKaraokeLyric,
  hasTranslationLyric,
  hasPronunciationLyric,
}) => ({
  karaokeVisible: karaokeVisiblePreference && hasKaraokeLyric,
  showTranslation: translationPreference && hasTranslationLyric,
  showPronunciation:
    (pronunciationPreference == null
      ? hasPronunciationLyric
      : pronunciationPreference) && hasPronunciationLyric,
})

export const togglePronunciationPreference = (
  previousPreference,
  hasPronunciationLyric,
) => {
  if (!hasPronunciationLyric) {
    return false
  }
  const currentPreference =
    previousPreference == null ? hasPronunciationLyric : previousPreference
  return !currentPreference
}
