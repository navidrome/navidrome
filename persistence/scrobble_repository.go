package persistence

import (
	"context"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
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

type dbHistoryEntry struct {
	dbMediaFile
	SubmissionTime int64 `structs:"-"`
}

func (h *dbHistoryEntry) PostScan() error {
	return h.dbMediaFile.PostScan()
}

func (r *scrobbleRepository) GetHistory(offset, count int) ([]model.HistoryEntry, error) {
	userID := loggedUser(r.ctx).ID
	sq := Select("m.*", "s.submission_time").
		From(r.tableName+" s").
		LeftJoin("media_file m ON m.id = s.media_file_id").
		Where(Eq{"s.user_id": userID}).
		OrderBy("s.submission_time DESC").
		Offset(uint64(offset)).
		Limit(uint64(count))

	var rows []dbHistoryEntry
	if err := r.queryAll(sq, &rows); err != nil {
		return nil, err
	}

	entries := make([]model.HistoryEntry, 0, len(rows))
	for i := range rows {
		entry := model.HistoryEntry{
			MediaFile: *rows[i].MediaFile,
			PlayedAt:  time.Unix(rows[i].SubmissionTime, 0),
		}
		var err error
		entry.Participants, err = r.getParticipants(rows[i].MediaFile)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

var _ model.ScrobbleRepository = (*scrobbleRepository)(nil)
