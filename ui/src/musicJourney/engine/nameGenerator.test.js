import { describe, it, expect } from 'vitest'
import { generateJourneyName } from './nameGenerator'
import { normalizePreferences } from './types'

describe('generateJourneyName', () => {
  const date = new Date(2026, 0, 10, 23, 30) // late night

  it('is deterministic for identical preferences', () => {
    const prefs = normalizePreferences({ mood: 'chill', durationMinutes: 45 })
    expect(generateJourneyName(prefs, date)).toBe(
      generateJourneyName(prefs, date),
    )
  })

  it('produces a non-empty title for every mood', () => {
    for (const mood of ['chill', 'focus', 'energy', 'happy', 'melancholic']) {
      const title = generateJourneyName(normalizePreferences({ mood }), date)
      expect(title.length).toBeGreaterThan(3)
    }
  })

  it('reflects a requested era', () => {
    const title = generateJourneyName(
      normalizePreferences({ mood: 'energy', decade: '1990' }),
      date,
    )
    expect(title).toMatch(/'90s/)
  })

  it('can vary via the variant parameter', () => {
    const prefs = normalizePreferences({ mood: 'focus' })
    const names = new Set(
      [0, 1, 2, 3, 4, 5].map((v) => generateJourneyName(prefs, date, v)),
    )
    expect(names.size).toBeGreaterThan(1)
  })
})
