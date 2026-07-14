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

func (r *scrobbleRepository) baseQuery(options ...model.QueryOptions) SelectBuilder {
	user := loggedUser(r.ctx)

	return r.newSelect(options...).
		Columns("id", "media_file_id", "submission_time").
		Where(Eq{"scrobbles.user_id": user.ID})
}

func NewScrobbleRepository(ctx context.Context, db dbx.Builder) model.ScrobbleRepository {
	r := &scrobbleRepository{}
	r.ctx = ctx
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

func (r *scrobbleRepository) RecordScrobble(mediaFileID string, submissionTime time.Time) error {
	userID := loggedUser(r.ctx).ID
	values := map[string]any{
		"media_file_id":   mediaFileID,
		"user_id":         userID,
		"submission_time": submissionTime.Unix(),
	}
	insert := Insert(r.tableName).SetMap(values)
	_, err := r.executeSQL(insert)
	return err
}

func (r *scrobbleRepository) CountAll(options ...model.QueryOptions) (int64, error) {
	return r.count(r.baseQuery(), options...)
}

func (r *scrobbleRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(r.parseRestOptions(r.ctx, options...))
}

func (r *scrobbleRepository) Get(id string) (*model.Scrobble, error) {
	sel := r.baseQuery().Where(Eq{"id": id})
	var res model.Scrobble
	err := r.queryOne(sel, &res)
	return &res, err
}

func (r *scrobbleRepository) GetAll(options ...model.QueryOptions) (model.Scrobbles, error) {
	sel := r.baseQuery(options...)
	var scrobbles model.Scrobbles
	err := r.queryAll(sel, &scrobbles)
	return scrobbles, err
}

func (r *scrobbleRepository) Read(id string) (any, error) {
	return r.Get(id)
}

func (r *scrobbleRepository) ReadAll(options ...rest.QueryOptions) (any, error) {
	return r.GetAll(r.parseRestOptions(r.ctx, options...))
}

func (r *scrobbleRepository) EntityName() string {
	return "scrobble"
}

func (r *scrobbleRepository) NewInstance() any {
	return &model.Scrobble{}
}

var _ model.ScrobbleRepository = (*scrobbleRepository)(nil)
var _ model.ResourceRepository = (*scrobbleRepository)(nil)
