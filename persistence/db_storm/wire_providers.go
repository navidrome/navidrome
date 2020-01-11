package db_storm

import "github.com/google/wire"

var Set = wire.NewSet(
	NewPropertyRepository,
	NewArtistRepository,
	NewAlbumRepository,
	NewMediaFileRepository,
	NewArtistIndexRepository,
)
