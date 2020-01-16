package model

import "errors"

var (
	ErrNotFound = errors.New("data not found")
)

// Filters use the same operators as Beego ORM: See https://beego.me/docs/mvc/model/query.md#operators
// Ex: var q = QueryOptions{Filters: Filters{"name__istartswith": "Deluan","age__gt": 25}}
// All conditions will be ANDed together
// TODO Implement filter in repositories' methods
type Filters map[string]interface{}

type QueryOptions struct {
	SortBy  string
	Desc    bool
	Offset  int
	Size    int
	Filters Filters
}
