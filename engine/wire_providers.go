package engine

import (
	"github.com/deluan/navidrome/engine/ffmpeg"
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
	ffmpeg.New,
	NewTranscodingCache,
)
