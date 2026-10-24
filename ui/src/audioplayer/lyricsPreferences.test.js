import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  readLetterWavePreference,
  readLyricsOffset,
  readPreferredSource,
  writeLetterWavePreference,
  writeLyricsOffset,
  writePreferredSource,
} from './lyricsPreferences'

describe('lyrics preferences', () => {
  afterEach(() => {
    localStorage.clear()
    vi.restoreAllMocks()
  })

  it('stores and reads the preferred source per track', () => {
    writePreferredSource('song-1', 'netease')

    expect(readPreferredSource('song-1')).toBe('netease')
    expect(localStorage.getItem('lyrics.source:song-1')).toBe('netease')

    writePreferredSource('song-1', null)
    expect(readPreferredSource('song-1')).toBeNull()
    expect(localStorage.getItem('lyrics.source:song-1')).toBeNull()
  })

  it('returns no preference for missing, blank, or unknown tracks', () => {
    expect(readPreferredSource('song-2')).toBeNull()
    expect(readPreferredSource(undefined)).toBeNull()

    writePreferredSource('song-2', '   ')
    expect(readPreferredSource('song-2')).toBeNull()
  })

  it('stores the offset per track and source', () => {
    writeLyricsOffset('song-1', 'netease', -250)

    expect(readLyricsOffset('song-1', 'netease')).toBe(-250)
    expect(readLyricsOffset('song-1', 'lrclib')).toBe(0)

    writeLyricsOffset('song-1', 'netease', 0)
    expect(localStorage.getItem('lyrics.offset:song-1:netease')).toBeNull()
  })

  it('returns zero for a corrupted stored offset', () => {
    localStorage.setItem('lyrics.offset:song-1:netease', 'not-a-number')

    expect(readLyricsOffset('song-1', 'netease')).toBe(0)
  })

  it('defaults the letter wave off and remembers when it is turned on', () => {
    expect(readLetterWavePreference()).toBe(false)

    writeLetterWavePreference(true)
    expect(readLetterWavePreference()).toBe(true)

    writeLetterWavePreference(false)
    expect(readLetterWavePreference()).toBe(false)
  })

  it('degrades silently when storage is unavailable', () => {
    const original = Object.getOwnPropertyDescriptor(window, 'localStorage')
    const deny = () => {
      throw new DOMException('Access denied', 'SecurityError')
    }
    Object.defineProperty(window, 'localStorage', {
      configurable: true,
      value: { getItem: deny, setItem: deny, removeItem: deny },
    })

    try {
      expect(() => writePreferredSource('song-1', 'netease')).not.toThrow()
      expect(() => writeLyricsOffset('song-1', 'netease', -250)).not.toThrow()
      expect(() => writeLetterWavePreference(false)).not.toThrow()
      expect(readPreferredSource('song-1')).toBeNull()
      expect(readLyricsOffset('song-1', 'netease')).toBe(0)
      expect(readLetterWavePreference()).toBe(false)
    } finally {
      if (original) Object.defineProperty(window, 'localStorage', original)
    }
  })
})
