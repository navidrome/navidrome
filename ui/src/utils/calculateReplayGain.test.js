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
  const nativeTrackOnlySong = {
    rgAlbumGain: null,
    rgAlbumPeak: null,
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
    expect(result).toBeCloseTo(10 ** (-3 / 20))
  })

  it('falls back to track gain in album mode when the album gain is null', () => {
    const result = calculateGain(
      { gainMode: 'album', preAmp },
      nativeTrackOnlySong,
    )
    expect(result).toBeCloseTo(10 ** (-3 / 20))
  })

  it.each([null, undefined])(
    'keeps the album gain in album mode when the album peak is %s',
    (rgAlbumPeak) => {
      const result = calculateGain(
        { gainMode: 'album', preAmp },
        { ...albumSong, rgAlbumPeak },
      )
      expect(result).toBeCloseTo(10 ** (-6 / 20))
    },
  )

  it('returns 1 in album mode when neither album nor track gain is present', () => {
    const result = calculateGain({ gainMode: 'album', preAmp }, noGainSong)
    expect(result).toBe(1)
  })

  it('uses track gain in track mode', () => {
    const result = calculateGain({ gainMode: 'track', preAmp }, albumSong)
    expect(result).toBeCloseTo(10 ** (-3 / 20))
  })

  it.each([null, undefined])(
    'keeps the track gain in track mode when the track peak is %s',
    (rgTrackPeak) => {
      const result = calculateGain(
        { gainMode: 'track', preAmp },
        { ...albumSong, rgTrackPeak },
      )
      expect(result).toBeCloseTo(10 ** (-3 / 20))
    },
  )

  it('returns 1 when gain is disabled', () => {
    const result = calculateGain({ gainMode: 'none', preAmp }, albumSong)
    expect(result).toBe(1)
  })
})
