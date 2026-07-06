import Typography from '@material-ui/core/Typography'
import { makeStyles, useTheme } from '@material-ui/core/styles'
import clsx from 'clsx'
import React, {
  memo,
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
  resolveKaraokeTokenWindow,
  resolveLayerLineForMain,
} from './lyrics'
import { buildSegmentsFromLine } from './lyricsSegments'

const KARAOKE_CLOCK_DRIFT_RESET_MS = 140
const KARAOKE_CLOCK_RESET_THRESHOLD_MS = 320
const KARAOKE_MONOTONIC_JITTER_MS = 60
const KARAOKE_RENDER_UPDATE_EPSILON_MS = 6
const KARAOKE_HIGHLIGHT_LEAD_MS = 85
const KARAOKE_ANIMATION_MS = 150
const KARAOKE_SCROLLBAR_VISIBLE_MS = 1400
const KARAOKE_MANUAL_SCROLL_PAUSE_MS = 2200
const KARAOKE_WORD_SETTLE_MS = 96
const KARAOKE_LINE_RELEASE_MS = 180
const KARAOKE_LINE_INCOMING_MS = 120
const KARAOKE_SCROLL_PRE_ROLL_MS = 220
const KARAOKE_DESKTOP_ACTIVE_LINE_ANCHOR_RATIO = 0.3
const KARAOKE_INLINE_ACTIVE_LINE_ANCHOR_RATIO = 0.42
const KARAOKE_SCROLL_SETTLE_PX = 2
const KARAOKE_SCROLL_DURATION_MS = 400
const KARAOKE_AUX_LINE_HEIGHT = 1.18
const KARAOKE_EASING = 'cubic-bezier(0.22, 1, 0.36, 1)'
const TOKEN_DONE_ALPHA = 1
const TOKEN_FUTURE_ALPHA = 0.34
const TOKEN_ACTIVE_ALPHA = 1
const TOKEN_WIPE_SOFT_SPREAD_PCT = 12
const TOKEN_WIPE_EDGE_PCT = 8
const TOKEN_SHORT_DURATION_MS = 180
const EMPHASIS_ROLES = new Set([
  'adlib',
  'backing',
  'backing vocals',
  'backing-vocals',
  'background',
  'background vocals',
  'background-vocals',
  'bg',
  'choir',
  'chorus',
  'group',
  'harmony',
])

const clamp = (value, min, max) => Math.min(max, Math.max(min, value))
const lerp = (from, to, progress) => from + (to - from) * progress
const easeInOut = (value) => {
  const clamped = clamp(value, 0, 1)
  return clamped < 0.5 ? 2 * clamped * clamped : 1 - (-2 * clamped + 2) ** 2 / 2
}
const easeOutCubic = (value) => 1 - (1 - clamp(value, 0, 1)) ** 3

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
    padding: theme.spacing(0, 2.25, 3.25),
    paddingTop: 'clamp(72px, 10vh, 128px)',
    overscrollBehavior: 'contain',
    scrollbarWidth: 'none',
    msOverflowStyle: 'none',
    maskImage:
      'linear-gradient(to bottom, #000 0, #000 calc(100% - 76px), rgba(0, 0, 0, 0.15) calc(100% - 22px), transparent 100%)',
    WebkitMaskImage:
      'linear-gradient(to bottom, #000 0, #000 calc(100% - 76px), rgba(0, 0, 0, 0.15) calc(100% - 22px), transparent 100%)',
    '&::-webkit-scrollbar': {
      width: 0,
      height: 0,
    },
  },
  bodyTopFade: {
    maskImage:
      'linear-gradient(to bottom, transparent 0, rgba(0, 0, 0, 0.15) 16px, #000 56px, #000 calc(100% - 76px), rgba(0, 0, 0, 0.15) calc(100% - 22px), transparent 100%)',
    WebkitMaskImage:
      'linear-gradient(to bottom, transparent 0, rgba(0, 0, 0, 0.15) 16px, #000 56px, #000 calc(100% - 76px), rgba(0, 0, 0, 0.15) calc(100% - 22px), transparent 100%)',
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
    gap: theme.spacing(3.5),
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
    transformOrigin: 'left center',
    transition: `opacity ${KARAOKE_ANIMATION_MS}ms ${KARAOKE_EASING}, color ${KARAOKE_ANIMATION_MS}ms ${KARAOKE_EASING}, transform ${KARAOKE_ANIMATION_MS}ms ${KARAOKE_EASING}`,
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
    transformOrigin: 'left center',
    transition: `opacity ${KARAOKE_ANIMATION_MS}ms ${KARAOKE_EASING}, color ${KARAOKE_ANIMATION_MS}ms ${KARAOKE_EASING}, transform ${KARAOKE_ANIMATION_MS}ms ${KARAOKE_EASING}`,
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

const parseColorRGB = (color) => {
  const hex = (color || '').match(/^#([0-9a-f]{3}|[0-9a-f]{6})$/i)
  if (hex) {
    const raw = hex[1]
    if (raw.length === 3) {
      return raw.split('').map((part) => parseInt(part + part, 16))
    }
    return [
      parseInt(raw.slice(0, 2), 16),
      parseInt(raw.slice(2, 4), 16),
      parseInt(raw.slice(4, 6), 16),
    ]
  }

  const rgb = (color || '').match(/rgba?\((\d+),\s*(\d+),\s*(\d+)/)
  return rgb
    ? [parseInt(rgb[1]), parseInt(rgb[2]), parseInt(rgb[3])]
    : [255, 255, 255]
}

const colorWithAlpha = (color, alpha) => {
  const [r, g, b] = parseColorRGB(color)
  return `rgba(${r}, ${g}, ${b}, ${alpha})`
}

const toFinitePlaybackMs = (value) =>
  Number.isFinite(Number(value)) ? Number(value) : 0

const normalizeRole = (role) =>
  String(role || '')
    .trim()
    .toLowerCase()

const isEmphasisRole = (token) =>
  EMPHASIS_ROLES.has(normalizeRole(token?.role)) ||
  EMPHASIS_ROLES.has(normalizeRole(token?.agentRole))

const buildEmphasisStyle = (token) =>
  isEmphasisRole(token) ? { fontStyle: 'italic' } : undefined

const getLineRenderWindow = (line, nextLineStart) => {
  let start = Number.isFinite(Number(line?.start)) ? Number(line.start) : null
  let end = Number.isFinite(Number(line?.end)) ? Number(line.end) : null
  const fallbackEnd = Number.isFinite(Number(nextLineStart))
    ? Number(nextLineStart)
    : null

  if (end == null) end = fallbackEnd

  const tokens = Array.isArray(line?.tokens) ? line.tokens : []
  if (tokens.length > 0) {
    const firstWindow = resolveKaraokeTokenWindow(line, 0, nextLineStart)
    const lastWindow = resolveKaraokeTokenWindow(
      line,
      tokens.length - 1,
      nextLineStart,
    )

    if (
      firstWindow.start != null &&
      (start == null || firstWindow.start < start)
    ) {
      start = firstWindow.start
    }
    if (lastWindow.end != null && (end == null || lastWindow.end > end)) {
      end = lastWindow.end
    }
  }

  return { start, end }
}

const shouldSkipLineFrame = (
  prevPlaybackMs,
  nextPlaybackMs,
  line,
  nextLineStart,
) => {
  if (prevPlaybackMs === nextPlaybackMs) return true

  const { start, end } = getLineRenderWindow(line, nextLineStart)
  if (start != null) {
    const activationStart = start - KARAOKE_SCROLL_PRE_ROLL_MS
    if (prevPlaybackMs < activationStart && nextPlaybackMs < activationStart) {
      return true
    }
  }

  if (end != null) {
    const settleEnd = end + KARAOKE_LINE_RELEASE_MS + 80
    if (prevPlaybackMs > settleEnd && nextPlaybackMs > settleEnd) {
      return true
    }
  }

  return false
}

const areLineStylesEqual = (prevStyle, nextStyle) => {
  const a = prevStyle || {}
  const b = nextStyle || {}
  return (
    a.opacity === b.opacity &&
    a.color === b.color &&
    a.fontSize === b.fontSize &&
    a.fontWeight === b.fontWeight &&
    a.lineHeight === b.lineHeight &&
    a.maxWidth === b.maxWidth &&
    a.fontStyle === b.fontStyle &&
    a.whiteSpace === b.whiteSpace &&
    a.transform === b.transform
  )
}

const buildTokenWipeStyle = ({
  fillProgress,
  highlightAlpha,
  futureAlpha,
  rgb,
  useCrossfade = false,
}) => {
  const [r, g, b] = rgb || [255, 255, 255]
  const fillPct = clamp(fillProgress, 0, 1) * 100
  const doneColor = `rgba(${r}, ${g}, ${b}, ${clamp(
    highlightAlpha,
    futureAlpha,
    TOKEN_ACTIVE_ALPHA,
  )})`
  const futureColor = `rgba(${r}, ${g}, ${b}, ${futureAlpha})`

  if (fillPct <= 0) {
    return { color: futureColor }
  }

  if (useCrossfade) {
    return {
      color: doneColor,
      WebkitTextFillColor: doneColor,
    }
  }

  const edgeStart = clamp(fillPct - TOKEN_WIPE_EDGE_PCT, 0, 100)
  const softEnd = clamp(fillPct + TOKEN_WIPE_SOFT_SPREAD_PCT, 0, 100)
  return {
    color: 'transparent',
    WebkitTextFillColor: 'transparent',
    backgroundImage: `linear-gradient(90deg, ${doneColor} 0%, ${doneColor} ${edgeStart}%, ${doneColor} ${fillPct}%, ${futureColor} ${softEnd}%, ${futureColor} 100%)`,
    backgroundSize: '100% 100%',
    backgroundClip: 'text',
    WebkitBackgroundClip: 'text',
  }
}

const buildSegmentTokenStyle = ({
  segment,
  line,
  nextLineStart,
  renderPlaybackMs,
  rgb,
  highlightAlphaScale = 1,
}) => {
  const { start: tokenStart, end: tokenEnd } = resolveKaraokeTokenWindow(
    line,
    segment.tokenIndex,
    nextLineStart,
  )
  const previousWindow =
    segment.tokenIndex > 0
      ? resolveKaraokeTokenWindow(line, segment.tokenIndex - 1, nextLineStart)
      : null
  const visualStart =
    tokenStart != null && previousWindow?.end != null
      ? Math.max(tokenStart, previousWindow.end)
      : tokenStart
  const visualEnd =
    tokenEnd != null && visualStart != null && tokenEnd <= visualStart
      ? visualStart + 1
      : tokenEnd
  const isDone = tokenEnd != null ? renderPlaybackMs >= tokenEnd : false
  const isActive =
    !isDone && visualStart != null && renderPlaybackMs >= visualStart
  const progress =
    isDone ||
    visualStart == null ||
    visualEnd == null ||
    visualEnd <= visualStart
      ? isDone
        ? 1
        : 0
      : clamp(
          (renderPlaybackMs - visualStart) / (visualEnd - visualStart),
          0,
          1,
        )
  const justEnded =
    tokenEnd != null &&
    renderPlaybackMs > tokenEnd &&
    renderPlaybackMs <= tokenEnd + KARAOKE_WORD_SETTLE_MS
  const settleProgress =
    justEnded && tokenEnd != null
      ? clamp((renderPlaybackMs - tokenEnd) / KARAOKE_WORD_SETTLE_MS, 0, 1)
      : 0

  let alpha = TOKEN_FUTURE_ALPHA
  if (isDone) {
    alpha = TOKEN_DONE_ALPHA
  } else if (isActive) {
    alpha = lerp(TOKEN_FUTURE_ALPHA, TOKEN_ACTIVE_ALPHA, easeInOut(progress))
  }
  if (justEnded) {
    alpha = lerp(
      TOKEN_ACTIVE_ALPHA,
      TOKEN_DONE_ALPHA,
      easeInOut(settleProgress),
    )
  }
  alpha = clamp(alpha, TOKEN_FUTURE_ALPHA, TOKEN_ACTIVE_ALPHA)
  alpha = lerp(TOKEN_FUTURE_ALPHA, alpha, clamp(highlightAlphaScale, 0, 1))
  const fillProgress = isDone ? 1 : isActive ? progress : 0
  const isEmphasis = isEmphasisRole(segment.token)
  const futureAlpha = isEmphasis
    ? TOKEN_FUTURE_ALPHA * 0.72
    : TOKEN_FUTURE_ALPHA
  const tokenDuration =
    visualStart != null && visualEnd != null ? visualEnd - visualStart : null
  const useCrossfade =
    (tokenDuration != null && tokenDuration <= TOKEN_SHORT_DURATION_MS) ||
    segment.text.trim().length <= 2

  if (!isActive && !isDone) {
    const [r, g, b] = rgb || [255, 255, 255]
    return {
      tokenStart,
      style: {
        color: `rgba(${r}, ${g}, ${b}, ${futureAlpha})`,
        ...buildEmphasisStyle(segment.token),
      },
    }
  }

  return {
    tokenStart,
    style: {
      ...buildTokenWipeStyle({
        fillProgress,
        highlightAlpha: isEmphasis ? alpha * 0.72 : alpha,
        futureAlpha,
        rgb,
        useCrossfade,
      }),
      ...buildEmphasisStyle(segment.token),
    },
  }
}

const KaraokeLineRow = memo(
  ({
    line,
    nextLineStart,
    renderPlaybackMs,
    className,
    style,
    tokenClassName,
    highlightTokens = true,
    highlightAlphaScale = 1,
    testId,
  }) => {
    const segments = useMemo(() => buildSegmentsFromLine(line), [line])
    const tokenRGB = useMemo(
      () => (style?.color ? parseColorRGB(style.color) : [255, 255, 255]),
      [style?.color],
    )
    const lineStyle = useMemo(
      () => ({ ...style, ...buildEmphasisStyle(line) }),
      [line, style],
    )

    return (
      <Typography
        className={className}
        component="div"
        data-testid={testId}
        style={lineStyle}
      >
        {segments.map((segment, idx) => {
          if (!segment.token)
            return <span key={`text-${idx}`}>{segment.text}</span>
          if (!highlightTokens) {
            return (
              <span
                key={`token-plain-${idx}`}
                className={tokenClassName}
                style={buildEmphasisStyle(segment.token)}
              >
                {segment.text}
              </span>
            )
          }

          const { tokenStart, style: tokenStyle } = buildSegmentTokenStyle({
            segment,
            line,
            nextLineStart,
            renderPlaybackMs,
            rgb: tokenRGB,
            highlightAlphaScale,
          })

          return (
            <span
              key={`token-${idx}-${tokenStart ?? 'na'}`}
              className={tokenClassName}
              data-testid="lyrics-token"
              style={tokenStyle}
            >
              {segment.text}
            </span>
          )
        })}
      </Typography>
    )
  },
  (prevProps, nextProps) => {
    if (
      prevProps.line !== nextProps.line ||
      prevProps.nextLineStart !== nextProps.nextLineStart ||
      prevProps.className !== nextProps.className ||
      prevProps.tokenClassName !== nextProps.tokenClassName ||
      prevProps.highlightTokens !== nextProps.highlightTokens ||
      prevProps.highlightAlphaScale !== nextProps.highlightAlphaScale ||
      prevProps.testId !== nextProps.testId ||
      !areLineStylesEqual(prevProps.style, nextProps.style)
    ) {
      return false
    }

    return shouldSkipLineFrame(
      prevProps.renderPlaybackMs,
      nextProps.renderPlaybackMs,
      nextProps.line,
      nextProps.nextLineStart,
    )
  },
)

KaraokeLineRow.displayName = 'KaraokeLineRow'

const splitTextSegments = (value) =>
  (value || '')
    .split(/(\s+)/)
    .filter((text) => text.length > 0)
    .map((text) => ({
      text,
      token: null,
      tokenIndex: -1,
      isWhitespace: /^\s+$/.test(text),
    }))

const buildPronunciationParts = (line) => {
  if (!line?.value) return []

  const segments = buildSegmentsFromLine(line)
  const tokenParts = segments
    .filter((segment) => segment.token && segment.text.trim())
    .map((segment) => ({
      text: segment.text.trim(),
      segment,
    }))

  if (tokenParts.length > 0) return tokenParts

  return splitTextSegments(line.value)
    .filter((segment) => !segment.isWhitespace)
    .map((segment) => ({
      text: segment.text.trim(),
      segment: null,
    }))
}

const buildStackedPronunciationSegments = (line, pronunciationLine) => {
  const lineSegments = buildSegmentsFromLine(line)
  const hasTokenSegments = lineSegments.some((segment) => segment.token)
  const mainSegments = hasTokenSegments
    ? lineSegments
    : splitTextSegments(line?.value || '')
  const pronunciationParts = buildPronunciationParts(pronunciationLine)
  const pairableSegments = mainSegments.filter(
    (segment) =>
      segment.token ||
      (!hasTokenSegments && !segment.isWhitespace && segment.text.trim()),
  )

  if (
    !hasTokenSegments &&
    pronunciationParts.length > 0 &&
    pairableSegments.length > 0 &&
    pronunciationParts.length !== pairableSegments.length
  ) {
    return [
      {
        text: line?.value || '',
        token: null,
        tokenIndex: -1,
        isWhitespace: false,
        pronunciation: pronunciationLine?.value?.trim() || '',
        pronunciationSegment: null,
      },
    ]
  }

  let pronunciationIndex = 0

  return mainSegments.map((segment) => {
    const canPair =
      segment.token ||
      (!hasTokenSegments && !segment.isWhitespace && segment.text.trim())

    if (!canPair) return { ...segment, pronunciation: '' }

    const pronunciationPart = pronunciationParts[pronunciationIndex] || null
    pronunciationIndex += 1
    return {
      ...segment,
      pronunciation: pronunciationPart?.text || '',
      pronunciationSegment: pronunciationPart?.segment || null,
    }
  })
}

const KaraokeStackedLineRow = memo(
  ({
    line,
    pronunciationLine,
    pronunciationStyle,
    nextLineStart,
    renderPlaybackMs,
    className,
    style,
    tokenClassName,
    classes,
    highlightTokens = true,
    highlightAlphaScale = 1,
    testId,
  }) => {
    const rowRef = useRef(null)
    const [isWrapped, setIsWrapped] = useState(false)
    const segments = useMemo(
      () => buildStackedPronunciationSegments(line, pronunciationLine),
      [line, pronunciationLine],
    )
    const tokenRGB = useMemo(
      () => (style?.color ? parseColorRGB(style.color) : [255, 255, 255]),
      [style?.color],
    )
    const pronunciationRGB = useMemo(
      () =>
        pronunciationStyle?.color
          ? parseColorRGB(pronunciationStyle.color)
          : [255, 255, 255],
      [pronunciationStyle?.color],
    )
    const lineStyle = useMemo(
      () => ({ ...style, ...buildEmphasisStyle(line) }),
      [line, style],
    )

    useLayoutEffect(() => {
      const row = rowRef.current
      if (!row) return undefined

      const updateWrappedState = () => {
        const tokenRows = new Set(
          Array.from(row.querySelectorAll('[data-stacked-token="true"]')).map(
            (node) => node.offsetTop,
          ),
        )
        const wrapped = tokenRows.size > 1
        setIsWrapped((current) => (current === wrapped ? current : wrapped))
      }

      updateWrappedState()

      const ResizeObserverConstructor =
        typeof window !== 'undefined' ? window.ResizeObserver : null
      if (!ResizeObserverConstructor) return undefined

      const resizeObserver = new ResizeObserverConstructor(updateWrappedState)
      resizeObserver.observe(row)
      return () => resizeObserver.disconnect()
    }, [segments])

    return (
      <Typography
        className={clsx(className, {
          [classes.wrappedStackedLine]: isWrapped,
        })}
        component="div"
        data-wrapped={isWrapped ? 'true' : 'false'}
        data-testid={testId}
        ref={rowRef}
        style={lineStyle}
      >
        {segments.map((segment, idx) => {
          if (!segment.pronunciation) {
            return (
              <span
                key={`text-${idx}`}
                style={buildEmphasisStyle(segment.token)}
              >
                {segment.text}
              </span>
            )
          }

          const tokenData =
            segment.token && highlightTokens
              ? buildSegmentTokenStyle({
                  segment,
                  line,
                  nextLineStart,
                  renderPlaybackMs,
                  rgb: tokenRGB,
                  highlightAlphaScale,
                })
              : null
          const pronunciationTokenData =
            highlightTokens && segment.pronunciationSegment?.token
              ? buildSegmentTokenStyle({
                  segment: segment.pronunciationSegment,
                  line: pronunciationLine,
                  nextLineStart: null,
                  renderPlaybackMs,
                  rgb: pronunciationRGB,
                  highlightAlphaScale,
                })
              : highlightTokens && segment.token
                ? buildSegmentTokenStyle({
                    segment,
                    line,
                    nextLineStart,
                    renderPlaybackMs,
                    rgb: pronunciationRGB,
                    highlightAlphaScale,
                  })
                : null

          return (
            <span
              key={`stacked-${idx}-${tokenData?.tokenStart ?? segment.text}`}
              className={classes.stackedToken}
              data-stacked-token={segment.pronunciation ? 'true' : undefined}
            >
              <span
                className={clsx(tokenClassName, classes.stackedMainText)}
                data-testid={segment.token ? 'lyrics-token' : undefined}
                style={tokenData?.style || buildEmphasisStyle(segment.token)}
              >
                {segment.text}
              </span>
              <span
                className={classes.stackedPronunciation}
                data-testid="lyrics-pronunciation-token"
                style={
                  pronunciationTokenData?.style || {
                    color: pronunciationStyle?.color,
                    ...buildEmphasisStyle(
                      segment.pronunciationSegment?.token || segment.token,
                    ),
                  }
                }
              >
                {segment.pronunciation}
              </span>
            </span>
          )
        })}
      </Typography>
    )
  },
  (prevProps, nextProps) => {
    if (
      prevProps.line !== nextProps.line ||
      prevProps.pronunciationLine !== nextProps.pronunciationLine ||
      prevProps.nextLineStart !== nextProps.nextLineStart ||
      prevProps.className !== nextProps.className ||
      prevProps.tokenClassName !== nextProps.tokenClassName ||
      prevProps.highlightTokens !== nextProps.highlightTokens ||
      prevProps.highlightAlphaScale !== nextProps.highlightAlphaScale ||
      prevProps.testId !== nextProps.testId ||
      prevProps.classes !== nextProps.classes ||
      !areLineStylesEqual(prevProps.style, nextProps.style) ||
      !areLineStylesEqual(
        prevProps.pronunciationStyle,
        nextProps.pronunciationStyle,
      )
    ) {
      return false
    }

    return shouldSkipLineFrame(
      prevProps.renderPlaybackMs,
      nextProps.renderPlaybackMs,
      nextProps.line,
      nextProps.nextLineStart,
    )
  },
)

KaraokeStackedLineRow.displayName = 'KaraokeStackedLineRow'

const normalizeLineText = (value) =>
  (value || '').trim().replace(/\s+/g, ' ').toLowerCase()

const shouldShowAuxLine = (mainLine, auxLine) =>
  Boolean(
    auxLine?.value &&
    normalizeLineText(auxLine.value) !== normalizeLineText(mainLine?.value),
  )

const getLineLanes = (line) =>
  Array.isArray(line?.lanes) && line.lanes.length > 0 ? line.lanes : [line]

const getLineFinishedTime = (line, nextLineStart) => {
  const tokens = Array.isArray(line?.tokens) ? line.tokens : []
  for (let i = tokens.length - 1; i >= 0; i -= 1) {
    const { end } = resolveKaraokeTokenWindow(line, i, nextLineStart)
    if (end != null) return end
  }

  return getLineRenderWindow(line, nextLineStart).end
}

const getLineLifecycleState = (line, nextLineStart, currentTimeMs) => {
  const current = toFinitePlaybackMs(currentTimeMs)
  const { start } = getLineRenderWindow(line, nextLineStart)
  const end = getLineFinishedTime(line, nextLineStart)

  if (start == null) {
    return {
      phase: 'idle',
      isActive: false,
      isRelease: false,
      isIncoming: false,
      isAnimating: false,
      highlightAlphaScale: 0,
    }
  }

  const hasKnownEnd = end != null
  const isActive = current >= start && (!hasKnownEnd || current < end)
  const isRelease =
    hasKnownEnd && current >= end && current < end + KARAOKE_LINE_RELEASE_MS
  const isIncoming =
    current >= start - KARAOKE_LINE_INCOMING_MS && current < start
  const releaseProgress =
    isRelease && hasKnownEnd
      ? clamp((current - end) / KARAOKE_LINE_RELEASE_MS, 0, 1)
      : 0

  return {
    phase: isActive
      ? 'active'
      : isRelease
        ? 'release'
        : isIncoming
          ? 'incoming'
          : 'idle',
    isActive,
    isRelease,
    isIncoming,
    isAnimating: isActive || isRelease,
    highlightAlphaScale: isRelease
      ? 1 - easeInOut(releaseProgress)
      : isActive
        ? 1
        : 0,
  }
}

const getStrictActiveLineIndex = (lines, currentTimeMs) => {
  if (!Array.isArray(lines) || lines.length === 0) return -1

  const current = toFinitePlaybackMs(currentTimeMs)
  let activeIndex = -1

  for (let i = 0; i < lines.length; i += 1) {
    const nextLineStart = lines[i + 1]?.start ?? null
    const { start } = getLineRenderWindow(lines[i], nextLineStart)
    const end = getLineFinishedTime(lines[i], nextLineStart)

    if (start != null && current < start) break
    if (end != null && current >= end) continue
    if (start == null || current >= start) activeIndex = i
  }

  return activeIndex
}

const usePlaybackClock = (visible, audioInstance) => {
  const [playbackMs, setPlaybackMs] = useState(0)

  useEffect(() => {
    if (!visible || !audioInstance) {
      setPlaybackMs(0)
      return undefined
    }

    let rafId = 0
    let cancelled = false
    let anchorAudioMs = 0
    let anchorPerfMs = 0
    let lastRenderMs = 0

    const readPlaybackMs = () => {
      const seconds = Number(audioInstance.currentTime)
      if (!Number.isFinite(seconds) || seconds < 0) return 0
      return seconds * 1000
    }

    const resetAnchor = (perfNow, observedMs) => {
      anchorAudioMs = observedMs
      anchorPerfMs = perfNow
    }

    const tick = () => {
      if (cancelled) return

      const observedMs = readPlaybackMs()
      const perfNow = performance.now()
      const playbackRate = Number(audioInstance.playbackRate)
      const canInterpolate =
        !audioInstance.paused &&
        !audioInstance.seeking &&
        Number.isFinite(playbackRate) &&
        playbackRate > 0
      let nowMs = observedMs

      if (!canInterpolate) {
        resetAnchor(perfNow, observedMs)
      } else if (anchorPerfMs === 0) {
        resetAnchor(perfNow, observedMs)
      } else {
        const predicted =
          anchorAudioMs + (perfNow - anchorPerfMs) * playbackRate
        const drift = observedMs - predicted
        if (Math.abs(drift) > KARAOKE_CLOCK_DRIFT_RESET_MS) {
          nowMs = observedMs
          resetAnchor(perfNow, observedMs)
        } else {
          nowMs = predicted
        }
      }

      const backwardsDrift = lastRenderMs - nowMs
      if (canInterpolate && backwardsDrift > 0) nowMs = lastRenderMs

      if (canInterpolate && backwardsDrift > KARAOKE_CLOCK_RESET_THRESHOLD_MS) {
        resetAnchor(perfNow, observedMs)
      } else if (
        !canInterpolate &&
        backwardsDrift > 0 &&
        backwardsDrift <= KARAOKE_MONOTONIC_JITTER_MS
      ) {
        nowMs = lastRenderMs
      }

      nowMs = Math.max(0, nowMs)
      lastRenderMs = nowMs

      setPlaybackMs((prev) =>
        Math.abs(prev - nowMs) >= KARAOKE_RENDER_UPDATE_EPSILON_MS
          ? nowMs
          : prev,
      )
      rafId = window.requestAnimationFrame(tick)
    }

    const initialMs = readPlaybackMs()
    resetAnchor(performance.now(), initialMs)
    lastRenderMs = initialMs
    setPlaybackMs(initialMs)
    rafId = window.requestAnimationFrame(tick)

    return () => {
      cancelled = true
      if (rafId) window.cancelAnimationFrame(rafId)
    }
  }, [audioInstance, visible])

  return playbackMs
}

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

const cancelScrollAnimation = (scrollAnimationRef) => {
  const animation = scrollAnimationRef.current
  if (animation?.frameId) window.cancelAnimationFrame(animation.frameId)
  scrollAnimationRef.current = null
}

const getScrollEndPadding = (body, anchorRatio) => {
  const height = Number(body?.clientHeight)
  if (!Number.isFinite(height) || height <= 0) return 0
  return Math.round(height * (1 - anchorRatio))
}

const getAnchoredScrollTop = (body, targetNode, anchorRatio) => {
  const bodyRect = body.getBoundingClientRect()
  const targetRect = targetNode.getBoundingClientRect()
  const activeAnchorTop = body.clientHeight * anchorRatio
  const deltaWithinBody = targetRect.top - bodyRect.top - activeAnchorTop
  const maxTop = Math.max(0, body.scrollHeight - body.clientHeight)
  return clamp(body.scrollTop + deltaWithinBody, 0, maxTop)
}

const animateScrollTop = ({
  body,
  targetTop,
  reducedMotion,
  scrollAnimationRef,
}) => {
  cancelScrollAnimation(scrollAnimationRef)

  const startTop = body.scrollTop
  const distance = targetTop - startTop
  if (Math.abs(distance) < KARAOKE_SCROLL_SETTLE_PX) return

  if (reducedMotion) {
    body.scrollTop = targetTop
    return
  }

  const startTime = performance.now()

  const animation = { frameId: 0 }
  const step = (now) => {
    if (scrollAnimationRef.current !== animation) return

    const elapsed = now - startTime
    const progress = Math.min(elapsed / KARAOKE_SCROLL_DURATION_MS, 1)
    const maxTop = Math.max(0, body.scrollHeight - body.clientHeight)
    const nextTargetTop = clamp(targetTop, 0, maxTop)
    body.scrollTop =
      startTop + (nextTargetTop - startTop) * easeOutCubic(progress)

    if (
      progress < 1 &&
      Math.abs(body.scrollTop - nextTargetTop) >= KARAOKE_SCROLL_SETTLE_PX
    ) {
      animation.frameId = window.requestAnimationFrame(step)
      return
    }

    body.scrollTop = nextTargetTop
    scrollAnimationRef.current = null
  }

  scrollAnimationRef.current = animation
  animation.frameId = window.requestAnimationFrame(step)
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
  const manualScrollUntilRef = useRef(0)
  const manualScrollActiveIndexRef = useRef(-1)
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
              isIncoming: false,
              isAnimating: false,
              highlightAlphaScale: 0,
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
      if (start == null || playbackMs < start) break
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
  const lineStyleReferenceIndex =
    activeIndex >= 0 ? activeIndex : scrollTargetIndex

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

  const markManualScrollIntent = useCallback(() => {
    cancelScrollAnimation(scrollAnimationRef)
    manualScrollActiveIndexRef.current = activeIndex
    manualScrollUntilRef.current =
      performance.now() + KARAOKE_MANUAL_SCROLL_PAUSE_MS
    showScrollbarForManualScroll()
  }, [activeIndex, showScrollbarForManualScroll])

  const resumeAutoScroll = useCallback(() => {
    manualScrollUntilRef.current = 0
    manualScrollActiveIndexRef.current = activeIndex
    setAutoScrollResumeKey((current) => current + 1)
  }, [activeIndex])

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
    manualScrollUntilRef.current = 0
    body.scrollTop = 0
    setHasTopFade(false)
  }, [mainLyric, visible])

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

  useEffect(() => {
    if (manualScrollUntilRef.current === 0) return
    if (activeIndex === manualScrollActiveIndexRef.current) return
    if (performance.now() < manualScrollUntilRef.current) return
    resumeAutoScroll()
  }, [activeIndex, resumeAutoScroll])

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

  const getLineStyle = (idx, layer) => {
    const sourceColor = colors[layer]
    if (!hasTimedMainLines) {
      return {
        opacity: layer === 'main' ? 1 : 0.9,
        color: colorWithAlpha(sourceColor, layer === 'main' ? 0.98 : 0.86),
      }
    }

    const referenceIndex =
      lineStyleReferenceIndex >= 0 ? lineStyleReferenceIndex : idx
    const delta = idx - referenceIndex
    const lifecycle = lineLifecycleStates[idx] || {}
    const isActive = Boolean(lifecycle.isActive)
    const isRelease = Boolean(lifecycle.isRelease)
    const isIncoming = Boolean(lifecycle.isIncoming)
    const activeAlpha = layer === 'main' ? 0.98 : 0.9
    const releaseAlpha = layer === 'main' ? 0.82 : 0.76
    const incomingAlpha = layer === 'main' ? 0.78 : 0.74
    const pastAlpha = layer === 'main' ? 0.66 : 0.62
    const futureAlpha = layer === 'main' ? 0.72 : 0.68
    let opacity = isActive
      ? 1
      : isIncoming
        ? 0.86
        : isRelease
          ? 0.84
          : delta < 0
            ? 0.72
            : 0.78

    if (delta > 1) opacity = Math.max(0.62, 0.78 - clamp(delta, 1, 6) * 0.03)
    if (delta < -1) {
      opacity = Math.max(0.62, 0.72 - clamp(Math.abs(delta), 1, 6) * 0.03)
    }
    const colorAlpha = isActive
      ? activeAlpha
      : isIncoming
        ? incomingAlpha
        : isRelease
          ? releaseAlpha
          : delta < 0
            ? pastAlpha
            : futureAlpha
    const transform = isActive
      ? 'scale(1) translateY(0px)'
      : isIncoming
        ? 'scale(0.995) translateY(1px)'
        : isRelease
          ? 'scale(0.992) translateY(1px)'
          : 'scale(0.985) translateY(2px)'

    return {
      opacity,
      color: colorWithAlpha(sourceColor, colorAlpha),
      transform,
    }
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
                style={{ cursor: line.start != null ? 'pointer' : undefined }}
                onClick={() => {
                  if (audioInstance && line.start != null) {
                    audioInstance.currentTime = line.start / 1000
                    resumeAutoScroll()
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
                      const highlightedLane = buildHighlightedMainLine(
                        lane,
                        mainNextLineStart,
                      )

                      return showPr && laneIdx === 0 ? (
                        <KaraokeStackedLineRow
                          key={lane.key || `lane-${laneIdx}`}
                          line={highlightedLane}
                          pronunciationLine={buildHighlightedAuxLine(
                            line,
                            prLine,
                            mainNextLineStart,
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
                    line={buildHighlightedMainLine(line, mainNextLineStart)}
                    pronunciationLine={buildHighlightedAuxLine(
                      line,
                      prLine,
                      mainNextLineStart,
                    )}
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
                    line={buildHighlightedMainLine(line, mainNextLineStart)}
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
                    line={buildHighlightedAuxLine(
                      line,
                      trLine,
                      mainNextLineStart,
                    )}
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
