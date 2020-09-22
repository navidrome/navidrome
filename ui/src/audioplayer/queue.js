import 'react-jinke-music-player/assets/index.css'
import get from 'lodash.get'
import { v4 as uuidv4 } from 'uuid'
import subsonic from '../subsonic'

const PLAYER_ADD_TRACKS = 'PLAYER_ADD_TRACKS'
const PLAYER_PLAY_NEXT = 'PLAYER_PLAY_NEXT'
const PLAYER_SET_TRACK = 'PLAYER_SET_TRACK'
const PLAYER_SYNC_QUEUE = 'PLAYER_SYNC_QUEUE'
const PLAYER_CLEAR_QUEUE = 'PLAYER_CLEAR_QUEUE'
const PLAYER_SCROBBLE = 'PLAYER_SCROBBLE'
const PLAYER_PLAY_TRACKS = 'PLAYER_PLAY_TRACKS'
const PLAYER_CURRENT = 'PLAYER_CURRENT'
const PLAYER_SET_VOLUME = 'PLAYER_SET_VOLUME'

const mapToAudioLists = (item) => {
  // If item comes from a playlist, id is mediaFileId
  const id = item.mediaFileId || item.id
  return {
    trackId: id,
    name: item.title,
    singer: item.artist,
    albumId: item.albumId,
    artistId: item.albumArtistId,
    duration: item.duration,
    cover: subsonic.url('getCoverArt', id, { size: 300 }),
    musicSrc: subsonic.url('stream', id, { ts: true }),
    scrobbled: false,
    uuid: uuidv4(),
  }
}

const setTrack = (data) => ({
  type: PLAYER_SET_TRACK,
  data,
})

const filterSongs = (data, ids) => {
  if (!ids) {
    return data
  }
  return ids.reduce((acc, id) => ({ ...acc, [id]: data[id] }), {})
}

const addTracks = (data, ids) => {
  const songs = filterSongs(data, ids)
  return {
    type: PLAYER_ADD_TRACKS,
    data: songs,
  }
}

const playNext = (data, ids) => {
  const songs = filterSongs(data, ids)
  return {
    type: PLAYER_PLAY_NEXT,
    data: songs,
  }
}

const shuffle = (data) => {
  const ids = Object.keys(data)
  for (let i = ids.length - 1; i > 0; i--) {
    let j = Math.floor(Math.random() * (i + 1))
    ;[ids[i], ids[j]] = [ids[j], ids[i]]
  }
  const shuffled = {}
  // The "_" is to force the object key to be a string, so it keeps the order when adding to object
  // or else the keys will always be in the same (numerically) order
  ids.forEach((id) => (shuffled['_' + id] = data[id]))
  return shuffled
}

const shuffleTracks = (data, ids) => {
  const songs = filterSongs(data, ids)
  const shuffled = shuffle(songs)
  const firstId = Object.keys(shuffled)[0]
  return {
    type: PLAYER_PLAY_TRACKS,
    id: firstId,
    data: shuffled,
  }
}

const playTracks = (data, ids, selectedId) => {
  const songs = filterSongs(data, ids)
  return {
    type: PLAYER_PLAY_TRACKS,
    id: selectedId || Object.keys(songs)[0],
    data: songs,
  }
}

const syncQueue = (id, data) => ({
  type: PLAYER_SYNC_QUEUE,
  id,
  data,
})

const clearQueue = () => ({
  type: PLAYER_CLEAR_QUEUE,
})

const scrobble = (id, submit) => ({
  type: PLAYER_SCROBBLE,
  id,
  submit,
})

const currentPlaying = (audioInfo) => ({
  type: PLAYER_CURRENT,
  data: audioInfo,
})

const setVolume = (volume) => ({
  type: PLAYER_SET_VOLUME,
  data: { volume },
})

const initialState = { queue: [], clear: true, current: {}, volume: 1 }

const playQueueReducer = (previousState = initialState, payload) => {
  let queue, current
  let newQueue
  const { type, data } = payload
  switch (type) {
    case PLAYER_CLEAR_QUEUE:
      return initialState
    case PLAYER_SET_VOLUME:
      return {
        ...previousState,
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
        volume: data.volume,
      }
    case PLAYER_ADD_TRACKS:
      queue = previousState.queue
      Object.keys(data).forEach((id) => {
        queue.push(mapToAudioLists(data[id]))
      })
      return { ...previousState, queue, clear: false }
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
      return { ...previousState, queue: newQueue, clear: true }
    case PLAYER_SET_TRACK:
      return {
        ...previousState,
        queue: [mapToAudioLists(data)],
        clear: true,
      }
    case PLAYER_SYNC_QUEUE:
      current = data.length > 0 ? previousState.current : {}
      return {
        ...previousState,
        queue: data,
        clear: false,
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
        clear: true,
      }
    default:
      return previousState
  }
}

export {
  addTracks,
  setTrack,
  playTracks,
  playNext,
  syncQueue,
  clearQueue,
  scrobble,
  currentPlaying,
  setVolume,
  shuffleTracks,
  playQueueReducer,
}
