export const SET_NOTIFICATIONS_STATE = 'SET_NOTIFICATIONS_STATE'
export const SET_TOGGLEABLE_FIELDS = 'SET_TOGGLEABLE_FIELDS'
export const SET_OMITTED_FIELDS = 'SET_OMITTED_FIELDS'
export const SET_PLAYER_RATING_CONTROL = 'SET_PLAYER_RATING_CONTROL'

export const setNotificationsState = (enabled) => ({
  type: SET_NOTIFICATIONS_STATE,
  data: enabled,
})

export const setToggleableFields = (obj) => ({
  type: SET_TOGGLEABLE_FIELDS,
  data: obj,
})

export const setOmittedFields = (obj) => ({
  type: SET_OMITTED_FIELDS,
  data: obj,
})

export const setPlayerRatingControl = (obj) => ({
  type: SET_PLAYER_RATING_CONTROL,
  data: obj,
})
