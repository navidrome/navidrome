package model

type Genre struct {
	ID         string `json:"id"               orm:"column(id)"`
	Name       string `json:"name"`
	SongCount  int    `json:"-"`
	AlbumCount int    `json:"-"`
}

type Genres []Genre

type GenreRepository interface {
	GetAll(...QueryOptions) (Genres, error)
	Put(*Genre) error
}
