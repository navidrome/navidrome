package tests

import (
	"sort"
	"time"

	"github.com/navidrome/navidrome/model"
)

type MockArtworkQueueRepo struct {
	model.ArtworkQueueRepository
	Data map[string]model.ArtworkQueueItem // key: kind + "|" + id + "|" + imageType
	Err  error
}

func CreateMockArtworkQueueRepo() *MockArtworkQueueRepo {
	return &MockArtworkQueueRepo{Data: map[string]model.ArtworkQueueItem{}}
}

func (m *MockArtworkQueueRepo) Enqueue(items ...model.ArtworkQueueItem) error {
	if m.Err != nil {
		return m.Err
	}
	for _, it := range items {
		k := iaKey(it.ItemKind, it.ItemID, it.ImageType)
		if prev, ok := m.Data[k]; ok && prev.Priority > it.Priority {
			it.Priority = prev.Priority
		}
		if it.EnqueuedAt.IsZero() {
			it.EnqueuedAt = time.Now()
		}
		if it.RetryAt.IsZero() {
			it.RetryAt = time.Now()
		}
		m.Data[k] = it
	}
	return nil
}

func (m *MockArtworkQueueRepo) DequeueBatch(n int) ([]model.ArtworkQueueItem, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	var res []model.ArtworkQueueItem
	now := time.Now()
	for _, it := range m.Data {
		if !it.RetryAt.After(now) {
			res = append(res, it)
		}
	}
	sort.Slice(res, func(i, j int) bool {
		if res[i].Priority != res[j].Priority {
			return res[i].Priority > res[j].Priority
		}
		return res[i].EnqueuedAt.Before(res[j].EnqueuedAt)
	})
	if len(res) > n {
		res = res[:n]
	}
	return res, nil
}

func (m *MockArtworkQueueRepo) MarkFailed(kind, id, imageType string, retryAt time.Time) error {
	if m.Err != nil {
		return m.Err
	}
	k := iaKey(kind, id, imageType)
	it, ok := m.Data[k]
	if !ok {
		return model.ErrNotFound
	}
	it.Attempts++
	it.RetryAt = retryAt
	m.Data[k] = it
	return nil
}

func (m *MockArtworkQueueRepo) Delete(kind, id, imageType string) error {
	if m.Err != nil {
		return m.Err
	}
	delete(m.Data, iaKey(kind, id, imageType))
	return nil
}

func (m *MockArtworkQueueRepo) Count() (int64, error) {
	if m.Err != nil {
		return 0, m.Err
	}
	return int64(len(m.Data)), nil
}
