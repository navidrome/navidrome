import {
  KARAOKE_CHARACTER_CREST_RATIO,
  KARAOKE_CHARACTER_LIFT_EM,
  KARAOKE_CHARACTER_SETTLE_EM,
  KARAOKE_CHARACTER_WAVE_MS,
  KARAOKE_CHARACTER_WAVE_SPAN_MAX_MS,
  KARAOKE_CHARACTER_WAVE_SPAN_RATIO,
  KARAOKE_EASING,
  KARAOKE_LINE_MOTION_EASING,
} from './lyricsKaraokeConstants'

const letterMaskFadePct = 8
const letterMaskImage = `linear-gradient(90deg, #000 0%, #000 50%, transparent calc(50% + ${letterMaskFadePct}%) 100%)`

// The overlay itself is unmasked. Each letter carries a mask sized in its own
// letter widths, so a letter can be transform-composited while the sweep moves
// across the word.
export const sweepLayerStyle = {
  display: 'block',
  inset: 0,
  position: 'absolute',
  whiteSpace: 'nowrap',
}

export const buildLetterMaskStyle = (letterCount) => {
  const maskSize = `${(letterCount + 2) * 100}% 100%`
  return {
    maskImage: letterMaskImage,
    WebkitMaskImage: letterMaskImage,
    maskSize,
    WebkitMaskSize: maskSize,
    maskRepeat: 'no-repeat',
    WebkitMaskRepeat: 'no-repeat',
    maskPosition: '0% 0%',
    WebkitMaskPosition: '0% 0%',
  }
}

// The visible edge runs slightly before the word to slightly past it, so each
// letter only animates during the slice where that edge crosses its box. The
// keyframes then hold the revealed position instead of sweeping past it.
const EDGE_START_FROM = -0.1
const EDGE_START_TO = 1.1
const EDGE_END_FROM = 0
const EDGE_END_TO = 1.2

// mask-position is paint-bound rather than compositor-friendly, but it is the
// only way to get a soft per-letter wipe; only the active line owns these
// animations, and a Chromium trace measured roughly two paint passes per frame.
export const buildLetterMaskSweep = (durationMs, letterCount, index) => {
  const span = letterCount + 2
  const positionAt = (edge) => {
    const q = (0.5 * span - (edge * letterCount - index)) / (span - 1)
    return `${q * 100}% 0%`
  }
  const timeWhereStart = (value) =>
    (durationMs * (value - EDGE_START_FROM)) / (EDGE_START_TO - EDGE_START_FROM)
  const timeWhereEnd = (value) =>
    (durationMs * (value - EDGE_END_FROM)) / (EDGE_END_TO - EDGE_END_FROM)
  const startAt = (timeMs) =>
    EDGE_START_FROM + ((EDGE_START_TO - EDGE_START_FROM) * timeMs) / durationMs

  const beginMs = Math.min(
    durationMs,
    Math.max(0, timeWhereEnd(index / letterCount)),
  )
  const finishMs = Math.min(
    durationMs,
    Math.max(beginMs + 1, timeWhereStart((index + 1) / letterCount)),
  )
  const from = positionAt(startAt(beginMs))
  const to = positionAt(startAt(finishMs))

  return {
    keyframes: [
      { maskPosition: from, WebkitMaskPosition: from },
      { maskPosition: to, WebkitMaskPosition: to },
    ],
    delayMs: beginMs,
    durationMs: finishMs - beginMs,
  }
}

export const characterAnimationKeyframes = [
  {
    transform: 'translateY(0)',
    easing: KARAOKE_EASING,
  },
  {
    transform: `translateY(-${KARAOKE_CHARACTER_LIFT_EM}em)`,
    offset: KARAOKE_CHARACTER_CREST_RATIO,
    easing: KARAOKE_LINE_MOTION_EASING,
  },
  {
    transform: `translateY(-${KARAOKE_CHARACTER_SETTLE_EM}em)`,
  },
]

export const getTokenDurationMs = (window) => {
  const start = Number(window?.start)
  const end = Number(window?.end)
  if (!Number.isFinite(start) || !Number.isFinite(end) || end <= start) {
    return 1
  }
  return end - start
}

export const buildCharacterWavePlan = (window, characterCount) => {
  const duration = getTokenDurationMs(window)
  const waveSpan =
    characterCount <= 1
      ? 0
      : Math.min(
          duration * KARAOKE_CHARACTER_WAVE_SPAN_RATIO,
          KARAOKE_CHARACTER_WAVE_SPAN_MAX_MS,
        )

  return Array.from({ length: characterCount }, (_, index) => ({
    delayMs:
      characterCount <= 1 ? 0 : (index / (characterCount - 1)) * waveSpan,
    durationMs: KARAOKE_CHARACTER_WAVE_MS,
  }))
}

export const isGecko = () => {
  try {
    return (
      typeof CSS !== 'undefined' &&
      typeof CSS.supports === 'function' &&
      CSS.supports('-moz-appearance', 'none')
    )
  } catch {
    return false
  }
}

// WebRender redraws a cached tile only after a quarter device pixel of movement,
// so small letter lifts step. An invisible additive opacity ripple changes every
// frame and forces the redraw.
export const GECKO_REDRAW_MS = 200
export const GECKO_REDRAW_KEYFRAMES = [
  { opacity: 0 },
  { opacity: -0.002 },
  { opacity: 0 },
]
