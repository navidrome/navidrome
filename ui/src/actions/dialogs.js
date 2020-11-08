export const ADD_TO_PLAYLIST_OPEN = 'ADD_TO_PLAYLIST_OPEN'
export const ADD_TO_PLAYLIST_CLOSE = 'ADD_TO_PLAYLIST_CLOSE'

export const openAddToPlaylist = ({ selectedIds, onSuccess }) => ({
  type: ADD_TO_PLAYLIST_OPEN,
  selectedIds,
  onSuccess,
})

export const closeAddToPlaylist = () => ({
  type: ADD_TO_PLAYLIST_CLOSE,
})
