package persistence

import (
	"context"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/pocketbase/dbx"
)

type podcastEpisodeRepository struct {
	sqlRepository
}

func NewPodcastEpisodeRepository(db dbx.Builder) model.PodcastEpisodeRepository {
	r := &podcastEpisodeRepository{}
	r.db = db
	r.registerModel(&model.PodcastEpisode{}, nil)
	return r
}

func (r *podcastEpisodeRepository) Get(ctx context.Context, epID string) (*model.PodcastEpisode, error) {
	sel := r.newSelect(ctx).Columns("*").Where(Eq{"id": epID})
	res := model.PodcastEpisode{}
	if err := r.queryOne(ctx, sel, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *podcastEpisodeRepository) GetNewest(ctx context.Context, count int) (model.PodcastEpisodes, error) {
	sel := r.newSelect(ctx).Columns("*").OrderBy("publish_date DESC").Limit(uint64(count))
	var eps model.PodcastEpisodes
	err := r.queryAll(ctx, sel, &eps)
	return eps, err
}

func (r *podcastEpisodeRepository) GetByChannel(ctx context.Context, channelID string) (model.PodcastEpisodes, error) {
	sel := r.newSelect(ctx).Columns("*").Where(Eq{"channel_id": channelID}).OrderBy("publish_date DESC")
	var eps model.PodcastEpisodes
	err := r.queryAll(ctx, sel, &eps)
	return eps, err
}

func (r *podcastEpisodeRepository) GetByChannels(ctx context.Context, channelIDs []string) (model.PodcastEpisodes, error) {
	if len(channelIDs) == 0 {
		return nil, nil
	}
	sel := r.newSelect(ctx).Columns("*").Where(Eq{"channel_id": channelIDs}).OrderBy("channel_id, publish_date DESC")
	var eps model.PodcastEpisodes
	err := r.queryAll(ctx, sel, &eps)
	return eps, err
}

func (r *podcastEpisodeRepository) GetByGUID(ctx context.Context, channelID, guid string) (*model.PodcastEpisode, error) {
	sel := r.newSelect(ctx).Columns("*").Where(Eq{"channel_id": channelID, "guid": guid})
	res := model.PodcastEpisode{}
	if err := r.queryOne(ctx, sel, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *podcastEpisodeRepository) Create(ctx context.Context, ep *model.PodcastEpisode) error {
	now := time.Now()
	ep.CreatedAt = now
	ep.UpdatedAt = now
	if ep.ID == "" {
		ep.ID = id.NewRandom()
	}
	_, err := r.put(ctx, ep.ID, ep)
	return err
}

func (r *podcastEpisodeRepository) Update(ctx context.Context, ep *model.PodcastEpisode) error {
	ep.UpdatedAt = time.Now()
	_, err := r.put(ctx, ep.ID, ep)
	return err
}

func (r *podcastEpisodeRepository) Delete(ctx context.Context, epID string) error {
	return r.delete(ctx, Eq{"id": epID})
}

var _ model.PodcastEpisodeRepository = (*podcastEpisodeRepository)(nil)
