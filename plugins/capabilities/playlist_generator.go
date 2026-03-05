package capabilities

// PlaylistGenerator provides dynamically-generated playlists (e.g., "Daily Mix",
// personalized recommendations). Plugins implementing this capability expose two
// functions: GetPlaylists for lightweight discovery and GetPlaylist for fetching
// the heavy payload (tracks, metadata).
//
//nd:capability name=playlistgenerator required=true
type PlaylistGenerator interface {
	// GetPlaylists returns the list of playlists this plugin provides.
	//nd:export name=nd_playlist_generator_get_playlists
	GetPlaylists(GetPlaylistsRequest) (GetPlaylistsResponse, error)

	// GetPlaylist returns the full data for a single playlist (tracks, metadata).
	//nd:export name=nd_playlist_generator_get_playlist
	GetPlaylist(GetPlaylistRequest) (GetPlaylistResponse, error)
}

// GetPlaylistsRequest is the request for GetPlaylists.
type GetPlaylistsRequest struct{}

// GetPlaylistsResponse is the response for GetPlaylists.
type GetPlaylistsResponse struct {
	// Playlists is the list of playlists provided by this plugin.
	Playlists []PlaylistInfo `json:"playlists"`
	// RefreshInterval is the number of seconds until the next GetPlaylists call.
	// 0 means never re-discover.
	RefreshInterval int64 `json:"refreshInterval"`
}

// PlaylistInfo identifies a plugin playlist and its target user.
type PlaylistInfo struct {
	// ID is the plugin-scoped unique identifier for this playlist.
	ID string `json:"id"`
	// OwnerUserID is the Navidrome user ID that owns this playlist.
	OwnerUserID string `json:"ownerUserId"`
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
