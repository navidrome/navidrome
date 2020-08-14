package engine

import (
	"github.com/google/wire"
)

var Set = wire.NewSet(
	NewListGenerator,
	NewPlaylists,
	NewScrobbler,
	NewSearch,
	NewNowPlayingRepository,
	NewUsers,
	NewPlayers,
)
