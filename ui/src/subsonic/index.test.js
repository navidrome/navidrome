import { vi } from 'vitest'
import config from '../config'
import subsonic from './index'
import { httpClient } from '../dataProvider'

vi.mock('../dataProvider', () => ({
  httpClient: vi.fn(),
  clientUniqueId: 'test-client-id',
  clientUniqueIdHeader: 'X-ND-Client-Unique-Id',
}))

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

    const url = subsonic.getCoverArtUrl(
      playlistRecord,
      config.uiCoverArtSize,
      true,
    )

    expect(url).toContain('pl-playlist-123')
    expect(url).toContain('size=600')
    expect(url).toContain('square=true')
    expect(url).toContain('_=2023-01-01T00%3A00%3A00Z')
  })

  it('should add timestamp for playlists without updatedAt', () => {
    const playlistRecord = {
      id: 'playlist-123',
      sync: true,
    }

    const url = subsonic.getCoverArtUrl(
      playlistRecord,
      config.uiCoverArtSize,
      true,
    )

    expect(url).toContain('pl-playlist-123')
    expect(url).toContain('size=600')
    expect(url).toContain('square=true')
    expect(url).not.toContain('_=')
  })

  it('should return album cover art URL for records with albumArtist', () => {
    const albumRecord = {
      id: 'album-123',
      albumArtist: 'Test Artist',
      updatedAt: '2023-01-01T00:00:00Z',
    }

    const url = subsonic.getCoverArtUrl(
      albumRecord,
      config.uiCoverArtSize,
      true,
    )

    expect(url).toContain('al-album-123')
    expect(url).toContain('size=600')
    expect(url).toContain('square=true')
  })

  it('should return media file cover art URL for records with album', () => {
    const songRecord = {
      id: 'song-123',
      album: 'Test Album',
      updatedAt: '2023-01-01T00:00:00Z',
    }

    const url = subsonic.getCoverArtUrl(songRecord, config.uiCoverArtSize, true)

    expect(url).toContain('mf-song-123')
    expect(url).toContain('size=600')
    expect(url).toContain('square=true')
  })

  it('should return artist cover art URL for other records', () => {
    const artistRecord = {
      id: 'artist-123',
      updatedAt: '2023-01-01T00:00:00Z',
    }

    const url = subsonic.getCoverArtUrl(
      artistRecord,
      config.uiCoverArtSize,
      true,
    )

    expect(url).toContain('ar-artist-123')
    expect(url).toContain('size=600')
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

  it('appends the content hash to the artwork id', () => {
    const url = subsonic.getCoverArtUrl({
      id: 'album-123',
      albumArtist: 'AA',
      imageHash: '0123456789abcdef',
    })
    expect(url).toContain('al-album-123_0123456789abcdef')
  })

  it('drops the updatedAt buster once the url is hash-versioned', () => {
    const url = subsonic.getCoverArtUrl({
      id: 'album-123',
      albumArtist: 'AA',
      updatedAt: '2023-01-01T00:00:00Z',
      imageHash: '0123456789abcdef',
    })
    expect(url).toContain('al-album-123_0123456789abcdef')
    expect(url).not.toContain('_=')
  })

  it('keeps the updatedAt buster while artwork is unresolved', () => {
    const url = subsonic.getCoverArtUrl({
      id: 'album-123',
      albumArtist: 'AA',
      updatedAt: '2023-01-01T00:00:00Z',
    })
    expect(url).toContain('al-album-123')
    expect(url).toContain('_=')
  })

  it('still builds a url for known-absent artwork so the server placeholder renders', () => {
    // Returning '' here made <img src=undefined> render as a broken icon; the server serves a
    // proper placeholder for absent art, so the client must still request it.
    const url = subsonic.getCoverArtUrl({
      id: 'album-123',
      albumArtist: 'AA',
      imageAbsent: true,
    })
    expect(url).toContain('al-album-123')
  })
})

describe('getDiscCoverArtUrl', () => {
  beforeEach(() => {
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

  it('should construct URL with dc-albumId:discNumber format, size, and cache param', () => {
    const url = subsonic.getDiscCoverArtUrl(
      'album-123',
      2,
      '2023-01-01T00:00:00Z',
      48,
    )

    expect(url).toContain('getCoverArt')
    expect(url).toContain('id=dc-album-123%3A2')
    expect(url).toContain('size=48')
    expect(url).toContain('_=2023-01-01T00%3A00%3A00Z')
  })

  it('should handle missing updatedAt', () => {
    const url = subsonic.getDiscCoverArtUrl('album-123', 1, undefined, 48)

    expect(url).toContain('id=dc-album-123%3A1')
    expect(url).toContain('size=48')
    expect(url).not.toContain('_=')
  })

  it('should handle missing size', () => {
    const url = subsonic.getDiscCoverArtUrl(
      'album-123',
      1,
      '2023-01-01T00:00:00Z',
    )

    expect(url).toContain('id=dc-album-123%3A1')
    expect(url).toContain('_=2023-01-01T00%3A00%3A00Z')
    expect(url).not.toContain('size=')
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

describe('reportPlayback', () => {
  beforeEach(() => {
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

  it('should construct reportPlayback URL with correct parameters', () => {
    const url = subsonic.url('reportPlayback', null, {
      mediaId: 'song-123',
      mediaType: 'song',
      positionMs: 5000,
      state: 'playing',
    })
    expect(url).toContain('reportPlayback')
    expect(url).toContain('mediaId=song-123')
    expect(url).toContain('mediaType=song')
    expect(url).toContain('positionMs=5000')
    expect(url).toContain('state=playing')
  })
})

describe('getTranscodeDecision', () => {
  const profile = {
    name: 'NavidromeUI',
    directPlayProfiles: [{ containers: ['mp3'], audioCodecs: ['mp3'] }],
  }
  const decision = { canDirectPlay: true, transcodeParams: 'jwt-token' }

  beforeEach(() => {
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
    httpClient.mockReset()
    httpClient.mockResolvedValue({
      json: {
        'subsonic-response': { status: 'ok', transcodeDecision: decision },
      },
    })
  })

  it('posts the browser profile to getTranscodeDecision for the song', async () => {
    await subsonic.getTranscodeDecision('song-1', profile)

    expect(httpClient).toHaveBeenCalledTimes(1)
    const [url, options] = httpClient.mock.calls[0]
    expect(url).toMatch(/^\/rest\/getTranscodeDecision\?/)
    expect(url).toContain('mediaId=song-1')
    expect(url).toContain('mediaType=song')
    expect(options.method).toBe('POST')
    expect(JSON.parse(options.body)).toEqual(profile)
  })

  it('returns the transcodeDecision from the subsonic response', async () => {
    await expect(
      subsonic.getTranscodeDecision('song-1', profile),
    ).resolves.toEqual(decision)
  })

  it('rejects when the request fails', async () => {
    httpClient.mockRejectedValue(new Error('Server Error'))

    await expect(
      subsonic.getTranscodeDecision('song-1', profile),
    ).rejects.toThrow('Server Error')
  })

  it('throws with the subsonic error code and message on a failed response', async () => {
    httpClient.mockResolvedValue({
      json: {
        'subsonic-response': {
          status: 'failed',
          error: { code: 70, message: 'not found' },
        },
      },
    })

    await expect(
      subsonic.getTranscodeDecision('song-1', profile),
    ).rejects.toThrow('getTranscodeDecision error: 70 not found')
  })
})
