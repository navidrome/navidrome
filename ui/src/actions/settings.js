export const SET_NOTIFICATIONS_STATE = 'SET_NOTIFICATIONS_STATE'
export const SET_TOGGLEABLE_FIELDS = 'SET_TOGGLEABLE_FIELDS'
export const SET_OMITTED_FIELDS = 'SET_OMITTED_FIELDS'
export const SET_SHOW_FOLDER_VIEW = 'SET_SHOW_FOLDER_VIEW'
export const SET_SHOW_PODCASTS = 'SET_SHOW_PODCASTS'
// Generic view-visibility toggle, for personal-menu switches that don't
// warrant their own dedicated action type (see the AI Genre/AI Mood/My Tags/
// standard Genre sidebar toggles) - data is { key, value }, merged directly
// into settings state under that key.
export const SET_VIEW_TOGGLE = 'SET_VIEW_TOGGLE'
export const SET_SIDEBAR_PLAYLISTS_FAVOURITES =
  'SET_SIDEBAR_PLAYLISTS_FAVOURITES'

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

export const setShowFolderView = (enabled) => ({
  type: SET_SHOW_FOLDER_VIEW,
  data: enabled,
})

export const setShowPodcasts = (enabled) => ({
  type: SET_SHOW_PODCASTS,
  data: enabled,
})

export const setViewToggle = (key, enabled) => ({
  type: SET_VIEW_TOGGLE,
  data: { key, value: enabled },
})

export const setSidebarPlaylistsOnlyFavourites = (enabled) => ({
  type: SET_SIDEBAR_PLAYLISTS_FAVOURITES,
  data: enabled,
})
