package persistence

import (
	"context"
	"errors"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/beego/beego/v2/client/orm"
	"github.com/navidrome/navidrome/model"
)

type scrobbleRadioRepository struct {
	sqlRepository
}

func NewScrobbleRadioRepository(ctx context.Context, o orm.QueryExecutor) model.ScrobbleRadioRepository {
	r := &scrobbleRadioRepository{}
	r.ctx = ctx
	r.ormer = o
	r.tableName = "scrobble_radio"
	return r
}

func (r *scrobbleRadioRepository) UserIDs(service string) ([]string, error) {
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

func (r *scrobbleRadioRepository) Enqueue(service, userId, artist, title string, playTime time.Time) error {
	ins := Insert(r.tableName).SetMap(map[string]interface{}{
		"artist":       artist,
		"enqueue_time": time.Now(),
		"play_time":    playTime,
		"service":      service,
		"title":        title,
		"user_id":      userId,
	})
	_, err := r.executeSQL(ins)
	return err
}

func (r *scrobbleRadioRepository) Next(service string, userId string) (*model.ScrobleRadioEntry, error) {
	sql := Select().Columns("*").
		From(r.tableName+" s").
		Where(And{
			Eq{"service": service},
			Eq{"user_id": userId},
		}).
		OrderBy("play_time", "s.rowid").Limit(1)

	res := model.ScrobleRadioEntry{}
	err := r.queryOne(sql, &res)
	if errors.Is(err, model.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *scrobbleRadioRepository) Dequeue(entry *model.ScrobleRadioEntry) error {
	return r.delete(And{
		Eq{"artist": entry.Artist},
		Eq{"play_time": entry.PlayTime},
		Eq{"title": entry.Title},
		Eq{"service": entry.Service},
	})
}

func (r *scrobbleRadioRepository) Length() (int64, error) {
	return r.count(Select())
}

var _ model.ScrobbleRadioRepository = (*scrobbleRadioRepository)(nil)
