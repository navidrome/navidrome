package model

import "errors"

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
