// Local-only lyrics preferences, keyed per track (and per source for the
// timing offset). Storage can be unavailable (private mode, blocked
// third-party contexts), so every access degrades to session behavior.

const storage = () =>
  typeof window === 'undefined' ? null : window.localStorage

const preferredSourceKey = (trackId) => `lyrics.source:${trackId}`

const offsetStorageKey = (trackId, sourceKey) =>
  `lyrics.offset:${trackId}:${sourceKey}`

export const readPreferredSource = (trackId) => {
  if (!trackId) return null
  try {
    const value = storage()?.getItem(preferredSourceKey(trackId))
    return value && value.trim() !== '' ? value : null
  } catch {
    return null
  }
}

export const writePreferredSource = (trackId, sourceKey) => {
  if (!trackId) return
  try {
    const key = preferredSourceKey(trackId)
    if (sourceKey && String(sourceKey).trim() !== '') {
      storage()?.setItem(key, String(sourceKey))
    } else {
      storage()?.removeItem(key)
    }
  } catch {
    // Session-only fallback.
  }
}

export const readLyricsOffset = (trackId, sourceKey) => {
  if (!trackId || !sourceKey) return 0
  try {
    const raw = storage()?.getItem(offsetStorageKey(trackId, sourceKey))
    if (raw == null || raw === '') return 0
    const value = Number(raw)
    return Number.isFinite(value) ? value : 0
  } catch {
    return 0
  }
}

export const writeLyricsOffset = (trackId, sourceKey, offsetMs) => {
  if (!trackId || !sourceKey) return
  try {
    const key = offsetStorageKey(trackId, sourceKey)
    const value = Number(offsetMs)
    if (!Number.isFinite(value) || value === 0) {
      storage()?.removeItem(key)
    } else {
      storage()?.setItem(key, String(value))
    }
  } catch {
    // Session-only fallback.
  }
}
