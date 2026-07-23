export const LYRICS_SIDEBAR_STORAGE_KEY = 'nd.lyricsSidebar.width.v1'
export const LYRICS_SIDEBAR_WIDTH_EVENT = 'nd:lyrics-sidebar-width'
export const LYRICS_SIDEBAR_DEFAULT_WIDTH = 360
export const LYRICS_SIDEBAR_MIN_WIDTH = 300
export const LYRICS_SIDEBAR_MAX_WIDTH = 520
export const LYRICS_SIDEBAR_WIDTH_STEP = 20
export const LYRICS_SIDEBAR_TRANSITION_MS = 260

export const clampSidebarWidth = (value) => {
  const numeric = Number(value)
  if (!Number.isFinite(numeric)) return LYRICS_SIDEBAR_DEFAULT_WIDTH
  return Math.round(
    Math.min(
      LYRICS_SIDEBAR_MAX_WIDTH,
      Math.max(LYRICS_SIDEBAR_MIN_WIDTH, numeric),
    ),
  )
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

export const notifySidebarWidthChange = (width) => {
  if (typeof window === 'undefined') return

  window.dispatchEvent(
    new CustomEvent(LYRICS_SIDEBAR_WIDTH_EVENT, {
      detail: { width: clampSidebarWidth(width) },
    }),
  )
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
