package tests

import (
	"context"
	"errors"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
)

type MockPodcastChannelRepo struct {
	model.PodcastChannelRepository
	Data map[string]*model.PodcastChannel
	Err  bool
}

func CreateMockPodcastChannelRepo() *MockPodcastChannelRepo {
	return &MockPodcastChannelRepo{Data: map[string]*model.PodcastChannel{}}
}

func (m *MockPodcastChannelRepo) Get(ctx context.Context, chanID string) (*model.PodcastChannel, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	if c, ok := m.Data[chanID]; ok {
		return c, nil
	}
	return nil, model.ErrNotFound
}

func (m *MockPodcastChannelRepo) GetAll(ctx context.Context, withEpisodes bool) (model.PodcastChannels, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	result := make(model.PodcastChannels, 0, len(m.Data))
	for _, c := range m.Data {
		result = append(result, *c)
	}
	return result, nil
}

func (m *MockPodcastChannelRepo) Create(ctx context.Context, channel *model.PodcastChannel) error {
	if m.Err {
		return errors.New("error")
	}
	if channel.ID == "" {
		channel.ID = id.NewRandom()
	}
	m.Data[channel.ID] = channel
	return nil
}

func (m *MockPodcastChannelRepo) UpdateChannel(ctx context.Context, channel *model.PodcastChannel) error {
	if m.Err {
		return errors.New("error")
	}
	m.Data[channel.ID] = channel
	return nil
}

func (m *MockPodcastChannelRepo) ExistsByURL(ctx context.Context, url string) (bool, error) {
	if m.Err {
		return false, errors.New("error")
	}
	for _, c := range m.Data {
		if c.URL == url {
			return true, nil
		}
	}
	return false, nil
}

func (m *MockPodcastChannelRepo) Delete(ctx context.Context, ids ...string) error {
	if m.Err {
		return errors.New("error")
	}
	for _, chanID := range ids {
		if _, ok := m.Data[chanID]; !ok {
			return model.ErrNotFound
		}
		delete(m.Data, chanID)
	}
	return nil
}
