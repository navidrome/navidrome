package capabilities

// Scrobbler provides scrobbling functionality to external services.
// This capability allows plugins to submit listening history to services like Last.fm,
// ListenBrainz, or custom scrobbling backends.
//
// All methods are required - plugins implementing this capability must provide
// all three functions: IsAuthorized, NowPlaying, and Scrobble.
//
//nd:capability name=scrobbler required=true
type Scrobbler interface {
	// IsAuthorized checks if a user is authorized to scrobble to this service.
	//nd:export name=nd_scrobbler_is_authorized
	IsAuthorized(IsAuthorizedRequest) (IsAuthorizedResponse, error)

	// NowPlaying sends a now playing notification to the scrobbling service.
	//nd:export name=nd_scrobbler_now_playing
	NowPlaying(NowPlayingRequest) (ScrobblerResponse, error)

	// Scrobble submits a completed scrobble to the scrobbling service.
	//nd:export name=nd_scrobbler_scrobble
	Scrobble(ScrobbleRequest) (ScrobblerResponse, error)
}

// IsAuthorizedRequest is the request for authorization check.
type IsAuthorizedRequest struct {
	// UserID is the internal Navidrome user ID.
	UserID string `json:"userId"`
	// Username is the username of the user.
	Username string `json:"username"`
}

// IsAuthorizedResponse is the response for authorization check.
type IsAuthorizedResponse struct {
	// Authorized indicates whether the user is authorized to scrobble.
	Authorized bool `json:"authorized"`
}

// TrackInfo contains track metadata for scrobbling.
type TrackInfo struct {
	// ID is the internal Navidrome track ID.
	ID string `json:"id"`
	// Title is the track title.
	Title string `json:"title"`
	// Album is the album name.
	Album string `json:"album"`
	// Artist is the track artist.
	Artist string `json:"artist"`
	// AlbumArtist is the album artist.
	AlbumArtist string `json:"albumArtist"`
	// Duration is the track duration in seconds.
	Duration float32 `json:"duration"`
	// TrackNumber is the track number on the album.
	TrackNumber int32 `json:"trackNumber"`
	// DiscNumber is the disc number.
	DiscNumber int32 `json:"discNumber"`
	// MBZRecordingID is the MusicBrainz recording ID.
	MBZRecordingID string `json:"mbzRecordingId,omitempty"`
	// MBZAlbumID is the MusicBrainz album/release ID.
	MBZAlbumID string `json:"mbzAlbumId,omitempty"`
	// MBZArtistID is the MusicBrainz artist ID.
	MBZArtistID string `json:"mbzArtistId,omitempty"`
	// MBZReleaseGroupID is the MusicBrainz release group ID.
	MBZReleaseGroupID string `json:"mbzReleaseGroupId,omitempty"`
	// MBZAlbumArtistID is the MusicBrainz album artist ID.
	MBZAlbumArtistID string `json:"mbzAlbumArtistId,omitempty"`
	// MBZReleaseTrackID is the MusicBrainz release track ID.
	MBZReleaseTrackID string `json:"mbzReleaseTrackId,omitempty"`
}

// NowPlayingRequest is the request for now playing notification.
type NowPlayingRequest struct {
	// UserID is the internal Navidrome user ID.
	UserID string `json:"userId"`
	// Username is the username of the user.
	Username string `json:"username"`
	// Track is the track currently playing.
	Track TrackInfo `json:"track"`
	// Position is the current playback position in seconds.
	Position int32 `json:"position"`
}

// ScrobbleRequest is the request for submitting a scrobble.
type ScrobbleRequest struct {
	// UserID is the internal Navidrome user ID.
	UserID string `json:"userId"`
	// Username is the username of the user.
	Username string `json:"username"`
	// Track is the track that was played.
	Track TrackInfo `json:"track"`
	// Timestamp is the Unix timestamp when the track started playing.
	Timestamp int64 `json:"timestamp"`
}

// ScrobblerErrorType indicates how Navidrome should handle scrobbler errors.
type ScrobblerErrorType string

const (
	// ScrobblerErrorNone indicates no error occurred.
	ScrobblerErrorNone ScrobblerErrorType = "none"
	// ScrobblerErrorNotAuthorized indicates the user is not authorized.
	ScrobblerErrorNotAuthorized ScrobblerErrorType = "not_authorized"
	// ScrobblerErrorRetryLater indicates the operation should be retried later.
	ScrobblerErrorRetryLater ScrobblerErrorType = "retry_later"
	// ScrobblerErrorUnrecoverable indicates an unrecoverable error.
	ScrobblerErrorUnrecoverable ScrobblerErrorType = "unrecoverable"
)

// ScrobblerResponse is the response for scrobbler operations.
type ScrobblerResponse struct {
	// Error is the error message if the operation failed.
	Error string `json:"error,omitempty"`
	// ErrorType indicates how Navidrome should handle the error.
	ErrorType ScrobblerErrorType `json:"errorType,omitempty"`
}
