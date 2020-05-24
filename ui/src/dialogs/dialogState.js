const NEW_PLAYLIST_OPEN = 'NEW_PLAYLIST_OPEN'
const NEW_PLAYLIST_CLOSE = 'NEW_PLAYLIST_CLOSE'

const openNewPlaylist = (albumId, selectedIds) => ({
  type: NEW_PLAYLIST_OPEN,
  albumId,
  selectedIds,
})

const closeNewPlaylist = () => ({
  type: NEW_PLAYLIST_CLOSE,
})

const newPlaylistDialogReducer = (
  previousState = {
    open: false,
  },
  payload
) => {
  const { type } = payload
  switch (type) {
    case NEW_PLAYLIST_OPEN:
      return {
        ...previousState,
        open: true,
        albumId: payload.albumId,
        selectedIds: payload.selectedIds,
      }
    case NEW_PLAYLIST_CLOSE:
      return { ...previousState, open: false }
    default:
      return previousState
  }
}

export { openNewPlaylist, closeNewPlaylist, newPlaylistDialogReducer }
