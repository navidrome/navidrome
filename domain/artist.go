package domain

type Artist struct {
	Id   string
	Name string
}

type ArtistRepository interface {
	BaseRepository
	Put(m *Artist) error
	Get(id string) (*Artist, error)
	GetByName(name string) (*Artist, error)
}
