import React from 'react'
import { MemoryRouter } from 'react-router-dom'
import { Provider } from 'react-redux'
import { combineReducers, legacy_createStore as createStore } from 'redux'
import { render, screen, fireEvent, act, cleanup } from '@testing-library/react'
import { createTheme } from '@material-ui/core/styles'
import config from '../../config'
import { httpClient } from '../../dataProvider'
import subsonic from '../../subsonic'
import { decisionService } from '../../transcode'
import { playerReducer } from '../../reducers/playerReducer'
import { addTracks, playNext, playTracks, shuffleTracks } from '../../actions'
import { Player } from '../Player'
import CastProvider from './CastProvider'
import CastButton from './CastButton'

vi.mock('sortablejs', () => {
  function Sortable() {
    this.destroy = vi.fn()
  }
  Sortable.mount = vi.fn()
  function Swap() {}
  return { default: Sortable, Swap }
})

vi.mock(
  'navidrome-music-player',
  () => import('navidrome-music-player/es/index.js'),
)

vi.mock('react-dnd', () => ({
  useDrag: () => [null, () => {}],
}))

vi.mock('react-admin', () => ({
  createMuiTheme: (t) => createTheme(t),
  useAuthState: () => ({ authenticated: true }),
  useDataProvider: () => ({ getOne: vi.fn(() => Promise.resolve({})) }),
  useTranslate: () => (_key, opts) => {
    if (opts?._) return opts._
    if (_key === 'player.playModeText') {
      return {
        order: 'In order',
        orderLoop: 'Repeat',
        singleLoop: 'Repeat One',
        shufflePlay: 'Shuffle',
      }
    }
    return _key
  },
  useGetOne: () => ({ data: { id: 'song-1', starred: false }, loading: false }),
}))

vi.mock('../../themes/useCurrentTheme', () => ({
  default: () => ({
    palette: { type: 'dark', primary: { main: '#31c27c', dark: '#28a066' } },
    player: { theme: 'dark' },
  }),
}))

vi.mock('../../transcode', () => ({
  detectBrowserProfile: () => null,
  decisionService: {
    getProfile: () => null,
    getCachedDecision: () => null,
    setProfile: vi.fn(),
    resolveStreamUrl: vi.fn((id) =>
      Promise.resolve(`/app/rest/stream?id=${id}`),
    ),
    prefetchDecisions: vi.fn(),
    invalidateAll: vi.fn(),
  },
}))

vi.mock('../../common', () => ({
  useInterval: () => {},
  LoveButton: ({ className, disabled }) => (
    <button data-testid="love-button" className={className} disabled={disabled}>
      Love
    </button>
  ),
  useToggleLove: () => [vi.fn(), false],
  QualityInfo: () => <span data-testid="quality-info" />,
}))

vi.mock('../../dataProvider', () => ({
  httpClient: vi.fn((_url, options) => {
    const parsed = options?.body ? JSON.parse(options.body) : { items: [] }
    const tokens = {}
    const expiresAt = Math.floor(Date.now() / 1000) + 14400
    ;(parsed.items || []).forEach((it) => {
      if (it.isRadio) {
        tokens[it.key] = {
          coverToken: `radio-cover-jwt-${it.id}`,
          expiresAt,
        }
      } else {
        tokens[it.key] = {
          streamToken: `cast-stream-jwt-${it.id}-${it.format || 'raw'}`,
          coverToken: `cast-cover-jwt-${it.id}`,
          expiresAt,
        }
      }
    })
    return Promise.resolve({ json: { tokens } })
  }),
}))

vi.mock('../../subsonic', () => ({
  default: {
    streamUrl: vi.fn((id, opts) => {
      const fmt = opts?.format ? `&format=${opts.format}` : ''
      return `/app/rest/stream?id=${id}${fmt}`
    }),
    getCoverArtUrl: vi.fn((item) => `/app/rest/getCoverArt?id=mf-${item.id}`),
    reportPlayback: vi.fn(() => Promise.resolve()),
    reportPlaybackKeepalive: vi.fn(),
  },
}))

const createFakeCastEnvironment = () => {
  const ctxListeners = {}
  const controllerListeners = {}
  let nextItemId = 100
  let mediaItems = []
  let mediaIdleReason = null

  const requestSession = vi.fn(() => Promise.resolve())
  const endCurrentSession = vi.fn()
  const playOrPause = vi.fn()
  const seek = vi.fn()
  const setVolumeLevel = vi.fn()
  const muteOrUnmute = vi.fn()
  const queueNext = vi.fn((onSuccess) => {
    onSuccess?.()
  })
  const queuePrev = vi.fn((onSuccess) => {
    onSuccess?.()
  })
  const queueJumpToItem = vi.fn((itemId, onSuccess) => {
    if (!mediaItems.some((it) => it?.itemId === itemId)) {
      return
    }
    onSuccess?.()
  })
  const queueUpdateItems = vi.fn((_req, onSuccess) => {
    onSuccess?.()
  })
  const queueSetRepeatMode = vi.fn((_mode, onSuccess) => {
    onSuccess?.()
  })
  const stop = vi.fn((_req, onSuccess) => {
    mediaItems = []
    onSuccess?.()
  })

  const queueInsertItems = vi.fn((req, onSuccess) => {
    const assigned = (req.items || []).map((qItem) => {
      nextItemId += 1
      return { ...qItem, itemId: nextItemId }
    })
    if (req.insertBefore == null) {
      mediaItems = [...mediaItems, ...assigned]
    } else {
      const idx = mediaItems.findIndex((it) => it.itemId === req.insertBefore)
      if (idx >= 0) {
        mediaItems = [
          ...mediaItems.slice(0, idx),
          ...assigned,
          ...mediaItems.slice(idx),
        ]
      } else {
        mediaItems = [...mediaItems, ...assigned]
      }
    }
    onSuccess?.()
  })

  const queueRemoveItems = vi.fn((req, onSuccess) => {
    const removeSet = new Set(req.itemIds || [])
    mediaItems = mediaItems.filter((it) => !removeSet.has(it.itemId))
    onSuccess?.()
  })

  const queueReorderItems = vi.fn((req, onSuccess) => {
    const byId = new Map(mediaItems.map((it) => [it.itemId, it]))
    mediaItems = (req.itemIds || []).map((id) => byId.get(id)).filter(Boolean)
    onSuccess?.()
  })

  const remotePlayer = {
    isConnected: false,
    playerState: 'IDLE',
    currentTime: 0,
    duration: 200,
    volumeLevel: 0.8,
    isMuted: false,
    mediaInfo: null,
  }

  const getMediaSessionObj = () => ({
    playerState: remotePlayer.playerState,
    get idleReason() {
      return mediaIdleReason
    },
    get items() {
      return mediaItems
    },
    queueNext,
    queuePrev,
    queueJumpToItem,
    queueUpdateItems,
    queueInsertItems,
    queueRemoveItems,
    queueReorderItems,
    queueSetRepeatMode,
    stop,
  })

  const queueLoad = vi.fn((req, onSuccess) => {
    mediaItems = (req.items || []).map((qItem) => {
      nextItemId += 1
      return { ...qItem, itemId: nextItemId }
    })
    onSuccess?.(getMediaSessionObj())
  })

  let currentSession = null

  const castContext = {
    setOptions: vi.fn(),
    getCastState: vi.fn(() => 'NOT_CONNECTED'),
    getCurrentSession: vi.fn(() => currentSession),
    requestSession,
    endCurrentSession,
    addEventListener: vi.fn((type, fn) => {
      ctxListeners[type] = fn
    }),
    removeEventListener: vi.fn(),
  }

  window.cast = {
    framework: {
      CastContextEventType: {
        CAST_STATE_CHANGED: 'caststatechanged',
        SESSION_STATE_CHANGED: 'sessionstatechanged',
      },
      SessionState: {
        SESSION_STARTED: 'SESSION_STARTED',
        SESSION_RESUMED: 'SESSION_RESUMED',
        SESSION_ENDED: 'SESSION_ENDED',
      },
      CastState: {
        CONNECTING: 'CONNECTING',
      },
      RemotePlayerEventType: {
        ANY_CHANGE: 'anyChanged',
      },
      CastContext: {
        getInstance: () => castContext,
      },
      RemotePlayer: function RemotePlayer() {
        return remotePlayer
      },
      RemotePlayerController: function RemotePlayerController() {
        return {
          playOrPause,
          seek,
          setVolumeLevel,
          muteOrUnmute,
          addEventListener: vi.fn((type, fn) => {
            controllerListeners[type] = fn
          }),
          removeEventListener: vi.fn(),
        }
      },
    },
  }

  window.chrome = {
    cast: {
      AutoJoinPolicy: { ORIGIN_SCOPED: 'origin_scoped' },
      Image: function Image(url) {
        this.url = url
      },
      media: {
        DEFAULT_MEDIA_RECEIVER_APP_ID: 'CC1AD845',
        StreamType: { BUFFERED: 'BUFFERED', LIVE: 'LIVE' },
        MetadataType: { MUSIC_TRACK: 3 },
        RepeatMode: {
          OFF: 'REPEAT_OFF',
          ALL: 'REPEAT_ALL',
          SINGLE: 'REPEAT_SINGLE',
          ALL_AND_SHUFFLE: 'REPEAT_ALL_AND_SHUFFLE',
        },
        IdleReason: { FINISHED: 'FINISHED' },
        MediaInfo: function MediaInfo(contentId, contentType) {
          this.contentId = contentId
          this.contentType = contentType
        },
        MusicTrackMediaMetadata: function MusicTrackMediaMetadata() {},
        QueueItem: function QueueItem(mediaInfo) {
          this.media = mediaInfo
        },
        QueueLoadRequest: function QueueLoadRequest(items) {
          this.items = items
        },
        QueueInsertItemsRequest: function QueueInsertItemsRequest(items) {
          this.items = items
          this.insertBefore = null
        },
        QueueRemoveItemsRequest: function QueueRemoveItemsRequest(itemIds) {
          this.itemIds = itemIds
        },
        QueueReorderItemsRequest: function QueueReorderItemsRequest(itemIds) {
          this.itemIds = itemIds
        },
        QueueJumpRequest: function QueueJumpRequest() {
          this.currentItemId = null
          this.jump = null
        },
        PlayRequest: function PlayRequest() {},
        PauseRequest: function PauseRequest() {},
        SeekRequest: function SeekRequest() {
          this.currentTime = 0
        },
        StopRequest: function StopRequest() {},
      },
    },
  }

  return {
    castContext,
    remotePlayer,
    queueLoad,
    queueInsertItems,
    queueRemoveItems,
    queueReorderItems,
    queueNext,
    queuePrev,
    queueJumpToItem,
    queueUpdateItems,
    queueSetRepeatMode,
    stop,
    requestSession,
    endCurrentSession,
    playOrPause,
    seek,
    setVolumeLevel,
    muteOrUnmute,
    getMediaItems: () => mediaItems,
    setMediaItems: (items) => {
      mediaItems = items
    },
    setIdleReason: (reason) => {
      mediaIdleReason = reason
    },
    startSession(friendlyName = 'Living Room Speaker') {
      currentSession = {
        getCastDevice: () => ({ friendlyName }),
        getSessionObj: () => ({
          receiver: { friendlyName },
          queueLoad,
          media: [getMediaSessionObj()],
        }),
        getMediaSession: getMediaSessionObj,
      }
      remotePlayer.isConnected = true
      ctxListeners.sessionstatechanged?.({ sessionState: 'SESSION_STARTED' })
    },
    emitRemoteChange() {
      controllerListeners.anyChanged?.()
    },
    endSession() {
      currentSession = null
      remotePlayer.isConnected = false
      ctxListeners.sessionstatechanged?.({ sessionState: 'SESSION_ENDED' })
    },
  }
}

describe('Chromecast Web Sender (<CastProvider />, <CastButton /> & <Player />)', () => {
  const origExternalServices = config.enableExternalServices
  const origMatchMedia = window.matchMedia

  beforeEach(() => {
    window.matchMedia = vi.fn().mockImplementation((query) => ({
      matches: !query.includes('max-width: 768px'),
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }))
    vi.spyOn(window.HTMLMediaElement.prototype, 'pause').mockImplementation(
      function () {},
    )
    vi.spyOn(window.HTMLMediaElement.prototype, 'play').mockImplementation(
      function () {
        return Promise.resolve()
      },
    )
    vi.spyOn(window.HTMLMediaElement.prototype, 'load').mockImplementation(
      function () {},
    )
  })

  afterEach(() => {
    cleanup()
    config.enableExternalServices = origExternalServices
    window.matchMedia = origMatchMedia
    delete window.cast
    delete window.chrome
    delete window.__ND_CAST_CALLBACK_INSTALLED__
    delete window.__ND_CAST_API_READY__
    delete window.__onGCastApiAvailable
    localStorage.removeItem('nd-cast-session-owner')
    localStorage.removeItem('userId')
    document
      .querySelectorAll('script[data-nd-cast-sdk="true"]')
      .forEach((node) => node.remove())
    vi.restoreAllMocks()
    vi.clearAllMocks()
  })

  const buildFullStore = () =>
    createStore(
      combineReducers({
        player: playerReducer,
        settings: (s = { notifications: false }) => s,
        replayGain: (s = { gainMode: 'none', preAmp: 0 }) => s,
      }),
    )

  const song1 = {
    id: 'song-1',
    title: 'Weird Fishes',
    artist: 'Radiohead',
    album: 'In Rainbows',
    albumId: 'alb-1',
    suffix: 'flac',
    duration: 318,
  }
  const song2 = {
    id: 'song-2',
    title: 'All I Need',
    artist: 'Radiohead',
    album: 'In Rainbows',
    albumId: 'alb-1',
    suffix: 'flac',
    duration: 228,
  }
  const song3 = {
    id: 'song-3',
    title: 'Reckoner',
    artist: 'Radiohead',
    album: 'In Rainbows',
    albumId: 'alb-1',
    suffix: 'flac',
    duration: 290,
  }
  const song4 = {
    id: 'song-4',
    title: 'House of Cards',
    artist: 'Radiohead',
    album: 'In Rainbows',
    albumId: 'alb-1',
    suffix: 'flac',
    duration: 328,
  }

  it('hides CastButton and makes no SDK request when Google Cast is unsupported', () => {
    const store = buildFullStore()
    render(
      <Provider store={store}>
        <MemoryRouter>
          <CastProvider audioInstance={null}>
            <CastButton />
          </CastProvider>
        </MemoryRouter>
      </Provider>,
    )

    expect(screen.queryByTestId('cast-button')).toBeNull()
    expect(document.querySelector('script[data-nd-cast-sdk="true"]')).toBeNull()
  })

  it('skips loading external Cast SDK when enableExternalServices is false', () => {
    config.enableExternalServices = false
    window.chrome = {}
    const store = buildFullStore()

    render(
      <Provider store={store}>
        <MemoryRouter>
          <CastProvider audioInstance={null}>
            <CastButton />
          </CastProvider>
        </MemoryRouter>
      </Provider>,
    )

    expect(document.querySelector('script[data-nd-cast-sdk="true"]')).toBeNull()
    expect(screen.queryByTestId('cast-button')).toBeNull()
  })

  it('dynamically injects Cast Web Sender SDK once on Chromium when authenticated', () => {
    config.enableExternalServices = true
    window.chrome = {}
    const store = buildFullStore()

    render(
      <Provider store={store}>
        <MemoryRouter>
          <CastProvider audioInstance={null} enabled={true}>
            <CastButton />
          </CastProvider>
        </MemoryRouter>
      </Provider>,
    )

    const scripts = document.querySelectorAll('script[data-nd-cast-sdk="true"]')
    expect(scripts).toHaveLength(1)
    expect(scripts[0].src).toContain('cast_sender.js?loadCastFramework=1')
  })

  it('uses a single unified player UI, preserves the queue drawer, and queues "Play Next" & "Add to Queue" without restarting the active song', async () => {
    const env = createFakeCastEnvironment()
    const store = buildFullStore()

    // Load initial 2-song queue into real Redux playerReducer
    store.dispatch(
      playTracks({ 'song-1': song1, 'song-2': song2 }, ['song-1', 'song-2']),
    )

    render(
      <Provider store={store}>
        <MemoryRouter>
          <Player />
        </MemoryRouter>
      </Provider>,
    )

    // Verify local playback queue exists in the single native player before casting
    expect(
      document.querySelectorAll('.react-jinke-music-player-main'),
    ).toHaveLength(1)
    expect(screen.queryByTestId('cast-player-bar')).toBeNull()
    expect(document.querySelector('.audio-lists-num')?.textContent).toBe('2')

    const audioEl = document.querySelector('audio.music-player-audio')
    expect(audioEl).not.toBeNull()
    audioEl.currentTime = 47.6

    // Click Cast button -> requests Cast session
    const castBtn = screen.getByTestId('cast-button')
    fireEvent.click(castBtn)
    expect(env.requestSession).toHaveBeenCalledTimes(1)

    // Start Cast session at 47s
    await act(async () => {
      env.startSession('Living Room Speaker')
    })

    // Still ONLY ONE player in the DOM (no second/superimposed CastPlayerBar!)
    expect(
      document.querySelectorAll('.react-jinke-music-player-main'),
    ).toHaveLength(1)
    expect(screen.queryByTestId('cast-player-bar')).toBeNull()
    expect(
      document.querySelector('.react-jinke-music-player-main'),
    ).toBeVisible()

    // Initial QueueLoadRequest sent once at 47s offset using scoped /share/cast & /share/img tokens (zero Subsonic u/t/s)
    expect(httpClient).toHaveBeenCalledWith(
      '/api/cast/tokens',
      expect.objectContaining({ method: 'POST' }),
    )
    expect(env.queueLoad).toHaveBeenCalledTimes(1)
    const initialLoadReq = env.queueLoad.mock.calls[0][0]
    expect(initialLoadReq.items).toHaveLength(2)
    expect(initialLoadReq.startIndex).toBe(0)
    expect(initialLoadReq.currentTime).toBe(47)
    expect(initialLoadReq.items[0].media.contentId).toContain(
      '/share/cast/cast-stream-jwt-song-1-raw',
    )
    expect(initialLoadReq.items[0].media.contentId).not.toContain('/rest/')
    expect(initialLoadReq.items[0].media.metadata.images[0].url).toContain(
      '/share/img/cast-cover-jwt-song-1?size=300',
    )

    const initialUuids = store.getState().player.queue.map((i) => i.uuid)
    const secondTrackCastItemId = env.getMediaItems()[1].itemId

    // Simulate Chromecast playing Song 1 at t = 85s
    await act(async () => {
      env.remotePlayer.playerState = 'PLAYING'
      env.remotePlayer.currentTime = 85
      env.remotePlayer.duration = 318
      env.remotePlayer.mediaInfo = {
        customData: {
          trackId: 'song-1',
          uuid: initialUuids[0],
          isRadio: false,
        },
      }
      env.emitRemoteChange()
    })

    expect(subsonic.reportPlayback).toHaveBeenCalledWith(
      'song-1',
      85000,
      'starting',
    )
    expect(subsonic.reportPlayback).toHaveBeenCalledWith(
      'song-1',
      85000,
      'playing',
    )
    expect(audioEl.currentTime).toBe(85)

    // Dispatch "Play Next" (song-3) while Song 1 is playing at 85s!
    await act(async () => {
      store.dispatch(playNext({ 'song-3': song3 }, ['song-3']))
    })

    // MUST NOT call queueLoad again or reset currentTime to 0!
    expect(env.queueLoad).toHaveBeenCalledTimes(1)
    expect(env.queueInsertItems).toHaveBeenCalledTimes(1)
    const playNextReq = env.queueInsertItems.mock.calls[0][0]
    expect(playNextReq.items).toHaveLength(1)
    expect(playNextReq.items[0].media.metadata.title).toBe('Reckoner')
    expect(playNextReq.insertBefore).toBe(secondTrackCastItemId)
    expect(audioEl.currentTime).toBe(85)

    // Native queue pill & drawer reflect all 3 tracks in order [Weird Fishes, Reckoner, All I Need]
    expect(document.querySelector('.audio-lists-num')?.textContent).toBe('3')
    expect(store.getState().player.queue.map((t) => t.name)).toEqual([
      'Weird Fishes',
      'Reckoner',
      'All I Need',
    ])

    // Dispatch "Add to Queue" (song-4) while Song 1 continues playing!
    await act(async () => {
      store.dispatch(addTracks({ 'song-4': song4 }, ['song-4']))
    })

    expect(env.queueLoad).toHaveBeenCalledTimes(1)
    expect(env.queueInsertItems).toHaveBeenCalledTimes(2)
    const addTracksReq = env.queueInsertItems.mock.calls[1][0]
    expect(addTracksReq.items).toHaveLength(1)
    expect(addTracksReq.items[0].media.metadata.title).toBe('House of Cards')
    expect(addTracksReq.insertBefore).toBeNull()
    expect(document.querySelector('.audio-lists-num')?.textContent).toBe('4')

    // Native player's Play/Pause button toggles RemotePlayerController cleanly
    const playToggleBtn = document.querySelector('.play-btn')
    expect(playToggleBtn).not.toBeNull()
    act(() => {
      fireEvent.click(playToggleBtn)
    })
    expect(env.playOrPause).toHaveBeenCalledTimes(1)

    // Clicking CastButton while casting opens disconnect menu and stops session cleanly
    fireEvent.click(screen.getByTestId('cast-button'))
    const stopItem = screen.getByTestId('cast-stop-button')
    expect(stopItem.textContent).toContain('Living Room Speaker')
    fireEvent.click(stopItem)
    expect(env.endCurrentSession).toHaveBeenCalledWith(true)

    act(() => {
      env.endSession()
    })

    // After Cast disconnects, ALL 4 tracks remain intact in the UI queue and local audio is restored unshimmed
    expect(document.querySelector('.audio-lists-num')?.textContent).toBe('4')
    expect(store.getState().player.queue.map((t) => t.name)).toEqual([
      'Weird Fishes',
      'Reckoner',
      'All I Need',
      'House of Cards',
    ])
    expect(Object.prototype.hasOwnProperty.call(audioEl, 'currentTime')).toBe(
      false,
    )
    expect(audioEl.currentTime).toBe(85)
  })

  it('forwards Queue Drawer track clicks via queueJumpToItem and syncs Chromecast auto-advance back into the native player', async () => {
    const env = createFakeCastEnvironment()
    const store = buildFullStore()

    store.dispatch(
      playTracks({ 'song-1': song1, 'song-2': song2, 'song-3': song3 }, [
        'song-1',
        'song-2',
        'song-3',
      ]),
    )

    render(
      <Provider store={store}>
        <MemoryRouter>
          <Player />
        </MemoryRouter>
      </Provider>,
    )

    await act(async () => {
      env.startSession('Studio Chromecast')
    })

    expect(env.queueLoad).toHaveBeenCalledTimes(1)
    const uuids = store.getState().player.queue.map((i) => i.uuid)
    const castItems = env.getMediaItems()

    await act(async () => {
      env.remotePlayer.playerState = 'PLAYING'
      env.remotePlayer.currentTime = 12
      env.remotePlayer.mediaInfo = {
        customData: { trackId: 'song-1', uuid: uuids[0], isRadio: false },
      }
      env.emitRemoteChange()
    })

    // Open native AudioListsPanel queue drawer and click Song 3
    fireEvent.click(document.querySelector('.audio-lists-btn'))
    const queueRows = document.querySelectorAll(
      '.audio-lists-panel-content .audio-item',
    )
    expect(queueRows).toHaveLength(3)

    await act(async () => {
      fireEvent.click(queueRows[2])
    })

    expect(env.queueJumpToItem).toHaveBeenCalledTimes(1)
    expect(env.queueJumpToItem.mock.calls[0][0]).toBe(castItems[2].itemId)
    expect(env.queueLoad).toHaveBeenCalledTimes(1)

    // Simulate Chromecast auto-advancing to Song 2 without triggering duplicate jumps
    await act(async () => {
      env.remotePlayer.playerState = 'PLAYING'
      env.remotePlayer.currentTime = 0
      env.remotePlayer.mediaInfo = {
        customData: { trackId: 'song-2', uuid: uuids[1], isRadio: false },
      }
      env.emitRemoteChange()
    })

    expect(store.getState().player.current.name).toBe('All I Need')
    expect(store.getState().player.savedPlayIndex).toBe(1)
    expect(env.queueJumpToItem).toHaveBeenCalledTimes(1)
    expect(env.queueLoad).toHaveBeenCalledTimes(1)
  })

  it('preserves the native UI queue and leaves HTMLAudioElement untouched when Chromecast is not active', async () => {
    createFakeCastEnvironment()
    const store = buildFullStore()

    store.dispatch(
      playTracks({ 'song-1': song1, 'song-2': song2 }, ['song-1', 'song-2']),
    )

    render(
      <Provider store={store}>
        <MemoryRouter>
          <Player />
        </MemoryRouter>
      </Provider>,
    )

    const audioEl = document.querySelector('audio.music-player-audio')
    expect(Object.prototype.hasOwnProperty.call(audioEl, 'currentTime')).toBe(
      false,
    )
    expect(Object.prototype.hasOwnProperty.call(audioEl, 'pause')).toBe(false)
    expect(document.querySelector('.audio-lists-num')?.textContent).toBe('2')

    await act(async () => {
      store.dispatch(playNext({ 'song-3': song3 }, ['song-3']))
      store.dispatch(addTracks({ 'song-4': song4 }, ['song-4']))
    })

    expect(document.querySelector('.audio-lists-num')?.textContent).toBe('4')
    fireEvent.click(document.querySelector('.audio-lists-btn'))
    const queueRows = document.querySelectorAll(
      '.audio-lists-panel-content .audio-item',
    )
    expect(queueRows).toHaveLength(4)
  })

  it('ends any active Cast session on logout and re-initializes Cast SDK listeners cleanly on next login', async () => {
    const env = createFakeCastEnvironment()
    const store = buildFullStore()
    store.dispatch(playTracks({ 'song-1': song1 }, ['song-1']))
    const fakeAudio = document.createElement('audio')

    const { rerender } = render(
      <Provider store={store}>
        <CastProvider audioInstance={fakeAudio} enabled={true}>
          <CastButton />
        </CastProvider>
      </Provider>,
    )

    expect(screen.getByTestId('cast-button')).toBeTruthy()
    await act(async () => {
      env.startSession('Shared Lounge TV')
    })
    expect(env.queueLoad).toHaveBeenCalledTimes(1)

    // Simulate logout while casting -> must terminate the Cast session so next user cannot take it over
    rerender(
      <Provider store={store}>
        <CastProvider audioInstance={fakeAudio} enabled={false}>
          <CastButton />
        </CastProvider>
      </Provider>,
    )
    expect(env.endCurrentSession).toHaveBeenCalledWith(true)
    expect(screen.queryByTestId('cast-button')).toBeNull()

    // Simulate login without page reload
    rerender(
      <Provider store={store}>
        <CastProvider audioInstance={fakeAudio} enabled={true}>
          <CastButton />
        </CastProvider>
      </Provider>,
    )
    expect(screen.getByTestId('cast-button')).toBeTruthy()

    await act(async () => {
      env.startSession('Office Speaker')
    })

    expect(env.queueLoad).toHaveBeenCalledTimes(2)
    fireEvent.click(screen.getByTestId('cast-button'))
    expect(screen.getByTestId('cast-stop-button')).toBeTruthy()
  })

  it('uses REPEAT_OFF for partial >40-track windows and advances only once across burst FINISHED events', async () => {
    const env = createFakeCastEnvironment()
    const store = buildFullStore()

    const songsById = {}
    const songIds = []
    for (let i = 1; i <= 90; i += 1) {
      const id = `song-${i}`
      songsById[id] = {
        ...song1,
        id,
        title: `Track ${i}`,
      }
      songIds.push(id)
    }
    store.dispatch(playTracks(songsById, songIds))
    store.dispatch({ type: 'PLAYER_SET_MODE', data: { mode: 'orderLoop' } })

    const fakeAudio = document.createElement('audio')
    render(
      <Provider store={store}>
        <CastProvider audioInstance={fakeAudio} enabled={true}>
          <CastButton />
        </CastProvider>
      </Provider>,
    )

    await act(async () => {
      env.startSession('Hallway Chromecast')
    })

    expect(env.queueLoad).toHaveBeenCalledTimes(1)
    // Partial window [0, 40) of 90 tracks must not send REPEAT_ALL to receiver
    expect(env.queueLoad.mock.calls[0][0].repeatMode).toBe('REPEAT_OFF')

    const queue = store.getState().player.queue
    await act(async () => {
      env.remotePlayer.playerState = 'PLAYING'
      env.remotePlayer.currentTime = 179
      env.remotePlayer.mediaInfo = {
        customData: {
          trackId: 'song-40',
          uuid: queue[39].uuid,
          isRadio: false,
        },
      }
      env.emitRemoteChange()
    })

    // Burst of 3 ANY_CHANGE events while receiver is IDLE + FINISHED
    await act(async () => {
      env.remotePlayer.playerState = 'IDLE'
      env.setIdleReason('FINISHED')
      env.emitRemoteChange()
      env.emitRemoteChange()
      env.emitRemoteChange()
    })

    // Must load the next window starting at index 40 exactly once (not skip to 75+)
    expect(env.queueLoad).toHaveBeenCalledTimes(2)
    expect(
      env.queueLoad.mock.calls[1][0].items[
        env.queueLoad.mock.calls[1][0].startIndex
      ].media.customData.uuid,
    ).toBe(queue[40].uuid)
  })

  it('ignores untrusted receiver trackIds not present in the local queue', async () => {
    const env = createFakeCastEnvironment()
    const store = buildFullStore()
    store.dispatch(playTracks({ 'song-1': song1 }, ['song-1']))

    const fakeAudio = document.createElement('audio')
    fakeAudio.currentTime = 17

    render(
      <Provider store={store}>
        <CastProvider audioInstance={fakeAudio} enabled={true}>
          <CastButton />
        </CastProvider>
      </Provider>,
    )

    await act(async () => {
      env.startSession('Shared Receiver')
    })
    subsonic.reportPlayback.mockClear()

    // Another LAN sender loads an external item with a traversal-like trackId
    await act(async () => {
      env.remotePlayer.playerState = 'PLAYING'
      env.remotePlayer.currentTime = 95
      env.remotePlayer.mediaInfo = {
        customData: {
          trackId: '../../api/user',
          uuid: 'foreign-uuid',
          isRadio: false,
        },
      }
      env.emitRemoteChange()
    })

    expect(subsonic.reportPlayback).not.toHaveBeenCalledWith(
      '../../api/user',
      expect.anything(),
      expect.anything(),
    )
  })

  it('recovers player controls when receiver transitions to IDLE with IdleReason.ERROR', async () => {
    const env = createFakeCastEnvironment()
    const store = buildFullStore()
    store.dispatch(playTracks({ 'song-1': song1 }, ['song-1']))
    const fakeAudio = document.createElement('audio')

    render(
      <Provider store={store}>
        <CastProvider audioInstance={fakeAudio} enabled={true}>
          <CastButton />
        </CastProvider>
      </Provider>,
    )

    await act(async () => {
      env.startSession('Bedroom Speaker')
    })
    expect(env.queueLoad).toHaveBeenCalledTimes(1)

    // Receiver fails to fetch stream and goes IDLE + ERROR
    await act(async () => {
      env.remotePlayer.playerState = 'IDLE'
      env.setIdleReason('ERROR')
      env.emitRemoteChange()
    })

    // Calling play() on the shimmed audio element must now recover and reload the queue instead of stalling
    await act(async () => {
      await fakeAudio.play()
    })
    expect(env.queueLoad).toHaveBeenCalledTimes(2)
  })

  it('advances elapsed time and updates track metadata when CAF RemotePlayer stays frozen after queueLoad(media === null)', async () => {
    vi.useFakeTimers()
    try {
      const env = createFakeCastEnvironment()
      const store = buildFullStore()
      store.dispatch(
        playTracks({ 'song-1': song1, 'song-2': song2 }, ['song-1', 'song-2']),
      )

      let liveMedia = null
      const updateListeners = new Set()
      let lastUpdatedAtMs = Date.now()

      const mediaPlay = vi.fn((_req, onSuccess) => {
        if (liveMedia) {
          liveMedia.playerState = 'PLAYING'
          lastUpdatedAtMs = Date.now()
        }
        onSuccess?.()
        updateListeners.forEach((fn) => fn(true))
      })
      const mediaPause = vi.fn((_req, onSuccess) => {
        if (liveMedia) {
          liveMedia.currentTime = liveMedia.getEstimatedTime()
          liveMedia.playerState = 'PAUSED'
          lastUpdatedAtMs = Date.now()
        }
        onSuccess?.()
        updateListeners.forEach((fn) => fn(true))
      })
      const mediaSeek = vi.fn((req, onSuccess) => {
        if (liveMedia) {
          liveMedia.currentTime = req.currentTime
          lastUpdatedAtMs = Date.now()
        }
        onSuccess?.()
        updateListeners.forEach((fn) => fn(true))
      })

      // Mirror real Google Cast Web Sender queueLoad: initial response has
      // playerState === 'IDLE', loadingItemId === 101, media === null, which
      // causes CAF RemotePlayerController's internal media pointer to stay null
      // and RemotePlayer to remain permanently frozen at IDLE / 0s / null.
      const realQueueLoad = vi.fn((req, onSuccess) => {
        const assignedItems = (req.items || []).map((qItem, idx) => ({
          ...qItem,
          itemId: 101 + idx,
        }))
        lastUpdatedAtMs = Date.now()
        liveMedia = {
          playerState: 'IDLE',
          idleReason: null,
          currentItemId: null,
          loadingItemId: assignedItems[0]?.itemId ?? null,
          media: null,
          currentTime: 0,
          playbackRate: 1,
          items: assignedItems,
          addUpdateListener: vi.fn((fn) => updateListeners.add(fn)),
          removeUpdateListener: vi.fn((fn) => updateListeners.delete(fn)),
          getEstimatedTime() {
            if (this.playerState === 'PLAYING') {
              const est =
                this.currentTime +
                ((Date.now() - lastUpdatedAtMs) / 1000) * this.playbackRate
              const dur = this.media?.duration
              return Number.isFinite(dur) && dur > 0 ? Math.min(est, dur) : est
            }
            return this.currentTime
          },
          play: mediaPlay,
          pause: mediaPause,
          seek: mediaSeek,
        }
        onSuccess?.(liveMedia)
      })

      const sessionObj = {
        receiver: { friendlyName: 'Office Chromecast' },
        queueLoad: realQueueLoad,
        addMediaListener: vi.fn(),
        removeMediaListener: vi.fn(),
        get media() {
          return liveMedia ? [liveMedia] : []
        },
      }
      const frozenSession = {
        getCastDevice: () => ({ friendlyName: 'Office Chromecast' }),
        getSessionObj: () => sessionObj,
        getMediaSession: () =>
          liveMedia && !liveMedia.idleReason ? liveMedia : null,
      }
      env.castContext.getCurrentSession.mockImplementation(() => frozenSession)

      render(
        <Provider store={store}>
          <MemoryRouter>
            <Player />
          </MemoryRouter>
        </Provider>,
      )

      const audioEl = document.querySelector('audio.music-player-audio')
      expect(audioEl).not.toBeNull()

      // Start casting while RemotePlayer stays frozen at IDLE / currentTime: 0 / mediaInfo: null
      // and NEVER fires RemotePlayerEventType.ANY_CHANGE.
      await act(async () => {
        env.remotePlayer.isConnected = true
        env.remotePlayer.playerState = 'IDLE'
        env.remotePlayer.currentTime = 0
        env.remotePlayer.duration = 0
        env.remotePlayer.mediaInfo = null
        const sessionHandler = env.castContext.addEventListener.mock.calls.find(
          (c) => c[0] === 'sessionstatechanged',
        )?.[1]
        sessionHandler?.({ sessionState: 'SESSION_STARTED' })
        await vi.advanceTimersByTimeAsync(20)
      })

      expect(realQueueLoad).toHaveBeenCalledTimes(1)
      expect(liveMedia).not.toBeNull()
      expect(liveMedia.addUpdateListener).toHaveBeenCalledTimes(1)

      // Receiver starts playing Song 1 (itemId 101) — notified ONLY via media.addUpdateListener
      await act(async () => {
        liveMedia.playerState = 'PLAYING'
        liveMedia.idleReason = null
        liveMedia.loadingItemId = null
        liveMedia.currentItemId = 101
        liveMedia.currentTime = 0
        liveMedia.media = {
          ...liveMedia.items[0].media,
          duration: 318,
        }
        lastUpdatedAtMs = Date.now()
        updateListeners.forEach((fn) => fn(true))
        await vi.advanceTimersByTimeAsync(20)
      })

      expect(document.querySelector('.audio-title')?.textContent).toContain(
        'Weird Fishes',
      )
      expect(document.querySelector('.duration')?.textContent).toBe('05:18')
      expect(document.querySelector('.current-time')?.textContent).toBe('00:00')

      // Advance 5 seconds with ZERO incoming MEDIA_STATUS packets — local ticker must advance progress bar
      await act(async () => {
        await vi.advanceTimersByTimeAsync(5000)
      })

      expect(document.querySelector('.current-time')?.textContent).toBe('00:05')
      expect(
        document
          .querySelector('.progress-bar [role="slider"]')
          ?.getAttribute('aria-valuenow'),
      ).toBe('5')

      // Receiver finishes Song 1 -> briefly emits IDLE + FINISHED + loadingItemId: 102 (media: null)
      // -> then transitions to PLAYING Song 2 (itemId 102, duration 228s = 03:48)
      await act(async () => {
        liveMedia.playerState = 'IDLE'
        liveMedia.idleReason = 'FINISHED'
        liveMedia.currentItemId = 101
        liveMedia.loadingItemId = 102
        liveMedia.media = null
        updateListeners.forEach((fn) => fn(true))
        await vi.advanceTimersByTimeAsync(20)

        liveMedia.playerState = 'PLAYING'
        liveMedia.idleReason = null
        liveMedia.loadingItemId = null
        liveMedia.currentItemId = 102
        liveMedia.currentTime = 0
        liveMedia.media = {
          ...liveMedia.items[1].media,
          duration: 228,
        }
        lastUpdatedAtMs = Date.now()
        updateListeners.forEach((fn) => fn(true))
        await vi.advanceTimersByTimeAsync(50)
      })

      // Must NOT reload the queue during track transition, and UI must switch to Song 2 ("All I Need")
      expect(realQueueLoad).toHaveBeenCalledTimes(1)
      expect(store.getState().player.current.name).toBe('All I Need')
      expect(store.getState().player.savedPlayIndex).toBe(1)
      expect(document.querySelector('.audio-title')?.textContent).toContain(
        'All I Need',
      )
      expect(document.querySelector('.duration')?.textContent).toBe('03:48')
      expect(document.querySelector('.current-time')?.textContent).toBe('00:00')

      // Advance 3 seconds on Song 2 — progress bar must tick 00:03 on the new track
      await act(async () => {
        await vi.advanceTimersByTimeAsync(3500)
      })
      expect(document.querySelector('.current-time')?.textContent).toBe('00:03')

      // Direct media.pause / media.play / media.seek must work even with frozen RemotePlayerController
      await act(async () => {
        fireEvent.click(document.querySelector('.play-btn'))
        await vi.advanceTimersByTimeAsync(20)
      })
      expect(mediaPause).toHaveBeenCalledTimes(1)

      await act(async () => {
        fireEvent.click(document.querySelector('.play-btn'))
        await vi.advanceTimersByTimeAsync(20)
      })
      expect(mediaPlay).toHaveBeenCalledTimes(1)

      await act(async () => {
        audioEl.currentTime = 120
        await vi.advanceTimersByTimeAsync(200)
      })
      expect(mediaSeek).toHaveBeenCalledTimes(1)
      expect(mediaSeek.mock.calls[0][0].currentTime).toBe(120)
      expect(document.querySelector('.current-time')?.textContent).toBe('02:00')
    } finally {
      vi.useRealTimers()
    }
  })

  it('shuffles a playlist containing the currently playing track while /api/cast/tokens is slow without leaking old elapsed time or self-cancelling queueLoad', async () => {
    vi.useFakeTimers()
    const defaultHttpImpl = httpClient.getMockImplementation()
    try {
      const env = createFakeCastEnvironment()
      const store = buildFullStore()

      const medahat = {
        ...song1,
        id: 'song-medahat',
        title: 'Medahat',
        artist: 'Acid Arab',
        duration: 272,
      }
      const watchMyHeart = {
        ...song2,
        id: 'song-watch',
        title: 'Watch my heart go...',
        artist: 'R3HAB',
        duration: 185,
      }
      const loseMyMind = {
        ...song3,
        id: 'song-lose',
        title: 'Lose My Mind (Acid Arab Remix)',
        artist: 'Emel, Acid Arab',
        duration: 259,
      }
      const laHafla = {
        ...song4,
        id: 'song-hafla',
        title: 'La Hafla',
        artist: 'Acid Arab',
        duration: 262,
      }

      store.dispatch(
        playTracks(
          {
            'song-medahat': medahat,
            'song-lose': loseMyMind,
          },
          ['song-medahat', 'song-lose'],
        ),
      )

      let liveMedia = null
      let lastUpdatedAtMs = Date.now()
      const makeLiveMedia = (items, baseItemId) => {
        const listeners = new Set()
        const assigned = items.map((qItem, idx) => ({
          ...qItem,
          itemId: baseItemId + idx,
        }))
        const mediaObj = {
          playerState: 'IDLE',
          idleReason: null,
          currentItemId: null,
          loadingItemId: assigned[0]?.itemId ?? null,
          media: null,
          currentTime: 0,
          playbackRate: 1,
          items: assigned.slice(0, 2),
          addUpdateListener: vi.fn((fn) => listeners.add(fn)),
          removeUpdateListener: vi.fn((fn) => listeners.delete(fn)),
          emit() {
            listeners.forEach((fn) => fn(true))
          },
          getEstimatedTime() {
            if (this.playerState === 'PLAYING') {
              return (
                this.currentTime +
                ((Date.now() - lastUpdatedAtMs) / 1000) * this.playbackRate
              )
            }
            return this.currentTime
          },
        }
        return mediaObj
      }

      const realQueueLoad = vi.fn((req, onSuccess) => {
        const baseId = realQueueLoad.mock.calls.length * 100
        lastUpdatedAtMs = Date.now()
        liveMedia = makeLiveMedia(req.items || [], baseId)
        onSuccess?.(liveMedia)
      })

      const sessionObj = {
        receiver: { friendlyName: 'Living Room Chromecast' },
        queueLoad: realQueueLoad,
        addMediaListener: vi.fn(),
        removeMediaListener: vi.fn(),
        get media() {
          return liveMedia ? [liveMedia] : []
        },
      }
      const castSession = {
        getCastDevice: () => ({ friendlyName: 'Living Room Chromecast' }),
        getSessionObj: () => sessionObj,
        getMediaSession: () => liveMedia,
      }
      env.castContext.getCurrentSession.mockImplementation(() => castSession)

      render(
        <Provider store={store}>
          <MemoryRouter>
            <Player />
          </MemoryRouter>
        </Provider>,
      )

      await act(async () => {
        env.remotePlayer.isConnected = true
        const sessionHandler = env.castContext.addEventListener.mock.calls.find(
          (c) => c[0] === 'sessionstatechanged',
        )?.[1]
        sessionHandler?.({ sessionState: 'SESSION_STARTED' })
        await vi.advanceTimersByTimeAsync(20)
      })

      expect(realQueueLoad).toHaveBeenCalledTimes(1)
      const oldMedia = liveMedia
      const initialUuids = store.getState().player.queue.map((i) => i.uuid)

      // Medahat is actively playing on Chromecast at 02:04 (124s)
      await act(async () => {
        oldMedia.playerState = 'PLAYING'
        oldMedia.loadingItemId = null
        oldMedia.currentItemId = 100
        oldMedia.currentTime = 124
        oldMedia.media = {
          customData: {
            trackId: 'song-medahat',
            uuid: initialUuids[0],
            isRadio: false,
          },
          duration: 272,
        }
        env.remotePlayer.playerState = 'PLAYING'
        env.remotePlayer.currentTime = 124
        env.remotePlayer.duration = 272
        env.remotePlayer.mediaInfo = oldMedia.media
        lastUpdatedAtMs = Date.now()
        oldMedia.emit()
        await vi.advanceTimersByTimeAsync(20)
      })

      expect(document.querySelector('.audio-title')?.textContent).toContain(
        'Medahat',
      )
      expect(document.querySelector('.current-time')?.textContent).toBe('02:04')

      // Simulate a slow 2000ms /api/cast/tokens response (e.g. SQLite lock during scan)
      // while the user clicks SHUFFLE on the same playlist (so song-medahat is STILL in the
      // new queue with a brand-new UUID, and Watch my heart go... is now at index 0).
      httpClient.mockClear()
      httpClient.mockImplementation(
        (_url, options) =>
          new Promise((resolve) => {
            setTimeout(() => {
              resolve(defaultHttpImpl(_url, options))
            }, 2000)
          }),
      )

      const randomSpy = vi
        .spyOn(Math, 'random')
        .mockReturnValueOnce(0)
        .mockReturnValueOnce(0)
        .mockReturnValueOnce(0)
      await act(async () => {
        // With Fisher-Yates & Math.random() === 0 on [medahat, watch, lose, hafla]:
        // i=3 -> [hafla, watch, lose, medahat]
        // i=2 -> [lose, watch, hafla, medahat]
        // i=1 -> [watch, lose, hafla, medahat]
        store.dispatch(
          shuffleTracks(
            {
              'song-medahat': medahat,
              'song-watch': watchMyHeart,
              'song-lose': loseMyMind,
              'song-hafla': laHafla,
            },
            ['song-medahat', 'song-watch', 'song-lose', 'song-hafla'],
          ),
        )
        await vi.advanceTimersByTimeAsync(50)
      })
      randomSpy.mockRestore()

      expect(store.getState().player.queue.map((t) => t.trackId)).toEqual([
        'song-watch',
        'song-lose',
        'song-hafla',
        'song-medahat',
      ])

      // While /api/cast/tokens is still in flight for 1.5s, oldMedia keeps playing Medahat
      // at 125s..126s and emitting MEDIA_STATUS + 500ms local ticks!
      await act(async () => {
        await vi.advanceTimersByTimeAsync(750)
        oldMedia.currentTime = 125
        env.remotePlayer.currentTime = 125
        lastUpdatedAtMs = Date.now()
        oldMedia.emit()
        env.emitRemoteChange()
        await vi.advanceTimersByTimeAsync(750)
      })

      // UI MUST stay on Watch my heart go... at 00:00 / 03:05 (NEVER 02:04/02:05!)
      // and MUST NOT self-cancel & re-fire /api/cast/tokens in a loop!
      expect(document.querySelector('.audio-title')?.textContent).toContain(
        'Watch my heart go...',
      )
      expect(document.querySelector('.duration')?.textContent).toBe('03:05')
      expect(document.querySelector('.current-time')?.textContent).toBe('00:00')
      expect(httpClient).toHaveBeenCalledTimes(1)
      expect(realQueueLoad).toHaveBeenCalledTimes(1)

      // Finish the 2000ms token request -> new shuffled queue MUST load on Chromecast!
      await act(async () => {
        await vi.advanceTimersByTimeAsync(600)
      })

      expect(httpClient).toHaveBeenCalledTimes(1)
      expect(realQueueLoad).toHaveBeenCalledTimes(2)
      const shuffledLoadReq = realQueueLoad.mock.calls[1][0]
      expect(shuffledLoadReq.startIndex).toBe(0)
      expect(shuffledLoadReq.items[0].media.metadata.title).toBe(
        'Watch my heart go...',
      )

      // Receiver begins playing Watch my heart go... -> elapsed time advances from 00:00 -> 00:03
      await act(async () => {
        liveMedia.playerState = 'PLAYING'
        liveMedia.loadingItemId = null
        liveMedia.currentItemId = 200
        liveMedia.currentTime = 0
        liveMedia.media = {
          ...liveMedia.items[0].media,
          duration: 185,
        }
        lastUpdatedAtMs = Date.now()
        liveMedia.emit()
        await vi.advanceTimersByTimeAsync(20)
      })

      await act(async () => {
        await vi.advanceTimersByTimeAsync(3500)
      })

      expect(document.querySelector('.audio-title')?.textContent).toContain(
        'Watch my heart go...',
      )
      expect(document.querySelector('.current-time')?.textContent).toBe('00:03')
    } finally {
      httpClient.mockImplementation(defaultHttpImpl)
      vi.useRealTimers()
    }
  })

  it('jumps across tracks when Chromecast trims MEDIA_STATUS.items to a 2-item window and reloads cleanly on >40-track shuffle jumps', async () => {
    const env = createFakeCastEnvironment()
    const store = buildFullStore()

    const songsById = {}
    const songIds = []
    for (let i = 1; i <= 60; i += 1) {
      const id = `song-${i}`
      songsById[id] = {
        ...song1,
        id,
        title: `Shuffled Track ${i}`,
        duration: 180 + i,
      }
      songIds.push(id)
    }
    store.dispatch(playTracks(songsById, songIds))

    // Mirror CC1AD845 Default Media Receiver: queueLoad accepts 40 items,
    // but MEDIA_STATUS.items only exposes a 2-item window [item 0, item 1].
    let fullLoadedItems = []
    env.queueLoad.mockImplementation((req, onSuccess) => {
      fullLoadedItems = (req.items || []).map((qItem, idx) => ({
        ...qItem,
        itemId: 500 + idx,
      }))
      const start = req.startIndex || 0
      env.setMediaItems(fullLoadedItems.slice(start, start + 2))
      onSuccess?.(env.castContext.getCurrentSession().getMediaSession())
    })

    render(
      <Provider store={store}>
        <MemoryRouter>
          <Player />
        </MemoryRouter>
      </Provider>,
    )

    await act(async () => {
      env.startSession('Living Room Speaker')
    })

    expect(env.queueLoad).toHaveBeenCalledTimes(1)
    const queue = store.getState().player.queue

    // Track 1 is playing on Chromecast; receiver only exposed itemIds 500 & 501
    await act(async () => {
      env.remotePlayer.playerState = 'PLAYING'
      env.remotePlayer.currentTime = 42
      env.remotePlayer.duration = 181
      env.remotePlayer.mediaInfo = {
        customData: {
          trackId: 'song-1',
          uuid: queue[0].uuid,
          isRadio: false,
        },
      }
      env.emitRemoteChange()
    })

    // User clicks Track 6 (index 5) in the Queue Drawer: its itemId (505) was NEVER
    // included in Chromecast's 2-item MEDIA_STATUS.items slice!
    fireEvent.click(document.querySelector('.audio-lists-btn'))
    const queueRows = document.querySelectorAll(
      '.audio-lists-panel-content .audio-item',
    )
    expect(queueRows).toHaveLength(60)

    await act(async () => {
      fireEvent.click(queueRows[5])
    })

    // Must NOT hang on silent-drop queueJumpToItem; must send QueueJumpRequest{ jump: 5 }!
    expect(env.queueJumpToItem).not.toHaveBeenCalled()
    expect(env.queueUpdateItems).toHaveBeenCalledTimes(1)
    expect(env.queueUpdateItems.mock.calls[0][0].jump).toBe(5)
    expect(env.queueLoad).toHaveBeenCalledTimes(1)
    expect(document.querySelector('.current-time')?.textContent).toBe('00:00')

    // Trailing stale MEDIA_STATUS from Track 1 at 43s arrives before receiver switches ->
    // UI must ignore it and stay on Shuffled Track 6 at 00:00!
    await act(async () => {
      env.remotePlayer.currentTime = 43
      env.emitRemoteChange()
    })
    expect(store.getState().player.current.name).toBe('Shuffled Track 6')
    expect(document.querySelector('.current-time')?.textContent).toBe('00:00')

    // Receiver now switches to Track 6 (exposing 2-item slice [505, 506])
    await act(async () => {
      env.setMediaItems(fullLoadedItems.slice(5, 7))
      env.remotePlayer.playerState = 'PLAYING'
      env.remotePlayer.currentTime = 0
      env.remotePlayer.duration = 186
      env.remotePlayer.mediaInfo = {
        customData: {
          trackId: 'song-6',
          uuid: queue[5].uuid,
          isRadio: false,
        },
      }
      env.emitRemoteChange()
    })

    // Now click back on Track 1 (index 0): its itemId (500) WAS previously recorded in
    // uuidToItemIdRef, even though media.items currently only holds [505, 506].
    // Must bypass queueJumpToItem's silent drop via QueueJumpRequest{ currentItemId: 500 }!
    await act(async () => {
      fireEvent.click(queueRows[0])
    })
    expect(env.queueJumpToItem).not.toHaveBeenCalled()
    expect(env.queueUpdateItems).toHaveBeenCalledTimes(2)
    expect(env.queueUpdateItems.mock.calls[1][0].currentItemId).toBe(500)

    // Finally click Track 52 (index 51), which lies outside the [0, 40) Cast window ->
    // must immediately reload a fresh 40-track window centered around Track 52!
    await act(async () => {
      fireEvent.click(queueRows[51])
    })
    expect(env.queueLoad).toHaveBeenCalledTimes(2)
    const secondWinReq = env.queueLoad.mock.calls[1][0]
    expect(
      secondWinReq.items[secondWinReq.startIndex].media.customData.uuid,
    ).toBe(queue[51].uuid)
  })

  it('clears the queue and stops Chromecast cleanly from both the Play Queue trash button and the player close button without resurrecting cleared tracks', async () => {
    const profileSpy = vi
      .spyOn(decisionService, 'getProfile')
      .mockReturnValue({ name: 'chrome' })
    const env = createFakeCastEnvironment()
    const store = buildFullStore()

    store.dispatch(
      playTracks({ 'song-1': song1, 'song-2': song2, 'song-3': song3 }, [
        'song-1',
        'song-2',
        'song-3',
      ]),
    )

    render(
      <Provider store={store}>
        <MemoryRouter>
          <Player />
        </MemoryRouter>
      </Provider>,
    )

    await act(async () => {
      env.startSession('Kitchen Speaker')
    })

    const initialUuids = store.getState().player.queue.map((i) => i.uuid)
    await act(async () => {
      env.remotePlayer.playerState = 'PLAYING'
      env.remotePlayer.currentTime = 35
      env.remotePlayer.mediaInfo = {
        customData: {
          trackId: 'song-1',
          uuid: initialUuids[0],
          isRadio: false,
        },
      }
      env.emitRemoteChange()
    })

    // 1. Open Play Queue drawer and click Trash button (.audio-lists-panel-header-delete-btn)
    fireEvent.click(document.querySelector('.audio-lists-btn'))
    const trashBtn = document.querySelector(
      '.audio-lists-panel-header-delete-btn',
    )
    expect(trashBtn).not.toBeNull()

    await act(async () => {
      fireEvent.click(trashBtn)
      await Promise.resolve()
      await Promise.resolve()
    })

    expect(store.getState().player.queue).toEqual([])
    expect(store.getState().player.clear).toBe(true)
    expect(env.stop).toHaveBeenCalledTimes(1)

    // Subsequent Add to Queue must NOT resurrect any of the 3 cleared tracks!
    await act(async () => {
      store.dispatch(addTracks({ 'song-4': song4 }, ['song-4']))
      await Promise.resolve()
    })
    expect(store.getState().player.queue.map((t) => t.name)).toEqual([
      'House of Cards',
    ])
    expect(env.queueLoad).toHaveBeenCalledTimes(2)

    // 2. Click player Destroy button (.destroy-btn) while casting -> empties queue & stops Cast again
    const destroyBtn = document.querySelector('.destroy-btn')
    expect(destroyBtn).not.toBeNull()
    await act(async () => {
      fireEvent.click(destroyBtn)
      await Promise.resolve()
    })

    expect(store.getState().player.queue).toEqual([])
    expect(store.getState().player.clear).toBe(true)
    expect(env.stop).toHaveBeenCalledTimes(2)
    profileSpy.mockRestore()
  })

  it('refuses to adopt or resume another user Cast session after a full-page external-IdP logout navigation', async () => {
    const env = createFakeCastEnvironment()
    const aliceStore = buildFullStore()
    aliceStore.dispatch(playTracks({ 'song-1': song1 }, ['song-1']))

    localStorage.setItem('userId', 'user-alice')
    const aliceAudio = document.createElement('audio')

    const aliceView = render(
      <Provider store={aliceStore}>
        <CastProvider audioInstance={aliceAudio} enabled={true}>
          <CastButton />
        </CastProvider>
      </Provider>,
    )

    await act(async () => {
      env.startSession('Alice Lounge Speaker')
    })

    expect(localStorage.getItem('nd-cast-session-owner')).toBe('user-alice')
    expect(
      Object.prototype.hasOwnProperty.call(aliceAudio, 'currentTime'),
    ).toBe(true)

    // Same user reloading the page must still adopt their own active session
    const activeAliceSession = env.castContext.getCurrentSession()
    env.endCurrentSession.mockImplementationOnce(() => {})
    aliceView.unmount()
    env.endCurrentSession.mockClear()

    const aliceReloadAudio = document.createElement('audio')
    const aliceReloadView = render(
      <Provider store={aliceStore}>
        <CastProvider audioInstance={aliceReloadAudio} enabled={true}>
          <CastButton />
        </CastProvider>
      </Provider>,
    )
    expect(env.endCurrentSession).not.toHaveBeenCalled()
    expect(
      Object.prototype.hasOwnProperty.call(aliceReloadAudio, 'currentTime'),
    ).toBe(true)

    // Simulate external-IdP logout (full-page navigation: authProvider removes userId
    // without running a clean SESSION_ENDED, leaving ORIGIN_SCOPED session alive)
    env.endCurrentSession.mockImplementationOnce(() => {})
    aliceReloadView.unmount()
    env.endCurrentSession.mockClear()
    localStorage.removeItem('userId')

    // User Bob signs in on the same browser; mount-time getCurrentSession() sees Alice's session
    localStorage.setItem('userId', 'user-bob')
    const bobStore = buildFullStore()
    const bobAudio = document.createElement('audio')

    const bobView = render(
      <Provider store={bobStore}>
        <CastProvider audioInstance={bobAudio} enabled={true}>
          <CastButton />
        </CastProvider>
      </Provider>,
    )

    expect(env.endCurrentSession).toHaveBeenCalledTimes(1)
    expect(env.endCurrentSession).toHaveBeenCalledWith(true)
    expect(Object.prototype.hasOwnProperty.call(bobAudio, 'currentTime')).toBe(
      false,
    )
    fireEvent.click(screen.getByTestId('cast-button'))
    expect(screen.queryByTestId('cast-stop-button')).toBeNull()
    bobView.unmount()

    // Also verify the async SESSION_RESUMED path when getCurrentSession() is initially null at mount
    localStorage.setItem('nd-cast-session-owner', 'user-alice')
    localStorage.setItem('userId', 'user-bob')
    env.endCurrentSession.mockClear()
    env.castContext.getCurrentSession.mockReturnValue(null)

    const bobAudio2 = document.createElement('audio')
    render(
      <Provider store={bobStore}>
        <CastProvider audioInstance={bobAudio2} enabled={true}>
          <CastButton />
        </CastProvider>
      </Provider>,
    )

    env.castContext.getCurrentSession.mockReturnValue(activeAliceSession)
    const sessionHandler = env.castContext.addEventListener.mock.calls.find(
      (c) => c[0] === 'sessionstatechanged',
    )?.[1]
    await act(async () => {
      sessionHandler?.({ sessionState: 'SESSION_RESUMED' })
    })

    expect(env.endCurrentSession).toHaveBeenCalledTimes(1)
    expect(env.endCurrentSession).toHaveBeenCalledWith(true)
    expect(Object.prototype.hasOwnProperty.call(bobAudio2, 'currentTime')).toBe(
      false,
    )
  })
})
