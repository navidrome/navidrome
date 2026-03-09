import subsonic from '../subsonic'
import { httpClient } from '../dataProvider'

export async function fetchTranscodeDecision(songId, browserProfile) {
  const fetchUrl = subsonic.url('getTranscodeDecision', null, {
    mediaId: songId,
    mediaType: 'song',
  })

  const { json } = await httpClient(fetchUrl, {
    method: 'POST',
    body: JSON.stringify(browserProfile),
  })

  const subsonicResponse = json['subsonic-response']

  if (subsonicResponse.status !== 'ok') {
    const err = subsonicResponse.error || {}
    throw new Error(`getTranscodeDecision error: ${err.code} ${err.message}`)
  }

  return subsonicResponse.transcodeDecision
}
