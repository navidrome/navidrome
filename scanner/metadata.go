package scanner

import "time"

type Metadata interface {
	Title() string
	Album() string
	Artist() string
	AlbumArtist() string
	SortTitle() string
	SortAlbum() string
	SortArtist() string
	SortAlbumArtist() string
	Composer() string
	Genre() string
	Year() int
	TrackNumber() (int, int)
	DiscNumber() (int, int)
	DiscSubtitle() string
	HasPicture() bool
	Comment() string
	Compilation() bool
	Duration() float32
	BitRate() int
	ModificationTime() time.Time
	FilePath() string
	Suffix() string
	Size() int64
}

type MetadataExtractor interface {
	Extract(files ...string) (map[string]Metadata, error)
}
