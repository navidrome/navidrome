package persistence

import (
	"context"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/pocketbase/dbx"
)

type podcastPersonRepository struct {
	sqlRepository
}

func NewPodcastPersonRepository(ctx context.Context, db dbx.Builder) model.PodcastPersonRepository {
	r := &podcastPersonRepository{}
	r.ctx = ctx
	r.db = db
	r.registerModel(&model.PodcastPerson{}, nil)
	return r
}

func (r *podcastPersonRepository) GetByChannel(channelID string) (model.PodcastPersons, error) {
	sel := r.newSelect().Columns("*").Where(Eq{"channel_id": channelID})
	var result model.PodcastPersons
	err := r.queryAll(sel, &result)
	return result, err
}

func (r *podcastPersonRepository) GetByEpisode(episodeID string) (model.PodcastPersons, error) {
	sel := r.newSelect().Columns("*").Where(Eq{"episode_id": episodeID})
	var result model.PodcastPersons
	err := r.queryAll(sel, &result)
	return result, err
}

func (r *podcastPersonRepository) GetByEpisodes(episodeIDs []string) (model.PodcastPersons, error) {
	if len(episodeIDs) == 0 {
		return nil, nil
	}
	sel := r.newSelect().Columns("*").Where(Eq{"episode_id": episodeIDs})
	var result model.PodcastPersons
	err := r.queryAll(sel, &result)
	return result, err
}

func (r *podcastPersonRepository) SaveForChannel(channelID string, persons []model.PodcastPerson) error {
	if err := r.delete(Eq{"channel_id": channelID}); err != nil {
		return err
	}
	now := time.Now()
	for i := range persons {
		persons[i].ID = id.NewRandom()
		persons[i].ChannelID = channelID
		persons[i].EpisodeID = ""
		persons[i].CreatedAt = now
		if _, err := r.put(persons[i].ID, &persons[i]); err != nil {
			return err
		}
	}
	return nil
}

func (r *podcastPersonRepository) SaveForEpisode(episodeID string, persons []model.PodcastPerson) error {
	if err := r.delete(Eq{"episode_id": episodeID}); err != nil {
		return err
	}
	now := time.Now()
	for i := range persons {
		persons[i].ID = id.NewRandom()
		persons[i].EpisodeID = episodeID
		persons[i].ChannelID = ""
		persons[i].CreatedAt = now
		if _, err := r.put(persons[i].ID, &persons[i]); err != nil {
			return err
		}
	}
	return nil
}

var _ model.PodcastPersonRepository = (*podcastPersonRepository)(nil)
