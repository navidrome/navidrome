export const KARAOKE_CLOCK_DRIFT_RESET_MS = 140
export const KARAOKE_HIGHLIGHT_LEAD_MS = 120
export const KARAOKE_ANIMATION_MS = 500
export const KARAOKE_LINE_OPACITY_MS = 200
export const KARAOKE_SCROLLBAR_VISIBLE_MS = 1400
export const KARAOKE_MANUAL_SCROLL_PAUSE_MS = 2200
export const KARAOKE_SCROLL_ANIMATION_MS = 300
export const KARAOKE_LINE_ENTER_MS = KARAOKE_LINE_OPACITY_MS
export const KARAOKE_LINE_LIFT_PX = 1.5
const KARAOKE_EASE_IN_OUT_X1 = 0.77
const KARAOKE_EASE_IN_OUT_Y1 = 0
const KARAOKE_EASE_IN_OUT_X2 = 0.175
const KARAOKE_EASE_IN_OUT_Y2 = 1
export const KARAOKE_LINE_MOTION_EASING = `cubic-bezier(${KARAOKE_EASE_IN_OUT_X1}, ${KARAOKE_EASE_IN_OUT_Y1}, ${KARAOKE_EASE_IN_OUT_X2}, ${KARAOKE_EASE_IN_OUT_Y2})`
export const KARAOKE_CHARACTER_LIFT_PX = KARAOKE_LINE_LIFT_PX
export const KARAOKE_CHARACTER_RISE_MS = KARAOKE_ANIMATION_MS
export const KARAOKE_CHARACTER_WAVE_SPAN_RATIO = 0.36
export const KARAOKE_CHARACTER_WAVE_SPAN_MAX_MS = 140
export const KARAOKE_TRANSLATION_OPACITY = 0.62
export const KARAOKE_IDLE_LAYER_OPACITY = 0.49
export const KARAOKE_LINE_RELEASE_MS = KARAOKE_LINE_OPACITY_MS
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

const cubicBezierCoordinate = (time, firstControl, secondControl) => {
  const inverse = 1 - time
  return (
    3 * inverse * inverse * time * firstControl +
    3 * inverse * time * time * secondControl +
    time * time * time
  )
}

const cubicBezierSlope = (time, firstControl, secondControl) => {
  const inverse = 1 - time
  return (
    3 * inverse * inverse * firstControl +
    6 * inverse * time * (secondControl - firstControl) +
    3 * time * time * (1 - secondControl)
  )
}

const solveStrongEaseInOutTime = (progress) => {
  let time = progress
  for (let iteration = 0; iteration < 6; iteration += 1) {
    const error =
      cubicBezierCoordinate(
        time,
        KARAOKE_EASE_IN_OUT_X1,
        KARAOKE_EASE_IN_OUT_X2,
      ) - progress
    if (Math.abs(error) < 0.000001) return time
    const slope = cubicBezierSlope(
      time,
      KARAOKE_EASE_IN_OUT_X1,
      KARAOKE_EASE_IN_OUT_X2,
    )
    if (Math.abs(slope) < 0.000001) break
    time = clamp(time - error / slope, 0, 1)
  }

  let lower = 0
  let upper = 1
  for (let iteration = 0; iteration < 12; iteration += 1) {
    time = (lower + upper) / 2
    if (
      cubicBezierCoordinate(
        time,
        KARAOKE_EASE_IN_OUT_X1,
        KARAOKE_EASE_IN_OUT_X2,
      ) < progress
    ) {
      lower = time
    } else {
      upper = time
    }
  }
  return time
}

export const easeKaraokeMotion = (value) => {
  const progress = clamp(value, 0, 1)
  if (progress === 0 || progress === 1) return progress
  return cubicBezierCoordinate(
    solveStrongEaseInOutTime(progress),
    KARAOKE_EASE_IN_OUT_Y1,
    KARAOKE_EASE_IN_OUT_Y2,
  )
}

export const easeInOut = (value) => {
  const progress = clamp(value, 0, 1)
  return progress < 0.5
    ? 2 * progress * progress
    : 1 - (-2 * progress + 2) ** 2 / 2
}
