import { describe, it, expect } from 'vitest'
import {
  allocatePhaseDurations,
  baseEnergyOf,
  buildNarrative,
  phaseIndexAt,
} from './phases'
import { normalizePreferences } from './types'

describe('phases', () => {
  it('uses mood-specific archetypes', () => {
    const chill = buildNarrative(normalizePreferences({ mood: 'chill' }))
    const energy = buildNarrative(normalizePreferences({ mood: 'energy' }))
    expect(chill.map((p) => p.name)).toEqual([
      'Arrival',
      'Flow',
      'Deep Focus',
      'Drift',
    ])
    expect(energy.map((p) => p.name)).toEqual([
      'Warm Up',
      'Build',
      'Peak',
      'Afterglow',
    ])
  })

  it('adapts the number of phases to the duration', () => {
    const short = buildNarrative(
      normalizePreferences({ mood: 'chill', durationMinutes: 10 }),
    )
    const medium = buildNarrative(
      normalizePreferences({ mood: 'chill', durationMinutes: 20 }),
    )
    const long = buildNarrative(
      normalizePreferences({ mood: 'chill', durationMinutes: 60 }),
    )
    expect(short.length).toBe(2)
    expect(medium.length).toBe(3)
    expect(long.length).toBe(4)
    // shortened arcs keep the journey's beginning and end
    expect(short[0].id).toBe('arrival')
    expect(short[short.length - 1].id).toBe('drift')
  })

  it('weights always sum to 1', () => {
    for (const mood of ['chill', 'focus', 'energy', 'happy', 'melancholic']) {
      const narrative = buildNarrative(normalizePreferences({ mood }))
      const sum = narrative.reduce((s, p) => s + p.weight, 0)
      expect(sum).toBeCloseTo(1)
    }
  })

  it('intensity moves the phase energy targets', () => {
    const calmPrefs = normalizePreferences({ mood: 'energy', intensity: 0.2 })
    const hotPrefs = normalizePreferences({ mood: 'energy', intensity: 0.9 })
    expect(baseEnergyOf(hotPrefs)).toBeGreaterThan(baseEnergyOf(calmPrefs))
    const calm = buildNarrative(calmPrefs)
    const hot = buildNarrative(hotPrefs)
    const peakCalm = calm.find((p) => p.id === 'peak')
    const peakHot = hot.find((p) => p.id === 'peak')
    expect(peakHot.targetEnergy).toBeGreaterThan(peakCalm.targetEnergy)
  })

  it('allocates duration proportionally and tracks elapsed phase', () => {
    const narrative = buildNarrative(
      normalizePreferences({ mood: 'energy', durationMinutes: 40 }),
    )
    const total = 2400
    const alloc = allocatePhaseDurations(narrative, total)
    const sum = alloc.reduce((s, a) => s + a.seconds, 0)
    expect(sum).toBeCloseTo(total)
    alloc.forEach((a) => expect(a.seconds).toBeGreaterThan(0))

    expect(phaseIndexAt(narrative, 0, total)).toBe(0)
    expect(phaseIndexAt(narrative, total - 1, total)).toBe(narrative.length - 1)
    const mid = narrative[0].weight * total + 1
    expect(phaseIndexAt(narrative, mid, total)).toBe(1)
  })
})
