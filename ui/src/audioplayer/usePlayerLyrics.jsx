import React, { useCallback, useMemo, useRef, useState } from 'react'
import LyricsLayerControls from './LyricsLayerControls'
import LyricsPanel from './LyricsPanel'
import MobileKaraokeLyricsPortal from './MobileKaraokeLyricsPortal'
import { hasStructuredLyricContent } from './lyrics'
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
  const lyricsSource =
    lyricLayers.main?.source ||
    lyricLayers.translation?.source ||
    lyricLayers.pronunciation?.source ||
    null
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
      viewSource: translate('player.viewLyricsSourceText'),
      sourceTitle: translate('player.lyricsSourceText'),
      embeddedSource: translate('player.embeddedLyricsSourceText'),
      sidecarSource: translate('player.sidecarLyricsSourceText'),
      pluginSource: translate('player.pluginLyricsSourceText'),
      provider: translate('player.lyricsProviderText'),
      format: translate('player.lyricsFormatText'),
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
      mainLyric: lyricLayers.main,
      translationLyric: lyricLayers.translation,
      pronunciationLyric: lyricLayers.pronunciation,
      showTranslation,
      showPronunciation,
      translationEnabled: hasTranslationLyric,
      pronunciationEnabled: hasPronunciationLyric,
      source: lyricsSource,
      onToggleTranslation: toggleTranslation,
      onTogglePronunciation: togglePronunciation,
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
      hasPronunciationLyric,
      hasTranslationLyric,
      isDesktop,
      labels,
      lyricLayers.main,
      lyricLayers.pronunciation,
      lyricLayers.translation,
      lyricsError,
      lyricsLoading,
      lyricsSource,
      lyricsVisible,
      obscuredByQueue,
      showPronunciation,
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
          mainLyric={lyricLayers.main}
          translationLyric={lyricLayers.translation}
          pronunciationLyric={lyricLayers.pronunciation}
          showTranslation={showTranslation}
          showPronunciation={showPronunciation}
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
          translationEnabled={hasTranslationLyric}
          pronunciationEnabled={hasPronunciationLyric}
          onToggleTranslation={toggleTranslation}
          onTogglePronunciation={togglePronunciation}
          source={lyricsSource}
          labels={labels}
          testId="lyrics-mobile-layer-controls"
        />
      </MobileKaraokeLyricsPortal>
    ),
    [
      audioInstance,
      hasPronunciationLyric,
      hasTranslationLyric,
      lyricLayers.main,
      lyricLayers.pronunciation,
      lyricLayers.translation,
      lyricsError,
      lyricsLoading,
      lyricsSource,
      labels,
      obscuredByQueue,
      showPronunciation,
      showTranslation,
      togglePronunciation,
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
