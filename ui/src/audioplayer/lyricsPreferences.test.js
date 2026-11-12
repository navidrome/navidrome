import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  readLetterWavePreference,
  writeLetterWavePreference,
} from './lyricsPreferences'

describe('lyrics preferences', () => {
  afterEach(() => {
    localStorage.clear()
    vi.restoreAllMocks()
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
      expect(() => writeLetterWavePreference(true)).not.toThrow()
      expect(readLetterWavePreference()).toBe(false)
    } finally {
      if (original) Object.defineProperty(window, 'localStorage', original)
    }
  })
})
