package model

import "context"

type SearchableRepository[T any] interface {
	Search(ctx context.Context, q string, options ...QueryOptions) (T, error)
}
