import {
  SET_NOTIFICATIONS_STATE,
  SET_OMITTED_FIELDS,
  SET_SIDEBAR_PLAYLISTS_FAVOURITES,
  SET_TOGGLEABLE_FIELDS,
  SET_SHOW_FOLDER_VIEW,
  SET_SHOW_PODCASTS,
  SET_VIEW_TOGGLE,
} from '../actions'

const initialState = {
  notifications: false,
  showFolderView: true,
  showPodcasts: true,
  // Default on, same as Folders/Podcasts - these are opt-out, not opt-in, so
  // the new dashboards are actually discoverable rather than invisible until
  // someone happens to find the Personal settings page.
  showGenreView: true,
  showAiGenreView: true,
  showAiMoodView: true,
  showMyTagsView: true,
  toggleableFields: {},
  omittedFields: {},
  sidebarPlaylistsOnlyFavourites: false,
}

export const settingsReducer = (previousState = initialState, payload) => {
  const { type, data } = payload

  if (previousState && previousState.showFolderView === undefined) {
    previousState = {
      ...previousState,
      showFolderView: true,
    }
  }
  if (previousState && previousState.showPodcasts === undefined) {
    previousState = {
      ...previousState,
      showPodcasts: true,
    }
  }
  if (previousState && previousState.showGenreView === undefined) {
    previousState = {
      ...previousState,
      showGenreView: true,
      showAiGenreView: true,
      showAiMoodView: true,
      showMyTagsView: true,
    }
  }

  switch (type) {
    case SET_NOTIFICATIONS_STATE:
      return {
        ...previousState,
        notifications: data,
      }
    case SET_SHOW_FOLDER_VIEW:
      return {
        ...previousState,
        showFolderView: data,
      }
    case SET_SHOW_PODCASTS:
      return {
        ...previousState,
        showPodcasts: data,
      }
    case SET_VIEW_TOGGLE:
      return {
        ...previousState,
        [data.key]: data.value,
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
