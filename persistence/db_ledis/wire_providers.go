package db_ledis

import (
	"github.com/google/wire"
)

var Set = wire.NewSet(
	NewCheckSumRepository,
	NewNowPlayingRepository,
	NewPlaylistRepository,
)
