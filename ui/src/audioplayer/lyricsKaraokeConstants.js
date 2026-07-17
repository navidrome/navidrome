export const KARAOKE_CLOCK_DRIFT_RESET_MS = 140
export const KARAOKE_HIGHLIGHT_LEAD_MS = 120
export const KARAOKE_ANIMATION_MS = 220
export const KARAOKE_SCROLLBAR_VISIBLE_MS = 1400
export const KARAOKE_MANUAL_SCROLL_PAUSE_MS = 2200
export const KARAOKE_SCROLL_ANIMATION_MS = 300
export const KARAOKE_LINE_ENTER_MS = KARAOKE_ANIMATION_MS
export const KARAOKE_LINE_LIFT_PX = 1.5
export const KARAOKE_LINE_MOTION_EASING = 'cubic-bezier(0.25, 0.1, 0.25, 1)'
export const KARAOKE_CHARACTER_LIFT_PX = 1.5
export const KARAOKE_CHARACTER_PHASE_SPREAD = 0.36
export const KARAOKE_TRANSLATION_OPACITY = 0.62
export const KARAOKE_LINE_RELEASE_MS = KARAOKE_ANIMATION_MS
export const KARAOKE_SCROLL_PRE_ROLL_MS = 320
export const KARAOKE_DESKTOP_ACTIVE_LINE_ANCHOR_RATIO = 0.1
export const KARAOKE_INLINE_ACTIVE_LINE_ANCHOR_RATIO = 0.42
export const KARAOKE_SCROLL_SETTLE_PX = 2
export const KARAOKE_AUX_LINE_HEIGHT = 1.18
export const KARAOKE_EASING = 'cubic-bezier(0.22, 1, 0.36, 1)'
export const TOKEN_FUTURE_ALPHA = 0.34
export const TOKEN_ACTIVE_ALPHA = 1
export const TOKEN_WIPE_SOFT_SPREAD_PCT = 34
export const TOKEN_WIPE_EDGE_PCT = 12

export const clamp = (value, min, max) => Math.min(max, Math.max(min, value))
export const easeInOut = (value) => {
  const clamped = clamp(value, 0, 1)
  return clamped < 0.5 ? 2 * clamped * clamped : 1 - (-2 * clamped + 2) ** 2 / 2
}
