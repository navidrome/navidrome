package persistence

import "github.com/google/wire"

var Set = wire.NewSet(
	NewAlbumRepository,
	NewArtistRepository,
	NewCheckSumRepository,
	NewArtistIndexRepository,
	NewMediaFileRepository,
	NewMediaFolderRepository,
	NewNowPlayingRepository,
	NewPlaylistRepository,
	NewPropertyRepository,
)
