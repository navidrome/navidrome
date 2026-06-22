package capabilities

import "github.com/navidrome/navidrome/plugins/types"

// Scrobbler provides scrobbling functionality to external services.
// This capability allows plugins to submit listening history to services like Last.fm,
// ListenBrainz, or custom scrobbling backends.
//
// All methods are required - plugins implementing this capability must provide
// all four functions: IsAuthorized, NowPlaying, Scrobble, and PlaybackReport.
//
//nd:capability name=scrobbler required=true
type Scrobbler interface {
	// IsAuthorized checks if a user is authorized to scrobble to this service.
	//nd:export name=nd_scrobbler_is_authorized
	IsAuthorized(IsAuthorizedRequest) (bool, error)

	// NowPlaying sends a now playing notification to the scrobbling service.
	//nd:export name=nd_scrobbler_now_playing
	NowPlaying(NowPlayingRequest) error

	// Scrobble submits a completed scrobble to the scrobbling service.
	//nd:export name=nd_scrobbler_scrobble
	Scrobble(ScrobbleRequest) error

	// PlaybackReport sends a playback state report to the scrobbling service.
	//nd:export name=nd_scrobbler_playback_report
	PlaybackReport(PlaybackReportRequest) error
}

// IsAuthorizedRequest is the request for authorization check.
type IsAuthorizedRequest struct {
	// Username is the username of the user.
	Username string `json:"username"`
}

// Deprecated: use types.ArtistRef.
type ArtistRef = types.ArtistRef

// Deprecated: use types.TrackInfo.
type TrackInfo = types.TrackInfo

// NowPlayingRequest is the request for now playing notification.
type NowPlayingRequest struct {
	// Username is the username of the user.
	Username string `json:"username"`
	// Track is the track currently playing.
	Track TrackInfo `json:"track"`
	// Position is the current playback position in seconds.
	Position int32 `json:"position"`
}

// ScrobbleRequest is the request for submitting a scrobble.
type ScrobbleRequest struct {
	// Username is the username of the user.
	Username string `json:"username"`
	// Track is the track that was played.
	Track TrackInfo `json:"track"`
	// Timestamp is the Unix timestamp when the track started playing.
	Timestamp int64 `json:"timestamp"`
}

// PlaybackReportRequest is the request for playback report notifications.
type PlaybackReportRequest struct {
	// Username is the username of the user.
	Username string `json:"username"`
	// Track is the track being played.
	Track TrackInfo `json:"track"`
	// State is the current playback state (starting/playing/paused/stopped/expired).
	State string `json:"state"`
	// PositionMs is the current playback position in milliseconds.
	PositionMs int64 `json:"positionMs"`
	// PlaybackRate is the playback speed (1.0 = normal).
	PlaybackRate float64 `json:"playbackRate"`
	// PlayerId is the unique client identifier.
	PlayerId string `json:"playerId"`
	// PlayerName is the human-readable player name.
	PlayerName string `json:"playerName"`
	// Timestamp is the Unix timestamp when this report was generated.
	Timestamp int64 `json:"timestamp"`
}

// ScrobblerError represents an error type for scrobbling operations.
type ScrobblerError string

const (
	// ScrobblerErrorNotAuthorized indicates the user is not authorized.
	ScrobblerErrorNotAuthorized ScrobblerError = "scrobbler(not_authorized)"
	// ScrobblerErrorRetryLater indicates the operation should be retried later.
	ScrobblerErrorRetryLater ScrobblerError = "scrobbler(retry_later)"
	// ScrobblerErrorUnrecoverable indicates an unrecoverable error.
	ScrobblerErrorUnrecoverable ScrobblerError = "scrobbler(unrecoverable)"
)

// Error implements the error interface for ScrobblerError.
func (e ScrobblerError) Error() string { return string(e) }
