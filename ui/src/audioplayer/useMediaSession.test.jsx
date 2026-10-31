import React from 'react'
import { render, act, cleanup } from '@testing-library/react'
import subsonic from '../subsonic'
import {
  buildMediaSessionArtwork,
  resolveActiveMediaItem,
  setMediaSessionProperty,
  toAbsoluteArtworkUrl,
  useMediaSession,
} from './useMediaSession'

vi.mock('../subsonic', () => ({
  default: {
    getCoverArtUrl: vi.fn(
      (record, size) => `/app/rest/getCoverArt?id=mf-${record.id}&size=${size}`,
    ),
  },
}))

const Harness = ({ audioInstance, playerState, onReady }) => {
  const api = useMediaSession(audioInstance, playerState)
  React.useEffect(() => {
    onReady?.(api)
  }, [api, onReady])
  return null
}

describe('useMediaSession', () => {
  const origMediaSessionDescriptor = Object.getOwnPropertyDescriptor(
    window.navigator,
    'mediaSession',
  )
  const origMediaMetadata = window.MediaMetadata

  const createFakeMediaSessionEnv = ({
    withExtensionOverride = false,
  } = {}) => {
    const handlers = {}
    const nativeWrites = { metadata: [], playbackState: [] }
    const extWrites = { metadata: [], playbackState: [] }

    let nativeMetadata = null
    let nativePlaybackState = 'none'
    let extMetadata = null
    let extPlaybackState = 'none'

    function FakeMediaSessionProto() {}
    Object.defineProperties(FakeMediaSessionProto.prototype, {
      metadata: {
        configurable: true,
        get() {
          return nativeMetadata
        },
        set(val) {
          nativeMetadata = val
          nativeWrites.metadata.push(val)
        },
      },
      playbackState: {
        configurable: true,
        get() {
          return nativePlaybackState
        },
        set(val) {
          nativePlaybackState = val
          nativeWrites.playbackState.push(val)
        },
      },
    })

    const ms = new FakeMediaSessionProto()
    ms.setActionHandler = vi.fn((action, fn) => {
      handlers[action] = fn
    })
    ms.setPositionState = vi.fn()

    const installExtensionShim = () => {
      extMetadata = null
      extPlaybackState = 'none'
      Object.defineProperty(ms, 'metadata', {
        configurable: true,
        get: () => extMetadata,
        set: (val) => {
          extMetadata = val
          extWrites.metadata.push(val)
        },
      })
      Object.defineProperty(ms, 'playbackState', {
        configurable: true,
        get: () => extPlaybackState,
        set: (val) => {
          extPlaybackState = val
          extWrites.playbackState.push(val)
        },
      })
    }

    if (withExtensionOverride) {
      installExtensionShim()
    }

    Object.defineProperty(window.navigator, 'mediaSession', {
      configurable: true,
      value: ms,
    })

    window.MediaMetadata = function MediaMetadata(init) {
      this.title = init.title || ''
      this.artist = init.artist || ''
      this.album = init.album || ''
      this.artwork = init.artwork || []
    }

    return {
      ms,
      handlers,
      nativeWrites,
      extWrites,
      getNativeMetadata: () => nativeMetadata,
      getExtMetadata: () => extMetadata,
      installExtensionShim,
    }
  }

  afterEach(() => {
    cleanup()
    if (origMediaSessionDescriptor) {
      Object.defineProperty(
        window.navigator,
        'mediaSession',
        origMediaSessionDescriptor,
      )
    } else {
      delete window.navigator.mediaSession
    }
    window.MediaMetadata = origMediaMetadata
    vi.clearAllMocks()
  })

  const songItem1 = {
    trackId: 'song-1',
    uuid: 'uuid-1',
    name: 'Duvet',
    singer: 'bôa',
    cover: '/app/rest/getCoverArt?id=mf-song-1&size=300',
    isRadio: false,
    song: {
      id: 'song-1',
      title: 'Duvet',
      artist: 'bôa',
      album: 'Twilight',
      imageHash: 'abc123',
      duration: 203,
    },
  }

  const songItem2 = {
    trackId: 'song-2',
    uuid: 'uuid-2',
    name: 'Weird Fishes',
    singer: 'Radiohead',
    cover: '/app/rest/getCoverArt?id=mf-song-2&size=300',
    isRadio: false,
    song: {
      id: 'song-2',
      title: 'Weird Fishes',
      artist: 'Radiohead',
      album: 'In Rainbows',
      duration: 318,
    },
  }

  describe('pure helpers', () => {
    it('converts relative artwork URLs into absolute URLs', () => {
      expect(
        toAbsoluteArtworkUrl(
          '/app/rest/getCoverArt?id=mf-1',
          'https://music.example.com/app/#/album',
        ),
      ).toBe('https://music.example.com/app/rest/getCoverArt?id=mf-1')
    })

    it('builds multi-size absolute artwork entries without hardcoded image/png MIME types', () => {
      const artwork = buildMediaSessionArtwork(
        songItem1,
        'https://music.example.com/app/',
      )
      expect(artwork).toEqual([
        {
          src: 'https://music.example.com/app/rest/getCoverArt?id=mf-song-1&size=96',
          sizes: '96x96',
        },
        {
          src: 'https://music.example.com/app/rest/getCoverArt?id=mf-song-1&size=192',
          sizes: '192x192',
        },
        {
          src: 'https://music.example.com/app/rest/getCoverArt?id=mf-song-1&size=300',
          sizes: '300x300',
        },
        {
          src: 'https://music.example.com/app/rest/getCoverArt?id=mf-song-1&size=512',
          sizes: '512x512',
        },
      ])
      expect(subsonic.getCoverArtUrl).toHaveBeenCalledWith(
        expect.objectContaining({
          id: 'song-1',
          album: 'Twilight',
          imageHash: 'abc123',
        }),
        512,
      )
      artwork.forEach((entry) => {
        expect(entry).not.toHaveProperty('type')
      })
    })

    it('resolves the active item immediately from playIndex before async stream resolution populates current', () => {
      const state = {
        queue: [songItem1, songItem2],
        playIndex: 0,
        current: {},
        savedPlayIndex: 0,
      }
      expect(resolveActiveMediaItem(state)).toBe(songItem1)
    })

    it('forwards property updates to both an extension own-property setter and MediaSession.prototype', () => {
      const env = createFakeMediaSessionEnv({ withExtensionOverride: true })
      const meta = new window.MediaMetadata({
        title: 'Duvet',
        artist: 'bôa',
        album: 'Twilight',
      })
      setMediaSessionProperty(env.ms, 'metadata', meta)

      expect(env.getExtMetadata()).toBe(meta)
      expect(env.getNativeMetadata()).toBe(meta)
    })
  })

  describe('hook behavior', () => {
    it('populates real album and cover artwork synchronously when a queue starts and deduplicates identical updates', () => {
      const env = createFakeMediaSessionEnv()
      const audio = document.createElement('audio')
      let hookApi = null

      const initialState = {
        queue: [songItem1, songItem2],
        playIndex: 0,
        current: {},
        savedPlayIndex: 0,
      }

      const { rerender } = render(
        <Harness
          audioInstance={audio}
          playerState={initialState}
          onReady={(api) => {
            hookApi = api
          }}
        />,
      )

      expect(env.ms.metadata).toBeTruthy()
      expect(env.ms.metadata.title).toBe('Duvet')
      expect(env.ms.metadata.artist).toBe('bôa')
      expect(env.ms.metadata.album).toBe('Twilight')
      expect(env.ms.metadata.artwork).toHaveLength(4)
      expect(env.nativeWrites.metadata).toHaveLength(1)

      // When async stream resolution finishes and currentPlaying(songItem1) updates Redux,
      // metadata must not be redundantly reassigned
      rerender(
        <Harness
          audioInstance={audio}
          playerState={{
            queue: [songItem1, songItem2],
            playIndex: undefined,
            current: songItem1,
            savedPlayIndex: 0,
          }}
          onReady={(api) => {
            hookApi = api
          }}
        />,
      )
      act(() => {
        hookApi.syncTrackMetadata(songItem1)
      })
      expect(env.nativeWrites.metadata).toHaveLength(1)

      // Switching tracks via syncTrackMetadata updates metadata immediately before onAudioPlay
      act(() => {
        hookApi.syncTrackMetadata(songItem2)
      })
      expect(env.ms.metadata.title).toBe('Weird Fishes')
      expect(env.ms.metadata.album).toBe('In Rainbows')
      expect(env.nativeWrites.metadata).toHaveLength(2)
    })

    it('re-populates metadata on play if a browser extension injected an own-property shim that reset metadata to null', () => {
      const env = createFakeMediaSessionEnv({ withExtensionOverride: false })
      const audio = document.createElement('audio')
      let hookApi = null

      render(
        <Harness
          audioInstance={audio}
          playerState={{
            queue: [songItem1],
            playIndex: undefined,
            current: songItem1,
            savedPlayIndex: 0,
          }}
          onReady={(api) => {
            hookApi = api
          }}
        />,
      )

      expect(env.getNativeMetadata()?.title).toBe('Duvet')

      // Extension loads late and shadows navigator.mediaSession.metadata with null
      env.installExtensionShim()
      expect(env.ms.metadata).toBeNull()

      act(() => {
        hookApi.syncTrackMetadata(songItem1)
      })

      expect(env.getExtMetadata()?.title).toBe('Duvet')
      expect(env.getExtMetadata()?.album).toBe('Twilight')
      expect(env.getNativeMetadata()?.title).toBe('Duvet')
    })

    it('guards play/pause actions against accidental state inversion and clears metadata for radio or empty queues', () => {
      const env = createFakeMediaSessionEnv()
      const audio = document.createElement('audio')
      let isPaused = true
      Object.defineProperty(audio, 'paused', {
        configurable: true,
        get: () => isPaused,
      })
      Object.defineProperty(audio, 'duration', {
        configurable: true,
        value: 200,
      })
      audio.currentTime = 30
      audio.togglePlay = vi.fn(() => {
        isPaused = !isPaused
      })
      audio.playNext = vi.fn()
      audio.playPrev = vi.fn()

      const { rerender } = render(
        <Harness
          audioInstance={audio}
          playerState={{
            queue: [songItem1],
            current: songItem1,
            savedPlayIndex: 0,
          }}
        />,
      )

      // Receiving an OS "pause" action while already paused must NOT resume playback
      env.handlers.pause()
      expect(audio.togglePlay).not.toHaveBeenCalled()

      // Receiving "play" while paused starts playback
      env.handlers.play()
      expect(audio.togglePlay).toHaveBeenCalledTimes(1)

      // Receiving "play" while already playing must NOT pause playback
      env.handlers.play()
      expect(audio.togglePlay).toHaveBeenCalledTimes(1)

      // Receiving "pause" while playing pauses playback
      env.handlers.pause()
      expect(audio.togglePlay).toHaveBeenCalledTimes(2)

      env.handlers.nexttrack()
      expect(audio.playNext).toHaveBeenCalledTimes(1)
      env.handlers.previoustrack()
      expect(audio.playPrev).toHaveBeenCalledTimes(1)

      env.handlers.seekforward({ seekOffset: 15 })
      expect(audio.currentTime).toBe(45)
      expect(env.ms.setPositionState).toHaveBeenLastCalledWith({
        duration: 200,
        playbackRate: 1,
        position: 45,
      })

      // Switching to a radio station clears MediaSession metadata and action handlers
      rerender(
        <Harness
          audioInstance={audio}
          playerState={{
            queue: [{ uuid: 'radio-1', name: 'NTS', isRadio: true }],
            current: { uuid: 'radio-1', name: 'NTS', isRadio: true },
            savedPlayIndex: 0,
          }}
        />,
      )
      expect(env.ms.metadata).toBeNull()
      expect(env.ms.playbackState).toBe('none')
      expect(env.handlers.play).toBeNull()
    })
  })
})
