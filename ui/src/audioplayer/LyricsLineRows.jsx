import Typography from '@material-ui/core/Typography'
import clsx from 'clsx'
import React, { memo, useLayoutEffect, useMemo, useRef, useState } from 'react'
import { resolveKaraokeTokenWindow } from './lyrics'
import { buildSegmentsFromLine } from './lyricsSegments'
import {
  KARAOKE_WORD_SETTLE_MS,
  TOKEN_ACTIVE_ALPHA,
  TOKEN_DONE_ALPHA,
  TOKEN_FUTURE_ALPHA,
  TOKEN_SHORT_DURATION_MS,
  TOKEN_WIPE_EDGE_PCT,
  TOKEN_WIPE_SOFT_SPREAD_PCT,
  clamp,
  easeInOut,
  lerp,
} from './lyricsKaraokeConstants'
import {
  buildEmphasisStyle,
  isEmphasisRole,
  parseColorRGB,
} from './lyricsKaraokeStyles'
import { shouldSkipLineFrame } from './lyricsTiming'

const EMPHASIS_TONE = 0.7

const tokenColor = (rgb, alpha) => {
  const [r, g, b] = rgb || [255, 255, 255]
  return `rgba(${r}, ${g}, ${b}, ${alpha})`
}

const toneEmphasisRGB = (rgb) =>
  rgb ? rgb.map((channel) => Math.round(channel * EMPHASIS_TONE)) : rgb

const getTokenRGB = (token, rgb) =>
  isEmphasisRole(token) ? toneEmphasisRGB(rgb) : rgb

const toneEmphasisColor = (color) => {
  const rgb = parseColorRGB(color)
  if (!rgb) return color

  const alpha = String(color).match(/rgba?\([^)]*?,\s*([\d.]+)\s*\)$/)?.[1]
  return tokenColor(toneEmphasisRGB(rgb), alpha == null ? 1 : Number(alpha))
}

const buildLineStyle = (line, style) => {
  const emphasisStyle = buildEmphasisStyle(line)
  if (!emphasisStyle) return style

  const emphasisColor = style?.color ? toneEmphasisColor(style.color) : null
  return {
    ...style,
    ...emphasisStyle,
    ...(emphasisColor
      ? {
          color: emphasisColor,
          WebkitTextFillColor: emphasisColor,
        }
      : {}),
  }
}

const buildInactiveTokenStyle = (token, rgb) => {
  const tonedRGB = getTokenRGB(token, rgb)
  const color = tokenColor(tonedRGB, TOKEN_FUTURE_ALPHA)
  return {
    color,
    WebkitTextFillColor: color,
    backgroundImage: 'none',
    ...buildEmphasisStyle(token),
  }
}

const buildStaticEmphasisStyle = (token, color) => {
  const emphasisStyle = buildEmphasisStyle(token)
  if (!emphasisStyle) return undefined

  const emphasisColor = color ? toneEmphasisColor(color) : null
  return {
    ...emphasisStyle,
    ...(emphasisColor
      ? {
          color: emphasisColor,
          WebkitTextFillColor: emphasisColor,
        }
      : {}),
  }
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
    a.transform === b.transform &&
    a.WebkitTextFillColor === b.WebkitTextFillColor
  )
}

const buildTokenWipeStyle = ({
  fillProgress,
  highlightAlpha,
  futureAlpha,
  rgb,
  useCrossfade = false,
}) => {
  const fillPct = clamp(fillProgress, 0, 1) * 100
  const doneColor = tokenColor(
    rgb,
    clamp(highlightAlpha, futureAlpha, TOKEN_ACTIVE_ALPHA),
  )
  const futureColor = tokenColor(rgb, futureAlpha)

  if (fillPct <= 0) {
    return {
      color: futureColor,
      WebkitTextFillColor: futureColor,
      backgroundImage: 'none',
    }
  }

  if (useCrossfade) {
    return {
      color: doneColor,
      WebkitTextFillColor: doneColor,
      backgroundImage: 'none',
    }
  }

  if (fillPct >= 100) {
    return {
      color: doneColor,
      WebkitTextFillColor: doneColor,
      backgroundImage: 'none',
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
  const tokenRGB = getTokenRGB(segment.token, rgb)
  const tokenDuration =
    visualStart != null && visualEnd != null ? visualEnd - visualStart : null
  const useCrossfade =
    (tokenDuration != null && tokenDuration <= TOKEN_SHORT_DURATION_MS) ||
    segment.text.trim().length <= 2

  return {
    tokenStart,
    style: {
      ...buildTokenWipeStyle({
        fillProgress,
        highlightAlpha: alpha,
        futureAlpha: TOKEN_FUTURE_ALPHA,
        rgb: tokenRGB,
        useCrossfade,
      }),
      ...buildEmphasisStyle(segment.token),
    },
  }
}

export const KaraokeLineRow = memo(
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
    const lineStyle = useMemo(() => buildLineStyle(line, style), [line, style])

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

          const { tokenStart, style: tokenStyle } = highlightTokens
            ? buildSegmentTokenStyle({
                segment,
                line,
                nextLineStart,
                renderPlaybackMs,
                rgb: tokenRGB,
                highlightAlphaScale,
              })
            : {
                tokenStart: segment.token?.start,
                style: buildInactiveTokenStyle(segment.token, tokenRGB),
              }

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

const canPairPronunciationSegment = (segment, hasTokenSegments) =>
  Boolean(
    segment.token ||
    (!hasTokenSegments && !segment.isWhitespace && segment.text.trim()),
  )

const buildStackedPronunciationSegments = (line, pronunciationLine) => {
  const lineSegments = buildSegmentsFromLine(line)
  const hasTokenSegments = lineSegments.some((segment) => segment.token)
  const mainSegments = hasTokenSegments
    ? lineSegments
    : splitTextSegments(line?.value || '')
  const pronunciationParts = buildPronunciationParts(pronunciationLine)
  const pairableSegments = mainSegments.filter((segment) =>
    canPairPronunciationSegment(segment, hasTokenSegments),
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

  const plainPronunciationParts = []
  const emphasisPronunciationParts = []
  for (const part of pronunciationParts) {
    if (hasTokenSegments && isEmphasisRole(part.segment?.token)) {
      emphasisPronunciationParts.push(part)
    } else {
      plainPronunciationParts.push(part)
    }
  }

  let pronunciationIndex = 0
  let plainPronunciationIndex = 0
  let emphasisPronunciationIndex = 0

  const getPronunciationPart = (segment) => {
    if (!hasTokenSegments) {
      const part = pronunciationParts[pronunciationIndex] || null
      pronunciationIndex += 1
      return part
    }

    if (isEmphasisRole(segment.token)) {
      const part =
        emphasisPronunciationParts[emphasisPronunciationIndex] || null
      emphasisPronunciationIndex += 1
      return part
    }

    const part = plainPronunciationParts[plainPronunciationIndex] || null
    plainPronunciationIndex += 1
    return part
  }

  return mainSegments.map((segment) => {
    const canPair = canPairPronunciationSegment(segment, hasTokenSegments)

    if (!canPair) return { ...segment, pronunciation: '' }

    const pronunciationPart = getPronunciationPart(segment)
    return {
      ...segment,
      pronunciation: pronunciationPart?.text || '',
      pronunciationSegment: pronunciationPart?.segment || null,
    }
  })
}

export const KaraokeStackedLineRow = memo(
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
    const lineStyle = useMemo(() => buildLineStyle(line, style), [line, style])

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
                style={buildStaticEmphasisStyle(segment.token, style?.color)}
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
              : segment.token
                ? {
                    tokenStart: segment.token.start,
                    style: buildInactiveTokenStyle(segment.token, tokenRGB),
                  }
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
                style={
                  tokenData?.style ||
                  buildStaticEmphasisStyle(segment.token, style?.color)
                }
              >
                {segment.text}
              </span>
              <span
                className={classes.stackedPronunciation}
                data-testid="lyrics-pronunciation-token"
                style={
                  pronunciationTokenData?.style || {
                    color: pronunciationStyle?.color,
                    WebkitTextFillColor: pronunciationStyle?.color,
                    backgroundImage: 'none',
                    ...buildStaticEmphasisStyle(
                      segment.pronunciationSegment?.token || segment.token,
                      pronunciationStyle?.color,
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
