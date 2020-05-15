import 'react-jinke-music-player/assets/index.css'
import subsonic from '../subsonic'

const PLAYER_ADD_TRACK = 'PLAYER_ADD_TRACK'
const PLAYER_SET_TRACK = 'PLAYER_SET_TRACK'
const PLAYER_SYNC_QUEUE = 'PLAYER_SYNC_QUEUE'
const PLAYER_SCROBBLE = 'PLAYER_SCROBBLE'
const PLAYER_PLAY_ALBUM = 'PLAYER_PLAY_ALBUM'

const mapToAudioLists = (item) => {
  // If item comes from a playlist, id is mediaFileId
  const id = item.mediaFileId || item.id
  return {
    trackId: id,
    name: item.title,
    singer: item.artist,
    duration: item.duration,
    cover: subsonic.url('getCoverArt', id, { size: 300 }),
    musicSrc: subsonic.url('stream', id, { ts: true }),
    scrobbled: false,
  }
}

const setTrack = (data) => ({
  type: PLAYER_SET_TRACK,
  data,
})

let filterAlbumSongs = function (data, ids) {
  if (!ids) {
    return data
  }
  return ids.reduce((acc, id) => ({ ...acc, [id]: data[id] }), {})
}

const addTracks = (data, ids) => {
  const songs = filterAlbumSongs(data, ids)
  return {
    type: PLAYER_ADD_TRACK,
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
  ids.forEach((id) => (shuffled[id] = data[id]))
  return shuffled
}

const shuffleTracks = (data, ids) => {
  const songs = filterAlbumSongs(data, ids)
  const shuffled = shuffle(songs)
  const firstId = Object.keys(shuffled)[0]
  return {
    type: PLAYER_PLAY_ALBUM,
    id: firstId,
    data: shuffled,
  }
}

const playTracks = (data, ids, selectedId) => {
  const songs = filterAlbumSongs(data, ids)
  return {
    type: PLAYER_PLAY_ALBUM,
    id: selectedId || Object.keys(songs)[0],
    data: songs,
  }
}

const syncQueue = (id, data) => ({
  type: PLAYER_SYNC_QUEUE,
  id,
  data,
})

const scrobble = (id, submit) => ({
  type: PLAYER_SCROBBLE,
  id,
  submit,
})

const playQueueReducer = (
  previousState = { queue: [], clear: true, playing: false },
  payload
) => {
  let queue
  const { type, data } = payload
  switch (type) {
    case PLAYER_ADD_TRACK:
      queue = previousState.queue
      Object.keys(data).forEach((id) => {
        queue.push(mapToAudioLists(data[id]))
      })
      return { ...previousState, queue, clear: false }
    case PLAYER_SET_TRACK:
      return {
        ...previousState,
        queue: [mapToAudioLists(data)],
        clear: true,
        playing: true,
      }
    case PLAYER_SYNC_QUEUE:
      return {
        ...previousState,
        queue: data,
        clear: false,
      }
    case PLAYER_SCROBBLE:
      const newQueue = previousState.queue.map((item) => {
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
        playing: true,
      }
    case PLAYER_PLAY_ALBUM:
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
        playing: true,
      }
    default:
      return previousState
  }
}

export {
  addTracks,
  setTrack,
  playTracks,
  syncQueue,
  scrobble,
  shuffleTracks,
  playQueueReducer,
}
