package engine

import (
	"github.com/deluan/navidrome/engine/transcoder"
	"github.com/google/wire"
)

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
	NewMediaStreamer,
	transcoder.New,
	NewTranscodingCache,
)
