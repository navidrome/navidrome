import { describe, expect, it, vi } from 'vitest'
import { playerReducer } from './playerReducer'
import {
  PLAYER_CURRENT,
  PLAYER_REFRESH_QUEUE,
  PLAYER_SET_TRACK,
  PLAYER_SYNC_QUEUE,
  PLAYER_UPDATE_LYRIC,
} from '../actions'

vi.mock('uuid', () => ({
  v4: () => 'test-uuid',
}))

vi.mock('../subsonic', () => ({
  default: {
    streamUrl: vi.fn((id) => `/rest/stream?id=${id}`),
    getCoverArtUrl: vi.fn(() => '/rest/getCoverArt?id=test'),
  },
}))

describe('playerReducer', () => {
  describe('pending track selection survives SYNC_QUEUE and premature CURRENT', () => {
    // Simulates the real sequence when clicking a new song while one is playing:
    // 1. PLAYER_PLAY_TRACKS sets playIndex and clear
    // 2. PLAYER_SYNC_QUEUE fires when music player syncs its internal queue
    // 3. PLAYER_CURRENT fires for the OLD still-playing track
    // 4. PLAYER_CURRENT fires for the NEW track (player switched)
    const stateAfterPlayTracks = {
      queue: [
        { trackId: 's1', uuid: 'aaa', name: 'Song 1' },
        { trackId: 's2', uuid: 'bbb', name: 'Song 2' },
        { trackId: 's3', uuid: 'ccc', name: 'Song 3' },
      ],
      current: { uuid: 'ccc', name: 'Song 3' },
      playIndex: 0, // user clicked Song 1
      savedPlayIndex: 2, // Song 3 was playing
      clear: true,
      volume: 1,
    }

    it('SYNC_QUEUE preserves pending playIndex and clear', () => {
      const newQueue = [
        { trackId: 's1', uuid: 'xxx', name: 'Song 1' },
        { trackId: 's2', uuid: 'yyy', name: 'Song 2' },
        { trackId: 's3', uuid: 'zzz', name: 'Song 3' },
      ]
      const action = {
        type: PLAYER_SYNC_QUEUE,
        data: { audioInfo: {}, audioLists: newQueue },
      }
      const result = playerReducer(stateAfterPlayTracks, action)
      expect(result.playIndex).toBe(0)
      expect(result.clear).toBe(true)
      expect(result.queue).toBe(newQueue)
    })

    it('SYNC_QUEUE clears playIndex when no pending selection', () => {
      const stateNoPending = { ...stateAfterPlayTracks, playIndex: undefined }
      const action = {
        type: PLAYER_SYNC_QUEUE,
        data: { audioInfo: {}, audioLists: stateNoPending.queue },
      }
      const result = playerReducer(stateNoPending, action)
      expect(result.playIndex).toBeUndefined()
      expect(result.clear).toBe(false)
    })

    it('CURRENT for old track preserves pending playIndex', () => {
      const stateAfterSync = {
        ...stateAfterPlayTracks,
        queue: [
          { trackId: 's1', uuid: 'xxx', name: 'Song 1' },
          { trackId: 's2', uuid: 'yyy', name: 'Song 2' },
          { trackId: 's3', uuid: 'zzz', name: 'Song 3' },
        ],
      }
      const action = {
        type: PLAYER_CURRENT,
        data: { uuid: 'zzz', name: 'Song 3', volume: 1 },
      }
      const result = playerReducer(stateAfterSync, action)
      expect(result.playIndex).toBe(0)
      expect(result.clear).toBe(true)
      expect(result.savedPlayIndex).toBe(2)
    })

    it('CURRENT for correct track consumes pending playIndex', () => {
      const stateAfterSync = {
        ...stateAfterPlayTracks,
        queue: [
          { trackId: 's1', uuid: 'xxx', name: 'Song 1' },
          { trackId: 's2', uuid: 'yyy', name: 'Song 2' },
          { trackId: 's3', uuid: 'zzz', name: 'Song 3' },
        ],
      }
      const action = {
        type: PLAYER_CURRENT,
        data: { uuid: 'xxx', name: 'Song 1', volume: 1 },
      }
      const result = playerReducer(stateAfterSync, action)
      expect(result.playIndex).toBeUndefined()
      expect(result.clear).toBe(false)
      expect(result.savedPlayIndex).toBe(0)
      expect(result.current.name).toBe('Song 1')
    })
  })

  describe('PLAYER_REFRESH_QUEUE', () => {
    it('clamps negative savedPlayIndex to 0', () => {
      const state = {
        queue: [
          { trackId: 'song-1', musicSrc: 'old-url', uuid: 'a' },
          { trackId: 'song-2', musicSrc: 'old-url', uuid: 'b' },
        ],
        savedPlayIndex: -1,
        current: {},
        clear: false,
        volume: 1,
      }
      const action = { type: PLAYER_REFRESH_QUEUE, data: {} }
      const result = playerReducer(state, action)
      expect(result.playIndex).toBe(0)
    })

    it('preserves valid savedPlayIndex', () => {
      const state = {
        queue: [
          { trackId: 'song-1', musicSrc: 'old-url', uuid: 'a' },
          { trackId: 'song-2', musicSrc: 'old-url', uuid: 'b' },
        ],
        savedPlayIndex: 1,
        current: {},
        clear: false,
        volume: 1,
      }
      const action = { type: PLAYER_REFRESH_QUEUE, data: {} }
      const result = playerReducer(state, action)
      expect(result.playIndex).toBe(1)
    })

    it('uses savedPlayIndex of 0 correctly', () => {
      const state = {
        queue: [{ trackId: 'song-1', musicSrc: 'old-url', uuid: 'a' }],
        savedPlayIndex: 0,
        current: {},
        clear: false,
        volume: 1,
      }
      const action = { type: PLAYER_REFRESH_QUEUE, data: {} }
      const result = playerReducer(state, action)
      expect(result.playIndex).toBe(0)
    })
  })

  it('maps embedded synced lyrics to LRC text', () => {
    const lyrics = JSON.stringify([
      {
        lang: 'eng',
        synced: true,
        line: [{ start: 1000, value: 'Line one' }],
      },
      {
        lang: 'eng',
        synced: false,
        line: [{ value: 'Unsynced line' }],
      },
    ])

    const state = playerReducer(undefined, {
      type: PLAYER_SET_TRACK,
      data: {
        id: 'song-1',
        title: 'Test Song',
        artist: 'Test Artist',
        album: 'Test Album',
        duration: 60,
        lyrics,
      },
    })

    expect(state.queue).toHaveLength(1)
    expect(state.queue[0].lyric).toBe('[00:01.00] Line one\n')
  })

  it('updates queue lyric by track id', () => {
    const initial = playerReducer(undefined, {
      type: PLAYER_SET_TRACK,
      data: {
        id: 'song-1',
        title: 'Test Song',
        artist: 'Test Artist',
        album: 'Test Album',
        duration: 60,
      },
    })

    const updated = playerReducer(initial, {
      type: PLAYER_UPDATE_LYRIC,
      data: {
        trackId: 'song-1',
        lyric: '[00:01.00] Updated lyric\n',
      },
    })

    expect(updated.queue[0].lyric).toBe('[00:01.00] Updated lyric\n')
  })

  it('returns same state when lyric update does not match any track', () => {
    const initial = playerReducer(undefined, {
      type: PLAYER_SET_TRACK,
      data: {
        id: 'song-1',
        title: 'Test Song',
        artist: 'Test Artist',
        album: 'Test Album',
        duration: 60,
      },
    })

    const updated = playerReducer(initial, {
      type: PLAYER_UPDATE_LYRIC,
      data: {
        trackId: 'missing-track',
        lyric: '[00:01.00] Updated lyric\n',
      },
    })

    expect(updated).toBe(initial)
  })
})
