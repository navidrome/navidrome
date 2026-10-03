import { describe, it, expect } from 'vitest'
import {
  buildJourney,
  JourneyError,
  regenerateRemaining,
} from './journeyEngine'
import { buildNarrative } from './phases'
import { normalizePreferences } from './types'
import { makeLibrary, makeSong, seededRng } from '../testing/fixtures'

const NOW = Date.UTC(2026, 2, 10)
const DEFAULT_PREFS = {
  mood: 'chill',
  durationMinutes: 30,
  intensity: 0.5,
  discovery: 0.3,
}

describe('buildJourney', () => {
  it('creates a journey of roughly the requested duration', () => {
    const journey = buildJourney({
      songs: makeLibrary(80),
      prefs: DEFAULT_PREFS,
      now: NOW,
      rng: seededRng(),
    })
    const target = DEFAULT_PREFS.durationMinutes * 60
    expect(journey.estimatedDuration).toBeGreaterThan(target * 0.6)
    // Overshoot bounded by the longest track in the pool.
    expect(journey.estimatedDuration).toBeLessThanOrEqual(target + 270)
    expect(journey.tracks.reduce((s, t) => s + t.track.duration, 0)).toBe(
      journey.estimatedDuration,
    )
  })

  it('never contains duplicate tracks', () => {
    const journey = buildJourney({
      songs: makeLibrary(100),
      prefs: DEFAULT_PREFS,
      now: NOW,
      rng: seededRng(7),
    })
    const ids = journey.tracks.map((t) => t.track.id)
    expect(new Set(ids).size).toBe(ids.length)
  })

  it('excludes missing tracks', () => {
    const songs = makeLibrary(30).map((s, i) =>
      i < 10 ? { ...s, missing: true } : s,
    )
    const journey = buildJourney({
      songs,
      prefs: DEFAULT_PREFS,
      now: NOW,
      rng: seededRng(),
    })
    journey.tracks.forEach((t) => expect(t.track.missing).toBe(false))
  })

  it('assigns every track to a phase from the narrative', () => {
    const prefs = normalizePreferences(DEFAULT_PREFS)
    const journey = buildJourney({
      songs: makeLibrary(80),
      prefs,
      now: NOW,
      rng: seededRng(3),
    })
    const narrativeIds = new Set(journey.narrative.map((p) => p.id))
    journey.tracks.forEach((t) =>
      expect(narrativeIds.has(t.stage.id)).toBe(true),
    )
    // The full arc is present when the pool is large enough.
    journey.narrative.forEach((phase) => {
      expect(journey.tracks.some((t) => t.stage.id === phase.id)).toBe(true)
    })
    expect(journey.narrative.map((p) => p.name)).toEqual(
      buildNarrative(prefs).map((p) => p.name),
    )
  })

  it('respects the decade filter', () => {
    const songs = [
      ...Array.from({ length: 12 }, (_, i) =>
        makeSong({ year: 1995, title: `nineties ${i}` }),
      ),
      ...Array.from({ length: 12 }, (_, i) =>
        makeSong({ year: 2018, title: `twenty-tens ${i}` }),
      ),
    ]
    const journey = buildJourney({
      songs,
      prefs: { ...DEFAULT_PREFS, decade: '1990' },
      now: NOW,
      rng: seededRng(),
    })
    journey.tracks.forEach((t) => expect(t.track.year).toBeLessThan(2000))
  })

  it('respects excludeGenres (used by live "Less <genre>" control)', () => {
    const journey = buildJourney({
      songs: makeLibrary(80),
      prefs: { ...DEFAULT_PREFS, excludeGenres: ['electronic'] },
      now: NOW,
      rng: seededRng(),
    })
    journey.tracks.forEach((t) =>
      expect(String(t.track.genre).toLowerCase()).not.toBe('electronic'),
    )
  })

  it('gives every track a data-driven reason', () => {
    const journey = buildJourney({
      songs: makeLibrary(60),
      prefs: DEFAULT_PREFS,
      now: NOW,
      rng: seededRng(),
    })
    journey.tracks.forEach((t) => {
      expect(typeof t.reason).toBe('string')
      expect(t.reason.length).toBeGreaterThan(0)
      expect(t.score).toBeGreaterThan(0)
    })
  })

  it('marks the journey truncated when the pool cannot fill the duration', () => {
    const songs = Array.from({ length: 8 }, () => makeSong({ duration: 60 }))
    const journey = buildJourney({
      songs,
      prefs: { ...DEFAULT_PREFS, durationMinutes: 60 },
      now: NOW,
      rng: seededRng(),
    })
    expect(journey.truncated).toBe(true)
    expect(journey.tracks.length).toBe(8)
  })

  it('handles a very small library gracefully', () => {
    expect(() =>
      buildJourney({
        songs: [makeSong(), makeSong(), makeSong()],
        prefs: DEFAULT_PREFS,
        now: NOW,
        rng: seededRng(),
      }),
    ).toThrow(JourneyError)

    // Exactly the minimum: still works (long enough window to use them all).
    const songs = Array.from({ length: 5 }, () => makeSong())
    const journey = buildJourney({
      songs,
      prefs: { ...DEFAULT_PREFS, durationMinutes: 20 },
      now: NOW,
      rng: seededRng(),
    })
    expect(journey.tracks.length).toBe(5)
    expect(journey.truncated).toBe(true)
  })

  it('throws EMPTY_LIBRARY for an empty pool', () => {
    try {
      buildJourney({ songs: [], prefs: DEFAULT_PREFS, now: NOW })
      expect.unreachable('should have thrown')
    } catch (e) {
      expect(e).toBeInstanceOf(JourneyError)
      expect(e.code).toBe('EMPTY_LIBRARY')
    }
  })

  it('throws NOT_ENOUGH_TRACKS with details when filters remove everything', () => {
    const songs = Array.from({ length: 10 }, () => makeSong({ year: 1985 }))
    try {
      buildJourney({
        songs,
        prefs: { ...DEFAULT_PREFS, decade: '2020' },
        now: NOW,
      })
      expect.unreachable('should have thrown')
    } catch (e) {
      expect(e.code).toBe('NOT_ENOUGH_TRACKS')
      expect(e.details.available).toBe(0)
    }
  })
})

describe('regenerateRemaining', () => {
  const build = () =>
    buildJourney({
      songs: makeLibrary(120),
      prefs: { ...DEFAULT_PREFS, durationMinutes: 45 },
      now: NOW,
      rng: seededRng(11),
    })

  it('keeps the played part untouched, including the current track', () => {
    const journey = build()
    const playedCount = 4 // three finished + one currently playing
    const rebuilt = regenerateRemaining({
      journey,
      playedCount,
      prefs: { ...journey.preferences, intensity: 0.9 },
      songs: makeLibrary(120),
      now: NOW + 15 * 60 * 1000,
      rng: seededRng(99),
    })

    expect(rebuilt.keptCount).toBe(playedCount)
    const keptIds = journey.tracks.slice(0, playedCount).map((t) => t.track.id)
    expect(rebuilt.tracks.slice(0, playedCount).map((t) => t.track.id)).toEqual(
      keptIds,
    )
    // The currently playing track object survives the rebuild verbatim.
    expect(rebuilt.tracks[playedCount - 1]).toBe(
      journey.tracks[playedCount - 1],
    )
  })

  it('only changes the future part', () => {
    const journey = build()
    const playedCount = 3
    const rebuilt = regenerateRemaining({
      journey,
      playedCount,
      prefs: { ...journey.preferences, discovery: 0.9 },
      songs: makeLibrary(120),
      now: NOW,
      rng: seededRng(5),
    })

    const keptIds = new Set(
      journey.tracks.slice(0, playedCount).map((t) => t.track.id),
    )
    rebuilt.tracks.slice(playedCount).forEach((t) => {
      expect(keptIds.has(t.track.id)).toBe(false)
    })
    // Journey identity is preserved.
    expect(rebuilt.id).toBe(journey.id)
    expect(rebuilt.version).toBe(journey.version + 1)
    expect(rebuilt.tracks.length).toBeGreaterThanOrEqual(playedCount)
  })

  it('still works when only a handful of fresh tracks remain', () => {
    const songs = Array.from({ length: 10 }, () => makeSong({ duration: 120 }))
    const journey = buildJourney({
      songs,
      prefs: { ...DEFAULT_PREFS, durationMinutes: 15 },
      now: NOW,
      rng: seededRng(2),
    })
    const rebuilt = regenerateRemaining({
      journey,
      playedCount: journey.tracks.length - 1,
      prefs: journey.preferences,
      songs,
      now: NOW,
      rng: seededRng(3),
    })
    expect(rebuilt.keptCount).toBe(journey.tracks.length - 1)
    expect(rebuilt.tracks.length).toBeGreaterThanOrEqual(rebuilt.keptCount)
    expect(rebuilt.truncated).toBe(true)
  })

  it('propagates EMPTY_LIBRARY when no fresh material exists', () => {
    const songs = Array.from({ length: 6 }, () => makeSong())
    const journey = buildJourney({
      songs,
      prefs: { ...DEFAULT_PREFS, durationMinutes: 15 },
      now: NOW,
      rng: seededRng(2),
    })
    expect(() =>
      regenerateRemaining({
        journey,
        playedCount: journey.tracks.length,
        prefs: journey.preferences,
        songs: [], // nothing left anywhere
        now: NOW,
        rng: seededRng(2),
      }),
    ).toThrow(JourneyError)
  })
})
