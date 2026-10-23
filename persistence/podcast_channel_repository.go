package persistence

import (
	"context"
	"errors"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/pocketbase/dbx"
)

type podcastChannelRepository struct {
	sqlRepository
}

func NewPodcastChannelRepository(db dbx.Builder) model.PodcastChannelRepository {
	r := &podcastChannelRepository{}
	r.db = db
	r.registerModel(&model.PodcastChannel{}, nil)
	return r
}

func (r *podcastChannelRepository) isPermitted(ctx context.Context) bool {
	return loggedUser(ctx).IsAdmin
}

func (r *podcastChannelRepository) Get(ctx context.Context, chanID string) (*model.PodcastChannel, error) {
	sel := r.newSelect(ctx).Columns("*").Where(Eq{"id": chanID})
	res := model.PodcastChannel{}
	if err := r.queryOne(ctx, sel, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *podcastChannelRepository) GetAll(ctx context.Context, withEpisodes bool) (model.PodcastChannels, error) {
	sel := r.newSelect(ctx).Columns("*").OrderBy("title")
	var channels model.PodcastChannels
	if err := r.queryAll(ctx, sel, &channels); err != nil {
		return nil, err
	}
	if withEpisodes && len(channels) > 0 {
		ids := make([]string, len(channels))
		for i, ch := range channels {
			ids[i] = ch.ID
		}
		epRepo := NewPodcastEpisodeRepository(r.db)
		allEps, err := epRepo.GetByChannels(ctx, ids)
		if err != nil {
			return nil, err
		}
		epsByChannel := make(map[string]model.PodcastEpisodes, len(channels))
		for _, ep := range allEps {
			epsByChannel[ep.ChannelID] = append(epsByChannel[ep.ChannelID], ep)
		}
		for i := range channels {
			channels[i].Episodes = epsByChannel[channels[i].ID]
		}
	}
	return channels, nil
}

func (r *podcastChannelRepository) ExistsByURL(ctx context.Context, url string) (bool, error) {
	sel := r.newSelect(ctx).Columns("count(*)").Where(Eq{"url": url})
	count, err := r.count(ctx, sel)
	return count > 0, err
}

func (r *podcastChannelRepository) Create(ctx context.Context, channel *model.PodcastChannel) error {
	if !r.isPermitted(ctx) {
		return rest.ErrPermissionDenied
	}
	now := time.Now()
	channel.CreatedAt = now
	channel.UpdatedAt = now
	if channel.ID == "" {
		channel.ID = id.NewRandom()
	}
	_, err := r.put(ctx, channel.ID, channel)
	return err
}

func (r *podcastChannelRepository) UpdateChannel(ctx context.Context, channel *model.PodcastChannel) error {
	if !r.isPermitted(ctx) {
		return rest.ErrPermissionDenied
	}
	channel.UpdatedAt = time.Now()
	_, err := r.put(ctx, channel.ID, channel)
	return err
}

func (r *podcastChannelRepository) Delete(ctx context.Context, ids ...string) error {
	if !r.isPermitted(ctx) {
		return rest.ErrPermissionDenied
	}
	if len(ids) == 0 {
		return nil
	}
	return r.delete(ctx, Eq{"id": ids})
}

func (r *podcastChannelRepository) Read(ctx context.Context, chanID string) (*model.PodcastChannel, error) {
	return r.Get(ctx, chanID)
}

func (r *podcastChannelRepository) ReadAll(ctx context.Context, options ...rest.QueryOptions) ([]model.PodcastChannel, error) {
	sel := r.newSelect(ctx, r.parseRestOptions(ctx, options...)).Columns("*")
	var channels model.PodcastChannels
	err := r.queryAll(ctx, sel, &channels)
	return channels, err
}

func (r *podcastChannelRepository) Save(ctx context.Context, ch *model.PodcastChannel) (string, error) {
	if !r.isPermitted(ctx) {
		return "", rest.ErrPermissionDenied
	}
	err := r.Create(ctx, ch)
	if errors.Is(err, model.ErrNotFound) {
		return "", rest.ErrNotFound
	}
	return ch.ID, err
}

func (r *podcastChannelRepository) Update(ctx context.Context, id string, entity model.PodcastChannel, cols ...string) error {
	ch := &entity
	ch.ID = id
	if !r.isPermitted(ctx) {
		return rest.ErrPermissionDenied
	}
	return r.UpdateChannel(ctx, ch)
}

func (r *podcastChannelRepository) Count(ctx context.Context, options ...rest.QueryOptions) (int64, error) {
	sql := r.newSelect(ctx, r.parseRestOptions(ctx, options...))
	return r.count(ctx, sql)
}

var _ model.PodcastChannelRepository = (*podcastChannelRepository)(nil)
