import { resolveKaraokeTokenWindow } from './lyrics'
import {
  KARAOKE_LINE_ENTER_MS,
  KARAOKE_LINE_RELEASE_MS,
  KARAOKE_SCROLL_PRE_ROLL_MS,
  clamp,
  easeInOut,
} from './lyricsKaraokeConstants'

export const toFinitePlaybackMs = (value) =>
  Number.isFinite(Number(value)) ? Number(value) : 0

export const getLineRenderWindow = (line, nextLineStart) => {
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

export const getLineFinishedTime = (line, nextLineStart) => {
  const tokens = Array.isArray(line?.tokens) ? line.tokens : []
  for (let i = tokens.length - 1; i >= 0; i -= 1) {
    const { end } = resolveKaraokeTokenWindow(line, i, nextLineStart)
    if (end != null) return end
  }

  return getLineRenderWindow(line, nextLineStart).end
}

export const getLineLifecycleState = (line, nextLineStart, currentTimeMs) => {
  const current = toFinitePlaybackMs(currentTimeMs)
  const { start } = getLineRenderWindow(line, nextLineStart)
  const end = getLineFinishedTime(line, nextLineStart)

  if (start == null) {
    return {
      phase: 'idle',
      isActive: false,
      isRelease: false,
      isAnimating: false,
      highlightAlphaScale: 0,
      lineFocusScale: 0,
    }
  }

  const hasKnownEnd = end != null
  const isActive = current >= start && (!hasKnownEnd || current < end)
  const isRelease =
    hasKnownEnd && current >= end && current < end + KARAOKE_LINE_RELEASE_MS
  const enterProgress = isActive
    ? clamp((current - start) / KARAOKE_LINE_ENTER_MS, 0, 1)
    : 0
  const releaseProgress =
    isRelease && hasKnownEnd
      ? clamp((current - end) / KARAOKE_LINE_RELEASE_MS, 0, 1)
      : 0
  const lineFocusScale = isRelease
    ? 1 - easeInOut(releaseProgress)
    : isActive
      ? easeInOut(enterProgress)
      : 0

  return {
    phase: isActive ? 'active' : isRelease ? 'release' : 'idle',
    isActive,
    isRelease,
    isAnimating: isActive || isRelease,
    highlightAlphaScale: lineFocusScale,
    lineFocusScale,
  }
}

export const getStrictActiveLineIndex = (lines, currentTimeMs) => {
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

export const shouldSkipLineFrame = (
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
