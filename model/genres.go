package model

type Genre struct {
	Name       string
	SongCount  int
	AlbumCount int
}

type Genres []Genre

type GenreRepository interface {
	GetAll() (Genres, error)
}
