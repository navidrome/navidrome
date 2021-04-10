import { RECENT_ALBUM, RECENT_PLAYLIST, RECENT_RESET } from '../actions'

const initialState = {
  type: '',
  id: '',
}

export const recentAlbumOrPlaylistReducer = (
  previousState = initialState,
  payload
) => {
  const { type, id } = payload
  switch (type) {
    case RECENT_ALBUM:
      return {
        type: 'album',
        id,
      }
    case RECENT_PLAYLIST:
      return {
        type: 'playlist',
        id,
      }
    case RECENT_RESET:
      return initialState
    default:
      return previousState
  }
}
