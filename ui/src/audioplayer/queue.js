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
})

const addTrack = (data) => ({
  type: PLAYER_ADD_TRACK,
  data,
})

const setTrack = (data) => ({
  type: PLAYER_SET_TRACK,
  data,
})

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
      queue.push(mapToAudioLists(data))
      return { ...previousState, queue, clear: false }
    case PLAYER_SET_TRACK:
      return {
        ...previousState,
        queue: [mapToAudioLists(data)],
        clear: true,
        playing: true,
        current: data.id,
      }
    case PLAYER_SYNC_QUEUE:
      const currentTrack = data.find((item) => item.id === data.id) || {}
      return {
        ...previousState,
        queue: data,
        clear: false,
        current: currentTrack.id,
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
        current: payload.id,
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
        current: payload.id,
      }
    default:
      return previousState
  }
}

export { addTrack, setTrack, playAlbum, syncQueue, scrobble, playQueueReducer }
