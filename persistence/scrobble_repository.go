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
	dbMediaFile
	RowId          int64 `structs:"row_id" json:"rowId"`
	SubmissionTime int64 `structs:"submission_time" json:"submissionTime"`
}

func (m dbScrobble) toScrobble() model.Scrobble {
	return model.Scrobble{
		MediaFile:      *m.MediaFile,
		RowId:          m.RowId,
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
		Columns("scrobbles.ROWID row_id", "submission_time", "media_file.*", "library.path as library_path", "library.name as library_name").
		Join("media_file on media_file.id = media_file_id").
		LeftJoin("library on media_file.library_id = library.id").
		LeftJoin("annotation on ("+
			"annotation.item_id = media_file.id"+
			" AND annotation.item_type = 'media_file'"+
			" AND annotation.user_id = '"+user.ID+"')").
		Columns(
			"coalesce(starred, 0) as starred",
			"coalesce(rating, 0) as rating",
			"starred_at",
			"play_date",
			"coalesce(play_count, 0) as play_count",
		).
		Where(Eq{"scrobbles.user_id": user.ID})
}

func NewScrobbleRepository(ctx context.Context, db dbx.Builder) model.ScrobbleRepository {
	r := &scrobbleRepository{}
	r.ctx = ctx
	r.db = db
	r.tableName = "scrobbles"
	r.registerModel(&model.Scrobble{}, map[string]filterFunc{
		"from":  fromTs,
		"to":    toTs,
		"title": fullTextFilter("media_file"),
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
	user := loggedUser(r.ctx)

	sel := r.newSelect().
		Columns("count(*) count").
		Join("media_file on media_file.id = media_file_id").
		Where(Eq{"user_id": user.ID})

	sel = r.applyFilters(sel, options...)

	var res struct{ Count int64 }
	err := r.queryOne(sel, &res)
	return res.Count, err
}

func (r *scrobbleRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(r.parseRestOptions(r.ctx, options...))
}

func (r *scrobbleRepository) Get(id string) (*model.Scrobble, error) {
	sel := r.baseQuery().Where(Eq{"row_id": id})
	res := dbScrobble{}
	err := r.queryOne(sel, &res)
	if err != nil {
		return nil, err
	}
	model := res.toScrobble()
	return &model, nil
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
