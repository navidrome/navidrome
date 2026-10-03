/**
 * Music Journey — core types and constants.
 *
 * The project is JavaScript-first (JSDoc instead of TS interfaces), so the
 * shapes are documented here and validated at runtime by the functions in
 * this file. Every record that crosses the Navidrome API boundary should go
 * through `normalizeSong()` so the engine never chokes on missing metadata.
 */

/**
 * Mood presets available in the Journey Creator.
 * @type {ReadonlyArray<string>}
 */
export const MOODS = ['chill', 'focus', 'energy', 'happy', 'melancholic']

/** Minimum number of distinct tracks required to start a journey. */
export const MIN_JOURNEY_TRACKS = 5

/** Hard cap of candidates the engine will ever score in one generation. */
export const MAX_CANDIDATES = 3000

/**
 * @typedef {Object} Song
 * A normalized Navidrome song (fields mirrored from /api/song).
 * @property {string} id
 * @property {string} title
 * @property {string} artist
 * @property {string} [artistId]
 * @property {string} [album]
 * @property {string} [albumId]
 * @property {string} [genre]
 * @property {number} [year]
 * @property {number} [duration] seconds
 * @property {number|null} [bpm]
 * @property {number} [playCount]
 * @property {string|null} [playDate] ISO date of last play
 * @property {number} [rating] 0-5
 * @property {boolean} [starred]
 * @property {string|null} [starredAt]
 * @property {boolean} [missing]
 */

/**
 * @typedef {Object} JourneyPreferences
 * @property {string} mood one of MOODS
 * @property {number} durationMinutes 10-180
 * @property {number} intensity 0..1 (calm .. high)
 * @property {number} discovery 0..1 (familiar .. surprise)
 * @property {string} [genre] optional genre name filter
 * @property {string} [genreId] optional genre id filter (used for API calls)
 * @property {string} [decade] optional decade, e.g. "1990"
 * @property {string[]} [excludeGenres] genres to avoid (from live controls)
 * @property {string} [seed] optional RNG seed for reproducibility
 */

/**
 * @typedef {Object} JourneyStage
 * One narrative phase of a journey.
 * @property {string} id
 * @property {string} name
 * @property {number} weight relative share of the journey duration
 * @property {number} energyBias multiplier applied to the base energy target
 */

/**
 * @typedef {Object} JourneyTrack
 * @property {Song} track
 * @property {number} score total score used for selection
 * @property {string} [reason] human readable explanation of the pick
 * @property {JourneyStage} stage
 */

/**
 * @typedef {Object} MusicJourney
 * @property {string} id
 * @property {string} title
 * @property {JourneyPreferences} preferences
 * @property {JourneyTrack[]} tracks
 * @property {number} estimatedDuration seconds
 * @property {JourneyStage[]} narrative
 */

/** Small deterministic PRNG so tests (and replayable journeys) are stable. */
export function mulberry32(seed) {
  let a = seed >>> 0
  return function () {
    a |= 0
    a = (a + 0x6d2b79f5) | 0
    let t = Math.imul(a ^ (a >>> 15), 1 | a)
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }
}

/** Derive a numeric seed from any string (used for journey titles/ids). */
export function hashSeed(str) {
  let h = 2166136261
  for (let i = 0; i < str.length; i++) {
    h ^= str.charCodeAt(i)
    h = Math.imul(h, 16777619)
  }
  return h >>> 0
}

const toDateMs = (value) => {
  if (!value) return null
  const t = new Date(value).getTime()
  return Number.isNaN(t) ? null : t
}

/**
 * Normalize a raw Navidrome song so the engine can rely on field presence.
 * Missing/unparseable values degrade to neutral defaults instead of throwing.
 */
export function normalizeSong(raw) {
  if (!raw || !raw.id) return null
  const duration = Number(raw.duration)
  return {
    id: String(raw.id),
    title: raw.title || raw.path || 'Unknown Title',
    artist: raw.artist || raw.albumArtist || 'Unknown Artist',
    artistId: raw.artistId || null,
    album: raw.album || '',
    albumId: raw.albumId || null,
    genre: typeof raw.genre === 'string' ? raw.genre : '',
    year: Number.isFinite(Number(raw.year)) ? Number(raw.year) : 0,
    duration: Number.isFinite(duration) && duration > 0 ? duration : 240,
    bpm:
      Number.isFinite(Number(raw.bpm)) && raw.bpm > 0 ? Number(raw.bpm) : null,
    playCount: Number.isFinite(Number(raw.playCount))
      ? Number(raw.playCount)
      : 0,
    playDate: toDateMs(raw.playDate),
    rating: Number.isFinite(Number(raw.rating)) ? Number(raw.rating) : 0,
    starred: Boolean(raw.starred),
    starredAt: toDateMs(raw.starredAt),
    createdAt: toDateMs(raw.createdAt),
    missing: Boolean(raw.missing),
  }
}

/** Validate user-facing preferences and clamp them into a safe range. */
export function normalizePreferences(prefs = {}) {
  const mood = MOODS.includes(prefs.mood) ? prefs.mood : 'chill'
  const clamp01 = (v, fallback) =>
    Number.isFinite(Number(v)) ? Math.min(1, Math.max(0, Number(v))) : fallback
  return {
    mood,
    durationMinutes: Math.min(
      180,
      Math.max(5, Number(prefs.durationMinutes) || 30),
    ),
    intensity: clamp01(prefs.intensity, 0.5),
    discovery: clamp01(prefs.discovery, 0.3),
    genre: prefs.genre || undefined,
    genreId: prefs.genreId || undefined,
    decade: prefs.decade || undefined,
    excludeGenres: Array.isArray(prefs.excludeGenres)
      ? prefs.excludeGenres.filter(Boolean)
      : [],
    seed: prefs.seed,
  }
}
