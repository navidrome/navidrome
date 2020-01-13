package db_sql

import (
	"github.com/cloudsonic/sonic-server/persistence"
	"github.com/cloudsonic/sonic-server/persistence/db_ledis"
	"github.com/google/wire"
)

var Set = wire.NewSet(
	NewArtistRepository,
	NewMediaFileRepository,
	NewAlbumRepository,
	NewArtistIndexRepository,
	NewCheckSumRepository,
	db_ledis.NewPropertyRepository,
	db_ledis.NewPlaylistRepository,
	persistence.NewNowPlayingRepository,
	persistence.NewMediaFolderRepository,
	wire.Value(persistence.ProviderIdentifier("sql")),
)
