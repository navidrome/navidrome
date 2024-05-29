import { baseUrl } from '../utils'
import { httpClient } from '../dataProvider'

const url = (command, id, options) => {
  const params = new URLSearchParams()
  params.append('u', localStorage.getItem('username'))
  params.append('t', localStorage.getItem('subsonic-token'))
  params.append('s', localStorage.getItem('subsonic-salt'))
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
  } else if (record.artist) {
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

export default {
  url,
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
  getAlbumInfo,
  getArtistInfo,
}
