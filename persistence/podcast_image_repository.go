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

func NewPodcastImageRepository(ctx context.Context, db dbx.Builder) model.PodcastImageRepository {
	r := &podcastImageRepository{}
	r.ctx = ctx
	r.db = db
	r.tableName = "podcast_image"
	r.registerModel(&model.PodcastImage{}, nil)
	return r
}

func (r *podcastImageRepository) GetByChannel(channelID string) (model.PodcastImages, error) {
	sel := r.newSelect().Columns("*").Where(Eq{"channel_id": channelID})
	var result model.PodcastImages
	err := r.queryAll(sel, &result)
	return result, err
}

func (r *podcastImageRepository) GetByChannels(channelIDs []string) (model.PodcastImages, error) {
	if len(channelIDs) == 0 {
		return nil, nil
	}
	sel := r.newSelect().Columns("*").Where(Eq{"channel_id": channelIDs})
	var result model.PodcastImages
	err := r.queryAll(sel, &result)
	return result, err
}

func (r *podcastImageRepository) GetByEpisode(episodeID string) (model.PodcastImages, error) {
	sel := r.newSelect().Columns("*").Where(Eq{"episode_id": episodeID})
	var result model.PodcastImages
	err := r.queryAll(sel, &result)
	return result, err
}

func (r *podcastImageRepository) GetByEpisodes(episodeIDs []string) (model.PodcastImages, error) {
	if len(episodeIDs) == 0 {
		return nil, nil
	}
	sel := r.newSelect().Columns("*").Where(Eq{"episode_id": episodeIDs})
	var result model.PodcastImages
	err := r.queryAll(sel, &result)
	return result, err
}

func (r *podcastImageRepository) SaveForChannel(channelID string, images []model.PodcastImage) error {
	if err := r.delete(Eq{"channel_id": channelID}); err != nil {
		return err
	}
	now := time.Now()
	for i := range images {
		images[i].ID = id.NewRandom()
		images[i].ChannelID = channelID
		images[i].EpisodeID = ""
		images[i].CreatedAt = now
		if _, err := r.put(images[i].ID, &images[i]); err != nil {
			return err
		}
	}
	return nil
}

func (r *podcastImageRepository) SaveForEpisode(episodeID string, images []model.PodcastImage) error {
	if err := r.delete(Eq{"episode_id": episodeID}); err != nil {
		return err
	}
	now := time.Now()
	for i := range images {
		images[i].ID = id.NewRandom()
		images[i].EpisodeID = episodeID
		images[i].ChannelID = ""
		images[i].CreatedAt = now
		if _, err := r.put(images[i].ID, &images[i]); err != nil {
			return err
		}
	}
	return nil
}

var _ model.PodcastImageRepository = (*podcastImageRepository)(nil)
