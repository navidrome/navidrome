export const resolveLyricsSidebarState = ({
  lyricsVisiblePreference,
  translationPreference,
  pronunciationPreference,
  hasTranslationLyric,
  hasPronunciationLyric,
}) => ({
  lyricsVisible: Boolean(lyricsVisiblePreference),
  showTranslation:
    (translationPreference == null ? true : translationPreference) &&
    hasTranslationLyric,
  showPronunciation:
    (pronunciationPreference == null ? true : pronunciationPreference) &&
    hasPronunciationLyric,
})

export const toggleLayerPreference = (previousPreference, hasLayer) => {
  if (!hasLayer) return false
  const currentPreference =
    previousPreference == null ? true : previousPreference
  return !currentPreference
}
