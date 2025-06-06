package core

import (
	"github.com/google/wire"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/external"
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
	NewLibrary,
	agents.GetAgents,
	external.NewProvider,
	wire.Bind(new(external.Agents), new(*agents.Agents)),
	ffmpeg.New,
	scrobbler.GetPlayTracker,
	playback.GetInstance,
	metrics.GetInstance,
)
