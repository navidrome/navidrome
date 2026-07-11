package tests

import (
	"errors"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
)

type MockedPodcastChannelRepo struct {
	model.PodcastChannelRepository
	Data map[string]*model.PodcastChannel
	All  model.PodcastChannels
	Err  bool
}

func CreateMockedPodcastChannelRepo() *MockedPodcastChannelRepo {
	return &MockedPodcastChannelRepo{Data: map[string]*model.PodcastChannel{}}
}

func (m *MockedPodcastChannelRepo) SetError(err bool) {
	m.Err = err
}

func (m *MockedPodcastChannelRepo) CountAll(options ...model.QueryOptions) (int64, error) {
	if m.Err {
		return 0, errors.New("error")
	}
	return int64(len(m.Data)), nil
}

func (m *MockedPodcastChannelRepo) Delete(id string) error {
	if m.Err {
		return errors.New("error")
	}
	if _, found := m.Data[id]; !found {
		return model.ErrNotFound
	}
	delete(m.Data, id)
	return nil
}

func (m *MockedPodcastChannelRepo) Get(id string) (*model.PodcastChannel, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	if d, ok := m.Data[id]; ok {
		return d, nil
	}
	return nil, model.ErrNotFound
}

func (m *MockedPodcastChannelRepo) GetAll(qo ...model.QueryOptions) (model.PodcastChannels, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	return m.All, nil
}

func (m *MockedPodcastChannelRepo) GetWithEpisodes(id string) (*model.PodcastChannel, error) {
	return m.Get(id)
}

func (m *MockedPodcastChannelRepo) FindByUrl(url string) (*model.PodcastChannel, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	for _, c := range m.Data {
		if c.Url == url {
			return c, nil
		}
	}
	return nil, model.ErrNotFound
}

func (m *MockedPodcastChannelRepo) Put(channel *model.PodcastChannel, _ ...string) error {
	if m.Err {
		return errors.New("error")
	}
	if channel.ID == "" {
		channel.ID = id.NewRandom()
	}
	m.Data[channel.ID] = channel
	return nil
}
