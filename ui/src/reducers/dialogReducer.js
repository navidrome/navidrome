import {
  ADD_TO_PLAYLIST_CLOSE,
  ADD_TO_PLAYLIST_OPEN,
  DUPLICATE_SONG_WARNING_OPEN,
  DUPLICATE_SONG_WARNING_CLOSE,
  EXTENDED_INFO_OPEN,
  EXTENDED_INFO_CLOSE,
  LISTENBRAINZ_TOKEN_OPEN,
  LISTENBRAINZ_TOKEN_CLOSE,
} from '../actions'

export const addToPlaylistDialogReducer = (
  previousState = {
    open: false,
    duplicateSong: false,
  },
  payload
) => {
  const { type } = payload
  switch (type) {
    case ADD_TO_PLAYLIST_OPEN:
      return {
        ...previousState,
        open: true,
        selectedIds: payload.selectedIds,
        onSuccess: payload.onSuccess,
      }
    case ADD_TO_PLAYLIST_CLOSE:
      return { ...previousState, open: false, onSuccess: undefined }
    case DUPLICATE_SONG_WARNING_OPEN:
      return {
        ...previousState,
        duplicateSong: true,
        duplicateIds: payload.duplicateIds,
      }
    case DUPLICATE_SONG_WARNING_CLOSE:
      return { ...previousState, duplicateSong: false }
    default:
      return previousState
  }
}

export const expandInfoDialogReducer = (
  previousState = {
    open: false,
  },
  payload
) => {
  const { type } = payload
  switch (type) {
    case EXTENDED_INFO_OPEN:
      return {
        ...previousState,
        open: true,
        record: payload.record,
      }
    case EXTENDED_INFO_CLOSE:
      return {
        ...previousState,
        open: false,
      }
    default:
      return previousState
  }
}

export const listenBrainzTokenDialogReducer = (
  previousState = {
    open: false,
  },
  payload
) => {
  const { type } = payload
  switch (type) {
    case LISTENBRAINZ_TOKEN_OPEN:
      return {
        ...previousState,
        open: true,
      }
    case LISTENBRAINZ_TOKEN_CLOSE:
      return {
        ...previousState,
        open: false,
      }
    default:
      return previousState
  }
}
