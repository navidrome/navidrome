import { describe, it, expect } from 'vitest'
import { artistDownloadSize } from './artist'

describe('artistDownloadSize', () => {
  it('returns the album-artist size', () => {
    expect(
      artistDownloadSize({ stats: { albumartist: { size: 1024 } } }),
    ).toEqual(1024)
  })

  it('returns undefined for a missing artist', () => {
    expect(
      artistDownloadSize({
        missing: true,
        stats: { albumartist: { size: 1024 } },
      }),
    ).toBeUndefined()
  })

  it('returns undefined when there is no album-artist content', () => {
    expect(artistDownloadSize({ stats: {} })).toBeUndefined()
    expect(artistDownloadSize(undefined)).toBeUndefined()
  })
})
