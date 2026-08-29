package tests

import (
	"errors"
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
)

type MockedPodcastEpisodeRepo struct {
	model.PodcastEpisodeRepository
	Data map[string]*model.PodcastEpisode
	All  model.PodcastEpisodes
	Err  bool
}

func CreateMockedPodcastEpisodeRepo() *MockedPodcastEpisodeRepo {
	return &MockedPodcastEpisodeRepo{Data: map[string]*model.PodcastEpisode{}}
}

func (m *MockedPodcastEpisodeRepo) SetError(err bool) {
	m.Err = err
}

func (m *MockedPodcastEpisodeRepo) CountAll(options ...model.QueryOptions) (int64, error) {
	if m.Err {
		return 0, errors.New("error")
	}
	return int64(len(m.Data)), nil
}

func (m *MockedPodcastEpisodeRepo) Delete(id string) error {
	if m.Err {
		return errors.New("error")
	}
	if _, found := m.Data[id]; !found {
		return model.ErrNotFound
	}
	delete(m.Data, id)
	return nil
}

func (m *MockedPodcastEpisodeRepo) Get(id string) (*model.PodcastEpisode, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	if d, ok := m.Data[id]; ok {
		return d, nil
	}
	return nil, model.ErrNotFound
}

func (m *MockedPodcastEpisodeRepo) GetAll(qo ...model.QueryOptions) (model.PodcastEpisodes, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	return m.All, nil
}

func (m *MockedPodcastEpisodeRepo) FindByGuid(channelID, guid string) (*model.PodcastEpisode, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	for _, e := range m.Data {
		if e.ChannelID == channelID && e.Guid == guid {
			return e, nil
		}
	}
	return nil, model.ErrNotFound
}

func (m *MockedPodcastEpisodeRepo) GetNewest(count int) (model.PodcastEpisodes, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	if count < len(m.All) {
		return m.All[:count], nil
	}
	return m.All, nil
}

func (m *MockedPodcastEpisodeRepo) Put(episode *model.PodcastEpisode, _ ...string) error {
	if m.Err {
		return errors.New("error")
	}
	if episode.ID == "" {
		episode.ID = id.NewRandom()
	}
	m.Data[episode.ID] = episode
	return nil
}

func (m *MockedPodcastEpisodeRepo) IncPlayCount(id string, _ time.Time) error {
	if m.Err {
		return errors.New("error")
	}
	if _, ok := m.Data[id]; !ok {
		return model.ErrNotFound
	}
	return nil
}

func (m *MockedPodcastEpisodeRepo) SetDownloaded(_ string, downloaded bool, ids ...string) error {
	if m.Err {
		return errors.New("error")
	}
	for _, epID := range ids {
		if e, ok := m.Data[epID]; ok {
			e.Downloaded = downloaded
		}
	}
	return nil
}

func (m *MockedPodcastEpisodeRepo) AnyUserWantsDownload(epID string) (bool, error) {
	if m.Err {
		return false, errors.New("error")
	}
	e, ok := m.Data[epID]
	return ok && e.Downloaded, nil
}

func (m *MockedPodcastEpisodeRepo) GetDownloadedForUser(_, channelID string) (model.PodcastEpisodes, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	var res model.PodcastEpisodes
	for _, e := range m.All {
		if e.ChannelID == channelID && e.Downloaded {
			res = append(res, e)
		}
	}
	return res, nil
}
