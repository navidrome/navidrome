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

  // Regression context for the 0.62 report:
  //   "When I play a song, it will sometimes skip to a random song, attempt to
  //    play said song, and then immediately back to the song I chose."
  //
  // Root cause: commit 2307a64da (#5441, 0.62-only) widened reduceSyncQueue's
  // hasPendingSwitch to also keep playIndex alive whenever clear=true. This let
  // a pending playIndex survive a SYNC_QUEUE that carries the OLD queue (the
  // music player library can emit a stale onAudioListsChange mid-transition),
  // leaving playIndex pointing at a track in the WRONG queue. Downstream, the
  // library's updatePlayIndex(playIndex) played that stale track for an instant
  // ("random song") before the new queue settled ("back to the song I chose").
  // The race was timing-dependent, hence "sometimes". The FIX block below guards
  // reduceSyncQueue against adopting a stale list; these tests pin the delta.
  describe('0.62 regression context', () => {
    // 0.61 logic, for contrast: a switch is pending ONLY when the index moved.
    const hasPendingSwitch061 = (s) =>
      s.playIndex != null && s.playIndex !== s.savedPlayIndex

    it('0.61 vs 0.62 differ ONLY when playIndex===savedPlayIndex && clear', () => {
      const hasPendingSwitch062 = (s) =>
        s.playIndex != null && (s.clear || s.playIndex !== s.savedPlayIndex)

      const cases = [
        { playIndex: 0, savedPlayIndex: 0, clear: true }, // the regression case
        { playIndex: 0, savedPlayIndex: 3, clear: true },
        { playIndex: 2, savedPlayIndex: 2, clear: false },
        { playIndex: 5, savedPlayIndex: 0, clear: false },
      ]
      const diffs = cases.filter(
        (c) => hasPendingSwitch061(c) !== hasPendingSwitch062(c),
      )
      expect(diffs).toEqual([{ playIndex: 0, savedPlayIndex: 0, clear: true }])
    })
  })

  // FIX for the 0.62 stale-playIndex regression. A SYNC_QUEUE that does NOT
  // contain the pending track (state.queue[playIndex]) is a stale snapshot of
  // the library's previous queue. Adopting it would point playIndex at a track
  // the user never chose. The fix ignores such a stale sync: it keeps our
  // intended queue and keeps the pending switch alive so the next (correct)
  // sync — once the library finishes loading the new queue — is adopted.
  describe('FIX: SYNC_QUEUE ignores a stale list missing the pending track', () => {
    it('keeps the intended queue and pending switch when sync is stale', () => {
      const intendedQueue = [{ trackId: 'b0', uuid: 'B0', name: 'Chosen Song' }]
      const stateAfterSetTrack = {
        queue: intendedQueue,
        current: {},
        playIndex: 0,
        savedPlayIndex: 0,
        clear: true,
        volume: 1,
      }
      const staleOldQueue = [
        { trackId: 'a0', uuid: 'A0', name: 'Old Song 0' },
        { trackId: 'a1', uuid: 'A1', name: 'Old Song 1' },
        { trackId: 'a2', uuid: 'A2', name: 'Old Song 2' },
      ]
      const result = playerReducer(stateAfterSetTrack, {
        type: PLAYER_SYNC_QUEUE,
        data: { audioInfo: {}, audioLists: staleOldQueue },
      })

      // Intended queue is preserved; the stale old queue is NOT adopted.
      expect(result.queue).toBe(intendedQueue)
      // Pending switch stays alive for the next, correct sync.
      expect(result.playIndex).toBe(0)
      expect(result.clear).toBe(true)
      // playIndex now points at the chosen track, never an old one.
      expect(result.queue[result.playIndex].name).toBe('Chosen Song')
    })

    it('adopts a valid sync that contains the pending track (by trackId)', () => {
      // Mirrors the real flow: state.queue and the synced list share trackIds
      // but have library-assigned uuids. The pending track (trackId s1) is
      // present, so the sync is valid and must be adopted.
      const stateAfterPlayTracks = {
        queue: [
          { trackId: 's1', uuid: 'aaa', name: 'Song 1' },
          { trackId: 's2', uuid: 'bbb', name: 'Song 2' },
          { trackId: 's3', uuid: 'ccc', name: 'Song 3' },
        ],
        current: { uuid: 'ccc', name: 'Song 3' },
        playIndex: 0,
        savedPlayIndex: 2,
        clear: true,
        volume: 1,
      }
      const validSync = [
        { trackId: 's1', uuid: 'xxx', name: 'Song 1' },
        { trackId: 's2', uuid: 'yyy', name: 'Song 2' },
        { trackId: 's3', uuid: 'zzz', name: 'Song 3' },
      ]
      const result = playerReducer(stateAfterPlayTracks, {
        type: PLAYER_SYNC_QUEUE,
        data: { audioInfo: {}, audioLists: validSync },
      })
      expect(result.queue).toBe(validSync)
      expect(result.playIndex).toBe(0)
      expect(result.clear).toBe(true)
    })

    it('adopts the sync normally when no switch is pending', () => {
      const stateNoPending = {
        queue: [{ trackId: 's1', uuid: 'aaa', name: 'Song 1' }],
        current: {},
        playIndex: undefined,
        savedPlayIndex: 0,
        clear: false,
        volume: 1,
      }
      const reordered = [{ trackId: 's1', uuid: 'aaa', name: 'Song 1' }]
      const result = playerReducer(stateNoPending, {
        type: PLAYER_SYNC_QUEUE,
        data: { audioInfo: {}, audioLists: reordered },
      })
      expect(result.queue).toBe(reordered)
      expect(result.playIndex).toBeUndefined()
      expect(result.clear).toBe(false)
    })
  })
})
