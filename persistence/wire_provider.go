package persistence

import (
	"github.com/google/wire"
)

var Set = wire.NewSet(
	NewArtistRepository,
	NewMediaFileRepository,
	NewAlbumRepository,
	NewArtistIndexRepository,
	NewCheckSumRepository,
	NewPropertyRepository,
	NewPlaylistRepository,
	NewNowPlayingRepository,
	NewMediaFolderRepository,
	NewGenreRepository,
)
