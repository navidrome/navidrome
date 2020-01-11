package db_storm

import (
	"github.com/cloudsonic/sonic-server/persistence"
	"github.com/google/wire"
)

var Set = wire.NewSet(
	NewPropertyRepository,
	NewArtistRepository,
	NewAlbumRepository,
	NewMediaFileRepository,
	NewArtistIndexRepository,
	NewPlaylistRepository,
	NewCheckSumRepository,
	persistence.NewNowPlayingRepository,
	persistence.NewMediaFolderRepository,
)
