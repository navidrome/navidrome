export const SET_NOTIFICATIONS_STATE = 'SET_NOTIFICATIONS_STATE'

export const setNotificationsState = (enabled) => ({
  type: SET_NOTIFICATIONS_STATE,
  data: enabled,
})
