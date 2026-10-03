import { describe, it, expect, beforeEach } from 'vitest'
import { fetchSongPool, clearSongPoolCache } from './fetchSongPool'

const makeDataProvider = (handlers) => ({
  getList: async (resource, params) => {
    const handler = handlers[params.sort?.field] || handlers['*']
    if (!handler) return { data: [] }
    const data = handler(params)
    if (data instanceof Error) throw data
    return { data }
  },
})

const song = (id, extra = {}) => ({
  id,
  title: `Song ${id}`,
  missing: false,
  ...extra,
})

describe('fetchSongPool', () => {
  beforeEach(() => clearSongPoolCache())

  it('merges and dedupes all library slices', async () => {
    const dp = makeDataProvider({
      random: () => [song('a'), song('b')],
      starredAt: () => [
        song('b', { starred: true }),
        song('c', { starred: true }),
      ],
      playCount: () => [song('a', { playCount: 9 }), song('d')],
      playDate: () => [song('e')],
      '*': () => [],
    })
    const pool = await fetchSongPool(dp, {})
    expect(pool.list.map((s) => s.id).sort()).toEqual(['a', 'b', 'c', 'd', 'e'])
    expect(pool.stats.base).toBe(2)
    expect(pool.fromCache).toBe(false)
  })

  it('serves from cache on the second call', async () => {
    const dp = makeDataProvider({ random: () => [song('x')] })
    await fetchSongPool(dp, {})
    const second = await fetchSongPool(dp, {})
    expect(second.fromCache).toBe(true)
    expect(second.list).toHaveLength(1)
  })

  it('degrades gracefully when individual slices fail', async () => {
    const dp = makeDataProvider({
      random: () => [song('ok')],
      starredAt: () => new Error('boom'),
      playCount: () => new Error('boom'),
      playDate: () => new Error('boom'),
      '*': () => [],
    })
    const pool = await fetchSongPool(dp, {})
    expect(pool.list).toHaveLength(1)
    expect(pool.stats.starred).toBe(0)
  })

  it('excludes missing tracks and passes the genre filter through', async () => {
    let seenFilter = null
    const dp = {
      getList: async (_, params) => {
        seenFilter = params.filter
        return { data: [song('a'), song('b', { missing: true })] }
      },
    }
    const pool = await fetchSongPool(dp, { genreId: 'g-1' })
    expect(seenFilter.genre_id).toEqual(['g-1'])
    expect(seenFilter.missing).toBe(false)
    expect(pool.list.map((s) => s.id)).toEqual(['a'])
  })
})
