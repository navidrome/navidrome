import { describe, it, expect } from 'vitest'
import { calculateGain } from './calculateReplayGain'

describe('calculateGain', () => {
  const preAmp = 0

  const albumSong = {
    rgAlbumGain: -6,
    rgAlbumPeak: 1,
    rgTrackGain: -3,
    rgTrackPeak: 1,
  }
  const trackOnlySong = {
    rgTrackGain: -3,
    rgTrackPeak: 1,
  }
  const noGainSong = {}

  it('uses album gain in album mode when it is present', () => {
    const result = calculateGain({ gainMode: 'album', preAmp }, albumSong)
    expect(result).toBeCloseTo(10 ** (-6 / 20))
  })

  it('falls back to track gain in album mode when the album gain is missing', () => {
    const result = calculateGain({ gainMode: 'album', preAmp }, trackOnlySong)
    // Without the fallback this returned 1 (no adjustment). It should now use
    // the track gain instead.
    expect(result).toBeCloseTo(10 ** (-3 / 20))
    expect(result).not.toBe(1)
  })

  it('returns 1 in album mode when neither album nor track gain is present', () => {
    const result = calculateGain({ gainMode: 'album', preAmp }, noGainSong)
    expect(result).toBe(1)
  })

  it('uses track gain in track mode', () => {
    const result = calculateGain({ gainMode: 'track', preAmp }, albumSong)
    expect(result).toBeCloseTo(10 ** (-3 / 20))
  })

  it('returns 1 when gain is disabled', () => {
    const result = calculateGain({ gainMode: 'none', preAmp }, albumSong)
    expect(result).toBe(1)
  })
})
