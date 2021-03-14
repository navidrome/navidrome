import {
  ADD_TO_PLAYLIST_CLOSE,
  ADD_TO_PLAYLIST_OPEN,
  DUPLICATE_SONG_WARNING_OPEN,
  DUPLICATE_SONG_WARNING_CLOSE,
  DUPLICATE_SONG_SKIP,
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
      return { ...previousState, duplicateSong: false, duplicateIds: undefined }
    case DUPLICATE_SONG_SKIP:
      return {
        ...previousState,
        duplicateSong: false,
        selectedIds: payload.selectedIds,
      }
    default:
      return previousState
  }
}
