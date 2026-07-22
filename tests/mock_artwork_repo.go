package tests

import (
	"time"

	"github.com/navidrome/navidrome/model"
)

type MockArtworkRepo struct {
	model.ArtworkRepository
	Data         map[string]model.Artwork
	ItemData     map[string]model.ItemArtwork // keyed by iaKey(kind, id, imageType)
	OrphanHashes []string
	Err          error
}

func CreateMockArtworkRepo() *MockArtworkRepo {
	return &MockArtworkRepo{Data: map[string]model.Artwork{}, ItemData: map[string]model.ItemArtwork{}}
}

func iaKey(kind, id, imageType string) string { return kind + "|" + id + "|" + imageType }

func (m *MockArtworkRepo) GetImage(hash string) (*model.Artwork, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	if a, ok := m.Data[hash]; ok {
		return &a, nil
	}
	return nil, model.ErrNotFound
}

func (m *MockArtworkRepo) PutImage(a *model.Artwork) error {
	if m.Err != nil {
		return m.Err
	}
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now()
	}
	m.Data[a.Hash] = *a
	return nil
}

func (m *MockArtworkRepo) GetImages(hashes []string) (map[string]model.Artwork, error) {
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
	return m.OrphanHashes, m.Err
}

func (m *MockArtworkRepo) GetAllHashes() ([]string, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	hashes := make([]string, 0, len(m.Data))
	for h := range m.Data {
		hashes = append(hashes, h)
	}
	return hashes, nil
}

func (m *MockArtworkRepo) DeleteOrphans(createdBefore time.Time, hashes []string) error {
	if m.Err != nil {
		return m.Err
	}
	// Mirror the SQL reference re-check; createdBefore is ignored in the mock for simplicity.
	for _, h := range hashes {
		if m.referenced(h) {
			continue
		}
		delete(m.Data, h)
	}
	return nil
}

func (m *MockArtworkRepo) referenced(hash string) bool {
	for _, ia := range m.ItemData {
		if ia.Hash == hash {
			return true
		}
	}
	return false
}

func (m *MockArtworkRepo) GetItemArtwork(kind, id, imageType string) (*model.ItemArtwork, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	if ia, ok := m.ItemData[iaKey(kind, id, imageType)]; ok {
		return &ia, nil
	}
	return nil, model.ErrNotFound
}

func (m *MockArtworkRepo) PutItemArtwork(ia *model.ItemArtwork) error {
	if m.Err != nil {
		return m.Err
	}
	if ia.ImageType == "" {
		ia.ImageType = model.ImageTypePrimary
	}
	ia.UpdatedAt = time.Now()
	if ia.AttemptedAt.IsZero() {
		ia.AttemptedAt = ia.UpdatedAt
	}
	m.ItemData[iaKey(ia.ItemKind, ia.ItemID, ia.ImageType)] = *ia
	return nil
}

func (m *MockArtworkRepo) DeleteForItem(kind, id string) error {
	if m.Err != nil {
		return m.Err
	}
	for k, ia := range m.ItemData {
		if ia.ItemKind == kind && ia.ItemID == id {
			delete(m.ItemData, k)
		}
	}
	return nil
}

func (m *MockArtworkRepo) GetInfoForItems(kind string, ids []string) (map[string]model.ItemArtworkInfo, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	res := map[string]model.ItemArtworkInfo{}
	for _, id := range ids {
		if ia, ok := m.ItemData[iaKey(kind, id, model.ImageTypePrimary)]; ok {
			info := model.ItemArtworkInfo{ItemID: id, Hash: ia.Hash}
			if a, ok := m.Data[ia.Hash]; ok {
				info.BlurHash = a.BlurHash
			}
			res[id] = info
		}
	}
	return res, nil
}
