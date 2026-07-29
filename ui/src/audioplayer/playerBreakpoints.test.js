import { describe, expect, it } from 'vitest'
import {
  LYRICS_SIDEBAR_DESKTOP_MEDIA_QUERY,
  LYRICS_SIDEBAR_DESKTOP_MIN_WIDTH,
  PLAYER_DESKTOP_MEDIA_QUERY,
  PLAYER_DESKTOP_MIN_WIDTH,
  PLAYER_MOBILE_MATCH_MEDIA_QUERY,
  PLAYER_MOBILE_MAX_WIDTH,
  PLAYER_MOBILE_MEDIA_QUERY,
  PLAYER_MEDIA_QUERY_STEP,
} from './playerBreakpoints'

describe('player breakpoints', () => {
  it('keeps compact and desktop player ranges from overlapping', () => {
    expect(PLAYER_DESKTOP_MIN_WIDTH - PLAYER_MOBILE_MAX_WIDTH).toBeCloseTo(
      PLAYER_MEDIA_QUERY_STEP,
    )
    expect(PLAYER_DESKTOP_MEDIA_QUERY).toBe(
      `(min-width:${PLAYER_DESKTOP_MIN_WIDTH}px)`,
    )
    expect(PLAYER_MOBILE_MEDIA_QUERY).toBe(
      `@media screen and (max-width:${PLAYER_MOBILE_MAX_WIDTH}px)`,
    )
    expect(PLAYER_MOBILE_MATCH_MEDIA_QUERY).toBe(
      `(max-width:${PLAYER_MOBILE_MAX_WIDTH}px)`,
    )
  })

  it('keeps tablet layouts on the desktop player and lyrics sidebar', () => {
    expect(PLAYER_DESKTOP_MIN_WIDTH).toBe(810)
    expect(PLAYER_MOBILE_MAX_WIDTH).toBe(809.95)
    expect(LYRICS_SIDEBAR_DESKTOP_MIN_WIDTH).toBe(810)
    expect(LYRICS_SIDEBAR_DESKTOP_MEDIA_QUERY).toBe(
      `(min-width:${LYRICS_SIDEBAR_DESKTOP_MIN_WIDTH}px)`,
    )
    expect(LYRICS_SIDEBAR_DESKTOP_MIN_WIDTH).toBe(PLAYER_DESKTOP_MIN_WIDTH)
  })
})
