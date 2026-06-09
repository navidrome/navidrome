import { describe, it, expect, vi, beforeEach } from 'vitest'
import { AUTO_THEME_ID, AUTO_THEME_CONFIG_VALUE } from '../consts'

describe('themeReducer', () => {
  beforeEach(() => {
    vi.resetModules()
  })

  it.each([
    {
      configTheme: AUTO_THEME_CONFIG_VALUE,
      expected: AUTO_THEME_ID,
      description: 'is "Auto"',
    },
    { configTheme: 'Dark', expected: 'DarkTheme', description: 'is "Dark"' },
    {
      configTheme: 'NonExistent',
      expected: 'DarkTheme',
      description: 'is unrecognized',
    },
  ])(
    'returns $expected when defaultTheme config $description',
    async ({ configTheme, expected }) => {
      vi.doMock('../config', () => ({
        default: { defaultTheme: configTheme },
      }))
      const { themeReducer } = await import('./themeReducer')
      const result = themeReducer(undefined, { type: 'UNKNOWN' })
      expect(result).toBe(expected)
    },
  )
})
