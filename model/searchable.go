package model

type SearchableRepository[T any] interface {
	Search(q string, offset, size int) (T, error)
}
