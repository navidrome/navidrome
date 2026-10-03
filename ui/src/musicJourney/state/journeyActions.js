export const JOURNEY_GENERATE_START = 'JOURNEY_GENERATE_START'
export const JOURNEY_GENERATE_SUCCESS = 'JOURNEY_GENERATE_SUCCESS'
export const JOURNEY_GENERATE_FAILURE = 'JOURNEY_GENERATE_FAILURE'
export const JOURNEY_REGENERATED = 'JOURNEY_REGENERATED'
export const JOURNEY_STATUS = 'JOURNEY_STATUS'
export const JOURNEY_ENDED = 'JOURNEY_ENDED'
export const JOURNEY_SKIPPED = 'JOURNEY_SKIPPED'
export const JOURNEY_TRACK_COMPLETED = 'JOURNEY_TRACK_COMPLETED'
export const JOURNEY_SET_PREFERENCES = 'JOURNEY_SET_PREFERENCES'
export const JOURNEY_HISTORY_DELETE = 'JOURNEY_HISTORY_DELETE'
export const JOURNEY_HISTORY_TOGGLE_LIKE = 'JOURNEY_HISTORY_TOGGLE_LIKE'
export const JOURNEY_DISMISS_ERROR = 'JOURNEY_DISMISS_ERROR'

// Generic player action, re-exported here for convenience.
export { PLAYER_TRACK_ABANDONED, playerTrackAbandoned } from '../../actions'

export const journeyGenerateStart = (preferences) => ({
  type: JOURNEY_GENERATE_START,
  data: { preferences },
})

export const journeyGenerateSuccess = (journey) => ({
  type: JOURNEY_GENERATE_SUCCESS,
  data: { journey },
})

export const journeyGenerateFailure = (error) => ({
  type: JOURNEY_GENERATE_FAILURE,
  data: { error },
})

export const journeyRegenerated = (journey) => ({
  type: JOURNEY_REGENERATED,
  data: { journey },
})

export const journeyStatus = (status) => ({
  type: JOURNEY_STATUS,
  data: { status },
})

export const journeyEnded = (reason) => ({
  type: JOURNEY_ENDED,
  data: { reason },
})

export const journeySkipped = (track) => ({
  type: JOURNEY_SKIPPED,
  data: { track },
})

export const journeyTrackCompleted = () => ({
  type: JOURNEY_TRACK_COMPLETED,
})

export const setJourneyPreferences = (preferences) => ({
  type: JOURNEY_SET_PREFERENCES,
  data: { preferences },
})

export const deleteJourneyHistoryEntry = (id) => ({
  type: JOURNEY_HISTORY_DELETE,
  data: { id },
})

export const toggleJourneyLike = (id) => ({
  type: JOURNEY_HISTORY_TOGGLE_LIKE,
  data: { id },
})

export const dismissJourneyError = () => ({
  type: JOURNEY_DISMISS_ERROR,
})
