/**
 * Candidate scoring.
 *
 * `scoreCandidate()` decomposes the total score into named components so
 * that (a) the weights stay configurable and (b) the "Why this track?"
 * explanation can be assembled from the factors that actually dominated —
 * never from invented facts.
 */
import { JOURNEY_WEIGHTS, JOURNEY_PENALTIES } from './weights'
import { affinityOf } from './personalization'

const clamp01 = (v) => Math.min(1, Math.max(0, v))

/** How well the track matches the energy/valence target of a phase. */
export function moodCompatibility(profile, target) {
  const energyFit = 1 - Math.abs(profile.energy - target.energy)
  // Valence only matters when the mood has a strong character.
  const valenceFit =
    target.valence == null ? 1 : 1 - Math.abs(profile.valence - target.valence)
  return clamp01(energyFit * 0.7 + valenceFit * 0.3)
}

/**
 * Freshness: recently played tracks are penalized, long-unplayed or never
 * played tracks get a small bonus. The curve is forgiving for the first two
 * days so a daily driver is not instantly banned.
 */
export function freshnessScore(profile) {
  if (profile.daysSincePlay == null) return 0.75 // never played
  const d = profile.daysSincePlay
  if (d < 1) return 0.05
  if (d < 3) return 0.2
  if (d < 14) return 0.5
  if (d < 60) return 0.7
  return 0.9 // long-unplayed: close to a rediscovery
}

/** Favorites/rating/play-count signal, log-scaled to avoid top-track lock-in. */
export function userPreferenceScore(song) {
  const star = song.starred ? 0.5 : 0
  const rating = song.rating > 0 ? Math.min(1, song.rating / 5) * 0.35 : 0
  const plays =
    song.playCount > 0
      ? Math.min(1, Math.log2(1 + song.playCount) / 5) * 0.4
      : 0
  return clamp01(star + rating + plays)
}

/**
 * Discovery term. At discovery=0 it rewards familiar tracks; at discovery=1
 * it rewards unknown ones. Exactly neutral at 0.5. This is the main lever of
 * the "Familiar <-> Surprise" slider and it never pulls music from outside
 * the user's library.
 */
export function discoveryTerm(profile, discovery) {
  const unknownness = 1 - profile.familiarity
  const signal = unknownness - profile.familiarity // -1 .. 1
  return 0.5 + signal * (discovery - 0.5)
}

/**
 * Temporary penalty for tracks resembling recently skipped content
 * (skip intelligence). Each skip event decays over 30 minutes.
 *
 * @param {{genre?: string, artistKey?: string, energy?: number}} songKeys
 * @param {Array<{genreKey?: string, artistKey?: string, energy?: number, at: number}>} skipEvents
 * @param {number} now
 */
export function skipPenalty(songKeys, skipEvents, now = Date.now()) {
  if (!skipEvents || skipEvents.length === 0) return 0
  const DECAY_MS = 30 * 60 * 1000
  let penalty = 0
  for (const ev of skipEvents) {
    const age = now - ev.at
    if (age < 0 || age > DECAY_MS) continue
    const decay = 1 - age / DECAY_MS
    let match = 0
    if (ev.genreKey && songKeys.genre === ev.genreKey) match += 0.6
    if (ev.artistKey && songKeys.artistKey === ev.artistKey) match += 0.4
    if (
      ev.energy != null &&
      songKeys.energy != null &&
      Math.abs(ev.energy - songKeys.energy) < 0.15
    ) {
      match += 0.3
    }
    penalty += Math.min(1, match) * decay
  }
  // Bounded so skip feedback can dampen, never mute, a genre entirely.
  return Math.min(0.35, penalty * 0.2)
}

/**
 * Score one candidate for a phase.
 *
 * @param {Object} ctx
 * @param {import('./types').Song} ctx.song normalized song
 * @param {Object} ctx.profile derived track profile
 * @param {{energy: number, valence?: number}} ctx.phaseTarget energy/valence target of the current phase
 * @param {import('./types').JourneyPreferences} ctx.prefs
 * @param {import('./personalization').UserProfile} ctx.userProfile
 * @param {number} [ctx.skipBias] precomputed skip penalty
 * @param {number} [ctx.artistRepeats] times this artist already appears in the journey
 * @param {Object} [ctx.prev] profile of the previously chosen track
 */
export function scoreCandidate({
  song,
  profile,
  phaseTarget,
  prefs,
  userProfile,
  skipBias = 0,
  artistRepeats = 0,
  prev = null,
}) {
  const genreKey = String(song.genre || '')
    .trim()
    .toLowerCase()
  const artistKey =
    song.artistId ||
    String(song.artist || '')
      .trim()
      .toLowerCase()

  const components = {
    mood: moodCompatibility(profile, phaseTarget),
    genre: affinityOf(userProfile.genreAffinity, genreKey),
    artist: affinityOf(userProfile.artistAffinity, artistKey, 0.35),
    userPreference: userPreferenceScore(song),
    freshness: freshnessScore(profile),
    discovery: discoveryTerm(profile, prefs.discovery),
  }

  // Decade preference: soft bonus when the era matches the user's habits or
  // the requested decade, mild penalty when a requested decade is missed.
  let eraBonus = 0
  if (prefs.decade) {
    eraBonus = profile.decade === prefs.decade ? 0.15 : -0.25
  } else if (profile.decade) {
    eraBonus =
      affinityOf(userProfile.decadeAffinity, profile.decade, 0.5) * 0.06 - 0.03
  }

  let penalties = skipBias
  if (artistRepeats >= 3) penalties += JOURNEY_PENALTIES.artistSaturation

  const total =
    JOURNEY_WEIGHTS.mood * components.mood +
    JOURNEY_WEIGHTS.genre * components.genre +
    JOURNEY_WEIGHTS.artist * components.artist +
    JOURNEY_WEIGHTS.transition *
      (prev ? transitionCompatibility(prev, profile, prefs) : components.mood) +
    JOURNEY_WEIGHTS.userPreference * components.userPreference +
    JOURNEY_WEIGHTS.freshness * components.freshness +
    JOURNEY_WEIGHTS.discovery * components.discovery +
    eraBonus -
    penalties

  return { total: clamp01(total), components, penalties, genreKey, artistKey }
}

/**
 * Smoothness of the step between two consecutive tracks, used both inside
 * scoring and exposed for tests. 1 = perfect flow, 0 = cliff.
 */
export function transitionCompatibility(prevProfile, nextProfile, prefs) {
  const energyDelta = Math.abs(prevProfile.energy - nextProfile.energy)
  const step = 0.35 + (prefs?.discovery || 0.3) * 0.35
  const energyFit = clamp01(1 - Math.max(0, energyDelta - step * 0.6) / step)

  let tempoFit = 1
  if (prevProfile.bpm && nextProfile.bpm) {
    const ratio =
      Math.max(prevProfile.bpm, nextProfile.bpm) /
      Math.min(prevProfile.bpm, nextProfile.bpm)
    tempoFit = clamp01(1 - Math.max(0, ratio - 1.4) / 1.2)
  }

  return energyFit * 0.7 + tempoFit * 0.3
}

/**
 * Assemble the "Why this track?" explanation from the dominant components.
 * Only computed facts are mentioned.
 */
export function buildReason({ song, components, phaseTarget, prefs, profile }) {
  const parts = []
  const energyDir =
    profile.energy > phaseTarget.energy + 0.12
      ? 'lifts the energy'
      : profile.energy < phaseTarget.energy - 0.12
        ? 'settles the energy'
        : null

  const ranked = Object.entries(components).sort((a, b) => b[1] - a[1])
  for (const [name, value] of ranked) {
    if (parts.length >= 2) break
    if (value < 0.55) continue
    switch (name) {
      case 'mood':
        parts.push(
          energyDir
            ? `it ${energyDir} toward this phase's target`
            : 'it matches the mood of this phase',
        )
        break
      case 'userPreference':
        if (song.starred) parts.push('it is one of your favourites')
        else if (song.rating >= 4) parts.push(`you rated it ${song.rating}/5`)
        else parts.push(`you have played it ${song.playCount} times`)
        break
      case 'genre':
        parts.push(`it fits your ${song.genre || 'usual'} listening`)
        break
      case 'freshness':
        parts.push(
          profile.daysSincePlay == null
            ? 'you have never played it before'
            : 'you have not heard it in a long time',
        )
        break
      case 'discovery':
        if (prefs.discovery >= 0.5)
          parts.push('a deeper cut you may have missed')
        break
      case 'artist':
        parts.push('an artist you listen to often')
        break
      default:
        break
    }
  }

  if (parts.length === 0) {
    parts.push(
      `closest overall match (score ${Math.round(
        (components.mood * 0.5 + components.userPreference * 0.5) * 100,
      )}%)`,
    )
  }
  const text = parts.join('; ')
  return text.charAt(0).toUpperCase() + text.slice(1)
}
