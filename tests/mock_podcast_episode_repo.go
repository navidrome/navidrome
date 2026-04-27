package tests

import (
	"errors"
	"sort"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
)

type MockPodcastEpisodeRepo struct {
	model.PodcastEpisodeRepository
	Data map[string]*model.PodcastEpisode
	Err  bool
}

func CreateMockPodcastEpisodeRepo() *MockPodcastEpisodeRepo {
	return &MockPodcastEpisodeRepo{Data: map[string]*model.PodcastEpisode{}}
}

func (m *MockPodcastEpisodeRepo) Get(epID string) (*model.PodcastEpisode, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	if ep, ok := m.Data[epID]; ok {
		return ep, nil
	}
	return nil, model.ErrNotFound
}

func (m *MockPodcastEpisodeRepo) GetNewest(count int) (model.PodcastEpisodes, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	all := make(model.PodcastEpisodes, 0, len(m.Data))
	for _, ep := range m.Data {
		all = append(all, *ep)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].PublishDate.After(all[j].PublishDate)
	})
	if count > 0 && len(all) > count {
		all = all[:count]
	}
	return all, nil
}

func (m *MockPodcastEpisodeRepo) GetByChannel(channelID string) (model.PodcastEpisodes, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	result := model.PodcastEpisodes{}
	for _, ep := range m.Data {
		if ep.ChannelID == channelID {
			result = append(result, *ep)
		}
	}
	return result, nil
}

func (m *MockPodcastEpisodeRepo) GetByGUID(channelID, guid string) (*model.PodcastEpisode, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	for _, ep := range m.Data {
		if ep.ChannelID == channelID && ep.GUID == guid {
			return ep, nil
		}
	}
	return nil, model.ErrNotFound
}

func (m *MockPodcastEpisodeRepo) Create(ep *model.PodcastEpisode) error {
	if m.Err {
		return errors.New("error")
	}
	if ep.ID == "" {
		ep.ID = id.NewRandom()
	}
	m.Data[ep.ID] = ep
	return nil
}

func (m *MockPodcastEpisodeRepo) Update(ep *model.PodcastEpisode) error {
	if m.Err {
		return errors.New("error")
	}
	m.Data[ep.ID] = ep
	return nil
}

func (m *MockPodcastEpisodeRepo) Delete(epID string) error {
	if m.Err {
		return errors.New("error")
	}
	delete(m.Data, epID)
	return nil
}
