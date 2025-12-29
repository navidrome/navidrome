package plugins

// --- Input/Output JSON structures for Scrobbler plugin calls ---

// scrobblerAuthInput is the input for IsAuthorized
type scrobblerAuthInput struct {
	UserID   string `json:"userId"`
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
	AlbumArtist       string  `json:"albumArtist"`
	Duration          float32 `json:"duration"`
	TrackNumber       int     `json:"trackNumber"`
	DiscNumber        int     `json:"discNumber"`
	MbzRecordingID    string  `json:"mbzRecordingId,omitempty"`
	MbzAlbumID        string  `json:"mbzAlbumId,omitempty"`
	MbzArtistID       string  `json:"mbzArtistId,omitempty"`
	MbzReleaseGroupID string  `json:"mbzReleaseGroupId,omitempty"`
	MbzAlbumArtistID  string  `json:"mbzAlbumArtistId,omitempty"`
	MbzReleaseTrackID string  `json:"mbzReleaseTrackId,omitempty"`
}

// scrobblerNowPlayingInput is the input for NowPlaying
type scrobblerNowPlayingInput struct {
	UserID   string             `json:"userId"`
	Username string             `json:"username"`
	Track    scrobblerTrackInfo `json:"track"`
	Position int                `json:"position"`
}

// scrobblerScrobbleInput is the input for Scrobble
type scrobblerScrobbleInput struct {
	UserID    string             `json:"userId"`
	Username  string             `json:"username"`
	Track     scrobblerTrackInfo `json:"track"`
	Timestamp int64              `json:"timestamp"`
}

// scrobblerOutput is the output for NowPlaying and Scrobble
type scrobblerOutput struct {
	Error     string `json:"error,omitempty"`
	ErrorType string `json:"errorType,omitempty"` // "none", "notAuthorized", "retryLater", "unrecoverable"
}

// scrobbler error type constants
const (
	scrobblerErrorNone          = "none"
	scrobblerErrorNotAuthorized = "not_authorized"
	scrobblerErrorRetryLater    = "retry_later"
	scrobblerErrorUnrecoverable = "unrecoverable"
)
