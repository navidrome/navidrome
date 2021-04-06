export const SET_NOTIFICATIONS_STATE = 'SET_NOTIFICATIONS_STATE'
export const SET_VISUALIZATION_STATE = 'SET_VISUALIZATION_STATE'

export const setNotificationsState = (enabled) => ({
  type: SET_NOTIFICATIONS_STATE,
  data: enabled,
})

export const setVisualizationState = (enabled) => ({
  type: SET_VISUALIZATION_STATE,
  data: enabled,
})
