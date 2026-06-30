package types

// ArtistRef is the minimal information a plugin returns for Navidrome to match an
// artist against the library. It is a reference, not a full artist entity: it
// carries only matching keys (name and optional internal/MusicBrainz IDs) plus a
// few projection fields used when describing a track's participants, never
// descriptive data such as biographies or images.
type ArtistRef struct {
	// ID is the internal Navidrome artist ID (if known).
	ID string `json:"id,omitempty"`
	// Name is the artist name.
	Name string `json:"name"`
	// MBID is the MusicBrainz ID for the artist.
	MBID string `json:"mbid,omitempty"`
	// SortName is the artist name used for sorting (if known).
	SortName string `json:"sortName,omitempty"`
	// SubRole is the artist's specific role within a participation (e.g. "remixer",
	// "composer"); empty for a plain artist credit.
	SubRole string `json:"subRole,omitempty"`
}

// SongRef is the minimal information exchanged between a plugin and Navidrome to
// match a song. It is used both as input (a song Navidrome already has) and as
// output (a song a plugin suggests, which may not be in the library yet). Unlike
// Track, it is an abstract recording reference carrying only matching keys (IDs,
// ISRC, and title/artist/album/duration) that Navidrome resolves to a library track.
type SongRef struct {
	// ID is the internal Navidrome mediafile ID (if known).
	ID string `json:"id,omitempty"`
	// Name is the song name.
	Name string `json:"name"` // TODO: rename to Title to align with Track.Title and model.MediaFile.Title; kept as Name for now for compatibility.
	// MBID is the MusicBrainz ID for the song.
	MBID string `json:"mbid,omitempty"`
	// ISRC is the International Standard Recording Code for the song.
	ISRC string `json:"isrc,omitempty"`
	// Artist is the artist name.
	Artist string `json:"artist,omitempty"`
	// ArtistMBID is the MusicBrainz artist ID.
	ArtistMBID string `json:"artistMbid,omitempty"`
	// Artists is the full artist list; when set, takes precedence over Artist/ArtistMBID for matching.
	Artists []ArtistRef `json:"artists,omitempty"`
	// Album is the album name.
	Album string `json:"album,omitempty"`
	// AlbumMBID is the MusicBrainz release ID.
	AlbumMBID string `json:"albumMbid,omitempty"`
	// Duration is the song duration in seconds.
	//
	// Deprecated: use DurationMs, which carries millisecond precision. When
	// DurationMs is non-zero it takes precedence; Duration is kept only for
	// backwards compatibility with plugins that still send seconds.
	Duration float32 `json:"duration,omitempty"`
	// DurationMs is the song duration in milliseconds. It supersedes Duration
	// when non-zero.
	DurationMs uint32 `json:"durationMs,omitempty"`
}

// DurationInMs returns the song duration in milliseconds, preferring the
// millisecond-precision DurationMs and falling back to the deprecated
// seconds-based Duration. It returns 0 when neither is set.
func (s SongRef) DurationInMs() uint32 {
	if s.DurationMs != 0 {
		return s.DurationMs
	}
	return uint32(s.Duration * 1000)
}

// SetDurationMs sets the song duration from a millisecond value, populating both
// DurationMs and the deprecated seconds-based Duration so that plugins reading
// either field see a consistent value. Use this when building a SongRef to send
// to a plugin.
func (s *SongRef) SetDurationMs(ms uint32) {
	s.DurationMs = ms
	s.Duration = float32(ms) / 1000
}
