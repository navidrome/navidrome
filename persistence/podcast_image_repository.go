package persistence

import (
	"context"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/pocketbase/dbx"
)

type podcastImageRepository struct {
	sqlRepository
}

func NewPodcastImageRepository(db dbx.Builder) model.PodcastImageRepository {
	r := &podcastImageRepository{}
	r.db = db
	r.tableName = "podcast_image"
	r.registerModel(&model.PodcastImage{}, nil)
	return r
}

func (r *podcastImageRepository) GetByChannel(ctx context.Context, channelID string) (model.PodcastImages, error) {
	sel := r.newSelect(ctx).Columns("*").Where(Eq{"channel_id": channelID})
	var result model.PodcastImages
	err := r.queryAll(ctx, sel, &result)
	return result, err
}

func (r *podcastImageRepository) GetByChannels(ctx context.Context, channelIDs []string) (model.PodcastImages, error) {
	if len(channelIDs) == 0 {
		return nil, nil
	}
	sel := r.newSelect(ctx).Columns("*").Where(Eq{"channel_id": channelIDs})
	var result model.PodcastImages
	err := r.queryAll(ctx, sel, &result)
	return result, err
}

func (r *podcastImageRepository) GetByEpisode(ctx context.Context, episodeID string) (model.PodcastImages, error) {
	sel := r.newSelect(ctx).Columns("*").Where(Eq{"episode_id": episodeID})
	var result model.PodcastImages
	err := r.queryAll(ctx, sel, &result)
	return result, err
}

func (r *podcastImageRepository) GetByEpisodes(ctx context.Context, episodeIDs []string) (model.PodcastImages, error) {
	if len(episodeIDs) == 0 {
		return nil, nil
	}
	sel := r.newSelect(ctx).Columns("*").Where(Eq{"episode_id": episodeIDs})
	var result model.PodcastImages
	err := r.queryAll(ctx, sel, &result)
	return result, err
}

func (r *podcastImageRepository) SaveForChannel(ctx context.Context, channelID string, images []model.PodcastImage) error {
	if err := r.delete(ctx, Eq{"channel_id": channelID}); err != nil {
		return err
	}
	now := time.Now()
	for i := range images {
		images[i].ID = id.NewRandom()
		images[i].ChannelID = channelID
		images[i].EpisodeID = ""
		images[i].CreatedAt = now
		if _, err := r.put(ctx, images[i].ID, &images[i]); err != nil {
			return err
		}
	}
	return nil
}

func (r *podcastImageRepository) SaveForEpisode(ctx context.Context, episodeID string, images []model.PodcastImage) error {
	if err := r.delete(ctx, Eq{"episode_id": episodeID}); err != nil {
		return err
	}
	now := time.Now()
	for i := range images {
		images[i].ID = id.NewRandom()
		images[i].EpisodeID = episodeID
		images[i].ChannelID = ""
		images[i].CreatedAt = now
		if _, err := r.put(ctx, images[i].ID, &images[i]); err != nil {
			return err
		}
	}
	return nil
}

var _ model.PodcastImageRepository = (*podcastImageRepository)(nil)
