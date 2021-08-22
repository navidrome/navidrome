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
	Error  error
}

func (m *MockShareRepo) Save(entity interface{}) (string, error) {
	if m.Error != nil {
		return "", m.Error
	}
	m.Entity = entity
	return "id", nil
}

func (m *MockShareRepo) Update(entity interface{}, cols ...string) error {
	if m.Error != nil {
		return m.Error
	}
	m.Entity = entity
	m.Cols = cols
	return nil
}
