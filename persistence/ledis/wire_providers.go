package ledis

import "github.com/google/wire"

var Set = wire.NewSet(
	NewAlbumRepository,
	NewCheckSumRepository,
	NewArtistIndexRepository,
	NewMediaFileRepository,
	NewMediaFolderRepository,
	NewNowPlayingRepository,
	NewPlaylistRepository,
)
