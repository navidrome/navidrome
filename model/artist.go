package model

type Artist struct {
	ID         string
	Name       string
	AlbumCount int
}

type ArtistRepository interface {
	CountAll() (int64, error)
	Exists(id string) (bool, error)
	Put(m *Artist) error
	Get(id string) (*Artist, error)
	PurgeInactive(active Artists) error
	Search(q string, offset int, size int) (Artists, error)
	Refresh(ids ...string) error
}

type Artists []Artist
