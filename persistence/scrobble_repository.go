package persistence

import (
	"context"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/pocketbase/dbx"
)

type scrobbleRepository struct {
	sqlRepository
}

func NewScrobbleRepository(ctx context.Context, db dbx.Builder) model.ScrobbleRepository {
	r := &scrobbleRepository{}
	r.ctx = ctx
	r.db = db
	r.tableName = "scrobbles"
	return r
}

func (r *scrobbleRepository) RecordScrobble(mediaFileID string, submissionTime time.Time, duration *int) (string, error) {
	userID := loggedUser(r.ctx).ID
	scrobbleID := id.NewRandom()
	values := map[string]interface{}{
		"id":              scrobbleID,
		"media_file_id":   mediaFileID,
		"user_id":         userID,
		"submission_time": submissionTime.Unix(),
		"duration":        duration,
	}
	insert := Insert(r.tableName).SetMap(values)
	_, err := r.executeSQL(insert)
	if err != nil {
		return "", err
	}
	return scrobbleID, nil
}

func (r *scrobbleRepository) UpdateDuration(id string, duration int) error {
	update := Update(r.tableName).Set("duration", duration).Where(Eq{"id": id})
	_, err := r.executeSQL(update)
	return err
}
