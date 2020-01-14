package domain

import "time"

type Album struct {
	ID           string
	Name         string
	ArtistID     string `parent:"artist"`
	CoverArtPath string
	CoverArtId   string
	Artist       string
	AlbumArtist  string
	Year         int `idx:"Year"`
	Compilation  bool
	Starred      bool
	PlayCount    int
	PlayDate     time.Time
	SongCount    int
	Duration     int
	Rating       int
	Genre        string
	StarredAt    time.Time `idx:"Starred"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Albums []Album

type AlbumRepository interface {
	BaseRepository
	Put(m *Album) error
	Get(id string) (*Album, error)
	FindByArtist(artistId string) (Albums, error)
	GetAll(...QueryOptions) (Albums, error)
	PurgeInactive(active Albums) error
	GetAllIds() ([]string, error)
	GetStarred(...QueryOptions) (Albums, error)
	Search(q string, offset int, size int) (Albums, error)
}
