/**
 * Shared fixtures for Music Journey tests. Songs mimic the shape returned
 * by the Navidrome /api/song endpoint.
 */

let seq = 0

export const makeSong = (overrides = {}) => {
  seq += 1
  return {
    id: `song-${seq}`,
    title: `Song ${seq}`,
    artist: `Artist ${seq % 7}`,
    artistId: `artist-${seq % 7}`,
    album: `Album ${seq % 5}`,
    albumId: `album-${seq % 5}`,
    genre: 'Rock',
    year: 2005,
    duration: 180,
    bpm: null,
    playCount: 1,
    playDate: null,
    rating: 0,
    starred: false,
    starredAt: null,
    createdAt: '2020-01-01T00:00:00Z',
    updatedAt: '2020-01-01T00:00:00Z',
    missing: false,
    ...overrides,
  }
}

/** A deterministic library spanning genres/eras for engine tests. */
export const makeLibrary = (size = 60) => {
  const genres = ['Rock', 'Electronic', 'Jazz', 'Ambient', 'Pop']
  const songs = []
  for (let i = 0; i < size; i++) {
    songs.push(
      makeSong({
        genre: genres[i % genres.length],
        year: 1970 + (i % 6) * 10,
        bpm: 60 + (i % 9) * 15,
        duration: 120 + (i % 5) * 30,
        playCount: i % 3,
        starred: i % 11 === 0,
        rating: i % 4,
      }),
    )
  }
  return songs
}

/** Seeded rng helper: deterministic pick sequence for tests. */
export const seededRng = (seed = 42) => {
  let a = seed >>> 0
  return () => {
    a |= 0
    a = (a + 0x6d2b79f5) | 0
    let t = Math.imul(a ^ (a >>> 15), 1 | a)
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }
}
