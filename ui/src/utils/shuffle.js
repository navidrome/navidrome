// Balanced shuffle: Fisher-Yates, then light adjacent artist/album repair.
// Full permutation (no drops/dupes). Does NOT drain largest artist piles —
// that over-played dominant artists (Unknown / Muse / etc).

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

const repairAdjacent = (items, getKeys) => {
  for (let i = 1; i < items.length; i++) {
    const prev = getKeys(items[i - 1])
    let cur = getKeys(items[i])

    if (prev.artist && cur.artist === prev.artist) {
      for (let j = i + 1; j < items.length; j++) {
        if (getKeys(items[j]).artist !== prev.artist) {
          ;[items[i], items[j]] = [items[j], items[i]]
          cur = getKeys(items[i])
          break
        }
      }
    }

    if (
      prev.artist &&
      cur.artist === prev.artist &&
      prev.album &&
      cur.album === prev.album
    ) {
      for (let j = i + 1; j < items.length; j++) {
        const k = getKeys(items[j])
        if (k.artist !== prev.artist || k.album !== prev.album) {
          ;[items[i], items[j]] = [items[j], items[i]]
          break
        }
      }
    }
  }
  return items
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
  return repairAdjacent(arr, getKeys)
}
