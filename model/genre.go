package model

type Genre struct {
	ID         string `structs:"id" json:"id,omitempty" toml:"id,omitempty" yaml:"id,omitempty"`
	Name       string `structs:"name" json:"name"`
	SongCount  int    `structs:"-" json:"-" toml:"-" yaml:"-"`
	AlbumCount int    `structs:"-" json:"-" toml:"-" yaml:"-"`
}

type Genres []Genre

type GenreRepository interface {
	GetAll(...QueryOptions) (Genres, error)
	Put(*Genre) error
}
