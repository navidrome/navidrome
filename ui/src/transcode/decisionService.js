import subsonic from '../subsonic'
import { baseUrl } from '../utils'

// Cache entries expire after 11 hours (tokens last 12h, this gives 1h buffer)
export const CACHE_TTL_MS = 11 * 60 * 60 * 1000

export function createDecisionService(fetchFn) {
  const cache = new Map()
  let currentProfile = null

  function isFresh(entry) {
    return Date.now() - entry.fetchedAt < CACHE_TTL_MS
  }

  function setProfile(profile) {
    currentProfile = profile
  }

  function getProfile() {
    return currentProfile
  }

  async function getDecision(songId, browserProfile) {
    const profile = browserProfile || currentProfile
    if (!profile) return null

    const cached = cache.get(songId)
    if (cached && isFresh(cached)) {
      return cached.decision
    }

    const decision = await fetchFn(songId, profile)
    cache.set(songId, { decision, fetchedAt: Date.now() })
    return decision
  }

  async function prefetchDecisions(songIds, browserProfile) {
    const profile = browserProfile || currentProfile
    if (!profile) return

    const uncached = songIds.filter((id) => {
      const entry = cache.get(id)
      return !entry || !isFresh(entry)
    })

    await Promise.allSettled(
      uncached.map(async (id) => {
        const decision = await fetchFn(id, profile)
        cache.set(id, { decision, fetchedAt: Date.now() })
      }),
    )
  }

  function invalidateAll() {
    cache.clear()
  }

  function buildStreamUrl(songId, transcodeParams, offset) {
    const params = {
      mediaId: songId,
      mediaType: 'song',
      transcodeParams,
    }
    if (offset != null && offset > 0) {
      params.offset = offset
    }
    return baseUrl(subsonic.url('getTranscodeStream', null, params))
  }

  function getCachedDecision(songId) {
    const entry = cache.get(songId)
    if (entry && isFresh(entry)) {
      return entry.decision
    }
    return null
  }

  return {
    getDecision,
    getCachedDecision,
    prefetchDecisions,
    invalidateAll,
    buildStreamUrl,
    setProfile,
    getProfile,
  }
}
