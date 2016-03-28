package domain

type Artist struct {
	Id         string
	Name       string
	AlbumCount int
}

type ArtistRepository interface {
	BaseRepository
	Put(m *Artist) error
	Get(id string) (*Artist, error)
	PurgeInactive(active Artists) ([]string, error)
}

type Artists []Artist
