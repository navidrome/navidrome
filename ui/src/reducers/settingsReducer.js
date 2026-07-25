import {
  SET_NOTIFICATIONS_STATE,
  SET_OMITTED_FIELDS,
  SET_SIDEBAR_PLAYLISTS_FAVOURITES,
  SET_TOGGLEABLE_FIELDS,
} from '../actions'

const initialState = {
  notifications: false,
  toggleableFields: {},
  omittedFields: {},
  sidebarPlaylistsOnlyFavourites: false,
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
    case SET_OMITTED_FIELDS:
      return {
        ...previousState,
        omittedFields: {
          ...previousState.omittedFields,
          ...data,
        },
      }
    case SET_SIDEBAR_PLAYLISTS_FAVOURITES:
      return {
        ...previousState,
        sidebarPlaylistsOnlyFavourites: data,
      }
    default:
      return previousState
  }
}
