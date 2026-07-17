import Typography from '@material-ui/core/Typography'
import clsx from 'clsx'
import React, {
  memo,
  useCallback,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from 'react'
import { buildSegmentsFromLine } from './lyricsSegments'
import {
  TOKEN_ACTIVE_ALPHA,
  TOKEN_FUTURE_ALPHA,
  TOKEN_WIPE_EDGE_PCT,
  TOKEN_WIPE_SOFT_SPREAD_PCT,
} from './lyricsKaraokeConstants'
import {
  buildEmphasisStyle,
  isEmphasisRole,
  parseColorRGB,
} from './lyricsKaraokeStyles'
import { resolveKaraokeTokenWindows } from './lyricsTimeline'

const EMPHASIS_TONE = 0.7
const EMPHASIS_PAINT_OVERHANG = '0.18em'

const emphasisPaintOverhangStyle = {
  marginInlineEnd: `-${EMPHASIS_PAINT_OVERHANG}`,
  paddingInlineEnd: EMPHASIS_PAINT_OVERHANG,
}

const graphemeSegmenter =
  typeof Intl !== 'undefined' && typeof Intl.Segmenter === 'function'
    ? new Intl.Segmenter(undefined, { granularity: 'grapheme' })
    : null

const splitGraphemes = (value) => {
  const text = String(value || '')
  return graphemeSegmenter
    ? Array.from(graphemeSegmenter.segment(text), ({ segment }) => segment)
    : Array.from(text)
}

const waveTextRootStyle = {
  display: 'inline-block',
  position: 'relative',
  verticalAlign: 'baseline',
}

const waveTextMeasureStyle = { visibility: 'hidden' }

const waveTextOverlayStyle = {
  display: 'block',
  inset: 0,
  position: 'absolute',
  whiteSpace: 'nowrap',
}

const renderWaveText = (text, eligible, enabled, emphasized, className) => {
  if (!eligible) return text

  return (
    <span
      aria-hidden="true"
      data-lyrics-wave-text="true"
      style={waveTextRootStyle}
    >
      {enabled ? (
        <>
          <span data-lyrics-wave-measure="true" style={waveTextMeasureStyle}>
            {text}
          </span>
          <span data-lyrics-wave-overlay="true" style={waveTextOverlayStyle}>
            {splitGraphemes(text).map((character, index) => {
              if (/^\s+$/.test(character)) return character
              return (
                <span
                  key={`${index}-${character}`}
                  className={className}
                  data-lyrics-character="true"
                  style={emphasized ? emphasisPaintOverhangStyle : undefined}
                >
                  {character}
                </span>
              )
            })}
          </span>
        </>
      ) : (
        text
      )}
    </span>
  )
}

const tokenColor = (rgb, alpha) => {
  const [r, g, b] = rgb || [255, 255, 255]
  return `rgba(${r}, ${g}, ${b}, ${alpha})`
}

const toneEmphasisRGB = (rgb) =>
  rgb ? rgb.map((channel) => Math.round(channel * EMPHASIS_TONE)) : rgb

const getTokenRGB = (token, rgb) =>
  isEmphasisRole(token) ? toneEmphasisRGB(rgb) : rgb

const stripLayerColors = (style) => {
  const result = { ...(style || {}) }
  delete result.opacity
  delete result.color
  delete result.WebkitTextFillColor
  delete result['--lyrics-active-color']
  return result
}

const buildLineStyle = (line, style) => {
  const emphasisStyle = buildEmphasisStyle(line)
  return {
    ...stripLayerColors(style),
    ...emphasisStyle,
    ...(emphasisStyle ? { filter: `brightness(${EMPHASIS_TONE})` } : {}),
  }
}

const buildStaticEmphasisStyle = (token) => {
  const emphasisStyle = buildEmphasisStyle(token)
  if (!emphasisStyle) return undefined
  return {
    ...emphasisStyle,
    filter: `brightness(${EMPHASIS_TONE})`,
  }
}

const buildTokenData = (token, rgb, line) => {
  const emphasized = isEmphasisRole(token) || isEmphasisRole(line)
  const tonedRGB = getTokenRGB(token, rgb)
  const futureColor = tokenColor(tonedRGB, TOKEN_FUTURE_ALPHA)
  const gradientDoneColor = tokenColor(
    tonedRGB,
    'var(--lyrics-token-active-alpha, 1)',
  )
  const softColor = tokenColor(
    tonedRGB,
    TOKEN_FUTURE_ALPHA + (TOKEN_ACTIVE_ALPHA - TOKEN_FUTURE_ALPHA) * 0.58,
  )
  const sweepRange = 100 + TOKEN_WIPE_SOFT_SPREAD_PCT
  const activeStop = `calc(var(--lyrics-progress) * ${sweepRange}% - ${TOKEN_WIPE_SOFT_SPREAD_PCT}%)`
  const softStop = `calc(var(--lyrics-progress) * ${sweepRange}% - ${TOKEN_WIPE_EDGE_PCT}%)`
  const futureStop = `calc(var(--lyrics-progress) * ${sweepRange}%)`
  const gradient = `linear-gradient(90deg, ${gradientDoneColor} 0%, ${gradientDoneColor} ${activeStop}, ${softColor} ${softStop}, ${futureColor} ${futureStop}, ${futureColor} 100%)`

  return {
    style: {
      '--lyrics-progress': 0,
      '--lyrics-token-active-alpha': TOKEN_ACTIVE_ALPHA,
      color: 'transparent',
      WebkitTextFillColor: 'transparent',
      backgroundImage: gradient,
      backgroundSize: '100% 100%',
      backgroundClip: 'text',
      WebkitBackgroundClip: 'text',
      ...buildEmphasisStyle(token),
      ...(emphasized ? emphasisPaintOverhangStyle : {}),
    },
    presentation: {
      futureAlpha: TOKEN_FUTURE_ALPHA,
      activeAlpha: TOKEN_ACTIVE_ALPHA,
      gradient,
    },
  }
}

const isTimedWindow = (window) =>
  Number.isFinite(window?.start) && Number.isFinite(window?.end)

const useStableTokenRef = (registerToken) => {
  const registerTokenRef = useRef(registerToken)
  const callbacksRef = useRef(new Map())
  registerTokenRef.current = registerToken

  return useCallback(
    ({ key, lineIndex, window, presentation, waveEnabled }) => {
      const signature = [
        lineIndex,
        window?.start ?? '',
        window?.end ?? '',
        presentation?.futureAlpha ?? '',
        presentation?.activeAlpha ?? '',
        presentation?.gradient ?? '',
        waveEnabled ? 1 : 0,
      ].join('\u0000')
      const cached = callbacksRef.current.get(key)
      if (cached?.signature === signature) return cached.callback

      const descriptor = { lineIndex, window, presentation }
      const callback = (node) => {
        registerTokenRef.current?.(key, descriptor, node)
      }
      callbacksRef.current.set(key, { signature, callback })
      return callback
    },
    [],
  )
}

export const KaraokeLineRow = memo(
  ({
    lineIndex,
    line,
    nextLineStart,
    className,
    style,
    tokenClassName,
    waveCharacterClassName,
    registerToken,
    renderCharacterWave = true,
    rowKey = 'main',
    testId,
  }) => {
    const getTokenRef = useStableTokenRef(registerToken)
    const segments = useMemo(() => buildSegmentsFromLine(line), [line])
    const windows = useMemo(
      () => resolveKaraokeTokenWindows(line, nextLineStart),
      [line, nextLineStart],
    )
    const tokenRGB = useMemo(
      () => (style?.color ? parseColorRGB(style.color) : [255, 255, 255]),
      [style?.color],
    )
    const hasTimedTokens = windows.some(
      (window) => window?.start != null && window?.end != null,
    )
    const lineStyle = useMemo(() => buildLineStyle(line, style), [line, style])

    return (
      <Typography
        className={className}
        component="div"
        data-testid={testId}
        data-tokenized={hasTimedTokens ? 'true' : 'false'}
        data-layer-animation={
          hasTimedTokens ? 'token-gradient' : 'shared-opacity'
        }
        style={lineStyle}
      >
        {segments.map((segment, idx) => {
          if (!segment.token)
            return <span key={`text-${idx}`}>{segment.text}</span>

          const window = windows[segment.tokenIndex]
          const timedWindow = isTimedWindow(window) ? window : null
          const key = `${lineIndex}:${rowKey}:${segment.tokenIndex}:main`
          const emphasized =
            isEmphasisRole(segment.token) || isEmphasisRole(line)
          const tokenData = buildTokenData(segment.token, tokenRGB, line)
          return (
            <span
              key={`token-${idx}-${window?.start ?? 'na'}`}
              className={tokenClassName}
              data-testid="lyrics-token"
              data-lyrics-state="future"
              ref={
                timedWindow
                  ? getTokenRef({
                      key,
                      lineIndex,
                      window: timedWindow,
                      presentation: tokenData.presentation,
                      waveEnabled: renderCharacterWave,
                    })
                  : undefined
              }
              style={tokenData.style}
              aria-label={segment.text}
            >
              {renderWaveText(
                segment.text,
                Boolean(timedWindow),
                Boolean(renderCharacterWave && timedWindow),
                emphasized,
                waveCharacterClassName,
              )}
            </span>
          )
        })}
      </Typography>
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
    if (!canPairPronunciationSegment(segment, hasTokenSegments)) {
      return { ...segment, pronunciation: '' }
    }
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
    lineIndex,
    line,
    pronunciationLine,
    pronunciationStyle,
    nextLineStart,
    className,
    style,
    tokenClassName,
    waveCharacterClassName,
    classes,
    registerToken,
    renderCharacterWave = true,
    rowKey = 'main',
    testId,
  }) => {
    const rowRef = useRef(null)
    const getTokenRef = useStableTokenRef(registerToken)
    const [isWrapped, setIsWrapped] = useState(false)
    const segments = useMemo(
      () => buildStackedPronunciationSegments(line, pronunciationLine),
      [line, pronunciationLine],
    )
    const mainWindows = useMemo(
      () => resolveKaraokeTokenWindows(line, nextLineStart),
      [line, nextLineStart],
    )
    const pronunciationWindows = useMemo(
      () => resolveKaraokeTokenWindows(pronunciationLine, null),
      [pronunciationLine],
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
    const hasTimedTokens = mainWindows.some(
      (window) => window?.start != null && window?.end != null,
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
        data-tokenized={hasTimedTokens ? 'true' : 'false'}
        data-layer-animation={
          hasTimedTokens ? 'token-gradient' : 'shared-opacity'
        }
        ref={rowRef}
        style={lineStyle}
      >
        {segments.map((segment, idx) => {
          const resolvedMainWindow = segment.token
            ? mainWindows[segment.tokenIndex]
            : null
          const mainWindow = isTimedWindow(resolvedMainWindow)
            ? resolvedMainWindow
            : null
          const mainKey = `${lineIndex}:${rowKey}:${segment.tokenIndex}:main`
          const mainTokenData = segment.token
            ? buildTokenData(segment.token, tokenRGB, line)
            : null
          const mainEmphasized =
            isEmphasisRole(segment.token) || isEmphasisRole(line)
          const mainText = segment.token ? (
            <span
              key={`main-${idx}`}
              className={clsx(tokenClassName, classes.stackedMainText)}
              data-testid="lyrics-token"
              data-lyrics-state="future"
              ref={
                mainWindow
                  ? getTokenRef({
                      key: mainKey,
                      lineIndex,
                      window: mainWindow,
                      presentation: mainTokenData.presentation,
                      waveEnabled: renderCharacterWave,
                    })
                  : undefined
              }
              style={mainTokenData.style}
              aria-label={segment.text}
            >
              {renderWaveText(
                segment.text,
                Boolean(mainWindow?.start != null && mainWindow?.end != null),
                Boolean(
                  renderCharacterWave &&
                  mainWindow?.start != null &&
                  mainWindow?.end != null,
                ),
                mainEmphasized,
                waveCharacterClassName,
              )}
            </span>
          ) : (
            <span
              key={`main-${idx}`}
              className={classes.stackedSpacer}
              style={buildStaticEmphasisStyle(segment.token)}
            >
              {segment.text}
            </span>
          )

          if (!segment.pronunciation) return mainText

          const pronunciationToken = segment.pronunciationSegment?.token
          const resolvedPronunciationWindow = pronunciationToken
            ? pronunciationWindows[segment.pronunciationSegment.tokenIndex]
            : null
          const pronunciationWindow =
            mainWindow ||
            (isTimedWindow(resolvedPronunciationWindow)
              ? resolvedPronunciationWindow
              : null)
          const pronunciationTokenIndex =
            segment.pronunciationSegment?.tokenIndex >= 0
              ? segment.pronunciationSegment.tokenIndex
              : segment.tokenIndex >= 0
                ? segment.tokenIndex
                : idx
          const pronunciationKey = `${lineIndex}:${rowKey}:${pronunciationTokenIndex}:pronunciation`
          const pronunciationTokenData = pronunciationWindow
            ? buildTokenData(
                pronunciationToken || segment.token,
                pronunciationRGB,
                pronunciationLine || line,
              )
            : null
          const pronunciationEmphasized = Boolean(
            isEmphasisRole(pronunciationToken) ||
            isEmphasisRole(segment.token) ||
            isEmphasisRole(pronunciationLine) ||
            isEmphasisRole(line),
          )
          return (
            <span
              key={`stacked-${idx}-${mainWindow?.start ?? segment.text}`}
              className={classes.stackedToken}
              data-stacked-token="true"
            >
              {mainText}
              <span
                className={classes.stackedPronunciation}
                data-testid="lyrics-pronunciation-token"
                data-lyrics-state="future"
                data-timed={pronunciationWindow ? 'true' : 'false'}
                aria-label={
                  pronunciationWindow ? segment.pronunciation : undefined
                }
                ref={
                  pronunciationWindow
                    ? getTokenRef({
                        key: pronunciationKey,
                        lineIndex,
                        window: pronunciationWindow,
                        presentation: pronunciationTokenData.presentation,
                        waveEnabled: renderCharacterWave,
                      })
                    : undefined
                }
                style={
                  pronunciationTokenData?.style || {
                    backgroundImage: 'none',
                    ...buildStaticEmphasisStyle(
                      pronunciationToken || segment.token,
                    ),
                  }
                }
              >
                {renderWaveText(
                  segment.pronunciation,
                  Boolean(pronunciationWindow),
                  Boolean(renderCharacterWave && pronunciationWindow),
                  pronunciationEmphasized,
                  waveCharacterClassName,
                )}
              </span>
            </span>
          )
        })}
      </Typography>
    )
  },
)

KaraokeStackedLineRow.displayName = 'KaraokeStackedLineRow'
