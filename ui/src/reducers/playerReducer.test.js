import { describe, expect, it, vi } from 'vitest'
import { playerReducer } from './playerReducer'
import { PLAYER_SET_TRACK, PLAYER_UPDATE_LYRIC } from '../actions'

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
