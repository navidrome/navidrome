/**
 * Candidate-pool fetching.
 *
 * Instead of pulling the whole library for every generation, we fetch a few
 * complementary "slices" of it (random base, favorites, most played,
 * recently played, least played) through the existing dataProvider — so
 * library selection, auth and API quirks are all reused. Every slice is
 * optional: any failure degrades gracefully.
 */

const POOL_CACHE_TTL_MS = 5 * 60 * 1000
const poolCache = new Map()

/** Same localStorage shape wrapperDataProvider uses for library filtering. */
const librarySelectionKey = () => {
  try {
    const state = JSON.parse(localStorage.getItem('state'))
    const selected = state?.library?.selectedLibraries || []
    return selected.join(',')
  } catch {
    return ''
  }
}

const safeGetList = async (dataProvider, params) => {
  try {
    const res = await dataProvider.getList('song', params)
    return Array.isArray(res?.data) ? res.data : []
  } catch {
    return []
  }
}

/**
 * Fetch (or read from cache) the candidate pool for journey generation.
 *
 * @param {Object} dataProvider react-admin data provider
 * @param {{genreId?: string, force?: boolean}} [options]
 * @returns {Promise<{songs: Object, list: Array, stats: Object, fromCache: boolean}>}
 *   `songs` is an id->raw-record map (fed straight into playTracks later),
 *   `list` the same records as an array.
 */
export async function fetchSongPool(dataProvider, { genreId, force } = {}) {
  const cacheKey = `genre:${genreId || '*'}|libs:${librarySelectionKey()}`
  if (!force) {
    const cached = poolCache.get(cacheKey)
    if (cached && Date.now() - cached.at < POOL_CACHE_TTL_MS) {
      return { ...cached.payload, fromCache: true }
    }
  }

  const baseFilter = { missing: false, ...(genreId && { genre_id: [genreId] }) }

  const slices = await Promise.all([
    // Broad random sample of the (filtered) library.
    safeGetList(dataProvider, {
      pagination: { page: 1, perPage: 400 },
      sort: { field: 'random', order: 'ASC' },
      filter: baseFilter,
    }),
    // Favorites.
    safeGetList(dataProvider, {
      pagination: { page: 1, perPage: 200 },
      sort: { field: 'starredAt', order: 'DESC' },
      filter: { ...baseFilter, starred: true },
    }),
    // Most played.
    safeGetList(dataProvider, {
      pagination: { page: 1, perPage: 200 },
      sort: { field: 'playCount', order: 'DESC' },
      filter: baseFilter,
    }),
    // Recently played.
    safeGetList(dataProvider, {
      pagination: { page: 1, perPage: 150 },
      sort: { field: 'playDate', order: 'DESC' },
      filter: baseFilter,
    }),
    // Longest-unplayed first: the discovery material.
    safeGetList(dataProvider, {
      pagination: { page: 1, perPage: 150 },
      sort: { field: 'playDate', order: 'ASC' },
      filter: baseFilter,
    }),
  ])

  const songs = {}
  const sliceNames = ['base', 'starred', 'top', 'recent', 'rare']
  const stats = {}
  slices.forEach((records, i) => {
    let added = 0
    for (const record of records) {
      if (record?.id && record.missing !== true && !songs[record.id]) {
        songs[record.id] = record
        added++
      }
    }
    stats[sliceNames[i]] = added
  })

  const payload = { songs, list: Object.values(songs), stats }
  poolCache.set(cacheKey, { at: Date.now(), payload })
  return { ...payload, fromCache: false }
}

/** Test helper / cache invalidation hook (e.g. after library scan). */
export const clearSongPoolCache = () => poolCache.clear()
