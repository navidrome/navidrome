package model

import "errors"

type BaseRepository interface {
	CountAll() (int64, error)
	Exists(id string) (bool, error)
}

var (
	ErrNotFound = errors.New("data not found")
)

type QueryOptions struct {
	SortBy string
	Alpha  bool
	Desc   bool
	Offset int
	Size   int
}
