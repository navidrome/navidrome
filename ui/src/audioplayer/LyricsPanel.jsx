import { alpha, makeStyles, useTheme } from '@material-ui/core/styles'
import clsx from 'clsx'
import React, {
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from 'react'
import {
  buildKaraokeLines,
  hasStructuredLyricContent,
  hasUsableKaraokeTiming,
} from './lyrics'
import { KaraokeLineRow, KaraokeStackedLineRow } from './LyricsLineRows'
import {
  KARAOKE_ANIMATION_MS,
  KARAOKE_AUX_LINE_HEIGHT,
  KARAOKE_DESKTOP_ACTIVE_LINE_ANCHOR_RATIO,
  KARAOKE_EASING,
  KARAOKE_INLINE_ACTIVE_LINE_ANCHOR_RATIO,
  KARAOKE_LINE_ENTER_MS,
  KARAOKE_LINE_LIFT_PX,
  KARAOKE_LINE_MOTION_EASING,
  KARAOKE_MANUAL_SCROLL_PAUSE_MS,
  KARAOKE_SCROLLBAR_VISIBLE_MS,
  KARAOKE_TRANSLATION_OPACITY,
} from './lyricsKaraokeConstants'
import {
  animateScrollTop,
  cancelScrollAnimation,
  getAnchoredScrollTop,
  getScrollEndPadding,
} from './lyricsScroll'
import useLyricsTimeline from './useLyricsTimeline'

const KARAOKE_LAYER_OPACITY_TRANSITION = `opacity ${KARAOKE_ANIMATION_MS}ms ${KARAOKE_EASING}`
const KARAOKE_IDLE_LAYER_OPACITY = 0.49
const KARAOKE_TRANSLATION_IDLE_OPACITY =
  KARAOKE_IDLE_LAYER_OPACITY * KARAOKE_TRANSLATION_OPACITY

const useStyles = makeStyles((theme) => ({
  root: {
    height: '100%',
    minHeight: 0,
    display: 'flex',
    flexDirection: 'column',
    color: theme.palette.text.primary,
  },
  inlineRoot: {
    borderRadius: 'inherit',
    background: 'transparent',
    backdropFilter: 'blur(16px)',
    WebkitBackdropFilter: 'blur(16px)',
  },
  body: {
    flex: 1,
    minHeight: 0,
    overflowY: 'auto',
    overflowX: 'hidden',
    padding: theme.spacing(4, 2.25, 3.25),
    overscrollBehavior: 'contain',
    scrollbarWidth: 'none',
    msOverflowStyle: 'none',
    maskImage:
      'linear-gradient(to bottom, transparent 0, rgba(0, 0, 0, 0.15) 12px, #000 40px, #000 calc(100% - 120px), rgba(0, 0, 0, 0.12) calc(100% - 48px), transparent 100%)',
    WebkitMaskImage:
      'linear-gradient(to bottom, transparent 0, rgba(0, 0, 0, 0.15) 12px, #000 40px, #000 calc(100% - 120px), rgba(0, 0, 0, 0.12) calc(100% - 48px), transparent 100%)',
    '&::-webkit-scrollbar': {
      width: 0,
      height: 0,
    },
  },
  bodyTopFade: {
    maskImage:
      'linear-gradient(to bottom, transparent 0, rgba(0, 0, 0, 0.15) 8px, #000 24px, #000 calc(100% - 120px), rgba(0, 0, 0, 0.12) calc(100% - 48px), transparent 100%)',
    WebkitMaskImage:
      'linear-gradient(to bottom, transparent 0, rgba(0, 0, 0, 0.15) 8px, #000 24px, #000 calc(100% - 120px), rgba(0, 0, 0, 0.12) calc(100% - 48px), transparent 100%)',
  },
  bodyUserScrolling: {
    scrollbarWidth: 'thin',
    msOverflowStyle: 'auto',
    '&::-webkit-scrollbar': {
      width: 8,
      height: 8,
    },
    '&::-webkit-scrollbar-thumb': {
      backgroundColor: theme.palette.action.disabled,
      borderRadius: 999,
    },
    '&::-webkit-scrollbar-track': {
      backgroundColor: 'transparent',
    },
  },
  inlineBody: {
    padding: theme.spacing(0.5, 1.25, 1.5),
    textAlign: 'center',
  },
  lines: {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'stretch',
    gap: theme.spacing(3),
  },
  lineGroup: {
    width: '100%',
    borderRadius: theme.shape.borderRadius,
    '--lyrics-main-current-color':
      'var(--lyrics-main-idle-color, currentColor)',
    '--lyrics-pronunciation-current-color':
      'var(--lyrics-pronunciation-idle-color, currentColor)',
    '--lyrics-translation-current-color':
      'var(--lyrics-translation-idle-color, currentColor)',
    '--lyrics-layer-opacity': KARAOKE_IDLE_LAYER_OPACITY,
    transform: 'translateY(0)',
    transition: `background-color 150ms ${KARAOKE_LINE_MOTION_EASING}`,
    '&[role="button"]:focus-visible': {
      backgroundColor: alpha(theme.palette.text.primary, 0.055),
    },
    '@media (hover: hover) and (pointer: fine)': {
      '&[role="button"]:hover': {
        backgroundColor: alpha(theme.palette.text.primary, 0.055),
      },
    },
    '&[data-raised="true"][data-line-motion="line"]': {
      transform: `translateY(-${KARAOKE_LINE_LIFT_PX}px)`,
      transition: `transform ${KARAOKE_LINE_ENTER_MS}ms ${KARAOKE_LINE_MOTION_EASING}, background-color 150ms ${KARAOKE_LINE_MOTION_EASING}`,
    },
    '&[data-raised="true"][data-line-motion="character"][data-character-wave="false"]':
      {
        transform: `translateY(-${KARAOKE_LINE_LIFT_PX}px)`,
      },
    '&[data-active="true"]': {
      '--lyrics-main-current-color':
        'var(--lyrics-main-active-color, var(--lyrics-main-idle-color, currentColor))',
      '--lyrics-pronunciation-current-color':
        'var(--lyrics-pronunciation-active-color, var(--lyrics-pronunciation-idle-color, currentColor))',
      '--lyrics-translation-current-color':
        'var(--lyrics-translation-active-color, var(--lyrics-translation-idle-color, currentColor))',
      '--lyrics-layer-opacity': 1,
    },
    '@media (prefers-reduced-motion: reduce)': {
      transition: 'none',
      transform: 'none',
    },
  },
  waveCharacter: {
    display: 'inline-block',
    verticalAlign: 'baseline',
    transform: 'translateY(0)',
    transformOrigin: 'center bottom',
    '@media (prefers-reduced-motion: reduce)': {
      transform: 'none !important',
    },
  },
  line: {
    display: 'inline-block',
    width: '100%',
    maxWidth: '100%',
    fontWeight: 700,
    fontSize: 24,
    lineHeight: 1.18,
    overflowWrap: 'anywhere',
    whiteSpace: 'pre-wrap',
    letterSpacing: 0,
    color: 'var(--lyrics-main-current-color, currentColor)',
    WebkitTextFillColor: 'var(--lyrics-main-current-color, currentColor)',
    '&[data-tokenized="false"]': {
      opacity: 'var(--lyrics-layer-opacity)',
      color: 'var(--lyrics-main-active-color, currentColor)',
      WebkitTextFillColor: 'var(--lyrics-main-active-color, currentColor)',
      transition: KARAOKE_LAYER_OPACITY_TRANSITION,
    },
    '@media (prefers-reduced-motion: reduce)': {
      transition: 'none',
    },
  },
  inlineLine: {
    fontSize: 24,
  },
  auxLine: {
    display: 'block',
    marginTop: theme.spacing(0.8),
    fontWeight: 600,
    fontSize: 15,
    lineHeight: KARAOKE_AUX_LINE_HEIGHT,
    overflowWrap: 'anywhere',
    whiteSpace: 'pre-wrap',
    letterSpacing: 0,
    opacity: KARAOKE_TRANSLATION_IDLE_OPACITY,
    color: 'var(--lyrics-translation-active-color, currentColor)',
    WebkitTextFillColor: 'var(--lyrics-translation-active-color, currentColor)',
    transition: KARAOKE_LAYER_OPACITY_TRANSITION,
    '@media (prefers-reduced-motion: reduce)': {
      transition: 'none',
      transform: 'none',
    },
  },
  characterWaveAuxLine: {
    transform: `translateY(-${KARAOKE_LINE_LIFT_PX}px)`,
    transition: `${KARAOKE_LAYER_OPACITY_TRANSITION}, transform ${KARAOKE_LINE_ENTER_MS}ms ${KARAOKE_LINE_MOTION_EASING}`,
    '@media (prefers-reduced-motion: reduce)': {
      transition: 'none',
      transform: 'none',
    },
  },
  activeAuxLine: {
    opacity: KARAOKE_TRANSLATION_OPACITY,
  },
  stackedToken: {
    display: 'inline-flex',
    flexDirection: 'column',
    alignItems: 'flex-start',
    verticalAlign: 'top',
    width: 'max-content',
    minWidth: 0,
    maxWidth: '100%',
    overflowWrap: 'anywhere',
    whiteSpace: 'pre-wrap',
    paddingRight: theme.spacing(0.5),
    marginBottom: theme.spacing(0.25),
  },
  wrappedStackedLine: {
    '& $stackedToken': {
      marginBottom: theme.spacing(0.95),
    },
  },
  stackedSpacer: {
    display: 'inline',
    whiteSpace: 'pre-wrap',
  },
  stackedMainText: {
    display: 'block',
    lineHeight: 1.05,
    maxWidth: '100%',
    overflowWrap: 'anywhere',
    whiteSpace: 'pre-wrap',
  },
  stackedPronunciation: {
    display: 'block',
    marginTop: theme.spacing(0.15),
    fontSize: 15,
    lineHeight: 1.05,
    fontWeight: 700,
    maxWidth: '100%',
    overflowWrap: 'anywhere',
    whiteSpace: 'pre-wrap',
    color: 'var(--lyrics-pronunciation-current-color, currentColor)',
    WebkitTextFillColor:
      'var(--lyrics-pronunciation-current-color, currentColor)',
    '&[data-timed="false"]': {
      color: 'var(--lyrics-pronunciation-active-color, currentColor)',
      WebkitTextFillColor:
        'var(--lyrics-pronunciation-active-color, currentColor)',
    },
    '&[data-timed="true"]': {
      transition: 'none',
    },
    '@media (prefers-reduced-motion: reduce)': {
      transition: 'none',
    },
  },
  translationLine: {
    fontWeight: 600,
  },
  token: {
    whiteSpace: 'pre-wrap',
    overflowWrap: 'anywhere',
    fontKerning: 'none',
    fontVariantLigatures: 'none',
  },
  voiceLanes: {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'stretch',
    gap: theme.spacing(0.7),
  },
  secondaryVoiceLane: {
    fontSize: 22,
  },
  emptyState: {
    flex: 1,
    minHeight: 0,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    padding: theme.spacing(3),
    color: alpha(theme.palette.text.primary, 0.68),
    fontWeight: 600,
    textAlign: 'center',
  },
}))

const normalizeLineText = (value) =>
  String(value || '')
    .normalize('NFKC')
    .toLocaleLowerCase()
    .replace(/[\p{P}\p{S}\s]+/gu, '')

const shouldShowAuxLine = (mainLine, auxLine) =>
  Boolean(
    auxLine?.value &&
    normalizeLineText(auxLine.value) !== normalizeLineText(mainLine?.value),
  )

const finiteLineTime = (value) => {
  if (value == null || value === '') return null
  const number = Number(value)
  return Number.isFinite(number) ? number : null
}

const getLayerMatchWindow = (lines, index) => {
  const line = lines[index]
  if (!line) return { start: null, end: null }
  return {
    start: finiteLineTime(line.start),
    end: finiteLineTime(line.end) ?? finiteLineTime(lines[index + 1]?.start),
  }
}

const buildUniqueLayerMap = (mainLines, layerLines) => {
  if (!mainLines.length || !layerLines.length) return {}

  const candidates = []
  for (let layerIndex = 0; layerIndex < layerLines.length; layerIndex += 1) {
    if (layerLines[layerIndex]?.renderable === false) continue
    const layerWindow = getLayerMatchWindow(layerLines, layerIndex)
    for (let mainIndex = 0; mainIndex < mainLines.length; mainIndex += 1) {
      if (mainLines[mainIndex]?.renderable === false) continue
      const mainWindow = getLayerMatchWindow(mainLines, mainIndex)
      let score = null

      if (layerWindow.start != null && mainWindow.start != null) {
        const mainEnd = mainWindow.end ?? mainWindow.start
        const layerEnd = layerWindow.end ?? layerWindow.start
        const overlap =
          Math.min(mainEnd, layerEnd) -
          Math.max(mainWindow.start, layerWindow.start)
        const mainDuration = Math.max(0, mainEnd - mainWindow.start)
        const maxDelta = Math.max(550, Math.min(1400, mainDuration + 420))
        const startDelta = Math.abs(layerWindow.start - mainWindow.start)
        if (overlap < 0 && startDelta > maxDelta) continue
        score =
          startDelta +
          Math.abs(layerIndex - mainIndex) * 30 +
          (overlap < 0 ? 200 : 0)
      } else if (layerIndex === mainIndex) {
        score = 2000
      }

      if (score != null) {
        candidates.push({ layerIndex, mainIndex, score })
      }
    }
  }

  candidates.sort(
    (left, right) =>
      left.score - right.score ||
      left.layerIndex - right.layerIndex ||
      left.mainIndex - right.mainIndex,
  )

  const usedLayers = new Set()
  const usedMainLines = new Set()
  const result = {}
  for (const candidate of candidates) {
    if (
      usedLayers.has(candidate.layerIndex) ||
      usedMainLines.has(candidate.mainIndex)
    ) {
      continue
    }
    usedLayers.add(candidate.layerIndex)
    usedMainLines.add(candidate.mainIndex)
    result[candidate.mainIndex] = layerLines[candidate.layerIndex]
  }
  return result
}

const getLineLanes = (line) =>
  Array.isArray(line?.lanes) && line.lanes.length > 0 ? line.lanes : [line]

const buildSynchronizedTranslationLine = (_mainLine, translationLine) => {
  if (!translationLine) return null
  return {
    ...translationLine,
    tokens: [],
    lanes: undefined,
  }
}

const buildLineGroupStyle = (canSeekLine, layerStyles) => ({
  cursor: canSeekLine ? 'pointer' : undefined,
  '--lyrics-main-idle-color': layerStyles.main.color,
  '--lyrics-main-active-color':
    layerStyles.main['--lyrics-active-color'] || layerStyles.main.color,
  '--lyrics-pronunciation-idle-color': layerStyles.pronunciation.color,
  '--lyrics-pronunciation-active-color':
    layerStyles.pronunciation['--lyrics-active-color'] ||
    layerStyles.pronunciation.color,
  '--lyrics-translation-idle-color': layerStyles.translation.color,
  '--lyrics-translation-active-color':
    layerStyles.translation['--lyrics-active-color'] ||
    layerStyles.translation.color,
})

const usePrefersReducedMotion = () => {
  const [prefersReducedMotion, setPrefersReducedMotion] = useState(false)

  useEffect(() => {
    if (typeof window === 'undefined' || !window.matchMedia) return undefined
    const mediaQuery = window.matchMedia('(prefers-reduced-motion: reduce)')
    const updatePreference = () =>
      setPrefersReducedMotion(Boolean(mediaQuery.matches))
    updatePreference()
    if (mediaQuery.addEventListener) {
      mediaQuery.addEventListener('change', updatePreference)
      return () => mediaQuery.removeEventListener('change', updatePreference)
    }
    mediaQuery.addListener(updatePreference)
    return () => mediaQuery.removeListener(updatePreference)
  }, [])

  return prefersReducedMotion
}

const LyricsPanel = ({
  visible = true,
  mainLyric,
  translationLyric,
  pronunciationLyric,
  showTranslation,
  showPronunciation,
  audioInstance,
  inline = false,
  loading = false,
  error = null,
  labels = {},
}) => {
  const classes = useStyles()
  const theme = useTheme()
  const bodyRef = useRef(null)
  const scrollAnimationRef = useRef(null)
  const scrollbarTimerRef = useRef(null)
  const manualScrollTimerRef = useRef(null)
  const manualScrollUntilRef = useRef(0)
  const [showScrollbar, setShowScrollbar] = useState(false)
  const [autoScrollResumeKey, setAutoScrollResumeKey] = useState(0)
  const [layoutVersion, setLayoutVersion] = useState(0)
  const [hasTopFade, setHasTopFade] = useState(false)
  const [scrollEndPadding, setScrollEndPadding] = useState(0)
  const prefersReducedMotion = usePrefersReducedMotion()
  const activeLineAnchorRatio = inline
    ? KARAOKE_INLINE_ACTIVE_LINE_ANCHOR_RATIO
    : KARAOKE_DESKTOP_ACTIVE_LINE_ANCHOR_RATIO

  const mainLines = useMemo(() => buildKaraokeLines(mainLyric), [mainLyric])
  const translationLines = useMemo(
    () => buildKaraokeLines(translationLyric),
    [translationLyric],
  )
  const pronunciationLines = useMemo(
    () => buildKaraokeLines(pronunciationLyric),
    [pronunciationLyric],
  )
  const hasTimedMainLines = useMemo(
    () => hasUsableKaraokeTiming(mainLines),
    [mainLines],
  )
  const {
    activeIndexes,
    primaryIndex: activeIndex,
    scrollTargetIndex,
    registerLine,
    registerToken,
    getLineNode,
    syncNow,
    timeline,
  } = useLyricsTimeline({
    lines: mainLines,
    audioInstance,
    visible: visible && hasTimedMainLines,
    reducedMotion: prefersReducedMotion,
  })
  const activeIndexSet = useMemo(() => new Set(activeIndexes), [activeIndexes])

  const trByMainIndex = useMemo(() => {
    if (!showTranslation || translationLines.length === 0) return {}
    return buildUniqueLayerMap(mainLines, translationLines)
  }, [mainLines, translationLines, showTranslation])

  const prByMainIndex = useMemo(() => {
    if (!showPronunciation || pronunciationLines.length === 0) return {}
    return buildUniqueLayerMap(mainLines, pronunciationLines)
  }, [mainLines, pronunciationLines, showPronunciation])

  const colors = useMemo(
    () => ({
      main: theme.palette.text.primary,
      pronunciation: theme.palette.primary.main,
      translation:
        theme.palette.text.secondary ||
        theme.palette.secondary?.main ||
        theme.palette.text.primary,
    }),
    [theme],
  )

  const layerStyles = useMemo(() => {
    const styleFor = (layer) => {
      const sourceColor = colors[layer]
      if (layer === 'translation') {
        return {
          opacity: 1,
          color: sourceColor,
          '--lyrics-active-color': sourceColor,
        }
      }
      if (!hasTimedMainLines) {
        return {
          opacity: 1,
          color: alpha(sourceColor, layer === 'main' ? 0.98 : 0.86),
        }
      }
      const activeAlpha = layer === 'main' ? 0.98 : 0.78
      const pronunciationFadeRatio = 0.38 / 0.78
      const idleAlpha = activeAlpha * pronunciationFadeRatio
      return {
        opacity: 1,
        color: alpha(sourceColor, idleAlpha),
        '--lyrics-active-color': alpha(sourceColor, activeAlpha),
      }
    }
    return {
      main: styleFor('main'),
      pronunciation: styleFor('pronunciation'),
      translation: styleFor('translation'),
    }
  }, [colors, hasTimedMainLines])

  const showScrollbarForManualScroll = useCallback(() => {
    if (scrollbarTimerRef.current) {
      window.clearTimeout(scrollbarTimerRef.current)
    }
    setShowScrollbar(true)
    scrollbarTimerRef.current = window.setTimeout(() => {
      setShowScrollbar(false)
      scrollbarTimerRef.current = null
    }, KARAOKE_SCROLLBAR_VISIBLE_MS)
  }, [])

  const clearManualScrollTimer = useCallback(() => {
    if (manualScrollTimerRef.current) {
      window.clearTimeout(manualScrollTimerRef.current)
      manualScrollTimerRef.current = null
    }
  }, [])

  const resumeAutoScroll = useCallback(() => {
    clearManualScrollTimer()
    manualScrollUntilRef.current = 0
    setAutoScrollResumeKey((current) => current + 1)
  }, [clearManualScrollTimer])

  const markManualScrollIntent = useCallback(() => {
    cancelScrollAnimation(scrollAnimationRef)
    clearManualScrollTimer()
    manualScrollUntilRef.current =
      performance.now() + KARAOKE_MANUAL_SCROLL_PAUSE_MS
    manualScrollTimerRef.current = window.setTimeout(() => {
      manualScrollTimerRef.current = null
      resumeAutoScroll()
    }, KARAOKE_MANUAL_SCROLL_PAUSE_MS)
    showScrollbarForManualScroll()
  }, [clearManualScrollTimer, resumeAutoScroll, showScrollbarForManualScroll])

  const updateTopFade = useCallback(() => {
    const body = bodyRef.current
    setHasTopFade((current) => {
      const next = Boolean(body && body.scrollTop > 1)
      return current === next ? current : next
    })
  }, [])

  useEffect(
    () => () => {
      if (scrollbarTimerRef.current) {
        window.clearTimeout(scrollbarTimerRef.current)
      }
      if (manualScrollTimerRef.current) {
        window.clearTimeout(manualScrollTimerRef.current)
      }
      cancelScrollAnimation(scrollAnimationRef)
    },
    [],
  )

  useLayoutEffect(() => {
    const body = bodyRef.current
    if (!visible || !body) return undefined
    const ResizeObserverConstructor =
      typeof window !== 'undefined' ? window.ResizeObserver : null
    if (!ResizeObserverConstructor) return undefined
    const resizeObserver = new ResizeObserverConstructor(() => {
      setLayoutVersion((current) => current + 1)
    })
    resizeObserver.observe(body)
    return () => resizeObserver.disconnect()
  }, [visible])

  useLayoutEffect(() => {
    const body = bodyRef.current
    if (!visible || !body) {
      setScrollEndPadding(0)
      return
    }
    const nextPadding = getScrollEndPadding(body, activeLineAnchorRatio)
    setScrollEndPadding((current) =>
      current === nextPadding ? current : nextPadding,
    )
  }, [activeLineAnchorRatio, layoutVersion, mainLines.length, visible])

  useLayoutEffect(() => {
    const body = bodyRef.current
    if (!visible || !body) return
    cancelScrollAnimation(scrollAnimationRef)
    clearManualScrollTimer()
    manualScrollUntilRef.current = 0
    body.scrollTop = 0
    setHasTopFade(false)
  }, [clearManualScrollTimer, mainLyric, visible])

  useEffect(() => {
    if (!visible || !hasTimedMainLines || scrollTargetIndex < 0) {
      cancelScrollAnimation(scrollAnimationRef)
      return undefined
    }

    const animFrameId = window.requestAnimationFrame(() => {
      if (manualScrollUntilRef.current > performance.now()) return
      const body = bodyRef.current
      const targetNode = getLineNode(scrollTargetIndex)
      if (!body || !targetNode) return
      animateScrollTop({
        body,
        targetTop: getAnchoredScrollTop(
          body,
          targetNode,
          activeLineAnchorRatio,
        ),
        reducedMotion: prefersReducedMotion,
        scrollAnimationRef,
      })
    })

    return () => window.cancelAnimationFrame(animFrameId)
  }, [
    activeLineAnchorRatio,
    autoScrollResumeKey,
    getLineNode,
    hasTimedMainLines,
    layoutVersion,
    prefersReducedMotion,
    scrollTargetIndex,
    visible,
  ])

  if (!visible) return null

  if (!hasStructuredLyricContent(mainLyric) || mainLines.length === 0) {
    const message = loading
      ? labels.loading || 'Loading lyrics'
      : error
        ? labels.unavailable || 'Lyrics unavailable'
        : labels.empty || 'No lyrics available'
    return (
      <div
        className={clsx(classes.root, { [classes.inlineRoot]: inline })}
        data-testid="karaoke-lyrics-panel"
        data-inline={inline ? 'true' : 'false'}
        onClick={inline ? (event) => event.stopPropagation() : undefined}
      >
        <div
          className={classes.emptyState}
          data-testid="lyrics-empty-state"
          aria-live="polite"
        >
          {message}
        </div>
      </div>
    )
  }

  const seekToLine = (line) => {
    if (!audioInstance || line.start == null) return
    audioInstance.currentTime = line.start / 1000
    syncNow(line.start, true)
    resumeAutoScroll()
  }

  return (
    <div
      className={clsx(classes.root, { [classes.inlineRoot]: inline })}
      data-testid="karaoke-lyrics-panel"
      data-inline={inline ? 'true' : 'false'}
      onClick={inline ? (event) => event.stopPropagation() : undefined}
    >
      <div
        className={clsx(classes.body, {
          [classes.inlineBody]: inline,
          [classes.bodyTopFade]: hasTopFade,
          [classes.bodyUserScrolling]: showScrollbar,
        })}
        ref={bodyRef}
        data-testid="lyrics-scroll-body"
        data-reduced-motion={prefersReducedMotion ? 'true' : 'false'}
        data-scrollbar-visible={showScrollbar ? 'true' : 'false'}
        data-top-fade-enabled={hasTopFade ? 'true' : 'false'}
        onScroll={updateTopFade}
        onWheel={markManualScrollIntent}
        onTouchMove={markManualScrollIntent}
      >
        <div
          className={classes.lines}
          data-scroll-end-padding={scrollEndPadding}
          style={
            scrollEndPadding > 0
              ? { paddingBottom: scrollEndPadding }
              : undefined
          }
        >
          {mainLines.map((line, idx) => {
            if (line.renderable === false) return null
            const trLine = trByMainIndex[idx]
            const prLine = prByMainIndex[idx]
            const mainNextLineStart =
              timeline.windows[idx]?.nextTimedStart ?? null
            const showTr = shouldShowAuxLine(line, trLine)
            const showPr = shouldShowAuxLine(line, prLine)
            const lineLanes = getLineLanes(line)
            const usesCharacterRise = lineLanes.some(
              (lane) => Array.isArray(lane?.tokens) && lane.tokens.length > 0,
            )
            const canSeekLine = Boolean(audioInstance && line.start != null)
            const isActiveLine = activeIndexSet.has(idx)
            const renderCharacterWave = Boolean(
              !prefersReducedMotion && hasTimedMainLines && isActiveLine,
            )
            const isStaticLine = !hasTimedMainLines
            return (
              <div
                key={`line-${line.index}-${line.start ?? idx}`}
                ref={
                  hasTimedMainLines
                    ? (node) => registerLine(idx, node)
                    : undefined
                }
                className={classes.lineGroup}
                data-line-motion={usesCharacterRise ? 'character' : 'line'}
                data-character-wave={renderCharacterWave ? 'true' : 'false'}
                data-active={isStaticLine || isActiveLine ? 'true' : 'false'}
                {...(isStaticLine
                  ? {
                      'data-lifecycle': 'active',
                      'data-highlight-active': 'true',
                      'data-raised': 'false',
                    }
                  : {})}
                aria-current={idx === activeIndex ? 'true' : undefined}
                data-scroll-target={
                  idx === scrollTargetIndex ? 'true' : 'false'
                }
                data-testid="lyrics-line-group"
                style={buildLineGroupStyle(canSeekLine, layerStyles)}
                role={canSeekLine ? 'button' : undefined}
                tabIndex={canSeekLine ? 0 : undefined}
                onClick={() => seekToLine(line)}
                onMouseDown={
                  canSeekLine
                    ? (event) => {
                        event.preventDefault()
                      }
                    : undefined
                }
                onKeyDown={(event) => {
                  if (
                    canSeekLine &&
                    (event.key === 'Enter' || event.key === ' ')
                  ) {
                    event.preventDefault()
                    seekToLine(line)
                  }
                }}
              >
                {lineLanes.length > 1 ? (
                  <div
                    className={classes.voiceLanes}
                    data-testid="lyrics-voice-lanes"
                  >
                    {lineLanes.map((lane, laneIdx) => {
                      const laneClassName = clsx(classes.line, {
                        [classes.inlineLine]: inline,
                        [classes.secondaryVoiceLane]: laneIdx > 0,
                      })
                      const rowKey = lane.key || `lane-${laneIdx}`

                      return showPr && laneIdx === 0 ? (
                        <KaraokeStackedLineRow
                          key={rowKey}
                          lineIndex={idx}
                          line={lane}
                          pronunciationLine={prLine}
                          pronunciationStyle={layerStyles.pronunciation}
                          nextLineStart={mainNextLineStart}
                          className={laneClassName}
                          style={layerStyles.main}
                          tokenClassName={classes.token}
                          waveCharacterClassName={classes.waveCharacter}
                          classes={classes}
                          registerToken={registerToken}
                          renderCharacterWave={renderCharacterWave}
                          rowKey={rowKey}
                          testId="lyrics-voice-lane"
                        />
                      ) : (
                        <KaraokeLineRow
                          key={rowKey}
                          lineIndex={idx}
                          line={lane}
                          nextLineStart={mainNextLineStart}
                          className={laneClassName}
                          style={layerStyles.main}
                          tokenClassName={classes.token}
                          waveCharacterClassName={classes.waveCharacter}
                          registerToken={registerToken}
                          renderCharacterWave={renderCharacterWave}
                          rowKey={rowKey}
                          testId="lyrics-voice-lane"
                        />
                      )
                    })}
                  </div>
                ) : showPr ? (
                  <KaraokeStackedLineRow
                    lineIndex={idx}
                    line={line}
                    pronunciationLine={prLine}
                    pronunciationStyle={layerStyles.pronunciation}
                    nextLineStart={mainNextLineStart}
                    className={clsx(classes.line, {
                      [classes.inlineLine]: inline,
                    })}
                    style={layerStyles.main}
                    tokenClassName={classes.token}
                    waveCharacterClassName={classes.waveCharacter}
                    classes={classes}
                    registerToken={registerToken}
                    renderCharacterWave={renderCharacterWave}
                    rowKey="main"
                  />
                ) : (
                  <KaraokeLineRow
                    lineIndex={idx}
                    line={line}
                    nextLineStart={mainNextLineStart}
                    className={clsx(classes.line, {
                      [classes.inlineLine]: inline,
                    })}
                    style={layerStyles.main}
                    tokenClassName={classes.token}
                    waveCharacterClassName={classes.waveCharacter}
                    registerToken={registerToken}
                    renderCharacterWave={renderCharacterWave}
                    rowKey="main"
                  />
                )}
                {showTr && (
                  <KaraokeLineRow
                    lineIndex={idx}
                    line={buildSynchronizedTranslationLine(line, trLine)}
                    nextLineStart={null}
                    className={clsx(classes.auxLine, classes.translationLine, {
                      [classes.characterWaveAuxLine]: renderCharacterWave,
                      [classes.activeAuxLine]: isStaticLine || isActiveLine,
                    })}
                    style={layerStyles.translation}
                    tokenClassName={classes.token}
                    rowKey="translation"
                  />
                )}
              </div>
            )
          })}
        </div>
      </div>
    </div>
  )
}

export default LyricsPanel
