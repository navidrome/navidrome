import subsonic from '../subsonic'
import { baseUrl } from '../utils'

export async function fetchTranscodeDecision(songId, browserProfile) {
  const fetchUrl = baseUrl(
    subsonic.url('getTranscodeDecision', null, {
      mediaId: songId,
      mediaType: 'song',
    }),
  )

  const response = await fetch(fetchUrl, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(browserProfile),
  })

  if (!response.ok) {
    throw new Error(
      `getTranscodeDecision failed: ${response.status} ${response.statusText}`,
    )
  }

  const data = await response.json()
  const subsonicResponse = data['subsonic-response']

  if (subsonicResponse.status !== 'ok') {
    const err = subsonicResponse.error || {}
    throw new Error(
      `getTranscodeDecision error: ${err.code} ${err.message}`,
    )
  }

  return subsonicResponse.transcodeDecision
}
