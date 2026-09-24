import { describe, it, expect } from 'vitest'
import { generateApiKey } from './apiKey'

describe('generateApiKey', () => {
  it('returns the prefix plus 22 base62 characters', () => {
    for (let i = 0; i < 50; i++) {
      expect(generateApiKey()).toMatch(/^nav_[0-9A-Za-z]{22}$/)
    }
  })

  it('returns a different key each time', () => {
    const keys = new Set(Array.from({ length: 100 }, generateApiKey))
    expect(keys.size).toBe(100)
  })
})
