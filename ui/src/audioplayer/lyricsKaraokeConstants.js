export const KARAOKE_CLOCK_DRIFT_RESET_MS = 140
export const KARAOKE_ANIMATION_SYNC_DRIFT_MS = 80
export const KARAOKE_HIGHLIGHT_LEAD_MS = 120
export const KARAOKE_LINE_OPACITY_MS = 240
export const KARAOKE_LINE_BLUR_MS = 150
export const KARAOKE_SCROLLBAR_VISIBLE_MS = 1400
export const KARAOKE_SCROLL_ANIMATION_MS = 300
export const KARAOKE_SCROLL_ANIMATION_MAX_MS = 500
const KARAOKE_EASE_IN_OUT_X1 = 0.77
const KARAOKE_EASE_IN_OUT_Y1 = 0
const KARAOKE_EASE_IN_OUT_X2 = 0.175
const KARAOKE_EASE_IN_OUT_Y2 = 1
const KARAOKE_OPACITY_EASING_X1 = 0.22
const KARAOKE_OPACITY_EASING_Y1 = 1
const KARAOKE_OPACITY_EASING_X2 = 0.36
const KARAOKE_OPACITY_EASING_Y2 = 1
export const KARAOKE_LINE_MOTION_EASING = `cubic-bezier(${KARAOKE_EASE_IN_OUT_X1}, ${KARAOKE_EASE_IN_OUT_Y1}, ${KARAOKE_EASE_IN_OUT_X2}, ${KARAOKE_EASE_IN_OUT_Y2})`
export const KARAOKE_CHARACTER_LIFT_EM = 0.04
export const KARAOKE_CHARACTER_SETTLE_EM = 0.015
export const KARAOKE_CHARACTER_WAVE_MS = 750
export const KARAOKE_CHARACTER_CREST_RATIO = 0.4
export const KARAOKE_CHARACTER_WAVE_SPAN_RATIO = 0.36
export const KARAOKE_CHARACTER_WAVE_SPAN_MAX_MS = 140
export const KARAOKE_IDLE_LAYER_OPACITY = 0.49
export const KARAOKE_LINE_RELEASE_MS = KARAOKE_LINE_OPACITY_MS
export const MOBILE_KARAOKE_LYRICS_TRANSITION_MS = 260
export const KARAOKE_SCROLL_PRE_ROLL_MS = 320
export const KARAOKE_ANIMATION_PRELOAD_MS = 2000
export const KARAOKE_DESKTOP_ACTIVE_LINE_ANCHOR_RATIO = 0.1
export const KARAOKE_INLINE_ACTIVE_LINE_ANCHOR_RATIO = 0.42
export const KARAOKE_SCROLL_SETTLE_PX = 2
export const KARAOKE_AUX_LINE_HEIGHT = 1.18
export const KARAOKE_EASING = `cubic-bezier(${KARAOKE_OPACITY_EASING_X1}, ${KARAOKE_OPACITY_EASING_Y1}, ${KARAOKE_OPACITY_EASING_X2}, ${KARAOKE_OPACITY_EASING_Y2})`
// State fades start gently; the strong ease-out above makes a 240ms unhighlight
// read as a snap on lines that are ending.
export const KARAOKE_FADE_EASING = 'cubic-bezier(0.4, 0, 0.2, 1)'
export const TOKEN_FUTURE_ALPHA = 0.56
export const TOKEN_ACTIVE_ALPHA = 1

export const clamp = (value, min, max) => Math.min(max, Math.max(min, value))

export const easeInOut = (value) => {
  const progress = clamp(value, 0, 1)
  return progress < 0.5
    ? 2 * progress * progress
    : 1 - (-2 * progress + 2) ** 2 / 2
}
