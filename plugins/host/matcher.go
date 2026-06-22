package host

import "context"

// MatchSong is a song to resolve against the local library. It mirrors the
// internal agents.Song. DurationMs is in milliseconds; 0 means unknown.
type MatchSong struct {
	ID         string `json:"id,omitempty"`
	Name       string `json:"name"`
	MBID       string `json:"mbid,omitempty"`
	ISRC       string `json:"isrc,omitempty"`
	Artist     string `json:"artist,omitempty"`
	ArtistMBID string `json:"artistMbid,omitempty"`
	Album      string `json:"album,omitempty"`
	AlbumMBID  string `json:"albumMbid,omitempty"`
	DurationMs uint32 `json:"durationMs,omitempty"`
}

// MatcherService resolves externally-obtained songs to local library tracks,
// reusing Navidrome's matching algorithm (ID > MBID > ISRC > fuzzy title).
//
//nd:hostservice name=Matcher permission=matcher
type MatcherService interface {
	// MatchSongs resolves each input song to its best-matching library track.
	// It returns one entry per input song, in the same order as the input; the
	// entry for an input song that had no match is empty (absent).
	//nd:hostfunc
	MatchSongs(ctx context.Context, songs []MatchSong) (results []*Track, err error)
}
