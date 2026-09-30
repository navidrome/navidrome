package persistence

import (
	"context"
	"errors"
	"strings"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/slice"
	"github.com/pocketbase/dbx"
)

type playQueueRepository struct {
	sqlRepository
}

func NewPlayQueueRepository(db dbx.Builder) model.PlayQueueRepository {
	r := &playQueueRepository{}
	r.db = db
	r.tableName = "playqueue"
	return r
}

type playQueue struct {
	ID        string    `structs:"id"`
	UserID    string    `structs:"user_id"`
	Current   int       `structs:"current"`
	Position  int64     `structs:"position"`
	ChangedBy string    `structs:"changed_by"`
	Items     string    `structs:"items"`
	CreatedAt time.Time `structs:"created_at"`
	UpdatedAt time.Time `structs:"updated_at"`
}

func (r *playQueueRepository) Store(ctx context.Context, q *model.PlayQueue, colNames ...string) error {
	u := loggedUser(ctx)

	// Always find existing playqueue for this user
	existingQueue, err := r.Retrieve(ctx, q.UserID)
	if err != nil && !errors.Is(err, model.ErrNotFound) {
		log.Error(ctx, "Error retrieving existing playqueue", "user", u.UserName, err)
		return err
	}

	// Use existing ID if found, otherwise keep the provided ID (which may be empty for new records)
	if !errors.Is(err, model.ErrNotFound) && existingQueue.ID != "" {
		q.ID = existingQueue.ID
	}

	// When no specific columns are provided, we replace the whole queue
	if len(colNames) == 0 {
		err := r.clearPlayQueue(ctx, q.UserID)
		if err != nil {
			log.Error(ctx, "Error deleting previous playqueue", "user", u.UserName, err)
			return err
		}
		if len(q.Items) == 0 {
			return nil
		}
	}

	pq := r.fromModel(q)
	if pq.ID == "" {
		pq.CreatedAt = time.Now()
	}
	pq.UpdatedAt = time.Now()
	_, err = r.put(ctx, pq.ID, pq, colNames...)
	if err != nil {
		log.Error(ctx, "Error saving playqueue", "user", u.UserName, err)
		return err
	}
	return nil
}

func (r *playQueueRepository) RetrieveWithMediaFiles(ctx context.Context, userId string) (*model.PlayQueue, error) {
	sel := r.newSelect(ctx).Columns("*").Where(Eq{"user_id": userId})
	var res playQueue
	err := r.queryOne(ctx, sel, &res)
	q := r.toModel(&res)
	q.Items = r.loadTracks(ctx, q.Items)
	return &q, err
}

func (r *playQueueRepository) Retrieve(ctx context.Context, userId string) (*model.PlayQueue, error) {
	sel := r.newSelect(ctx).Columns("*").Where(Eq{"user_id": userId})
	var res playQueue
	err := r.queryOne(ctx, sel, &res)
	return new(r.toModel(&res)), err
}

func (r *playQueueRepository) fromModel(q *model.PlayQueue) playQueue {
	pq := playQueue{
		ID:        q.ID,
		UserID:    q.UserID,
		Current:   q.Current,
		Position:  q.Position,
		ChangedBy: q.ChangedBy,
		CreatedAt: q.CreatedAt,
		UpdatedAt: q.UpdatedAt,
	}
	var itemIDs []string
	for _, t := range q.Items {
		itemIDs = append(itemIDs, t.ID)
	}
	pq.Items = strings.Join(itemIDs, ",")
	return pq
}

func (r *playQueueRepository) toModel(pq *playQueue) model.PlayQueue {
	q := model.PlayQueue{
		ID:        pq.ID,
		UserID:    pq.UserID,
		Current:   pq.Current,
		Position:  pq.Position,
		ChangedBy: pq.ChangedBy,
		CreatedAt: pq.CreatedAt,
		UpdatedAt: pq.UpdatedAt,
	}
	if strings.TrimSpace(pq.Items) != "" {
		tracks := strings.SplitSeq(pq.Items, ",")
		for t := range tracks {
			q.Items = append(q.Items, model.MediaFile{ID: t})
		}
	}
	return q
}

// loadTracks loads the tracks from the database. It receives a list of track IDs and returns a list of MediaFiles
// in the same order as the input list.
func (r *playQueueRepository) loadTracks(ctx context.Context, tracks model.MediaFiles) model.MediaFiles {
	if len(tracks) == 0 {
		return nil
	}

	mfRepo := NewMediaFileRepository(r.db)
	trackMap := map[string]model.MediaFile{}

	// Create an iterator to collect all track IDs
	ids := slice.SeqFunc(tracks, func(t model.MediaFile) string { return t.ID })

	// Break the list in chunks, up to 500 items, to avoid hitting SQLITE_MAX_VARIABLE_NUMBER limit
	for chunk := range slice.CollectChunks(ids, 500) {
		idsFilter := Eq{"media_file.id": chunk}
		tracks, err := mfRepo.GetAll(ctx, model.QueryOptions{Filters: idsFilter})
		if err != nil {
			u := loggedUser(ctx)
			log.Error(ctx, "Could not load playqueue/bookmark's tracks", "user", u.UserName, err)
		}
		for _, t := range tracks {
			trackMap[t.ID] = t
		}
	}

	// Create a new list of tracks with the same order as the original
	// Exclude tracks that are not in the DB anymore
	newTracks := make(model.MediaFiles, 0, len(tracks))
	for _, t := range tracks {
		if track, ok := trackMap[t.ID]; ok {
			newTracks = append(newTracks, track)
		}
	}
	return newTracks
}

func (r *playQueueRepository) clearPlayQueue(ctx context.Context, userId string) error {
	return r.delete(ctx, Eq{"user_id": userId})
}

func (r *playQueueRepository) Clear(ctx context.Context, userId string) error {
	return r.clearPlayQueue(ctx, userId)
}

var _ model.PlayQueueRepository = (*playQueueRepository)(nil)
