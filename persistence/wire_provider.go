package persistence

import (
	"github.com/google/wire"
)

var Set = wire.NewSet(
	//NewArtistRepository,
	//NewMediaFileRepository,
	//NewAlbumRepository,
	//NewCheckSumRepository,
	//NewPropertyRepository,
	//NewPlaylistRepository,
	//NewMediaFolderRepository,
	//NewGenreRepository,
	New,
)
