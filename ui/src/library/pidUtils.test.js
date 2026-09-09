import { describe, it, expect } from 'vitest'
import { pidChanged } from './pidUtils'

describe('pidChanged', () => {
  it('is false when nothing changed', () => {
    expect(pidChanged({ pidAlbum: 'folder' }, { pidAlbum: 'folder' })).toBe(
      false,
    )
  })

  it('is false when both sides are empty in different ways', () => {
    expect(pidChanged({}, { pidAlbum: '', pidTrack: undefined })).toBe(false)
  })

  it('is true when the album spec changed', () => {
    expect(pidChanged({ pidAlbum: '' }, { pidAlbum: 'folder' })).toBe(true)
  })

  it('is true when the track spec changed', () => {
    expect(pidChanged({ pidTrack: '' }, { pidTrack: 'title' })).toBe(true)
  })

  it('ignores unrelated fields', () => {
    expect(pidChanged({ name: 'a' }, { name: 'b' })).toBe(false)
  })
})
