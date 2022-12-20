package core

import (
	"github.com/google/wire"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/core/scrobbler"
)

var Set = wire.NewSet(
	NewArtwork,
	NewMediaStreamer,
	GetTranscodingCache,
	GetImageCache,
	NewArchiver,
	NewExternalMetadata,
	NewPlayers,
	agents.New,
	ffmpeg.New,
	scrobbler.GetPlayTracker,
	NewShare,
	NewPlaylists,
)
