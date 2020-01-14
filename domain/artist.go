package domain

type Artist struct {
	ID         string
	Name       string
	AlbumCount int
}

type ArtistRepository interface {
	BaseRepository
	Put(m *Artist) error
	Get(id string) (*Artist, error)
	PurgeInactive(active Artists) error
	Search(q string, offset int, size int) (Artists, error)
}

type Artists []Artist
