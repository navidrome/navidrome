import { SET_NOTIFICATIONS_STATE, SET_TOGGLEABLE_FIELDS } from '../actions'

const initialState = {
  notifications: false,
  toggleableFields: {},
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
    default:
      return previousState
  }
}
