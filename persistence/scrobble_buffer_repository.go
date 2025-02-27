package persistence

import (
	"context"
	"errors"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/pocketbase/dbx"
)

type scrobbleBufferRepository struct {
	sqlRepository
}

type dbScrobbleBuffer struct {
	dbMediaFile
	*model.ScrobbleEntry `structs:",flatten"`
}

func (t *dbScrobbleBuffer) PostScan() error {
	if err := t.dbMediaFile.PostScan(); err != nil {
		return err
	}
	t.ScrobbleEntry.MediaFile = *t.dbMediaFile.MediaFile
	t.ScrobbleEntry.MediaFile.ID = t.MediaFileID
	return nil
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
		"id":            id.NewRandom(),
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
	// Put `s.*` last or else m.id overrides s.id
	sql := Select().Columns("m.*, s.*").
		From(r.tableName+" s").
		LeftJoin("media_file m on m.id = s.media_file_id").
		Where(And{
			Eq{"service": service},
			Eq{"user_id": userId},
		}).
		OrderBy("play_time", "s.rowid").Limit(1)

	var res dbScrobbleBuffer
	err := r.queryOne(sql, &res)
	if errors.Is(err, model.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	res.ScrobbleEntry.Participants, err = r.getParticipants(&res.ScrobbleEntry.MediaFile)
	if err != nil {
		return nil, err
	}
	return res.ScrobbleEntry, nil
}

func (r *scrobbleBufferRepository) Dequeue(entry *model.ScrobbleEntry) error {
	return r.delete(Eq{"id": entry.ID})
}

func (r *scrobbleBufferRepository) Length() (int64, error) {
	return r.count(Select())
}

var _ model.ScrobbleBufferRepository = (*scrobbleBufferRepository)(nil)
