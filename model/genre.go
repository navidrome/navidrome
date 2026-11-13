package model

import (
	"context"

	"github.com/deluan/rest"
)

type Genre struct {
	ID         string `structs:"id" json:"id,omitempty" toml:"id,omitempty" yaml:"id,omitempty"`
	Name       string `structs:"name" json:"name"`
	SongCount  int    `structs:"-" json:"-" toml:"-" yaml:"-"`
	AlbumCount int    `structs:"-" json:"-" toml:"-" yaml:"-"`
}

type Genres []Genre

type GenreRepository interface {
	rest.Repository[Genre]
	GetAll(ctx context.Context, options ...QueryOptions) (Genres, error)
	Get(ctx context.Context, id string) (*Genre, error)
}
