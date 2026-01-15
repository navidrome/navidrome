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
	IsAuthorized(IsAuthorizedRequest) (bool, error)

	// NowPlaying sends a now playing notification to the scrobbling service.
	//nd:export name=nd_scrobbler_now_playing
	NowPlaying(NowPlayingRequest) error

	// Scrobble submits a completed scrobble to the scrobbling service.
	//nd:export name=nd_scrobbler_scrobble
	Scrobble(ScrobbleRequest) error
}

// IsAuthorizedRequest is the request for authorization check.
type IsAuthorizedRequest struct {
	// Username is the username of the user.
	Username string `json:"username"`
}

// ArtistRef is a reference to an artist with name and optional MBID.
type ArtistRef struct {
	// ID is the internal Navidrome artist ID (if known).
	ID string `json:"id,omitempty"`
	// Name is the artist name.
	Name string `json:"name"`
	// MBID is the MusicBrainz ID for the artist.
	MBID string `json:"mbid,omitempty"`
}

// TrackInfo contains track metadata for scrobbling.
type TrackInfo struct {
	// ID is the internal Navidrome track ID.
	ID string `json:"id"`
	// Title is the track title.
	Title string `json:"title"`
	// Album is the album name.
	Album string `json:"album"`
	// Artist is the formatted artist name for display (e.g., "Artist1 • Artist2").
	Artist string `json:"artist"`
	// AlbumArtist is the formatted album artist name for display.
	AlbumArtist string `json:"albumArtist"`
	// Artists is the list of track artists.
	Artists []ArtistRef `json:"artists"`
	// AlbumArtists is the list of album artists.
	AlbumArtists []ArtistRef `json:"albumArtists"`
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
	// MBZReleaseGroupID is the MusicBrainz release group ID.
	MBZReleaseGroupID string `json:"mbzReleaseGroupId,omitempty"`
	// MBZReleaseTrackID is the MusicBrainz release track ID.
	MBZReleaseTrackID string `json:"mbzReleaseTrackId,omitempty"`
}

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
