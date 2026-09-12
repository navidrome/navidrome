export const resolveLyricsSidebarState = ({
  lyricsVisiblePreference,
  translationPreference,
  pronunciationPreference,
  hasTranslationLyric,
  hasPronunciationLyric,
}) => ({
  lyricsVisible: Boolean(lyricsVisiblePreference),
  showTranslation: Boolean(
    (translationPreference ?? true) && hasTranslationLyric,
  ),
  showPronunciation: Boolean(
    (pronunciationPreference ?? true) && hasPronunciationLyric,
  ),
})

export const toggleLayerPreference = (
  previousPreference,
  hasLayer,
  defaultEnabled = true,
) => {
  if (!hasLayer) return false
  return !(previousPreference ?? defaultEnabled)
}
