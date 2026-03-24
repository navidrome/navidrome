import subsonic from '../subsonic/index.js'
import { playTracks } from '../actions/index.js'

const shuffleArray = (array) => {
  const shuffled = [...array]
  for (let i = shuffled.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1))
    ;[shuffled[i], shuffled[j]] = [shuffled[j], shuffled[i]]
  }
  return shuffled
}

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

export const playSimilar = async (dispatch, notify, id, options = {}) => {
  const { seedRecord = null, shuffle = false } = options

  const res = await subsonic.getSimilarSongs2(id, 100)
  const data = res.json['subsonic-response']

  if (data.status !== 'ok') {
    throw new Error(
      `Error fetching similar songs: ${data.error?.message || 'Unknown error'} (Code: ${data.error?.code || 'unknown'})`,
    )
  }

  let songs = data.similarSongs2?.song || []

  // Randomize similar songs if requested
  if (shuffle) {
    songs = shuffleArray(songs)
  }

  // If no similar songs found and no seed, show warning
  if (!songs.length && !seedRecord) {
    notify('message.noSimilarSongsFound', 'warning')
    return
  }

  const { songData, ids } = processSongsForPlayback(songs)

  // Prepend seed song if provided
  if (seedRecord) {
    const seedId = seedRecord.mediaFileId || seedRecord.id
    // Remove seed from similar songs if it appears there
    const filteredIds = ids.filter((songId) => songId !== seedId)
    songData[seedId] = mapReplayGain(seedRecord)
    dispatch(playTracks(songData, [seedId, ...filteredIds]))
  } else {
    dispatch(playTracks(songData, ids))
  }
}
