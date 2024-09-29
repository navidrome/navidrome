export const PLAYER_ADD_TRACKS = 'PLAYER_ADD_TRACKS'
export const PLAYER_PLAY_NEXT = 'PLAYER_PLAY_NEXT'
export const PLAYER_SET_TRACK = 'PLAYER_SET_TRACK'
export const PLAYER_SYNC_QUEUE = 'PLAYER_SYNC_QUEUE'
export const PLAYER_CLEAR_QUEUE = 'PLAYER_CLEAR_QUEUE'
export const PLAYER_PLAY_TRACKS = 'PLAYER_PLAY_TRACKS'
export const PLAYER_CURRENT = 'PLAYER_CURRENT'
export const PLAYER_SET_VOLUME = 'PLAYER_SET_VOLUME'
export const PLAYER_SET_MODE = 'PLAYER_SET_MODE'

export const setTrack = (data) => ({
  type: PLAYER_SET_TRACK,
  data,
})

export const filterSongs = (data, ids) => {
  if (!ids) {
    return data
  }
  return ids.reduce((acc, id) => ({ ...acc, [id]: data[id] }), {})
}

export const addTracks = (data, ids) => {
  const songs = filterSongs(data, ids)
  return {
    type: PLAYER_ADD_TRACKS,
    data: songs,
  }
}

export const playNext = (data, ids) => {
  const songs = filterSongs(data, ids)
  return {
    type: PLAYER_PLAY_NEXT,
    data: songs,
  }
}

export const shuffle = (data) => {
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

export const shuffleTracks = (data, ids) => {
  const songs = filterSongs(data, ids)
  const shuffled = shuffle(songs)
  const firstId = Object.keys(shuffled)[0]
  return {
    type: PLAYER_PLAY_TRACKS,
    id: firstId,
    data: shuffled,
  }
}

export const playTracks = (data, ids, selectedId) => {
  const songs = filterSongs(data, ids)
  return {
    type: PLAYER_PLAY_TRACKS,
    id: selectedId || Object.keys(songs)[0],
    data: songs,
  }
}

export const syncQueue = (audioInfo, audioLists) => ({
  type: PLAYER_SYNC_QUEUE,
  data: {
    audioInfo,
    audioLists,
  },
})

export const clearQueue = () => ({
  type: PLAYER_CLEAR_QUEUE,
})

export const currentPlaying = (audioInfo) => ({
  type: PLAYER_CURRENT,
  data: audioInfo,
})

export const setVolume = (volume) => ({
  type: PLAYER_SET_VOLUME,
  data: { volume },
})

export const setPlayMode = (mode) => ({
  type: PLAYER_SET_MODE,
  data: { mode },
})
