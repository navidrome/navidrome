package types

// ArtistRef is a reference to an artist with name and optional MBID.
type ArtistRef struct {
	// ID is the internal Navidrome artist ID (if known).
	ID string `json:"id,omitempty"`
	// Name is the artist name.
	Name string `json:"name"`
	// MBID is the MusicBrainz ID for the artist.
	MBID string `json:"mbid,omitempty"`
}

// TrackInfo contains track metadata.
type TrackInfo struct {
	// ID is the internal Navidrome track ID.
	ID string `json:"id"`
	// Title is the track title.
	Title string `json:"title"`
	// Album is the album name.
	Album string `json:"album"`
	// Artist is the formatted artist name for display (e.g., "Artist1 • Artist2").
	Artist string `json:"artist"`
	// AlbumArtist is the formatted album artist name for display.
	AlbumArtist string `json:"albumArtist"`
	// Artists is the list of track artists.
	Artists []ArtistRef `json:"artists"`
	// AlbumArtists is the list of album artists.
	AlbumArtists []ArtistRef `json:"albumArtists"`
	// Duration is the track duration in seconds.
	Duration float32 `json:"duration"`
	// TrackNumber is the track number on the album.
	TrackNumber int32 `json:"trackNumber"`
	// DiscNumber is the disc number.
	DiscNumber int32 `json:"discNumber"`
	// MBZRecordingID is the MusicBrainz recording ID.
	MBZRecordingID string `json:"mbzRecordingId,omitempty"`
	// MBZAlbumID is the MusicBrainz album/release ID.
	MBZAlbumID string `json:"mbzAlbumId,omitempty"`
	// MBZReleaseGroupID is the MusicBrainz release group ID.
	MBZReleaseGroupID string `json:"mbzReleaseGroupId,omitempty"`
	// MBZReleaseTrackID is the MusicBrainz release track ID.
	MBZReleaseTrackID string `json:"mbzReleaseTrackId,omitempty"`
	// LibraryID is the ID of the library the track belongs to.
	// Only included if the plugin has library permission with filesystem access for the track's library.
	LibraryID int32 `json:"libraryId,omitempty"`
	// Path is the full path to the track file, relative to the library root.
	// Only included if the plugin has library permission with filesystem access for the track's library.
	Path string `json:"path,omitempty"`
}

// SongRef is a reference to a song with metadata for matching.
type SongRef struct {
	// ID is the internal Navidrome mediafile ID (if known).
	ID string `json:"id,omitempty"`
	// Name is the song name.
	Name string `json:"name"`
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
