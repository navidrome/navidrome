import { SET_NOTIFICATIONS_STATE } from '../actions'

const initialState = {
  notifications: false,
}

export const settingsReducer = (previousState = initialState, payload) => {
  const { type, data } = payload
  switch (type) {
    case SET_NOTIFICATIONS_STATE:
      return {
        ...previousState,
        notifications: data
      }
    default:
      return previousState
  }
}
