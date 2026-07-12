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

type podcastEpisodeRepository struct {
	sqlRepository
}

func NewPodcastEpisodeRepository(ctx context.Context, db dbx.Builder) model.PodcastEpisodeRepository {
	r := &podcastEpisodeRepository{}
	r.ctx = ctx
	r.db = db
	r.registerModel(&model.PodcastEpisode{}, map[string]filterFunc{
		"title":           containsFilter("title"),
		"channel_id":      eqFilter,
		"download_status": eqFilter,
	})
	return r
}

func (r *podcastEpisodeRepository) isPermitted() bool {
	user := loggedUser(r.ctx)
	return user.IsAdmin
}

func (r *podcastEpisodeRepository) CountAll(options ...model.QueryOptions) (int64, error) {
	sql := r.newSelect()
	return r.count(sql, options...)
}

func (r *podcastEpisodeRepository) Delete(id string) error {
	if !r.isPermitted() {
		return rest.ErrPermissionDenied
	}
	return r.delete(Eq{"id": id})
}

// withPlayAnnotation left-joins the current user's annotation row, exposing
// play_count/play_date as a "listened" signal. Doesn't reuse
// sqlRepository.withAnnotation, which also selects starred/rating/
// average_rating - podcast episodes have no average_rating column and don't
// support starring/rating (yet).
func (r *podcastEpisodeRepository) withPlayAnnotation(sel SelectBuilder) SelectBuilder {
	userID := loggedUser(r.ctx).ID
	if userID == invalidUserId {
		return sel
	}
	return sel.
		LeftJoin("annotation on ("+
			"annotation.item_id = podcast_episode.id"+
			" AND annotation.item_type = 'podcast_episode'"+
			" AND annotation.user_id = '"+userID+"')").
		Columns("coalesce(play_count, 0) as play_count", "play_date")
}

func (r *podcastEpisodeRepository) Get(id string) (*model.PodcastEpisode, error) {
	sel := r.withPlayAnnotation(r.newSelect().Where(Eq{"id": id}).Columns("*"))
	res := model.PodcastEpisode{}
	err := r.queryOne(sel, &res)
	return &res, err
}

func (r *podcastEpisodeRepository) GetAll(options ...model.QueryOptions) (model.PodcastEpisodes, error) {
	sel := r.withPlayAnnotation(r.newSelect(options...).Columns("*"))
	res := model.PodcastEpisodes{}
	err := r.queryAll(sel, &res)
	return res, err
}

func (r *podcastEpisodeRepository) FindByGuid(channelID, guid string) (*model.PodcastEpisode, error) {
	sel := r.newSelect().Where(And{Eq{"channel_id": channelID}, Eq{"guid": guid}}).Columns("*")
	res := model.PodcastEpisode{}
	err := r.queryOne(sel, &res)
	return &res, err
}

func (r *podcastEpisodeRepository) GetNewest(count int) (model.PodcastEpisodes, error) {
	sel := r.withPlayAnnotation(r.newSelect().Columns("*")).OrderBy("publish_date desc").Limit(uint64(count))
	res := model.PodcastEpisodes{}
	err := r.queryAll(sel, &res)
	return res, err
}

func (r *podcastEpisodeRepository) Put(episode *model.PodcastEpisode, colsToUpdate ...string) error {
	if !r.isPermitted() {
		return rest.ErrPermissionDenied
	}

	episode.UpdatedAt = time.Now()
	if episode.ID == "" {
		episode.CreatedAt = time.Now()
		episode.ID = id.NewRandom()
	}
	if len(colsToUpdate) > 0 {
		colsToUpdate = append(colsToUpdate, "UpdatedAt")
	}
	_, err := r.put(episode.ID, episode, colsToUpdate...)
	return err
}

func (r *podcastEpisodeRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(r.parseRestOptions(r.ctx, options...))
}

func (r *podcastEpisodeRepository) EntityName() string {
	return "podcastEpisode"
}

func (r *podcastEpisodeRepository) NewInstance() any {
	return &model.PodcastEpisode{}
}

func (r *podcastEpisodeRepository) Read(id string) (any, error) {
	return r.Get(id)
}

func (r *podcastEpisodeRepository) ReadAll(options ...rest.QueryOptions) (any, error) {
	return r.GetAll(r.parseRestOptions(r.ctx, options...))
}

func (r *podcastEpisodeRepository) Save(entity any) (string, error) {
	t := entity.(*model.PodcastEpisode)
	if !r.isPermitted() {
		return "", rest.ErrPermissionDenied
	}
	err := r.Put(t)
	if errors.Is(err, model.ErrNotFound) {
		return "", rest.ErrNotFound
	}
	return t.ID, err
}

func (r *podcastEpisodeRepository) Update(id string, entity any, cols ...string) error {
	t := entity.(*model.PodcastEpisode)
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

var _ model.PodcastEpisodeRepository = (*podcastEpisodeRepository)(nil)
var _ rest.Repository = (*podcastEpisodeRepository)(nil)
var _ rest.Persistable = (*podcastEpisodeRepository)(nil)
