package engine

import (
	"errors"
	"time"

	"github.com/cloudsonic/sonic-server/domain"
)

func CreateMockNowPlayingRepo() *MockNowPlaying {
	return &MockNowPlaying{}
}

type MockNowPlaying struct {
	domain.NowPlayingRepository
	data []domain.NowPlayingInfo
	t    time.Time
	err  bool
}

func (m *MockNowPlaying) SetError(err bool) {
	m.err = err
}

func (m *MockNowPlaying) Enqueue(info *domain.NowPlayingInfo) error {
	if m.err {
		return errors.New("Error!")
	}

	m.data = append(m.data, domain.NowPlayingInfo{})
	copy(m.data[1:], m.data[0:])
	m.data[0] = *info

	if !m.t.IsZero() {
		m.data[0].Start = m.t
		m.t = time.Time{}
	}

	return nil
}

func (m *MockNowPlaying) Dequeue(playerId int) (*domain.NowPlayingInfo, error) {
	if len(m.data) == 0 {
		return nil, nil
	}
	l := len(m.data)
	info := m.data[l-1]
	m.data = m.data[:l-1]

	return &info, nil
}

func (m *MockNowPlaying) Count(playerId int) (int64, error) {
	return int64(len(m.data)), nil
}

func (m *MockNowPlaying) GetAll() ([]*domain.NowPlayingInfo, error) {
	np, err := m.Head(1)
	if np == nil || err != nil {
		return nil, err
	}
	return []*domain.NowPlayingInfo{np}, err
}

func (m *MockNowPlaying) Head(playerId int) (*domain.NowPlayingInfo, error) {
	if len(m.data) == 0 {
		return nil, nil
	}
	info := m.data[0]
	return &info, nil
}

func (m *MockNowPlaying) Tail(playerId int) (*domain.NowPlayingInfo, error) {
	if len(m.data) == 0 {
		return nil, nil
	}
	info := m.data[len(m.data)-1]
	return &info, nil
}

func (m *MockNowPlaying) ClearAll() {
	m.data = make([]domain.NowPlayingInfo, 0)
	m.err = false
}

func (m *MockNowPlaying) OverrideNow(t time.Time) {
	m.t = t
}
