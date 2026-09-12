package persistence

import (
	"context"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/pocketbase/dbx"
)

type podcastTranscriptRepository struct {
	sqlRepository
}

func NewPodcastTranscriptRepository(ctx context.Context, db dbx.Builder) model.PodcastTranscriptRepository {
	r := &podcastTranscriptRepository{}
	r.ctx = ctx
	r.db = db
	r.registerModel(&model.PodcastTranscript{}, nil)
	return r
}

func (r *podcastTranscriptRepository) GetByEpisode(episodeID string) (model.PodcastTranscripts, error) {
	sel := r.newSelect().Columns("*").Where(Eq{"episode_id": episodeID})
	var result model.PodcastTranscripts
	err := r.queryAll(sel, &result)
	return result, err
}

func (r *podcastTranscriptRepository) GetByEpisodes(episodeIDs []string) (model.PodcastTranscripts, error) {
	if len(episodeIDs) == 0 {
		return nil, nil
	}
	sel := r.newSelect().Columns("*").Where(Eq{"episode_id": episodeIDs})
	var result model.PodcastTranscripts
	err := r.queryAll(sel, &result)
	return result, err
}

func (r *podcastTranscriptRepository) Save(transcripts []model.PodcastTranscript) error {
	for i := range transcripts {
		if transcripts[i].ID == "" {
			transcripts[i].ID = id.NewRandom()
		}
		transcripts[i].CreatedAt = time.Now()
		if _, err := r.put(transcripts[i].ID, &transcripts[i]); err != nil {
			return err
		}
	}
	return nil
}

func (r *podcastTranscriptRepository) DeleteByEpisode(episodeID string) error {
	return r.delete(Eq{"episode_id": episodeID})
}

var _ model.PodcastTranscriptRepository = (*podcastTranscriptRepository)(nil)
