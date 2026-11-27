package host

import (
	"context"

	"github.com/navidrome/navidrome/plugins/types"
)

// MatchOptions carries optional parameters for a match request.
type MatchOptions struct {
	// Username runs the match as that user (case-insensitive): their favourites and
	// ratings inform tiebreaking, and the returned tracks carry their annotations.
	Username string `json:"username,omitempty"`
}

// MatcherService resolves externally-obtained songs to local library tracks,
// reusing Navidrome's matching algorithm (ID > MBID > ISRC > fuzzy title).
//
//nd:hostservice name=Matcher permission=matcher
type MatcherService interface {
	// MatchSongs resolves each input song to its best-matching library track.
	// It returns one entry per input song, in the same order as the input; the
	// entry for an input song that had no match is empty (absent). Results are
	// limited to the libraries the plugin (and the scoped user, if any) can access.
	//nd:hostfunc
	MatchSongs(ctx context.Context, songs []types.SongRef, opts MatchOptions) (results []*types.Track, err error)
}
