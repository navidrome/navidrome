import { describe, it, expect } from 'vitest'
import {
  buildReason,
  discoveryTerm,
  freshnessScore,
  moodCompatibility,
  scoreCandidate,
  skipPenalty,
  userPreferenceScore,
} from './scoring'
import { buildUserProfile } from './personalization'
import { getTrackProfile } from './trackProfile'
import { normalizePreferences, normalizeSong } from './types'
import { makeSong } from '../testing/fixtures'

const NOW = Date.UTC(2026, 0, 15)

describe('scoring', () => {
  describe('moodCompatibility', () => {
    it('rewards tracks close to the phase energy target', () => {
      const calm = getTrackProfile(makeSong({ genre: 'Ambient' }), NOW)
      const hot = getTrackProfile(makeSong({ genre: 'Metal', bpm: 170 }), NOW)
      expect(moodCompatibility(calm, { energy: 0.2 })).toBeGreaterThan(
        moodCompatibility(hot, { energy: 0.2 }),
      )
    })
  })

  describe('genre affinity', () => {
    it('gives a higher score to the genre the user listens to', () => {
      // Profile built from history: the user clearly prefers Rock.
      const history = [
        makeSong({ genre: 'Rock', playCount: 30 }),
        makeSong({ genre: 'Rock', playCount: 22 }),
        makeSong({ genre: 'Polka', playCount: 1 }),
      ]
      const userProfile = buildUserProfile(history)
      const prefs = normalizePreferences({ mood: 'chill', discovery: 0.3 })
      const phaseTarget = { energy: 0.5, valence: null }

      const rock = makeSong({ genre: 'Rock' })
      const polka = makeSong({ genre: 'Polka' })

      const scoreRock = scoreCandidate({
        song: rock,
        profile: getTrackProfile(rock, NOW),
        phaseTarget,
        prefs,
        userProfile,
      })
      const scorePolka = scoreCandidate({
        song: polka,
        profile: getTrackProfile(polka, NOW),
        phaseTarget,
        prefs,
        userProfile,
      })
      expect(scoreRock.total).toBeGreaterThan(scorePolka.total)
      expect(scoreRock.components.genre).toBeGreaterThan(
        scorePolka.components.genre,
      )
    })
  })

  describe('user preference', () => {
    it('boosts favourite tracks', () => {
      const loved = makeSong({ starred: true, rating: 5, playCount: 8 })
      const plain = makeSong({ starred: false, rating: 0, playCount: 0 })
      expect(userPreferenceScore(loved)).toBeGreaterThan(
        userPreferenceScore(plain),
      )
    })
  })

  describe('freshness', () => {
    it('penalizes recently played tracks', () => {
      const recent = getTrackProfile(
        normalizeSong(
          makeSong({ playDate: new Date(NOW - 3600e3).toISOString() }),
        ),
        NOW,
      )
      const never = getTrackProfile(
        normalizeSong(makeSong({ playDate: null })),
        NOW,
      )
      const old = getTrackProfile(
        normalizeSong(
          makeSong({ playDate: new Date(NOW - 400 * 86400e3).toISOString() }),
        ),
        NOW,
      )
      expect(freshnessScore(recent)).toBeLessThan(freshnessScore(never))
      expect(freshnessScore(old)).toBeGreaterThan(freshnessScore(recent))
    })
  })

  describe('discovery', () => {
    const familiar = { familiarity: 0.9 }
    const unknown = { familiarity: 0.05 }

    it('at low discovery familiar tracks win', () => {
      expect(discoveryTerm(familiar, 0)).toBeGreaterThan(
        discoveryTerm(unknown, 0),
      )
    })

    it('at high discovery unknown tracks win', () => {
      expect(discoveryTerm(unknown, 1)).toBeGreaterThan(
        discoveryTerm(familiar, 1),
      )
    })

    it('is neutral at 0.5', () => {
      expect(discoveryTerm(familiar, 0.5)).toBeCloseTo(0.5)
      expect(discoveryTerm(unknown, 0.5)).toBeCloseTo(0.5)
    })
  })

  describe('skipPenalty', () => {
    it('penalizes tracks matching recently skipped content and decays over time', () => {
      const events = [
        { genreKey: 'metal', artistKey: 'a1', energy: 0.8, at: NOW },
      ]
      const match = skipPenalty(
        { genre: 'metal', artistKey: 'a1', energy: 0.82 },
        events,
        NOW,
      )
      const noMatch = skipPenalty(
        { genre: 'jazz', artistKey: 'a2', energy: 0.3 },
        events,
        NOW,
      )
      const tooOld = skipPenalty(
        { genre: 'metal', artistKey: 'a1', energy: 0.82 },
        [{ ...events[0], at: NOW - 40 * 60 * 1000 }],
        NOW,
      )
      expect(match).toBeGreaterThan(noMatch)
      expect(match).toBeGreaterThan(0)
      expect(tooOld).toBe(0)
      // Bounded so skip feedback cannot fully mute a genre.
      expect(match).toBeLessThanOrEqual(0.35)
    })
  })

  describe('buildReason', () => {
    it('mentions real data only (starred track)', () => {
      const song = makeSong({ starred: true, genre: 'Jazz' })
      const reason = buildReason({
        song,
        components: {
          mood: 0.4,
          genre: 0.4,
          artist: 0.4,
          userPreference: 0.9,
          freshness: 0.3,
          discovery: 0.4,
        },
        phaseTarget: { energy: 0.5, valence: null },
        prefs: normalizePreferences({}),
        profile: getTrackProfile(song, NOW),
      })
      expect(reason).toMatch(/favourite/i)
    })
  })
})
