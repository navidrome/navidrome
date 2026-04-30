package tests

import (
	"errors"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
)

type MockPodcastPersonRepo struct {
	model.PodcastPersonRepository
	Data map[string]*model.PodcastPerson
	Err  bool
}

func CreateMockPodcastPersonRepo() *MockPodcastPersonRepo {
	return &MockPodcastPersonRepo{Data: map[string]*model.PodcastPerson{}}
}

func (m *MockPodcastPersonRepo) GetByChannel(channelID string) (model.PodcastPersons, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	var result model.PodcastPersons
	for _, p := range m.Data {
		if p.ChannelID == channelID {
			result = append(result, *p)
		}
	}
	return result, nil
}

func (m *MockPodcastPersonRepo) GetByEpisode(episodeID string) (model.PodcastPersons, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	var result model.PodcastPersons
	for _, p := range m.Data {
		if p.EpisodeID == episodeID {
			result = append(result, *p)
		}
	}
	return result, nil
}

func (m *MockPodcastPersonRepo) GetByEpisodes(episodeIDs []string) (model.PodcastPersons, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	ids := make(map[string]bool, len(episodeIDs))
	for _, id := range episodeIDs {
		ids[id] = true
	}
	var result model.PodcastPersons
	for _, p := range m.Data {
		if ids[p.EpisodeID] {
			result = append(result, *p)
		}
	}
	return result, nil
}

func (m *MockPodcastPersonRepo) SaveForChannel(channelID string, persons []model.PodcastPerson) error {
	if m.Err {
		return errors.New("error")
	}
	for k, p := range m.Data {
		if p.ChannelID == channelID {
			delete(m.Data, k)
		}
	}
	for i := range persons {
		persons[i].ID = id.NewRandom()
		persons[i].ChannelID = channelID
		p := persons[i]
		m.Data[p.ID] = &p
	}
	return nil
}

func (m *MockPodcastPersonRepo) SaveForEpisode(episodeID string, persons []model.PodcastPerson) error {
	if m.Err {
		return errors.New("error")
	}
	for k, p := range m.Data {
		if p.EpisodeID == episodeID {
			delete(m.Data, k)
		}
	}
	for i := range persons {
		persons[i].ID = id.NewRandom()
		persons[i].EpisodeID = episodeID
		p := persons[i]
		m.Data[p.ID] = &p
	}
	return nil
}
