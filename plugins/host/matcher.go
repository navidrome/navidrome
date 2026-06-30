package host

import (
	"context"

	"github.com/navidrome/navidrome/plugins/types"
)

// MatcherService resolves externally-obtained songs to local library tracks,
// reusing Navidrome's matching algorithm (ID > MBID > ISRC > fuzzy title).
//
//nd:hostservice name=Matcher permission=matcher
type MatcherService interface {
	// MatchSongs resolves each input song to its best-matching library track.
	// It returns one entry per input song, in the same order as the input; the
	// entry for an input song that had no match is empty (absent).
	//nd:hostfunc
	MatchSongs(ctx context.Context, songs []types.SongRef) (results []*types.Track, err error)
}
