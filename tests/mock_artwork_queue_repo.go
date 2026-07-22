package tests

import (
	"sort"
	"time"

	"github.com/navidrome/navidrome/model"
)

type MockArtworkQueueRepo struct {
	model.ArtworkQueueRepository
	Data map[string]model.ArtworkQueueItem // keyed by iaKey(kind, id, imageType)
	Err  error
	// ItemArtworkSource, when set, backs EnqueueStaleAbsent with real item_artwork state.
	ItemArtworkSource *MockArtworkRepo
	// ExistingIDs, keyed by item_kind, backs PurgeDangling; a nil per-kind map keeps that kind.
	ExistingIDs map[string]map[string]bool
}

func CreateMockArtworkQueueRepo() *MockArtworkQueueRepo {
	return &MockArtworkQueueRepo{Data: map[string]model.ArtworkQueueItem{}}
}

func (m *MockArtworkQueueRepo) Enqueue(items ...model.ArtworkQueueItem) error {
	if m.Err != nil {
		return m.Err
	}
	now := time.Now()
	for _, it := range items {
		if it.ImageType == "" {
			it.ImageType = model.ImageTypePrimary
		}
		k := iaKey(it.ItemKind, it.ItemID, it.ImageType)
		// Mirror the SQL: retry_at/enqueued_at are server-set, never taken from the caller.
		if prev, ok := m.Data[k]; ok {
			prev.Priority = max(prev.Priority, it.Priority)
			prev.RetryAt = now
			m.Data[k] = prev
			continue
		}
		it.Attempts = 0
		it.RetryAt = now
		it.EnqueuedAt = now
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

func (m *MockArtworkQueueRepo) MarkFailedIfUnchanged(kind, id, imageType string, seenRetryAt, retryAt time.Time) error {
	if m.Err != nil {
		return m.Err
	}
	k := iaKey(kind, id, imageType)
	if it, ok := m.Data[k]; ok && it.RetryAt.Equal(seenRetryAt) {
		it.Attempts++
		it.RetryAt = retryAt
		m.Data[k] = it
	}
	return nil
}

func (m *MockArtworkQueueRepo) Delete(kind, id, imageType string) error {
	if m.Err != nil {
		return m.Err
	}
	delete(m.Data, iaKey(kind, id, imageType))
	return nil
}

func (m *MockArtworkQueueRepo) DeleteIfUnchanged(kind, id, imageType string, retryAt time.Time) error {
	if m.Err != nil {
		return m.Err
	}
	k := iaKey(kind, id, imageType)
	if it, ok := m.Data[k]; ok && it.RetryAt.Equal(retryAt) {
		delete(m.Data, k)
	}
	return nil
}

func (m *MockArtworkQueueRepo) PurgeDangling() (int64, error) {
	if m.Err != nil {
		return 0, m.Err
	}
	var purged int64
	for k, it := range m.Data {
		existing := m.ExistingIDs[it.ItemKind]
		if existing == nil {
			continue
		}
		if !existing[it.ItemID] {
			delete(m.Data, k)
			purged++
		}
	}
	return purged, nil
}

func (m *MockArtworkQueueRepo) Count() (int64, error) {
	if m.Err != nil {
		return 0, m.Err
	}
	return int64(len(m.Data)), nil
}

func (m *MockArtworkQueueRepo) EnqueueStaleAbsent(kind string, attemptedBefore time.Time) (int64, error) {
	if m.Err != nil || m.ItemArtworkSource == nil {
		return 0, m.Err
	}
	now := time.Now()
	var inserted int64
	for _, ia := range m.ItemArtworkSource.ItemData {
		if ia.ItemKind != kind || ia.Hash != "" || !ia.AttemptedAt.Before(attemptedBefore) {
			continue
		}
		k := iaKey(ia.ItemKind, ia.ItemID, ia.ImageType)
		if _, ok := m.Data[k]; ok { // DO NOTHING: never touch existing queue rows
			continue
		}
		m.Data[k] = model.ArtworkQueueItem{
			ItemKind:   ia.ItemKind,
			ItemID:     ia.ItemID,
			ImageType:  ia.ImageType,
			Priority:   model.ArtworkPriorityRecheck,
			RetryAt:    now,
			EnqueuedAt: now,
		}
		inserted++
	}
	return inserted, nil
}
