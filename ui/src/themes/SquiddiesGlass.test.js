import { afterEach, describe, expect, it, vi } from 'vitest'

const PULSE = 'pulse 1.5s ease-in-out infinite alternate'
const SPIN = 'spin 6s linear infinite'

const hadAppConfig = Object.prototype.hasOwnProperty.call(
  window,
  '__APP_CONFIG__',
)
const originalAppConfig = window.__APP_CONFIG__

function restoreAppConfig() {
  if (hadAppConfig) {
    window.__APP_CONFIG__ = originalAppConfig
  } else {
    delete window.__APP_CONFIG__
  }
}

// Config and the theme are evaluated at import time, so each case reloads both.
async function loadTheme() {
  vi.resetModules()
  const { default: config } = await import('../config')
  const { default: theme } = await import('./SquiddiesGlass')
  return { config, theme }
}

function coverParent(theme) {
  return theme.overrides.NDAlbumDetails.coverParent
}

describe('Squiddies Glass cover animation', () => {
  afterEach(() => {
    restoreAppConfig()
    vi.resetModules()
  })

  it('stops both cover pseudo-elements when enableCoverAnimation is false', async () => {
    window.__APP_CONFIG__ = JSON.stringify({ enableCoverAnimation: false })
    const { config, theme } = await loadTheme()
    const cover = coverParent(theme)

    expect(config.enableCoverAnimation).toBe(false)
    expect(cover['&::before'].animation).toBe('none')
    expect(cover['&::after'].animation).toBe('none')
    expect(cover['&::after'].background).toContain('repeating-conic-gradient')
    expect(theme.overrides.NDAlbumDetails.root.animation).toBe(
      'gradientFlow 8s ease-in-out infinite',
    )
  })

  it('keeps pulse and spin when enableCoverAnimation is true', async () => {
    window.__APP_CONFIG__ = JSON.stringify({ enableCoverAnimation: true })
    const { config, theme } = await loadTheme()
    const cover = coverParent(theme)

    expect(config.enableCoverAnimation).toBe(true)
    expect(cover['&::before'].animation).toBe(PULSE)
    expect(cover['&::after'].animation).toBe(SPIN)
  })

  it('keeps cover animation when the setting is missing', async () => {
    window.__APP_CONFIG__ = JSON.stringify({ version: 'dev' })
    const { config, theme } = await loadTheme()
    const cover = coverParent(theme)

    expect(config.enableCoverAnimation).toBe(true)
    expect(cover['&::before'].animation).toBe(PULSE)
    expect(cover['&::after'].animation).toBe(SPIN)
  })

  it('keeps cover animation when window.__APP_CONFIG__ is malformed', async () => {
    window.__APP_CONFIG__ = '{not-json'
    const { config, theme } = await loadTheme()
    const cover = coverParent(theme)

    expect(config.enableCoverAnimation).toBe(true)
    expect(cover['&::before'].animation).toBe(PULSE)
    expect(cover['&::after'].animation).toBe(SPIN)
  })

  it('keeps cover animation when window.__APP_CONFIG__ is absent', async () => {
    delete window.__APP_CONFIG__
    const { config, theme } = await loadTheme()
    const cover = coverParent(theme)

    expect(config.enableCoverAnimation).toBe(true)
    expect(cover['&::before'].animation).toBe(PULSE)
    expect(cover['&::after'].animation).toBe(SPIN)
  })
})
