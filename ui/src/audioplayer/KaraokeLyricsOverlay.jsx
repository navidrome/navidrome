import React, {
  memo,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react'
import clsx from 'clsx'
import Button from '@material-ui/core/Button'
import IconButton from '@material-ui/core/IconButton'
import Popover from '@material-ui/core/Popover'
import Slider from '@material-ui/core/Slider'
import Typography from '@material-ui/core/Typography'
import CloseIcon from '@material-ui/icons/Close'
import TuneIcon from '@material-ui/icons/Tune'
import { makeStyles } from '@material-ui/core/styles'
import {
  buildKaraokeLines,
  getActiveKaraokeState,
  hasStructuredLyricContent,
  resolveLayerLineForMain,
  resolveKaraokeTokenWindow,
} from './lyrics'

const KARAOKE_RENDER_LEAD_MS = 24
const KARAOKE_CLOCK_DRIFT_RESET_MS = 140
const KARAOKE_CLOCK_RESET_THRESHOLD_MS = 320
const KARAOKE_MONOTONIC_JITTER_MS = 60
const KARAOKE_RENDER_UPDATE_EPSILON_MS = 6
const KARAOKE_WORD_SETTLE_MS = 96
const KARAOKE_ANIMATION_MS = 150
const KARAOKE_DEFAULT_HEIGHT_PX = 300
const KARAOKE_MIN_HEIGHT_PX = 150
const KARAOKE_MAX_HEIGHT_RATIO = 0.72
const KARAOKE_MAX_HEIGHT_PX = 760
const KARAOKE_CENTER_SPACER_RATIO = 0.5
const KARAOKE_CENTER_SPACER_MIN_PX = 132

const TOKEN_DONE_ALPHA = 1
const TOKEN_FUTURE_ALPHA = 0.34
const TOKEN_ACTIVE_ALPHA = 1
const TOKEN_WIPE_EDGE_PCT = 8
const TOKEN_WIPE_GLOW_PCT = 16

const COLOR_PRESETS = [
  { key: 'white', label: 'White', value: 'rgba(255, 255, 255, 0.92)' },
  { key: 'blue', label: 'Blue', value: 'rgba(120, 160, 220, 0.75)' },
  { key: 'green', label: 'Green', value: 'rgba(100, 200, 130, 0.7)' },
  { key: 'pink', label: 'Pink', value: 'rgba(240, 140, 170, 0.75)' },
  { key: 'purple', label: 'Purple', value: 'rgba(180, 140, 240, 0.75)' },
  { key: 'orange', label: 'Orange', value: 'rgba(240, 180, 100, 0.75)' },
  { key: 'cyan', label: 'Cyan', value: 'rgba(100, 210, 220, 0.75)' },
  { key: 'yellow', label: 'Yellow', value: 'rgba(240, 230, 110, 0.75)' },
]

const DEFAULT_LYRICS_SETTINGS = {
  tr: { fontSize: 14, colorKey: 'blue' },
  main: { fontSize: 24, colorKey: 'white' },
  pr: { fontSize: 14, colorKey: 'green' },
}

const SETTINGS_STORAGE_KEY = 'karaoke-lyrics-settings'

const loadLyricsSettings = () => {
  try {
    const raw = localStorage.getItem(SETTINGS_STORAGE_KEY)
    if (raw) {
      const parsed = JSON.parse(raw)
      return {
        tr: { ...DEFAULT_LYRICS_SETTINGS.tr, ...parsed.tr },
        main: { ...DEFAULT_LYRICS_SETTINGS.main, ...parsed.main },
        pr: { ...DEFAULT_LYRICS_SETTINGS.pr, ...parsed.pr },
      }
    }
  } catch {
    /* ignore */
  }
  return { ...DEFAULT_LYRICS_SETTINGS }
}

const saveLyricsSettings = (settings) => {
  try {
    localStorage.setItem(SETTINGS_STORAGE_KEY, JSON.stringify(settings))
  } catch {
    /* ignore */
  }
}

const getColorValue = (colorKey) =>
  COLOR_PRESETS.find((c) => c.key === colorKey)?.value || COLOR_PRESETS[0].value

const useStyles = makeStyles((theme) => ({
  overlay: {
    position: 'fixed',
    left: '50%',
    bottom: 100,
    transform: 'translateX(-50%)',
    zIndex: 1400,
    width: 'min(900px, calc(100vw - 32px))',
    minHeight: KARAOKE_MIN_HEIGHT_PX,
    background: 'rgba(6, 8, 12, 0.9)',
    borderRadius: 12,
    border: '1px solid rgba(255, 255, 255, 0.12)',
    boxShadow: '0 18px 48px rgba(0, 0, 0, 0.42)',
    backdropFilter: 'blur(10px)',
    color: theme.palette.common.white,
    display: 'flex',
    flexDirection: 'column',
    overflow: 'hidden',
    '@media (max-width:810px)': {
      bottom: 78,
      width: 'calc(100vw - 12px)',
      borderRadius: 8,
      minHeight: 180,
      maxHeight: '65vh',
    },
  },
  resizeHandle: {
    height: 14,
    cursor: 'ns-resize',
    flexShrink: 0,
    position: 'relative',
    '&::after': {
      content: '""',
      position: 'absolute',
      left: '50%',
      top: 4,
      transform: 'translateX(-50%)',
      width: 56,
      height: 3,
      borderRadius: 999,
      background: 'rgba(255, 255, 255, 0.22)',
    },
    '@media (max-width:810px)': {
      display: 'none',
    },
  },
  header: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    gap: theme.spacing(1),
    padding: theme.spacing(0.3, 1.3, 0.4, 1.3),
  },
  headerLeft: {
    display: 'flex',
    alignItems: 'center',
    gap: theme.spacing(1),
    minWidth: 0,
  },
  language: {
    fontSize: 11,
    letterSpacing: '0.08em',
    opacity: 0.72,
    textTransform: 'uppercase',
    whiteSpace: 'nowrap',
  },
  layerControls: {
    display: 'flex',
    alignItems: 'center',
    gap: theme.spacing(0.5),
  },
  layerToggle: {
    minWidth: 34,
    minHeight: 24,
    padding: theme.spacing(0, 0.8),
    fontSize: 10,
    letterSpacing: '0.08em',
    borderRadius: 999,
    color: 'rgba(203, 213, 225, 0.95)',
    background: 'rgba(100, 116, 139, 0.26)',
    border: '1px solid rgba(148, 163, 184, 0.45)',
    transition: `all ${KARAOKE_ANIMATION_MS}ms ease-in-out`,
    '&.Mui-disabled': {
      color: 'rgba(148, 163, 184, 0.45)',
      borderColor: 'rgba(100, 116, 139, 0.3)',
      background: 'rgba(71, 85, 105, 0.2)',
    },
  },
  layerToggleActive: {
    color: 'rgba(220, 252, 231, 0.98)',
    borderColor: 'rgba(34, 197, 94, 0.96)',
    background: 'rgba(34, 197, 94, 0.28)',
  },
  closeButton: {
    color: 'rgba(255, 255, 255, 0.72)',
  },
  inlineTr: {
    margin: '0 0 2px 0',
    textAlign: 'center',
    fontWeight: 400,
    lineHeight: 1.2,
    letterSpacing: '0.01em',
    transition: `opacity ${KARAOKE_ANIMATION_MS}ms ease-in-out, font-size ${KARAOKE_ANIMATION_MS}ms ease-in-out`,
  },
  inlinePr: {
    margin: '2px 0 0 0',
    textAlign: 'center',
    fontWeight: 400,
    lineHeight: 1.2,
    letterSpacing: '0.01em',
    transition: `opacity ${KARAOKE_ANIMATION_MS}ms ease-in-out, font-size ${KARAOKE_ANIMATION_MS}ms ease-in-out`,
  },
  body: {
    padding: theme.spacing(0.5, 2, 1.4, 2),
    overflowY: 'auto',
    overflowX: 'hidden',
    scrollBehavior: 'smooth',
    flex: 1,
    overscrollBehavior: 'contain',
    scrollbarWidth: 'none',
    msOverflowStyle: 'none',
    '&::-webkit-scrollbar': {
      display: 'none',
      width: 0,
      height: 0,
    },
    '@media (max-width:810px)': {
      padding: theme.spacing(0.35, 1.2, 1.2, 1.2),
    },
  },
  lines: {
    display: 'flex',
    flexDirection: 'column',
    gap: theme.spacing(1.24),
    paddingBottom: theme.spacing(1),
  },
  line: {
    margin: 0,
    fontWeight: 600,
    lineHeight: 1.24,
    letterSpacing: '0.01em',
    textAlign: 'center',
    color: 'rgba(255, 255, 255, 0.62)',
    transition: `opacity ${KARAOKE_ANIMATION_MS}ms ease-in-out, color ${KARAOKE_ANIMATION_MS}ms ease-in-out, font-size 280ms ease-in-out`,
  },
  token: {
    display: 'inline-block',
    whiteSpace: 'pre-wrap',
    transition: `color ${KARAOKE_ANIMATION_MS}ms ease-in-out, text-shadow ${KARAOKE_ANIMATION_MS}ms ease-in-out`,
  },
  settingsButton: {
    color: 'rgba(255, 255, 255, 0.55)',
    padding: 4,
    '&:hover': {
      color: 'rgba(255, 255, 255, 0.85)',
    },
  },
  settingsPanel: {
    background: 'rgba(12, 14, 20, 0.96)',
    border: '1px solid rgba(255, 255, 255, 0.12)',
    borderRadius: 10,
    padding: theme.spacing(1.5, 2),
    width: 260,
    backdropFilter: 'blur(12px)',
  },
  settingsSection: {
    marginBottom: theme.spacing(1.2),
    '&:last-child': {
      marginBottom: 0,
    },
  },
  settingsLabel: {
    fontSize: 10,
    fontWeight: 600,
    letterSpacing: '0.1em',
    textTransform: 'uppercase',
    color: 'rgba(255, 255, 255, 0.55)',
    marginBottom: 4,
  },
  settingsRow: {
    display: 'flex',
    alignItems: 'center',
    gap: theme.spacing(1),
  },
  settingsSlider: {
    flex: 1,
    color: 'rgba(255, 255, 255, 0.6)',
    '& .MuiSlider-thumb': {
      width: 12,
      height: 12,
    },
    '& .MuiSlider-rail': {
      opacity: 0.3,
    },
  },
  settingsSliderValue: {
    fontSize: 11,
    color: 'rgba(255, 255, 255, 0.5)',
    minWidth: 22,
    textAlign: 'right',
  },
  colorDots: {
    display: 'flex',
    gap: 5,
    marginTop: 4,
  },
  colorDot: {
    width: 16,
    height: 16,
    borderRadius: '50%',
    border: '2px solid transparent',
    cursor: 'pointer',
    transition: 'border-color 120ms ease, transform 120ms ease',
    '&:hover': {
      transform: 'scale(1.2)',
    },
  },
  colorDotActive: {
    borderColor: 'rgba(255, 255, 255, 0.85)',
  },
}))

const clamp = (v, min, max) => Math.max(min, Math.min(max, v))
const lerp = (from, to, t) => from + (to - from) * t

const normalizeForComparison = (text) =>
  (text || '').replace(/[\s\p{P}]/gu, '').toLowerCase()

const shouldShowAuxLine = (mainLine, auxLine) => {
  if (!auxLine || !auxLine.value) return false
  return (
    normalizeForComparison(auxLine.value) !==
    normalizeForComparison(mainLine.value)
  )
}

const SettingsSection = ({ label, layer, settings, onChange, classes }) => {
  const s = settings[layer]
  return (
    <div className={classes.settingsSection}>
      <div className={classes.settingsLabel}>{label}</div>
      <div className={classes.settingsRow}>
        <Slider
          className={classes.settingsSlider}
          min={8}
          max={40}
          step={1}
          value={s.fontSize}
          onChange={(_, val) =>
            onChange({ ...settings, [layer]: { ...s, fontSize: val } })
          }
        />
        <span className={classes.settingsSliderValue}>{s.fontSize}</span>
      </div>
      <div className={classes.colorDots}>
        {COLOR_PRESETS.map((preset) => (
          <div
            key={preset.key}
            className={clsx(classes.colorDot, {
              [classes.colorDotActive]: s.colorKey === preset.key,
            })}
            style={{ background: preset.value }}
            title={preset.label}
            onClick={() =>
              onChange({ ...settings, [layer]: { ...s, colorKey: preset.key } })
            }
          />
        ))}
      </div>
    </div>
  )
}

const LyricsSettingsPopover = ({ settings, onChange }) => {
  const classes = useStyles()
  const [anchorEl, setAnchorEl] = useState(null)

  const handleToggle = useCallback((e) => {
    e.stopPropagation()
    setAnchorEl((prev) => (prev ? null : e.currentTarget))
  }, [])

  const handleClose = useCallback(() => setAnchorEl(null), [])

  return (
    <>
      <IconButton
        className={classes.settingsButton}
        size="small"
        onClick={handleToggle}
        aria-label="Lyrics settings"
      >
        <TuneIcon style={{ fontSize: 18 }} />
      </IconButton>
      <Popover
        open={Boolean(anchorEl)}
        anchorEl={anchorEl}
        onClose={handleClose}
        anchorOrigin={{ vertical: 'top', horizontal: 'center' }}
        transformOrigin={{ vertical: 'bottom', horizontal: 'center' }}
        PaperProps={{ className: classes.settingsPanel }}
        style={{ zIndex: 1500 }}
      >
        <SettingsSection
          label="Translation"
          layer="tr"
          settings={settings}
          onChange={onChange}
          classes={classes}
        />
        <SettingsSection
          label="Default"
          layer="main"
          settings={settings}
          onChange={onChange}
          classes={classes}
        />
        <SettingsSection
          label="Pronunciation"
          layer="pr"
          settings={settings}
          onChange={onChange}
          classes={classes}
        />
      </Popover>
    </>
  )
}

const easeInOut = (v) => {
  const clamped = clamp(v, 0, 1)
  return clamped < 0.5
    ? 2 * clamped * clamped
    : 1 - Math.pow(-2 * clamped + 2, 2) / 2
}

const getMaxHeightPx = () => {
  if (typeof window === 'undefined') {
    return KARAOKE_MAX_HEIGHT_PX
  }
  return Math.min(
    Math.floor(window.innerHeight * KARAOKE_MAX_HEIGHT_RATIO),
    KARAOKE_MAX_HEIGHT_PX,
  )
}

const buildSegmentsFromLine = (line) => {
  if (!line || !Array.isArray(line.tokens) || line.tokens.length === 0) {
    return [{ text: line?.value || '', token: null, tokenIndex: -1 }]
  }

  const text = line.value || ''
  const matchedSegments = []
  const fallbackSegments = []
  let cursor = 0
  let allMatched = text.length > 0
  let anyMatched = false

  const pushFallbackSeparatorIfNeeded = (nextTokenText) => {
    if (fallbackSegments.length === 0) {
      return
    }
    const prevText = fallbackSegments[fallbackSegments.length - 1].text || ''
    if (!prevText || !nextTokenText) {
      return
    }
    if (/\s$/.test(prevText) || /^\s/.test(nextTokenText)) {
      return
    }
    if (/[A-Za-z0-9]$/.test(prevText) && /^[A-Za-z0-9]/.test(nextTokenText)) {
      fallbackSegments.push({ text: ' ', token: null, tokenIndex: -1 })
    }
  }

  for (let tokenIndex = 0; tokenIndex < line.tokens.length; tokenIndex += 1) {
    const token = line.tokens[tokenIndex]
    const tokenText = token.value || ''
    if (!tokenText) {
      continue
    }

    pushFallbackSeparatorIfNeeded(tokenText)
    fallbackSegments.push({ text: tokenText, token, tokenIndex })

    if (!text) {
      allMatched = false
      continue
    }

    const foundAt = text.indexOf(tokenText, cursor)
    const normalizedFoundAt =
      foundAt >= 0
        ? foundAt
        : text.toLowerCase().indexOf(tokenText.toLowerCase(), cursor)

    if (normalizedFoundAt >= 0) {
      anyMatched = true
      if (normalizedFoundAt > cursor) {
        matchedSegments.push({
          text: text.slice(cursor, normalizedFoundAt),
          token: null,
          tokenIndex: -1,
        })
      }
      const matchedTokenText = text.slice(
        normalizedFoundAt,
        normalizedFoundAt + tokenText.length,
      )
      matchedSegments.push({
        text: matchedTokenText || tokenText,
        token,
        tokenIndex,
      })
      cursor = normalizedFoundAt + tokenText.length
    } else {
      allMatched = false
    }
  }

  if (allMatched && anyMatched) {
    if (cursor < text.length) {
      matchedSegments.push({
        text: text.slice(cursor),
        token: null,
        tokenIndex: -1,
      })
    }
    return matchedSegments
  }

  if (fallbackSegments.length > 0) {
    return fallbackSegments
  }

  return [{ text, token: null, tokenIndex: -1 }]
}

const getLineRenderWindow = (line, nextLineStart) => {
  let start = Number.isFinite(Number(line?.start)) ? Number(line.start) : null
  let end = Number.isFinite(Number(line?.end)) ? Number(line.end) : null
  const fallbackEnd = Number.isFinite(Number(nextLineStart))
    ? Number(nextLineStart)
    : null

  if (end == null) {
    end = fallbackEnd
  }

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
  if (prevPlaybackMs === nextPlaybackMs) {
    return true
  }

  const { start, end } = getLineRenderWindow(line, nextLineStart)

  if (start != null) {
    const activationStart = start - 220
    if (prevPlaybackMs < activationStart && nextPlaybackMs < activationStart) {
      return true
    }
  }

  if (end != null) {
    const settleEnd = end + KARAOKE_WORD_SETTLE_MS + 160
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
    a.fontWeight === b.fontWeight
  )
}

const parseColorRGB = (rgba) => {
  const m = (rgba || '').match(/rgba?\((\d+),\s*(\d+),\s*(\d+)/)
  return m ? [parseInt(m[1]), parseInt(m[2]), parseInt(m[3])] : [255, 255, 255]
}

const buildTokenWipeStyle = ({
  fillProgress,
  highlightAlpha,
  futureAlpha,
  rgb,
}) => {
  const [r, g, b] = rgb || [255, 255, 255]
  const fillPct = clamp(fillProgress, 0, 1) * 100
  const doneColor = `rgba(${r}, ${g}, ${b}, ${clamp(highlightAlpha, TOKEN_DONE_ALPHA, TOKEN_ACTIVE_ALPHA)})`
  const futureColor = `rgba(${r}, ${g}, ${b}, ${futureAlpha})`
  const activeShadow = `0 0 8px rgba(${r}, ${g}, ${b}, 0.34)`

  if (fillPct <= 0) {
    return { color: futureColor, textShadow: 'none' }
  }

  const edgeStart = clamp(fillPct - TOKEN_WIPE_EDGE_PCT, 0, 100)
  const glowStop = clamp(fillPct + TOKEN_WIPE_GLOW_PCT, 0, 100)
  const glowColor = `rgba(${r}, ${g}, ${b}, ${clamp(highlightAlpha + 0.18, TOKEN_DONE_ALPHA, TOKEN_ACTIVE_ALPHA)})`
  return {
    color: 'transparent',
    WebkitTextFillColor: 'transparent',
    backgroundImage: `linear-gradient(90deg, ${doneColor} 0%, ${doneColor} ${edgeStart}%, ${glowColor} ${fillPct}%, ${futureColor} ${glowStop}%, ${futureColor} 100%)`,
    backgroundClip: 'text',
    WebkitBackgroundClip: 'text',
    textShadow: activeShadow,
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
  }) => {
    const segments = buildSegmentsFromLine(line)
    const tokenRGB = useMemo(
      () => (style?.color ? parseColorRGB(style.color) : [255, 255, 255]),
      [style?.color],
    )

    return (
      <Typography className={className} component="div" style={style}>
        {segments.map((segment, idx) => {
          if (!segment.token) {
            return <span key={`text-${idx}`}>{segment.text}</span>
          }

          if (!highlightTokens) {
            return <span key={`token-plain-${idx}`}>{segment.text}</span>
          }

          const { start: tokenStart, end: tokenEnd } =
            resolveKaraokeTokenWindow(line, segment.tokenIndex, nextLineStart)

          const isDone = tokenEnd != null ? renderPlaybackMs >= tokenEnd : false
          const isActive =
            !isDone && tokenStart != null && renderPlaybackMs >= tokenStart

          const progress =
            isDone ||
            tokenStart == null ||
            tokenEnd == null ||
            tokenEnd <= tokenStart
              ? isDone
                ? 1
                : 0
              : clamp(
                  (renderPlaybackMs - tokenStart) / (tokenEnd - tokenStart),
                  0,
                  1,
                )

          const justEnded =
            tokenEnd != null &&
            renderPlaybackMs > tokenEnd &&
            renderPlaybackMs <= tokenEnd + KARAOKE_WORD_SETTLE_MS

          const settleProgress =
            justEnded && tokenEnd != null
              ? clamp(
                  (renderPlaybackMs - tokenEnd) / KARAOKE_WORD_SETTLE_MS,
                  0,
                  1,
                )
              : 0

          let alpha = TOKEN_FUTURE_ALPHA
          if (isDone) {
            alpha = TOKEN_DONE_ALPHA
          } else if (isActive) {
            alpha = lerp(
              TOKEN_FUTURE_ALPHA,
              TOKEN_ACTIVE_ALPHA,
              easeInOut(progress),
            )
          }
          if (justEnded) {
            alpha = lerp(
              TOKEN_ACTIVE_ALPHA,
              TOKEN_DONE_ALPHA,
              easeInOut(settleProgress),
            )
          }
          alpha = clamp(alpha, TOKEN_FUTURE_ALPHA, TOKEN_ACTIVE_ALPHA)
          const fillProgress = isDone ? 1 : isActive ? progress : 0

          return (
            <span
              key={`token-${idx}-${tokenStart ?? 'na'}`}
              className={tokenClassName}
              style={buildTokenWipeStyle({
                fillProgress,
                highlightAlpha: alpha,
                futureAlpha: TOKEN_FUTURE_ALPHA,
                rgb: tokenRGB,
              })}
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

const KaraokeLyricsOverlay = ({
  visible,
  mainLyric,
  translationLyric,
  pronunciationLyric,
  showTranslation,
  showPronunciation,
  translationEnabled,
  pronunciationEnabled,
  onToggleTranslation,
  onTogglePronunciation,
  audioInstance,
  onClose,
}) => {
  const classes = useStyles()
  const [playbackMs, setPlaybackMs] = useState(0)
  const [overlayHeight, setOverlayHeight] = useState(KARAOKE_DEFAULT_HEIGHT_PX)
  const [maxHeightPx, setMaxHeightPx] = useState(getMaxHeightPx())
  const [bodyViewportHeight, setBodyViewportHeight] = useState(0)
  const [isCompact, setIsCompact] = useState(
    typeof window !== 'undefined' ? window.innerWidth <= 810 : false,
  )
  const [lyricsSettings, setLyricsSettings] = useState(loadLyricsSettings)

  const handleSettingsChange = useCallback((next) => {
    setLyricsSettings(next)
    saveLyricsSettings(next)
  }, [])

  const bodyRef = useRef(null)
  const activeLineRef = useRef(null)

  const mainLines = useMemo(() => buildKaraokeLines(mainLyric), [mainLyric])
  const translationLines = useMemo(
    () => buildKaraokeLines(translationLyric),
    [translationLyric],
  )
  const pronunciationLines = useMemo(
    () => buildKaraokeLines(pronunciationLyric),
    [pronunciationLyric],
  )

  useEffect(() => {
    const onResize = () => {
      const nextMaxHeight = getMaxHeightPx()
      setIsCompact(window.innerWidth <= 810)
      setMaxHeightPx(nextMaxHeight)
      setOverlayHeight((previous) =>
        clamp(previous, KARAOKE_MIN_HEIGHT_PX, nextMaxHeight),
      )
    }

    onResize()
    window.addEventListener('resize', onResize)
    return () => window.removeEventListener('resize', onResize)
  }, [])

  useEffect(() => {
    const body = bodyRef.current
    if (!body) {
      return undefined
    }

    const updateViewportHeight = () => {
      setBodyViewportHeight(body.clientHeight || 0)
    }

    updateViewportHeight()

    if (typeof ResizeObserver !== 'undefined') {
      const observer = new ResizeObserver(updateViewportHeight)
      observer.observe(body)
      return () => observer.disconnect()
    }

    window.addEventListener('resize', updateViewportHeight)
    return () => window.removeEventListener('resize', updateViewportHeight)
  }, [overlayHeight, isCompact, showTranslation, showPronunciation, visible])

  const onResizeStart = useCallback(
    (event) => {
      if (isCompact) {
        return
      }

      event.preventDefault()
      const startY = event.clientY
      const startHeight = overlayHeight

      const onMove = (moveEvent) => {
        const delta = startY - moveEvent.clientY
        setOverlayHeight(
          clamp(startHeight + delta, KARAOKE_MIN_HEIGHT_PX, maxHeightPx),
        )
      }

      const onUp = () => {
        window.removeEventListener('mousemove', onMove)
        window.removeEventListener('mouseup', onUp)
      }

      window.addEventListener('mousemove', onMove)
      window.addEventListener('mouseup', onUp)
    },
    [isCompact, maxHeightPx, overlayHeight],
  )

  useEffect(() => {
    if (!visible || !audioInstance) {
      setPlaybackMs(0)
      return
    }

    let rafId = 0
    let cancelled = false
    let anchorAudioMs = 0
    let anchorPerfMs = 0
    let lastRenderMs = 0

    const readPlaybackMs = () => {
      const seconds = Number(audioInstance.currentTime)
      if (!Number.isFinite(seconds) || seconds < 0) {
        return 0
      }
      return seconds * 1000
    }

    const resetAnchor = (perfNow, observedMs) => {
      anchorAudioMs = observedMs
      anchorPerfMs = perfNow
    }

    const tick = () => {
      if (cancelled) {
        return
      }

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
      if (canInterpolate && backwardsDrift > 0) {
        nowMs = lastRenderMs
      }

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
      if (rafId) {
        window.cancelAnimationFrame(rafId)
      }
    }
  }, [audioInstance, visible])

  const renderPlaybackMs = playbackMs + KARAOKE_RENDER_LEAD_MS

  const { lineIndex } = useMemo(
    () => getActiveKaraokeState(mainLines, renderPlaybackMs),
    [mainLines, renderPlaybackMs],
  )

  const activeIndex = lineIndex >= 0 ? lineIndex : 0

  const trByMainIndex = useMemo(() => {
    if (!showTranslation || translationLines.length === 0) return {}
    const map = {}
    for (let i = 0; i < mainLines.length; i++) {
      const { line } = resolveLayerLineForMain(mainLines, translationLines, i)
      if (line) map[i] = line
    }
    return map
  }, [mainLines, translationLines, showTranslation])

  const prByMainIndex = useMemo(() => {
    if (!showPronunciation || pronunciationLines.length === 0) return {}
    const map = {}
    for (let i = 0; i < mainLines.length; i++) {
      const { line } = resolveLayerLineForMain(mainLines, pronunciationLines, i)
      if (line) map[i] = line
    }
    return map
  }, [mainLines, pronunciationLines, showPronunciation])

  const hasTranslationLine = showTranslation && translationLines.length > 0
  const hasPronunciationLine =
    showPronunciation && pronunciationLines.length > 0
  const measuredViewportHeight = bodyRef.current?.clientHeight || 0
  const estimatedViewportHeight =
    measuredViewportHeight > 0
      ? measuredViewportHeight
      : bodyViewportHeight > 0
        ? bodyViewportHeight
        : isCompact
          ? 260
          : Math.max(220, overlayHeight - 170)
  const centerSpacerPx = Math.max(
    KARAOKE_CENTER_SPACER_MIN_PX,
    Math.floor(estimatedViewportHeight * KARAOKE_CENTER_SPACER_RATIO),
  )

  useEffect(() => {
    if (!visible) {
      return
    }

    const rafId = window.requestAnimationFrame(() => {
      const body = bodyRef.current
      const activeNode = activeLineRef.current
      if (!body || !activeNode) {
        return
      }

      const bodyRect = body.getBoundingClientRect()
      const activeRect = activeNode.getBoundingClientRect()
      const deltaWithinBody =
        activeRect.top -
        bodyRect.top -
        (body.clientHeight - activeRect.height) / 2
      const maxTop = Math.max(0, body.scrollHeight - body.clientHeight)
      const centeredTop = clamp(body.scrollTop + deltaWithinBody, 0, maxTop)

      if (Math.abs(body.scrollTop - centeredTop) < 2) {
        return
      }

      if (typeof body.scrollTo === 'function') {
        body.scrollTo({
          top: centeredTop,
          behavior: 'smooth',
        })
      } else {
        body.scrollTop = centeredTop
      }
    })

    return () => window.cancelAnimationFrame(rafId)
  }, [
    centerSpacerPx,
    hasPronunciationLine,
    hasTranslationLine,
    lineIndex,
    overlayHeight,
    visible,
  ])

  if (
    !visible ||
    !hasStructuredLyricContent(mainLyric) ||
    mainLines.length === 0
  ) {
    return null
  }

  const getMainLineStyle = (idx) => {
    const delta = idx - activeIndex
    const isActive = delta === 0
    let opacity = isActive ? 1 : delta < 0 ? 0.6 : 0.72
    const [r, g, b] = parseColorRGB(getColorValue(lyricsSettings.main.colorKey))
    let color = isActive
      ? `rgba(${r}, ${g}, ${b}, 0.98)`
      : delta < 0
        ? `rgba(${r}, ${g}, ${b}, 0.4)`
        : `rgba(${r}, ${g}, ${b}, 0.54)`

    if (delta > 1) {
      const level = clamp(delta, 1, 6)
      opacity = Math.max(0.36, 0.74 - level * 0.08)
    }

    if (delta < -1) {
      const level = clamp(Math.abs(delta), 1, 6)
      opacity = Math.max(0.28, 0.62 - level * 0.08)
    }

    const baseFontSize = lyricsSettings.main.fontSize
    const fontSize = isActive ? baseFontSize : Math.round(baseFontSize * 0.8)

    return {
      opacity,
      color,
      fontSize,
    }
  }

  const overlayStyle = isCompact
    ? undefined
    : {
        height: overlayHeight,
        maxHeight: maxHeightPx,
      }

  return (
    <div
      className={classes.overlay}
      data-testid="karaoke-lyrics-overlay"
      style={overlayStyle}
    >
      <div className={classes.resizeHandle} onMouseDown={onResizeStart} />

      <div className={classes.header}>
        <div className={classes.headerLeft}>
          <Typography className={classes.language}>
            {mainLyric?.lang || 'xxx'}
          </Typography>
          <div className={classes.layerControls}>
            <Button
              size="small"
              onClick={onToggleTranslation}
              disabled={!translationEnabled}
              className={clsx(classes.layerToggle, {
                [classes.layerToggleActive]: showTranslation,
              })}
              data-testid="lyrics-toggle-translation"
            >
              TR
            </Button>
            <Button
              size="small"
              onClick={onTogglePronunciation}
              disabled={!pronunciationEnabled}
              className={clsx(classes.layerToggle, {
                [classes.layerToggleActive]: showPronunciation,
              })}
              data-testid="lyrics-toggle-pronunciation"
            >
              PR
            </Button>
          </div>
        </div>

        <div style={{ display: 'flex', alignItems: 'center', gap: 2 }}>
          <LyricsSettingsPopover
            settings={lyricsSettings}
            onChange={handleSettingsChange}
          />
          <IconButton
            className={classes.closeButton}
            size="small"
            onClick={onClose}
            aria-label="Close lyrics"
          >
            <CloseIcon fontSize="small" />
          </IconButton>
        </div>
      </div>

      <div className={classes.body} ref={bodyRef}>
        <div className={classes.lines}>
          <div aria-hidden style={{ height: centerSpacerPx }} />
          {mainLines.map((line, idx) => {
            const trLine = trByMainIndex[idx]
            const prLine = prByMainIndex[idx]
            const showTr = shouldShowAuxLine(line, trLine)
            const showPr = shouldShowAuxLine(line, prLine)
            const lineStyle = getMainLineStyle(idx)
            const auxOpacity =
              lineStyle.opacity != null ? lineStyle.opacity * 0.85 : 1
            const trStyle = {
              opacity: auxOpacity,
              fontSize: lyricsSettings.tr.fontSize,
              color: getColorValue(lyricsSettings.tr.colorKey),
            }
            const prStyle = {
              opacity: auxOpacity,
              fontSize: lyricsSettings.pr.fontSize,
              color: getColorValue(lyricsSettings.pr.colorKey),
            }
            return (
              <div
                key={`line-${line.index}-${line.start ?? idx}`}
                ref={idx === activeIndex ? activeLineRef : null}
                style={{ cursor: line.start != null ? 'pointer' : undefined }}
                onClick={() => {
                  if (audioInstance && line.start != null) {
                    audioInstance.currentTime = line.start / 1000
                  }
                }}
              >
                {showTr && (
                  <KaraokeLineRow
                    line={trLine}
                    nextLineStart={null}
                    renderPlaybackMs={renderPlaybackMs}
                    className={classes.inlineTr}
                    style={trStyle}
                    tokenClassName={classes.token}
                    highlightTokens={false}
                  />
                )}
                <KaraokeLineRow
                  line={line}
                  nextLineStart={mainLines[idx + 1]?.start ?? null}
                  renderPlaybackMs={renderPlaybackMs}
                  className={classes.line}
                  style={lineStyle}
                  tokenClassName={classes.token}
                />
                {showPr && (
                  <KaraokeLineRow
                    line={prLine}
                    nextLineStart={null}
                    renderPlaybackMs={renderPlaybackMs}
                    className={classes.inlinePr}
                    style={prStyle}
                    tokenClassName={classes.token}
                  />
                )}
              </div>
            )
          })}
          <div aria-hidden style={{ height: centerSpacerPx }} />
        </div>
      </div>
    </div>
  )
}

export default KaraokeLyricsOverlay
