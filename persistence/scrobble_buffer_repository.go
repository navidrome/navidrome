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

func NewScrobbleBufferRepository(db dbx.Builder) model.ScrobbleBufferRepository {
	r := &scrobbleBufferRepository{}
	r.db = db
	r.tableName = "scrobble_buffer"
	return r
}

func (r *scrobbleBufferRepository) UserIDs(ctx context.Context, service string) ([]string, error) {
	sql := Select().Columns("user_id").
		From(r.tableName).
		Where(And{
			Eq{"service": service},
		}).
		GroupBy("user_id").
		OrderBy("count(*)")
	var userIds []string
	err := r.queryAllSlice(ctx, sql, &userIds)
	return userIds, err
}

func (r *scrobbleBufferRepository) Enqueue(ctx context.Context, service, userId, mediaFileId string, playTime time.Time) error {
	ins := Insert(r.tableName).SetMap(map[string]any{
		"id":            id.NewRandom(),
		"user_id":       userId,
		"service":       service,
		"media_file_id": mediaFileId,
		"play_time":     playTime,
		"enqueue_time":  time.Now(),
	})
	_, err := r.executeSQL(ctx, ins)
	return err
}

func (r *scrobbleBufferRepository) Next(ctx context.Context, service string, userId string) (*model.ScrobbleEntry, error) {
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
	err := r.queryOne(ctx, sql, &res)
	if errors.Is(err, model.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	res.ScrobbleEntry.Participants, err = r.getParticipants(ctx, &res.ScrobbleEntry.MediaFile)
	if err != nil {
		return nil, err
	}
	return res.ScrobbleEntry, nil
}

func (r *scrobbleBufferRepository) Dequeue(ctx context.Context, entry *model.ScrobbleEntry) error {
	return r.delete(ctx, Eq{"id": entry.ID})
}

func (r *scrobbleBufferRepository) Discard(ctx context.Context, service string) error {
	return r.delete(ctx, Eq{"service": service})
}

func (r *scrobbleBufferRepository) Length(ctx context.Context) (int64, error) {
	return r.count(ctx, Select())
}

var _ model.ScrobbleBufferRepository = (*scrobbleBufferRepository)(nil)
