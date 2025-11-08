package tests

import (
	"encoding/base64"

	"github.com/navidrome/navidrome/model"
)

func CreateMockPlayerRepo() *MockedPlayerRepo {
	return &MockedPlayerRepo{
		Data: make(map[string]*model.Player),
	}
}

type MockedPlayerRepo struct {
	model.PlayerRepository
	Error error
	Data  map[string]*model.Player
}

func (m *MockedPlayerRepo) Get(id string) (*model.Player, error) {
	if m.Error != nil {
		return nil, m.Error
	}

	player, exists := m.Data[id]
	if !exists {
		return nil, model.ErrNotFound
	}
	return player, nil
}

func (m *MockedPlayerRepo) FindMatch(userId, client, userAgent string) (*model.Player, error) {
	if m.Error != nil {
		return nil, m.Error
	}

	for _, player := range m.Data {
		if player.UserId == userId && player.Client == client && player.UserAgent == userAgent {
			return player, nil
		}
	}
	return nil, model.ErrNotFound
}

func (m *MockedPlayerRepo) Put(p *model.Player) error {
	if m.Error != nil {
		return m.Error
	}

	if p.ID == "" {
		p.ID = base64.StdEncoding.EncodeToString([]byte(p.Name + "_" + p.UserId + "_" + p.Client))
	}
	m.Data[p.ID] = p
	return nil
}

func (m *MockedPlayerRepo) CountAll(_ ...model.QueryOptions) (int64, error) {
	if m.Error != nil {
		return 0, m.Error
	}
	return int64(len(m.Data)), nil
}

func (m *MockedPlayerRepo) CountByClient(_ ...model.QueryOptions) (map[string]int64, error) {
	if m.Error != nil {
		return nil, m.Error
	}

	result := make(map[string]int64)
	for _, player := range m.Data {
		result[player.Client]++
	}
	return result, nil
}

func (m *MockedPlayerRepo) FindByAPIKey(key string) (*model.Player, error) {
	if m.Error != nil {
		return nil, m.Error
	}

	for _, player := range m.Data {
		if player.APIKey == key {
			return player, nil
		}
	}
	return nil, model.ErrNotFound
}
