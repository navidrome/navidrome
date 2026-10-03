import { describe, it, expect, beforeEach, vi } from 'vitest'
import { buildInitialJourneyState, journeyReducer } from './journeyReducer'
import {
  journeyEnded,
  journeyGenerateFailure,
  journeyGenerateStart,
  journeyGenerateSuccess,
  journeyRegenerated,
  journeySkipped,
  journeyStatus,
  journeyTrackCompleted,
  deleteJourneyHistoryEntry,
  toggleJourneyLike,
} from './journeyActions'
import { playerTrackAbandoned } from '../../actions'

const NOW = Date.UTC(2026, 4, 5)

const fakeJourney = (overrides = {}) => ({
  id: 'j-1',
  title: 'Midnight Drive',
  preferences: {
    mood: 'chill',
    durationMinutes: 30,
    intensity: 0.4,
    discovery: 0.3,
  },
  tracks: [
    {
      track: {
        id: 't1',
        title: 'A',
        artist: 'X',
        genre: 'Jazz',
        duration: 200,
      },
      score: 0.8,
      stage: { id: 'arrival', name: 'Arrival' },
    },
    {
      track: {
        id: 't2',
        title: 'B',
        artist: 'Y',
        genre: 'Jazz',
        duration: 220,
      },
      score: 0.7,
      stage: { id: 'arrival', name: 'Arrival' },
    },
    {
      track: {
        id: 't3',
        title: 'C',
        artist: 'Z',
        genre: 'Rock',
        duration: 180,
      },
      score: 0.6,
      stage: { id: 'flow', name: 'Flow' },
    },
  ],
  estimatedDuration: 600,
  narrative: [
    { id: 'arrival', name: 'Arrival' },
    { id: 'flow', name: 'Flow' },
  ],
  truncated: false,
  version: 1,
  ...overrides,
})

describe('journeyReducer', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.restoreAllMocks()
  })

  const start = (state = buildInitialJourneyState()) => state

  it('generation lifecycle: start -> success activates journey and records history', () => {
    let state = journeyReducer(
      start(),
      journeyGenerateStart(fakeJourney().preferences),
    )
    expect(state.generating).toBe(true)

    state = journeyReducer(state, journeyGenerateSuccess(fakeJourney()))
    expect(state.generating).toBe(false)
    expect(state.active.journey.id).toBe('j-1')
    expect(state.active.status).toBe('playing')
    expect(state.history).toHaveLength(1)
    expect(state.history[0].title).toBe('Midnight Drive')
    expect(state.history[0].trackIds).toEqual(['t1', 't2', 't3'])
  })

  it('generation failure stores a structured error, never a stack trace', () => {
    let state = journeyReducer(start(), journeyGenerateStart({}))
    state = journeyReducer(
      state,
      journeyGenerateFailure({
        code: 'NOT_ENOUGH_TRACKS',
        details: { available: 2 },
      }),
    )
    expect(state.generating).toBe(false)
    expect(state.error.code).toBe('NOT_ENOUGH_TRACKS')
    expect(state.active).toBeNull()
  })

  it('skip feedback accumulates and completed tracks reset the streak', () => {
    let state = journeyReducer(start(), journeyGenerateSuccess(fakeJourney()))
    const skippedTrack = state.active.journey.tracks[2].track

    state = journeyReducer(state, journeySkipped(skippedTrack))
    state = journeyReducer(state, journeySkipped(skippedTrack))
    expect(state.feedback.consecutiveSkips).toBe(2)
    expect(state.feedback.events).toHaveLength(2)
    expect(state.feedback.events[0].genreKey).toBe('rock')
    expect(state.history[0].skipped).toBe(2)

    state = journeyReducer(state, journeyTrackCompleted())
    expect(state.feedback.consecutiveSkips).toBe(0)
    // events stay around for the decay window
    expect(state.feedback.events).toHaveLength(2)
  })

  it('PLAYER_TRACK_ABANDONED is recorded for the watcher', () => {
    const state = journeyReducer(
      start(),
      playerTrackAbandoned({ trackId: 't9', positionMs: 42000 }),
    )
    expect(state.lastAbandoned.trackId).toBe('t9')
    expect(state.lastAbandoned.positionMs).toBe(42000)
  })

  it('regeneration replaces the journey but keeps identity and updates history', () => {
    let state = journeyReducer(start(), journeyGenerateSuccess(fakeJourney()))
    const rebuilt = fakeJourney({
      version: 2,
      tracks: [
        ...fakeJourney().tracks.slice(0, 1),
        {
          track: {
            id: 't9',
            title: 'N',
            artist: 'N',
            genre: 'Pop',
            duration: 100,
          },
          score: 0.5,
          stage: { id: 'flow', name: 'Flow' },
        },
      ],
      estimatedDuration: 300,
    })
    state = journeyReducer(state, journeyRegenerated(rebuilt))
    expect(state.active.journey.version).toBe(2)
    expect(state.history[0].trackIds).toEqual(['t1', 't9'])
    expect(state.history[0].trackCount).toBe(2)
  })

  it('status updates and journey end write through to history', () => {
    let state = journeyReducer(start(), journeyGenerateSuccess(fakeJourney()))
    state = journeyReducer(state, journeyStatus('paused'))
    expect(state.active.status).toBe('paused')

    state = journeyReducer(state, journeyEnded('completed'))
    expect(state.active).toBeNull()
    expect(state.history[0].status).toBe('completed')
  })

  it('history delete and like actions', () => {
    let state = journeyReducer(start(), journeyGenerateSuccess(fakeJourney()))
    state = journeyReducer(state, toggleJourneyLike('j-1'))
    expect(state.history[0].liked).toBe(true)
    state = journeyReducer(state, deleteJourneyHistoryEntry('j-1'))
    expect(state.history).toHaveLength(0)
  })

  it('persists history and preferences and rehydrates a fresh state', () => {
    let state = journeyReducer(start(), journeyGenerateSuccess(fakeJourney()))
    state = journeyReducer(state, journeyEnded('completed'))

    const rehydrated = buildInitialJourneyState()
    expect(rehydrated.history).toHaveLength(1)
    expect(rehydrated.history[0].id).toBe('j-1')
    expect(rehydrated.preferences.mood).toBe('chill')
  })

  it('ignores corrupted persisted data', () => {
    localStorage.setItem('musicJourney.v1', '{not json')
    const state = buildInitialJourneyState()
    expect(state.history).toEqual([])
  })
})
