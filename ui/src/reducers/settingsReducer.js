import {
  SET_NOTIFICATIONS_STATE,
  SET_OMITTED_FIELDS,
  SET_TOGGLEABLE_FIELDS,
  SET_PLAYER_RATING_CONTROL,
} from '../actions'

const initialState = {
  notifications: false,
  toggleableFields: {},
  omittedFields: {},
  playerRatingControl: localStorage.getItem('playerRating') ?? 'love',
}

export const settingsReducer = (previousState = initialState, payload) => {
  const { type, data } = payload
  switch (type) {
    case SET_NOTIFICATIONS_STATE:
      return {
        ...previousState,
        notifications: data,
      }
    case SET_TOGGLEABLE_FIELDS:
      return {
        ...previousState,
        toggleableFields: {
          ...previousState.toggleableFields,
          ...data,
        },
      }
    case SET_OMITTED_FIELDS:
      return {
        ...previousState,
        omittedFields: {
          ...previousState.omittedFields,
          ...data,
        },
      }
    case SET_PLAYER_RATING_CONTROL:
      localStorage.setItem('playerRating', data)
      return {
        ...previousState,
        playerRatingControl: data,
      }
    default:
      return previousState
  }
}
