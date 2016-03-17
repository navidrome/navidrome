package engine

import (
	"errors"
	"time"
)

func CreateMockNowPlayingRepo() *MockNowPlaying {
	return &MockNowPlaying{}
}

type MockNowPlaying struct {
	NowPlayingRepository
	id    string
	start time.Time
	err   bool
}

func (m *MockNowPlaying) SetError(err bool) {
	m.err = err
}

func (m *MockNowPlaying) Set(id string) error {
	if m.err {
		return errors.New("Error!")
	}
	m.id = id
	m.start = time.Now()
	return nil
}

func (m *MockNowPlaying) Current() (string, time.Time) {
	return m.id, m.start
}
