package tests

import (
	"context"
	"errors"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
)

type MockPodcastImageRepo struct {
	model.PodcastImageRepository
	Data map[string]*model.PodcastImage
	Err  bool
}

func CreateMockPodcastImageRepo() *MockPodcastImageRepo {
	return &MockPodcastImageRepo{Data: map[string]*model.PodcastImage{}}
}

func (m *MockPodcastImageRepo) GetByChannel(ctx context.Context, channelID string) (model.PodcastImages, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	var result model.PodcastImages
	for _, img := range m.Data {
		if img.ChannelID == channelID {
			result = append(result, *img)
		}
	}
	return result, nil
}

func (m *MockPodcastImageRepo) GetByChannels(ctx context.Context, channelIDs []string) (model.PodcastImages, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	ids := make(map[string]bool, len(channelIDs))
	for _, id := range channelIDs {
		ids[id] = true
	}
	var result model.PodcastImages
	for _, img := range m.Data {
		if ids[img.ChannelID] {
			result = append(result, *img)
		}
	}
	return result, nil
}

func (m *MockPodcastImageRepo) GetByEpisode(ctx context.Context, episodeID string) (model.PodcastImages, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	var result model.PodcastImages
	for _, img := range m.Data {
		if img.EpisodeID == episodeID {
			result = append(result, *img)
		}
	}
	return result, nil
}

func (m *MockPodcastImageRepo) GetByEpisodes(ctx context.Context, episodeIDs []string) (model.PodcastImages, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	ids := make(map[string]bool, len(episodeIDs))
	for _, id := range episodeIDs {
		ids[id] = true
	}
	var result model.PodcastImages
	for _, img := range m.Data {
		if ids[img.EpisodeID] {
			result = append(result, *img)
		}
	}
	return result, nil
}

func (m *MockPodcastImageRepo) SaveForChannel(ctx context.Context, channelID string, images []model.PodcastImage) error {
	if m.Err {
		return errors.New("error")
	}
	for k, img := range m.Data {
		if img.ChannelID == channelID {
			delete(m.Data, k)
		}
	}
	for i := range images {
		images[i].ID = id.NewRandom()
		images[i].ChannelID = channelID
		img := images[i]
		m.Data[img.ID] = &img
	}
	return nil
}

func (m *MockPodcastImageRepo) SaveForEpisode(ctx context.Context, episodeID string, images []model.PodcastImage) error {
	if m.Err {
		return errors.New("error")
	}
	for k, img := range m.Data {
		if img.EpisodeID == episodeID {
			delete(m.Data, k)
		}
	}
	for i := range images {
		images[i].ID = id.NewRandom()
		images[i].EpisodeID = episodeID
		img := images[i]
		m.Data[img.ID] = &img
	}
	return nil
}
