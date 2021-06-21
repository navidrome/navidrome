package core

import (
	"github.com/google/wire"
	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/core/transcoder"
)

var Set = wire.NewSet(
	NewArtwork,
	NewMediaStreamer,
	GetTranscodingCache,
	GetImageCache,
	NewArchiver,
	NewExternalMetadata,
	NewCacheWarmer,
	NewPlayers,
	transcoder.New,
	scrobbler.GetInstance,
	NewShare,
)
