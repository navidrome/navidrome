import { vi } from 'vitest'
import subsonic from './index'

describe('getCoverArtUrl', () => {
  beforeEach(() => {
    // Mock window.location
    delete window.location
    window.location = { href: 'http://localhost:3000/app' }

    // Mock localStorage values required by subsonic
    const localStorageMock = {
      getItem: vi.fn((key) => {
        const values = {
          username: 'testuser',
          'subsonic-token': 'testtoken',
          'subsonic-salt': 'testsalt',
        }
        return values[key] || null
      }),
      setItem: vi.fn(),
      clear: vi.fn(),
    }
    Object.defineProperty(window, 'localStorage', { value: localStorageMock })
  })

  it('should return playlist cover art URL for records with sync property', () => {
    const playlistRecord = {
      id: 'playlist-123',
      sync: true,
      updatedAt: '2023-01-01T00:00:00Z',
    }

    const url = subsonic.getCoverArtUrl(playlistRecord, 300, true)

    expect(url).toContain('pl-playlist-123')
    expect(url).toContain('size=300')
    expect(url).toContain('square=true')
    expect(url).toContain('_=2023-01-01T00%3A00%3A00Z')
  })

  it('should add timestamp for playlists without updatedAt', () => {
    const playlistRecord = {
      id: 'playlist-123',
      sync: true,
    }

    const url = subsonic.getCoverArtUrl(playlistRecord, 300, true)

    expect(url).toContain('pl-playlist-123')
    expect(url).toContain('size=300')
    expect(url).toContain('square=true')
    expect(url).not.toContain('_=')
  })

  it('should return album cover art URL for records with albumArtist', () => {
    const albumRecord = {
      id: 'album-123',
      albumArtist: 'Test Artist',
      updatedAt: '2023-01-01T00:00:00Z',
    }

    const url = subsonic.getCoverArtUrl(albumRecord, 300, true)

    expect(url).toContain('al-album-123')
    expect(url).toContain('size=300')
    expect(url).toContain('square=true')
  })

  it('should return media file cover art URL for records with album', () => {
    const songRecord = {
      id: 'song-123',
      album: 'Test Album',
      updatedAt: '2023-01-01T00:00:00Z',
    }

    const url = subsonic.getCoverArtUrl(songRecord, 300, true)

    expect(url).toContain('mf-song-123')
    expect(url).toContain('size=300')
    expect(url).toContain('square=true')
  })

  it('should return artist cover art URL for other records', () => {
    const artistRecord = {
      id: 'artist-123',
      updatedAt: '2023-01-01T00:00:00Z',
    }

    const url = subsonic.getCoverArtUrl(artistRecord, 300, true)

    expect(url).toContain('ar-artist-123')
    expect(url).toContain('size=300')
    expect(url).toContain('square=true')
  })

  it('should handle records without updatedAt', () => {
    const record = {
      id: 'test-123',
    }

    const url = subsonic.getCoverArtUrl(record)

    expect(url).toContain('ar-test-123')
    expect(url).not.toContain('_=')
  })
})

describe('getAvatarUrl', () => {
  beforeEach(() => {
    // Mock localStorage values required by subsonic
    const localStorageMock = {
      getItem: vi.fn((key) => {
        const values = {
          username: 'testuser',
          'subsonic-token': 'testtoken',
          'subsonic-salt': 'testsalt',
        }
        return values[key] || null
      }),
    }
    Object.defineProperty(window, 'localStorage', { value: localStorageMock })
  })

  it('should include username parameter', () => {
    const url = subsonic.getAvatarUrl('john')
    expect(url).toContain('getAvatar')
    expect(url).toContain('username=john')
  })
})
