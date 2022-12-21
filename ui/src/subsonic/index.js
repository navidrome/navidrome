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
    })
  )

const nowPlaying = (id) => scrobble(id, null, false)

const star = (id) => httpClient(url('star', id))

const unstar = (id) => httpClient(url('unstar', id))

const setRating = (id, rating) => httpClient(url('setRating', id, { rating }))

const download = (id, format = 'raw', bitrate = '0') =>
  (window.location.href = baseUrl(url('download', id, { format, bitrate })))

const startScan = (options) => httpClient(url('startScan', null, options))

const getScanStatus = () => httpClient(url('getScanStatus'))

const getCoverArtUrl = (record, size) => {
  const options = {
    ...(record.updatedAt && { _: record.updatedAt }),
    ...(size && { size }),
  }

  const lastUpdate = Math.floor(Date.parse(record.updatedAt) / 1000)
  const id = record.id + '-' + Math.max(lastUpdate, 0).toString(16)
  if (record.album) {
    return baseUrl(url('getCoverArt', 'mf-' + id, options))
  } else {
    return baseUrl(url('getCoverArt', 'al-' + id, options))
  }
}

const getArtistInfo = (id) => {
  return httpClient(url('getArtistInfo', id))
}

const streamUrl = (id) => {
  return baseUrl(url('stream', id, { ts: true }))
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
  getArtistInfo,
}
