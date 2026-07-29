import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
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
  const [lyricsRequested, setLyricsRequested] = useState(false)
  const [translationPreference, setTranslationPreference] = useState(null)
  const [pronunciationPreference, setPronunciationPreference] = useState(null)
  const lyricsToggleRef = useRef(null)

  useEffect(() => {
    setLyricsRequested(lyricsVisiblePreference)
  }, [trackId]) // eslint-disable-line react-hooks/exhaustive-deps

  const {
    layers: lyricLayers,
    loading: lyricsLoading,
    error: lyricsError,
    retry: retryLyrics,
  } = useEnhancedLyrics({
    trackId,
    updatedAt: trackUpdatedAt,
    disabled: isRadio,
    requested: lyricsRequested || lyricsVisiblePreference,
  })

  const hasTranslationLyric = hasStructuredLyricContent(lyricLayers.translation)
  const hasPronunciationLyric = hasStructuredLyricContent(
    lyricLayers.pronunciation,
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
      empty: translate('player.emptyLyricText'),
      resize: translate('player.resizeLyricsSidebarText'),
      showTranslation: translate('player.showLyricsTranslationText'),
      hideTranslation: translate('player.hideLyricsTranslationText'),
      showPronunciation: translate('player.showLyricsPronunciationText'),
      hidePronunciation: translate('player.hideLyricsPronunciationText'),
    }),
    [translate],
  )

  const toggleLyrics = useCallback(() => {
    const next = !lyricsVisiblePreference
    setLyricsVisiblePreference(next)
    setLyricsRequested(next)
    if (next) {
      if (lyricsError) retryLyrics()
    }
  }, [lyricsError, lyricsVisiblePreference, retryLyrics])

  const closeLyrics = useCallback(() => {
    setLyricsVisiblePreference(false)
    setLyricsRequested(false)
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
      onToggleTranslation: toggleTranslation,
      onTogglePronunciation: togglePronunciation,
      audioInstance,
      loading: lyricsLoading,
      error: lyricsError,
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
