import type { Identifier, Record } from 'ra-core'

export const ADD_TO_PLAYLIST_OPEN = 'ADD_TO_PLAYLIST_OPEN'
export const ADD_TO_PLAYLIST_CLOSE = 'ADD_TO_PLAYLIST_CLOSE'
export const DOWNLOAD_MENU_OPEN = 'DOWNLOAD_MENU_OPEN'
export const DOWNLOAD_MENU_CLOSE = 'DOWNLOAD_MENU_CLOSE'
export const DUPLICATE_SONG_WARNING_OPEN = 'DUPLICATE_SONG_WARNING_OPEN'
export const DUPLICATE_SONG_WARNING_CLOSE = 'DUPLICATE_SONG_WARNING_CLOSE'
export const EXTENDED_INFO_OPEN = 'EXTENDED_INFO_OPEN'
export const EXTENDED_INFO_CLOSE = 'EXTENDED_INFO_CLOSE'
export const LISTENBRAINZ_TOKEN_OPEN = 'LISTENBRAINZ_TOKEN_OPEN'
export const LISTENBRAINZ_TOKEN_CLOSE = 'LISTENBRAINZ_TOKEN_CLOSE'
export const DOWNLOAD_MENU_ALBUM = 'album'
export const DOWNLOAD_MENU_ARTIST = 'artist'
export const DOWNLOAD_MENU_PLAY = 'playlist'
export const DOWNLOAD_MENU_SONG = 'song'
export const SHARE_MENU_OPEN = 'SHARE_MENU_OPEN'
export const SHARE_MENU_CLOSE = 'SHARE_MENU_CLOSE'
export const MOVE_TO_INDEX_OPEN = 'MOVE_TO_INDEX_OPEN'
export const MOVE_TO_INDEX_CLOSE = 'MOVE_TO_INDEX_CLOSE'

// Maybe this type should be somewhere else
export type Resource = 'album' | 'playlist' | 'song' | 'artist'

export const openShareMenu = (
  ids: Identifier[],
  resource: Resource,
  name: string,
  label: string,
) => ({
  type: SHARE_MENU_OPEN,
  ids,
  resource,
  name,
  label,
})

export const closeShareMenu = () => ({
  type: SHARE_MENU_CLOSE,
})

export interface OpenAddToPlaylistArguments {
  selectedIds: number[]
  onSuccess: (id: number) => void
}

export const openAddToPlaylist = ({
  selectedIds,
  onSuccess,
}: OpenAddToPlaylistArguments) => ({
  type: ADD_TO_PLAYLIST_OPEN,
  selectedIds,
  onSuccess,
})

export const closeAddToPlaylist = () => ({
  type: ADD_TO_PLAYLIST_CLOSE,
})

export const openMoveToIndexDialog = (record: Record) => ({
  type: MOVE_TO_INDEX_OPEN,
  record,
})

export const closeMoveToIndexDialog = () => ({
  type: MOVE_TO_INDEX_CLOSE,
})

export const openDownloadMenu = (record: Record, recordType: Resource) => {
  return {
    type: DOWNLOAD_MENU_OPEN,
    recordType,
    record,
  }
}

export const closeDownloadMenu = () => ({
  type: DOWNLOAD_MENU_CLOSE,
})

export const openDuplicateSongWarning = (duplicateIds: number[]) => ({
  type: DUPLICATE_SONG_WARNING_OPEN,
  duplicateIds,
})

export const closeDuplicateSongDialog = () => ({
  type: DUPLICATE_SONG_WARNING_CLOSE,
})

export const openExtendedInfoDialog = (record: Record) => {
  return {
    type: EXTENDED_INFO_OPEN,
    record,
  }
}

export const closeExtendedInfoDialog = () => ({
  type: EXTENDED_INFO_CLOSE,
})

export const openListenBrainzTokenDialog = () => ({
  type: LISTENBRAINZ_TOKEN_OPEN,
})

export const closeListenBrainzTokenDialog = () => ({
  type: LISTENBRAINZ_TOKEN_CLOSE,
})
