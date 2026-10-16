export const LYRICS_SIDEBAR_STORAGE_KEY = 'nd.lyricsSidebar.width.v1'
export const LYRICS_SIDEBAR_DEFAULT_WIDTH = 360
export const LYRICS_SIDEBAR_MIN_WIDTH = 300
export const LYRICS_SIDEBAR_MAX_WIDTH = 520
export const LYRICS_ROUTE_MIN_WIDTH = 320
export const LYRICS_SIDEBAR_WIDTH_STEP = 20
export const LYRICS_SIDEBAR_TRANSITION_MS = 260

export const clampSidebarMaxWidth = (value) => {
  const numeric = Number(value)
  if (!Number.isFinite(numeric)) return LYRICS_SIDEBAR_MAX_WIDTH
  return Math.round(
    Math.min(
      LYRICS_SIDEBAR_MAX_WIDTH,
      Math.max(LYRICS_SIDEBAR_MIN_WIDTH, numeric),
    ),
  )
}

export const clampSidebarWidth = (
  value,
  maxWidth = LYRICS_SIDEBAR_MAX_WIDTH,
) => {
  const resolvedMaxWidth = clampSidebarMaxWidth(maxWidth)
  const numeric = Number(value)
  if (!Number.isFinite(numeric)) {
    return Math.min(LYRICS_SIDEBAR_DEFAULT_WIDTH, resolvedMaxWidth)
  }
  return Math.round(
    Math.min(resolvedMaxWidth, Math.max(LYRICS_SIDEBAR_MIN_WIDTH, numeric)),
  )
}

export const getSidebarMaxWidth = (contentWidth) => {
  const numeric = Number(contentWidth)
  if (!Number.isFinite(numeric) || numeric <= 0) {
    return LYRICS_SIDEBAR_MAX_WIDTH
  }
  return clampSidebarMaxWidth(numeric - LYRICS_ROUTE_MIN_WIDTH)
}

const hasLocalStorage = () => {
  try {
    return typeof window !== 'undefined' && Boolean(window.localStorage)
  } catch {
    return false
  }
}

export const loadSidebarWidth = () => {
  if (!hasLocalStorage()) return LYRICS_SIDEBAR_DEFAULT_WIDTH
  try {
    const stored = window.localStorage.getItem(LYRICS_SIDEBAR_STORAGE_KEY)
    return stored == null
      ? LYRICS_SIDEBAR_DEFAULT_WIDTH
      : clampSidebarWidth(stored)
  } catch {
    return LYRICS_SIDEBAR_DEFAULT_WIDTH
  }
}

export const saveSidebarWidth = (width) => {
  const resolvedWidth = clampSidebarWidth(width)
  if (!hasLocalStorage()) return
  try {
    window.localStorage.setItem(
      LYRICS_SIDEBAR_STORAGE_KEY,
      String(resolvedWidth),
    )
  } catch {
    // Ignore storage failures.
  }
}
