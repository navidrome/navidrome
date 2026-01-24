import subsonic from '../subsonic/index.js'
import { playTracks } from '../actions/index.js'

const mapReplayGain = (song) => {
  const { replayGain: rg } = song
  if (!rg) {
    return song
  }

  return {
    ...song,
    ...(rg.albumGain !== undefined && { rgAlbumGain: rg.albumGain }),
    ...(rg.albumPeak !== undefined && { rgAlbumPeak: rg.albumPeak }),
    ...(rg.trackGain !== undefined && { rgTrackGain: rg.trackGain }),
    ...(rg.trackPeak !== undefined && { rgTrackPeak: rg.trackPeak }),
  }
}

export const processSongsForPlayback = (songs) => {
  const songData = {}
  const ids = []
  songs.forEach((s) => {
    const song = mapReplayGain(s)
    songData[song.id] = song
    ids.push(song.id)
  })
  return { songData, ids }
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

  const { songData, ids } = processSongsForPlayback(songs)
  dispatch(playTracks(songData, ids))
}
