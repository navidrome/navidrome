export const SET_NOTIFICATIONS_STATE = 'SET_NOTIFICATIONS_STATE'
export const SET_TOGGLEABLE_FIELDS = 'SET_TOGGLEABLE_FIELDS'

export const setNotificationsState = (enabled) => ({
  type: SET_NOTIFICATIONS_STATE,
  data: enabled,
})

export const setToggleableFields = (obj) => ({
  type: SET_TOGGLEABLE_FIELDS,
  data: obj,
})
