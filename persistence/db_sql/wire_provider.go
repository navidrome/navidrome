package db_sql

import (
	"github.com/cloudsonic/sonic-server/persistence"
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
	persistence.NewNowPlayingRepository,
	persistence.NewMediaFolderRepository,
	wire.Value(persistence.ProviderIdentifier("sql")),
)
