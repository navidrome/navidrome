package tests

import (
	"time"

	"github.com/navidrome/navidrome/model"
)

type MockedScrobbleRadioRepo struct {
	Error error
	data  model.ScrobbleRadioEntries
}

func CreateMockedScrobbleRadioRepo() *MockedScrobbleRadioRepo {
	return &MockedScrobbleRadioRepo{}
}

func (m *MockedScrobbleRadioRepo) UserIDs(service string) ([]string, error) {
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

func (m *MockedScrobbleRadioRepo) Enqueue(service, userId, artist, title string, playTime time.Time) error {
	if m.Error != nil {
		return m.Error
	}
	m.data = append(m.data, model.ScrobleRadioEntry{
		Artist:      artist,
		EnqueueTime: time.Now(),
		PlayTime:    playTime,
		Title:       title,
		Service:     service,
		UserID:      userId,
	})
	return nil
}

func (m *MockedScrobbleRadioRepo) Next(service, userId string) (*model.ScrobleRadioEntry, error) {
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

func (m *MockedScrobbleRadioRepo) Dequeue(entry *model.ScrobleRadioEntry) error {
	if m.Error != nil {
		return m.Error
	}
	newData := model.ScrobbleRadioEntries{}
	for _, e := range m.data {
		if e.Service == entry.Service &&
			e.UserID == entry.UserID &&
			e.PlayTime == entry.PlayTime &&
			e.Artist == entry.Artist &&
			e.Title == entry.Title {
			continue
		}
		newData = append(newData, e)
	}
	m.data = newData
	return nil
}

func (m *MockedScrobbleRadioRepo) Length() (int64, error) {
	if m.Error != nil {
		return 0, m.Error
	}
	return int64(len(m.data)), nil
}
