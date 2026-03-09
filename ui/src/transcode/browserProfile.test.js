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
      if (mime === 'audio/mpeg') return 'probably'
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

  it('excludes codecs that return "maybe"', () => {
    mockCanPlayType.mockReturnValue('maybe')

    const profile = detectBrowserProfile()
    expect(profile.directPlayProfiles).toEqual([])
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
      if (mime === 'audio/mpeg') return 'probably'
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

  it('includes platform info', () => {
    const profile = detectBrowserProfile()
    expect(typeof profile.platform).toBe('string')
  })
})
