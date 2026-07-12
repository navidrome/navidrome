import subsonic from '../subsonic'
import config from '../config'

// Episodes are queued like any other track: the player resolves
// musicSrc via subsonic.streamUrl(episode.id), which now works for every
// episode regardless of download state - stream.go serves the local file
// if downloaded, or transparently proxies the source URL otherwise.
export function songFromPodcastEpisode(episode, channel) {
  if (!episode) {
    return undefined
  }

  const cover =
    channel?.uploadedImage || channel?.coverArtUrl
      ? subsonic.getCoverArtUrl(channel, config.uiCoverArtSize, true)
      : undefined

  return {
    ...episode,
    title: episode.title,
    album: channel?.title || '',
    artist: channel?.title || '',
    duration: episode.duration,
    cover,
  }
}
