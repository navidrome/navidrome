import { baseUrl } from '../utils'
import {
  httpClient,
  clientUniqueId,
  clientUniqueIdHeader,
} from '../dataProvider'

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
      const value = options[k]
      // Handle array parameters by appending each value separately
      if (Array.isArray(value)) {
        value.forEach((v) => params.append(k, v))
      } else {
        params.append(k, value)
      }
    })
  }
  return `/rest/${command}?${params.toString()}`
}

const ping = () => httpClient(url('ping'))

const reportPlaybackUrl = (mediaId, positionMs, state) =>
  url('reportPlayback', null, { mediaId, mediaType: 'song', positionMs, state })

const reportPlayback = (mediaId, positionMs, state) =>
  httpClient(reportPlaybackUrl(mediaId, positionMs, state))

const reportPlaybackKeepalive = (mediaId, positionMs, state) => {
  const u = reportPlaybackUrl(mediaId, positionMs, state)
  if (u) {
    fetch(baseUrl(u), {
      keepalive: true,
      headers: { [clientUniqueIdHeader]: clientUniqueId },
    })
  }
}

const star = (id) => httpClient(url('star', id))

const unstar = (id) => httpClient(url('unstar', id))

const setRating = (id, rating) => httpClient(url('setRating', id, { rating }))

const download = (id, format = 'raw', bitrate = '0') =>
  (window.location.href = baseUrl(url('download', id, { format, bitrate })))

const startScan = (options) => httpClient(url('startScan', null, options))

const getScanStatus = () => httpClient(url('getScanStatus'))

const getNowPlaying = () => httpClient(url('getNowPlaying'))

const getAvatarUrl = (username, size) =>
  baseUrl(
    url('getAvatar', null, {
      username,
      ...(size && { size }),
    }),
  )

const getCoverArtUrl = (record, size, square) => {
  const suffix = record.imageHash ? '_' + record.imageHash : ''
  const options = {
    // A hash-suffixed url is already pixel-versioned; the buster would defeat immutable caching.
    ...(!record.imageHash && record.updatedAt && { _: record.updatedAt }),
    ...(size && { size }),
    ...(square && { square }),
  }

  // TODO Move this logic to server
  if (record.album) {
    return baseUrl(url('getCoverArt', 'mf-' + record.id + suffix, options))
  } else if (record.albumArtist) {
    return baseUrl(url('getCoverArt', 'al-' + record.id + suffix, options))
  } else if (record.sync !== undefined) {
    // This is a playlist
    return baseUrl(url('getCoverArt', 'pl-' + record.id + suffix, options))
  } else if (record.streamUrl !== undefined) {
    // This is a radio station
    return baseUrl(url('getCoverArt', 'ra-' + record.id + suffix, options))
  } else {
    return baseUrl(url('getCoverArt', 'ar-' + record.id + suffix, options))
  }
}

const getDiscCoverArtUrl = (albumId, discNumber, updatedAt, size) => {
  const options = {
    ...(updatedAt && { _: updatedAt }),
    ...(size && { size }),
  }
  return baseUrl(
    url('getCoverArt', 'dc-' + albumId + ':' + discNumber, options),
  )
}

const getArtistInfo = (id) => {
  return httpClient(url('getArtistInfo', id))
}

const getAlbumInfo = (id) => {
  return httpClient(url('getAlbumInfo', id))
}

const getSimilarSongs2 = (id, count = 100) => {
  return httpClient(url('getSimilarSongs2', id, { count }))
}

const getTopSongs = (artist, count = 50) => {
  return httpClient(url('getTopSongs', null, { artist, count }))
}

const requireSuccessfulLyricsResponse = (response) => {
  const subsonicResponse = response?.json?.['subsonic-response']
  if (subsonicResponse?.status !== 'failed') return response

  const error = new Error(
    subsonicResponse.error?.message || 'Lyrics request failed',
  )
  error.body = response.json
  error.code = subsonicResponse.error?.code
  throw error
}

const getLyricsBySongId = (id, requestOptions) => {
  const { allSources, ...httpOptions } = requestOptions || {}
  const requestUrl = url('getLyricsBySongId', id, {
    enhanced: true,
    ...(allSources ? { allSources: true } : {}),
  })
  const request = Object.keys(httpOptions).length
    ? httpClient(requestUrl, httpOptions)
    : httpClient(requestUrl)
  return request.then(requireSuccessfulLyricsResponse)
}

const getLyricsVariants = (id, requestOptions) => {
  const requestUrl = url('getLyricsVariants', id)
  const request = requestOptions
    ? httpClient(requestUrl, requestOptions)
    : httpClient(requestUrl)
  return request.then(requireSuccessfulLyricsResponse)
}

const parseContentDispositionFilename = (header) => {
  if (typeof header !== 'string' || header.trim() === '') return ''
  const extended = /filename\*\s*=\s*(?:utf-8'')?([^;]+)/i.exec(header)
  if (extended) {
    try {
      return decodeURIComponent(extended[1].trim().replace(/^"(.*)"$/, '$1'))
    } catch {
      // Fall through to the plain filename parameter.
    }
  }
  const plain = /filename\s*=\s*("(?:[^"\\]|\\.)*"|[^;]+)/i.exec(header)
  if (!plain) return ''
  const value = plain[1].trim()
  if (value.startsWith('"') && value.endsWith('"')) {
    return value.slice(1, -1).replace(/\\(["\\])/g, '$1')
  }
  return value
}

const getLyricsOriginal = async (id, selection, requestOptions = {}) => {
  const requestUrl = url('getLyricsOriginal', id, { selection })
  if (!requestUrl) throw new Error('Not authenticated')
  const response = await fetch(baseUrl(requestUrl), {
    ...requestOptions,
    headers: {
      ...(requestOptions.headers || {}),
      [clientUniqueIdHeader]: clientUniqueId,
    },
  })
  const contentType = response.headers.get('Content-Type') || ''
  if (contentType.toLowerCase().includes('json')) {
    let body = null
    try {
      body = await response.json()
    } catch {
      // Fall through to the generic message.
    }
    const subsonicResponse = body?.['subsonic-response']
    const error = new Error(
      subsonicResponse?.error?.message || 'Lyrics original unavailable',
    )
    error.body = body
    error.code = subsonicResponse?.error?.code
    throw error
  }
  if (!response.ok) {
    throw new Error(`Lyrics original unavailable (${response.status})`)
  }
  const blob = await response.blob()
  if (!blob || blob.size === 0) {
    throw new Error('Lyrics original unavailable')
  }
  return {
    blob,
    filename: parseContentDispositionFilename(
      response.headers.get('Content-Disposition'),
    ),
    mediaType: contentType.split(';')[0].trim() || 'application/octet-stream',
  }
}

// updateLyricsPreference is a deliberate listener action. It is POST-only on
// the host so caches, prefetchers and links cannot mutate personal state.
const updateLyricsPreference = (id, payload, requestOptions = {}) => {
  const requestUrl = url('setLyricsPreference', id, payload)
  if (!requestUrl) return Promise.reject(new Error('Not authenticated'))
  return httpClient(requestUrl, {
    ...requestOptions,
    method: 'POST',
  }).then(requireSuccessfulLyricsResponse)
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
  ping,
  reportPlayback,
  reportPlaybackKeepalive,
  download,
  star,
  unstar,
  setRating,
  startScan,
  getScanStatus,
  getNowPlaying,
  getCoverArtUrl,
  getDiscCoverArtUrl,
  getAvatarUrl,
  streamUrl,
  getAlbumInfo,
  getArtistInfo,
  getTopSongs,
  getSimilarSongs2,
  getLyricsBySongId,
  getLyricsVariants,
  getLyricsOriginal,
  updateLyricsPreference,
}
