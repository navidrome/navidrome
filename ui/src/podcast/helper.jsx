import subsonic from '../subsonic'
import config from '../config'

// Phase 1 plays episodes by streaming the RSS enclosure URL directly,
// reusing the player's isRadio bypass (no local download, no /rest/stream
// involvement yet). Phase 2 replaces this with a normal subsonic.streamUrl()
// call once stream.go can resolve podcast episode ids.
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
    streamUrl: episode.enclosureUrl,
    isRadio: true,
  }
}
