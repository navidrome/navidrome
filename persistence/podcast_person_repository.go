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

func NewPodcastPersonRepository(db dbx.Builder) model.PodcastPersonRepository {
	r := &podcastPersonRepository{}
	r.db = db
	r.registerModel(&model.PodcastPerson{}, nil)
	return r
}

func (r *podcastPersonRepository) GetByChannel(ctx context.Context, channelID string) (model.PodcastPersons, error) {
	sel := r.newSelect(ctx).Columns("*").Where(Eq{"channel_id": channelID})
	var result model.PodcastPersons
	err := r.queryAll(ctx, sel, &result)
	return result, err
}

func (r *podcastPersonRepository) GetByEpisode(ctx context.Context, episodeID string) (model.PodcastPersons, error) {
	sel := r.newSelect(ctx).Columns("*").Where(Eq{"episode_id": episodeID})
	var result model.PodcastPersons
	err := r.queryAll(ctx, sel, &result)
	return result, err
}

func (r *podcastPersonRepository) GetByEpisodes(ctx context.Context, episodeIDs []string) (model.PodcastPersons, error) {
	if len(episodeIDs) == 0 {
		return nil, nil
	}
	sel := r.newSelect(ctx).Columns("*").Where(Eq{"episode_id": episodeIDs})
	var result model.PodcastPersons
	err := r.queryAll(ctx, sel, &result)
	return result, err
}

func (r *podcastPersonRepository) SaveForChannel(ctx context.Context, channelID string, persons []model.PodcastPerson) error {
	if err := r.delete(ctx, Eq{"channel_id": channelID}); err != nil {
		return err
	}
	now := time.Now()
	for i := range persons {
		persons[i].ID = id.NewRandom()
		persons[i].ChannelID = channelID
		persons[i].EpisodeID = ""
		persons[i].CreatedAt = now
		if _, err := r.put(ctx, persons[i].ID, &persons[i]); err != nil {
			return err
		}
	}
	return nil
}

func (r *podcastPersonRepository) SaveForEpisode(ctx context.Context, episodeID string, persons []model.PodcastPerson) error {
	if err := r.delete(ctx, Eq{"episode_id": episodeID}); err != nil {
		return err
	}
	now := time.Now()
	for i := range persons {
		persons[i].ID = id.NewRandom()
		persons[i].EpisodeID = episodeID
		persons[i].ChannelID = ""
		persons[i].CreatedAt = now
		if _, err := r.put(ctx, persons[i].ID, &persons[i]); err != nil {
			return err
		}
	}
	return nil
}

var _ model.PodcastPersonRepository = (*podcastPersonRepository)(nil)
