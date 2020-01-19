package model

import "time"

type Artist struct {
	ID         string
	Name       string
	AlbumCount int
	Starred    bool
	StarredAt  time.Time
}
type Artists []Artist

type ArtistIndex struct {
	ID      string
	Artists Artists
}
type ArtistIndexes []ArtistIndex

type ArtistRepository interface {
	CountAll() (int64, error)
	Exists(id string) (bool, error)
	Put(m *Artist) error
	Get(id string) (*Artist, error)
	PurgeInactive(active Artists) error
	GetStarred(...QueryOptions) (Artists, error)
	SetStar(star bool, ids ...string) error
	Search(q string, offset int, size int) (Artists, error)
	Refresh(ids ...string) error
	GetIndex() (ArtistIndexes, error)
	PurgeEmpty() error
}
