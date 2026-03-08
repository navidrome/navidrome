import subsonic from '../subsonic'
import { baseUrl } from '../utils'

// Decode the exp claim from a JWT token (no signature verification needed client-side)
export function decodeJwtExp(token) {
  try {
    if (!token) return null
    const parts = token.split('.')
    if (parts.length !== 3) return null
    const payload = JSON.parse(atob(parts[1]))
    return typeof payload.exp === 'number' ? payload.exp : null
  } catch {
    return null
  }
}

export function createDecisionService(fetchFn) {
  const cache = new Map()
  let currentProfile = null

  function isFresh(entry) {
    const exp = decodeJwtExp(entry.decision?.transcodeParams)
    if (exp == null) return false
    // exp is in seconds, Date.now() in milliseconds; 60s buffer avoids mid-request expiry
    return Date.now() < (exp - 60) * 1000
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
    cache.set(songId, { decision })
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
        cache.set(id, { decision })
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
