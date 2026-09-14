// Balanced shuffle: Fisher-Yates, then greedy artist/album spacing.
// Full permutation (no drops/dupes). Falls back when spacing is impossible.

const randomFloat = () => {
  if (typeof crypto !== 'undefined' && crypto.getRandomValues) {
    const buf = new Uint32Array(1)
    crypto.getRandomValues(buf)
    return buf[0] / 2 ** 32
  }
  return Math.random()
}

const fisherYates = (items, random) => {
  for (let i = items.length - 1; i > 0; i--) {
    const j = Math.floor(random() * (i + 1))
    ;[items[i], items[j]] = [items[j], items[i]]
  }
}

export const songKeys = (song) => ({
  artist: song?.artist || song?.singer || song?.albumArtist || '',
  album: song?.album || '',
})

const pickKey = (items, last, hasLast, keyFn) => {
  const counts = new Map()
  const order = []
  for (const item of items) {
    const k = keyFn(item)
    if (!counts.has(k)) {
      order.push(k)
      counts.set(k, 0)
    }
    counts.set(k, counts.get(k) + 1)
  }

  let others = 0
  if (hasLast) {
    for (const k of counts.keys()) {
      if (k !== last) {
        others++
      }
    }
  }

  let best = order[0]
  let bestN = -1
  for (const k of order) {
    if (hasLast && others > 0 && k === last) {
      continue
    }
    const n = counts.get(k)
    if (n > bestN) {
      bestN = n
      best = k
    }
  }
  return best
}

const space = (items, getKeys) => {
  const remaining = items.slice()
  const result = []
  let hasLast = false
  let lastArtist = ''
  let lastAlbum = ''

  while (remaining.length > 0) {
    const artist = pickKey(
      remaining,
      lastArtist,
      hasLast,
      (item) => getKeys(item).artist || '',
    )
    const sameArtist = remaining.filter(
      (item) => (getKeys(item).artist || '') === artist,
    )
    const album = pickKey(
      sameArtist,
      lastAlbum,
      hasLast,
      (item) => getKeys(item).album || '',
    )
    const pick = remaining.findIndex((item) => {
      const keys = getKeys(item)
      return (keys.artist || '') === artist && (keys.album || '') === album
    })
    const chosen = remaining.splice(pick, 1)[0]
    result.push(chosen)
    const keys = getKeys(chosen)
    lastArtist = keys.artist || ''
    lastAlbum = keys.album || ''
    hasLast = true
  }

  return result
}

export const balancedShuffle = (
  items,
  getKeys = songKeys,
  random = randomFloat,
) => {
  const arr = items.slice()
  if (arr.length < 2) {
    return arr
  }
  fisherYates(arr, random)
  return space(arr, getKeys)
}
