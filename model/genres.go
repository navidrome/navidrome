package model

type Genre struct {
	ID         string `json:"id"               orm:"column(id)"`
	Name       string
	SongCount  int
	AlbumCount int
}

type Genres []Genre

type GenreRepository interface {
	GetAll() (Genres, error)
	Put(m *Genre) error
}
