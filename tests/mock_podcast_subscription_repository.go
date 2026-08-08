package tests

import (
	"errors"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
)

type MockedPodcastSubscriptionRepo struct {
	Data map[string]*model.PodcastSubscription
	Err  bool
}

func CreateMockedPodcastSubscriptionRepo() *MockedPodcastSubscriptionRepo {
	return &MockedPodcastSubscriptionRepo{Data: map[string]*model.PodcastSubscription{}}
}

func (m *MockedPodcastSubscriptionRepo) SetError(err bool) {
	m.Err = err
}

func (m *MockedPodcastSubscriptionRepo) CountAll(options ...model.QueryOptions) (int64, error) {
	if m.Err {
		return 0, errors.New("error")
	}
	return int64(len(m.Data)), nil
}

func (m *MockedPodcastSubscriptionRepo) Get(subID string) (*model.PodcastSubscription, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	if d, ok := m.Data[subID]; ok {
		return d, nil
	}
	return nil, model.ErrNotFound
}

func (m *MockedPodcastSubscriptionRepo) GetAll(qo ...model.QueryOptions) (model.PodcastSubscriptions, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	var res model.PodcastSubscriptions
	for _, s := range m.Data {
		res = append(res, *s)
	}
	return res, nil
}

func (m *MockedPodcastSubscriptionRepo) FindByChannelAndUser(channelID, userID string) (*model.PodcastSubscription, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	for _, s := range m.Data {
		if s.ChannelID == channelID && s.UserID == userID {
			return s, nil
		}
	}
	return nil, model.ErrNotFound
}

func (m *MockedPodcastSubscriptionRepo) FindByChannel(channelID string) (model.PodcastSubscriptions, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	var res model.PodcastSubscriptions
	for _, s := range m.Data {
		if s.ChannelID == channelID {
			res = append(res, *s)
		}
	}
	return res, nil
}

func (m *MockedPodcastSubscriptionRepo) Put(s *model.PodcastSubscription) error {
	if m.Err {
		return errors.New("error")
	}
	if s.ID == "" {
		s.ID = id.NewRandom()
	}
	m.Data[s.ID] = s
	return nil
}

func (m *MockedPodcastSubscriptionRepo) Delete(subID string) (int64, error) {
	if m.Err {
		return 0, errors.New("error")
	}
	sub, ok := m.Data[subID]
	if !ok {
		return 0, model.ErrNotFound
	}
	delete(m.Data, subID)
	var remaining int64
	for _, s := range m.Data {
		if s.ChannelID == sub.ChannelID {
			remaining++
		}
	}
	return remaining, nil
}

var _ model.PodcastSubscriptionRepository = (*MockedPodcastSubscriptionRepo)(nil)
