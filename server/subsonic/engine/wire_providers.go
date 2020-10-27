package engine

import (
	"github.com/deluan/navidrome/core"
	"github.com/google/wire"
)

var Set = wire.NewSet(
	NewPlaylists,
	core.NewPlayers,
)
