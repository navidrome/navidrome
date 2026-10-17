import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import LyricsLayerControls from './LyricsLayerControls'
import LyricsPanel from './LyricsPanel'
import MobileKaraokeLyricsPortal from './MobileKaraokeLyricsPortal'
import { hasNudgeableTiming, hasStructuredLyricContent } from './lyrics'
import { downloadLyrics } from './lyricsExport'
import {
  readLyricsOffset,
  readPreferredSource,
  writeLyricsOffset,
  writePreferredSource,
} from './lyricsPreferences'
import { getVariantLabel } from './lyricsSource'
import {
  resolveLyricsSidebarState,
  toggleLayerPreference,
} from './lyricsSidebarState'
import useEnhancedLyrics from './useEnhancedLyrics'

const SOURCE_REFRESH_INTERVAL_MS = 5_000
const SOURCE_REFRESH_MAX_TICKS = 12

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
  const [selectedSourceKey, setSelectedSourceKey] = useState(() =>
    readPreferredSource(trackId),
  )
  const [offsetMs, setOffsetMs] = useState(0)
  const [sourceMenuOpen, setSourceMenuOpen] = useState(false)
  const lyricsToggleRef = useRef(null)

  const {
    variants,
    layers: lyricLayers,
    loading: lyricsLoading,
    refreshing: lyricsRefreshing,
    error: lyricsError,
    retryAfterSeconds: lyricsRetryAfterSeconds,
    retry: retryLyrics,
    refresh: refreshLyrics,
  } = useEnhancedLyrics({
    trackId,
    updatedAt: trackUpdatedAt,
    disabled: isRadio,
    requested: lyricsVisiblePreference,
    selectedSourceKey,
  })

  // The selected variant falls back to the plugin's winner when the saved
  // source is not in this track's response (cache expiry, config change).
  const resolvedVariantKey = useMemo(() => {
    if (
      selectedSourceKey &&
      variants.some((variant) => variant.key === selectedSourceKey)
    ) {
      return selectedSourceKey
    }
    return variants[0]?.key ?? null
  }, [selectedSourceKey, variants])

  useEffect(() => {
    setSelectedSourceKey(readPreferredSource(trackId))
    setSourceMenuOpen(false)
  }, [trackId])

  useEffect(() => {
    setOffsetMs(readLyricsOffset(trackId, resolvedVariantKey))
  }, [resolvedVariantKey, trackId])

  // Progressive fill: while the source menu is open, ask for the served
  // variants again on a bounded interval so sweep-filled sources appear.
  useEffect(() => {
    if (!sourceMenuOpen) return undefined
    refreshLyrics()
    let ticks = 0
    const timer = window.setInterval(() => {
      ticks += 1
      refreshLyrics()
      if (ticks >= SOURCE_REFRESH_MAX_TICKS) window.clearInterval(timer)
    }, SOURCE_REFRESH_INTERVAL_MS)
    return () => window.clearInterval(timer)
  }, [refreshLyrics, sourceMenuOpen])

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
  const offsetEnabled = hasNudgeableTiming(lyricLayers.main)

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
      autoSource: translate('player.autoLyricsSourceText'),
      refreshingSources: translate('player.refreshingLyricsSourcesText'),
      wordSynced: translate('player.wordSyncedLyricText'),
      lineSynced: translate('player.lineSyncedLyricText'),
      plainSynced: translate('player.plainLyricText'),
      offset: translate('player.lyricsOffsetText'),
      offsetCoarseEarlier: translate('player.lyricsOffsetEarlierText', {
        seconds: '0.5',
      }),
      offsetFineEarlier: translate('player.lyricsOffsetEarlierText', {
        seconds: '0.05',
      }),
      offsetCoarseLater: translate('player.lyricsOffsetLaterText', {
        seconds: '0.5',
      }),
      offsetFineLater: translate('player.lyricsOffsetLaterText', {
        seconds: '0.05',
      }),
      offsetReset: translate('player.lyricsOffsetResetText'),
      download: translate('player.downloadLyricsText'),
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

  const selectSource = useCallback(
    (sourceKey) => {
      const next = sourceKey || null
      setSelectedSourceKey(next)
      writePreferredSource(trackId, next)
    },
    [trackId],
  )

  const changeOffset = useCallback(
    (nextOffsetMs) => {
      setOffsetMs(nextOffsetMs)
      writeLyricsOffset(trackId, resolvedVariantKey, nextOffsetMs)
    },
    [resolvedVariantKey, trackId],
  )

  const handleSourceMenuOpenChange = useCallback((open) => {
    setSourceMenuOpen(open)
  }, [])

  const downloadSelected = useCallback(() => {
    downloadLyrics(lyricLayers.main, {
      format: lyricLayers.main?.source?.format,
      artist: lyricLayers.main?.displayArtist,
      title: lyricLayers.main?.displayTitle,
      label: getVariantLabel(lyricLayers.main?.source, labels),
    })
  }, [labels, lyricLayers.main])

  const variantControlProps = useMemo(
    () => ({
      variants,
      selectedSourceKey,
      onSelectSource: selectSource,
      onSourceMenuOpenChange: handleSourceMenuOpenChange,
      refreshing: lyricsRefreshing,
      offsetMs,
      offsetEnabled,
      onOffsetChange: changeOffset,
      onDownload: downloadSelected,
    }),
    [
      changeOffset,
      downloadSelected,
      handleSourceMenuOpenChange,
      lyricsRefreshing,
      offsetEnabled,
      offsetMs,
      selectSource,
      selectedSourceKey,
      variants,
    ],
  )

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
      ...variantControlProps,
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
      lyricsVisible,
      obscuredByQueue,
      showPronunciation,
      showTranslation,
      togglePronunciation,
      toggleTranslation,
      variantControlProps,
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
          offsetMs={offsetMs}
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
          {...variantControlProps}
          active={useInlineMobileLyrics && !obscuredByQueue}
          labels={labels}
          testId="lyrics-mobile-layer-controls"
        />
      </MobileKaraokeLyricsPortal>
    ),
    [
      audioInstance,
      hasPronunciationLyric,
      hasTranslationLyric,
      labels,
      lyricLayers.main,
      lyricLayers.pronunciation,
      lyricLayers.translation,
      lyricsError,
      lyricsLoading,
      obscuredByQueue,
      offsetMs,
      showPronunciation,
      showTranslation,
      togglePronunciation,
      toggleTranslation,
      useInlineMobileLyrics,
      variantControlProps,
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
