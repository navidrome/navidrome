package tests

import (
	"time"

	"github.com/navidrome/navidrome/model"
)

type MockArtworkRepo struct {
	model.ArtworkRepository
	Data map[string]model.Artwork
	Err  error
}

func CreateMockArtworkRepo() *MockArtworkRepo {
	return &MockArtworkRepo{Data: map[string]model.Artwork{}}
}

func (m *MockArtworkRepo) Get(hash string) (*model.Artwork, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	if a, ok := m.Data[hash]; ok {
		return &a, nil
	}
	return nil, model.ErrNotFound
}

func (m *MockArtworkRepo) Put(a *model.Artwork) error {
	if m.Err != nil {
		return m.Err
	}
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now()
	}
	m.Data[a.Hash] = *a
	return nil
}

func (m *MockArtworkRepo) GetBatch(hashes []string) (map[string]model.Artwork, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	res := map[string]model.Artwork{}
	for _, h := range hashes {
		if a, ok := m.Data[h]; ok {
			res[h] = a
		}
	}
	return res, nil
}

func (m *MockArtworkRepo) GetOrphanHashes(createdBefore time.Time) ([]string, error) {
	return nil, m.Err
}

func (m *MockArtworkRepo) Delete(hashes ...string) error {
	if m.Err != nil {
		return m.Err
	}
	for _, h := range hashes {
		delete(m.Data, h)
	}
	return nil
}
