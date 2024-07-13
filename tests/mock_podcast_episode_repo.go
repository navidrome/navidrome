package tests

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/gg"
)

type MockedPodcastEpisodeRepo struct {
	model.PodcastEpisodeRepository
	err  bool
	data map[string]*model.PodcastEpisode
}

func CreateMockedPodcastEpisodeRepo() *MockedPodcastEpisodeRepo {
	return &MockedPodcastEpisodeRepo{
		data: make(map[string]*model.PodcastEpisode),
	}
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
	return nil
}

func (m *MockedPodcastEpisodeRepo) SetData(episodes model.PodcastEpisodes) {
	m.data = make(map[string]*model.PodcastEpisode)
	for i, e := range episodes {
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
