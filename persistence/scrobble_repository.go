package persistence

import (
	"context"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/model"
	"github.com/pocketbase/dbx"
)

type scrobbleRepository struct {
	sqlRepository
}

func fromTs(_ string, value any) Sqlizer {
	return GtOrEq{"scrobbles.submission_time": value}
}

func toTs(_ string, value any) Sqlizer {
	return LtOrEq{"scrobbles.submission_time": value}
}

func (r *scrobbleRepository) baseQuery(ctx context.Context, options ...model.QueryOptions) SelectBuilder {
	user := loggedUser(ctx)

	return r.newSelect(ctx, options...).
		Columns("id", "media_file_id", "submission_time").
		Where(Eq{"scrobbles.user_id": user.ID})
}

func NewScrobbleRepository(db dbx.Builder) model.ScrobbleRepository {
	r := &scrobbleRepository{}
	r.db = db
	r.tableName = "scrobbles"
	r.registerModel(&model.Scrobble{}, map[string]filterFunc{
		"from": fromTs,
		"to":   toTs,
	})
	r.setSortMappings(map[string]string{
		"submission_time": "submission_time",
	})
	return r
}

func (r *scrobbleRepository) RecordScrobble(ctx context.Context, mediaFileID string, submissionTime time.Time) error {
	userID := loggedUser(ctx).ID
	values := map[string]any{
		"media_file_id":   mediaFileID,
		"user_id":         userID,
		"submission_time": submissionTime.Unix(),
	}
	insert := Insert(r.tableName).SetMap(values)
	_, err := r.executeSQL(ctx, insert)
	return err
}

func (r *scrobbleRepository) CountAll(ctx context.Context, options ...model.QueryOptions) (int64, error) {
	return r.count(ctx, r.baseQuery(ctx), options...)
}

func (r *scrobbleRepository) Count(ctx context.Context, options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(ctx, r.parseRestOptions(ctx, options...))
}

func (r *scrobbleRepository) Get(ctx context.Context, id string) (*model.Scrobble, error) {
	sel := r.baseQuery(ctx).Where(Eq{"id": id})
	var res model.Scrobble
	err := r.queryOne(ctx, sel, &res)
	return &res, err
}

func (r *scrobbleRepository) GetAll(ctx context.Context, options ...model.QueryOptions) (model.Scrobbles, error) {
	sel := r.baseQuery(ctx, options...)
	var scrobbles model.Scrobbles
	err := r.queryAll(ctx, sel, &scrobbles)
	return scrobbles, err
}

func (r *scrobbleRepository) Read(ctx context.Context, id string) (*model.Scrobble, error) {
	return r.Get(ctx, id)
}

func (r *scrobbleRepository) ReadAll(ctx context.Context, options ...rest.QueryOptions) ([]model.Scrobble, error) {
	return r.GetAll(ctx, r.parseRestOptions(ctx, options...))
}

var _ model.ScrobbleRepository = (*scrobbleRepository)(nil)
var _ rest.Repository[model.Scrobble] = (*scrobbleRepository)(nil)
