package tests

import (
	"errors"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
)

type MockPodcastTranscriptRepo struct {
	model.PodcastTranscriptRepository
	Data map[string]*model.PodcastTranscript
	Err  bool
}

func CreateMockPodcastTranscriptRepo() *MockPodcastTranscriptRepo {
	return &MockPodcastTranscriptRepo{Data: map[string]*model.PodcastTranscript{}}
}

func (m *MockPodcastTranscriptRepo) GetByEpisode(episodeID string) (model.PodcastTranscripts, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	var result model.PodcastTranscripts
	for _, t := range m.Data {
		if t.EpisodeID == episodeID {
			result = append(result, *t)
		}
	}
	return result, nil
}

func (m *MockPodcastTranscriptRepo) GetByEpisodes(episodeIDs []string) (model.PodcastTranscripts, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	ids := make(map[string]bool, len(episodeIDs))
	for _, id := range episodeIDs {
		ids[id] = true
	}
	var result model.PodcastTranscripts
	for _, t := range m.Data {
		if ids[t.EpisodeID] {
			result = append(result, *t)
		}
	}
	return result, nil
}

func (m *MockPodcastTranscriptRepo) Save(transcripts []model.PodcastTranscript) error {
	if m.Err {
		return errors.New("error")
	}
	for i := range transcripts {
		if transcripts[i].ID == "" {
			transcripts[i].ID = id.NewRandom()
		}
		t := transcripts[i]
		m.Data[t.ID] = &t
	}
	return nil
}

func (m *MockPodcastTranscriptRepo) DeleteByEpisode(episodeID string) error {
	if m.Err {
		return errors.New("error")
	}
	for k, t := range m.Data {
		if t.EpisodeID == episodeID {
			delete(m.Data, k)
		}
	}
	return nil
}
