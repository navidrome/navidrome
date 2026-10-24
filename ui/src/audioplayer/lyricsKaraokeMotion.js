import {
  KARAOKE_CHARACTER_CREST_RATIO,
  KARAOKE_CHARACTER_LIFT_EM,
  KARAOKE_CHARACTER_SETTLE_EM,
  KARAOKE_CHARACTER_WAVE_MS,
  KARAOKE_CHARACTER_WAVE_SPAN_MAX_MS,
  KARAOKE_CHARACTER_WAVE_SPAN_RATIO,
  KARAOKE_EASING,
  KARAOKE_LINE_MOTION_EASING,
  TOKEN_WIPE_EDGE_PCT,
} from './lyricsKaraokeConstants'

const sweepFadePct = TOKEN_WIPE_EDGE_PCT / 2
// Sliding the mask one full edge-width past the token hides the trailing fade
// as well as the text, otherwise the first token letters peek through at rest.
export const sweepStartPct = 100 + TOKEN_WIPE_EDGE_PCT

// The overlay is masked by a static gradient twice its width. Sliding that mask
// from 112% (edge before the token) to 0% (edge past it) sweeps the reveal, and
// the whole thing is one compositor-friendly property instead of per-glyph paint.
const sweepMaskImage = `linear-gradient(90deg, #000 0%, #000 50%, transparent calc(50% + ${sweepFadePct}%) 100%)`

export const sweepLayerStyle = {
  display: 'block',
  inset: 0,
  position: 'absolute',
  whiteSpace: 'nowrap',
  maskImage: sweepMaskImage,
  WebkitMaskImage: sweepMaskImage,
  maskSize: '200% 100%',
  WebkitMaskSize: '200% 100%',
  maskRepeat: 'no-repeat',
  WebkitMaskRepeat: 'no-repeat',
  maskPosition: `${sweepStartPct}% 0%`,
  WebkitMaskPosition: `${sweepStartPct}% 0%`,
}

export const sweepAnimationKeyframes = [
  {
    maskPosition: `${sweepStartPct}% 0%`,
    WebkitMaskPosition: `${sweepStartPct}% 0%`,
  },
  {
    maskPosition: '0% 0%',
    WebkitMaskPosition: '0% 0%',
  },
]

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
