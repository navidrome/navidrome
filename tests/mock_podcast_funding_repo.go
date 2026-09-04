package tests

import (
	"errors"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
)

type MockPodcastFundingRepo struct {
	model.PodcastFundingRepository
	Data map[string]*model.PodcastFundingItem
	Err  bool
}

func CreateMockPodcastFundingRepo() *MockPodcastFundingRepo {
	return &MockPodcastFundingRepo{Data: map[string]*model.PodcastFundingItem{}}
}

func (m *MockPodcastFundingRepo) GetByChannel(channelID string) (model.PodcastFundingItems, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	var result model.PodcastFundingItems
	for _, f := range m.Data {
		if f.ChannelID == channelID {
			result = append(result, *f)
		}
	}
	return result, nil
}

func (m *MockPodcastFundingRepo) GetByChannels(channelIDs []string) (model.PodcastFundingItems, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	ids := make(map[string]bool, len(channelIDs))
	for _, id := range channelIDs {
		ids[id] = true
	}
	var result model.PodcastFundingItems
	for _, f := range m.Data {
		if ids[f.ChannelID] {
			result = append(result, *f)
		}
	}
	return result, nil
}

func (m *MockPodcastFundingRepo) SaveForChannel(channelID string, items []model.PodcastFundingItem) error {
	if m.Err {
		return errors.New("error")
	}
	for k, f := range m.Data {
		if f.ChannelID == channelID {
			delete(m.Data, k)
		}
	}
	for i := range items {
		items[i].ID = id.NewRandom()
		items[i].ChannelID = channelID
		f := items[i]
		m.Data[f.ID] = &f
	}
	return nil
}
