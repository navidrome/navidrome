package nativeapi

import (
	"github.com/deluan/rest"
)

// mappedRepository wraps a rest.Repository and transforms its Read/ReadAll results
// using a mapping function. This allows the native API to return different types than
// what the persistence layer provides, enabling calculated fields and decoupling the
// API response shape from the database model.
//
// The mapFunc receives the raw result from the underlying repository and returns the
// transformed result for the API response. It handles both single items (from Read)
// and collections (from ReadAll).
type mappedRepository struct {
	repo    rest.Repository
	mapOne  func(any) (any, error)
	mapMany func(any) (any, error)
}

func (m *mappedRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return m.repo.Count(options...)
}

func (m *mappedRepository) Read(id string) (any, error) {
	result, err := m.repo.Read(id)
	if err != nil {
		return nil, err
	}
	return m.mapOne(result)
}

func (m *mappedRepository) ReadAll(options ...rest.QueryOptions) (any, error) {
	result, err := m.repo.ReadAll(options...)
	if err != nil {
		return nil, err
	}
	return m.mapMany(result)
}

func (m *mappedRepository) EntityName() string {
	return m.repo.EntityName()
}

func (m *mappedRepository) NewInstance() any {
	return m.repo.NewInstance()
}

var _ rest.Repository = (*mappedRepository)(nil)
