import { describe, it, expect } from 'vitest'
import { createTheme } from '@material-ui/core/styles'
import palettes from './vibrantPalettes'
import { createVibrantTheme } from './vibrant'
import themes, { isThemeInPicker } from './index'

const luminance = (hex) => {
  const [r, g, b] = hex
    .replace('#', '')
    .match(/.{2}/g)
    .map((v) => parseInt(v, 16) / 255)
    .map((v) => (v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4))
  return 0.2126 * r + 0.7152 * g + 0.0722 * b
}

const contrast = (a, b) => {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x)
  return (hi + 0.05) / (lo + 0.05)
}

const variants = Object.values(palettes).flatMap((p) =>
  ['dark', 'light'].map((mode) => [`${p.name} ${mode}`, mode, p[mode]]),
)

describe('Vibrant theme palettes', () => {
  // [foreground, background] pairs that carry text
  const textPairs = [
    ['text', 'background'],
    ['text', 'paper'],
    ['text', 'elevated'],
    ['textSecondary', 'background'],
    ['textSecondary', 'paper'],
    ['accentText', 'background'],
    ['accentText', 'paper'],
    ['onAccent', 'accent'],
    ['onHighlight', 'highlight'],
  ]

  describe.each(variants)('%s', (name, mode, palette) => {
    it.each(textPairs)('%s on %s meets WCAG AA', (fg, bg) => {
      expect(contrast(palette[fg], palette[bg])).toBeGreaterThanOrEqual(4.5)
    })

    it('builds a valid MUI theme', () => {
      const theme = createTheme(createVibrantTheme(name, mode, palette))
      expect(theme.palette.type).toBe(mode)
      expect(theme.palette.primary.main).toBe(palette.accent)
    })
  })

  it('registers a dark and light theme for every palette', () => {
    const names = Object.values(themes).map((t) => t.themeName)
    Object.values(palettes).forEach((p) => {
      expect(names).toContain(`${p.name} Dark`)
      expect(names).toContain(`${p.name} Light`)
    })
  })
})

describe('theme picker', () => {
  it('only offers Vibrant themes', () => {
    const shown = Object.keys(themes).filter(isThemeInPicker)
    expect(shown).toContain('EmberDarkTheme')
    expect(shown).toContain('EmberLightTheme')
    expect(shown).not.toContain('DarkTheme')
    expect(shown).not.toContain('LightTheme')
  })
})
