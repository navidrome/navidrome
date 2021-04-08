export const RECENT_ALBUM = 'RECENT_ALBUM'
export const RECENT_PLAYLIST = 'RECENT_PLAYLIST'
export const RECENT_RESET = 'RECENT_RESET'

export const recentAlbum = () => {
  return {
    type: RECENT_ALBUM,
  }
}

export const recentPlaylist = () => {
  return {
    type: RECENT_PLAYLIST,
  }
}

export const recentReset = () => {
  return {
    type: RECENT_RESET,
  }
}
