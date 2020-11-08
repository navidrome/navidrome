import { ADD_TO_PLAYLIST_CLOSE, ADD_TO_PLAYLIST_OPEN } from '../actions'

export const addToPlaylistDialogReducer = (
  previousState = {
    open: false,
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
    default:
      return previousState
  }
}
