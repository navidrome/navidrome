package engine

import (
	"errors"
	"time"
)

func CreateMockNowPlayingRepo() *MockNowPlaying {
	return &MockNowPlaying{data: make(map[string]time.Time)}
}

type MockNowPlaying struct {
	NowPlayingRepository
	data map[string]time.Time
	err  bool
}

func (m *MockNowPlaying) SetError(err bool) {
	m.err = err
}

func (m *MockNowPlaying) Add(id string) error {
	if m.err {
		return errors.New("Error!")
	}
	m.data[id] = time.Now()
	return nil
}
