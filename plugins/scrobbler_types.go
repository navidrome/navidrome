package plugins

// --- Input/Output JSON structures for Scrobbler plugin calls ---

// scrobblerAuthInput is the input for IsAuthorized
type scrobblerAuthInput struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

// scrobblerAuthOutput is the output for IsAuthorized
type scrobblerAuthOutput struct {
	Authorized bool `json:"authorized"`
}

// scrobblerTrackInfo contains track metadata for scrobbling
type scrobblerTrackInfo struct {
	ID                string  `json:"id"`
	Title             string  `json:"title"`
	Album             string  `json:"album"`
	Artist            string  `json:"artist"`
	AlbumArtist       string  `json:"album_artist"`
	Duration          float32 `json:"duration"`
	TrackNumber       int     `json:"track_number"`
	DiscNumber        int     `json:"disc_number"`
	MbzRecordingID    string  `json:"mbz_recording_id,omitempty"`
	MbzAlbumID        string  `json:"mbz_album_id,omitempty"`
	MbzArtistID       string  `json:"mbz_artist_id,omitempty"`
	MbzReleaseGroupID string  `json:"mbz_release_group_id,omitempty"`
	MbzAlbumArtistID  string  `json:"mbz_album_artist_id,omitempty"`
	MbzReleaseTrackID string  `json:"mbz_release_track_id,omitempty"`
}

// scrobblerNowPlayingInput is the input for NowPlaying
type scrobblerNowPlayingInput struct {
	UserID   string             `json:"user_id"`
	Username string             `json:"username"`
	Track    scrobblerTrackInfo `json:"track"`
	Position int                `json:"position"`
}

// scrobblerScrobbleInput is the input for Scrobble
type scrobblerScrobbleInput struct {
	UserID    string             `json:"user_id"`
	Username  string             `json:"username"`
	Track     scrobblerTrackInfo `json:"track"`
	Timestamp int64              `json:"timestamp"`
}

// scrobblerOutput is the output for NowPlaying and Scrobble
type scrobblerOutput struct {
	Error     string `json:"error,omitempty"`
	ErrorType string `json:"error_type,omitempty"` // "none", "not_authorized", "retry_later", "unrecoverable"
}

// scrobbler error type constants
const (
	scrobblerErrorNone          = "none"
	scrobblerErrorNotAuthorized = "not_authorized"
	scrobblerErrorRetryLater    = "retry_later"
	scrobblerErrorUnrecoverable = "unrecoverable"
)
