import SpotifyTheme from './spotify'
import NordTheme from './nord'
import LigeraTheme from './ligera'
import { describe, it, expect } from 'vitest'

describe('NDPlaylistDetails styles', () => {
  const themes = [
    ['SpotifyTheme', SpotifyTheme],
    ['NordTheme', NordTheme],
    ['LigeraTheme', LigeraTheme],
  ]

  it.each(themes)('%s should not set minWidth on details', (_, theme) => {
    const details = theme.overrides?.NDPlaylistDetails?.details
    expect(
      details && 'minWidth' in details ? details.minWidth : undefined,
    ).toBeUndefined()
  })
})
