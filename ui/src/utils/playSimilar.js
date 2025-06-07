import subsonic from '../subsonic'
import { playTracks } from '../actions'

export const playSimilar = (dispatch, notify, id) => {
  return subsonic
    .getSimilarSongs2(id, 100)
    .then((res) => res.json['subsonic-response'])
    .then((data) => {
      if (data.status !== 'ok') {
        notify('ra.page.error', 'warning')
        return
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
    })
    .catch(() => {
      notify('ra.page.error', 'warning')
    })
}
