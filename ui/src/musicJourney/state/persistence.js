/**
 * localStorage persistence for Music Journey history and last preferences.
 * Versioned key so future schema changes can migrate or drop old data.
 */
const STORAGE_KEY = 'musicJourney.v1'

export const HISTORY_LIMIT = 50
export const TRACK_IDS_LIMIT = 500

export function loadPersistedJourney() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return null
    const parsed = JSON.parse(raw)
    return {
      history: Array.isArray(parsed.history) ? parsed.history : [],
      preferences: parsed.preferences || null,
    }
  } catch {
    return null
  }
}

export function persistJourney({ history, preferences }) {
  try {
    localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({
        history: (history || []).slice(0, HISTORY_LIMIT),
        preferences,
      }),
    )
  } catch {
    // Storage may be full or unavailable; history is not critical data.
  }
}

/** Convert an active journey into a compact history record. */
export function journeyToHistoryEntry(journey, now = Date.now()) {
  return {
    id: journey.id,
    title: journey.title,
    createdAt: now,
    mood: journey.preferences.mood,
    durationMinutes: journey.preferences.durationMinutes,
    intensity: journey.preferences.intensity,
    discovery: journey.preferences.discovery,
    genre: journey.preferences.genre,
    decade: journey.preferences.decade,
    trackIds: journey.tracks.map((t) => t.track.id).slice(0, TRACK_IDS_LIMIT),
    trackCount: journey.tracks.length,
    estimatedDuration: journey.estimatedDuration,
    narrative: journey.narrative.map((p) => p.name),
    status: 'active',
    skipped: 0,
    liked: false,
    version: journey.version || 1,
  }
}
