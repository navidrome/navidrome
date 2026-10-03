import { describe, it, expect } from 'vitest'
import { computeJourneyPosition, dominantRemainingGenre } from './progress'

const journey = {
  preferences: {
    mood: 'energy',
    durationMinutes: 30,
    intensity: 0.5,
    discovery: 0.3,
  },
  estimatedDuration: 900,
  tracks: [
    {
      track: { id: 't1', duration: 200 },
      stage: { id: 'warm-up', name: 'Warm Up' },
    },
    {
      track: { id: 't2', duration: 250 },
      stage: { id: 'build', name: 'Build' },
    },
    { track: { id: 't3', duration: 250 }, stage: { id: 'peak', name: 'Peak' } },
    {
      track: { id: 't4', duration: 200 },
      stage: { id: 'afterglow', name: 'Afterglow' },
    },
  ],
}

describe('computeJourneyPosition', () => {
  it('locates the current track and computes elapsed time', () => {
    const queue = [
      { uuid: 'u1', trackId: 't1' },
      { uuid: 'u2', trackId: 't2' },
      { uuid: 'u3', trackId: 't3' },
    ]
    const current = { uuid: 'u2', trackId: 't2', currentTime: 50 }
    const pos = computeJourneyPosition(journey, queue, current)
    expect(pos.currentIdx).toBe(1)
    expect(pos.elapsed).toBe(250) // 200 finished + 50 in progress
    expect(pos.next.track.id).toBe('t3')
    expect(pos.journeyInQueue).toBe(true)
  })

  it('handles an idle player', () => {
    const pos = computeJourneyPosition(journey, [], {})
    expect(pos.currentIdx).toBe(-1)
    expect(pos.elapsed).toBe(0)
    expect(pos.journeyInQueue).toBe(false)
  })

  it('reports the finished state on the last track', () => {
    const current = { uuid: 'u4', trackId: 't4', currentTime: 0, ended: true }
    const pos = computeJourneyPosition(
      journey,
      [{ uuid: 'u4', trackId: 't4' }],
      current,
    )
    expect(pos.finished).toBe(true)
  })

  it('phase index moves with elapsed time', () => {
    const early = computeJourneyPosition(journey, [], {
      trackId: 't1',
      currentTime: 10,
    })
    const late = computeJourneyPosition(journey, [], {
      trackId: 't4',
      currentTime: 100,
    })
    expect(early.phaseIdx).toBe(0)
    expect(late.phaseIdx).toBe(journey.tracks.length - 1)
  })
})

describe('dominantRemainingGenre', () => {
  const j = {
    tracks: [
      { track: { genre: 'Rock' } },
      { track: { genre: 'Jazz' } },
      { track: { genre: 'Jazz' } },
      { track: { genre: 'Pop' } },
    ],
  }

  it('finds the most frequent genre ahead of the cursor', () => {
    expect(dominantRemainingGenre(j, 1)).toBe('jazz')
    expect(dominantRemainingGenre(j, 3)).toBe('pop')
    expect(dominantRemainingGenre(j, 4)).toBe(null)
  })
})
