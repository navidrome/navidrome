import { describe, it, expect } from 'vitest'
import { playerReducer } from './playerReducer'
import {
  PLAYER_SYNC_QUEUE,
  PLAYER_CURRENT,
  PLAYER_REFRESH_QUEUE,
  EVENT_RADIO_NOW_PLAYING,
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

  describe('radioNowPlaying event', () => {
    const radioItem = {
      trackId: 'rd-1',
      uuid: 'radio-uuid',
      name: 'Test Station',
      musicSrc: 'https://stream.example.test/radio',
      isRadio: true,
      song: {
        id: 'rd-1',
        title: 'Test Station',
        artist: 'Test Station',
        album: 'https://stream.example.test/radio',
        streamUrl: 'https://stream.example.test/radio',
      },
    }

    it('updates the active radio with live title metadata', () => {
      const state = {
        queue: [radioItem],
        current: radioItem,
        savedPlayIndex: 0,
        clear: false,
        volume: 1,
      }
      const result = playerReducer(state, {
        type: EVENT_RADIO_NOW_PLAYING,
        data: {
          radioId: 'rd-1',
          title: 'Artist - Title',
          updatedAt: '2026-06-30T12:34:56Z',
        },
      })

      expect(result.current.radioTitle).toBe('Artist - Title')
      expect(result.current.song.radioTitle).toBe('Artist - Title')
      expect(result.current.name).toBe('Test Station')
      expect(result.current.song.title).toBe('Test Station')
      expect(result.current.musicSrc).toBe('https://stream.example.test/radio')
      expect(result.queue[0].radioTitle).toBe('Artist - Title')
    })

    it('ignores events for non-active radios', () => {
      const state = {
        queue: [radioItem],
        current: radioItem,
        savedPlayIndex: 0,
        clear: false,
        volume: 1,
      }
      const result = playerReducer(state, {
        type: EVENT_RADIO_NOW_PLAYING,
        data: { radioId: 'rd-2', title: 'Other Station - Title' },
      })

      expect(result).toBe(state)
    })

    it('clears live title metadata when the title is empty', () => {
      const liveRadio = {
        ...radioItem,
        radioTitle: 'Old Title',
        song: { ...radioItem.song, radioTitle: 'Old Title' },
      }
      const state = {
        queue: [liveRadio],
        current: liveRadio,
        savedPlayIndex: 0,
        clear: false,
        volume: 1,
      }
      const result = playerReducer(state, {
        type: EVENT_RADIO_NOW_PLAYING,
        data: { radioId: 'rd-1', title: '' },
      })

      expect(result.current.radioTitle).toBeUndefined()
      expect(result.current.song.radioTitle).toBeUndefined()
      expect(result.current.song.title).toBe('Test Station')
      expect(result.current.musicSrc).toBe('https://stream.example.test/radio')
    })

    it('updates the queued active radio before current is set', () => {
      const state = {
        queue: [radioItem],
        current: {},
        savedPlayIndex: 0,
        clear: false,
        volume: 1,
      }
      const result = playerReducer(state, {
        type: EVENT_RADIO_NOW_PLAYING,
        data: { radioId: 'rd-1', title: 'Artist - Title' },
      })

      expect(result.current).toEqual({})
      expect(result.queue[0].radioTitle).toBe('Artist - Title')
    })
  })
})
