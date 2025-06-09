import subsonic from '../subsonic'
import { playTracks } from '../actions'

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
