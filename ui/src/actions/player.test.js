import { PLAYER_PLAY_TRACKS, shuffle, shuffleTracks } from './player'

const songs = (spec) =>
  Object.fromEntries(
    spec.flatMap(([artistId, count]) =>
      [...Array(count).keys()].map((i) => [
        `${artistId}${i}`,
        { id: `${artistId}${i}`, artistId, albumId: `${artistId}-album` },
      ]),
    ),
  )

describe('shuffle', () => {
  it('keeps every song and prefixes the keys to preserve the order', () => {
    const data = songs([
      ['a', 5],
      ['b', 5],
    ])
    const shuffled = shuffle(data)
    expect(Object.keys(shuffled).sort()).toEqual(
      Object.keys(data)
        .map((id) => `_${id}`)
        .sort(),
    )
    Object.entries(shuffled).forEach(([key, song]) => {
      expect(song).toBe(data[key.substring(1)])
    })
  })

  it('does not play the same artist twice in a row when avoidable', () => {
    const data = songs([
      ['a', 10],
      ['b', 10],
      ['c', 10],
    ])
    for (let n = 0; n < 50; n++) {
      const artists = Object.values(shuffle(data)).map((s) => s.artistId)
      artists.slice(1).forEach((artist, i) => {
        expect(artist).not.toBe(artists[i])
      })
    }
  })

  it('spreads the albums of an artist', () => {
    const data = Object.fromEntries(
      [...Array(10).keys()].map((i) => [
        `s${i}`,
        { id: `s${i}`, artistId: 'a', albumId: `album-${i % 2}` },
      ]),
    )
    let albumRepeats = 0
    for (let n = 0; n < 50; n++) {
      const albums = Object.values(shuffle(data)).map((s) => s.albumId)
      albumRepeats += albums.slice(1).filter((a, i) => a === albums[i]).length
    }
    // a uniform shuffle averages 9 * 4/9 = 4 adjacent tracks from one album
    expect(albumRepeats / 50).toBeLessThan(1)
  })

  it('falls back to the artist name when there is no artistId', () => {
    const data = {
      1: { id: '1', artist: 'x' },
      2: { id: '2', artist: 'x' },
      3: { id: '3', artist: 'y' },
    }
    for (let n = 0; n < 20; n++) {
      const artists = Object.values(shuffle(data)).map((s) => s.artist)
      expect(artists).toEqual(['x', 'y', 'x'])
    }
  })
})

describe('shuffleTracks', () => {
  it('plays all non-missing songs starting with the first shuffled one', () => {
    const data = {
      ...songs([
        ['a', 3],
        ['b', 3],
      ]),
      gone: { id: 'gone', artistId: 'a', missing: true },
    }
    const action = shuffleTracks(data)
    expect(action.type).toBe(PLAYER_PLAY_TRACKS)
    expect(Object.keys(action.data)).toHaveLength(6)
    expect(action.data).not.toHaveProperty('_gone')
    expect(action.id).toBe(Object.keys(action.data)[0])
  })

  it('only shuffles the selected ids', () => {
    const data = songs([['a', 4]])
    const action = shuffleTracks(data, ['a0', 'a2'])
    expect(Object.keys(action.data).sort()).toEqual(['_a0', '_a2'])
  })
})
