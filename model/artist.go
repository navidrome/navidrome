package model

import "time"

type Artist struct {
	ID         string `json:"id"          orm:"column(id)"`
	Name       string `json:"name"`
	AlbumCount int    `json:"albumCount"  orm:"column(album_count)"`

	// Annotations
	PlayCount int       `json:"-"   orm:"-"`
	PlayDate  time.Time `json:"-"   orm:"-"`
	Rating    int       `json:"-"   orm:"-"`
	Starred   bool      `json:"-"   orm:"-"`
	StarredAt time.Time `json:"-"   orm:"-"`
}

type Artists []Artist

type ArtistIndex struct {
	ID      string
	Artists Artists
}
type ArtistIndexes []ArtistIndex

type ArtistRepository interface {
	CountAll(options ...QueryOptions) (int64, error)
	Exists(id string) (bool, error)
	Put(m *Artist) error
	Get(id string) (*Artist, error)
	GetStarred(options ...QueryOptions) (Artists, error)
	Search(q string, offset int, size int) (Artists, error)
	Refresh(ids ...string) error
	GetIndex() (ArtistIndexes, error)
	PurgeEmpty() error
	AnnotatedRepository
}
