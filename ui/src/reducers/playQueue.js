import 'react-jinke-music-player/assets/index.css'
import get from 'lodash.get'
import { v4 as uuidv4 } from 'uuid'
import subsonic from '../subsonic'
import config from '../config'

import {
  PLAYER_CLEAR_QUEUE,
  PLAYER_SET_VOLUME,
  PLAYER_CURRENT,
  PLAYER_ADD_TRACKS,
  PLAYER_PLAY_NEXT,
  PLAYER_SET_TRACK,
  PLAYER_SYNC_QUEUE,
  PLAYER_SCROBBLE,
  PLAYER_PLAY_TRACKS,
  PLAYER_PAUSE_TRACKS,
  RECENT_ALBUM,
  RECENT_PLAYLIST,
  RECENT_RESET,
  PAUSE_PLAYER,
  RESET_PLAYER,
} from '../actions'

const mapToAudioLists = (item) => {
  // If item comes from a playlist, id is mediaFileId
  const id = item.mediaFileId || item.id
  return {
    trackId: id,
    name: item.title,
    singer: item.artist,
    album: item.album,
    albumId: item.albumId,
    artistId: item.albumArtistId,
    duration: item.duration,
    suffix: item.suffix,
    bitRate: item.bitRate,
    musicSrc: subsonic.url('stream', id, { ts: true }),
    cover: subsonic.getCoverArtUrl(
      {
        coverArtId: config.devFastAccessCoverArt ? item.albumId : id,
        updatedAt: item.updatedAt,
      },
      300
    ),
    scrobbled: false,
    uuid: uuidv4(),
  }
}

const initialState = {
  queue: [],
  clear: true,
  current: {},
  volume: 1,
  playIndex: 0,
  albumOrPlaylistId: '',
  recentAlbumOrPlaylist: {},
  action: '',
}

export const playQueueReducer = (previousState = initialState, payload) => {
  let queue, current
  let newQueue
  const { type, data, albumOrPlaylistId, id } = payload
  switch (type) {
    case PLAYER_CLEAR_QUEUE:
      return initialState
    case PLAYER_SET_VOLUME:
      return {
        ...previousState,
        playIndex: undefined,
        volume: data.volume,
      }
    case PLAYER_CURRENT:
      queue = previousState.queue
      current = data.ended
        ? {}
        : {
            trackId: data.trackId,
            uuid: data.uuid,
            paused: data.paused,
          }
      return {
        ...previousState,
        current,
        playIndex: undefined,
        volume: data.volume,
      }
    case PLAYER_ADD_TRACKS:
      queue = previousState.queue
      Object.keys(data).forEach((id) => {
        queue.push(mapToAudioLists(data[id]))
      })
      return { ...previousState, queue, clear: false, playIndex: undefined }
    case PLAYER_PLAY_NEXT:
      current = get(previousState.current, 'uuid', '')
      newQueue = []
      let foundPos = false
      previousState.queue.forEach((item) => {
        newQueue.push(item)
        if (item.uuid === current) {
          foundPos = true
          Object.keys(data).forEach((id) => {
            newQueue.push(mapToAudioLists(data[id]))
          })
        }
      })
      if (!foundPos) {
        Object.keys(data).forEach((id) => {
          newQueue.push(mapToAudioLists(data[id]))
        })
      }
      return {
        ...previousState,
        queue: newQueue,
        clear: true,
        playIndex: undefined,
      }
    case PLAYER_SET_TRACK:
      return {
        ...previousState,
        queue: [mapToAudioLists(data)],
        clear: true,
        playIndex: 0,
      }
    case PLAYER_SYNC_QUEUE:
      current = data.length > 0 ? previousState.current : {}
      return {
        ...previousState,
        queue: data,
        clear: false,
        playIndex: undefined,
        current,
      }
    case PLAYER_SCROBBLE:
      newQueue = previousState.queue.map((item) => {
        return {
          ...item,
          scrobbled:
            item.scrobbled || (item.trackId === payload.id && payload.submit),
        }
      })
      return {
        ...previousState,
        queue: newQueue,
        playIndex: undefined,
        clear: false,
      }
    case PLAYER_PLAY_TRACKS:
      queue = []
      let match = false
      Object.keys(data).forEach((id) => {
        if (id === payload.id) {
          match = true
        }
        if (match) {
          queue.push(mapToAudioLists(data[id]))
        }
      })
      return {
        ...previousState,
        queue,
        playIndex: 0,
        clear: true,
        albumOrPlaylistId,
      }
    case PLAYER_PAUSE_TRACKS:
      return {
        ...previousState,
        current: {
          ...previousState.current,
          paused: true,
        },
      }
    case RECENT_ALBUM:
      return {
        ...previousState,
        recentAlbumOrPlaylist: {
          type: 'album',
          id,
        },
      }
    case RECENT_PLAYLIST:
      return {
        ...previousState,
        recentAlbumOrPlaylist: {
          type: 'playlist',
          id,
        },
      }
    case RECENT_RESET:
      return {
        ...previousState,
        recentAlbumOrPlaylist: {
          type: '',
          id: '',
        },
      }
    case PAUSE_PLAYER:
      return {
        ...previousState,
        action: 'pause',
      }
    case RESET_PLAYER:
      return {
        ...previousState,
        action: '',
      }
    default:
      return previousState
  }
}
