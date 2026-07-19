package model

type Genre struct {
	ID         string `structs:"id" json:"id,omitempty" toml:"id,omitempty" yaml:"id,omitempty"`
	Name       string `structs:"name" json:"name"`
	SongCount  int    `structs:"-" json:"songCount,omitempty" toml:"-" yaml:"-"`
	AlbumCount int    `structs:"-" json:"albumCount,omitempty" toml:"-" yaml:"-"`
}

type Genres []Genre

type GenreRepository interface {
	GetAll(...QueryOptions) (Genres, error)
}
