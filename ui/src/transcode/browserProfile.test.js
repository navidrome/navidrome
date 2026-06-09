import { describe, it, expect, beforeEach, vi } from 'vitest'
import { detectBrowserProfile, CODEC_PROBES } from './browserProfile'

describe('detectBrowserProfile', () => {
  let mockCanPlayType

  beforeEach(() => {
    mockCanPlayType = vi.fn()
    vi.stubGlobal(
      'Audio',
      class {
        canPlayType = mockCanPlayType
      },
    )
  })

  it('includes codecs that return "probably"', () => {
    mockCanPlayType.mockImplementation((mime) => {
      if (mime === 'audio/mpeg; codecs="mp3"') return 'probably'
      if (mime === 'audio/ogg; codecs="opus"') return 'probably'
      return ''
    })

    const profile = detectBrowserProfile()

    expect(profile.name).toBe('NavidromeUI')
    expect(profile.directPlayProfiles.length).toBe(2)

    const codecs = profile.directPlayProfiles.flatMap((p) => p.audioCodecs)
    expect(codecs).toContain('mp3')
    expect(codecs).toContain('opus')
  })

  it('includes codecs that return "maybe"', () => {
    mockCanPlayType.mockImplementation((mime) => {
      if (mime === 'audio/flac') return 'maybe'
      return ''
    })

    const profile = detectBrowserProfile()
    const codecs = profile.directPlayProfiles.flatMap((p) => p.audioCodecs)
    expect(codecs).toContain('flac')
  })

  it('excludes codecs that return empty string', () => {
    mockCanPlayType.mockReturnValue('')

    const profile = detectBrowserProfile()
    expect(profile.directPlayProfiles).toEqual([])
  })

  it('sets protocol to "http" for all direct play profiles', () => {
    mockCanPlayType.mockReturnValue('probably')

    const profile = detectBrowserProfile()
    profile.directPlayProfiles.forEach((p) => {
      expect(p.protocols).toEqual(['http'])
    })
  })

  it('filters transcoding profiles by canPlayType', () => {
    mockCanPlayType.mockImplementation((mime) => {
      if (mime === 'audio/mpeg; codecs="mp3"') return 'probably'
      if (mime === 'audio/ogg; codecs="opus"') return 'probably'
      return ''
    })

    const profile = detectBrowserProfile()
    const codecs = profile.transcodingProfiles.map((p) => p.audioCodec)
    expect(codecs).toEqual(['opus', 'mp3'])
    expect(codecs).not.toContain('flac')
    profile.transcodingProfiles.forEach((p) => {
      expect(p.protocol).toBe('http')
    })
  })

  it('always includes mp3 fallback in transcoding profiles', () => {
    mockCanPlayType.mockReturnValue('')

    const profile = detectBrowserProfile()
    expect(profile.transcodingProfiles.length).toBe(1)
    expect(profile.transcodingProfiles[0].audioCodec).toBe('mp3')
    expect(profile.transcodingProfiles[0].protocol).toBe('http')
  })

  it('does not duplicate mp3 when canPlayType supports it', () => {
    mockCanPlayType.mockReturnValue('probably')

    const profile = detectBrowserProfile()
    const mp3Count = profile.transcodingProfiles.filter(
      (p) => p.audioCodec === 'mp3',
    ).length
    expect(mp3Count).toBe(1)
  })

  it('preserves transcoding profile preference order', () => {
    mockCanPlayType.mockReturnValue('probably')

    const profile = detectBrowserProfile()
    const codecs = profile.transcodingProfiles.map((p) => p.audioCodec)
    expect(codecs).toEqual(['flac', 'opus', 'mp3'])
  })

  it('sets codecProfiles to empty array', () => {
    mockCanPlayType.mockReturnValue('probably')

    const profile = detectBrowserProfile()
    expect(profile.codecProfiles).toEqual([])
  })

  it('matches codec when any mime variant returns "probably"', () => {
    mockCanPlayType.mockImplementation((mime) => {
      if (mime === 'audio/flac; codecs="flac"') return 'probably'
      return ''
    })

    const profile = detectBrowserProfile()
    const codecs = profile.directPlayProfiles.flatMap((p) => p.audioCodecs)
    expect(codecs).toContain('flac')
  })

  it('includes platform info', () => {
    const profile = detectBrowserProfile()
    expect(typeof profile.platform).toBe('string')
  })

  describe('Safari restrictions', () => {
    beforeEach(() => {
      // Safari reports canPlayType for Ogg as positive, but can't actually
      // stream transcoded Ogg. Simulate Safari: supports everything.
      mockCanPlayType.mockReturnValue('probably')
    })

    it('still includes ogg in direct play profiles on Safari', () => {
      vi.stubGlobal('navigator', {
        userAgent:
          'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.2 Safari/605.1.15',
      })

      const profile = detectBrowserProfile()
      const containers = profile.directPlayProfiles.flatMap((p) => p.containers)
      expect(containers).toContain('ogg')
    })

    it('limits Safari transcoding to mp3 only', () => {
      vi.stubGlobal('navigator', {
        userAgent:
          'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.2 Safari/605.1.15',
      })

      const profile = detectBrowserProfile()
      const codecs = profile.transcodingProfiles.map((p) => p.audioCodec)
      expect(codecs).toEqual(['mp3'])
    })

    it('does NOT restrict transcoding on Chrome', () => {
      vi.stubGlobal('navigator', {
        userAgent:
          'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
      })

      const profile = detectBrowserProfile()
      const codecs = profile.transcodingProfiles.map((p) => p.audioCodec)
      expect(codecs).toContain('opus')
      expect(codecs).toContain('flac')
    })

    it('applies same restrictions on iOS Safari', () => {
      vi.stubGlobal('navigator', {
        userAgent:
          'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1',
      })

      const profile = detectBrowserProfile()
      const codecs = profile.transcodingProfiles.map((p) => p.audioCodec)
      expect(codecs).toEqual(['mp3'])
    })
  })
})
