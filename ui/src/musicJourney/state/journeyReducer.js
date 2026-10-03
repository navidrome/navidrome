/**
 * Redux state for Music Journey: the active journey, generation lifecycle,
 * persistent history, last preferences and skip-intelligence feedback.
 */
import {
  JOURNEY_DISMISS_ERROR,
  JOURNEY_ENDED,
  JOURNEY_GENERATE_FAILURE,
  JOURNEY_GENERATE_START,
  JOURNEY_GENERATE_SUCCESS,
  JOURNEY_HISTORY_DELETE,
  JOURNEY_HISTORY_TOGGLE_LIKE,
  JOURNEY_REGENERATED,
  JOURNEY_SET_PREFERENCES,
  JOURNEY_SKIPPED,
  JOURNEY_STATUS,
  JOURNEY_TRACK_COMPLETED,
  PLAYER_TRACK_ABANDONED,
} from './journeyActions'
import {
  HISTORY_LIMIT,
  journeyToHistoryEntry,
  loadPersistedJourney,
  persistJourney,
} from './persistence'
import { getTrackProfile } from '../engine/trackProfile'
import { normalizePreferences } from '../engine/types'

export const DEFAULT_JOURNEY_PREFERENCES = normalizePreferences({
  mood: 'chill',
  durationMinutes: 30,
  intensity: 0.4,
  discovery: 0.3,
})

const MAX_SKIP_EVENTS = 12

/**
 * Per-store initial state. Hydrating inside the default-parameter expression
 * means each Redux store instance reads localStorage exactly once (on the
 * @@redux/INIT dispatch), with no module-level flags that would leak state
 * between tests.
 */
export const buildInitialJourneyState = () => {
  const base = {
    active: null,
    generating: false,
    error: null,
    history: [],
    preferences: DEFAULT_JOURNEY_PREFERENCES,
    feedback: { consecutiveSkips: 0, events: [] },
    lastAbandoned: null,
  }
  const persisted = loadPersistedJourney()
  if (!persisted) return base
  return {
    ...base,
    history: persisted.history,
    preferences: persisted.preferences
      ? normalizePreferences(persisted.preferences)
      : base.preferences,
  }
}

export const initialJourneyState = buildInitialJourneyState()

const withPersistence = (state) => {
  persistJourney({ history: state.history, preferences: state.preferences })
  return state
}

const updateHistoryEntry = (history, id, patch) =>
  history.map((entry) => (entry.id === id ? { ...entry, ...patch } : entry))

const skipEventOf = (track, at) => {
  const profile = getTrackProfile(track, at)
  return {
    genreKey:
      String(track.genre || '')
        .trim()
        .toLowerCase() || null,
    artistKey:
      track.artistId ||
      String(track.artist || '')
        .trim()
        .toLowerCase() ||
      null,
    energy: profile.energy,
    at,
  }
}

export const journeyReducer = (
  previousState = buildInitialJourneyState(),
  payload,
) => {
  const state = previousState
  const { type, data } = payload || {}

  switch (type) {
    case JOURNEY_GENERATE_START:
      return {
        ...state,
        generating: true,
        error: null,
        preferences: normalizePreferences(data.preferences),
      }

    case JOURNEY_GENERATE_SUCCESS: {
      const entry = journeyToHistoryEntry(data.journey)
      return withPersistence({
        ...state,
        generating: false,
        error: null,
        active: {
          journey: data.journey,
          status: 'playing',
          startedAt: Date.now(),
        },
        history: [entry, ...state.history].slice(0, HISTORY_LIMIT),
      })
    }

    case JOURNEY_GENERATE_FAILURE:
      return { ...state, generating: false, error: data.error }

    case JOURNEY_REGENERATED: {
      if (!state.active) return state
      const { journey } = data
      const patch = {
        trackIds: journey.tracks.map((t) => t.track.id).slice(0, 500),
        trackCount: journey.tracks.length,
        estimatedDuration: journey.estimatedDuration,
        mood: journey.preferences.mood,
        durationMinutes: journey.preferences.durationMinutes,
        intensity: journey.preferences.intensity,
        discovery: journey.preferences.discovery,
        version: journey.version,
      }
      return withPersistence({
        ...state,
        active: { ...state.active, journey },
        preferences: normalizePreferences(journey.preferences),
        history: updateHistoryEntry(state.history, journey.id, patch),
      })
    }

    case JOURNEY_STATUS: {
      if (!state.active) return state
      return { ...state, active: { ...state.active, status: data.status } }
    }

    case JOURNEY_ENDED: {
      if (!state.active) return state
      const status = data.reason || 'stopped'
      return withPersistence({
        ...state,
        active: null,
        history: updateHistoryEntry(state.history, state.active.journey.id, {
          status,
        }),
      })
    }

    case JOURNEY_SKIPPED: {
      const at = Date.now()
      const events = [...state.feedback.events, skipEventOf(data.track, at)]
        // drop events older than the engine's decay window while we're here
        .filter((ev) => at - ev.at < 30 * 60 * 1000)
        .slice(-MAX_SKIP_EVENTS)
      let history = state.history
      if (state.active) {
        const journeyId = state.active.journey.id
        history = history.map((entry) =>
          entry.id === journeyId
            ? { ...entry, skipped: (entry.skipped || 0) + 1 }
            : entry,
        )
      }
      return withPersistence({
        ...state,
        history,
        feedback: {
          consecutiveSkips: state.feedback.consecutiveSkips + 1,
          events,
        },
      })
    }

    case JOURNEY_TRACK_COMPLETED:
      if (!state.feedback.consecutiveSkips) return state
      return { ...state, feedback: { ...state.feedback, consecutiveSkips: 0 } }

    case JOURNEY_SET_PREFERENCES:
      return withPersistence({
        ...state,
        preferences: normalizePreferences(data.preferences),
      })

    case JOURNEY_HISTORY_DELETE:
      return withPersistence({
        ...state,
        history: state.history.filter((entry) => entry.id !== data.id),
      })

    case JOURNEY_HISTORY_TOGGLE_LIKE:
      return withPersistence({
        ...state,
        history: state.history.map((entry) =>
          entry.id === data.id ? { ...entry, liked: !entry.liked } : entry,
        ),
      })

    case JOURNEY_DISMISS_ERROR:
      return { ...state, error: null }

    case PLAYER_TRACK_ABANDONED:
      return { ...state, lastAbandoned: data }

    default:
      return state
  }
}
