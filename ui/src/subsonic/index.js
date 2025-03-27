import { baseUrl } from '../utils'
import { httpClient } from '../dataProvider'

const url = (command, id, options) => {
  const username = localStorage.getItem('username')
  const token = localStorage.getItem('subsonic-token')
  const salt = localStorage.getItem('subsonic-salt')
  if (!username || !token || !salt) {
    return ''
  }

  const params = new URLSearchParams()
  params.append('u', username)
  params.append('t', token)
  params.append('s', salt)
  params.append('f', 'json')
  params.append('v', '1.8.0')
  params.append('c', 'NavidromeUI')
  id && params.append('id', id)
  if (options) {
    if (options.ts) {
      options['_'] = new Date().getTime()
      delete options.ts
    }
    Object.keys(options).forEach((k) => {
      params.append(k, options[k])
    })
  }
  return `/rest/${command}?${params.toString()}`
}

const ping = () => httpClient(url('ping'))

const scrobble = (id, time, submission = true) =>
  httpClient(
    url('scrobble', id, {
      ...(submission && time && { time }),
      submission,
    }),
  )

const nowPlaying = (id) => scrobble(id, null, false)

const star = (id) => httpClient(url('star', id))

const unstar = (id) => httpClient(url('unstar', id))

const setRating = (id, rating) => httpClient(url('setRating', id, { rating }))

const download = (id, format = 'raw', bitrate = '0') =>
  (window.location.href = baseUrl(url('download', id, { format, bitrate })))

const startScan = (options) => httpClient(url('startScan', null, options))

const getScanStatus = () => httpClient(url('getScanStatus'))

const getCoverArtUrl = (record, size, square) => {
  const options = {
    ...(record.updatedAt && { _: record.updatedAt }),
    ...(size && { size }),
    ...(square && { square }),
  }

  // TODO Move this logic to server. `song` and `album` should have a CoverArtID
  if (record.album) {
    return baseUrl(url('getCoverArt', 'mf-' + record.id, options))
  } else if (record.albumArtist) {
    return baseUrl(url('getCoverArt', 'al-' + record.id, options))
  } else {
    return baseUrl(url('getCoverArt', 'ar-' + record.id, options))
  }
}

const getArtistInfo = (id) => {
  return httpClient(url('getArtistInfo', id))
}

const getAlbumInfo = (id) => {
  return httpClient(url('getAlbumInfo', id))
}

const streamUrl = (id, options) => {
  return baseUrl(
    url('stream', id, {
      ts: true,
      ...options,
    }),
  )
}

const syncPlayQueue = (current, queue) => {
  return current === undefined
    ? httpClient(url('savePlayQueue') + queue)
    : httpClient(
        url('savePlayQueue') +
          queue +
          `&current=${current.song.id}` +
          syncTimePlayed(current),
      )
}
const syncTimePlayed = (current) => {
  // TODO: add the time to a environment variable or to sync settings option
  return current.duration > 480
    ? `&position=${Math.trunc(current.currentTime) * 1000}`
    : ''
}

const getStoredQueue = () => httpClient(url('getPlayQueue'))

export default {
  url,
  ping,
  scrobble,
  nowPlaying,
  download,
  star,
  unstar,
  setRating,
  startScan,
  getScanStatus,
  getCoverArtUrl,
  streamUrl,
  syncPlayQueue,
  getStoredQueue,
  getAlbumInfo,
  getArtistInfo,
}
