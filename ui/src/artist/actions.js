import subsonic from '../subsonic/index.js'
import { playTracks } from '../actions/index.js'

export const playTopSongs = async (dispatch, notify, artistName) => {
  const res = await subsonic.getTopSongs(artistName, 100)
  const data = res.json['subsonic-response']

  if (data.status !== 'ok') {
    throw new Error(
      `Error fetching top songs: ${data.error?.message || 'Unknown error'} (Code: ${data.error?.code || 'unknown'})`,
    )
  }

  const songs = data.topSongs?.song || []
  if (!songs.length) {
    notify('message.noTopSongsFound', 'warning')
    return
  }

  const songData = {}
  const ids = []
  songs.forEach((s) => {
    songData[s.id] = s
    ids.push(s.id)
  })
  dispatch(playTracks(songData, ids))
}

export const playSimilar = async (dispatch, notify, id) => {
  const res = await subsonic.getSimilarSongs2(id, 100)
  const data = res.json['subsonic-response']

  if (data.status !== 'ok') {
    throw new Error(
      `Error fetching similar songs: ${data.error?.message || 'Unknown error'} (Code: ${data.error?.code || 'unknown'})`,
    )
  }

  const songs = data.similarSongs2?.song || []
  if (!songs.length) {
    notify('message.noSimilarSongsFound', 'warning')
    return
  }

  const songData = {}
  const ids = []
  songs.forEach((s) => {
    songData[s.id] = s
    ids.push(s.id)
  })
  dispatch(playTracks(songData, ids))
}

export const playShuffle = async (dataProvider, dispatch, id) => {
  const res = await dataProvider.getList('song', {
    pagination: { page: 1, perPage: 500 },
    sort: { field: 'random', order: 'ASC' },
    filter: { album_artist_id: id, missing: false },
  })

  const data = {}
  const ids = []
  res.data.forEach((s) => {
    data[s.id] = s
    ids.push(s.id)
  })
  dispatch(playTracks(data, ids))
}
