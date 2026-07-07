export const KARAOKE_CLOCK_DRIFT_RESET_MS = 140
export const KARAOKE_CLOCK_RESET_THRESHOLD_MS = 320
export const KARAOKE_MONOTONIC_JITTER_MS = 60
export const KARAOKE_RENDER_UPDATE_EPSILON_MS = 6
export const KARAOKE_HIGHLIGHT_LEAD_MS = 120
export const KARAOKE_ANIMATION_MS = 150
export const KARAOKE_SCROLLBAR_VISIBLE_MS = 1400
export const KARAOKE_MANUAL_SCROLL_PAUSE_MS = 2200
export const KARAOKE_SCROLL_ANIMATION_MS = 260
export const KARAOKE_WORD_SETTLE_MS = 96
export const KARAOKE_LINE_ENTER_MS = 180
export const KARAOKE_LINE_RELEASE_MS = 180
export const KARAOKE_SCROLL_PRE_ROLL_MS = 320
export const KARAOKE_DESKTOP_ACTIVE_LINE_ANCHOR_RATIO = 0.1
export const KARAOKE_INLINE_ACTIVE_LINE_ANCHOR_RATIO = 0.42
export const KARAOKE_SCROLL_SETTLE_PX = 2
export const KARAOKE_AUX_LINE_HEIGHT = 1.18
export const KARAOKE_EASING = 'cubic-bezier(0.22, 1, 0.36, 1)'
export const TOKEN_DONE_ALPHA = 1
export const TOKEN_FUTURE_ALPHA = 0.34
export const TOKEN_ACTIVE_ALPHA = 1
export const TOKEN_WIPE_SOFT_SPREAD_PCT = 12
export const TOKEN_WIPE_EDGE_PCT = 8
export const TOKEN_SHORT_DURATION_MS = 180

export const clamp = (value, min, max) => Math.min(max, Math.max(min, value))
export const lerp = (from, to, progress) => from + (to - from) * progress
export const easeInOut = (value) => {
  const clamped = clamp(value, 0, 1)
  return clamped < 0.5 ? 2 * clamped * clamped : 1 - (-2 * clamped + 2) ** 2 / 2
}
