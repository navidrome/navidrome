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

  it('includes transcoding profiles for common formats', () => {
    mockCanPlayType.mockReturnValue('')

    const profile = detectBrowserProfile()
    expect(profile.transcodingProfiles.length).toBeGreaterThan(0)
    expect(profile.transcodingProfiles[0].protocol).toBe('http')
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
