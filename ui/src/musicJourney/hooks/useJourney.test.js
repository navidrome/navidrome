import { describe, it, expect } from 'vitest'
import { computeKeptCount } from './useJourney'

const journeyWith = (ids) => ({
  tracks: ids.map((id) => ({ track: { id } })),
})

describe('computeKeptCount', () => {
  const queue = [
    { uuid: 'u1', trackId: 't1' },
    { uuid: 'u2', trackId: 't2' },
    { uuid: 'u3', trackId: 't3' },
  ]

  it('returns 0 without a current track', () => {
    expect(computeKeptCount(queue, null, journeyWith(['t1', 't2', 't3']))).toBe(
      0,
    )
  })

  it('counts journey tracks up to and including the current one', () => {
    const journey = journeyWith(['t1', 't2', 't3'])
    expect(computeKeptCount(queue, 'u2', journey)).toBe(2)
    expect(computeKeptCount(queue, 'u3', journey)).toBe(3)
  })

  it('ignores non-journey items mixed into the prefix', () => {
    const mixed = [
      { uuid: 'u1', trackId: 't1' },
      { uuid: 'ux', trackId: 'foreign' },
      { uuid: 'u2', trackId: 't2' },
    ]
    const journey = journeyWith(['t1', 't2'])
    expect(computeKeptCount(mixed, 'u2', journey)).toBe(2)
  })

  it('never exceeds the journey length', () => {
    const journey = journeyWith(['t1'])
    expect(computeKeptCount(queue, 'u3', journey)).toBe(1)
  })
})
