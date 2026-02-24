package model

type SearchableRepository[T any] interface {
	Search(q string, options ...QueryOptions) (T, error)
}
