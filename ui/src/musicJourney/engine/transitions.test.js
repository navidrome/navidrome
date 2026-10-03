import { describe, it, expect } from 'vitest'
import { transitionScore } from './transitions'
import { getTrackProfile } from './trackProfile'
import { normalizePreferences } from './types'
import { makeSong } from '../testing/fixtures'

const pair = (a, b) => ({
  song: a,
  profile: getTrackProfile(a),
})

describe('transitionScore', () => {
  const lowDiscovery = normalizePreferences({ discovery: 0 })
  const highDiscovery = normalizePreferences({ discovery: 1 })

  it('scores compatible consecutive tracks highly', () => {
    const a = makeSong({ genre: 'House', bpm: 124 })
    const b = makeSong({ genre: 'Techno', bpm: 128 })
    const { score } = transitionScore(pair(a), pair(b), lowDiscovery)
    expect(score).toBeGreaterThan(0.8)
  })

  it('penalizes extreme energy cliffs (ambient -> aggressive metal)', () => {
    const ambient = makeSong({ genre: 'Ambient' })
    const metal = makeSong({ genre: 'Death Metal', bpm: 180 })
    const smooth = makeSong({ genre: 'Rock', bpm: 120 })

    const cliff = transitionScore(pair(ambient), pair(metal), lowDiscovery)
    const fine = transitionScore(pair(ambient), pair(smooth), lowDiscovery)

    expect(cliff.penalties).toContain('energy-cliff')
    expect(cliff.score).toBeLessThan(fine.score)
  })

  it('user preferences soften transition penalties', () => {
    const ambient = makeSong({ genre: 'Ambient' })
    const metal = makeSong({ genre: 'Death Metal', bpm: 180 })
    const strict = transitionScore(pair(ambient), pair(metal), lowDiscovery)
    const loose = transitionScore(pair(ambient), pair(metal), highDiscovery)
    expect(loose.score).toBeGreaterThan(strict.score)
  })

  it('penalizes tempo jumps', () => {
    const slow = makeSong({ genre: 'Jazz', bpm: 70 })
    const fast = makeSong({ genre: 'Drum and bass', bpm: 174 })
    const { penalties } = transitionScore(pair(slow), pair(fast), lowDiscovery)
    expect(penalties).toContain('tempo-jump')
  })

  it('rewards album continuity but penalizes artist monotony', () => {
    const base = makeSong({ genre: 'Rock', bpm: 120 })
    const sameAlbum = makeSong({
      genre: 'Rock',
      bpm: 121,
      albumId: base.albumId,
      artistId: 'someone-else',
    })
    const sameArtist = makeSong({
      genre: 'Rock',
      bpm: 121,
      albumId: 'other-album',
      artistId: base.artistId,
    })
    const albumScore = transitionScore(
      pair(base),
      pair(sameAlbum),
      lowDiscovery,
    )
    const artistScore = transitionScore(
      pair(base),
      pair(sameArtist),
      lowDiscovery,
    )
    expect(albumScore.score).toBeGreaterThan(artistScore.score)
    expect(artistScore.penalties).toContain('same-artist')
  })

  it('gives a neutral 1 for the first track (no previous)', () => {
    const t = makeSong({})
    const { score } = transitionScore(null, pair(t), lowDiscovery)
    expect(score).toBe(1)
  })
})
