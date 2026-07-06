export const resolveLyricsSidebarState = ({
  lyricsVisiblePreference,
  translationPreference,
  pronunciationPreference,
  hasTranslationLyric,
  hasPronunciationLyric,
}) => ({
  lyricsVisible: Boolean(lyricsVisiblePreference),
  showTranslation: Boolean(
    (translationPreference == null ? true : translationPreference) &&
    hasTranslationLyric,
  ),
  showPronunciation: Boolean(
    (pronunciationPreference == null ? false : pronunciationPreference) &&
    hasPronunciationLyric,
  ),
})

export const toggleLayerPreference = (
  previousPreference,
  hasLayer,
  defaultEnabled = true,
) => {
  if (!hasLayer) return false
  const currentPreference =
    previousPreference == null ? defaultEnabled : previousPreference
  return !currentPreference
}
