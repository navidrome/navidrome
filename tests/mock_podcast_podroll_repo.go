package tests

import (
	"errors"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
)

// MockPodcastPodrollRepo is a manual in-memory mock for model.PodcastPodrollRepository.
type MockPodcastPodrollRepo struct {
	model.PodcastPodrollRepository
	Data map[string]*model.PodcastPodrollItem // keyed by item ID
	Err  bool
}

// CreateMockPodcastPodrollRepo returns an initialized MockPodcastPodrollRepo.
func CreateMockPodcastPodrollRepo() *MockPodcastPodrollRepo {
	return &MockPodcastPodrollRepo{Data: map[string]*model.PodcastPodrollItem{}}
}

func (m *MockPodcastPodrollRepo) GetByChannel(channelID string) (model.PodcastPodrollItems, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	var result model.PodcastPodrollItems
	for _, item := range m.Data {
		if item.ChannelID == channelID {
			result = append(result, *item)
		}
	}
	return result, nil
}

func (m *MockPodcastPodrollRepo) GetByChannels(channelIDs []string) (model.PodcastPodrollItems, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	ids := make(map[string]bool)
	for _, cid := range channelIDs {
		ids[cid] = true
	}
	var result model.PodcastPodrollItems
	for _, item := range m.Data {
		if ids[item.ChannelID] {
			result = append(result, *item)
		}
	}
	return result, nil
}

func (m *MockPodcastPodrollRepo) SaveForChannel(channelID string, items []model.PodcastPodrollItem) error {
	if m.Err {
		return errors.New("error")
	}
	// Remove old items for this channel.
	for k, v := range m.Data {
		if v.ChannelID == channelID {
			delete(m.Data, k)
		}
	}
	for i := range items {
		items[i].ID = id.NewRandom()
		items[i].ChannelID = channelID
		items[i].SortOrder = i
		cp := items[i]
		m.Data[cp.ID] = &cp
	}
	return nil
}
