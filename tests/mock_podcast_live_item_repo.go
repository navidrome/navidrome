package tests

import (
	"context"
	"errors"
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
)

// MockPodcastLiveItemRepo is a manual in-memory mock for model.PodcastLiveItemRepository.
type MockPodcastLiveItemRepo struct {
	model.PodcastLiveItemRepository
	Data map[string]*model.PodcastLiveItem // keyed by channelID
	Err  bool
}

// CreateMockPodcastLiveItemRepo returns an initialized MockPodcastLiveItemRepo.
func CreateMockPodcastLiveItemRepo() *MockPodcastLiveItemRepo {
	return &MockPodcastLiveItemRepo{Data: map[string]*model.PodcastLiveItem{}}
}

func (m *MockPodcastLiveItemRepo) GetByChannel(ctx context.Context, channelID string) (*model.PodcastLiveItem, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	if item, ok := m.Data[channelID]; ok {
		return item, nil
	}
	return nil, model.ErrNotFound
}

func (m *MockPodcastLiveItemRepo) Upsert(ctx context.Context, item *model.PodcastLiveItem) error {
	if m.Err {
		return errors.New("error")
	}
	if _, ok := m.Data[item.ChannelID]; !ok {
		item.ID = id.NewRandom()
		item.CreatedAt = time.Now()
	}
	item.UpdatedAt = time.Now()
	cp := *item
	m.Data[item.ChannelID] = &cp
	return nil
}

func (m *MockPodcastLiveItemRepo) DeleteByChannel(ctx context.Context, channelID string) error {
	if m.Err {
		return errors.New("error")
	}
	delete(m.Data, channelID)
	return nil
}
