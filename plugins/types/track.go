package types

// Track is a stable, public projection of a library media file for plugin consumption.
// It is a sane subset of the internal model.MediaFile, intended for reuse across host
// services and capabilities. Timestamps are Unix epoch seconds.
//
// Unlike SongRef, which is an abstract recording reference carrying only matching keys,
// Track is a concrete library entity: it identifies a specific media file that exists
// (or once existed) in the library and exposes its full descriptive metadata.
type Track struct {
	// Identity & location
	ID          string `json:"id"`
	LibraryID   int32  `json:"libraryId"`
	LibraryName string `json:"libraryName,omitempty"`
	Path        string `json:"path,omitempty"`
	Missing     bool   `json:"missing"`

	// Core metadata
	Title          string `json:"title"`
	Album          string `json:"album"`
	Artist         string `json:"artist"`
	AlbumArtist    string `json:"albumArtist,omitempty"`
	AlbumID        string `json:"albumId,omitempty"`
	SortTitle      string `json:"sortTitle,omitempty"`
	SortAlbumName  string `json:"sortAlbumName,omitempty"`
	SortArtistName string `json:"sortArtistName,omitempty"`

	// Track / disc / dates
	TrackNumber  int32  `json:"trackNumber"`
	DiscNumber   int32  `json:"discNumber"`
	DiscSubtitle string `json:"discSubtitle,omitempty"`
	Year         int32  `json:"year"`
	Date         string `json:"date,omitempty"`
	OriginalYear int32  `json:"originalYear"`
	OriginalDate string `json:"originalDate,omitempty"`
	ReleaseYear  int32  `json:"releaseYear"`
	ReleaseDate  string `json:"releaseDate,omitempty"`

	// Audio / file
	Size       int64   `json:"size"`
	Suffix     string  `json:"suffix,omitempty"`
	Duration   float64 `json:"duration"`
	BitRate    int32   `json:"bitRate"`
	SampleRate int32   `json:"sampleRate"`
	BitDepth   *int32  `json:"bitDepth,omitempty"`
	Channels   int32   `json:"channels"`
	Codec      string  `json:"codec,omitempty"`

	// Descriptive
	Genres         []string `json:"genres,omitempty"`
	Comment        string   `json:"comment,omitempty"`
	BPM            *int32   `json:"bpm,omitempty"`
	ExplicitStatus string   `json:"explicitStatus,omitempty"`
	CatalogNum     string   `json:"catalogNum,omitempty"`
	Compilation    bool     `json:"compilation"`
	HasCoverArt    bool     `json:"hasCoverArt"`

	// MusicBrainz
	MbzRecordingID    string `json:"mbzRecordingId,omitempty"`
	MbzReleaseTrackID string `json:"mbzReleaseTrackId,omitempty"`
	MbzAlbumID        string `json:"mbzAlbumId,omitempty"`
	MbzReleaseGroupID string `json:"mbzReleaseGroupId,omitempty"`
	MbzAlbumType      string `json:"mbzAlbumType,omitempty"`
	MbzAlbumComment   string `json:"mbzAlbumComment,omitempty"`

	// ReplayGain — nil means no data; 0 is a valid measured value, so these
	// must stay pointers to distinguish "absent" from "0".
	RGAlbumGain *float64 `json:"rgAlbumGain,omitempty"`
	RGAlbumPeak *float64 `json:"rgAlbumPeak,omitempty"`
	RGTrackGain *float64 `json:"rgTrackGain,omitempty"`
	RGTrackPeak *float64 `json:"rgTrackPeak,omitempty"`

	// Timestamps (Unix epoch seconds)
	BirthTime int64 `json:"birthTime"`
	CreatedAt int64 `json:"createdAt"`
	UpdatedAt int64 `json:"updatedAt"`

	// AverageRating is the track's mean rating across all users (always set; 0 when unrated).
	AverageRating float64 `json:"averageRating"`

	// Per-user annotations, set only for a user-scoped match. Timestamps are Unix
	// seconds; a nil pointer means "no value".
	Starred   bool   `json:"starred,omitempty"`
	StarredAt *int64 `json:"starredAt,omitempty"`
	Rating    int32  `json:"rating,omitempty"`
	PlayCount int64  `json:"playCount,omitempty"`
	PlayDate  *int64 `json:"playDate,omitempty"`

	// Composite
	Tags map[string][]string `json:"tags,omitempty"`
	// Participants lists the track's artists across all roles, each tagged with its Role.
	Participants []ArtistRef `json:"participants,omitempty"`
}
