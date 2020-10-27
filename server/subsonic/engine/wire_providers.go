package engine

import (
	"github.com/google/wire"
)

var Set = wire.NewSet(
	NewPlaylists,
	NewPlayers,
)
