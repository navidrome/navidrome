import 'react-jinke-music-player/assets/index.css'
import { subsonicUrl } from '../subsonic'

const PLAYER_ADD_TRACK = 'PLAYER_ADD_TRACK'
const PLAYER_SET_TRACK = 'PLAYER_SET_TRACK'
const PLAYER_SYNC_QUEUE = 'PLAYER_SYNC_QUEUE'
const PLAYER_SCROBBLE = 'PLAYER_SCROBBLE'

const mapToAudioLists = (item) => ({
  id: item.id,
  name: item.title,
  singer: item.artist,
  cover: subsonicUrl('getCoverArt', item.id, 'size=300'),
  musicSrc: subsonicUrl('stream', item.id),
  scrobble: (submit) => subsonicUrl('scrobble', item.id, `submission=${submit}`)
})

const addTrack = (data) => ({
  type: PLAYER_ADD_TRACK,
  data
})

const setTrack = (data) => ({
  type: PLAYER_SET_TRACK,
  data
})

const syncQueue = (data) => ({
  type: PLAYER_SYNC_QUEUE,
  data
})

const scrobble = (id) => ({
  type: PLAYER_SCROBBLE,
  data: id
})

const playQueueReducer = (
  previousState = { queue: [], clear: true },
  { type, data }
) => {
  switch (type) {
    case PLAYER_ADD_TRACK:
      const queue = previousState.queue
      queue.push(mapToAudioLists(data))
      return { queue, clear: false }
    case PLAYER_SET_TRACK:
      return { queue: [mapToAudioLists(data)], clear: true }
    case PLAYER_SYNC_QUEUE:
      return { queue: data, clear: false }
    case PLAYER_SCROBBLE:
      const newQueue = previousState.queue.map((item) => {
        return {
          ...item,
          scrobbled: item.scrobbled || item.id === data
        }
      })
      return { queue: newQueue, clear: false }
    default:
      return previousState
  }
}

export { addTrack, setTrack, syncQueue, scrobble, playQueueReducer }
