package tests

import (
	"context"
	"maps"
	"slices"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
)

func CreateMockPlayerRepo() *MockPlayerRepo {
	return &MockPlayerRepo{Data: map[string]*model.Player{}, APIKeys: map[string]string{}}
}

// MockPlayerRepo keeps API keys in plaintext (key -> player ID); hashing belongs to the real repository.
type MockPlayerRepo struct {
	model.PlayerRepository
	Error   error
	Data    map[string]*model.Player
	APIKeys map[string]string
}

func (m *MockPlayerRepo) Get(_ context.Context, id string) (*model.Player, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	p, ok := m.Data[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	cp := *p
	cp.HasAPIKey = slices.Contains(slices.Collect(maps.Values(m.APIKeys)), id)
	return &cp, nil
}

func (m *MockPlayerRepo) Put(_ context.Context, p *model.Player) error {
	if m.Error != nil {
		return m.Error
	}
	if p.ID == "" {
		p.ID = id.NewRandom()
	}
	cp := *p
	m.Data[p.ID] = &cp
	return nil
}

func (m *MockPlayerRepo) FindByAPIKey(ctx context.Context, key string) (*model.Player, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	if playerID, ok := m.APIKeys[key]; ok {
		return m.Get(ctx, playerID)
	}
	return nil, model.ErrNotFound
}

func (m *MockPlayerRepo) SetAPIKey(_ context.Context, playerID, key string) error {
	if m.Error != nil {
		return m.Error
	}
	if _, ok := m.Data[playerID]; !ok {
		return model.ErrNotFound
	}
	m.removeKeys(playerID)
	if key != "" {
		m.APIKeys[key] = playerID
	}
	return nil
}

func (m *MockPlayerRepo) removeKeys(playerID string) {
	maps.DeleteFunc(m.APIKeys, func(_ string, v string) bool { return v == playerID })
}
