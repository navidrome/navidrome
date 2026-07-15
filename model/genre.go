package model

import "iter"

type Genre struct {
	ID         string `structs:"id" json:"id,omitempty" toml:"id,omitempty" yaml:"id,omitempty"`
	Name       string `structs:"name" json:"name"`
	SongCount  int    `structs:"-" json:"-" toml:"-" yaml:"-"`
	AlbumCount int    `structs:"-" json:"-" toml:"-" yaml:"-"`
}

type Genres []Genre

type GenreCursor iter.Seq2[Genre, error]

type GenreRepository interface {
	GetAll(...QueryOptions) (Genres, error)
	// GetCursor returns the same rows as GetAll, yielded one at a time, so large result sets can be
	// streamed without materializing every genre.
	GetCursor(...QueryOptions) (GenreCursor, error)
}
