import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { createDecisionService, CACHE_TTL_MS } from './decisionService'

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

  const fakeDecision = {
    canDirectPlay: true,
    canTranscode: false,
    transcodeParams: 'jwt-token-123',
    sourceStream: { codec: 'mp3', container: 'mp3' },
  }

  beforeEach(() => {
    localStorage.setItem('username', 'testuser')
    localStorage.setItem('subsonic-token', 'testtoken')
    localStorage.setItem('subsonic-salt', 'testsalt')
    mockFetchFn = vi.fn().mockResolvedValue(fakeDecision)
    service = createDecisionService(mockFetchFn)
  })

  afterEach(() => {
    vi.restoreAllMocks()
    localStorage.clear()
  })

  describe('getDecision', () => {
    it('fetches and caches a decision', async () => {
      const result = await service.getDecision('song-1', fakeProfile)
      expect(result).toEqual(fakeDecision)
      expect(mockFetchFn).toHaveBeenCalledTimes(1)
      expect(mockFetchFn).toHaveBeenCalledWith('song-1', fakeProfile)

      // Second call uses cache
      const result2 = await service.getDecision('song-1', fakeProfile)
      expect(result2).toEqual(fakeDecision)
      expect(mockFetchFn).toHaveBeenCalledTimes(1)
    })

    it('re-fetches after TTL expires', async () => {
      vi.useFakeTimers()
      await service.getDecision('song-1', fakeProfile)
      expect(mockFetchFn).toHaveBeenCalledTimes(1)

      vi.advanceTimersByTime(CACHE_TTL_MS + 1)
      await service.getDecision('song-1', fakeProfile)
      expect(mockFetchFn).toHaveBeenCalledTimes(2)
      vi.useRealTimers()
    })

    it('does not re-fetch before TTL expires', async () => {
      vi.useFakeTimers()
      await service.getDecision('song-1', fakeProfile)

      vi.advanceTimersByTime(CACHE_TTL_MS - 1000)
      await service.getDecision('song-1', fakeProfile)
      expect(mockFetchFn).toHaveBeenCalledTimes(1)
      vi.useRealTimers()
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
      expect(service.getCachedDecision('song-1')).toEqual(fakeDecision)
    })

    it('returns null after cache is invalidated', async () => {
      await service.getDecision('song-1', fakeProfile)
      service.invalidateAll()
      expect(service.getCachedDecision('song-1')).toBeNull()
    })

    it('returns null after TTL expires', async () => {
      vi.useFakeTimers()
      await service.getDecision('song-1', fakeProfile)
      vi.advanceTimersByTime(CACHE_TTL_MS + 1)
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
