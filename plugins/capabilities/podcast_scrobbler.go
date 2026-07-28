package capabilities

// PodcastScrobbler allows plugins to receive podcast episode play events.
// Plugins implementing this capability can track podcast listening history
// for analytics, cross-device sync, or custom podcast tracking backends.
//
//nd:capability name=podcast_scrobbler required=true
type PodcastScrobbler interface {
	// OnPodcastPlayed is called when a user plays a podcast episode.
	//nd:export name=nd_podcast_scrobbler_on_played
	OnPodcastPlayed(PodcastPlayedRequest) error
}

// PodcastPlayedRequest is the request sent when a podcast episode is played.
type PodcastPlayedRequest struct {
	// Username is the username of the user who played the episode.
	Username string `json:"username"`
	// Episode contains metadata about the episode that was played.
	Episode EpisodeInfo `json:"episode"`
}

// EpisodeInfo contains podcast episode metadata.
type EpisodeInfo struct {
	// ID is the internal Navidrome episode ID.
	ID string `json:"id"`
	// Title is the episode title.
	Title string `json:"title"`
	// ChannelID is the podcast channel's internal ID.
	ChannelID string `json:"channelId"`
	// ChannelTitle is the podcast channel name.
	ChannelTitle string `json:"channelTitle"`
	// Duration is the episode duration in seconds (0 if unknown).
	Duration float32 `json:"duration,omitempty"`
	// Timestamp is the Unix timestamp when the episode play was recorded.
	Timestamp int64 `json:"timestamp"`
}
