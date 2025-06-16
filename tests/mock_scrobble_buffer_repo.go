package tests

import (
	"sync"
	"time"

	"github.com/navidrome/navidrome/model"
)

type MockedScrobbleBufferRepo struct {
	Error error
	Data  model.ScrobbleEntries
	mu    sync.RWMutex
}

func CreateMockedScrobbleBufferRepo() *MockedScrobbleBufferRepo {
	return &MockedScrobbleBufferRepo{}
}

func (m *MockedScrobbleBufferRepo) UserIDs(service string) ([]string, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	userIds := make(map[string]struct{})
	for _, e := range m.Data {
		if e.Service == service {
			userIds[e.UserID] = struct{}{}
		}
	}
	var result []string
	for uid := range userIds {
		result = append(result, uid)
	}
	return result, nil
}

func (m *MockedScrobbleBufferRepo) Enqueue(service, userId, mediaFileId string, playTime time.Time) error {
	if m.Error != nil {
		return m.Error
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Data = append(m.Data, model.ScrobbleEntry{
		MediaFile:   model.MediaFile{ID: mediaFileId},
		Service:     service,
		UserID:      userId,
		PlayTime:    playTime,
		EnqueueTime: time.Now(),
	})
	return nil
}

func (m *MockedScrobbleBufferRepo) Next(service, userId string) (*model.ScrobbleEntry, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, e := range m.Data {
		if e.Service == service && e.UserID == userId {
			return &e, nil
		}
	}
	return nil, nil
}

func (m *MockedScrobbleBufferRepo) Dequeue(entry *model.ScrobbleEntry) error {
	if m.Error != nil {
		return m.Error
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	newData := model.ScrobbleEntries{}
	for _, e := range m.Data {
		if e.Service == entry.Service && e.UserID == entry.UserID && e.PlayTime == entry.PlayTime && e.MediaFile.ID == entry.MediaFile.ID {
			continue
		}
		newData = append(newData, e)
	}
	m.Data = newData
	return nil
}

func (m *MockedScrobbleBufferRepo) Length() (int64, error) {
	if m.Error != nil {
		return 0, m.Error
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return int64(len(m.Data)), nil
}
