import { describe, it, expect, vi, beforeEach } from 'vitest'
import wrapperDataProvider from './wrapperDataProvider'

const { mockProvider, mockHttpClient } = vi.hoisted(() => ({
  mockProvider: {
    update: vi.fn(),
    create: vi.fn(),
    getOne: vi.fn(),
  },
  mockHttpClient: vi.fn(),
}))

vi.mock('ra-data-json-server', () => ({ default: () => mockProvider }))
vi.mock('./httpClient', () => ({ default: mockHttpClient }))

describe('wrapperDataProvider', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    mockProvider.update.mockResolvedValue({ data: { id: 'u1' } })
    mockProvider.create.mockResolvedValue({ data: { id: 'u1' } })
    mockHttpClient.mockResolvedValue({ json: [] })
  })

  describe('update user', () => {
    it('sets library associations when an admin edits a non-admin user', async () => {
      localStorage.setItem('role', 'admin')

      await wrapperDataProvider.update('user', {
        id: 'u1',
        data: { name: 'Sam', isAdmin: false, libraryIds: [1] },
      })

      expect(mockProvider.update).toHaveBeenCalledWith(
        'user',
        expect.objectContaining({ id: 'u1' }),
      )
      expect(mockHttpClient).toHaveBeenCalledWith('/api/user/u1/library', {
        method: 'PUT',
        body: JSON.stringify({ libraryIds: [1] }),
      })
    })

    it('does not call the admin-only library endpoint when a non-admin edits their own profile', async () => {
      localStorage.setItem('role', 'regular')

      await wrapperDataProvider.update('user', {
        id: 'u1',
        data: {
          name: 'Sam',
          isAdmin: false,
          libraryIds: [1],
          currentPassword: 'old',
          password: 'new',
        },
      })

      expect(mockProvider.update).toHaveBeenCalled()
      expect(mockHttpClient).not.toHaveBeenCalled()
    })

    it('does not set library associations when the edited user is an admin', async () => {
      localStorage.setItem('role', 'admin')

      await wrapperDataProvider.update('user', {
        id: 'u1',
        data: { name: 'Sam', isAdmin: true, libraryIds: [1] },
      })

      expect(mockProvider.update).toHaveBeenCalled()
      expect(mockHttpClient).not.toHaveBeenCalled()
    })

    it('strips libraryIds from the user update payload', async () => {
      localStorage.setItem('role', 'admin')

      await wrapperDataProvider.update('user', {
        id: 'u1',
        data: { name: 'Sam', isAdmin: false, libraryIds: [1] },
      })

      expect(mockProvider.update).toHaveBeenCalledWith(
        'user',
        expect.objectContaining({
          data: { name: 'Sam', isAdmin: false },
        }),
      )
    })
  })

  describe('create player', () => {
    it('returns the server record, never the plaintext API key', async () => {
      const data = { name: 'Phone', apiKey: 'nds_0123456789abcdefghijkl' }
      const saved = { id: 'p1', name: 'Phone', hasApiKey: true, userId: 'u1' }
      mockProvider.create.mockResolvedValue({ data: { ...data, id: 'p1' } })
      mockProvider.getOne.mockResolvedValue({ data: saved })

      const result = await wrapperDataProvider.create('player', { data })

      expect(mockProvider.create).toHaveBeenCalledWith('player', { data })
      expect(mockProvider.getOne).toHaveBeenCalledWith('player', { id: 'p1' })
      expect(result.data).toEqual(saved)
    })
  })

  describe('refreshMetadata', () => {
    it('posts to the album metadata refresh endpoint', () => {
      mockHttpClient.mockResolvedValue({ json: {} })
      wrapperDataProvider.refreshMetadata('album', 'al-1')
      expect(mockHttpClient).toHaveBeenCalledWith(
        expect.stringContaining('/metadata/al/al-1/refresh'),
        { method: 'POST' },
      )
    })

    it('posts to the artist metadata refresh endpoint', () => {
      mockHttpClient.mockResolvedValue({ json: {} })
      wrapperDataProvider.refreshMetadata('artist', 'ar-1')
      expect(mockHttpClient).toHaveBeenCalledWith(
        expect.stringContaining('/metadata/ar/ar-1/refresh'),
        { method: 'POST' },
      )
    })

    // react-admin rejects a custom method whose response has no `data` key, and the
    // endpoint answers 204 with no body.
    it('resolves to a react-admin shaped response', async () => {
      mockHttpClient.mockResolvedValue({
        status: 204,
        body: '',
        json: undefined,
      })
      await expect(
        wrapperDataProvider.refreshMetadata('album', 'al-1'),
      ).resolves.toEqual({ data: { id: 'al-1' } })
    })
  })

  describe('image upload', () => {
    it('posts the file as multipart form data under the image field', async () => {
      const file = new File(['img'], 'cover.png', { type: 'image/png' })

      await wrapperDataProvider.uploadImage('artist', { id: 'ar-1', file })

      const [url, options] = mockHttpClient.mock.calls[0]
      expect(url).toBe('/api/artist/ar-1/image')
      expect(options.method).toBe('POST')
      expect(options.body).toBeInstanceOf(FormData)
      expect(options.body.get('image')).toBe(file)
      // An empty Headers keeps httpClient from forcing a JSON content type over the multipart boundary
      expect([...options.headers.keys()]).toEqual([])
    })

    it('deletes the uploaded image', async () => {
      await wrapperDataProvider.deleteImage('playlist', 'pl-1')

      expect(mockHttpClient).toHaveBeenCalledWith('/api/playlist/pl-1/image', {
        method: 'DELETE',
      })
    })
  })

  describe('playlist tracks', () => {
    it('fetches the tracks of a playlist', async () => {
      const tracks = [{ id: '1', mediaFileId: 'mf-1' }]
      mockHttpClient.mockResolvedValue({ json: tracks })

      const result = await wrapperDataProvider.getPlaylistTracks('pl-1')

      expect(mockHttpClient).toHaveBeenCalledWith('/api/playlist/pl-1/tracks')
      expect(result).toEqual({ data: tracks })
    })

    it('exports a playlist as M3U and returns the raw body', async () => {
      mockHttpClient.mockResolvedValue({ body: '#EXTM3U\n', json: undefined })

      const result = await wrapperDataProvider.exportPlaylist('pl-1')

      const [url, options] = mockHttpClient.mock.calls[0]
      expect(url).toBe('/api/playlist/pl-1/tracks')
      expect(options.headers.get('Accept')).toBe('audio/x-mpegurl')
      expect(result).toEqual({ data: '#EXTM3U\n' })
    })
  })

  describe('scrobbler links', () => {
    it('reads the Last.fm link status', async () => {
      mockHttpClient.mockResolvedValue({ json: { status: true, apiKey: 'k' } })

      const result = await wrapperDataProvider.getLastfmLink()

      expect(mockHttpClient).toHaveBeenCalledWith('/api/lastfm/link')
      expect(result).toEqual({ data: { status: true, apiKey: 'k' } })
    })

    it('unlinks Last.fm', async () => {
      await wrapperDataProvider.unlinkLastfm()

      expect(mockHttpClient).toHaveBeenCalledWith('/api/lastfm/link', {
        method: 'DELETE',
      })
    })

    it('reads the ListenBrainz link status', async () => {
      mockHttpClient.mockResolvedValue({ json: { status: false } })

      const result = await wrapperDataProvider.getListenBrainzLink()

      expect(mockHttpClient).toHaveBeenCalledWith('/api/listenbrainz/link')
      expect(result).toEqual({ data: { status: false } })
    })

    it('links ListenBrainz with the given token', async () => {
      mockHttpClient.mockResolvedValue({ json: { user: 'sam' } })

      const result = await wrapperDataProvider.linkListenBrainz('tok')

      expect(mockHttpClient).toHaveBeenCalledWith('/api/listenbrainz/link', {
        method: 'PUT',
        body: JSON.stringify({ token: 'tok' }),
      })
      expect(result).toEqual({ data: { user: 'sam' } })
    })

    it('unlinks ListenBrainz', async () => {
      await wrapperDataProvider.unlinkListenBrainz()

      expect(mockHttpClient).toHaveBeenCalledWith('/api/listenbrainz/link', {
        method: 'DELETE',
      })
    })
  })

  describe('rescanPlugins', () => {
    it('posts to the plugin rescan endpoint', async () => {
      await wrapperDataProvider.rescanPlugins()

      expect(mockHttpClient).toHaveBeenCalledWith('/api/plugin/rescan', {
        method: 'POST',
      })
    })
  })
})
