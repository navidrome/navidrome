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
	// ExistingIDs, keyed by item_kind, backs PurgeDanglingItemArtwork; nil map keeps everything.
	ExistingIDs map[string]map[string]bool
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
	// Mirrors the SQL repository: every upsert refreshes created_at. Age fixtures via Data directly.
	a.CreatedAt = time.Now()
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

func (m *MockArtworkRepo) GetAllMimes() (map[string]string, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	mimes := make(map[string]string, len(m.Data))
	for h, a := range m.Data {
		mimes[h] = a.Mime
	}
	return mimes, nil
}

func (m *MockArtworkRepo) PurgeDanglingItemArtwork() (int64, error) {
	if m.Err != nil {
		return 0, m.Err
	}
	var purged int64
	for k, ia := range m.ItemData {
		// A nil per-kind map means that kind isn't tracked by the test, so keep it.
		existing := m.ExistingIDs[ia.ItemKind]
		if existing == nil {
			continue
		}
		if !existing[ia.ItemID] {
			delete(m.ItemData, k)
			purged++
		}
	}
	return purged, nil
}

func (m *MockArtworkRepo) DeleteOrphans(createdBefore time.Time, hashes []string) error {
	if m.Err != nil {
		return m.Err
	}
	// Mirror the SQL re-check: only unreferenced rows older than the cutoff are deleted.
	for _, h := range hashes {
		if m.referenced(h) {
			continue
		}
		if a, ok := m.Data[h]; ok && !a.CreatedAt.Before(createdBefore) {
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

func (m *MockArtworkRepo) DeleteForItems(kind string, ids []string) error {
	if m.Err != nil {
		return m.Err
	}
	idSet := make(map[string]bool, len(ids))
	for _, id := range ids {
		idSet[id] = true
	}
	for k, ia := range m.ItemData {
		if ia.ItemKind == kind && idSet[ia.ItemID] {
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
