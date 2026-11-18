import { afterEach, vi } from 'vitest'
import config from '../config'
import { httpClient } from '../dataProvider'
import subsonic from './index'

vi.mock('../dataProvider', () => ({
  httpClient: vi.fn((target) => Promise.resolve({ target })),
  clientUniqueId: 'test-client-id',
  clientUniqueIdHeader: 'x-test-client-id',
}))

const originalLocalStorageDescriptor = Object.getOwnPropertyDescriptor(
  window,
  'localStorage',
)

const mockSubsonicLocalStorage = () => {
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

  Object.defineProperty(window, 'localStorage', {
    configurable: true,
    value: localStorageMock,
  })
}

afterEach(() => {
  if (originalLocalStorageDescriptor) {
    Object.defineProperty(
      window,
      'localStorage',
      originalLocalStorageDescriptor,
    )
  }
})

describe('getCoverArtUrl', () => {
  beforeEach(() => {
    // Mock window.location
    delete window.location
    window.location = { href: 'http://localhost:3000/app' }

    mockSubsonicLocalStorage()
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

describe('getLyricsBySongId', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockSubsonicLocalStorage()
  })

  it('requests enhanced structured lyrics for the song id', async () => {
    await subsonic.getLyricsBySongId('song-123')

    expect(httpClient).toHaveBeenCalledTimes(1)
    const requestedUrl = httpClient.mock.calls[0][0]
    expect(requestedUrl).toContain('/rest/getLyricsBySongId?')
    expect(requestedUrl).toContain('id=song-123')
    expect(requestedUrl).toContain('enhanced=true')
  })

  it('requests all sources separately from HTTP request options', async () => {
    const controller = new AbortController()
    await subsonic.getLyricsBySongId('song-123', {
      allSources: true,
      signal: controller.signal,
    })
    expect(httpClient).toHaveBeenCalledWith(
      expect.stringContaining('allSources=true'),
      { signal: controller.signal },
    )
  })

  it('rejects a failed Subsonic lyrics response with its server message', async () => {
    httpClient.mockResolvedValueOnce({
      json: {
        'subsonic-response': {
          status: 'failed',
          error: {
            code: 0,
            message: 'error getting lyrics; retry after 30s',
          },
        },
      },
    })

    await expect(subsonic.getLyricsBySongId('song-123')).rejects.toThrow(
      'error getting lyrics; retry after 30s',
    )
  })
})

describe('lyrics personal preference mutation', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockSubsonicLocalStorage()
  })

  it('POSTs a deliberate preference action', async () => {
    await subsonic.updateLyricsPreference('song-123', {
      action: 'select',
      local: 'local:v1:abc',
    })

    expect(httpClient).toHaveBeenCalledWith(
      expect.stringContaining('/rest/setLyricsPreference?'),
      { method: 'POST' },
    )
    const requestedUrl = httpClient.mock.calls[0][0]
    expect(requestedUrl).toContain('id=song-123')
    expect(requestedUrl).toContain('action=select')
    expect(requestedUrl).toContain('local=local%3Av1%3Aabc')
  })

  it('rejects a failed preference mutation with the server message', async () => {
    httpClient.mockResolvedValueOnce({
      json: {
        'subsonic-response': {
          status: 'failed',
          error: { code: 0, message: 'could not save the lyrics preference' },
        },
      },
    })
    await expect(
      subsonic.updateLyricsPreference('song-123', { action: 'auto' }),
    ).rejects.toThrow('could not save the lyrics preference')
  })
})

describe('lyrics original retrieval', () => {
  const response = ({
    body = new Blob(['document']),
    contentType = 'application/ttml+xml',
    disposition = 'attachment; filename="Artist - Song.ttml"',
    status = 200,
    json = null,
  } = {}) => ({
    ok: status >= 200 && status < 300,
    status,
    headers: {
      get: (name) =>
        ({
          'Content-Type': contentType,
          'Content-Disposition': disposition,
        })[name] || null,
    },
    json: async () => json,
    blob: async () => body,
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  beforeEach(() => {
    vi.clearAllMocks()
    mockSubsonicLocalStorage()
  })

  it('requests the variant listing for the song id', async () => {
    await subsonic.getLyricsVariants('song-123')
    expect(httpClient).toHaveBeenCalledTimes(1)
    expect(httpClient.mock.calls[0][0]).toContain('/rest/getLyricsVariants?')
    expect(httpClient.mock.calls[0][0]).toContain('id=song-123')
  })

  it('returns exact original bytes with the server filename and media type', async () => {
    const fetchMock = vi.fn().mockResolvedValue(response())
    vi.stubGlobal('fetch', fetchMock)

    const original = await subsonic.getLyricsOriginal('song-1', 'selection-1')

    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining('/rest/getLyricsOriginal?'),
      expect.objectContaining({
        headers: expect.objectContaining({
          'x-test-client-id': 'test-client-id',
        }),
      }),
    )
    expect(fetchMock.mock.calls[0][0]).toContain('selection=selection-1')
    expect(original.filename).toBe('Artist - Song.ttml')
    expect(original.mediaType).toBe('application/ttml+xml')
    expect(original.blob).toBeInstanceOf(Blob)
  })

  it('decodes RFC 5987 filenames', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        response({
          disposition: "attachment; filename*=utf-8''%E6%AD%8C%20-%20Song.qrc",
        }),
      ),
    )
    const original = await subsonic.getLyricsOriginal('song-1', 'selection-1')
    expect(original.filename).toBe('歌 - Song.qrc')
  })

  it('surfaces a JSON error instead of downloading the error envelope', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        response({
          contentType: 'application/json; charset=utf-8',
          json: {
            'subsonic-response': {
              status: 'failed',
              error: { code: 70, message: 'lyrics original unavailable' },
            },
          },
        }),
      ),
    )

    await expect(
      subsonic.getLyricsOriginal('song-1', 'selection-1'),
    ).rejects.toThrow('lyrics original unavailable')
  })

  it('rejects non-success responses without a document', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response({ status: 500 })))
    await expect(
      subsonic.getLyricsOriginal('song-1', 'selection-1'),
    ).rejects.toThrow('Lyrics original unavailable (500)')
  })
})

describe('getDiscCoverArtUrl', () => {
  beforeEach(() => {
    mockSubsonicLocalStorage()
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
    mockSubsonicLocalStorage()
  })

  it('should include username parameter', () => {
    const url = subsonic.getAvatarUrl('john')
    expect(url).toContain('getAvatar')
    expect(url).toContain('username=john')
  })
})

describe('reportPlayback', () => {
  beforeEach(() => {
    mockSubsonicLocalStorage()
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
