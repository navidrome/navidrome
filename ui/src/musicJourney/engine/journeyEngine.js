/**
 * MusicJourneyEngine — turns a pool of Navidrome songs and a set of
 * preferences into a narrative track sequence.
 *
 * Design notes:
 *  - No randomness sources other than the injectable `rng`, which makes
 *    generation reproducible in tests.
 *  - The engine never touches the network; the pool arrives pre-fetched and
 *    normalized. This keeps it trivially testable and lets a future ML
 *    strategy provider wrap or replace the scoring loop.
 *  - Selection is "scored top-K sampling", not shuffle and not pure argmax:
 *    candidates are scored against the *current phase target* and the
 *    previous track, then one of the top picks is drawn with a temperature
 *    controlled by the discovery slider.
 */
import {
  MIN_JOURNEY_TRACKS,
  hashSeed,
  mulberry32,
  normalizePreferences,
  normalizeSong,
} from './types'
import { getTrackProfile } from './trackProfile'
import { buildUserProfile } from './personalization'
import { buildReason, scoreCandidate, skipPenalty } from './scoring'
import { transitionScore } from './transitions'
import { allocatePhaseDurations, buildNarrative } from './phases'
import { generateJourneyName } from './nameGenerator'

/** Error with a stable machine-readable code (UI maps it to friendly copy). */
export class JourneyError extends Error {
  constructor(code, details = {}) {
    super(code)
    this.code = code
    this.details = details
  }
}

export const JOURNEY_ERROR_CODES = {
  EMPTY_LIBRARY: 'EMPTY_LIBRARY',
  NOT_ENOUGH_TRACKS: 'NOT_ENOUGH_TRACKS',
}

const sameDecade = (year, decade) =>
  year && decade && Math.floor(year / 10) * 10 === Number(decade)

/** Hard filters: era selection and explicit genre exclusions. */
export function applyHardFilters(songs, prefs) {
  return songs.filter((s) => {
    if (!s || s.missing) return false
    if (prefs.decade && !sameDecade(s.year, prefs.decade)) return false
    if (
      prefs.excludeGenres?.length &&
      prefs.excludeGenres.some(
        (g) =>
          String(s.genre || '')
            .trim()
            .toLowerCase() === g.toLowerCase(),
      )
    ) {
      return false
    }
    return true
  })
}

/**
 * Pick the next track: score every remaining candidate for the phase, then
 * sample from the top-K. Discovery flattens the sampling temperature so
 * surprising (lower-ranked but valid) picks can surface.
 */
function chooseNext({
  candidates,
  usedIds,
  phase,
  prefs,
  userProfile,
  profileOf,
  prev,
  artistCounts,
  skipEvents,
  now,
  rng,
}) {
  const scored = []
  for (const song of candidates) {
    if (usedIds.has(song.id)) continue
    const profile = profileOf(song)
    const artistKey = song.artistId || String(song.artist || '').toLowerCase()
    const genreKey = String(song.genre || '')
      .trim()
      .toLowerCase()
    const bias = skipPenalty(
      { genre: genreKey, artistKey, energy: profile.energy },
      skipEvents,
      now,
    )
    const result = scoreCandidate({
      song,
      profile,
      phaseTarget: { energy: phase.targetEnergy, valence: phase.targetValence },
      prefs,
      userProfile,
      skipBias: bias,
      artistRepeats: artistCounts.get(artistKey) || 0,
      prev,
    })
    scored.push({ song, profile, ...result })
  }
  if (!scored.length) return null

  scored.sort((a, b) => b.total - a.total)
  const k = Math.min(scored.length, Math.round(8 + prefs.discovery * 20))
  const top = scored.slice(0, k)

  // Temperature: low discovery -> strongly prefer the very best match;
  // high discovery -> flatter distribution over the top-K.
  const sharpness = 6 - prefs.discovery * 4.5
  const weights = top.map((c) => Math.pow(Math.max(c.total, 0.01), sharpness))
  const sum = weights.reduce((a, b) => a + b, 0)
  let pick = rng() * sum
  let chosen = top[top.length - 1]
  for (let i = 0; i < top.length; i++) {
    pick -= weights[i]
    if (pick <= 0) {
      chosen = top[i]
      break
    }
  }

  const transition = prev
    ? transitionScore(
        { song: prev.song, profile: prev },
        { song: chosen.song, profile: chosen.profile },
        prefs,
      )
    : null

  const reason = buildReason({
    song: chosen.song,
    components: chosen.components,
    phaseTarget: { energy: phase.targetEnergy, valence: phase.targetValence },
    prefs,
    profile: chosen.profile,
  })

  return { ...chosen, reason, transitionPenalties: transition?.penalties || [] }
}

/**
 * Build a complete journey.
 *
 * @param {Object} options
 * @param {Array<import('./types').Song>} options.songs raw or normalized songs
 * @param {import('./types').JourneyPreferences} options.prefs
 * @param {Array} [options.skipEvents] recent skip feedback
 * @param {number} [options.now]
 * @param {Function} [options.rng]
 * @param {Date} [options.date] used for title generation
 * @param {string} [options.id] stable journey id (regenerations keep theirs)
 * @param {number} [options.minTracks] minimum pool size; only lowered by
 *   live regeneration so a journey can continue on a nearly-exhausted pool
 * @returns {import('./types').MusicJourney & {truncated: boolean, version: number}}
 */
export function buildJourney({
  songs,
  prefs,
  skipEvents = [],
  now = Date.now(),
  rng,
  date = new Date(now),
  id = null,
  minTracks = MIN_JOURNEY_TRACKS,
}) {
  const preferences = normalizePreferences(prefs)
  const pool = (songs || []).map((s) => normalizeSong(s)).filter(Boolean)

  if (!pool.length) {
    throw new JourneyError(JOURNEY_ERROR_CODES.EMPTY_LIBRARY)
  }

  const candidates = applyHardFilters(pool, preferences)
  if (candidates.length < minTracks) {
    throw new JourneyError(JOURNEY_ERROR_CODES.NOT_ENOUGH_TRACKS, {
      available: candidates.length,
      required: minTracks,
    })
  }

  const seedSource =
    preferences.seed != null
      ? String(preferences.seed)
      : `${preferences.mood}|${now}|${candidates.length}`
  const random = rng || mulberry32(hashSeed(seedSource))

  const profileCache = new Map()
  const profileOf = (song) => {
    if (!profileCache.has(song.id)) {
      profileCache.set(song.id, getTrackProfile(song, now))
    }
    return profileCache.get(song.id)
  }

  const userProfile = buildUserProfile(pool, profileOf)
  const narrative = buildNarrative(preferences)
  const allocation = allocatePhaseDurations(
    narrative,
    preferences.durationMinutes * 60,
  )

  const tracks = []
  const usedIds = new Set()
  const artistCounts = new Map()
  let prev = null
  let truncated = false

  for (const { phase, seconds } of allocation) {
    let phaseSeconds = 0
    while (phaseSeconds < seconds && usedIds.size < candidates.length) {
      const chosen = chooseNext({
        candidates,
        usedIds,
        phase,
        prefs: preferences,
        userProfile,
        profileOf,
        prev,
        artistCounts,
        skipEvents,
        now,
        rng: random,
      })
      if (!chosen) break
      usedIds.add(chosen.song.id)
      const artistKey =
        chosen.song.artistId || String(chosen.song.artist || '').toLowerCase()
      artistCounts.set(artistKey, (artistCounts.get(artistKey) || 0) + 1)
      tracks.push({
        track: chosen.song,
        score: chosen.total,
        reason: chosen.reason,
        stage: { id: phase.id, name: phase.name },
      })
      phaseSeconds += chosen.song.duration
      prev = { ...chosen.profile, song: chosen.song }
    }
    if (usedIds.size >= candidates.length) truncated = true
  }

  if (tracks.length < minTracks) {
    throw new JourneyError(JOURNEY_ERROR_CODES.NOT_ENOUGH_TRACKS, {
      available: candidates.length,
      required: minTracks,
    })
  }

  return {
    id: id || `j-${hashSeed(seedSource).toString(36)}-${now.toString(36)}`,
    title: generateJourneyName(preferences, date),
    preferences,
    tracks,
    estimatedDuration: tracks.reduce((s, t) => s + t.track.duration, 0),
    narrative: narrative.map((p) => ({ id: p.id, name: p.name })),
    truncated,
    version: 1,
  }
}

/**
 * Rebuild only the remaining part of a journey while listening.
 *
 * `playedCount` covers tracks that were already played *plus* the currently
 * playing one — those are copied verbatim (same objects, same stages), so
 * the rebuild can never alter what the user already heard.
 *
 * @returns {import('./types').MusicJourney & {truncated: boolean, version: number, keptCount: number}}
 */
export function regenerateRemaining({
  journey,
  playedCount,
  prefs,
  songs,
  skipEvents = [],
  now = Date.now(),
  rng,
  date = new Date(now),
}) {
  if (!journey || !Array.isArray(journey.tracks)) {
    throw new JourneyError(JOURNEY_ERROR_CODES.EMPTY_LIBRARY)
  }
  const kept = journey.tracks.slice(0, Math.max(0, playedCount))
  const keptIds = new Set(kept.map((t) => t.track.id))

  const preferences = normalizePreferences(prefs)
  const pool = (songs || [])
    .map((s) => normalizeSong(s))
    .filter((s) => s && !keptIds.has(s.id))

  const keptSeconds = kept.reduce((s, t) => s + t.track.duration, 0)
  const remainingMinutes = Math.max(
    5,
    Math.round((journey.preferences.durationMinutes * 60 - keptSeconds) / 60),
  )

  let freshTracks = []
  let truncated = false
  try {
    const rebuilt = buildJourney({
      songs: pool,
      prefs: { ...preferences, durationMinutes: remainingMinutes },
      skipEvents,
      now,
      rng,
      date,
      id: journey.id,
    })
    freshTracks = rebuilt.tracks
    truncated = rebuilt.truncated
  } catch (err) {
    // Not enough fresh material for the full remaining time: degrade in two
    // steps instead of killing the journey mid-flight — first a shorter
    // window, then whatever is left at all (a live journey should never
    // die of starvation when *some* music still exists).
    if (
      err instanceof JourneyError &&
      err.code === JOURNEY_ERROR_CODES.NOT_ENOUGH_TRACKS
    ) {
      const rebuilt = buildJourney({
        songs: pool,
        prefs: { ...preferences, durationMinutes: 5 },
        skipEvents,
        now,
        rng,
        date,
        id: journey.id,
        minTracks: 1,
      })
      freshTracks = rebuilt.tracks
      truncated = true
    } else {
      throw err
    }
  }

  // Fresh phases come from the *new* narrative; kept tracks keep theirs.
  const freshNarrative = buildNarrative({
    ...preferences,
    durationMinutes: remainingMinutes,
  })
  const narrative = [
    ...new Map([
      ...kept.map((t) => [t.stage.id, t.stage]),
      ...freshNarrative.map((p) => [p.id, { id: p.id, name: p.name }]),
    ]).values(),
  ]

  const tracks = [...kept, ...freshTracks]
  return {
    ...journey,
    preferences,
    tracks,
    estimatedDuration: tracks.reduce((s, t) => s + t.track.duration, 0),
    narrative,
    truncated,
    version: (journey.version || 1) + 1,
    keptCount: kept.length,
  }
}
