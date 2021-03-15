import { SET_NOTIFICATIONS_STATE, SET_MOBILE_RESOLUTION } from '../actions'

const initialState = {
  notifications: false,
}

export const settingsReducer = (previousState = initialState, payload) => {
  const { type, data } = payload
  switch (type) {
    case SET_NOTIFICATIONS_STATE:
      return {
        ...previousState,
        notifications: data,
      }
    case SET_MOBILE_RESOLUTION:
      return {
        ...previousState,
        resolution: data,
      }

    default:
      return previousState
  }
}
