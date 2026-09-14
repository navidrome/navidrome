import { describe, it, expect } from 'vitest'
import { shuffle, shuffleTracks, PLAYER_PLAY_TRACKS } from './player'

describe('shuffle', () => {
  it('keeps every track and prefixes keys so insertion order is preserved', () => {
    const data = {
      a: { id: 'a', artist: 'A', album: 'X' },
      b: { id: 'b', artist: 'B', album: 'Y' },
      c: { id: 'c', artist: 'C', album: 'Z' },
    }
    const out = shuffle(data)
    const keys = Object.keys(out)
    expect(keys.every((k) => k.startsWith('_'))).toBe(true)
    expect(keys.map((k) => k.slice(1)).sort()).toEqual(['a', 'b', 'c'])
    expect(
      Object.values(out)
        .map((s) => s.id)
        .sort(),
    ).toEqual(['a', 'b', 'c'])
  })
})

describe('shuffleTracks', () => {
  it('plays from the first track of the shuffled object', () => {
    const data = {
      a: { id: 'a', artist: 'A', album: 'X' },
      b: { id: 'b', artist: 'B', album: 'Y' },
      c: { id: 'c', artist: 'C', album: 'Z' },
    }
    const action = shuffleTracks(data)
    const firstKey = Object.keys(action.data)[0]
    expect(action.type).toBe(PLAYER_PLAY_TRACKS)
    expect(action.id).toBe(firstKey)
    expect(firstKey.startsWith('_')).toBe(true)
  })

  it('drops missing tracks before shuffling', () => {
    const data = {
      a: { id: 'a', artist: 'A', album: 'X' },
      b: { id: 'b', artist: 'B', album: 'Y', missing: true },
    }
    const action = shuffleTracks(data)
    expect(Object.values(action.data).map((s) => s.id)).toEqual(['a'])
  })
})
