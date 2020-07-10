package engine

import (
	"github.com/google/wire"
)

var Set = wire.NewSet(
	NewBrowser,
	NewListGenerator,
	NewPlaylists,
	NewRatings,
	NewScrobbler,
	NewSearch,
	NewNowPlayingRepository,
	NewUsers,
	NewPlayers,
)
