package model

import "time"

type Album struct {
	ID           string    `json:"id"            orm:"column(id)"`
	Name         string    `json:"name"`
	ArtistID     string    `json:"artistId"      orm:"pk;column(artist_id)"`
	CoverArtPath string    `json:"coverArtPath"`
	CoverArtId   string    `json:"coverArtId"`
	Artist       string    `json:"artist"`
	AlbumArtist  string    `json:"albumArtist"`
	Year         int       `json:"year"`
	Compilation  bool      `json:"compilation"`
	SongCount    int       `json:"songCount"`
	Duration     int       `json:"duration"`
	Genre        string    `json:"genre"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`

	// Annotations
	PlayCount int       `json:"-"   orm:"-"`
	PlayDate  time.Time `json:"-"   orm:"-"`
	Rating    int       `json:"-"   orm:"-"`
	Starred   bool      `json:"-"   orm:"-"`
	StarredAt time.Time `json:"-"   orm:"-"`
}

type Albums []Album

type AlbumRepository interface {
	CountAll(...QueryOptions) (int64, error)
	Exists(id string) (bool, error)
	Put(m *Album) error
	Get(id string) (*Album, error)
	FindByArtist(artistId string) (Albums, error)
	GetAll(...QueryOptions) (Albums, error)
	GetRandom(...QueryOptions) (Albums, error)
	GetStarred(options ...QueryOptions) (Albums, error)
	Search(q string, offset int, size int) (Albums, error)
	Refresh(ids ...string) error
	PurgeEmpty() error
}
