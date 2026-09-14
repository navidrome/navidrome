import { describe, it, expect } from 'vitest'
import { balancedShuffle, songKeys } from './shuffle'

const ids = (tracks) => tracks.map((t) => t.id)

const adjacentSame = (tracks, field) => {
  let n = 0
  for (let i = 1; i < tracks.length; i++) {
    if (field(tracks[i]) && field(tracks[i]) === field(tracks[i - 1])) {
      n++
    }
  }
  return n
}

const minAdjacent = (n, maxFreq) => {
  if (n === 0) {
    return 0
  }
  return Math.max(0, 2 * maxFreq - n - 1)
}

describe('songKeys', () => {
  it('prefers artist, then singer, then albumArtist', () => {
    expect(
      songKeys({ artist: 'A', singer: 'S', albumArtist: 'AA', album: 'X' }),
    ).toEqual({
      artist: 'A',
      album: 'X',
    })
    expect(songKeys({ singer: 'S', albumArtist: 'AA' })).toEqual({
      artist: 'S',
      album: '',
    })
    expect(songKeys({ albumArtist: 'AA', album: 'Y' })).toEqual({
      artist: 'AA',
      album: 'Y',
    })
    expect(songKeys({})).toEqual({ artist: '', album: '' })
  })
})

describe('balancedShuffle', () => {
  it('leaves empty and single-item lists unchanged', () => {
    expect(balancedShuffle([])).toEqual([])
    expect(balancedShuffle([{ id: '1', artist: 'A', album: 'X' }])).toEqual([
      { id: '1', artist: 'A', album: 'X' },
    ])
  })

  it('permutes two items without dropping or duplicating', () => {
    const tracks = [
      { id: '1', artist: 'A', album: 'X' },
      { id: '2', artist: 'B', album: 'Y' },
    ]
    expect(ids(balancedShuffle(tracks, songKeys))).toEqual(
      expect.arrayContaining(['1', '2']),
    )
    expect(balancedShuffle(tracks, songKeys)).toHaveLength(2)
  })

  it('does not mutate the input array', () => {
    const tracks = [
      { id: '1', artist: 'A', album: 'X' },
      { id: '2', artist: 'B', album: 'Y' },
      { id: '3', artist: 'C', album: 'Z' },
    ]
    const snapshot = tracks.slice()
    balancedShuffle(tracks, songKeys)
    expect(tracks).toEqual(snapshot)
  })

  it('is a full permutation of a larger list', () => {
    const tracks = Array.from({ length: 20 }, (_, i) => ({
      id: String(i),
      artist: `Artist${i % 5}`,
      album: `Album${i % 7}`,
    }))
    const shuffled = balancedShuffle(tracks, songKeys)
    expect(ids(shuffled).sort()).toEqual(ids(tracks).sort())
  })

  it('still shuffles a single-artist playlist', () => {
    const tracks = [
      { id: '1', artist: 'A', album: 'X' },
      { id: '2', artist: 'A', album: 'X' },
      { id: '3', artist: 'A', album: 'X' },
      { id: '4', artist: 'A', album: 'X' },
    ]
    const shuffled = balancedShuffle(tracks, songKeys)
    expect(ids(shuffled).sort()).toEqual(['1', '2', '3', '4'])
    expect(adjacentSame(shuffled, (t) => t.artist)).toBe(3)
  })

  it('produces no adjacent same-artist pairs when artists are balanced', () => {
    const original = Array.from({ length: 24 }, (_, i) => ({
      id: String(i),
      artist: `Artist${i % 6}`,
      album: `Album${i}`,
    }))
    for (let n = 0; n < 25; n++) {
      const shuffled = balancedShuffle(original, songKeys)
      expect(ids(shuffled).sort()).toEqual(ids(original).sort())
      expect(adjacentSame(shuffled, (t) => t.artist)).toBe(0)
    }
  })

  it('meets the minimum adjacent-artist count when one artist dominates', () => {
    const original = [
      ...Array.from({ length: 11 }, (_, i) => ({
        id: `A${i}`,
        artist: 'A',
        album: 'X',
      })),
      ...Array.from({ length: 3 }, (_, i) => ({
        id: `B${i}`,
        artist: 'B',
        album: 'Y',
      })),
      ...Array.from({ length: 2 }, (_, i) => ({
        id: `C${i}`,
        artist: 'C',
        album: 'Z',
      })),
    ]
    for (let n = 0; n < 20; n++) {
      const shuffled = balancedShuffle(original, songKeys)
      expect(ids(shuffled).sort()).toEqual(ids(original).sort())
      expect(adjacentSame(shuffled, (t) => t.artist)).toBe(minAdjacent(16, 11))
    }
  })

  it('spaces albums when every track is the same artist', () => {
    const original = Array.from({ length: 12 }, (_, i) => ({
      id: String(i),
      artist: 'A',
      album: `Album${i % 3}`,
    }))
    for (let n = 0; n < 20; n++) {
      const shuffled = balancedShuffle(original, songKeys)
      expect(ids(shuffled).sort()).toEqual(ids(original).sort())
      expect(adjacentSame(shuffled, (t) => t.album)).toBe(0)
    }
  })
})
