export const ADD_TO_PLAYLIST_OPEN = 'ADD_TO_PLAYLIST_OPEN'
export const ADD_TO_PLAYLIST_CLOSE = 'ADD_TO_PLAYLIST_CLOSE'
export const DUPLICATE_SONG_WARNING_OPEN = 'DUPLICATE_SONG_WARNING_OPEN'
export const DUPLICATE_SONG_WARNING_CLOSE = 'DUPLICATE_SONG_WARNING_CLOSE'
export const ALBUM_INFO_OPEN = 'ALBUM_INFO_OPEN'
export const ALBUM_INFO_CLOSE = 'ALBUM_INFO_CLOSE'

export const openAddToPlaylist = ({ selectedIds, onSuccess }) => ({
  type: ADD_TO_PLAYLIST_OPEN,
  selectedIds,
  onSuccess,
})

export const closeAddToPlaylist = () => ({
  type: ADD_TO_PLAYLIST_CLOSE,
})

export const openDuplicateSongWarning = (duplicateIds) => ({
  type: DUPLICATE_SONG_WARNING_OPEN,
  duplicateIds,
})

export const closeDuplicateSongDialog = () => ({
  type: DUPLICATE_SONG_WARNING_CLOSE,
})

export const openExpandInfoDialog = (data, ids = null) => {
  const record = ids && ids.length > 0 ? data[ids[0]] : data
  return {
    type: ALBUM_INFO_OPEN,
    record,
  }
}

export const closeExpandInfoDialog = () => ({
  type: ALBUM_INFO_CLOSE,
})
