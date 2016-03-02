package domain

type ArtistInfo struct {
	ArtistId string
	Artist string
}

type ArtistIndex struct {
	Id string
	Artists []ArtistInfo
}


type ArtistIndexRepository interface {
	Put(m *ArtistIndex) error
	Get(id string) (*ArtistIndex, error)
	GetAll() ([]ArtistIndex, error)
}
