import { makeStyles, useTheme } from '@material-ui/core/styles'
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
  buildHighlightedAuxLine,
  buildHighlightedMainLine,
  buildKaraokeLines,
  hasStructuredLyricContent,
  hasUsableKaraokeTiming,
  resolveLayerLineForMain,
} from './lyrics'
import { KaraokeLineRow, KaraokeStackedLineRow } from './LyricsLineRows'
import {
  KARAOKE_ANIMATION_MS,
  KARAOKE_AUX_LINE_HEIGHT,
  KARAOKE_DESKTOP_ACTIVE_LINE_ANCHOR_RATIO,
  KARAOKE_EASING,
  KARAOKE_HIGHLIGHT_LEAD_MS,
  KARAOKE_INLINE_ACTIVE_LINE_ANCHOR_RATIO,
  KARAOKE_MANUAL_SCROLL_PAUSE_MS,
  KARAOKE_SCROLL_PRE_ROLL_MS,
  KARAOKE_SCROLLBAR_VISIBLE_MS,
  clamp,
  lerp,
} from './lyricsKaraokeConstants'
import { colorWithAlpha } from './lyricsKaraokeStyles'
import {
  animateScrollTop,
  cancelScrollAnimation,
  getAnchoredScrollTop,
  getScrollEndPadding,
} from './lyricsScroll'
import {
  getLineFinishedTime,
  getLineLifecycleState,
  getLineRenderWindow,
  getStrictActiveLineIndex,
} from './lyricsTiming'
import usePlaybackClock from './usePlaybackClock'

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
  },
  line: {
    display: 'inline-block',
    maxWidth: '100%',
    fontWeight: 700,
    fontSize: 24,
    lineHeight: 1.18,
    overflowWrap: 'anywhere',
    whiteSpace: 'pre-wrap',
    letterSpacing: 0,
    transition: `opacity ${KARAOKE_ANIMATION_MS}ms ${KARAOKE_EASING}, color ${KARAOKE_ANIMATION_MS}ms ${KARAOKE_EASING}`,
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
    transition: `opacity ${KARAOKE_ANIMATION_MS}ms ${KARAOKE_EASING}, color ${KARAOKE_ANIMATION_MS}ms ${KARAOKE_EASING}`,
    '@media (prefers-reduced-motion: reduce)': {
      transition: 'none',
    },
  },
  stackedToken: {
    display: 'inline-flex',
    flexDirection: 'column',
    alignItems: 'flex-start',
    verticalAlign: 'top',
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
  },
  translationLine: {
    fontWeight: 600,
  },
  token: {
    whiteSpace: 'pre-wrap',
    overflowWrap: 'anywhere',
    transition: `color ${KARAOKE_ANIMATION_MS}ms ${KARAOKE_EASING}`,
    '@media (prefers-reduced-motion: reduce)': {
      transition: 'none',
    },
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
    color: colorWithAlpha(theme.palette.text.primary, 0.68),
    fontWeight: 600,
    textAlign: 'center',
  },
}))

const normalizeLineText = (value) =>
  (value || '').trim().replace(/\s+/g, ' ').toLowerCase()

const shouldShowAuxLine = (mainLine, auxLine) =>
  Boolean(
    auxLine?.value &&
    normalizeLineText(auxLine.value) !== normalizeLineText(mainLine?.value),
  )

const getLineLanes = (line) =>
  Array.isArray(line?.lanes) && line.lanes.length > 0 ? line.lanes : [line]

const usePrefersReducedMotion = () => {
  const [prefersReducedMotion, setPrefersReducedMotion] = useState(false)

  useEffect(() => {
    if (typeof window === 'undefined' || !window.matchMedia) {
      return undefined
    }

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
}) => {
  const classes = useStyles()
  const theme = useTheme()
  const bodyRef = useRef(null)
  const scrollTargetLineRef = useRef(null)
  const scrollAnimationRef = useRef(null)
  const scrollbarTimerRef = useRef(null)
  const manualScrollTimerRef = useRef(null)
  const manualScrollUntilRef = useRef(0)
  const [showScrollbar, setShowScrollbar] = useState(false)
  const [autoScrollResumeKey, setAutoScrollResumeKey] = useState(0)
  const [layoutVersion, setLayoutVersion] = useState(0)
  const [hasTopFade, setHasTopFade] = useState(false)
  const [scrollEndPadding, setScrollEndPadding] = useState(0)
  const playbackMs = usePlaybackClock(visible, audioInstance)
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
  const renderPlaybackMs = playbackMs + KARAOKE_HIGHLIGHT_LEAD_MS
  const activeIndex = useMemo(
    () =>
      hasTimedMainLines ? getStrictActiveLineIndex(mainLines, playbackMs) : -1,
    [hasTimedMainLines, mainLines, playbackMs],
  )
  const lineLifecycleStates = useMemo(
    () =>
      mainLines.map((line, idx) =>
        hasTimedMainLines
          ? getLineLifecycleState(
              line,
              mainLines[idx + 1]?.start ?? null,
              playbackMs,
            )
          : {
              phase: 'idle',
              isActive: false,
              isRelease: false,
              isAnimating: false,
              highlightAlphaScale: 0,
              lineFocusScale: 0,
            },
      ),
    [hasTimedMainLines, mainLines, playbackMs],
  )
  const scrollTargetIndex = useMemo(() => {
    if (!hasTimedMainLines || mainLines.length === 0) return -1

    for (let i = 0; i < mainLines.length; i += 1) {
      const { start } = getLineRenderWindow(
        mainLines[i],
        mainLines[i + 1]?.start ?? null,
      )
      if (
        start != null &&
        playbackMs >= start - KARAOKE_SCROLL_PRE_ROLL_MS &&
        playbackMs < start
      ) {
        return i
      }
    }

    if (activeIndex >= 0) return activeIndex

    let latestStartedIndex = -1
    for (let i = 0; i < mainLines.length; i += 1) {
      const nextLineStart = mainLines[i + 1]?.start ?? null
      const { start } = getLineRenderWindow(mainLines[i], nextLineStart)
      if (start == null) continue
      if (playbackMs < start) break
      latestStartedIndex = i
    }

    if (latestStartedIndex >= 0) {
      const nextLineStart = mainLines[latestStartedIndex + 1]?.start ?? null
      const finishedTime = getLineFinishedTime(
        mainLines[latestStartedIndex],
        nextLineStart,
      )
      if (
        finishedTime != null &&
        playbackMs >= finishedTime &&
        latestStartedIndex + 1 < mainLines.length
      ) {
        const nextIndex = latestStartedIndex + 1
        const nextNextLineStart = mainLines[nextIndex + 1]?.start ?? null
        const { start: nextStart } = getLineRenderWindow(
          mainLines[nextIndex],
          nextNextLineStart,
        )
        if (
          nextStart != null &&
          playbackMs >= nextStart - KARAOKE_SCROLL_PRE_ROLL_MS
        ) {
          return nextIndex
        }
      }
    }

    return latestStartedIndex
  }, [activeIndex, hasTimedMainLines, mainLines, playbackMs])
  const trByMainIndex = useMemo(() => {
    if (!showTranslation || translationLines.length === 0) return {}
    const map = {}
    for (let i = 0; i < mainLines.length; i += 1) {
      const { line } = resolveLayerLineForMain(mainLines, translationLines, i)
      if (line) map[i] = line
    }
    return map
  }, [mainLines, translationLines, showTranslation])

  const prByMainIndex = useMemo(() => {
    if (!showPronunciation || pronunciationLines.length === 0) return {}
    const map = {}
    for (let i = 0; i < mainLines.length; i += 1) {
      const { line } = resolveLayerLineForMain(mainLines, pronunciationLines, i)
      if (line) map[i] = line
    }
    return map
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
    if (!visible || !body || !hasTimedMainLines) {
      setScrollEndPadding(0)
      return
    }

    const nextPadding = getScrollEndPadding(body, activeLineAnchorRatio)
    setScrollEndPadding((current) =>
      current === nextPadding ? current : nextPadding,
    )
  }, [
    activeLineAnchorRatio,
    hasTimedMainLines,
    layoutVersion,
    mainLines.length,
    visible,
  ])

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
    if (!visible || !hasTimedMainLines) {
      cancelScrollAnimation(scrollAnimationRef)
      return undefined
    }

    let animFrameId = window.requestAnimationFrame(() => {
      if (manualScrollUntilRef.current > 0) return

      const body = bodyRef.current
      const targetNode = scrollTargetLineRef.current
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

    return () => {
      if (animFrameId) window.cancelAnimationFrame(animFrameId)
    }
  }, [
    autoScrollResumeKey,
    activeLineAnchorRatio,
    hasTimedMainLines,
    layoutVersion,
    prefersReducedMotion,
    scrollTargetIndex,
    visible,
  ])

  if (!visible) {
    return null
  }

  if (!hasStructuredLyricContent(mainLyric) || mainLines.length === 0) {
    const message = loading
      ? 'Loading lyrics'
      : error
        ? 'Lyrics unavailable'
        : 'No lyrics available'

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

  const getLayerActiveAlpha = (layer) => {
    if (layer === 'main') return 0.98
    if (layer === 'translation') return 0.72
    return 0.78
  }

  const getLayerIdleAlpha = (layer) => {
    if (layer === 'main') return 0.46
    if (layer === 'translation') return 0.34
    return 0.38
  }

  const getLineFocusScale = (idx) =>
    clamp(lineLifecycleStates[idx]?.lineFocusScale ?? 0, 0, 1)

  const getLineStyle = (idx, layer) => {
    const sourceColor = colors[layer]
    if (!hasTimedMainLines) {
      return {
        opacity: 1,
        color: colorWithAlpha(sourceColor, layer === 'main' ? 0.98 : 0.86),
      }
    }

    const focusScale = getLineFocusScale(idx)
    const colorAlpha = lerp(
      getLayerIdleAlpha(layer),
      getLayerActiveAlpha(layer),
      focusScale,
    )

    return {
      opacity: 1,
      color: colorWithAlpha(sourceColor, colorAlpha),
    }
  }

  const seekToLine = (line) => {
    if (!audioInstance || line.start == null) return
    audioInstance.currentTime = line.start / 1000
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
        onPointerDown={markManualScrollIntent}
        onTouchStart={markManualScrollIntent}
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
            const trLine = trByMainIndex[idx]
            const prLine = prByMainIndex[idx]
            const mainNextLineStart = mainLines[idx + 1]?.start ?? null
            const showTr = shouldShowAuxLine(line, trLine)
            const showPr = shouldShowAuxLine(line, prLine)
            const mainLineStyle = getLineStyle(idx, 'main')
            const pronunciationStyle = getLineStyle(idx, 'pronunciation')
            const lifecycle = lineLifecycleStates[idx] || {}
            const isActiveLine = Boolean(lifecycle.isActive)
            const shouldHighlightTokens =
              hasTimedMainLines && Boolean(lifecycle.isAnimating)
            const highlightAlphaScale = lifecycle.highlightAlphaScale ?? 0
            const lineLanes = getLineLanes(line)
            const canSeekLine = Boolean(audioInstance && line.start != null)
            return (
              <div
                key={`line-${line.index}-${line.start ?? idx}`}
                ref={
                  idx === scrollTargetIndex && hasTimedMainLines
                    ? scrollTargetLineRef
                    : null
                }
                className={classes.lineGroup}
                data-active={isActiveLine ? 'true' : 'false'}
                data-lifecycle={lifecycle.phase || 'idle'}
                data-highlight-active={shouldHighlightTokens ? 'true' : 'false'}
                aria-current={idx === activeIndex ? 'true' : undefined}
                data-scroll-target={
                  idx === scrollTargetIndex ? 'true' : 'false'
                }
                data-testid="lyrics-line-group"
                style={{
                  cursor: canSeekLine ? 'pointer' : undefined,
                }}
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
                      const highlightedLane = buildHighlightedMainLine(lane)

                      return showPr && laneIdx === 0 ? (
                        <KaraokeStackedLineRow
                          key={lane.key || `lane-${laneIdx}`}
                          line={highlightedLane}
                          pronunciationLine={buildHighlightedAuxLine(
                            line,
                            prLine,
                          )}
                          pronunciationStyle={pronunciationStyle}
                          nextLineStart={mainNextLineStart}
                          renderPlaybackMs={renderPlaybackMs}
                          className={laneClassName}
                          style={mainLineStyle}
                          tokenClassName={classes.token}
                          classes={classes}
                          highlightTokens={shouldHighlightTokens}
                          highlightAlphaScale={highlightAlphaScale}
                          testId="lyrics-voice-lane"
                        />
                      ) : (
                        <KaraokeLineRow
                          key={lane.key || `lane-${laneIdx}`}
                          line={highlightedLane}
                          nextLineStart={mainNextLineStart}
                          renderPlaybackMs={renderPlaybackMs}
                          className={laneClassName}
                          style={mainLineStyle}
                          tokenClassName={classes.token}
                          highlightTokens={shouldHighlightTokens}
                          highlightAlphaScale={highlightAlphaScale}
                          testId="lyrics-voice-lane"
                        />
                      )
                    })}
                  </div>
                ) : showPr ? (
                  <KaraokeStackedLineRow
                    line={buildHighlightedMainLine(line)}
                    pronunciationLine={buildHighlightedAuxLine(line, prLine)}
                    pronunciationStyle={pronunciationStyle}
                    nextLineStart={mainNextLineStart}
                    renderPlaybackMs={renderPlaybackMs}
                    className={clsx(classes.line, {
                      [classes.inlineLine]: inline,
                    })}
                    style={mainLineStyle}
                    tokenClassName={classes.token}
                    classes={classes}
                    highlightTokens={shouldHighlightTokens}
                    highlightAlphaScale={highlightAlphaScale}
                  />
                ) : (
                  <KaraokeLineRow
                    line={buildHighlightedMainLine(line)}
                    nextLineStart={mainNextLineStart}
                    renderPlaybackMs={renderPlaybackMs}
                    className={clsx(classes.line, {
                      [classes.inlineLine]: inline,
                    })}
                    style={mainLineStyle}
                    tokenClassName={classes.token}
                    highlightTokens={shouldHighlightTokens}
                    highlightAlphaScale={highlightAlphaScale}
                  />
                )}
                {showTr && (
                  <KaraokeLineRow
                    line={buildHighlightedAuxLine(line, trLine)}
                    nextLineStart={null}
                    renderPlaybackMs={renderPlaybackMs}
                    className={clsx(classes.auxLine, classes.translationLine)}
                    style={getLineStyle(idx, 'translation')}
                    tokenClassName={classes.token}
                    highlightTokens={shouldHighlightTokens}
                    highlightAlphaScale={highlightAlphaScale}
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
