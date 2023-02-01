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
	ID     string
	Cols   []string
	Error  error
}

func (m *MockShareRepo) Save(entity interface{}) (string, error) {
	if m.Error != nil {
		return "", m.Error
	}
	s := entity.(*model.Share)
	if s.ID == "" {
		s.ID = "id"
	}
	m.Entity = s
	return s.ID, nil
}

func (m *MockShareRepo) Update(id string, entity interface{}, cols ...string) error {
	if m.Error != nil {
		return m.Error
	}
	m.ID = id
	m.Entity = entity
	m.Cols = cols
	return nil
}

func (m *MockShareRepo) Exists(id string) (bool, error) {
	if m.Error != nil {
		return false, m.Error
	}
	return id == m.ID, nil
}
