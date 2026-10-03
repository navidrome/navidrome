/**
 * Track profile derivation.
 *
 * Navidrome does not expose "energy" or "valence" features, so we derive
 * proxies from the metadata it *does* provide: BPM tags, genre names, year
 * and listening history. Everything below is a local heuristic — no external
 * AI service is involved, and every input comes from the user's own library.
 *
 * The genre priors table is intentionally keyword-based (a genre like
 * "Melodic Death Metal" still matches "metal") and every lookup falls back
 * to neutral values, so unknown genres never break generation.
 */

/**
 * Genre keyword priors: [energy 0..1, valence 0..1, typical BPM].
 * Order matters: earlier keywords win when several match.
 */
export const GENRE_PRIORS = [
  [['ambient', 'drone'], 0.12, 0.45, 70],
  [['classical', 'orchestral', 'opera', 'baroque'], 0.22, 0.5, 80],
  [['lofi', 'lo-fi'], 0.2, 0.55, 80],
  [
    ['chillout', 'chill', 'downtempo', 'lounge', 'easy listening'],
    0.22,
    0.6,
    85,
  ],
  [['jazz', 'bossa nova'], 0.32, 0.6, 110],
  [['blues'], 0.38, 0.42, 90],
  [
    ['folk', 'acoustic', 'singer-songwriter', 'country', 'bluegrass'],
    0.36,
    0.62,
    100,
  ],
  [['soul', 'rnb', 'r&b', 'gospel'], 0.48, 0.7, 95],
  [['reggae', 'ska', 'dub'], 0.5, 0.78, 100],
  [['soundtrack', 'score', 'cinematic'], 0.3, 0.5, 90],
  [['indie'], 0.46, 0.58, 115],
  [['pop', 'disco'], 0.62, 0.78, 118],
  [['funk'], 0.64, 0.8, 110],
  [['synthwave', 'retrowave'], 0.6, 0.6, 112],
  [['rock', 'alternative', 'grunge'], 0.62, 0.52, 130],
  [['latin', 'salsa', 'cumbia', 'flamenco'], 0.66, 0.8, 115],
  [['hip hop', 'hip-hop', 'hiphop', 'rap', 'trap'], 0.66, 0.58, 95],
  [['punk', 'hardcore'], 0.82, 0.5, 165],
  [['metal'], 0.85, 0.35, 150],
  [['electronic', 'electronica', 'idm'], 0.66, 0.58, 125],
  [['house', 'dance', 'edm', 'big room'], 0.75, 0.68, 126],
  [['techno'], 0.78, 0.5, 132],
  [['trance', 'progressive'], 0.72, 0.58, 136],
  [['dubstep'], 0.8, 0.45, 140],
  [
    ['drum and bass', 'drum&bass', 'dnb', 'jungle', 'breakbeat'],
    0.85,
    0.55,
    172,
  ],
]

const clamp01 = (v) => Math.min(1, Math.max(0, v))

/** Look up genre priors. Returns null when nothing matches. */
export function matchGenrePrior(genre) {
  if (!genre) return null
  const g = String(genre).toLowerCase()
  for (const [keywords, energy, valence, bpm] of GENRE_PRIORS) {
    if (keywords.some((k) => g.includes(k))) {
      return { energy, valence, bpm }
    }
  }
  return null
}

/** Map a BPM value to a 0..1 energy estimate (40..190 BPM span). */
export function bpmToEnergy(bpm) {
  return clamp01((bpm - 40) / 150)
}

/**
 * Compute the derived profile for a normalized song.
 *
 * Energy blends BPM (when tagged) with genre priors; when neither is
 * available we stay exactly neutral (0.5) so unknown tracks are neither
 * rewarded nor punished by mood scoring.
 *
 * @param {import('./types').Song} song
 * @param {number} [now] current time in ms, injectable for tests
 */
export function getTrackProfile(song, now = Date.now()) {
  const prior = matchGenrePrior(song.genre)

  let energy
  let tempoSource = 'none'
  if (song.bpm) {
    // BPM is direct evidence: it dominates, the genre prior only nudges.
    energy = clamp01(
      bpmToEnergy(song.bpm) * 0.8 + (prior ? prior.energy * 0.2 : 0.1),
    )
    tempoSource = 'bpm'
  } else if (prior) {
    energy = prior.energy
    tempoSource = 'genre'
  } else {
    energy = 0.5
  }

  const valence = prior ? prior.valence : 0.5

  // Familiarity: how well the user knows this track, from plays/love/recency.
  const playPart = Math.min(1, song.playCount / 10)
  const starPart = song.starred ? 0.4 : 0
  const ratingPart = song.rating > 0 ? Math.min(1, song.rating / 5) * 0.4 : 0
  const familiarity = clamp01(playPart * 0.5 + starPart + ratingPart)

  // playDate is expected as epoch ms (normalizeSong output); anything else
  // is treated as "never played" rather than producing NaN comparisons.
  let daysSincePlay = null
  if (typeof song.playDate === 'number' && Number.isFinite(song.playDate)) {
    daysSincePlay = Math.max(0, (now - song.playDate) / 86400000)
  }

  return {
    energy,
    valence,
    tempoSource,
    bpm: song.bpm || (prior ? prior.bpm : null),
    familiarity,
    daysSincePlay,
    decade: song.year ? String(Math.floor(song.year / 10) * 10) : null,
  }
}
