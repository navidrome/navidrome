package engine

import (
	"github.com/google/wire"
)

var Set = wire.NewSet(
	NewBrowser,
	NewListGenerator,
	NewPlaylists,
	NewScrobbler,
	NewSearch,
	NewNowPlayingRepository,
	NewUsers,
	NewPlayers,
)
