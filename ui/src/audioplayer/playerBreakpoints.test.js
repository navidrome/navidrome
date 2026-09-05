import { describe, expect, it } from 'vitest'
import {
  PLAYER_DESKTOP_MEDIA_QUERY,
  PLAYER_DESKTOP_MIN_WIDTH,
  PLAYER_MOBILE_MATCH_MEDIA_QUERY,
  PLAYER_MOBILE_MAX_WIDTH,
  PLAYER_MOBILE_MEDIA_QUERY,
  PLAYER_MEDIA_QUERY_STEP,
} from './playerBreakpoints'

describe('player breakpoints', () => {
  it('keeps compact and desktop player ranges from overlapping', () => {
    expect(PLAYER_DESKTOP_MIN_WIDTH).toBe(810)
    expect(PLAYER_MOBILE_MAX_WIDTH).toBe(809.95)
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
})
