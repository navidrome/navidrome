package domain

import "errors"

type BaseRepository interface {
	NewId(fields ...string) string
	CountAll() (int64, error)
	Exists(id string) (bool, error)
}

var (
	ErrNotFound = errors.New("Data not found")
)

type QueryOptions struct {
	SortBy string
	Alpha  bool
	Desc   bool
	Offset int
	Size   int
}
