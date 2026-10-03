/**
 * Personalization model built exclusively from the user's own Navidrome
 * data (play counts, stars, ratings, last-played dates). Nothing leaves the
 * browser.
 *
 * The profile is derived from the candidate pool itself, which already
 * contains enriched slices of the library (top played, recently played,
 * favorites, random), so no extra API round-trips are needed here.
 */

/**
 * @typedef {Object} UserProfile
 * @property {Map<string, number>} genreAffinity normalized 0..1 per genre
 * @property {Map<string, number>} artistAffinity normalized 0..1 per artist id/name
 * @property {Map<string, number>} decadeAffinity normalized 0..1 per decade
 * @property {number} avgEnergy mean energy of the user's most played tracks
 * @property {boolean} hasHistory false when no listening data is available
 */

const keyOfGenre = (g) =>
  String(g || '')
    .trim()
    .toLowerCase()
const keyOfArtist = (song) =>
  song.artistId ||
  String(song.artist || '')
    .trim()
    .toLowerCase()

/**
 * Build the user profile from a pool of normalized songs.
 *
 * Affinity is play-count weighted (a starred track counts as a few plays so
 * love is reflected even without scrobble history). Genres/artists are
 * normalized against the strongest one, which keeps the scoring stable for
 * both tiny and huge libraries.
 *
 * @param {import('./types').Song[]} songs
 * @param {(song: import('./types').Song) => {energy: number}} [profileOf]
 * @returns {UserProfile}
 */
export function buildUserProfile(songs, profileOf) {
  const genreWeight = new Map()
  const artistWeight = new Map()
  const decadeWeight = new Map()
  let energySum = 0
  let energyCount = 0

  for (const song of songs) {
    // History signals: real plays count most; stars/ratings imply affinity.
    const weight = song.playCount + (song.starred ? 3 : 0) + song.rating * 0.5
    if (weight <= 0) continue

    const genre = keyOfGenre(song.genre)
    if (genre) {
      genreWeight.set(genre, (genreWeight.get(genre) || 0) + weight)
    }
    const artist = keyOfArtist(song)
    if (artist) {
      artistWeight.set(artist, (artistWeight.get(artist) || 0) + weight)
    }
    if (song.year) {
      const decade = String(Math.floor(song.year / 10) * 10)
      decadeWeight.set(decade, (decadeWeight.get(decade) || 0) + weight)
    }
    if (profileOf) {
      const p = profileOf(song)
      if (p) {
        energySum += p.energy
        energyCount++
      }
    }
  }

  const normalize = (map) => {
    let max = 0
    for (const v of map.values()) max = Math.max(max, v)
    if (max === 0) return new Map()
    const out = new Map()
    for (const [k, v] of map) out.set(k, v / max)
    return out
  }

  return {
    genreAffinity: normalize(genreWeight),
    artistAffinity: normalize(artistWeight),
    decadeAffinity: normalize(decadeWeight),
    avgEnergy: energyCount ? energySum / energyCount : 0.5,
    hasHistory: genreWeight.size > 0 || artistWeight.size > 0,
  }
}

/** Affinity lookup with a neutral fallback for unknown keys. */
export const affinityOf = (map, key, fallback = 0.4) =>
  map && key ? (map.has(key) ? map.get(key) : fallback) : fallback
