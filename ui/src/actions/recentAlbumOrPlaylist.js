export const RECENT_ALBUM = 'RECENT_ALBUM'
export const RECENT_PLAYLIST = 'RECENT_PLAYLIST'
export const RECENT_RESET = 'RECENT_RESET'

export const recentAlbum = (id) => {
  return {
    type: RECENT_ALBUM,
    id,
  }
}

export const recentPlaylist = (id) => {
  return {
    type: RECENT_PLAYLIST,
    id,
  }
}

export const recentReset = () => {
  return {
    type: RECENT_RESET,
  }
}
