package storm

import "github.com/google/wire"

var Set = wire.NewSet(
	NewPropertyRepository,
	NewArtistRepository,
	NewAlbumRepository,
)
