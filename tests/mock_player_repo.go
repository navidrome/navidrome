package tests

import (
	"maps"

	"github.com/navidrome/navidrome/consts"
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

func (m *MockPlayerRepo) Get(id string) (*model.Player, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	p, ok := m.Data[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	cp := *p
	return &cp, nil
}

func (m *MockPlayerRepo) Put(p *model.Player) error {
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

func (m *MockPlayerRepo) FindByAPIKey(key string) (*model.Player, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	if playerID, ok := m.APIKeys[key]; ok {
		return m.Get(playerID)
	}
	return nil, model.ErrNotFound
}

func (m *MockPlayerRepo) GenerateAPIKey(playerID string) (string, error) {
	if m.Error != nil {
		return "", m.Error
	}
	p, ok := m.Data[playerID]
	if !ok {
		return "", model.ErrNotFound
	}
	m.removeKeys(playerID)
	key := consts.APIKeyPrefix + id.NewRandom()
	m.APIKeys[key] = playerID
	p.HasAPIKey = true
	return key, nil
}

func (m *MockPlayerRepo) RevokeAPIKey(playerID string) error {
	if m.Error != nil {
		return m.Error
	}
	p, ok := m.Data[playerID]
	if !ok {
		return model.ErrNotFound
	}
	m.removeKeys(playerID)
	p.HasAPIKey = false
	return nil
}

func (m *MockPlayerRepo) removeKeys(playerID string) {
	maps.DeleteFunc(m.APIKeys, func(_ string, v string) bool { return v == playerID })
}
