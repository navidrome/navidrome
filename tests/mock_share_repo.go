package tests

import (
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/model"
)

type MockShareRepo struct {
	model.ShareRepository
	rest.Repository
	rest.Persistable

	Entity interface{}
	Cols   []string
	Err    error
}

func (m *MockShareRepo) Save(entity interface{}) (string, error) {
	m.Entity = entity
	return "id", m.Err
}

func (m *MockShareRepo) Update(entity interface{}, cols ...string) error {
	m.Entity = entity
	m.Cols = cols
	return m.Err
}
