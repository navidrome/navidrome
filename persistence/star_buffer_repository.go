package persistence

import (
	"context"
	"errors"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/beego/beego/v2/client/orm"
	"github.com/navidrome/navidrome/model"
)

type starBufferRepository struct {
	sqlRepository
}

func NewStarBufferRepository(ctx context.Context, o orm.QueryExecutor) model.StarBufferRepository {
	r := &starBufferRepository{}
	r.ctx = ctx
	r.ormer = o
	r.tableName = "star_buffer"
	return r
}

func (r *starBufferRepository) UserIDs(service string) ([]string, error) {
	sql := Select().Columns("user_id").
		From(r.tableName).
		Where(And{
			Eq{"service": service},
		}).
		GroupBy("user_id").
		OrderBy("count(*)")
	var userIds []string
	err := r.queryAll(sql, &userIds)
	return userIds, err
}

func (r *starBufferRepository) TryUpdate(service, userId, mediaFileId string, isStar bool) (bool, error) {
	upd := Update(r.tableName).
		Set("is_star", isStar).
		Where(Eq{
			"user_id":       userId,
			"media_file_id": mediaFileId,
		})
	count, err := r.executeSQL(upd)

	if err != nil {
		return false, err
	}
	return count == 1, err
}

func (r *starBufferRepository) Enqueue(service, userId, mediaFileId string, isStar bool) error {
	ins := Insert(r.tableName).SetMap(map[string]interface{}{
		"user_id":       userId,
		"service":       service,
		"media_file_id": mediaFileId,
		"enqueue_time":  time.Now(),
		"is_star":       isStar,
	})
	_, err := r.executeSQL(ins)
	return err
}

func (r *starBufferRepository) Next(service string, userId string) (*model.StarEntry, error) {
	sql := Select().Columns("s.*, m.*").
		From(r.tableName + " s").
		LeftJoin("media_file m on m.id = s.media_file_id").
		Where(And{
			Eq{"service": service},
			Eq{"user_id": userId},
		}).
		OrderBy("s.rowid").Limit(1)

	res := model.StarEntries{}
	// TODO Rewrite queryOne to use QueryRows, to workaround the recursive embedded structs issue
	err := r.queryAll(sql, &res)
	if errors.Is(err, model.ErrNotFound) || len(res) == 0 {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &res[0], nil
}

func (r *starBufferRepository) Dequeue(entry *model.StarEntry) error {
	return r.delete(And{
		Eq{"service": entry.Service},
		Eq{"media_file_id": entry.MediaFile.ID},
		Eq{"user_id": entry.UserID},
	})
}

func (r *starBufferRepository) Length() (int64, error) {
	return r.count(Select())
}

var _ model.StarBufferRepository = (*starBufferRepository)(nil)
