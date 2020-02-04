import 'react-jinke-music-player/assets/index.css'

const PLAYER_ADD_TRACK = 'PLAYER_ADD_TRACK'
const PLAYER_SET_TRACK = 'PLAYER_SET_TRACK'
const PLAYER_SYNC_QUEUE = 'PLAYER_SYNC_QUEUE'

const mapToAudioLists = (item) => ({
  id: item.id,
  name: item.title,
  singer: item.artist,
  cover: `/rest/getCoverArt.view?u=admin&p=enc:73756e6461&f=json&v=1.8.0&c=Jamstash&size=300&id=${item.id}`,
  musicSrc: `/rest/stream.view?u=admin&p=enc:73756e6461&f=json&v=1.8.0&c=Jamstash&id=${
    item.id
  }&ts=${new Date().getTime()}`
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
    default:
      return previousState
  }
}

export { addTrack, setTrack, syncQueue, playQueueReducer }
