package core

import (
	"github.com/google/wire"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/extdata"
	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/core/metrics"
	"github.com/navidrome/navidrome/core/playback"
	"github.com/navidrome/navidrome/core/scrobbler"
)

var Set = wire.NewSet(
	NewMediaStreamer,
	GetTranscodingCache,
	NewArchiver,
	NewPlayers,
	NewShare,
	NewPlaylists,
	agents.GetAgents,
	extdata.NewExternalMetadata,
	ffmpeg.New,
	scrobbler.GetPlayTracker,
	playback.GetInstance,
	metrics.GetInstance,
)
