import {
  buildCastQueueItems,
  buildCastQueueWindow,
  canDirectPlayOnCast,
  canLoadCastSdk,
  formatCastTime,
  isCredentialFreeCastUrl,
  planCastQueueSync,
  resolveCastCoverUrl,
  resolveCastStream,
  toAbsoluteUrl,
} from './castUtils'

describe('castUtils', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('canLoadCastSdk', () => {
    it('returns false when enableExternalServices is false', () => {
      expect(
        canLoadCastSdk(
          { enableExternalServices: false },
          { chrome: {}, navigator: { userAgent: 'Mozilla/5.0 Chrome/130.0' } },
        ),
      ).toBe(false)
    })

    it('returns false on non-Chromium browsers', () => {
      expect(
        canLoadCastSdk(
          { enableExternalServices: true },
          {
            navigator: {
              userAgent:
                'Mozilla/5.0 (X11; Linux x86_64; rv:132.0) Gecko/20100101 Firefox/132.0',
            },
          },
        ),
      ).toBe(false)
    })

    it('returns true on Chromium browsers when external services are enabled', () => {
      expect(
        canLoadCastSdk(
          { enableExternalServices: true },
          { chrome: {}, navigator: { userAgent: 'Mozilla/5.0 Chrome/130.0' } },
        ),
      ).toBe(true)
    })
  })

  describe('toAbsoluteUrl', () => {
    it('resolves relative paths against the provided origin', () => {
      expect(
        toAbsoluteUrl(
          '/app/rest/stream?id=track-1',
          'https://music.example.com',
        ),
      ).toBe('https://music.example.com/app/rest/stream?id=track-1')
    })

    it('preserves already absolute URLs', () => {
      expect(
        toAbsoluteUrl(
          'https://radio.example.com/live.mp3',
          'https://music.example.com',
        ),
      ).toBe('https://radio.example.com/live.mp3')
    })

    it('returns empty string for falsy values', () => {
      expect(toAbsoluteUrl('', 'https://music.example.com')).toBe('')
      expect(toAbsoluteUrl(null, 'https://music.example.com')).toBe('')
    })
  })

  describe('canDirectPlayOnCast', () => {
    it('allows standard 44.1kHz/96kHz stereo FLAC, MP3, AAC/M4A, and Opus', () => {
      expect(
        canDirectPlayOnCast({
          suffix: 'flac',
          codec: 'flac',
          sampleRate: 96000,
          channels: 2,
        }),
      ).toBe(true)
      expect(canDirectPlayOnCast({ suffix: 'mp3' })).toBe(true)
      expect(canDirectPlayOnCast({ suffix: 'm4a', codec: 'aac' })).toBe(true)
      expect(canDirectPlayOnCast({ suffix: 'opus', codec: 'opus' })).toBe(true)
    })

    it('rejects ALAC inside .m4a, WMA, DSD, >96kHz FLAC, and >2 channels', () => {
      expect(canDirectPlayOnCast({ suffix: 'm4a', codec: 'alac' })).toBe(false)
      expect(canDirectPlayOnCast({ suffix: 'wma', codec: 'wmav2' })).toBe(false)
      expect(canDirectPlayOnCast({ suffix: 'dsf', codec: 'dsd_lsbf' })).toBe(
        false,
      )
      expect(
        canDirectPlayOnCast({
          suffix: 'flac',
          codec: 'flac',
          sampleRate: 192000,
          channels: 2,
        }),
      ).toBe(false)
      expect(
        canDirectPlayOnCast({
          suffix: 'flac',
          codec: 'flac',
          sampleRate: 48000,
          channels: 6,
        }),
      ).toBe(false)
    })
  })

  describe('resolveCastStream & resolveCastCoverUrl', () => {
    const origin = 'https://music.example.com'

    it('uses scoped /share/cast token URL for standard FLAC tracks without Subsonic credentials', () => {
      const tokenMap = {
        'song:t-flac:raw:0': {
          streamToken: 'jwt-flac-raw',
          coverToken: 'jwt-cover-flac',
        },
      }
      const item = {
        trackId: 't-flac',
        cover: '/app/rest/getCoverArt?u=alice&t=secret&s=salt&id=mf-t-flac',
        song: { id: 't-flac', suffix: 'flac', sampleRate: 44100 },
      }
      const res = resolveCastStream(item, origin, tokenMap)
      expect(res).toEqual({
        url: 'https://music.example.com/share/cast/jwt-flac-raw',
        contentType: 'audio/flac',
        isLive: false,
      })
      expect(resolveCastCoverUrl(item, origin, tokenMap)).toBe(
        'https://music.example.com/share/img/jwt-cover-flac?size=300',
      )
    })

    it('uses 320kbps MP3 transcode token profile for ALAC or 192kHz files', () => {
      const tokenMap = {
        'song:t-alac:mp3:320': {
          streamToken: 'jwt-alac-mp3',
          coverToken: 'jwt-cover-alac',
        },
      }
      const res = resolveCastStream(
        {
          trackId: 't-alac',
          song: { id: 't-alac', suffix: 'm4a', codec: 'alac' },
        },
        origin,
        tokenMap,
      )
      expect(res).toEqual({
        url: 'https://music.example.com/share/cast/jwt-alac-mp3',
        contentType: 'audio/mpeg',
        isLive: false,
      })
    })

    it('never leaks Subsonic u/t/s URLs when a token is absent or for radio uploaded images', () => {
      const songWithoutToken = {
        trackId: 't-missing',
        cover: '/rest/getCoverArt?u=alice&t=md5hash&s=salt&id=mf-t-missing',
        song: { id: 't-missing', suffix: 'mp3' },
      }
      expect(resolveCastStream(songWithoutToken, origin, {}).url).toBe('')
      expect(resolveCastCoverUrl(songWithoutToken, origin, {})).toBe('')

      const radioWithSubsonicCover = {
        trackId: 'r-1',
        isRadio: true,
        cover: '/rest/getCoverArt?u=alice&t=md5hash&s=salt&id=ra-r-1',
        song: {
          id: 'r-1',
          streamUrl: 'https://icecast.example.org/stream.aac',
        },
      }
      expect(resolveCastCoverUrl(radioWithSubsonicCover, origin, {})).toBe('')
      expect(
        resolveCastCoverUrl(radioWithSubsonicCover, origin, {
          'radio:r-1': { coverToken: 'jwt-radio-cover' },
        }),
      ).toBe(
        'https://music.example.com/share/img/jwt-radio-cover?size=300&square=true',
      )
      expect(
        isCredentialFreeCastUrl(
          '/rest/stream?u=alice&t=abc&s=xyz&id=1',
          origin,
        ),
      ).toBe(false)
    })

    it('handles live internet radio streams and external favicons', () => {
      const radioItem = {
        trackId: 'r-1',
        isRadio: true,
        cover: 'https://icecast.example.org/favicon.ico',
        song: { streamUrl: 'https://icecast.example.org/stream.aac' },
      }
      const res = resolveCastStream(radioItem, origin, {})
      expect(res).toEqual({
        url: 'https://icecast.example.org/stream.aac',
        contentType: 'audio/mpeg',
        isLive: true,
      })
      expect(resolveCastCoverUrl(radioItem, origin, {})).toBe(
        'https://icecast.example.org/favicon.ico',
      )
    })
  })

  describe('buildCastQueueWindow', () => {
    it('returns the full queue when smaller than maxItems', () => {
      const queue = Array.from({ length: 10 }, (_, idx) => ({
        uuid: `u-${idx}`,
      }))
      const win = buildCastQueueWindow(queue, 4, 40)
      expect(win.items).toHaveLength(10)
      expect(win.startOffset).toBe(4)
      expect(win.windowStartIndex).toBe(0)
      expect(win.windowEndIndex).toBe(10)
    })

    it('windows large playlists to stay well below the 64KB Cast IPC limit', () => {
      const queue = Array.from({ length: 200 }, (_, idx) => ({
        uuid: `u-${idx}`,
      }))
      const win = buildCastQueueWindow(queue, 80, 40)
      expect(win.items).toHaveLength(40)
      expect(win.items[win.startOffset].uuid).toBe('u-80')
      expect(win.windowStartIndex).toBe(75)
      expect(win.windowEndIndex).toBe(115)
    })
  })

  describe('planCastQueueSync', () => {
    const mk = (uuid) => ({ uuid, trackId: `t-${uuid}` })

    it('returns STOP when nextQueue is empty', () => {
      expect(
        planCastQueueSync(['u-1', 'u-2'], [], { activeUuid: 'u-1' }),
      ).toEqual({ action: 'STOP' })
    })

    it('returns LOAD when no previous Cast window is loaded', () => {
      const next = [mk('u-1'), mk('u-2')]
      expect(
        planCastQueueSync([], next, { explicitPlayIndex: 1 }),
      ).toMatchObject({
        action: 'LOAD',
        targetIndex: 1,
        startSeconds: 0,
      })
    })

    it('returns INSERT with insertBeforeUuid for "Play Next" middle insertion without restarting active track', () => {
      const next = [mk('u-1'), mk('u-next'), mk('u-2'), mk('u-3')]
      const plan = planCastQueueSync(['u-1', 'u-2', 'u-3'], next, {
        activeUuid: 'u-1',
        savedPlayIndex: 0,
      })

      expect(plan.action).toBe('INSERT')
      expect(plan.insertedItems.map((i) => i.uuid)).toEqual(['u-next'])
      expect(plan.insertBeforeUuid).toBe('u-2')
      expect(plan.activeIndex).toBe(0)
      expect(plan.nextWindowUuids).toEqual(['u-1', 'u-next', 'u-2', 'u-3'])
    })

    it('returns INSERT with insertBeforeUuid: null for "Add to Queue" end append', () => {
      const next = [mk('u-1'), mk('u-2'), mk('u-add-1'), mk('u-add-2')]
      const plan = planCastQueueSync(['u-1', 'u-2'], next, {
        activeUuid: 'u-1',
        savedPlayIndex: 0,
      })

      expect(plan.action).toBe('INSERT')
      expect(plan.insertedItems.map((i) => i.uuid)).toEqual([
        'u-add-1',
        'u-add-2',
      ])
      expect(plan.insertBeforeUuid).toBeNull()
      expect(plan.activeIndex).toBe(0)
    })

    it('returns REMOVE when non-active tracks are deleted from the queue drawer', () => {
      const next = [mk('u-1'), mk('u-3')]
      const plan = planCastQueueSync(['u-1', 'u-2', 'u-3'], next, {
        activeUuid: 'u-1',
        savedPlayIndex: 0,
      })

      expect(plan.action).toBe('REMOVE')
      expect(plan.removedUuids).toEqual(['u-2'])
      expect(plan.nextWindowUuids).toEqual(['u-1', 'u-3'])
    })

    it('returns REORDER when tracks are reordered in the queue drawer', () => {
      const next = [mk('u-1'), mk('u-3'), mk('u-2')]
      const plan = planCastQueueSync(['u-1', 'u-2', 'u-3'], next, {
        activeUuid: 'u-1',
        savedPlayIndex: 0,
      })

      expect(plan.action).toBe('REORDER')
      expect(plan.orderedUuids).toEqual(['u-1', 'u-3', 'u-2'])
    })

    it('returns JUMP when explicitPlayIndex selects another track in the same queue', () => {
      const next = [mk('u-1'), mk('u-2'), mk('u-3')]
      const plan = planCastQueueSync(['u-1', 'u-2', 'u-3'], next, {
        activeUuid: 'u-1',
        explicitPlayIndex: 2,
      })

      expect(plan).toMatchObject({
        action: 'JUMP',
        targetIndex: 2,
        targetUuid: 'u-3',
      })
    })

    it('returns NONE on PLAYER_REFRESH_QUEUE when UUIDs and active track are unchanged', () => {
      const next = [mk('u-1'), mk('u-2')]
      const plan = planCastQueueSync(['u-1', 'u-2'], next, {
        activeUuid: 'u-1',
        explicitPlayIndex: 0,
        savedPlayIndex: 0,
      })

      expect(plan.action).toBe('NONE')
    })

    it('returns LOAD at 0s when a brand-new album replaces the queue', () => {
      const next = [mk('new-1'), mk('new-2')]
      const plan = planCastQueueSync(['u-1', 'u-2'], next, {
        activeUuid: 'u-1',
        explicitPlayIndex: 0,
      })

      expect(plan).toEqual({
        action: 'LOAD',
        targetIndex: 0,
        startSeconds: 0,
      })
    })

    it('returns NONE during auto-advance across a >40-track queue when the loaded window is unchanged', () => {
      const queue = Array.from({ length: 90 }, (_, i) => mk(`u-${i}`))
      const prevWindow = queue.slice(0, 40).map((it) => it.uuid)
      const plan = planCastQueueSync(prevWindow, queue, {
        activeUuid: 'u-25',
        savedPlayIndex: 25,
      })

      expect(plan.action).toBe('NONE')
    })

    it('returns INSERT instead of reloading when "Play Next" is triggered inside a >40-track queue', () => {
      const base = Array.from({ length: 90 }, (_, i) => mk(`u-${i}`))
      const prevWindow = base.slice(0, 40).map((it) => it.uuid)
      const next = [...base.slice(0, 11), mk('u-next'), ...base.slice(11)]
      const plan = planCastQueueSync(prevWindow, next, {
        activeUuid: 'u-10',
        savedPlayIndex: 10,
      })

      expect(plan.action).toBe('INSERT')
      expect(plan.insertedItems.map((i) => i.uuid)).toEqual(['u-next'])
      expect(plan.insertBeforeUuid).toBe('u-11')
      expect(plan.nextWindowUuids).toHaveLength(41)
    })
  })

  describe('buildCastQueueItems', () => {
    it('builds QueueItem entries with MusicTrackMediaMetadata and preloadTime', () => {
      const fakeCast = {
        Image: function Image(url) {
          this.url = url
        },
        media: {
          StreamType: { BUFFERED: 'BUFFERED', LIVE: 'LIVE' },
          MetadataType: { MUSIC_TRACK: 3 },
          MediaInfo: function MediaInfo(contentId, contentType) {
            this.contentId = contentId
            this.contentType = contentType
          },
          MusicTrackMediaMetadata: function MusicTrackMediaMetadata() {},
          QueueItem: function QueueItem(mediaInfo) {
            this.media = mediaInfo
          },
        },
      }

      const items = buildCastQueueItems(
        fakeCast,
        [
          {
            trackId: 't-1',
            uuid: 'u-1',
            duration: 215,
            cover: '/app/rest/getCoverArt?u=alice&t=secret&s=salt&id=mf-t-1',
            song: {
              id: 't-1',
              title: 'Nude',
              artist: 'Radiohead',
              album: 'In Rainbows',
              albumId: 'alb-1',
              suffix: 'flac',
              trackNumber: 3,
              discNumber: 1,
            },
          },
        ],
        'https://music.example.com',
        {
          'song:t-1:raw:0': {
            streamToken: 'tok-stream-t1',
            coverToken: 'tok-cover-t1',
          },
        },
      )

      expect(items).toHaveLength(1)
      expect(items[0].autoplay).toBe(true)
      expect(items[0].preloadTime).toBe(10)
      expect(items[0].media.contentId).toBe(
        'https://music.example.com/share/cast/tok-stream-t1',
      )
      expect(items[0].media.contentType).toBe('audio/flac')
      expect(items[0].media.metadata.title).toBe('Nude')
      expect(items[0].media.metadata.images[0].url).toBe(
        'https://music.example.com/share/img/tok-cover-t1?size=300',
      )
      expect(items[0].media.customData).toEqual({
        trackId: 't-1',
        uuid: 'u-1',
        isRadio: false,
        albumId: 'alb-1',
      })
    })
  })

  describe('formatCastTime', () => {
    it('formats mm:ss and h:mm:ss cleanly', () => {
      expect(formatCastTime(0)).toBe('0:00')
      expect(formatCastTime(65)).toBe('1:05')
      expect(formatCastTime(3661)).toBe('1:01:01')
      expect(formatCastTime(NaN)).toBe('0:00')
    })
  })
})
