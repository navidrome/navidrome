package model

import "time"

type Album struct {
	ID           string
	Name         string
	ArtistID     string
	CoverArtPath string
	CoverArtId   string
	Artist       string
	AlbumArtist  string
	Year         int
	Compilation  bool
	SongCount    int
	Duration     int
	Genre        string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Albums []Album

type AlbumRepository interface {
	CountAll() (int64, error)
	Exists(id string) (bool, error)
	Put(m *Album) error
	Get(id string) (*Album, error)
	FindByArtist(artistId string) (Albums, error)
	GetAll(...QueryOptions) (Albums, error)
	GetRandom(...QueryOptions) (Albums, error)
	GetStarred(userId string, options ...QueryOptions) (Albums, error)
	Search(q string, offset int, size int) (Albums, error)
	Refresh(ids ...string) error
	PurgeEmpty() error
}
