package tests

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/gg"
	"github.com/navidrome/navidrome/utils/slice"
	"golang.org/x/exp/maps"
)

type MockedPodcastRepo struct {
	Cleaned bool
	model.PodcastRepository
	err  bool
	data map[string]*model.Podcast
}

func CreateMockPodcastRepo() *MockedPodcastRepo {
	return &MockedPodcastRepo{
		data: make(map[string]*model.Podcast),
	}
}

func (m *MockedPodcastRepo) Cleanup() error {
	if m.err {
		return errors.New("Error")
	}
	m.Cleaned = true
	return nil
}

func (m *MockedPodcastRepo) DeleteInternal(id string) error {
	if m.err {
		return errors.New("Error")
	}

	_, ok := m.data[id]
	if !ok {
		return model.ErrNotFound
	}

	delete(m.data, id)
	return nil
}

func (m *MockedPodcastRepo) IncPlayCount(id string, timestamp time.Time) error {
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

func (m *MockedPodcastRepo) Get(id string, withEpisodes bool) (*model.Podcast, error) {
	if m.err {
		return nil, errors.New("Error")
	}

	item, ok := m.data[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return item, nil
}

func (m *MockedPodcastRepo) GetAll(_ bool, qo ...model.QueryOptions) (model.Podcasts, error) {
	if m.err {
		return nil, errors.New("Error!")
	}
	values := maps.Values(m.data)
	return slice.Map(values, func(p *model.Podcast) model.Podcast {
		return *p
	}), nil
}

func (m *MockedPodcastRepo) Put(pd *model.Podcast) error {
	if m.err {
		return errors.New("error")
	}
	if pd.ID == "" {
		pd.ID = uuid.NewString()
	}
	m.data[pd.ID] = pd
	return nil
}

func (m *MockedPodcastRepo) PutInternal(pd *model.Podcast) error {
	return m.Put(pd)
}

func (m *MockedPodcastRepo) SetData(podcasts model.Podcasts) {
	m.data = make(map[string]*model.Podcast)
	for i, e := range podcasts {
		m.data[e.ID] = &podcasts[i]
	}
}

func (m *MockedPodcastRepo) SetError(err bool) {
	m.err = err
}

func (m *MockedPodcastRepo) SetStar(starred bool, itemIDs ...string) error {
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

func (m *MockedPodcastRepo) SetRating(rating int, itemID string) error {
	if m.err {
		return errors.New("error")
	}

	if d, ok := m.data[itemID]; ok {
		d.Rating = rating
		return nil
	}
	return model.ErrNotFound
}
