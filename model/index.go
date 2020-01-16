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

// TODO Combine ArtistIndex with Artist
type ArtistIndexRepository interface {
	Put(m *ArtistIndex) error
	Refresh() error
	GetAll() (ArtistIndexes, error)
	DeleteAll() error
}
