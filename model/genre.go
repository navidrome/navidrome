package model

type Genre struct {
	ID         string `structs:"id" json:"id"`
	Name       string `structs:"name" json:"name"`
	SongCount  int    `structs:"-" json:"-"`
	AlbumCount int    `structs:"-" json:"-"`
}

type Genres []Genre

type GenreRepository interface {
	GetAll(...QueryOptions) (Genres, error)
	Put(*Genre) error
}
