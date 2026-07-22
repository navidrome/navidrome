package tests

import (
	"time"

	"github.com/navidrome/navidrome/model"
)

type MockItemArtworkRepo struct {
	model.ItemArtworkRepository
	Data map[string]model.ItemArtwork // key: kind + "|" + id + "|" + imageType
	Err  error
}

func CreateMockItemArtworkRepo() *MockItemArtworkRepo {
	return &MockItemArtworkRepo{Data: map[string]model.ItemArtwork{}}
}

func iaKey(kind, id, imageType string) string { return kind + "|" + id + "|" + imageType }

func (m *MockItemArtworkRepo) Get(kind, id, imageType string) (*model.ItemArtwork, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	if ia, ok := m.Data[iaKey(kind, id, imageType)]; ok {
		return &ia, nil
	}
	return nil, model.ErrNotFound
}

func (m *MockItemArtworkRepo) Put(ia *model.ItemArtwork) error {
	if m.Err != nil {
		return m.Err
	}
	ia.UpdatedAt = time.Now()
	m.Data[iaKey(ia.ItemKind, ia.ItemID, ia.ImageType)] = *ia
	return nil
}

func (m *MockItemArtworkRepo) DeleteForItem(kind, id string) error {
	if m.Err != nil {
		return m.Err
	}
	for k, ia := range m.Data {
		if ia.ItemKind == kind && ia.ItemID == id {
			delete(m.Data, k)
		}
	}
	return nil
}

func (m *MockItemArtworkRepo) GetInfoForItems(kind string, ids []string) (map[string]model.ItemArtworkInfo, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	res := map[string]model.ItemArtworkInfo{}
	for _, id := range ids {
		if ia, ok := m.Data[iaKey(kind, id, model.ImageTypePrimary)]; ok {
			res[id] = model.ItemArtworkInfo{ItemID: id, Hash: ia.Hash, Absent: ia.Hash == ""}
		}
	}
	return res, nil
}

func (m *MockItemArtworkRepo) EnqueueStaleAbsent(kind string, attemptedBefore time.Time) (int64, error) {
	return 0, m.Err
}
