export const SET_NOTIFICATIONS_STATE = 'SET_NOTIFICATIONS_STATE'
export const SET_MOBILE_RESOLUTION = 'SET_MOBILE_RESOLUTION'

export const setNotificationsState = (enabled) => ({
  type: SET_NOTIFICATIONS_STATE,
  data: enabled,
})

export const setMobileResolution = (pixels) => ({
  type: SET_MOBILE_RESOLUTION,
  data: pixels,
})
