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
	info NowPlayingInfo
	err  bool
}

func (m *MockNowPlaying) SetError(err bool) {
	m.err = err
}

func (m *MockNowPlaying) Set(id, username string, playerId int, playerName string) error {
	if m.err {
		return errors.New("Error!")
	}
	m.info.TrackId = id
	m.info.Username = username
	m.info.Start = time.Now()
	m.info.PlayerId = playerId
	m.info.PlayerName = playerName
	return nil
}

func (m *MockNowPlaying) Current() NowPlayingInfo {
	return m.info
}
