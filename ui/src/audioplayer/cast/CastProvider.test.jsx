import React from 'react'
import { MemoryRouter } from 'react-router-dom'
import { Provider } from 'react-redux'
import { combineReducers, legacy_createStore as createStore } from 'redux'
import { render, screen, fireEvent, act, cleanup } from '@testing-library/react'
import { createTheme } from '@material-ui/core/styles'
import config from '../../config'
import subsonic from '../../subsonic'
import { playerReducer } from '../../reducers/playerReducer'
import { addTracks, playNext, playTracks } from '../../actions'
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
  const queueNext = vi.fn()
  const queuePrev = vi.fn()
  const queueJumpToItem = vi.fn((itemId, onSuccess) => {
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
    queueJumpToItem,
    queueSetRepeatMode,
    stop,
    requestSession,
    endCurrentSession,
    playOrPause,
    seek,
    setVolumeLevel,
    muteOrUnmute,
    getMediaItems: () => mediaItems,
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
    act(() => {
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

    // Initial QueueLoadRequest sent once at 47s offset
    expect(env.queueLoad).toHaveBeenCalledTimes(1)
    const initialLoadReq = env.queueLoad.mock.calls[0][0]
    expect(initialLoadReq.items).toHaveLength(2)
    expect(initialLoadReq.startIndex).toBe(0)
    expect(initialLoadReq.currentTime).toBe(47)

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

    act(() => {
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

  it('re-initializes Cast SDK listeners cleanly after logout (enabled=false) and login (enabled=true)', () => {
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

    // Simulate logout
    rerender(
      <Provider store={store}>
        <CastProvider audioInstance={fakeAudio} enabled={false}>
          <CastButton />
        </CastProvider>
      </Provider>,
    )
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

    act(() => {
      env.startSession('Office Speaker')
    })

    expect(env.queueLoad).toHaveBeenCalledTimes(1)
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

    act(() => {
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

    act(() => {
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
})
