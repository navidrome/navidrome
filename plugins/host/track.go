package host

// Artist is a trimmed, public projection of an artist that participated in a track.
type Artist struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	SortName    string `json:"sortName,omitempty"`
	MbzArtistID string `json:"mbzArtistId,omitempty"`
	SubRole     string `json:"subRole,omitempty"`
}

// Track is a stable, public projection of a library media file for plugin consumption.
// It is a sane subset of the internal model.MediaFile, intended for reuse across host
// services and capabilities. Timestamps are Unix epoch seconds.
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

	// Composite
	Tags         map[string][]string `json:"tags,omitempty"`
	Participants map[string][]Artist `json:"participants,omitempty"`
}
