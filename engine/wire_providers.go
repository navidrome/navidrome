package engine

import "github.com/google/wire"

var Set = wire.NewSet(
	NewBrowser,
	NewCover,
	NewListGenerator,
	NewPlaylists,
	NewRatings,
	NewScrobbler,
	NewSearch,
	NewNowPlayingRepository,
	NewUsers,
)
