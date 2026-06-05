import { describe, it, expect } from 'vitest'
import { playerReducer } from './playerReducer'
import {
  PLAYER_SYNC_QUEUE,
  PLAYER_CURRENT,
  PLAYER_REFRESH_QUEUE,
} from '../actions'

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
      // After SYNC_QUEUE, queue has new UUIDs. The old track's UUID (zzz)
      // is at index 2, but playIndex is 0. This is a premature callback.
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
      expect(result.savedPlayIndex).toBe(2) // preserved from before
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
      // Player switched to Song 1 (uuid 'xxx', index 0 == playIndex)
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

  describe('play new album after closing player (issue #5440)', () => {
    it('SYNC_QUEUE preserves pending playIndex=0 after clearQueue', () => {
      // Scenario: user plays album A, advances to track 3, closes player,
      // then plays album B. After clearQueue, savedPlayIndex=0.
      // PLAYER_PLAY_TRACKS sets playIndex=0. SYNC_QUEUE must NOT clear it.
      const stateAfterClearThenPlay = {
        queue: [
          { trackId: 'b1', uuid: 'u1', name: 'B Song 1' },
          { trackId: 'b2', uuid: 'u2', name: 'B Song 2' },
          { trackId: 'b3', uuid: 'u3', name: 'B Song 3' },
        ],
        current: {},
        playIndex: 0,
        savedPlayIndex: 0, // reset by clearQueue
        clear: true,
        volume: 1,
      }

      const action = {
        type: PLAYER_SYNC_QUEUE,
        data: {
          audioInfo: {},
          audioLists: stateAfterClearThenPlay.queue,
        },
      }
      const result = playerReducer(stateAfterClearThenPlay, action)
      expect(result.playIndex).toBe(0)
      expect(result.clear).toBe(true)
    })

    it('CURRENT for wrong track preserves pending playIndex=0 after clearQueue', () => {
      // The music player fires onAudioPlay for the old track (at index 3)
      // before switching to the new track at index 0.
      const stateAfterClearThenPlay = {
        queue: [
          { trackId: 'b1', uuid: 'u1', name: 'B Song 1' },
          { trackId: 'b2', uuid: 'u2', name: 'B Song 2' },
          { trackId: 'b3', uuid: 'u3', name: 'B Song 3' },
          { trackId: 'b4', uuid: 'u4', name: 'B Song 4' },
        ],
        current: {},
        playIndex: 0,
        savedPlayIndex: 0,
        clear: true,
        volume: 1,
      }

      // Player reports track at index 3 as current (stale callback)
      const action = {
        type: PLAYER_CURRENT,
        data: { uuid: 'u4', name: 'B Song 4', volume: 1 },
      }
      const result = playerReducer(stateAfterClearThenPlay, action)
      expect(result.playIndex).toBe(0)
      expect(result.clear).toBe(true)
    })

    it('CURRENT for correct track consumes pending playIndex=0', () => {
      const stateAfterClearThenPlay = {
        queue: [
          { trackId: 'b1', uuid: 'u1', name: 'B Song 1' },
          { trackId: 'b2', uuid: 'u2', name: 'B Song 2' },
        ],
        current: {},
        playIndex: 0,
        savedPlayIndex: 0,
        clear: true,
        volume: 1,
      }

      // Player confirms it switched to track at index 0
      const action = {
        type: PLAYER_CURRENT,
        data: { uuid: 'u1', name: 'B Song 1', volume: 1 },
      }
      const result = playerReducer(stateAfterClearThenPlay, action)
      expect(result.playIndex).toBeUndefined()
      expect(result.clear).toBe(false)
      expect(result.savedPlayIndex).toBe(0)
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
})
