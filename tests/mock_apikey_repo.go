package tests

import (
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"strings"
)

func CreateMockApiKeyRepo() *MockedAPIKeyRepo {
	return &MockedAPIKeyRepo{
		Data: map[string]*model.APIKey{},
	}
}

type MockedAPIKeyRepo struct {
	model.APIKeyRepository
	Error error
	Data  map[string]*model.APIKey
}

func (m *MockedAPIKeyRepo) CountAll(_ ...model.QueryOptions) (int64, error) {
	if m.Error != nil {
		return 0, m.Error
	}
	return int64(len(m.Data)), nil
}

func (m *MockedAPIKeyRepo) Save(entity interface{}) (string, error) {
	if m.Error != nil {
		return "", m.Error
	}
	apiKey := entity.(*model.APIKey)
	apiKey.Key = "nav_" + id.NewRandom()
	m.Data[strings.ToLower(apiKey.Key)] = apiKey
	return apiKey.ID, nil
}

func (m *MockedAPIKeyRepo) FindByKey(key string) (*model.APIKey, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	apiKey, exists := m.Data[strings.ToLower(key)]
	if !exists {
		return nil, model.ErrNotFound
	}
	return apiKey, nil
}

func (m *MockedAPIKeyRepo) Get(id string) (*model.APIKey, error) {
	if m.Error != nil {
		return nil, m.Error
	}

	for _, apiKey := range m.Data {
		if apiKey.ID == id {
			return apiKey, nil
		}
	}
	return nil, model.ErrNotFound
}
