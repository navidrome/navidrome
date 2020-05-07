import 'react-jinke-music-player/assets/index.css'
import subsonic from '../subsonic'

const PLAYER_ADD_TRACK = 'PLAYER_ADD_TRACK'
const PLAYER_SET_TRACK = 'PLAYER_SET_TRACK'
const PLAYER_SYNC_QUEUE = 'PLAYER_SYNC_QUEUE'
const PLAYER_SCROBBLE = 'PLAYER_SCROBBLE'
const PLAYER_PLAY_ALBUM = 'PLAYER_PLAY_ALBUM'

const mapToAudioLists = (item) => ({
  id: item.id,
  trackId: item.id,
  name: item.title,
  singer: item.artist,
  cover: subsonic.url('getCoverArt', item.id, { size: 300 }),
  musicSrc: subsonic.url('stream', item.id, { ts: true }),
  scrobbled: false,
})

const setTrack = (data) => ({
  type: PLAYER_SET_TRACK,
  data,
})

const addTracks = (data) => ({
  type: PLAYER_ADD_TRACK,
  data,
})

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

const shuffleAlbum = (data) => {
  const shuffled = shuffle(data)
  const firstId = Object.keys(shuffled)[0]
  return {
    type: PLAYER_PLAY_ALBUM,
    id: firstId,
    data: shuffled,
  }
}

const playAlbum = (id, data) => ({
  type: PLAYER_PLAY_ALBUM,
  id,
  data,
})

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
      data.forEach((item) => {
        queue.push(mapToAudioLists(item))
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
  playAlbum,
  syncQueue,
  scrobble,
  shuffleAlbum,
  playQueueReducer,
}
