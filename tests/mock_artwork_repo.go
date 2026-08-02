package tests

import (
	"maps"
	"sync"
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/slice"
)

type MockArtworkRepo struct {
	model.ArtworkRepository
	// mu guards the maps so the worker's concurrent drain can hit this mock race-free.
	mu       sync.Mutex
	Data     map[string]model.Artwork
	ItemData map[string]model.ItemArtwork // keyed by iaKey(kind, id, imageType)
	Err      error
	// ExistingIDs, keyed by item_kind, backs PurgeDanglingItemArtwork; nil map keeps everything.
	ExistingIDs map[string]map[string]bool
}

func CreateMockArtworkRepo() *MockArtworkRepo {
	return &MockArtworkRepo{Data: map[string]model.Artwork{}, ItemData: map[string]model.ItemArtwork{}}
}

func iaKey(kind, id, imageType string) string { return kind + "|" + id + "|" + imageType }

func (m *MockArtworkRepo) GetImage(hash string) (*model.Artwork, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return nil, m.Err
	}
	if a, ok := m.Data[hash]; ok {
		return &a, nil
	}
	return nil, model.ErrNotFound
}

func (m *MockArtworkRepo) PutImage(a *model.Artwork) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return m.Err
	}
	// Mirrors the SQL repository: every upsert refreshes created_at. Age fixtures via Data directly.
	a.CreatedAt = time.Now()
	m.Data[a.Hash] = *a
	return nil
}

func (m *MockArtworkRepo) GetMimeByHash() (map[string]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
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
	m.mu.Lock()
	defer m.mu.Unlock()
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

func (m *MockArtworkRepo) DeleteOrphans(createdBefore time.Time) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return 0, m.Err
	}
	var deleted int64
	for h, a := range m.Data {
		if m.referenced(h) || !a.CreatedAt.Before(createdBefore) {
			continue
		}
		delete(m.Data, h)
		deleted++
	}
	return deleted, nil
}

func (m *MockArtworkRepo) referenced(hash string) bool {
	for _, ia := range m.ItemData {
		if ia.Hash == hash {
			return true
		}
	}
	return false
}

func (m *MockArtworkRepo) GetItemArtwork(kind model.Kind, id, imageType string) (*model.ItemArtwork, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return nil, m.Err
	}
	if ia, ok := m.ItemData[iaKey(kind.Prefix(), id, imageType)]; ok {
		return &ia, nil
	}
	return nil, model.ErrNotFound
}

func (m *MockArtworkRepo) PutItemArtwork(ia *model.ItemArtwork) error {
	m.mu.Lock()
	defer m.mu.Unlock()
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

func (m *MockArtworkRepo) DeleteForItem(kind model.Kind, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return m.Err
	}
	maps.DeleteFunc(m.ItemData, func(_ string, ia model.ItemArtwork) bool {
		return ia.ItemKind == kind.Prefix() && ia.ItemID == id
	})
	return nil
}

func (m *MockArtworkRepo) DeleteForItems(kind model.Kind, ids []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return m.Err
	}
	idSet := slice.ToSet(ids)
	maps.DeleteFunc(m.ItemData, func(_ string, ia model.ItemArtwork) bool {
		_, ok := idSet[ia.ItemID]
		return ok && ia.ItemKind == kind.Prefix()
	})
	return nil
}

func (m *MockArtworkRepo) GetInfoForItems(kind model.Kind, ids []string) (map[string]model.ItemArtworkInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return nil, m.Err
	}
	res := map[string]model.ItemArtworkInfo{}
	for _, id := range ids {
		if ia, ok := m.ItemData[iaKey(kind.Prefix(), id, model.ImageTypePrimary)]; ok {
			info := model.ItemArtworkInfo{ItemID: id, Hash: ia.Hash}
			if a, ok := m.Data[ia.Hash]; ok {
				info.BlurHash, info.ThumbHash = a.BlurHash, a.ThumbHash
				info.Width, info.Height = a.Width, a.Height
			}
			res[id] = info
		}
	}
	return res, nil
}
