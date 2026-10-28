package core

import (
	"github.com/google/wire"
	"github.com/SunilSherkhane/navidrome/core/agents"
	"github.com/SunilSherkhane/navidrome/core/external"
	"github.com/SunilSherkhane/navidrome/core/ffmpeg"
	"github.com/SunilSherkhane/navidrome/core/lyrics"
	"github.com/SunilSherkhane/navidrome/core/matcher"
	"github.com/SunilSherkhane/navidrome/core/metrics"
	"github.com/SunilSherkhane/navidrome/core/playback"
	"github.com/SunilSherkhane/navidrome/core/playlists"
	"github.com/SunilSherkhane/navidrome/core/quickconnect"
	"github.com/SunilSherkhane/navidrome/core/scrobbler"
	"github.com/SunilSherkhane/navidrome/core/stream"
)

var Set = wire.NewSet(
	stream.NewMediaStreamer,
	stream.GetTranscodingCache,
	NewArchiver,
	NewPlayers,
	NewShare,
	playlists.NewPlaylists,
	NewLibrary,
	NewUser,
	NewMaintenance,
	stream.NewTranscodeDecider,
	agents.GetAgents,
	external.NewProvider,
	matcher.New,
	wire.Bind(new(external.Agents), new(*agents.Agents)),
	ffmpeg.New,
	scrobbler.GetPlayTracker,
	playback.GetInstance,
	quickconnect.GetInstance,
	metrics.GetInstance,
	lyrics.NewLyrics,
)
