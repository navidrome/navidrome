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

func NewPodcastChannelRepository(ctx context.Context, db dbx.Builder) model.PodcastChannelRepository {
	r := &podcastChannelRepository{}
	r.ctx = ctx
	r.db = db
	r.registerModel(&model.PodcastChannel{}, nil)
	return r
}

func (r *podcastChannelRepository) isPermitted() bool {
	return loggedUser(r.ctx).IsAdmin
}

func (r *podcastChannelRepository) Get(chanID string) (*model.PodcastChannel, error) {
	sel := r.newSelect().Columns("*").Where(Eq{"id": chanID})
	res := model.PodcastChannel{}
	if err := r.queryOne(sel, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *podcastChannelRepository) GetAll(withEpisodes bool) (model.PodcastChannels, error) {
	sel := r.newSelect().Columns("*").OrderBy("title")
	var channels model.PodcastChannels
	if err := r.queryAll(sel, &channels); err != nil {
		return nil, err
	}
	if withEpisodes && len(channels) > 0 {
		ids := make([]string, len(channels))
		for i, ch := range channels {
			ids[i] = ch.ID
		}
		epRepo := NewPodcastEpisodeRepository(r.ctx, r.db)
		allEps, err := epRepo.GetByChannels(ids)
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

func (r *podcastChannelRepository) ExistsByURL(url string) (bool, error) {
	sel := r.newSelect().Columns("count(*)").Where(Eq{"url": url})
	count, err := r.count(sel)
	return count > 0, err
}

func (r *podcastChannelRepository) Create(channel *model.PodcastChannel) error {
	if !r.isPermitted() {
		return rest.ErrPermissionDenied
	}
	now := time.Now()
	channel.CreatedAt = now
	channel.UpdatedAt = now
	if channel.ID == "" {
		channel.ID = id.NewRandom()
	}
	_, err := r.put(channel.ID, channel)
	return err
}

func (r *podcastChannelRepository) UpdateChannel(channel *model.PodcastChannel) error {
	if !r.isPermitted() {
		return rest.ErrPermissionDenied
	}
	channel.UpdatedAt = time.Now()
	_, err := r.put(channel.ID, channel)
	return err
}

func (r *podcastChannelRepository) Delete(chanID string) error {
	if !r.isPermitted() {
		return rest.ErrPermissionDenied
	}
	return r.delete(Eq{"id": chanID})
}

func (r *podcastChannelRepository) EntityName() string {
	return "podcast_channel"
}

func (r *podcastChannelRepository) NewInstance() any {
	return &model.PodcastChannel{}
}

func (r *podcastChannelRepository) Read(chanID string) (any, error) {
	return r.Get(chanID)
}

func (r *podcastChannelRepository) ReadAll(options ...rest.QueryOptions) (any, error) {
	sel := r.newSelect(r.parseRestOptions(r.ctx, options...)).Columns("*")
	var channels model.PodcastChannels
	err := r.queryAll(sel, &channels)
	return channels, err
}

func (r *podcastChannelRepository) Save(entity any) (string, error) {
	ch := entity.(*model.PodcastChannel)
	if !r.isPermitted() {
		return "", rest.ErrPermissionDenied
	}
	err := r.Create(ch)
	if errors.Is(err, model.ErrNotFound) {
		return "", rest.ErrNotFound
	}
	return ch.ID, err
}

func (r *podcastChannelRepository) Update(id string, entity any, cols ...string) error {
	ch := entity.(*model.PodcastChannel)
	ch.ID = id
	if !r.isPermitted() {
		return rest.ErrPermissionDenied
	}
	return r.UpdateChannel(ch)
}

func (r *podcastChannelRepository) Count(options ...rest.QueryOptions) (int64, error) {
	sql := r.newSelect(r.parseRestOptions(r.ctx, options...))
	return r.count(sql)
}

var _ model.PodcastChannelRepository = (*podcastChannelRepository)(nil)
var _ rest.Repository = (*podcastChannelRepository)(nil)
