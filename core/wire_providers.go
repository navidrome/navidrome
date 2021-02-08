package core

import (
	"github.com/google/wire"
	"github.com/navidrome/navidrome/core/transcoder"
)

var Set = wire.NewSet(
	NewArtwork,
	NewMediaStreamer,
	GetTranscodingCache,
	GetImageCache,
	NewArchiver,
	NewNowPlayingRepository,
	NewExternalInfo2,
	NewCacheWarmer,
	NewPlayers,
	transcoder.New,
)
