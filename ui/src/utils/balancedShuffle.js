/* balancedShuffle: Return a shuffled copy of the input array in which items
 * sharing an artist are spread across the whole list instead of clustering.
 *
 * A uniform shuffle often places tracks by the same artist next to each other
 * (with 10 tracks by one artist in a 100-track playlist, 63% of all orders do),
 * which listeners do not perceive as random. This follows the approach Spotify
 * described in "How to shuffle songs?" (2014): every artist's tracks are
 * stretched evenly along the list with a random offset and a small random
 * jitter, and the list is ordered by those positions. Tracks of one artist are
 * spread across their albums the same way. Positions wrap around, so an artist
 * with many tracks is as likely to open the list as any other track. A final
 * pass removes the adjacent repeats the spreading leaves behind. When one
 * artist has more tracks than all others together (plus one), repeats are
 * unavoidable; the artist's tracks are then split into runs of equal length.
 *
 * Options:
 * - artistKey(item) / albumKey(item): grouping keys. Items without a key are
 *   treated as their own group.
 * - random: a () => [0, 1) generator, Math.random by default.
 */
export const balancedShuffle = (items, options = {}) => {
  const {
    artistKey = () => undefined,
    albumKey = () => undefined,
    random = Math.random,
  } = options
  const entries = items.map((item) => ({
    item,
    artist: keyOrUnique(artistKey(item)),
    album: keyOrUnique(albumKey(item)),
  }))
  return balance(entries, random).map((e) => e.item)
}

const keyOrUnique = (key) =>
  key === undefined || key === null || key === '' ? Symbol() : key

const fisherYates = (list, random) => {
  const shuffled = [...list]
  for (let i = shuffled.length - 1; i > 0; i--) {
    const j = Math.floor(random() * (i + 1))
    ;[shuffled[i], shuffled[j]] = [shuffled[j], shuffled[i]]
  }
  return shuffled
}

const groupBy = (entries, field) => {
  const groups = new Map()
  entries.forEach((e) => {
    if (!groups.has(e[field])) {
      groups.set(e[field], [])
    }
    groups.get(e[field]).push(e)
  })
  return groups
}

const dominantArtist = (entries) => {
  const groups = groupBy(entries, 'artist')
  let dominant = { artist: undefined, count: 0 }
  groups.forEach((group, artist) => {
    if (group.length > dominant.count) {
      dominant = { artist, count: group.length }
    }
  })
  return dominant
}

// Adjacent repeats can be avoided only if no artist holds more than half
// of the items (rounded up).
const isAvoidable = (entries) => {
  const { count } = dominantArtist(entries)
  return count <= entries.length - count + 1
}

// Stretch every group evenly along a circle of length 1: the k-th of n items
// lands at k/n plus a random offset per group plus up to 10% of the spacing
// as jitter. orderGroup decides which item of a group takes which slot.
const spread = (entries, field, random, orderGroup) => {
  const positioned = []
  groupBy(entries, field).forEach((group) => {
    const ordered = orderGroup(group)
    const spacing = 1 / ordered.length
    const offset = random()
    ordered.forEach((entry, k) => {
      const jitter = (random() * 2 - 1) * 0.1 * spacing
      const position = (((k * spacing + offset + jitter) % 1) + 1) % 1
      positioned.push({ entry, position, tiebreak: random() })
    })
  })
  positioned.sort((a, b) => a.position - b.position || a.tiebreak - b.tiebreak)
  return positioned.map((p) => p.entry)
}

// Remove adjacent repeats of the same artist. A repeat at index i is fixed by
// pulling the next item of another artist forward; when none is left, the
// item moves to a random earlier gap between two other artists.
const repairAdjacent = (entries, random) => {
  const list = [...entries]
  let i = 1
  while (i < list.length) {
    if (list[i].artist !== list[i - 1].artist) {
      i++
      continue
    }
    const previous = list[i - 1].artist
    let j = i + 1
    while (j < list.length && list[j].artist === previous) {
      j++
    }
    if (j < list.length) {
      list.splice(i, 0, ...list.splice(j, 1))
      i++
      continue
    }
    const [entry] = list.splice(i, 1)
    const gaps = list[0].artist !== entry.artist ? [0] : []
    for (let p = 1; p < i; p++) {
      if (
        list[p - 1].artist !== entry.artist &&
        list[p].artist !== entry.artist
      ) {
        gaps.push(p)
      }
    }
    if (gaps.length === 0) {
      list.splice(i, 0, entry)
      i++
      continue
    }
    list.splice(gaps[Math.floor(random() * gaps.length)], 0, entry)
  }
  return list
}

const balance = (entries, random) => {
  if (entries.length < 2) {
    return [...entries]
  }
  const byAlbum = (group) =>
    spread(group, 'album', random, (album) => fisherYates(album, random))
  if (isAvoidable(entries)) {
    return repairAdjacent(spread(entries, 'artist', random, byAlbum), random)
  }

  // Too many items by one artist: place the others first, then fill the
  // gaps around them with runs of the dominant artist of (almost) equal size.
  const { artist } = dominantArtist(entries)
  const others = balance(
    entries.filter((e) => e.artist !== artist),
    random,
  )
  const dominant = byAlbum(entries.filter((e) => e.artist === artist))
  const slots = others.length + 1
  const baseRun = Math.floor(dominant.length / slots)
  const longerRuns = new Set(
    fisherYates([...Array(slots).keys()], random).slice(
      0,
      dominant.length % slots,
    ),
  )
  const result = []
  let next = 0
  for (let slot = 0; slot < slots; slot++) {
    const run = baseRun + (longerRuns.has(slot) ? 1 : 0)
    result.push(...dominant.slice(next, next + run))
    next += run
    if (slot < others.length) {
      result.push(others[slot])
    }
  }
  return result
}
