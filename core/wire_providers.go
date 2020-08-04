package core

import (
	"github.com/deluan/navidrome/core/transcoder"
	"github.com/google/wire"
)

var Set = wire.NewSet(
	NewArtwork,
	NewMediaStreamer,
	NewTranscodingCache,
	NewImageCache,
	NewArchiver,
	transcoder.New,
)
