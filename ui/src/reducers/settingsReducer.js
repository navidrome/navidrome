import { SET_NOTIFICATIONS_STATE, SET_VISUALIZATION_STATE } from '../actions'

const initialState = {
  notifications: false,
  visualization: false,
}

export const settingsReducer = (previousState = initialState, payload) => {
  const { type, data } = payload
  switch (type) {
    case SET_NOTIFICATIONS_STATE:
      return {
        ...previousState,
        notifications: data,
      }
    case SET_VISUALIZATION_STATE:
      return {
        ...previousState,
        visualization: data,
      }
    default:
      return previousState
  }
}
