import { fetchUtils } from 'react-admin'
import { baseUrl } from '../utils'

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
  const url = `/rest/${command}?${params.toString()}`
  return baseUrl(url)
}

const scrobble = (id, submit) =>
  fetchUtils.fetchJson(url('scrobble', id, { submission: submit }))

const star = (id) => fetchUtils.fetchJson(url('star', id))

const unstar = (id) => fetchUtils.fetchJson(url('unstar', id))

const download = (id) => (window.location.href = url('download', id))

const getCoverArtUrl = (record, size) => {
  const options = {
    ...(record.updatedAt && { _: record.updatedAt }),
    ...(size && { size }),
  }
  return url('getCoverArt', record.coverArtId || 'not_found', options)
}

export const fetchArtistInfoExtra = (id) => {
  fetchUtils.fetchJson(url('getArtistInfo', id))
}

export default {
  url,
  getCoverArtUrl: getCoverArtUrl,
  scrobble,
  download,
  star,
  unstar,
}
