package types

// ArtistRef is the minimal information a plugin returns for Navidrome to match an
// artist against the library. It is a reference, not a full artist entity: it
// carries only matching keys (name and optional internal/MusicBrainz IDs), never
// descriptive data such as biographies or images.
type ArtistRef struct {
	// ID is the internal Navidrome artist ID (if known).
	ID string `json:"id,omitempty"`
	// Name is the artist name.
	Name string `json:"name"`
	// MBID is the MusicBrainz ID for the artist.
	MBID string `json:"mbid,omitempty"`
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
	Duration float32 `json:"duration,omitempty"`
}
