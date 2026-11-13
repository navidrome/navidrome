package tests

import (
	"context"

	"github.com/navidrome/navidrome/model"
)

type MockShareRepo struct {
	model.ShareRepository

	Entity any
	ID     string
	Cols   []string
	Error  error
}

func (m *MockShareRepo) Save(_ context.Context, s *model.Share) (string, error) {
	if m.Error != nil {
		return "", m.Error
	}
	if s.ID == "" {
		s.ID = "id"
	}
	m.Entity = s
	return s.ID, nil
}

func (m *MockShareRepo) Update(_ context.Context, id string, entity model.Share, cols ...string) error {
	if m.Error != nil {
		return m.Error
	}
	m.ID = id
	m.Entity = &entity
	m.Cols = cols
	return nil
}

func (m *MockShareRepo) Exists(_ context.Context, id string) (bool, error) {
	if m.Error != nil {
		return false, m.Error
	}
	return id == m.ID, nil
}

func (m *MockShareRepo) Get(_ context.Context, id string) (*model.Share, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	if id != m.ID {
		return nil, model.ErrNotFound
	}
	if s, ok := m.Entity.(*model.Share); ok {
		return s, nil
	}
	return &model.Share{ID: id}, nil
}
