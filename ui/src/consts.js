export const REST_URL = '/api'

export const M3U_MIME_TYPE = 'audio/x-mpegurl'

export const AUTO_THEME_ID = 'AUTO_THEME_ID'

export const DraggableTypes = {
  SONG: 'song',
  ALBUM: 'album',
  DISC: 'disc',
  ARTIST: 'artist',
  ALL: [],
}

DraggableTypes.ALL.push(
  DraggableTypes.SONG,
  DraggableTypes.ALBUM,
  DraggableTypes.DISC,
  DraggableTypes.ARTIST
)

export const MAX_SIDEBAR_PLAYLISTS = 100
