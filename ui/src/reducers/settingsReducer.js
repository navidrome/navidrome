import {
  SET_NOTIFICATIONS_STATE,
  SET_OMITTED_FIELDS,
  SET_TOGGLEABLE_FIELDS,
  SET_SHOW_FOLDER_VIEW,
  SET_SHOW_PODCASTS,
} from '../actions'

const initialState = {
  notifications: false,
  showFolderView: true,
  showPodcasts: true,
  toggleableFields: {},
  omittedFields: {},
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
    default:
      return previousState
  }
}
