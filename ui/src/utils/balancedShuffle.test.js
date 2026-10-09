import { balancedShuffle } from './balancedShuffle'

// Deterministic generator (mulberry32) so every assertion is reproducible
const seeded = (seed) => () => {
  seed = (seed + 0x6d2b79f5) | 0
  let t = Math.imul(seed ^ (seed >>> 15), 1 | seed)
  t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t
  return ((t ^ (t >>> 14)) >>> 0) / 4294967296
}

// spec: [[artist, albums, tracksPerAlbum], ...]
const library = (spec) =>
  spec.flatMap(([artist, albums, tracks]) =>
    [...Array(albums * tracks).keys()].map((i) => ({
      id: `${artist}-${i}`,
      artist,
      album: `${artist}-album-${i % albums}`,
    })),
  )

const singles = (count) =>
  library([...Array(count).keys()].map((i) => [`single${i}`, 1, 1]))

const options = (random) => ({
  artistKey: (t) => t.artist,
  albumKey: (t) => t.album,
  random,
})

const adjacentRepeats = (list, key = (t) => t.artist) =>
  list.slice(1).filter((t, i) => key(t) === key(list[i])).length

const longestRun = (list) => {
  let longest = list.length ? 1 : 0
  let run = 1
  for (let i = 1; i < list.length; i++) {
    run = list[i].artist === list[i - 1].artist ? run + 1 : 1
    longest = Math.max(longest, run)
  }
  return longest
}

const runs = (tracks, times, seed = 1) => {
  const random = seeded(seed)
  return [...Array(times)].map(() => balancedShuffle(tracks, options(random)))
}

const sortedIds = (list) => list.map((t) => t.id).sort()

describe('balancedShuffle', () => {
  it.each([0, 1, 2, 1000])('returns a permutation of %i items', (count) => {
    const tracks = [...Array(count).keys()].map((i) => ({
      id: `t${i}`,
      artist: ['a', 'b', 'c', 'c'][i % 4],
      album: `album-${i % 3}`,
    }))
    const result = balancedShuffle(tracks, options(seeded(7)))
    expect(result).toHaveLength(tracks.length)
    expect(sortedIds(result)).toEqual(sortedIds(tracks))
  })

  it('does not modify the input array', () => {
    const tracks = library([
      ['a', 1, 5],
      ['b', 1, 5],
    ])
    const copy = [...tracks]
    balancedShuffle(tracks, options(seeded(1)))
    expect(tracks).toEqual(copy)
  })

  it('is deterministic for the same random source', () => {
    const tracks = library([
      ['a', 2, 5],
      ['b', 1, 7],
    ]).concat(singles(20))
    expect(balancedShuffle(tracks, options(seeded(42)))).toEqual(
      balancedShuffle(tracks, options(seeded(42))),
    )
  })

  it('works with Math.random and without keys', () => {
    const tracks = library([['a', 1, 10]])
    expect(sortedIds(balancedShuffle(tracks))).toEqual(sortedIds(tracks))
  })

  it.each([
    [
      '10 artists with 10 tracks each',
      library([...Array(10).keys()].map((i) => [`a${i}`, 2, 5])),
    ],
    [
      'one artist with 10 of 100 tracks',
      library([['a', 2, 5]]).concat(singles(90)),
    ],
    [
      'artists with 50, 30 and 20 tracks',
      library([
        ['a', 5, 10],
        ['b', 3, 10],
        ['c', 2, 10],
      ]),
    ],
    [
      'one artist with 40 of 100 tracks',
      library([['a', 4, 10]]).concat(singles(60)),
    ],
    [
      'one artist with exactly half of the tracks',
      library([['a', 5, 10]]).concat(singles(50)),
    ],
  ])('never plays the same artist twice in a row: %s', (_, tracks) => {
    runs(tracks, 200).forEach((result) => {
      expect(adjacentRepeats(result)).toBe(0)
    })
  })

  it.each([
    ['60 of 100', 60, 40, 19, 2],
    ['80 of 100', 80, 20, 59, 4],
  ])(
    'splits unavoidable repeats into even runs: one artist with %s',
    (_, dominant, others, minimalRepeats, maxRun) => {
      const tracks = library([['a', 4, dominant / 4]]).concat(singles(others))
      runs(tracks, 200).forEach((result) => {
        expect(adjacentRepeats(result)).toBe(minimalRepeats)
        expect(longestRun(result)).toBe(maxRun)
      })
    },
  )

  it('spreads the albums of a single artist', () => {
    const tracks = library([['a', 3, 10]])
    const albumRepeats = runs(tracks, 200).map((result) =>
      adjacentRepeats(result, (t) => t.album),
    )
    const mean = albumRepeats.reduce((a, b) => a + b, 0) / albumRepeats.length
    // a uniform shuffle averages 29 * 9/29 = 9 adjacent tracks from one album
    expect(mean).toBeLessThan(1)
  })

  it.each([
    ['empty', ''],
    ['missing', undefined],
    ['null', null],
  ])('treats tracks with %s artist as distinct artists', (_, artist) => {
    const tracks = [...Array(10).keys()]
      .map((i) => ({ id: `t${i}`, artist }))
      .concat(library([['b', 1, 5]]))
    const results = runs(tracks, 200)
    results.forEach((result) => {
      expect(sortedIds(result)).toEqual(sortedIds(tracks))
    })
    // as one artist they would hold 10 of 15 tracks and always open the list
    expect(results.some((result) => result[0].artist === 'b')).toBe(true)
  })

  it('spreads the tracks of an artist evenly across the list', () => {
    const tracks = library([['a', 1, 10]]).concat(singles(90))
    const smallestGaps = runs(tracks, 200).map((result) => {
      const positions = result.flatMap((t, i) => (t.artist === 'a' ? [i] : []))
      return Math.min(...positions.slice(1).map((p, i) => p - positions[i]))
    })
    const mean = smallestGaps.reduce((a, b) => a + b, 0) / smallestGaps.length
    // a uniform order with its neighbours separated averages about 2
    expect(mean).toBeGreaterThan(4)
  })

  it('gives every track an even chance of every position', () => {
    const tracks = library([['a', 2, 5]]).concat(singles(90))
    const watched = [tracks[0], tracks[50]]
    const deciles = watched.map(() => Array(10).fill(0))
    const results = runs(tracks, 3000, 11)
    results.forEach((result) => {
      watched.forEach((track, w) => {
        deciles[w][Math.floor(result.indexOf(track) / 10)]++
      })
    })
    deciles.flat().forEach((count) => {
      expect(count / results.length).toBeGreaterThan(0.07)
      expect(count / results.length).toBeLessThan(0.13)
    })
  })

  it('lets an artist with many tracks open the list as often as by chance', () => {
    const tracks = library([['a', 2, 5]]).concat(singles(90))
    const results = runs(tracks, 3000, 5)
    const opened = results.filter((r) => r[0].artist === 'a').length
    expect(opened / results.length).toBeGreaterThan(0.07)
    expect(opened / results.length).toBeLessThan(0.13)
  })

  it.each([
    ['one artist', library([['a', 200, 100]])],
    ['one artist with 95%', library([['a', 190, 100]]).concat(singles(1000))],
    ['one artist with half', library([['a', 100, 100]]).concat(singles(10000))],
    [
      '100 artists',
      library([...Array(100).keys()].map((i) => [`a${i}`, 5, 20])),
    ],
  ])('shuffles 10,000+ tracks quickly: %s', (_, tracks) => {
    const start = performance.now()
    const result = balancedShuffle(tracks, options(seeded(1)))
    // generous for slow CI runners; a quadratic repair pass takes much longer
    expect(performance.now() - start).toBeLessThan(5000)
    expect(result).toHaveLength(tracks.length)
  })
})
