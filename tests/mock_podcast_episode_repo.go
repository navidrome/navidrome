package tests

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/gg"
)

type MockedPodcastEpisodeRepo struct {
	Cleaned bool
	model.PodcastEpisodeRepository
	err  bool
	all  model.PodcastEpisodes
	data map[string]*model.PodcastEpisode
}

func CreateMockedPodcastEpisodeRepo() *MockedPodcastEpisodeRepo {
	return &MockedPodcastEpisodeRepo{
		all:  make(model.PodcastEpisodes, 0),
		data: make(map[string]*model.PodcastEpisode),
	}
}

func (m *MockedPodcastEpisodeRepo) Cleanup() error {
	if m.err {
		return errors.New("Error")
	}
	m.Cleaned = true
	return nil
}

func (m *MockedPodcastEpisodeRepo) DeleteInternal(id string) error {
	if m.err {
		return errors.New("Error")
	}

	_, ok := m.data[id]
	if !ok {
		return model.ErrNotFound
	}

	delete(m.data, id)

	newArray := make(model.PodcastEpisodes, len(m.all)-1)
	for _, item := range m.all {
		if item.ID != id {
			newArray = append(newArray, item)
		}
	}
	m.all = newArray
	return nil
}

func (m *MockedPodcastEpisodeRepo) Get(id string) (*model.PodcastEpisode, error) {
	if m.err {
		return nil, errors.New("Error")
	}

	item, ok := m.data[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return item, nil
}

func (m *MockedPodcastEpisodeRepo) GetAll(qo ...model.QueryOptions) (model.PodcastEpisodes, error) {
	if m.err {
		return nil, errors.New("Error!")
	}
	return m.all, nil
}

func (m *MockedPodcastEpisodeRepo) GetEpisodeGuids(id string) (map[string]bool, error) {
	if m.err {
		return nil, errors.New("Error!")
	}

	mapping := map[string]bool{}
	for _, item := range m.data {
		mapping[item.Guid] = true
	}

	return mapping, nil
}

func (m *MockedPodcastEpisodeRepo) GetNewestEpisodes(count int) (model.PodcastEpisodes, error) {
	if m.err {
		return nil, errors.New("Error")
	}

	if len(m.all) > count {
		return m.all[:count], nil
	}

	return m.all, nil
}

func (m *MockedPodcastEpisodeRepo) IncPlayCount(id string, timestamp time.Time) error {
	if m.err {
		return errors.New("error")
	}
	if d, ok := m.data[id]; ok {
		d.PlayCount++
		d.PlayDate = &timestamp
		return nil
	}
	return model.ErrNotFound
}

func (m *MockedPodcastEpisodeRepo) Put(pe *model.PodcastEpisode) error {
	if m.err {
		return errors.New("error")
	}
	if pe.ID == "" {
		pe.ID = uuid.NewString()
	}
	m.data[pe.ID] = pe
	m.all = append(m.all, *pe)
	return nil
}

func (m *MockedPodcastEpisodeRepo) SetData(episodes model.PodcastEpisodes) {
	m.all = episodes
	m.data = make(map[string]*model.PodcastEpisode)
	for i, e := range m.all {
		m.data[e.ID] = &episodes[i]
	}
}

func (m *MockedPodcastEpisodeRepo) SetStar(starred bool, itemIDs ...string) error {
	if m.err {
		return errors.New("error")
	}

	for _, item := range itemIDs {
		if d, ok := m.data[item]; ok {
			d.Starred = starred
			d.StarredAt = gg.P(time.Now())
		} else {
			return model.ErrNotFound
		}
	}

	return nil
}

func (m *MockedPodcastEpisodeRepo) SetRating(rating int, itemID string) error {
	if m.err {
		return errors.New("error")
	}

	if d, ok := m.data[itemID]; ok {
		d.Rating = rating
		return nil
	}
	return model.ErrNotFound
}

func (m *MockedPodcastEpisodeRepo) SetError(err bool) {
	m.err = err
}
