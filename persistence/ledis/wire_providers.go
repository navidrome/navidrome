package ledis

import "github.com/google/wire"

var Set = wire.NewSet(
	NewCheckSumRepository,
	NewArtistIndexRepository,
	NewMediaFileRepository,
	NewMediaFolderRepository,
	NewNowPlayingRepository,
	NewPlaylistRepository,
)
