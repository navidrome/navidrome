package capabilities

// PlaylistProvider provides dynamically-generated playlists (e.g., "Daily Mix",
// personalized recommendations). Plugins implementing this capability expose two
// functions: GetAvailablePlaylists for lightweight discovery and GetPlaylist for
// fetching the heavy payload (tracks, metadata).
//
//nd:capability name=playlistprovider required=true
type PlaylistProvider interface {
	// GetAvailablePlaylists returns the list of playlists this plugin provides.
	//nd:export name=nd_playlist_provider_get_available_playlists
	GetAvailablePlaylists(GetAvailablePlaylistsRequest) (GetAvailablePlaylistsResponse, error)

	// GetPlaylist returns the full data for a single playlist (tracks, metadata).
	//nd:export name=nd_playlist_provider_get_playlist
	GetPlaylist(GetPlaylistRequest) (GetPlaylistResponse, error)
}

// GetAvailablePlaylistsRequest is the request for GetAvailablePlaylists.
type GetAvailablePlaylistsRequest struct{}

// GetAvailablePlaylistsResponse is the response for GetAvailablePlaylists.
type GetAvailablePlaylistsResponse struct {
	// Playlists is the list of playlists provided by this plugin.
	Playlists []PlaylistInfo `json:"playlists"`
	// RefreshInterval is the number of seconds until the next GetAvailablePlaylists call.
	// 0 means never re-discover.
	RefreshInterval int64 `json:"refreshInterval"`
	// RetryInterval is the number of seconds before retrying a failed GetPlaylist call.
	// 0 means no automatic retry for transient errors.
	RetryInterval int64 `json:"retryInterval"`
}

// PlaylistInfo identifies a plugin playlist and its target user.
type PlaylistInfo struct {
	// ID is the plugin-scoped unique identifier for this playlist.
	ID string `json:"id"`
	// OwnerUsername is the Navidrome username that owns this playlist.
	OwnerUsername string `json:"ownerUsername"`
}

// GetPlaylistRequest is the request for GetPlaylist.
type GetPlaylistRequest struct {
	// ID is the plugin-scoped playlist ID.
	ID string `json:"id"`
}

// GetPlaylistResponse is the response for GetPlaylist.
type GetPlaylistResponse struct {
	// Name is the display name of the playlist.
	Name string `json:"name"`
	// Description is an optional description for the playlist.
	Description string `json:"description,omitempty"`
	// CoverArtURL is an optional external URL for the playlist cover art.
	CoverArtURL string `json:"coverArtUrl,omitempty"`
	// Tracks is the list of songs in the playlist, using SongRef for matching.
	Tracks []SongRef `json:"tracks"`
	// ValidUntil is a unix timestamp indicating when this playlist data expires.
	// 0 means static (never refresh).
	ValidUntil int64 `json:"validUntil"`
}

// PlaylistProviderError represents an error type for playlist provider operations.
type PlaylistProviderError string

const (
	// PlaylistProviderErrorNotFound indicates a playlist is currently unavailable.
	PlaylistProviderErrorNotFound PlaylistProviderError = "playlist_provider(not_found)"
)

// Error implements the error interface for PlaylistProviderError.
func (e PlaylistProviderError) Error() string { return string(e) }
