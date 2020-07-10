package core

import (
	"github.com/deluan/navidrome/core/transcoder"
	"github.com/google/wire"
)

var Set = wire.NewSet(
	NewCover,
	NewMediaStreamer,
	NewTranscodingCache,
	NewImageCache,
	transcoder.New,
)
