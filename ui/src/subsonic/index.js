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

const scrobble = (id, submit) =>
  httpClient(url('scrobble', id, { submission: submit }))

const star = (id) => httpClient(url('star', id))

const unstar = (id) => httpClient(url('unstar', id))

const setRating = (id, rating) => httpClient(url('setRating', id, { rating }))

const download = (id) => (window.location.href = baseUrl(url('download', id)))

const startScan = (options) => httpClient(url('startScan', null, options))

const getScanStatus = () => httpClient(url('getScanStatus'))

const getCoverArtUrl = (record, size) => {
  const options = {
    ...(record.updatedAt && { _: record.updatedAt }),
    ...(size && { size }),
  }
  return baseUrl(url('getCoverArt', record.coverArtId || 'not_found', options))
}

export default {
  url,
  scrobble,
  download,
  star,
  unstar,
  setRating,
  startScan,
  getScanStatus,
  getCoverArtUrl,
}
