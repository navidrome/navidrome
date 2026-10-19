import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import LyricsLayerControls from './LyricsLayerControls'
import LyricsPanel from './LyricsPanel'
import MobileKaraokeLyricsPortal from './MobileKaraokeLyricsPortal'
import {
  findLyricVariant,
  hasNudgeableTiming,
  hasStructuredLyricContent,
  legacyVariantKey,
} from './lyrics'
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
const SOURCE_REFRESH_SLOW_TICKS = 6

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
  const [discoveryGeneration, setDiscoveryGeneration] = useState(0)
  const [discovering, setDiscovering] = useState(false)
  const lyricsToggleRef = useRef(null)
  const discoveryStateRef = useRef(null)

  const {
    variants,
    layers: lyricLayers,
    loading: lyricsLoading,
    refreshing: lyricsRefreshing,
    sourcesIncomplete,
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

  // The selected variant falls back to source priority when the saved
  // source is not in this track's response (cache expiry, config change).
  const resolvedVariantKey = useMemo(() => {
    return (
      findLyricVariant(variants, selectedSourceKey)?.key ??
      variants[0]?.key ??
      null
    )
  }, [selectedSourceKey, variants])

  useEffect(() => {
    setSelectedSourceKey(readPreferredSource(trackId))
  }, [trackId])

  const legacyOffsetKey = useMemo(() => {
    const variant = findLyricVariant(variants, resolvedVariantKey)
    const legacyKey = variant ? legacyVariantKey(variant.source) : null
    const unambiguous =
      legacyKey &&
      variants.filter((entry) => legacyVariantKey(entry.source) === legacyKey)
        .length === 1
    return unambiguous ? legacyKey : null
  }, [resolvedVariantKey, variants])

  useEffect(() => {
    setOffsetMs(readLyricsOffset(trackId, resolvedVariantKey, legacyOffsetKey))
  }, [legacyOffsetKey, resolvedVariantKey, trackId])

  useEffect(() => {
    const saved = findLyricVariant(variants, selectedSourceKey)
    if (saved && saved.key !== selectedSourceKey) {
      setSelectedSourceKey(saved.key)
      writePreferredSource(trackId, saved.key)
    }
  }, [selectedSourceKey, trackId, variants])

  useEffect(() => {
    discoveryStateRef.current = {
      loading: lyricsLoading,
      refreshing: lyricsRefreshing,
      error: lyricsError,
      main: lyricLayers.main,
      hasPlugin: variants.some((variant) => variant.source?.type === 'plugin'),
      sourcesIncomplete,
    }
  }, [
    lyricLayers.main,
    lyricsError,
    lyricsLoading,
    lyricsRefreshing,
    variants,
    sourcesIncomplete,
  ])

  // Older providers expose no completion signal. Check quickly for one minute,
  // then every 30 visible seconds so queued sweeps can still deliver late hits.
  useEffect(() => {
    if (!lyricsVisiblePreference || !trackId || isRadio) {
      setDiscovering(false)
      return undefined
    }
    setDiscovering(true)
    let ticks = 0
    let requestedDiscovery = false
    const timer = window.setInterval(() => {
      if (document.visibilityState === 'hidden') return
      ticks += 1
      const current = discoveryStateRef.current
      const localOnly =
        current?.main && !current.hasPlugin && !current.sourcesIncomplete
      if (localOnly && requestedDiscovery && !current.refreshing) {
        window.clearInterval(timer)
        setDiscovering(false)
        return
      }
      if (ticks >= SOURCE_REFRESH_MAX_TICKS) setDiscovering(false)
      const shouldRefresh =
        ticks <= SOURCE_REFRESH_MAX_TICKS ||
        (ticks - SOURCE_REFRESH_MAX_TICKS) % SOURCE_REFRESH_SLOW_TICKS === 0
      // Error retries already honor provider cooldowns and transient backoff.
      if (shouldRefresh && !current?.loading && !current?.error) {
        requestedDiscovery = refreshLyrics() || requestedDiscovery
      }
    }, SOURCE_REFRESH_INTERVAL_MS)
    return () => window.clearInterval(timer)
  }, [
    discoveryGeneration,
    isRadio,
    lyricsVisiblePreference,
    refreshLyrics,
    trackId,
    trackUpdatedAt,
  ])

  const waitingForLyrics =
    lyricsLoading || (discovering && !lyricsError && variants.length === 0)
  const checkingMoreSources =
    discovering && !lyricsLoading && !lyricsError && variants.length === 0

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
      loading: translate(
        checkingMoreSources
          ? 'player.lyricsSearchingMoreText'
          : 'player.lyricsLoadingText',
      ),
      unavailable: translate('player.lyricsUnavailableText'),
      rateLimited:
        lyricsRetryAfterSeconds == null
          ? null
          : translate('player.lyricsRateLimitedText', {
              smart_count: lyricsRetryAfterSeconds,
            }),
      empty: translate('player.lyricsNotFoundYetText'),
      returnToCurrentLine: translate('player.returnToCurrentLyricText'),
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
    [checkingMoreSources, lyricsRetryAfterSeconds, translate],
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

  const handleSourceMenuOpenChange = useCallback(
    (open) => {
      if (!open) return
      refreshLyrics()
      setDiscoveryGeneration((current) => current + 1)
    },
    [refreshLyrics],
  )

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
      lyricsLoading: waitingForLyrics,
      lyricsLabel: labels.toggle,
      lyricsLoadingLabel: labels.loading,
      lyricsToggleRef,
    }),
    [
      labels,
      waitingForLyrics,
      lyricsToggleDisabled,
      lyricsVisible,
      toggleLyrics,
    ],
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
      translationEnabled: hasTranslationLyric,
      pronunciationEnabled: hasPronunciationLyric,
      onToggleTranslation: toggleTranslation,
      onTogglePronunciation: togglePronunciation,
      ...variantControlProps,
      audioInstance,
      loading: waitingForLyrics,
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
      waitingForLyrics,
      lyricsVisible,
      obscuredByQueue,
      showPronunciation,
      showTranslation,
      togglePronunciation,
      toggleTranslation,
      trackId,
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
          trackId={trackId}
          selectedSourceKey={selectedSourceKey}
          obscured={obscuredByQueue}
          mainLyric={lyricLayers.main}
          translationLyric={lyricLayers.translation}
          pronunciationLyric={lyricLayers.pronunciation}
          showTranslation={showTranslation}
          showPronunciation={showPronunciation}
          audioInstance={audioInstance}
          offsetMs={offsetMs}
          loading={waitingForLyrics}
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
      waitingForLyrics,
      obscuredByQueue,
      offsetMs,
      selectedSourceKey,
      showPronunciation,
      showTranslation,
      togglePronunciation,
      toggleTranslation,
      trackId,
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
