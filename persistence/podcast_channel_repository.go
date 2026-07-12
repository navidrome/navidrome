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
	r.registerModel(&model.PodcastChannel{}, map[string]filterFunc{
		"title":  containsFilter("title"),
		"status": eqFilter,
	})
	return r
}

func (r *podcastChannelRepository) isPermitted() bool {
	user := loggedUser(r.ctx)
	return user.IsAdmin
}

func (r *podcastChannelRepository) CountAll(options ...model.QueryOptions) (int64, error) {
	sql := r.newSelect()
	return r.count(sql, options...)
}

func (r *podcastChannelRepository) Delete(id string) error {
	if !r.isPermitted() {
		return rest.ErrPermissionDenied
	}
	return r.delete(Eq{"id": id})
}

func (r *podcastChannelRepository) Get(id string) (*model.PodcastChannel, error) {
	sel := r.newSelect().Where(Eq{"id": id}).Columns("*")
	res := model.PodcastChannel{}
	err := r.queryOne(sel, &res)
	return &res, err
}

func (r *podcastChannelRepository) GetAll(options ...model.QueryOptions) (model.PodcastChannels, error) {
	sel := r.newSelect(options...).Columns("*")
	res := model.PodcastChannels{}
	err := r.queryAll(sel, &res)
	return res, err
}

func (r *podcastChannelRepository) GetWithEpisodes(id string) (*model.PodcastChannel, error) {
	channel, err := r.Get(id)
	if err != nil {
		return nil, err
	}
	episodeRepo := NewPodcastEpisodeRepository(r.ctx, r.db)
	episodes, err := episodeRepo.GetAll(model.QueryOptions{
		Filters: Eq{"channel_id": id},
		Sort:    "publish_date",
		Order:   "desc",
	})
	if err != nil {
		return nil, err
	}
	channel.Episodes = episodes
	return channel, nil
}

func (r *podcastChannelRepository) FindByUrl(url string) (*model.PodcastChannel, error) {
	sel := r.newSelect().Where(Eq{"url": url}).Columns("*")
	res := model.PodcastChannel{}
	err := r.queryOne(sel, &res)
	return &res, err
}

func (r *podcastChannelRepository) Put(channel *model.PodcastChannel, colsToUpdate ...string) error {
	if !r.isPermitted() {
		return rest.ErrPermissionDenied
	}

	channel.UpdatedAt = time.Now()
	if channel.ID == "" {
		channel.CreatedAt = time.Now()
		channel.ID = id.NewRandom()
	}
	if len(colsToUpdate) > 0 {
		colsToUpdate = append(colsToUpdate, "UpdatedAt")
	}
	_, err := r.put(channel.ID, channel, colsToUpdate...)
	return err
}

func (r *podcastChannelRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(r.parseRestOptions(r.ctx, options...))
}

func (r *podcastChannelRepository) EntityName() string {
	return "podcastChannel"
}

func (r *podcastChannelRepository) NewInstance() any {
	return &model.PodcastChannel{}
}

func (r *podcastChannelRepository) Read(id string) (any, error) {
	return r.Get(id)
}

func (r *podcastChannelRepository) ReadAll(options ...rest.QueryOptions) (any, error) {
	return r.GetAll(r.parseRestOptions(r.ctx, options...))
}

func (r *podcastChannelRepository) Save(entity any) (string, error) {
	t := entity.(*model.PodcastChannel)
	if !r.isPermitted() {
		return "", rest.ErrPermissionDenied
	}
	err := r.Put(t)
	if errors.Is(err, model.ErrNotFound) {
		return "", rest.ErrNotFound
	}
	return t.ID, err
}

func (r *podcastChannelRepository) Update(id string, entity any, cols ...string) error {
	t := entity.(*model.PodcastChannel)
	t.ID = id
	if !r.isPermitted() {
		return rest.ErrPermissionDenied
	}
	err := r.Put(t)
	if errors.Is(err, model.ErrNotFound) {
		return rest.ErrNotFound
	}
	return err
}

var _ model.PodcastChannelRepository = (*podcastChannelRepository)(nil)
var _ rest.Repository = (*podcastChannelRepository)(nil)
var _ rest.Persistable = (*podcastChannelRepository)(nil)
