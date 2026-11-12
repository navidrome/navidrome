import React, { useCallback, useMemo, useRef, useState } from 'react'
import LyricsLayerControls from './LyricsLayerControls'
import LyricsPanel from './LyricsPanel'
import MobileKaraokeLyricsPortal from './MobileKaraokeLyricsPortal'
import { hasStructuredLyricContent, hasWordTiming } from './lyrics'
import {
  readLetterWavePreference,
  writeLetterWavePreference,
} from './lyricsPreferences'
import {
  resolveLyricsSidebarState,
  toggleLayerPreference,
} from './lyricsSidebarState'
import useEnhancedLyrics from './useEnhancedLyrics'

const usePlayerLyrics = ({
  trackId,
  trackUpdatedAt,
  isRadio,
  audioInstance,
  isDesktop,
  obscuredByQueue = false,
  translate,
}) => {
  const [lyricsVisiblePreference, setLyricsVisiblePreference] = useState(false)
  const [translationPreference, setTranslationPreference] = useState(null)
  const [pronunciationPreference, setPronunciationPreference] = useState(null)
  const [letterWavePreference, setLetterWavePreference] = useState(
    readLetterWavePreference,
  )
  const lyricsToggleRef = useRef(null)

  const {
    layers: lyricLayers,
    loading: lyricsLoading,
    error: lyricsError,
    retryAfterSeconds: lyricsRetryAfterSeconds,
    retry: retryLyrics,
  } = useEnhancedLyrics({
    trackId,
    updatedAt: trackUpdatedAt,
    disabled: isRadio,
    requested: lyricsVisiblePreference,
  })

  const hasTranslationLyric = hasStructuredLyricContent(lyricLayers.translation)
  const hasPronunciationLyric = hasStructuredLyricContent(
    lyricLayers.pronunciation,
  )
  const hasLetterWave = useMemo(
    () => hasWordTiming(lyricLayers.main),
    [lyricLayers.main],
  )
  const { lyricsVisible, showTranslation, showPronunciation } =
    resolveLyricsSidebarState({
      lyricsVisiblePreference,
      translationPreference,
      pronunciationPreference,
      hasTranslationLyric,
      hasPronunciationLyric,
    })
  const lyricsToggleDisabled = !lyricsVisiblePreference && (isRadio || !trackId)
  const useInlineMobileLyrics = lyricsVisible && !isDesktop

  const labels = useMemo(
    () => ({
      title: translate('player.lyricsTitleText'),
      toggle: translate('player.toggleLyricText'),
      loading: translate('player.lyricsLoadingText'),
      unavailable: translate('player.lyricsUnavailableText'),
      rateLimited:
        lyricsRetryAfterSeconds == null
          ? null
          : translate('player.lyricsRateLimitedText', {
              smart_count: lyricsRetryAfterSeconds,
            }),
      empty: translate('player.emptyLyricText'),
      resize: translate('player.resizeLyricsSidebarText'),
      showTranslation: translate('player.showLyricsTranslationText'),
      hideTranslation: translate('player.hideLyricsTranslationText'),
      showPronunciation: translate('player.showLyricsPronunciationText'),
      hidePronunciation: translate('player.hideLyricsPronunciationText'),
      showLetterWave: translate('player.showLyricsLetterWaveText'),
      hideLetterWave: translate('player.hideLyricsLetterWaveText'),
    }),
    [lyricsRetryAfterSeconds, translate],
  )

  const toggleLyrics = useCallback(() => {
    const next = !lyricsVisiblePreference
    setLyricsVisiblePreference(next)
    if (next) {
      if (lyricsError) retryLyrics()
    }
  }, [lyricsError, lyricsVisiblePreference, retryLyrics])

  const closeLyrics = useCallback(() => {
    setLyricsVisiblePreference(false)
  }, [])

  const toggleTranslation = useCallback(() => {
    setTranslationPreference((current) =>
      toggleLayerPreference(current, hasTranslationLyric),
    )
  }, [hasTranslationLyric])

  const togglePronunciation = useCallback(() => {
    setPronunciationPreference((current) =>
      toggleLayerPreference(current, hasPronunciationLyric, true),
    )
  }, [hasPronunciationLyric])

  const toggleLetterWave = useCallback(() => {
    setLetterWavePreference((current) => {
      const next = !current
      writeLetterWavePreference(next)
      return next
    })
  }, [])

  const toolbarLyricsProps = useMemo(
    () => ({
      onToggleLyrics: toggleLyrics,
      lyricsActive: lyricsVisible,
      lyricsDisabled: lyricsToggleDisabled,
      lyricsLoading,
      lyricsLabel: labels.toggle,
      lyricsLoadingLabel: labels.loading,
      lyricsToggleRef,
    }),
    [labels, lyricsLoading, lyricsToggleDisabled, lyricsVisible, toggleLyrics],
  )

  const desktopLyricsProps = useMemo(
    () => ({
      visible: isDesktop && lyricsVisible,
      trackId,
      mainLyric: lyricLayers.main,
      translationLyric: lyricLayers.translation,
      pronunciationLyric: lyricLayers.pronunciation,
      showTranslation,
      showPronunciation,
      letterWave: letterWavePreference,
      translationEnabled: hasTranslationLyric,
      pronunciationEnabled: hasPronunciationLyric,
      letterWaveEnabled: hasLetterWave,
      onToggleTranslation: toggleTranslation,
      onTogglePronunciation: togglePronunciation,
      onToggleLetterWave: toggleLetterWave,
      audioInstance,
      loading: lyricsLoading,
      error: lyricsError,
      errorMessage: labels.rateLimited,
      labels,
      obscuredByQueue,
      returnFocusRef: lyricsToggleRef,
    }),
    [
      audioInstance,
      hasLetterWave,
      hasPronunciationLyric,
      hasTranslationLyric,
      isDesktop,
      letterWavePreference,
      toggleLetterWave,
      labels,
      lyricLayers.main,
      lyricLayers.pronunciation,
      lyricLayers.translation,
      lyricsError,
      lyricsLoading,
      lyricsVisible,
      obscuredByQueue,
      showPronunciation,
      trackId,
      showTranslation,
      togglePronunciation,
      toggleTranslation,
    ],
  )

  const mobileLyricsSurface = useMemo(
    () => (
      <MobileKaraokeLyricsPortal
        active={useInlineMobileLyrics}
        obscured={obscuredByQueue}
        returnFocusRef={lyricsToggleRef}
      >
        <LyricsPanel
          visible
          obscured={obscuredByQueue}
          trackId={trackId}
          mainLyric={lyricLayers.main}
          translationLyric={lyricLayers.translation}
          pronunciationLyric={lyricLayers.pronunciation}
          showTranslation={showTranslation}
          showPronunciation={showPronunciation}
          letterWave={letterWavePreference}
          audioInstance={audioInstance}
          loading={lyricsLoading}
          error={lyricsError}
          errorMessage={labels.rateLimited}
          labels={labels}
          inline
        />
        <LyricsLayerControls
          placement="mobile"
          showTranslation={showTranslation}
          showPronunciation={showPronunciation}
          letterWave={letterWavePreference}
          translationEnabled={hasTranslationLyric}
          pronunciationEnabled={hasPronunciationLyric}
          letterWaveEnabled={hasLetterWave}
          onToggleTranslation={toggleTranslation}
          onTogglePronunciation={togglePronunciation}
          onToggleLetterWave={toggleLetterWave}
          labels={labels}
          testId="lyrics-mobile-layer-controls"
        />
      </MobileKaraokeLyricsPortal>
    ),
    [
      audioInstance,
      hasLetterWave,
      hasPronunciationLyric,
      hasTranslationLyric,
      letterWavePreference,
      toggleLetterWave,
      lyricLayers.main,
      lyricLayers.pronunciation,
      lyricLayers.translation,
      lyricsError,
      lyricsLoading,
      labels,
      obscuredByQueue,
      showPronunciation,
      showTranslation,
      togglePronunciation,
      trackId,
      toggleTranslation,
      useInlineMobileLyrics,
    ],
  )

  return {
    toolbarLyricsProps,
    desktopLyricsProps,
    mobileLyricsSurface,
    useInlineMobileLyrics,
    closeLyrics,
  }
}

export default usePlayerLyrics
