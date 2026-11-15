package tests

import "github.com/navidrome/navidrome/model"

// MockedGrantRepo exists so MockDataStore satisfies DataStore; auth tests use a real database.
type MockedGrantRepo struct {
	model.GrantRepository
}
