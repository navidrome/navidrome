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
	Starred      bool
	PlayCount    int
	PlayDate     time.Time
	SongCount    int
	Duration     int
	Rating       int
	Genre        string
	StarredAt    time.Time
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
	GetAllIds() ([]string, error)
	GetStarred(...QueryOptions) (Albums, error)
	Search(q string, offset int, size int) (Albums, error)
	Refresh(ids ...string) error
	PurgeEmpty() error
	SetStar(star bool, ids ...string) error
	MarkAsPlayed(id string, playDate time.Time) error
}
