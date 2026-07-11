package persistence

import (
	"context"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/slice"
	"github.com/pocketbase/dbx"
)

type scrobbleRepository struct {
	sqlRepository
}

type dbScrobble struct {
	MediaFileID    string `structs:"media_file_id" json:"mediaFileId"`
	RowId          int64  `structs:"row_id" json:"rowId"`
	SubmissionTime int64  `structs:"submission_time" json:"submissionTime"`
}

func (m dbScrobble) toScrobble() model.Scrobble {
	return model.Scrobble{
		MediaFileID:    m.MediaFileID,
		ID:             m.RowId,
		SubmissionTime: time.Unix(m.SubmissionTime, 0),
	}
}

type dbScrobbles []dbScrobble

func (m dbScrobbles) toModels() model.Scrobbles {
	return slice.Map(m, func(db dbScrobble) model.Scrobble {
		return db.toScrobble()
	})
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
		Columns("scrobbles.ROWID row_id", "media_file_id", "submission_time").
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
	count := r.baseQuery(options...).RemoveColumns().Column("COUNT() as count").RemoveOffset().RemoveLimit().OrderBy("scrobbles.ROWID")
	var res struct{ Count int64 }
	err := r.queryOne(count, &res)
	return res.Count, err
}

func (r *scrobbleRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(r.parseRestOptions(r.ctx, options...))
}

func (r *scrobbleRepository) Get(id string) (*model.Scrobble, error) {
	sel := r.baseQuery().Where(Eq{"row_id": id})
	var res dbScrobble
	err := r.queryOne(sel, &res)
	if err != nil {
		return nil, err
	}
	model := res.toScrobble()
	return &model, err
}

func (r *scrobbleRepository) GetAll(options ...model.QueryOptions) (model.Scrobbles, error) {
	sel := r.baseQuery(options...)
	var scrobbles dbScrobbles
	err := r.queryAll(sel, &scrobbles)
	if err != nil {
		return nil, err
	}
	return scrobbles.toModels(), nil
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
