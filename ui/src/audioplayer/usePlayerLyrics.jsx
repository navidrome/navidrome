import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import LyricsLayerControls from './LyricsLayerControls'
import LyricsPanel from './LyricsPanel'
import MobileKaraokeLyricsPortal from './MobileKaraokeLyricsPortal'
import {
  findLyricVariant,
  hasNudgeableTiming,
  hasStructuredLyricContent,
  hasWordTiming,
  isPlayableVariant,
  legacyVariantKey,
} from './lyrics'
import {
  buildConvertedLyricsDownload,
  downloadBlob,
  downloadOriginalLyrics,
} from './lyricsExport'
import subsonic from '../subsonic'
import {
  readLetterWavePreference,
  readLyricsOffset,
  readPreferredSource,
  writeLetterWavePreference,
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

const isAbortError = (error) =>
  error?.name === 'AbortError' || error?.code === 'ABORT_ERR'

const operationKey = (target) =>
  [
    target?.trackId || '',
    target?.variantKey || '',
    target?.selection || '',
  ].join('\u0000')

const usePlayerLyrics = ({
  trackId,
  trackUpdatedAt,
  isRadio,
  audioInstance,
  isDesktop,
  obscuredByQueue = false,
  translate,
  onExportError = () => {},
}) => {
  const [lyricsVisiblePreference, setLyricsVisiblePreference] = useState(false)
  const [translationPreference, setTranslationPreference] = useState(null)
  const [pronunciationPreference, setPronunciationPreference] = useState(null)
  const [letterWavePreference, setLetterWavePreference] = useState(
    readLetterWavePreference,
  )
  const [selectedSourceKey, setSelectedSourceKey] = useState(() =>
    readPreferredSource(trackId),
  )
  const [offsetMs, setOffsetMs] = useState(0)
  const [discoveryGeneration, setDiscoveryGeneration] = useState(0)
  const [discovering, setDiscovering] = useState(false)
  const [downloadMenuOpen, setDownloadMenuOpen] = useState(false)
  const [originalInfo, setOriginalInfo] = useState(null)
  const lyricsToggleRef = useRef(null)
  const discoveryStateRef = useRef(null)
  const downloadMenuTargetRef = useRef('')
  const targetGenerationRef = useRef(0)
  const targetKeyRef = useRef('')
  const currentTargetRef = useRef({
    trackId: null,
    variantKey: null,
    selection: '',
  })
  const listingTokenRef = useRef(0)
  const exportTokenRef = useRef(0)
  const listingAbortRef = useRef(null)
  const exportAbortRef = useRef(null)

  const {
    variants,
    evidence = [],
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
      variants.find(isPlayableVariant)?.key ??
      null
    )
  }, [selectedSourceKey, variants])

  const selectedVariant = useMemo(
    () => findLyricVariant(variants, resolvedVariantKey),
    [resolvedVariantKey, variants],
  )
  const downloadFormat =
    lyricLayers.main?.source?.format || selectedVariant?.source?.format || ''
  const exportSelection = lyricLayers.main?.selection || ''
  const exportTarget = {
    trackId,
    variantKey: resolvedVariantKey,
    selection: exportSelection,
    format: downloadFormat,
  }
  const exportTargetKey = operationKey(exportTarget)
  if (targetKeyRef.current !== exportTargetKey) {
    targetKeyRef.current = exportTargetKey
    targetGenerationRef.current += 1
  }
  currentTargetRef.current = exportTarget

  const isCurrentTarget = useCallback((target) => {
    const current = currentTargetRef.current
    return (
      current.trackId === target.trackId &&
      current.variantKey === target.variantKey &&
      current.selection === target.selection
    )
  }, [])

  const startOperation = useCallback((kind) => {
    const controller = new AbortController()
    const token =
      kind === 'listing' ? ++listingTokenRef.current : ++exportTokenRef.current
    const activeRef = kind === 'listing' ? listingAbortRef : exportAbortRef
    activeRef.current?.abort()
    activeRef.current = controller
    return {
      kind,
      token,
      generation: targetGenerationRef.current,
      target: currentTargetRef.current,
      controller,
    }
  }, [])

  const isOperationCurrent = useCallback((operation) => {
    const current = currentTargetRef.current
    const token =
      operation.kind === 'listing'
        ? listingTokenRef.current
        : exportTokenRef.current
    return (
      operation.token === token &&
      operation.generation === targetGenerationRef.current &&
      current.trackId === operation.target.trackId &&
      current.variantKey === operation.target.variantKey &&
      current.selection === operation.target.selection
    )
  }, [])

  const commitIfCurrent = useCallback(
    (operation, commit) => {
      if (isOperationCurrent(operation)) commit()
    },
    [isOperationCurrent],
  )

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
    setDownloadMenuOpen(false)
    setOriginalInfo(null)
    downloadMenuTargetRef.current = ''
    listingAbortRef.current?.abort()
    exportAbortRef.current?.abort()
    listingTokenRef.current += 1
    exportTokenRef.current += 1
  }, [exportTargetKey])

  useEffect(
    () => () => {
      targetGenerationRef.current += 1
      listingAbortRef.current?.abort()
      exportAbortRef.current?.abort()
      listingTokenRef.current += 1
      exportTokenRef.current += 1
    },
    [],
  )

  useEffect(() => {
    if (
      !downloadMenuOpen ||
      downloadMenuTargetRef.current !== exportTargetKey
    ) {
      return undefined
    }
    const target = currentTargetRef.current
    if (!target.selection) {
      setOriginalInfo({
        status: 'unavailable',
        reason: translate('player.downloadLyricsUnavailableLegacyText'),
        selection: '',
      })
      return undefined
    }
    const operation = startOperation('listing')
    setOriginalInfo({ status: 'loading', selection: target.selection })
    subsonic
      .getLyricsVariants(target.trackId, {
        signal: operation.controller.signal,
      })
      .then((response) => {
        const listing = response?.json?.['subsonic-response']?.lyricsVariants
        const variants = listing?.variant || listing?.variants
        const match = Array.isArray(variants)
          ? variants.find((entry) => entry?.selection === target.selection)
          : null
        commitIfCurrent(operation, () => {
          if (!match) {
            setOriginalInfo({
              status: 'unavailable',
              reason: translate(
                'player.downloadLyricsUnavailableSelectionText',
              ),
              selection: target.selection,
            })
            return
          }
          const info = match.original || {}
          if (!info.available) {
            setOriginalInfo({
              status: 'unavailable',
              reason:
                info.reason ||
                translate('player.downloadLyricsUnavailableNativeText'),
              format: info.format,
              extension: info.extension,
              selection: target.selection,
            })
            return
          }
          setOriginalInfo({
            status: 'available',
            format: info.format,
            extension: info.extension,
            selection: target.selection,
          })
        })
      })
      .catch((error) => {
        if (operation.controller.signal.aborted || isAbortError(error)) return
        commitIfCurrent(operation, () => {
          setOriginalInfo({
            status: 'unavailable',
            reason: translate('player.downloadLyricsUnavailableCheckText'),
            selection: target.selection,
          })
        })
      })
      .finally(() => {
        if (listingAbortRef.current === operation.controller) {
          listingAbortRef.current = null
        }
      })
    return () => {
      operation.controller.abort()
      if (listingTokenRef.current === operation.token) {
        listingTokenRef.current += 1
      }
      if (listingAbortRef.current === operation.controller) {
        listingAbortRef.current = null
      }
    }
  }, [
    commitIfCurrent,
    downloadMenuOpen,
    exportTargetKey,
    startOperation,
    translate,
  ])

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
    lyricsLoading ||
    (discovering &&
      !lyricsError &&
      variants.length === 0 &&
      evidence.length === 0)
  const checkingMoreSources =
    discovering &&
    !lyricsLoading &&
    !lyricsError &&
    variants.length === 0 &&
    evidence.length === 0

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
      showLetterWave: translate('player.showLyricsLetterWaveText'),
      hideLetterWave: translate('player.hideLyricsLetterWaveText'),
      viewSource: translate('player.viewLyricsSourceText'),
      sourceTitle: translate('player.lyricsSourceText'),
      embeddedSource: translate('player.embeddedLyricsSourceText'),
      sidecarSource: translate('player.sidecarLyricsSourceText'),
      pluginSource: translate('player.pluginLyricsSourceText'),
      autoSource: translate('player.autoLyricsSourceText'),
      suggestion: translate('player.lyricsSuggestionText'),
      instrumentalTrack: translate('player.instrumentalTrackText'),
      instrumentalSimilar: translate('player.instrumentalSimilarText'),
      instrumentalOther: translate('player.instrumentalOtherText'),
      instrumentalMatched: translate('player.instrumentalMatchedText'),
      instrumentalConflict: translate('player.instrumentalConflictText'),
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
      downloadTitle: translate('player.downloadLyricsTitleText'),
      downloadOriginal: translate('player.downloadLyricsOriginalText'),
      downloadOriginalHint: translate('player.downloadLyricsOriginalHintText'),
      downloadOriginalChecking: translate(
        'player.downloadLyricsCheckingOriginalText',
      ),
      downloadOriginalCheckingHint: translate(
        'player.downloadLyricsCheckingOriginalHintText',
      ),
      downloadOriginalUnavailable: translate(
        'player.downloadLyricsOriginalUnavailableText',
      ),
      downloadOriginalUnavailableHint: translate(
        'player.downloadLyricsOriginalUnavailableHintText',
      ),
      downloadConverted: translate('player.downloadLyricsConvertedText'),
      downloadConvertedHint: translate(
        'player.downloadLyricsConvertedHintText',
      ),
      downloadAdjusted: translate('player.downloadLyricsAdjustedText'),
      downloadAdjustedHint: translate('player.downloadLyricsAdjustedHintText'),
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

  const toggleLetterWave = useCallback(() => {
    setLetterWavePreference((current) => {
      const next = !current
      writeLetterWavePreference(next)
      return next
    })
  }, [])

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

  const downloadSelected = useCallback(
    async (mode = 'converted') => {
      const lyric = lyricLayers.main
      const target = { ...currentTargetRef.current }
      if (!lyric || !target.variantKey) return
      setDownloadMenuOpen(false)
      const label = getVariantLabel(lyric.source, labels)
      if (mode === 'original') {
        if (!target.selection) {
          onExportError(translate('player.downloadLyricsUnavailableLegacyText'))
          return
        }
        const operation = startOperation('export')
        try {
          const original = await subsonic.getLyricsOriginal(
            target.trackId,
            target.selection,
            { signal: operation.controller.signal },
          )
          commitIfCurrent(operation, () => {
            const extension =
              originalInfo?.selection === target.selection
                ? originalInfo.extension
                : '.bin'
            downloadOriginalLyrics(original, {
              artist: lyric.displayArtist,
              title: lyric.displayTitle,
              extension,
            })
          })
        } catch (error) {
          if (operation.controller.signal.aborted || isAbortError(error)) {
            return
          }
          commitIfCurrent(operation, () => {
            setOriginalInfo({
              status: 'unavailable',
              reason:
                error?.message ||
                translate('player.downloadLyricsUnavailableCheckText'),
              selection: target.selection,
            })
            onExportError(translate('player.downloadLyricsErrorText'))
          })
        } finally {
          if (exportAbortRef.current === operation.controller) {
            exportAbortRef.current = null
          }
        }
        return
      }

      const download = buildConvertedLyricsDownload(lyric, {
        format: target.format,
        artist: lyric.displayArtist,
        title: lyric.displayTitle,
        label,
        mode,
        offsetMs: mode === 'adjusted' ? offsetMs : 0,
      })
      if (!download) return
      // Revalidate immediately before dispatching the browser download.
      if (!isCurrentTarget(target)) return
      downloadBlob(
        new Blob([download.text], { type: download.mediaType }),
        download.filename,
      )
    },
    [
      commitIfCurrent,
      isCurrentTarget,
      labels,
      lyricLayers.main,
      offsetMs,
      onExportError,
      originalInfo,
      startOperation,
      translate,
    ],
  )

  const handleDownloadMenuOpenChange = useCallback((open) => {
    downloadMenuTargetRef.current = open ? targetKeyRef.current : ''
    setDownloadMenuOpen(Boolean(open))
  }, [])

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
      downloadOpen: downloadMenuOpen,
      onDownloadMenuOpenChange: handleDownloadMenuOpenChange,
      originalInfo,
      downloadFormat,
    }),
    [
      changeOffset,
      downloadFormat,
      downloadMenuOpen,
      downloadSelected,
      handleDownloadMenuOpenChange,
      handleSourceMenuOpenChange,
      lyricsRefreshing,
      offsetEnabled,
      offsetMs,
      originalInfo,
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
      evidence,
      showTranslation,
      showPronunciation,
      letterWave: letterWavePreference,
      translationEnabled: hasTranslationLyric,
      pronunciationEnabled: hasPronunciationLyric,
      letterWaveEnabled: hasLetterWave,
      onToggleTranslation: toggleTranslation,
      onTogglePronunciation: togglePronunciation,
      onToggleLetterWave: toggleLetterWave,
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
      hasLetterWave,
      hasPronunciationLyric,
      hasTranslationLyric,
      isDesktop,
      labels,
      letterWavePreference,
      lyricLayers.main,
      lyricLayers.pronunciation,
      lyricLayers.translation,
      evidence,
      lyricsError,
      waitingForLyrics,
      lyricsVisible,
      obscuredByQueue,
      showPronunciation,
      showTranslation,
      toggleLetterWave,
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
          evidence={evidence}
          showTranslation={showTranslation}
          showPronunciation={showPronunciation}
          letterWave={letterWavePreference}
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
          letterWave={letterWavePreference}
          translationEnabled={hasTranslationLyric}
          pronunciationEnabled={hasPronunciationLyric}
          letterWaveEnabled={hasLetterWave}
          onToggleTranslation={toggleTranslation}
          onTogglePronunciation={togglePronunciation}
          onToggleLetterWave={toggleLetterWave}
          {...variantControlProps}
          active={useInlineMobileLyrics && !obscuredByQueue}
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
      labels,
      letterWavePreference,
      lyricLayers.main,
      lyricLayers.pronunciation,
      lyricLayers.translation,
      evidence,
      lyricsError,
      waitingForLyrics,
      obscuredByQueue,
      offsetMs,
      selectedSourceKey,
      showPronunciation,
      showTranslation,
      toggleLetterWave,
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
