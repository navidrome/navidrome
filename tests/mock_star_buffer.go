package tests

import (
	"time"

	"github.com/navidrome/navidrome/model"
)

type MockedStarBufferRepo struct {
	Error error
	data  model.StarEntries
}

func CreateMockedStarBufferRepo() *MockedStarBufferRepo {
	return &MockedStarBufferRepo{}
}

func (m *MockedStarBufferRepo) UserIDs(service string) ([]string, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	userIds := make(map[string]struct{})
	for _, e := range m.data {
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

func (m *MockedStarBufferRepo) TryUpdate(service, userId, mediaFileId string, isStar bool) (bool, error) {
	if m.Error != nil {
		return false, m.Error
	}

	for _, e := range m.data {
		if e.Service == service && e.UserID == userId && e.MediaFile.ID == mediaFileId {
			e.IsStar = isStar
			return true, nil
		}
	}
	return false, nil
}

func (m *MockedStarBufferRepo) Enqueue(service, userId, mediaFileId string, isStar bool) error {
	if m.Error != nil {
		return m.Error
	}
	m.data = append(m.data, model.StarEntry{
		MediaFile:   model.MediaFile{ID: mediaFileId},
		Service:     service,
		UserID:      userId,
		IsStar:      isStar,
		EnqueueTime: time.Now(),
	})
	return nil
}

func (m *MockedStarBufferRepo) Next(service, userId string) (*model.StarEntry, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	for _, e := range m.data {
		if e.Service == service && e.UserID == userId {
			return &e, nil
		}
	}
	return nil, nil
}

func (m *MockedStarBufferRepo) Dequeue(entry *model.StarEntry) error {
	if m.Error != nil {
		return m.Error
	}
	newData := model.StarEntries{}
	for _, e := range m.data {
		if e.Service == entry.Service && e.UserID == entry.UserID && e.MediaFile.ID == entry.MediaFile.ID {
			continue
		}
		newData = append(newData, e)
	}
	m.data = newData
	return nil
}

func (m *MockedStarBufferRepo) Length() (int64, error) {
	if m.Error != nil {
		return 0, m.Error
	}
	return int64(len(m.data)), nil
}
