package domain

import "time"

type Album struct {
	Id           string
	Name         string
	ArtistId     string `parent:"artist"`
	CoverArtPath string // TODO http://stackoverflow.com/questions/13795842/linking-itunes-itc2-files-and-ituneslibrary-xml
	CoverArtId   string
	Artist       string
	AlbumArtist  string
	Year         int
	Compilation  bool
	Starred      bool
	PlayCount    int
	PlayDate     time.Time
	Rating       int
	Genre        string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Albums []Album

type AlbumRepository interface {
	BaseRepository
	Put(m *Album) error
	Get(id string) (*Album, error)
	FindByArtist(artistId string) (*Albums, error)
	GetAll(QueryOptions) (*Albums, error)
	PurgeInactive(active *Albums) error
	GetAllIds() (*[]string, error)
}
