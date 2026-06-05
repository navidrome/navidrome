import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { createDecisionService, decodeJwtExp } from './decisionService'

// Helper: create a fake JWT with a given exp (seconds since epoch)
function fakeJwt(expSeconds) {
  const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }))
  const payload = btoa(JSON.stringify({ exp: expSeconds }))
  return `${header}.${payload}.fake-signature`
}

// Helper: create a fake JWT with no exp claim
function fakeJwtNoExp() {
  const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }))
  const payload = btoa(JSON.stringify({ sub: 'test' }))
  return `${header}.${payload}.fake-signature`
}

describe('decodeJwtExp', () => {
  it('extracts exp from a valid JWT', () => {
    const exp = 1700000000
    expect(decodeJwtExp(fakeJwt(exp))).toBe(exp)
  })

  it('returns null for JWT without exp claim', () => {
    expect(decodeJwtExp(fakeJwtNoExp())).toBeNull()
  })

  it('returns null for non-JWT string', () => {
    expect(decodeJwtExp('not-a-jwt')).toBeNull()
  })

  it('returns null for empty string', () => {
    expect(decodeJwtExp('')).toBeNull()
  })

  it('returns null for null/undefined', () => {
    expect(decodeJwtExp(null)).toBeNull()
    expect(decodeJwtExp(undefined)).toBeNull()
  })
})

describe('decisionService', () => {
  let service
  let mockFetchFn

  const fakeProfile = {
    name: 'NavidromeUI',
    platform: 'test',
    directPlayProfiles: [],
    transcodingProfiles: [],
    codecProfiles: [],
  }

  // Token that expires 1 hour from "now" (will be relative to fake timers)
  function makeFakeDecision(expiresInMs = 3600 * 1000) {
    const expSeconds = Math.floor((Date.now() + expiresInMs) / 1000)
    return {
      canDirectPlay: true,
      canTranscode: false,
      transcodeParams: fakeJwt(expSeconds),
      sourceStream: { codec: 'mp3', container: 'mp3' },
    }
  }

  beforeEach(() => {
    localStorage.setItem('username', 'testuser')
    localStorage.setItem('subsonic-token', 'testtoken')
    localStorage.setItem('subsonic-salt', 'testsalt')
    mockFetchFn = vi.fn().mockImplementation(() => {
      return Promise.resolve(makeFakeDecision())
    })
    service = createDecisionService(mockFetchFn)
  })

  afterEach(() => {
    vi.restoreAllMocks()
    localStorage.clear()
  })

  describe('getDecision', () => {
    it('fetches and caches a decision', async () => {
      const result = await service.getDecision('song-1', fakeProfile)
      expect(result.canDirectPlay).toBe(true)
      expect(mockFetchFn).toHaveBeenCalledTimes(1)
      expect(mockFetchFn).toHaveBeenCalledWith('song-1', fakeProfile)

      // Second call uses cache
      const result2 = await service.getDecision('song-1', fakeProfile)
      expect(result2).toEqual(result)
      expect(mockFetchFn).toHaveBeenCalledTimes(1)
    })

    it('re-fetches after token expires', async () => {
      vi.useFakeTimers()

      // Token expires in 1 hour
      mockFetchFn.mockResolvedValue(makeFakeDecision(3600 * 1000))
      await service.getDecision('song-1', fakeProfile)
      expect(mockFetchFn).toHaveBeenCalledTimes(1)

      // Advance past expiration
      vi.advanceTimersByTime(3600 * 1000 + 1000)
      await service.getDecision('song-1', fakeProfile)
      expect(mockFetchFn).toHaveBeenCalledTimes(2)
      vi.useRealTimers()
    })

    it('does not re-fetch before token expires', async () => {
      vi.useFakeTimers()

      // Token expires in 1 hour
      mockFetchFn.mockResolvedValue(makeFakeDecision(3600 * 1000))
      await service.getDecision('song-1', fakeProfile)

      // 30 minutes later — still fresh
      vi.advanceTimersByTime(1800 * 1000)
      await service.getDecision('song-1', fakeProfile)
      expect(mockFetchFn).toHaveBeenCalledTimes(1)
      vi.useRealTimers()
    })

    it('re-fetches immediately when token has no exp claim', async () => {
      const noExpDecision = {
        canDirectPlay: true,
        canTranscode: false,
        transcodeParams: fakeJwtNoExp(),
        sourceStream: { codec: 'mp3', container: 'mp3' },
      }
      mockFetchFn.mockResolvedValue(noExpDecision)
      await service.getDecision('song-1', fakeProfile)

      // Should re-fetch because token has no exp
      await service.getDecision('song-1', fakeProfile)
      expect(mockFetchFn).toHaveBeenCalledTimes(2)
    })

    it('caches different songs independently', async () => {
      await service.getDecision('song-1', fakeProfile)
      await service.getDecision('song-2', fakeProfile)
      expect(mockFetchFn).toHaveBeenCalledTimes(2)
    })
  })

  describe('getCachedDecision', () => {
    it('returns null when song is not cached', () => {
      expect(service.getCachedDecision('song-1')).toBeNull()
    })

    it('returns cached decision after getDecision', async () => {
      await service.getDecision('song-1', fakeProfile)
      const cached = service.getCachedDecision('song-1')
      expect(cached).not.toBeNull()
      expect(cached.canDirectPlay).toBe(true)
    })

    it('returns null after cache is invalidated', async () => {
      await service.getDecision('song-1', fakeProfile)
      service.invalidateAll()
      expect(service.getCachedDecision('song-1')).toBeNull()
    })

    it('returns null after token expires', async () => {
      vi.useFakeTimers()
      mockFetchFn.mockResolvedValue(makeFakeDecision(3600 * 1000))
      await service.getDecision('song-1', fakeProfile)

      vi.advanceTimersByTime(3600 * 1000 + 1000)
      expect(service.getCachedDecision('song-1')).toBeNull()
      vi.useRealTimers()
    })
  })

  describe('prefetchDecisions', () => {
    it('fetches decisions for uncached songs', async () => {
      await service.prefetchDecisions(['song-1', 'song-2'], fakeProfile)
      expect(mockFetchFn).toHaveBeenCalledTimes(2)
    })

    it('skips already cached songs', async () => {
      await service.getDecision('song-1', fakeProfile)
      mockFetchFn.mockClear()

      await service.prefetchDecisions(['song-1', 'song-2'], fakeProfile)
      expect(mockFetchFn).toHaveBeenCalledTimes(1)
      expect(mockFetchFn).toHaveBeenCalledWith('song-2', fakeProfile)
    })

    it('silently ignores fetch errors', async () => {
      mockFetchFn.mockRejectedValue(new Error('network error'))
      await expect(
        service.prefetchDecisions(['song-1'], fakeProfile),
      ).resolves.not.toThrow()
    })
  })

  describe('invalidateAll', () => {
    it('clears cache so next getDecision re-fetches', async () => {
      await service.getDecision('song-1', fakeProfile)
      expect(mockFetchFn).toHaveBeenCalledTimes(1)

      service.invalidateAll()

      await service.getDecision('song-1', fakeProfile)
      expect(mockFetchFn).toHaveBeenCalledTimes(2)
    })
  })

  describe('resolveStreamUrl', () => {
    it('fetches decision and returns built URL', async () => {
      service.setProfile(fakeProfile)
      const url = await service.resolveStreamUrl('song-1')
      expect(url).toContain('getTranscodeStream')
      expect(url).toContain('mediaId=song-1')
      expect(mockFetchFn).toHaveBeenCalledTimes(1)
    })

    it('falls back to stream URL when decision has no transcodeParams', async () => {
      service.setProfile(fakeProfile)
      mockFetchFn.mockResolvedValue({
        canDirectPlay: true,
        canTranscode: false,
      })
      const url = await service.resolveStreamUrl('song-1')
      expect(url).toContain('stream')
      expect(url).not.toContain('getTranscodeStream')
    })

    it('falls back to stream URL when decision is null', async () => {
      service.setProfile(fakeProfile)
      mockFetchFn.mockResolvedValue(null)
      const url = await service.resolveStreamUrl('song-1')
      expect(url).toContain('stream')
      expect(url).not.toContain('getTranscodeStream')
    })
  })

  describe('buildStreamUrl', () => {
    it('builds URL with required parameters', () => {
      const url = service.buildStreamUrl('song-1', 'jwt-token-123')
      expect(url).toContain('getTranscodeStream')
      expect(url).toContain('mediaId=song-1')
      expect(url).toContain('mediaType=song')
      expect(url).toContain('transcodeParams=jwt-token-123')
    })

    it('includes offset when provided', () => {
      const url = service.buildStreamUrl('song-1', 'jwt-token-123', 30)
      expect(url).toContain('offset=30')
    })

    it('omits offset when not provided', () => {
      const url = service.buildStreamUrl('song-1', 'jwt-token-123')
      expect(url).not.toContain('offset')
    })
  })
})
