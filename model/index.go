package model

type ArtistInfo struct {
	ArtistID   string
	Artist     string
	AlbumCount int
}

type ArtistIndex struct {
	ID      string
	Artists ArtistInfos
}

type ArtistInfos []ArtistInfo
type ArtistIndexes []ArtistIndex

type ArtistIndexRepository interface {
	CountAll() (int64, error)
	Exists(id string) (bool, error)
	Put(m *ArtistIndex) error
	Get(id string) (*ArtistIndex, error)
	GetAll() (ArtistIndexes, error)
	DeleteAll() error
}
