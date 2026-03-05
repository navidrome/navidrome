package plugins

const (
	CapabilityPlaylistGenerator Capability = "PlaylistGenerator"

	FuncPlaylistGeneratorGetPlaylists = "nd_playlist_generator_get_playlists"
	FuncPlaylistGeneratorGetPlaylist  = "nd_playlist_generator_get_playlist"
)

func init() {
	registerCapability(
		CapabilityPlaylistGenerator,
		FuncPlaylistGeneratorGetPlaylists,
		FuncPlaylistGeneratorGetPlaylist,
	)
}
