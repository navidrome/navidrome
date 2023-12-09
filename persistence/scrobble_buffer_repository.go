package persistence

import (
	"context"
	"errors"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/pocketbase/dbx"
)

type scrobbleBufferRepository struct {
	sqlRepository
}

func NewScrobbleBufferRepository(ctx context.Context, db dbx.Builder) model.ScrobbleBufferRepository {
	r := &scrobbleBufferRepository{}
	r.ctx = ctx
	r.db = db
	r.tableName = "scrobble_buffer"
	return r
}

func (r *scrobbleBufferRepository) UserIDs(service string) ([]string, error) {
	sql := Select().Columns("user_id").
		From(r.tableName).
		Where(And{
			Eq{"service": service},
		}).
		GroupBy("user_id").
		OrderBy("count(*)")
	var userIds []string
	err := r.queryAllSlice(sql, &userIds)
	return userIds, err
}

func (r *scrobbleBufferRepository) Enqueue(service, userId, mediaFileId string, playTime time.Time) error {
	ins := Insert(r.tableName).SetMap(map[string]interface{}{
		"user_id":       userId,
		"service":       service,
		"media_file_id": mediaFileId,
		"play_time":     playTime,
		"enqueue_time":  time.Now(),
	})
	_, err := r.executeSQL(ins)
	return err
}

func (r *scrobbleBufferRepository) Next(service string, userId string) (*model.ScrobbleEntry, error) {
	sql := Select().Columns("s.*, m.*").
		From(r.tableName+" s").
		LeftJoin("media_file m on m.id = s.media_file_id").
		Where(And{
			Eq{"service": service},
			Eq{"user_id": userId},
		}).
		OrderBy("play_time", "s.rowid").Limit(1)

	res := model.ScrobbleEntries{}
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

func (r *scrobbleBufferRepository) Dequeue(entry *model.ScrobbleEntry) error {
	return r.delete(And{
		Eq{"service": entry.Service},
		Eq{"media_file_id": entry.MediaFile.ID},
		Eq{"play_time": entry.PlayTime},
	})
}

func (r *scrobbleBufferRepository) Length() (int64, error) {
	return r.count(Select())
}

var _ model.ScrobbleBufferRepository = (*scrobbleBufferRepository)(nil)
