package external

import (
	"context"

	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/matcher"
	"github.com/navidrome/navidrome/model"
)

func (e *provider) matchSongsToLibrary(ctx context.Context, songs []agents.Song, count int) (model.MediaFiles, error) {
	m := matcher.New(e.ds)
	return m.MatchSongsToLibrary(ctx, songs, count)
}
